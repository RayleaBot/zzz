package app

import (
	"context"
	"encoding/json"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// 公告, 资讯, 活动, 米游社搜索, 帖子 and 预估 answer the way the Yunzai 原神插件's
// mysNews does: a post drawn as html/mysNews after a line naming it, or the
// newest five as html/mysNews-list for 公告列表. The templates are the
// plugin's news and news-list; their images come from the Yunzai 原神插件
// source.

// newsGameName is mysNews's name of the game.
const newsGameName = "绝区零"

var newsKinds = map[string]struct {
	Type int
	Name string
}{"news": {1, "公告"}, "info": {3, "资讯"}, "events": {2, "活动"}}

var newsArtwork = [][3]string{
	{"iconfont", "yunzai-genshin", "resources/html/mysNews/iconfont.fb3712d.woff2"},
	{"mys-logo", "yunzai-genshin", "resources/html/mysNews/mys.png"},
	{"tttgbnumber", "yunzai-genshin", "resources/font/tttgbnumber.ttf"},
	{"list-bg", "yunzai-genshin", "resources/html/mysNews-list/蒙德.png"},
}

// newsEmoticons are 米游社's emoticons by name, read once as upstream does.
type newsEmoticons struct {
	mu    sync.Mutex
	icons map[string]string
}

func (a *App) newsCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	gid := bbsGID
	game := newsGameName
	word := strings.Join(args, "")
	switch command {
	case "news", "info", "events":
		kind := newsKinds[command]
		size := 20
		if strings.Contains(word, "列表") {
			size = 5
		}
		data, err := a.Content.get(ctx, "https://bbs-api-static.miyoushe.com/painter/wapi/getNewsList?"+url.Values{"gids": {gid}, "page_size": {strconv.Itoa(size)}, "type": {strconv.Itoa(kind.Type)}}.Encode(), nil)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		list := asList(data["list"])
		if len(list) == 0 {
			return event.Result(map[string]any{"handled": true})
		}
		if size == 5 {
			title := game + kind.Name + "：米游社" + game + kind.Name + "列表"
			text := title
			for index, item := range list {
				text += "\n" + strconv.Itoa(index+1) + ". " + asText(asObject(asObject(item)["post"])["subject"])
			}
			return a.sendNewsImage(ctx, event, title, text, func() Image { return a.newsListImage(ctx, game, kind.Name, list) })
		}
		index, err := strconv.Atoi(strings.TrimSpace(word))
		if word == "" {
			index, err = 1, nil
		}
		if err != nil || index < 1 || index > len(list) {
			return event.SendText("目前只查前20条最新的公告，请输入1-20之间的整数。")
		}
		return a.sendNewsPost(ctx, event, asText(asObject(asObject(list[index-1])["post"])["post_id"]), game+kind.Name+"：")
	case "search":
		// A last digit picks that result, counting from 0, as upstream.
		index := 0
		if last := word[max(0, len(word)-1):]; last >= "0" && last <= "9" {
			index = int(last[0] - '0')
			word = strings.Trim(word, last)
		}
		if word == "" {
			return event.SendText("请输入关键字，如" + a.Game.Prefix + "米游社七七")
		}
		data, err := a.Content.get(ctx, "https://bbs-api.miyoushe.com/post/wapi/searchPosts?"+url.Values{"gids": {gid}, "size": {"20"}, "keyword": {word}}.Encode(), nil)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		posts := asList(data["posts"])
		if index >= len(posts) {
			return event.SendText("搜索不到您要的结果，换个关键词试试呗~")
		}
		return a.sendNewsPost(ctx, event, asText(asObject(asObject(posts[index])["post"])["post_id"]), "")
	case "post":
		id, err := publicPostID(word)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return a.sendNewsPost(ctx, event, id, "")
	}
	return a.newsEstimate(ctx, event)
}

// newsEstimate answers 预估 with the author's newest 菲林统计汇总 post, drawn
// as the Yunzai 原神插件 draws its posts.
func (a *App) newsEstimate(ctx context.Context, event *rayleabot.EventContext) error {
	data, err := a.Content.get(ctx, "https://bbs-api.miyoushe.com/painter/api/user_instant/search/list?"+url.Values{"keyword": {"菲林统计汇总"}, "uid": {"137101761"}, "size": {"20"}, "offset": {"0"}, "sort_type": {"2"}}.Encode(), nil)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	list := asList(data["list"])
	if len(list) == 0 || asText(asObject(asObject(asObject(list[0])["post"])["post"])["post_id"]) == "" {
		return event.SendText("暂无数据")
	}
	post := asObject(asObject(asObject(list[0])["post"])["post"])
	return a.sendNewsPost(ctx, event, asText(post["post_id"]), "")
}

// sendNewsPost draws a post after a line with its title.
func (a *App) sendNewsPost(ctx context.Context, event *rayleabot.EventContext, id, title string) error {
	gid := bbsGID
	data, err := a.Content.get(ctx, "https://bbs-api.miyoushe.com/post/wapi/getPostFull?"+url.Values{"gids": {gid}, "read": {"1"}, "post_id": {id}}.Encode(), nil)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	full := asObject(data["post"])
	post := asObject(full["post"])
	if asText(post["post_id"]) != id {
		return event.SendText("帖子编号与请求不一致。")
	}
	path := bbsPath
	title += asText(post["subject"])
	return a.sendNewsImage(ctx, event, title, title+"\nhttps://www.miyoushe.com/"+path+"/article/"+id, func() Image { return a.newsPostImage(ctx, full) })
}

// sendNewsImage sends the line and the page as one message, or the text
// when image replies are off or the page cannot be drawn.
func (a *App) sendNewsImage(ctx context.Context, event *rayleabot.EventContext, title, text string, build func() Image) error {
	if settings(event).ImageReplies {
		image := build()
		result, err := event.Actions().RenderImage(ctx, rayleabot.RenderImageRequest{Template: image.Template, Output: "jpeg", FallbackText: text, Data: image.Data, Resources: image.Resources})
		if path := asText(result["image_path"]); err == nil && path != "" {
			return event.Send(event.Event.Target.Type, event.Event.Target.ID, rayleabot.Text(title), rayleabot.Image(path))
		}
	}
	return event.SendText(text)
}

func (a *App) newsResources(ctx context.Context) *ImageResources {
	resources := &ImageResources{Context: a.imageContext(ctx)}
	for _, item := range newsArtwork {
		resources.Artwork(item[0], item[1], item[2])
	}
	return resources
}

// newsPostImage fills html/mysNews: the author with level and official
// mark, the post's link as a QR code, the board, counts, time, content and
// topics.
func (a *App) newsPostImage(ctx context.Context, full map[string]any) Image {
	post, user, stat := asObject(full["post"]), asObject(full["user"]), asObject(full["stat"])
	path := bbsPath
	id := asText(post["post_id"])
	link := "https://www.miyoushe.com/" + path + "/article/" + id
	resources := a.newsResources(ctx)
	picture := func(address string) string {
		if address = publicImage(address); address == "" {
			return ""
		}
		return resources.URL("mihoyo", address)
	}
	pictures := newsStructuredPictures(asText(post["structured_content"]))
	resources.Prefetch("mihoyo", append(slices.Clone(asList(post["images"])), user["avatar_url"])...)
	content := newsContent(asText(post["content"]), pictures, a.newsEmoticons(ctx), picture)
	// A post of pictures keeps them as JSON; upstream draws them as image
	// blocks.
	var gallery struct {
		Images []string `json:"imgs"`
	}
	if json.Unmarshal([]byte(asText(post["content"])), &gallery) == nil && len(gallery.Images) > 0 {
		content = ""
		for _, image := range gallery.Images {
			if id := picture(image); id != "" {
				content += `<div class="ql-image-box"><img data-render-resource="` + id + `"></div>`
			}
		}
	}
	count := func(value any) string {
		n, ok := challengeNumber(value)
		if ok && n > 10000 {
			return strconv.FormatFloat(n/10000, 'f', 2, 64) + "万"
		}
		return asText(value)
	}
	topics := []string{}
	for _, topic := range asList(full["topics"]) {
		topics = append(topics, asText(asObject(topic)["name"]))
	}
	level := asText(asObject(user["level_exp"])["level"])
	certification := asObject(user["certification"])
	data := map[string]any{
		"qr":        newsQRCode(link),
		"avatar":    picture(asText(user["avatar_url"])),
		"nickname":  asText(user["nickname"]),
		"level":     resources.URL("mihoyo", "https://img-static.mihoyo.com/level/level"+level+".png"),
		"certified": asText(certification["type"]) == "1",
		"label":     asText(certification["label"]),
		"subject":   asText(post["subject"]),
		"forum":     asText(asObject(full["forum"])["name"]),
		"views":     count(stat["view_num"]),
		"likes":     count(stat["like_num"]),
		"replies":   count(stat["reply_num"]),
		"created":   newsTime(post["created_at"]),
		"content":   content,
		"topics":    topics,
	}
	return Image{Template: "news", Data: data, Resources: resources.List}
}

// newsListImage fills html/mysNews-list with the newest posts.
func (a *App) newsListImage(ctx context.Context, game, kind string, list []any) Image {
	resources := a.newsResources(ctx)
	covers := []any{}
	for _, item := range list {
		images := asList(asObject(asObject(item)["post"])["images"])
		cover := ""
		if len(images) > 0 {
			cover = publicImage(asText(images[0]))
		}
		covers = append(covers, cover)
	}
	resources.Prefetch("mihoyo", covers...)
	items := []any{}
	for index, item := range list {
		post := asObject(asObject(item)["post"])
		cover := ""
		if address := asText(covers[index]); address != "" {
			cover = resources.URL("mihoyo", address)
		}
		items = append(items, map[string]any{"num": index + 1, "last": index == len(list)-1, "subject": asText(post["subject"]), "content": asText(post["content"]), "created": newsTime(post["created_at"]), "cover": cover})
	}
	prefix := a.Game.Prefix
	return Image{Template: "news-list", Data: map[string]any{"game": game, "kind": kind, "items": items, "hint": "详情内容使用【" + prefix + kind + "】加上序号查看，如【" + prefix + kind + "2】"}, Resources: resources.List}
}

// newsTime is Date.toLocaleString in the zh-CN form upstream runs with.
func newsTime(value any) string {
	stamp, err := strconv.ParseInt(asText(value), 10, 64)
	if err != nil || stamp <= 0 {
		return ""
	}
	return time.Unix(stamp, 0).In(time.FixedZone("UTC+8", 8*3600)).Format("2006/1/2 15:04:05")
}

func (a *App) newsEmoticons(ctx context.Context) map[string]string {
	a.emoticons.mu.Lock()
	defer a.emoticons.mu.Unlock()
	if a.emoticons.icons != nil {
		return a.emoticons.icons
	}
	gid := bbsGID
	data, err := a.Content.get(ctx, "https://bbs-api.miyoushe.com/misc/api/emoticon_set?gids="+gid, nil)
	if err != nil {
		return nil
	}
	icons := map[string]string{}
	for _, set := range asList(data["list"]) {
		for _, item := range asList(asObject(set)["list"]) {
			if icon := asText(asObject(item)["icon"]); icon != "" {
				icons[asText(asObject(item)["name"])] = icon
			}
		}
	}
	a.emoticons.icons = icons
	return icons
}
