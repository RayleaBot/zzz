package app

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// Chat 攻略 sends the image ZZZ-Plugin's strategy command does: the largest
// image of the first collection post naming the character.

var guideWord = regexp.MustCompile(`^(更新|刷新)?(.+?)攻略([0-9]+|all)?$`)

// guideOSS is the resize the upstream commands ask the image host for.
var guideOSS = "?x-oss-process=image/resize,s_1200/quality,q_90/auto-orient,0/interlace,1/format,jpg"

// guideSamples are the characters the upstream 攻略帮助 replies show.
var guideSamples = "艾莲"

var (
	guideBookTag   = regexp.MustCompile(`【[^】]*本[^】]*】`)
	guideNameChars = regexp.MustCompile(`[^a-zA-Z0-9\x{4e00}-\x{9fa5}]`)
)

// guidePostImage is the image a post gives for a character's guide, or ""
// when the post is not the character's: its largest image by file size, the
// last of equal ones, as ZZZ-Plugin picks it. GIFs are skipped, and 【…本…】
// tags in the title do not count toward the name.
func guidePostImage(name string, item map[string]any) string {
	subject := guideBookTag.ReplaceAllString(asText(asObject(item["post"])["subject"]), "")
	if !strings.Contains(subject, guideNameChars.ReplaceAllString(name, "")) {
		return ""
	}
	images, _ := item["image_list"].([]any)
	best, size := "", -1.0
	for _, image := range images {
		if asText(asObject(image)["format"]) == "gif" {
			continue
		}
		if value, _ := strconv.ParseFloat(asText(asObject(image)["size"]), 64); value >= size {
			best, size = asText(asObject(image)["url"]), value
		}
	}
	return best
}

// guidePicture reads a source's collections and returns the guide image of
// the first post naming the character, resized as upstream asks; "" when no
// post names it.
func (c PublicContentClient) guidePicture(ctx context.Context, source GuideSource, name string) (string, error) {
	gid := bbsGID
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
			if found := guidePostImage(name, asObject(post)); found != "" {
				link, err := url.Parse(found)
				if err != nil || link.Scheme != "https" || !(strings.HasSuffix(link.Hostname(), ".miyoushe.com") || strings.HasSuffix(link.Hostname(), ".mihoyo.com")) {
					return "", gameError("public_invalid", "攻略图片地址不在允许范围内。")
				}
				return found + guideOSS, nil
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
	sources := guideSources
	config, err := a.GuideSettings.Manage("guides.settings", nil)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	settings := config["settings"].(GuideConfig)
	switch command {
	case "guide-help":
		sample := guideSamples
		lines := []string{a.Game.Name + "攻略帮助:", a.Game.Prefix + sample + "攻略[来源序号]", a.Game.Prefix + "更新" + sample + "攻略[来源序号]", a.Game.Prefix + "设置默认攻略[来源序号]", a.Game.Prefix + "设置所有攻略显示个数[个数]", "示例: " + a.Game.Prefix + sample + "攻略2", "", "攻略来源:"}
		for _, source := range sources {
			lines = append(lines, source.ID+"——"+source.Name)
		}
		return event.SendText(strings.Join(lines, "\n"))
	case "guide-default":
		// As upstream, all or 0 is the collection of the first sources.
		arg := ""
		if len(args) > 0 {
			arg = args[0]
		}
		if arg == "all" {
			arg = "0"
		}
		index, err := strconv.Atoi(arg)
		if err != nil || index < 0 || index > len(sources) {
			return event.SendText(strings.Join([]string{a.Game.Name + "默认攻略设置方式为:", a.Game.Prefix + "设置默认攻略[0123...]", "请增加数字0-" + strconv.Itoa(len(sources)) + "其中一个，或者增加 all 以显示所有攻略", "攻略来源请输入 " + a.Game.Prefix + "攻略帮助 查看"}, "\n"))
		}
		name := "all"
		if index > 0 {
			name = sources[index-1].Name
		}
		if _, err := a.GuideSettings.Manage("guides.configure", map[string]any{"revision": settings.Revision, "default_source": strconv.Itoa(index)}); err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.SendText(a.Game.Name + "默认攻略已设置为: " + strconv.Itoa(index) + " (" + name + ")")
	case "guide-forward-count":
		count := -1
		if len(args) > 0 {
			if parsed, err := strconv.Atoi(args[0]); err == nil {
				count = parsed
			}
		}
		switch {
		case count < 1:
			return event.SendText("所有攻略显示个数不能小于1")
		case count > len(sources):
			return event.SendText("所有攻略显示个数不能大于" + strconv.Itoa(len(sources)))
		}
		if _, err := a.GuideSettings.Manage("guides.configure", map[string]any{"revision": settings.Revision, "default_source": settings.Default, "forward_count": count}); err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.SendText(a.Game.Name + "所有攻略显示个数已设置为: " + strconv.Itoa(count))
	}
	match := guideWord.FindStringSubmatch(strings.TrimSpace(event.Event.Command()))
	if match == nil {
		return event.Result(map[string]any{"handled": false})
	}
	refresh, raw, group := match[1] != "", strings.TrimSpace(match[2]), match[3]
	entry, ok := a.Catalog.Resolve(raw, "character", a.aliasMap(event))
	if !ok {
		return event.SendText("该角色不存在")
	}
	name := entry.Name
	picked := []int{}
	if group == "" {
		group = settings.Default
	}
	if group == "all" || group == "0" {
		for i := range min(len(sources), settings.forwardCount()) {
			picked = append(picked, i)
		}
	} else {
		index, _ := strconv.Atoi(group)
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
			if link, err = a.Content.guidePicture(ctx, source, name); err != nil {
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
