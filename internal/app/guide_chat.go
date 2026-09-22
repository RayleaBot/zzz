package app

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// Chat 攻略 and 地图 send images the way their upstream commands do: a guide
// is the largest image of the first collection post naming the character,
// as the Yunzai 原神插件, StarRail-plugin and ZZZ-Plugin strategy commands
// pick it, and a map is the image xiaoyao's #xx在哪里 fetches from minigg.

var guideWord = regexp.MustCompile(`^(更新|刷新)?(.+?)攻略([0-9]+|all)?$`)

// guideOSS is the resize the upstream commands ask the image host for.
var guideOSS = map[string]string{
	"genshin":  "?x-oss-process=image//resize,s_1200/quality,q_90/auto-orient,0/interlace,1/format,jpg",
	"starrail": "?x-oss-process=image//resize,s_1200/quality,q_90/auto-orient,0/interlace,1/format,jpg",
	"zzz":      "?x-oss-process=image/resize,s_1200/quality,q_90/auto-orient,0/interlace,1/format,jpg",
}

// guideSamples are the characters the upstream 攻略帮助 replies show.
var guideSamples = map[string]string{"genshin": "心海", "starrail": "希儿", "zzz": "艾莲"}

var (
	guideBookTag   = regexp.MustCompile(`【[^】]*本[^】]*】`)
	guideNameChars = regexp.MustCompile(`[^a-zA-Z0-9\x{4e00}-\x{9fa5}]`)
)

// guidePostImage is the image a post gives for a character's guide, or ""
// when the post is not the character's.
func guidePostImage(game, sourceID, name string, item map[string]any) string {
	post := asObject(item["post"])
	images, _ := item["image_list"].([]any)
	subject := asText(post["subject"])
	number := func(image any, key string) float64 {
		value, _ := strconv.ParseFloat(asText(asObject(image)[key]), 64)
		return value
	}
	largest := func(key string, skipGIF, last bool) string {
		best := -1
		for i, image := range images {
			if skipGIF && asText(asObject(image)["format"]) == "gif" {
				continue
			}
			if best < 0 || number(image, key) > number(images[best], key) || last && number(image, key) == number(images[best], key) {
				best = i
			}
		}
		if best < 0 {
			return ""
		}
		return asText(asObject(images[best])["url"])
	}
	switch {
	case game == "genshin" && sourceID == "4":
		// This source puts every character in one post; the character's
		// section names the image after 【name】.
		content := strings.ReplaceAll(asText(post["structured_content"]), `\/{}`, "")
		if !strings.Contains(content, name+"】") {
			return ""
		}
		match := regexp.MustCompile(regexp.QuoteMeta(name) + `】.*?image\\?":\\?"(.*?)\\?"`).FindStringSubmatch(content)
		if match == nil {
			return ""
		}
		for _, image := range images {
			if asText(asObject(image)["image_id"]) == match[1] {
				return asText(asObject(image)["url"])
			}
		}
		return ""
	case game == "zzz":
		if !strings.Contains(guideBookTag.ReplaceAllString(subject, ""), guideNameChars.ReplaceAllString(name, "")) {
			return ""
		}
		return largest("size", true, true)
	case game == "starrail":
		base, path, trailblazer := strings.Cut(name, "·")
		if !strings.Contains(subject, name) && !(trailblazer && base == "开拓者" && strings.Contains(subject, base) && strings.Contains(subject, path)) {
			return ""
		}
		return largest("height", false, false)
	default:
		if !strings.Contains(subject, name) {
			return ""
		}
		return largest("size", false, true)
	}
}

// guidePicture reads a source's collections and returns the guide image of
// the first post naming the character, resized as upstream asks; "" when no
// post names it.
func (c PublicContentClient) guidePicture(ctx context.Context, game string, source GuideSource, name string) (string, error) {
	gid, _ := publicGame(game)
	pages := make([]map[string]any, len(source.Collections))
	failures := make([]error, len(source.Collections))
	var wait sync.WaitGroup
	for i, id := range source.Collections {
		wait.Add(1)
		go func() {
			defer wait.Done()
			params := url.Values{"gids": {gid}, "order_type": {"2"}, "collection_id": {id}}
			pages[i], failures[i] = c.get(ctx, "https://bbs-api.mihoyo.com/post/wapi/getPostFullInCollection?"+params.Encode(), nil)
		}()
	}
	wait.Wait()
	if err := errors.Join(failures...); err != nil {
		return "", err
	}
	for _, page := range pages {
		posts, _ := page["posts"].([]any)
		for _, post := range posts {
			if found := guidePostImage(game, source.ID, name, asObject(post)); found != "" {
				link, err := url.Parse(found)
				if err != nil || link.Scheme != "https" || !(strings.HasSuffix(link.Hostname(), ".miyoushe.com") || strings.HasSuffix(link.Hostname(), ".mihoyo.com")) {
					return "", gameError("public_invalid", "攻略图片地址不在允许范围内。")
				}
				return found + guideOSS[game], nil
			}
		}
	}
	return "", nil
}

// guideCache keeps the guide images found, as upstream keeps the downloaded
// files, until 更新 asks again or a day passes.
type guideCache struct {
	mu    sync.Mutex
	items map[string]guideCached
}

type guideCached struct {
	url string
	at  time.Time
}

func (c *guideCache) get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.items[key]
	return item.url, ok && time.Since(item.at) < 24*time.Hour
}

func (c *guideCache) put(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.items == nil || len(c.items) >= 256 {
		c.items = map[string]guideCached{}
	}
	c.items[key] = guideCached{value, time.Now()}
}

// guideCommand answers 攻略, 攻略帮助 and 设置默认攻略.
func (a *App) guideCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	sources := guideSources[a.Game.ID]
	config, err := a.GuideSettings.Manage(a.Game.ID, "guides.settings", nil)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	settings := config["settings"].(GuideConfig)
	switch command {
	case "guide-help":
		sample := guideSamples[a.Game.ID]
		lines := []string{a.Game.Name + "攻略帮助:", a.Game.Prefix + sample + "攻略[来源序号]", a.Game.Prefix + "更新" + sample + "攻略[来源序号]", a.Game.Prefix + "设置默认攻略[来源序号]", "示例: " + a.Game.Prefix + sample + "攻略2", "", "攻略来源:"}
		for _, source := range sources {
			lines = append(lines, source.ID+"——"+source.Name)
		}
		return event.SendText(strings.Join(lines, "\n"))
	case "guide-default":
		if len(args) == 0 {
			return event.SendText("默认攻略设置方式为: \n" + a.Game.Prefix + "设置默认攻略[1-" + strconv.Itoa(len(sources)) + "]")
		}
		if _, err := a.GuideSettings.Manage(a.Game.ID, "guides.configure", map[string]any{"revision": settings.Revision, "default_source": args[0]}); err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.SendText("默认攻略已设置为: " + args[0])
	}
	match := guideWord.FindStringSubmatch(strings.TrimSpace(event.Event.Command()))
	if match == nil {
		return event.Result(map[string]any{"handled": false})
	}
	refresh, raw, group := match[1] != "", strings.TrimSpace(match[2]), match[3]
	entry, ok := a.Catalog.Resolve(raw, "character", a.aliasMap(event))
	if !ok {
		if a.Game.ID == "zzz" {
			return event.SendText("该角色不存在")
		}
		return event.Result(map[string]any{"handled": false})
	}
	name := entry.Name
	switch {
	case a.Game.ID == "genshin" && (entry.ID == "10000005" || entry.ID == "10000007" || entry.ID == "20000000"):
		travelers := []string{"风主", "岩主", "雷主", "草主", "水主"}
		if !slices.Contains(travelers, raw) {
			choices := []string{}
			for _, traveler := range travelers {
				choices = append(choices, traveler+"攻略"+cmpOr(group, settings.Default))
			}
			return event.SendText("请选择：" + strings.Join(choices, "、"))
		}
		name = raw
	case a.Game.ID == "starrail":
		name = strings.ReplaceAll(name, "•", "·")
		if name == "托帕&账账" {
			name = "托帕"
		}
	}
	picked := []int{}
	if group == "all" || group == "0" {
		for i := range min(len(sources), 4) {
			picked = append(picked, i)
		}
	} else {
		index, _ := strconv.Atoi(cmpOr(group, settings.Default))
		if index < 1 || index > len(sources) {
			return event.SendText("超过攻略数量（" + strconv.Itoa(len(sources)) + "）")
		}
		picked = []int{index - 1}
	}
	segments := []rayleabot.Segment{}
	for _, index := range picked {
		source := sources[index]
		key := a.Game.ID + "/" + source.ID + "/" + name
		link, cached := a.guides.get(key)
		if !cached || refresh {
			if link, err = a.Content.guidePicture(ctx, a.Game.ID, source, name); err != nil {
				return event.SendText("暂无攻略数据，请稍后再试")
			}
			if link != "" {
				a.guides.put(key, link)
			}
		}
		switch {
		case link != "":
			segments = append(segments, rayleabot.Image(link))
		case len(picked) == 1:
			return event.SendText("暂无" + name + "攻略（" + source.Name + "）\n请尝试其他的攻略来源查询\n" + a.Game.Prefix + "攻略帮助，查看说明")
		default:
			segments = append(segments, rayleabot.Text("暂无"+name+"攻略（"+source.Name+"）\n"))
		}
	}
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, segments...)
}

// mapWords are the underground maps xiaoyao recognizes in the word, by
// minigg map ID.
var mapWords = []struct {
	id    string
	words *regexp.Regexp
}{{"7", regexp.MustCompile(`渊下宫|渊下`)}, {"9", regexp.MustCompile(`璃月地下|层岩地下|层岩`)}}

// mapPicture fetches minigg's map image of a resource; its refusals come
// back as the service's own message.
func (a *App) mapPicture(ctx context.Context, resource, mapID string) ([]byte, error) {
	params := url.Values{"resource_name": {resource}, "is_cluster": {"false"}}
	if mapID != "" {
		params.Set("map_id", mapID)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://map.minigg.cn/map/get_map?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	client := a.Content.HTTP
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		var certificate *x509.CertificateInvalidError
		var unknown x509.UnknownAuthorityError
		var timeout net.Error
		reason := "无法连接"
		switch {
		case errors.As(err, &certificate) || errors.As(err, &unknown):
			reason = "证书校验失败"
		case errors.As(err, &timeout) && timeout.Timeout():
			reason = "连接超时"
		case strings.Contains(err.Error(), "handshake") || strings.Contains(err.Error(), "tls"):
			reason = "TLS 握手失败"
		}
		return nil, gameError("public_unavailable", "地图服务 map.minigg.cn 暂不可用："+reason+"。")
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return nil, gameError("public_unavailable", "地图服务 map.minigg.cn 的图片读取失败。")
	}
	var refusal struct {
		Retcode json.Number `json:"retcode"`
		Message string      `json:"message"`
	}
	if json.Unmarshal(body, &refusal) == nil {
		return nil, gameError("public_unavailable", cmpOr(plainGameText(refusal.Message), "地图服务没有找到该资源。"))
	}
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(http.DetectContentType(body), "image/") {
		return nil, gameError("public_unavailable", "地图服务 map.minigg.cn 暂不可用：HTTP "+strconv.Itoa(response.StatusCode)+"。")
	}
	return body, nil
}

// mapCommand is xiaoyao's #xx在哪里: the underground map is named in the
// word, and 刷新 or 更新 skips the kept image.
func (a *App) mapCommand(ctx context.Context, event *rayleabot.EventContext) error {
	word := strings.TrimSpace(event.Event.Command())
	refresh := strings.HasPrefix(word, "刷新") || strings.HasPrefix(word, "更新")
	resource := strings.NewReplacer("哪", "", "那", "", "里", "", "在", "", "刷新", "", "更新", "").Replace(word)
	mapID := ""
	for _, item := range mapWords {
		if item.words.MatchString(resource) {
			mapID, resource = item.id, item.words.ReplaceAllString(resource, "")
			break
		}
	}
	resource = strings.TrimSpace(resource)
	if resource == "" || resource == "地图" || len([]rune(resource)) > 40 {
		return event.SendText("使用“" + a.Game.Prefix + "甜甜花在哪里”查询资源位置，渊下宫、层岩地下的资源在名称中写明地图，如“" + a.Game.Prefix + "渊下宫夜泊石在哪里”。")
	}
	key := "map/" + mapID + "/" + resource
	a.maps.mu.Lock()
	image, cached := a.maps.items[key]
	a.maps.mu.Unlock()
	if !cached || refresh {
		var err error
		if image, err = a.mapPicture(ctx, resource, mapID); err != nil {
			return event.SendText(friendlyError(err))
		}
		a.maps.mu.Lock()
		if a.maps.items == nil || len(a.maps.items) >= 32 {
			a.maps.items = map[string][]byte{}
		}
		a.maps.items[key] = image
		a.maps.mu.Unlock()
	}
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, rayleabot.Image("base64://"+base64.StdEncoding.EncodeToString(image)))
}

// mapImages keeps the fetched maps, as xiaoyao keeps them in data/map.
type mapImages struct {
	mu    sync.Mutex
	items map[string][]byte
}
