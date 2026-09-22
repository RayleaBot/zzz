package app

import (
	"context"
	"encoding/json"
	"html"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

type ContentQuery struct {
	Source           string `json:"source"`
	CollectionOffset int    `json:"collection_offset"`
	Kind             string `json:"kind"`
	Type             int    `json:"type"`
	LastID           string `json:"last_id"`
	Query            string `json:"query"`
	PostID           string `json:"post_id"`
	Offset           int    `json:"offset"`
}
type PublicPost struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	URL     string   `json:"url"`
	Created string   `json:"created"`
	Author  string   `json:"author,omitempty"`
	Parts   []string `json:"parts,omitempty"`
	Images  []string `json:"images,omitempty"`
}
type PublicActivity struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Text       string `json:"text"`
	Start      string `json:"start,omitempty"`
	End        string `json:"end,omitempty"`
	StartMS    int64  `json:"start_ms,omitempty"`
	EndMS      int64  `json:"end_ms,omitempty"`
	TimeStatus string `json:"time_status"`
}

var publicDigits = regexp.MustCompile(`^[0-9]{1,20}$`)
var publicTags = regexp.MustCompile(`(?s)<[^>]*>`)
var publicBreaks = regexp.MustCompile(`(?i)</?(?:p|div|h[1-6]|li|br|section|tr)[^>]*>`)
var publicScript = regexp.MustCompile(`(?is)<(?:script|style|iframe)[^>]*>.*?</(?:script|style|iframe)>`)
var publicDate = regexp.MustCompile(`20[0-9]{2}[-/][0-9]{1,2}[-/][0-9]{1,2}[ T][0-9]{1,2}:[0-9]{2}(?::[0-9]{2})?`)
var publicDatePair = regexp.MustCompile(`(` + publicDate.String() + `)\s*(?:~|～|至|—|-)\s*(` + publicDate.String() + `)`)
var publicDateEnd = regexp.MustCompile(`(?:更新后|完成后|开放后|至|~|～)\s*(?:~|～|至)?\s*(` + publicDate.String() + `)`)

func publicText(v string) string {
	v = html.UnescapeString(v)
	v = publicScript.ReplaceAllString(v, "")
	v = publicBreaks.ReplaceAllString(v, "\n")
	v = html.UnescapeString(publicTags.ReplaceAllString(v, ""))
	return strings.TrimSpace(strings.ReplaceAll(v, "\u00a0", " "))
}

// bbsGID and bbsPath name Zenless Zone Zero on 米游社: its forum ID and URL
// path.
const (
	bbsGID  = "8"
	bbsPath = "zzz"
)

func publicPostID(value string) (string, error) {
	if publicDigits.MatchString(value) {
		return value, nil
	}
	u, err := url.Parse(value)
	path := bbsPath
	if err != nil || u.Scheme != "https" || u.User != nil || !slices.Contains([]string{"www.miyoushe.com", "miyoushe.com", "bbs.mihoyo.com"}, u.Host) || u.RawQuery != "" {
		return "", gameError("input_invalid", "请输入本游戏米游社文章编号或官方文章链接。")
	}
	route := u.Path
	if u.Fragment != "" {
		route = u.Fragment
	}
	prefix := "/" + path + "/article/"
	if !strings.HasPrefix(route, prefix) || !publicDigits.MatchString(strings.TrimPrefix(route, prefix)) {
		return "", gameError("input_invalid", "文章链接格式或游戏不匹配。")
	}
	return strings.TrimPrefix(route, prefix), nil
}
func publicImage(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Fragment != "" {
		return ""
	}
	if !slices.Contains([]string{"upload-bbs.miyoushe.com", "upload-bbs.mihoyo.com", "act-upload.mihoyo.com", "fastcdn.mihoyo.com", "bbs-static.miyoushe.com"}, u.Host) {
		return ""
	}
	u.RawQuery = ""
	return u.String()
}
func normalizePublicPost(raw any, full bool) (PublicPost, error) {
	v := asObject(raw)
	p := asObject(v["post"])
	if p == nil {
		p = v
	}
	id := asText(p["post_id"])
	gid, path := bbsGID, bbsPath
	if !publicDigits.MatchString(id) || p["game_id"] != nil && asText(p["game_id"]) != "0" && asText(p["game_id"]) != gid {
		return PublicPost{}, gameError("public_invalid", "帖子编号或所属游戏无效。")
	}
	out := PublicPost{ID: id, Title: publicText(asText(p["subject"])), URL: "https://www.miyoushe.com/" + path + "/article/" + id, Created: calendarTime(p["created_at"]), Author: publicText(asText(asObject(v["user"])["nickname"]))}
	if len([]rune(out.Title)) > 512 {
		return out, gameError("public_invalid", "帖子标题过长。")
	}
	if !full {
		return out, nil
	}
	text := asText(p["content"])
	var rich any
	if json.Unmarshal([]byte(text), &rich) == nil {
		if obj := asObject(rich); obj != nil {
			if s := asText(obj["text"]); s != "" {
				text = s
			} else if ops := asList(obj["ops"]); len(ops) > 0 {
				text = ""
				for _, op := range ops {
					if s, ok := asObject(op)["insert"].(string); ok {
						text += s
					}
				}
			}
		}
	}
	text = publicText(text)
	if len(text) > 256*1024 {
		return out, gameError("public_invalid", "帖子正文超过 256 KiB。")
	}
	runes := []rune(text)
	out.Parts = []string{}
	for len(runes) > 0 {
		n := min(8000, len(runes))
		out.Parts = append(out.Parts, string(runes[:n]))
		runes = runes[n:]
	}
	out.Images = []string{}
	for _, v := range asList(p["images"]) {
		if image := publicImage(asText(v)); image != "" && !slices.Contains(out.Images, image) {
			out.Images = append(out.Images, image)
			if len(out.Images) >= 100 {
				break
			}
		}
	}
	return out, nil
}
func (c PublicContentClient) posts(ctx context.Context, q ContentQuery) (map[string]any, error) {
	gid := bbsGID
	params := url.Values{"gids": {gid}}
	host := "https://bbs-api.miyoushe.com"
	path := ""
	switch q.Kind {
	case "news":
		if q.Type < 1 || q.Type > 3 || q.LastID != "" && !publicDigits.MatchString(q.LastID) {
			return nil, gameError("input_invalid", "公告类别或分页编号无效。")
		}
		host = "https://bbs-api-static.miyoushe.com"
		path = "/painter/wapi/getNewsList"
		params.Set("type", strconv.Itoa(q.Type))
		params.Set("page_size", "20")
		if q.LastID != "" {
			params.Set("last_id", q.LastID)
		}
	case "search":
		if len([]rune(q.Query)) < 1 || len([]rune(q.Query)) > 100 || q.Offset < 0 || q.Offset > 1000 {
			return nil, gameError("input_invalid", "搜索词或偏移无效。")
		}
		path = "/post/wapi/searchPosts"
		params.Set("keyword", q.Query)
		params.Set("size", "20")
		if q.LastID != "" {
			if !publicDigits.MatchString(q.LastID) {
				return nil, gameError("input_invalid", "搜索游标无效。")
			}
			params.Set("last_id", q.LastID)
		}
	case "post":
		id, err := publicPostID(q.PostID)
		if err != nil {
			return nil, err
		}
		path = "/post/wapi/getPostFull"
		params.Set("post_id", id)
		params.Set("read", "0")
	default:
		return nil, gameError("input_invalid", "帖子查询类型无效。")
	}
	data, err := c.get(ctx, host+path+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	if q.Kind == "post" {
		post, err := normalizePublicPost(data["post"], true)
		if err != nil {
			return nil, err
		}
		if post.ID != params.Get("post_id") {
			return nil, gameError("public_invalid", "帖子编号与请求不一致。")
		}
		return map[string]any{"post": post, "fetched_at_ms": time.Now().UnixMilli()}, nil
	}
	field := "list"
	if q.Kind == "search" {
		field = "posts"
	}
	raw, ok := data[field].([]any)
	if !ok || len(raw) > 20 {
		return nil, gameError("public_invalid", "帖子列表格式暂不兼容。")
	}
	items := []PublicPost{}
	for _, v := range raw {
		p, err := normalizePublicPost(v, false)
		if err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	next := ""
	if data["is_last"] != true && len(items) == 20 {
		next = asText(data["last_id"])
		if !publicDigits.MatchString(next) || next == q.LastID {
			return nil, gameError("public_invalid", "官方分页未向前推进。")
		}
	}
	return map[string]any{"items": items, "next_id": next, "next_offset": q.Offset + len(items), "more": next != "", "fetched_at_ms": time.Now().UnixMilli()}, nil
}
func activityWindow(text string) (time.Time, time.Time) {
	for _, label := range []string{"活动时间", "活动跃迁", "开放时间", "开启时间", "折扣时间", "上架时间", "调频时间"} {
		at := strings.Index(text, label)
		if at < 0 {
			continue
		}
		section := []rune(text[at+len(label):])
		section = section[:min(len(section), 500)]
		s := strings.TrimLeft(string(section), "〓■※：: \r\n\t")
		if end := strings.Index(s, "〓"); end >= 0 {
			s = s[:end]
		}
		parse := func(value string) time.Time {
			value = strings.ReplaceAll(strings.ReplaceAll(value, "/", "-"), "T", " ")
			for _, format := range []string{"2006-1-2 15:04:05", "2006-1-2 15:04"} {
				if t, err := time.ParseInLocation(format, value, time.FixedZone("UTC+8", 28800)); err == nil {
					return t
				}
			}
			return time.Time{}
		}
		pairAt, endAt := publicDatePair.FindStringIndex(s), publicDateEnd.FindStringIndex(s)
		if pair := publicDatePair.FindStringSubmatch(s); len(pair) == 3 && (endAt == nil || pairAt[0] <= endAt[0]) {
			a, b := parse(pair[1]), parse(pair[2])
			if !a.IsZero() && b.After(a) {
				return a, b
			}
			return time.Time{}, time.Time{}
		}
		if end := publicDateEnd.FindStringSubmatch(s); len(end) == 2 {
			if b := parse(end[1]); !b.IsZero() {
				return time.Time{}, b
			}
		}
	}
	return time.Time{}, time.Time{}
}

// Announcements are a game's official announcements as the announcement API
// returns them: the list's groups (upstream calendars read the first as game
// notices and the second as events) and every announcement's body, and the
// picture announcements the game lists apart.
type Announcements struct {
	Groups          []any
	Contents        []any
	PictureGroups   []any
	PictureContents []any
}

// announcements reads a game's announcement list and bodies.
func (c PublicContentClient) announcements(ctx context.Context) (Announcements, error) {
	spec := [5]string{"https://announcement-api.mihoyo.com", "https://announcement-static.mihoyo.com", "nap", "prod_gf_cn", "70"}
	biz := spec[2] + "_cn"
	params := url.Values{"game": {spec[2]}, "game_biz": {biz}, "lang": {"zh-cn"}, "bundle_id": {biz}, "platform": {"pc"}, "region": {spec[3]}, "level": {spec[4]}, "uid": {"100000000"}, "channel_id": {"1"}}
	root := "/common/" + biz + "/announcement/api/"
	list, err := c.get(ctx, spec[0]+root+"getAnnList?"+params.Encode(), nil)
	if err != nil {
		return Announcements{}, err
	}
	details, err := c.get(ctx, spec[1]+root+"getAnnContent?"+params.Encode(), nil)
	if err != nil {
		return Announcements{}, err
	}
	groups, ok := list["list"].([]any)
	if !ok {
		return Announcements{}, gameError("public_invalid", "公告列表缺失。")
	}
	contents, ok := details["list"].([]any)
	if !ok {
		return Announcements{}, gameError("public_invalid", "公告正文列表缺失。")
	}
	pictureGroups, _ := list["pic_list"].([]any)
	pictureContents, _ := details["pic_list"].([]any)
	return Announcements{Groups: groups, Contents: contents, PictureGroups: pictureGroups, PictureContents: pictureContents}, nil
}

func (c PublicContentClient) calendar(ctx context.Context) (map[string]any, error) {
	announcements, err := c.announcements(ctx)
	if err != nil {
		return nil, err
	}
	return calendarItems(announcements)
}

// calendarItems lists the announcements with the time window each states.
func calendarItems(announcements Announcements) (map[string]any, error) {
	lookup := map[string]map[string]any{}
	for _, v := range announcements.Contents {
		m := asObject(v)
		lookup[asText(m["ann_id"])] = m
	}
	items := []PublicActivity{}
	seen := map[string]bool{}
	for _, group := range announcements.Groups {
		for _, v := range asList(asObject(group)["list"]) {
			m := asObject(v)
			id := asText(m["ann_id"])
			if !publicDigits.MatchString(id) || seen[id] {
				continue
			}
			seen[id] = true
			detail := lookup[id]
			text := publicText(asText(detail["content"]))
			if len(text) > 256*1024 {
				return nil, gameError("public_invalid", "单篇公告过大。")
			}
			title := publicText(firstText(m, "title", "subtitle"))
			if title == "" {
				title = publicText(asText(detail["title"]))
			}
			item := PublicActivity{ID: id, Title: title, Text: text, TimeStatus: "unknown"}
			start, end := activityWindow(text)
			if !end.IsZero() {
				item.End = end.Format("2006-01-02 15:04:05 UTC+8")
				item.EndMS = end.UnixMilli()
				item.TimeStatus = "explicit_end"
				if !start.IsZero() {
					item.Start = start.Format("2006-01-02 15:04:05 UTC+8")
					item.StartMS = start.UnixMilli()
					item.TimeStatus = "explicit"
				}
			}
			items = append(items, item)
			if len(items) > 300 {
				return nil, gameError("public_invalid", "公告数量超过处理上限。")
			}
		}
	}
	return map[string]any{"items": items, "fetched_at_ms": time.Now().UnixMilli(), "source": "official_announcements_cn"}, nil
}
