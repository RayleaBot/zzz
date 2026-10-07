package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/zzz/internal/localdata"
)

// Yunzai's 米游社推送 (mysNews): a group administrator turns on 公告 and 资讯
// pushes for the group. Every five minutes a group gets at most one new post
// of the last two hours whose subject has none of Yunzai's banned words,
// drawn as mysNews after a line naming it, each post once in ten hours.
// 推送公告 runs every group's check at once. Yunzai warns of ending
// activities only for Genshin and Star Rail.

// pushBanWords are Yunzai's pushNews banWord for Zenless Zone Zero.
var pushBanWords = regexp.MustCompile(`作品展示|已开奖|大别野`)

// pushKinds are the pushes a group can turn on, with Yunzai's names; the
// news kinds with their getNewsList type.
var pushKinds = []struct {
	kind, name string
	newsType   int
}{{"announce", "公告", 1}, {"info", "资讯", 3}}

// ContentSubscription is a group's pushes, checked by its scheduled task.
type ContentSubscription struct {
	Ref     string   `json:"ref"`
	Owner   Subject  `json:"owner"`
	GroupID string   `json:"group_id"`
	Kinds   []string `json:"kinds"`
	// Sent are the posts pushed in the last ten hours, as upstream's redis
	// keys.
	Sent     map[string]int64 `json:"sent"`
	LastCode string           `json:"last_code,omitempty"`
}

type ContentSubscriptions struct {
	mu   sync.Mutex
	Path string
}

func (s *ContentSubscriptions) read() ([]ContentSubscription, error) {
	items := []ContentSubscription{}
	err := localdata.Read(s.Path, &items)
	return items, err
}

func (s *ContentSubscriptions) List() ([]ContentSubscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read()
}

func (s *ContentSubscriptions) edit(ref string, fn func(*[]ContentSubscription, int) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return err
	}
	i := slices.IndexFunc(items, func(v ContentSubscription) bool { return v.Ref == ref })
	if err = fn(&items, i); err != nil {
		return err
	}
	return localdata.Write(s.Path, items)
}

// pushPost is a post of getNewsList with the push it belongs to.
type pushPost struct {
	id, subject, kind string
	created           int64
}

// postToPush is the post a group gets now: the first by ID of those created
// in the last two hours, free of banned words, of a kind the group has on
// and not pushed to it in ten hours.
func postToPush(posts []pushPost, sub ContentSubscription, now time.Time) (pushPost, bool) {
	slices.SortStableFunc(posts, func(a, b pushPost) int {
		if len(a.id) != len(b.id) {
			return len(a.id) - len(b.id)
		}
		return strings.Compare(a.id, b.id)
	})
	for _, post := range posts {
		sent, seen := sub.Sent[post.id]
		if slices.Contains(sub.Kinds, post.kind) && now.Unix()-post.created <= 7200 && !pushBanWords.MatchString(post.subject) && (!seen || now.UnixMilli()-sent >= int64(10*time.Hour/time.Millisecond)) {
			return post, true
		}
	}
	return pushPost{}, false
}

// pushLists hold the news lists a round of checks reads,
// for a minute, so groups checked together share them.
type pushLists struct {
	mu   sync.Mutex
	at   time.Time
	data map[string]any
}

func (p *pushLists) get(key string, read func() (any, error)) (any, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if time.Since(p.at) > time.Minute {
		p.at, p.data = time.Now(), map[string]any{}
	}
	if value, ok := p.data[key]; ok {
		return value, nil
	}
	value, err := read()
	if err == nil {
		p.data[key] = value
	}
	return value, err
}

// pushGroup runs a group's check: the post it gets, sent through send.
func (a *App) pushGroup(ctx context.Context, event *rayleabot.EventContext, sub *ContentSubscription, now time.Time, send func(...rayleabot.Segment) error) error {
	posts := []pushPost{}
	for _, kind := range pushKinds {
		if !slices.Contains(sub.Kinds, kind.kind) {
			continue
		}
		raw, err := a.pushLists.get(kind.kind, func() (any, error) {
			return a.Content.get(ctx, "https://bbs-api-static.miyoushe.com/painter/wapi/getNewsList?"+url.Values{"gids": {bbsGID}, "page_size": {"10"}, "type": {strconv.Itoa(kind.newsType)}}.Encode(), nil)
		})
		if err != nil {
			return err
		}
		for _, item := range asList(asObject(raw)["list"]) {
			post := asObject(asObject(item)["post"])
			created, _ := strconv.ParseInt(asText(post["created_at"]), 10, 64)
			posts = append(posts, pushPost{asText(post["post_id"]), asText(post["subject"]), kind.kind, created})
		}
	}
	for id, at := range sub.Sent {
		if now.UnixMilli()-at >= int64(10*time.Hour/time.Millisecond) {
			delete(sub.Sent, id)
		}
	}
	if post, ok := postToPush(posts, *sub, now); ok {
		if sub.Sent == nil {
			sub.Sent = map[string]int64{}
		}
		sub.Sent[post.id] = now.UnixMilli()
		name := "公告"
		if post.kind == "info" {
			name = "资讯"
		}
		if err := a.pushPost(ctx, event, post.id, newsGameName+name+"推送："+post.subject, send); err != nil {
			return err
		}
	}
	return nil
}

// pushPost draws a post after its line, or sends the line and the link when
// it cannot be drawn.
func (a *App) pushPost(ctx context.Context, event *rayleabot.EventContext, id, title string, send func(...rayleabot.Segment) error) error {
	data, err := a.Content.get(ctx, "https://bbs-api.miyoushe.com/post/wapi/getPostFull?"+url.Values{"gids": {bbsGID}, "read": {"1"}, "post_id": {id}}.Encode(), nil)
	if err != nil {
		return err
	}
	full := asObject(data["post"])
	if asText(asObject(full["post"])["post_id"]) != id {
		return gameError("public_invalid", "帖子编号与请求不一致。")
	}
	if settings(event).ImageReplies {
		image := a.newsPostImage(ctx, full)
		result, err := event.Actions().RenderImage(ctx, rayleabot.RenderImageRequest{Template: image.Template, Output: "jpeg", FallbackText: title, Data: image.Data, Resources: image.Resources})
		if path := asText(result["image_path"]); err == nil && path != "" {
			return send(rayleabot.Text(title), rayleabot.Image(path))
		}
	}
	return send(rayleabot.Text(title + "\nhttps://www.miyoushe.com/" + bbsPath + "/article/" + id))
}

// checkPushes runs the checks of the given subscriptions, or of all when ref
// is empty, sending to each group through its own bot.
func (a *App) checkPushes(ctx context.Context, event *rayleabot.EventContext, ref string) error {
	items, err := a.Subscriptions.List()
	if err != nil {
		return err
	}
	for _, item := range items {
		if ref != "" && item.Ref != ref {
			continue
		}
		code, online := "checked", false
		for _, bot := range event.Bots {
			online = online || bot.ID == item.Owner.BotID && bot.SourceAdapter == item.Owner.SourceAdapter && bot.SourceProtocol == item.Owner.SourceProtocol
		}
		if !online {
			code = "bot_missing"
		} else if config, err := a.Groups.Config(GroupScope{item.Owner.SourceProtocol, item.Owner.SourceAdapter, item.Owner.BotID, item.GroupID}); err != nil || config.Enabled != nil && !*config.Enabled {
			code = "group_disabled"
		} else {
			send := func(segments ...rayleabot.Segment) error {
				_, err := event.Actions().MessageSend(ctx, rayleabot.MessageSendRequest{SourceProtocol: item.Owner.SourceProtocol, SourceAdapter: item.Owner.SourceAdapter, TargetType: "group", TargetID: item.GroupID, Message: rayleabot.MessageOut{Segments: segments}})
				return err
			}
			if err := a.pushGroup(ctx, event, &item, time.Now(), send); err != nil {
				code = PublicError(err).Code
			}
		}
		err := a.Subscriptions.edit(item.Ref, func(items *[]ContentSubscription, i int) error {
			if i >= 0 {
				(*items)[i].Sent, (*items)[i].LastCode = item.Sent, code
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *App) subscriptionManage(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	if action == "content.subscription.list" {
		items, err := a.Subscriptions.List()
		return map[string]any{"items": items}, err
	}
	if action != "content.subscription.remove" || input["confirm"] != true {
		return nil, gameError("input_invalid", "请确认移除此推送。")
	}
	ref := asText(input["ref"])
	err := a.Subscriptions.edit(ref, func(items *[]ContentSubscription, i int) error {
		if i < 0 {
			return gameError("subscription_missing", "推送已不存在。")
		}
		*items = slices.Delete(*items, i, i+1)
		return nil
	})
	if err == nil {
		_, _ = event.Actions().SchedulerDelete(ctx, ref)
	}
	return map[string]any{"removed": err == nil}, err
}

// subscriptionCommand is Yunzai's setPush: a group administrator turns a push
// on or off for the group.
func (a *App) subscriptionCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	if command == "content-push" {
		// Every group's check may together take far longer than the event.
		// As upstream, the pushes are the only answer.
		if detached, err := detachChat(ctx, event); !detached {
			return err
		}
		if err := a.checkPushes(ctx, event, ""); err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.Result(map[string]any{"handled": true})
	}
	if event.Event.EventType != "message.group" {
		return event.SendText("推送请在群聊中设置")
	}
	if !groupAdministrator(event) {
		return event.SendText("暂无权限，只有管理员才能操作")
	}
	kind := pushKinds[0]
	if len(args) > 0 && args[0] == "资讯" {
		kind = pushKinds[1]
	}
	on := command == "subscribe"
	owner := Subject{event.Event.SourceProtocol, event.Event.SourceAdapter, event.Bot.ID, event.Event.Actor.ID}
	identity := strings.Join([]string{owner.SourceProtocol, owner.SourceAdapter, owner.BotID, event.Event.Target.ID}, "\x00")
	sum := sha256.Sum256([]byte(identity))
	ref := "game.content." + a.Game.ID + "." + hex.EncodeToString(sum[:16])
	created, emptied := false, false
	err := a.Subscriptions.edit(ref, func(items *[]ContentSubscription, i int) error {
		if i < 0 {
			if !on {
				return nil
			}
			*items = append(*items, ContentSubscription{Ref: ref, Owner: owner, GroupID: event.Event.Target.ID, Sent: map[string]int64{}})
			i, created = len(*items)-1, true
		}
		item := &(*items)[i]
		item.Kinds = slices.DeleteFunc(item.Kinds, func(k string) bool { return k == kind.kind })
		if on {
			item.Kinds = append(item.Kinds, kind.kind)
		} else if len(item.Kinds) == 0 {
			*items, emptied = slices.Delete(*items, i, i+1), true
		}
		return nil
	})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if created {
		if _, err := event.Actions().SchedulerCreate(ctx, a.contentJob(ref)); err != nil {
			_ = a.Subscriptions.edit(ref, func(items *[]ContentSubscription, i int) error {
				if i >= 0 {
					*items = slices.Delete(*items, i, i+1)
				}
				return nil
			})
			return event.SendText(friendlyError(err))
		}
	}
	if emptied {
		_, _ = event.Actions().SchedulerDelete(ctx, ref)
	}
	if on {
		return event.SendText(newsGameName + kind.name + "推送已开启\n如有最新" + kind.name + "将自动推送至此")
	}
	return event.SendText(newsGameName + kind.name + "推送已关闭")
}

// runContentSubscription checks a group on its scheduled task; a task whose
// group turned every push off is removed.
func (a *App) runContentSubscription(ctx context.Context, event *rayleabot.EventContext) error {
	if event.Event.SourceProtocol != "scheduler" || event.Event.SourceAdapter != "scheduler.internal" {
		return event.Fail("plugin.game_source_invalid", "任务来源无效。")
	}
	ref := event.Event.TaskID()
	items, err := a.Subscriptions.List()
	if err != nil {
		return event.Fail(PublicError(err).Code, PublicError(err).Message)
	}
	if !slices.ContainsFunc(items, func(item ContentSubscription) bool { return item.Ref == ref }) {
		_, _ = event.Actions().SchedulerDelete(ctx, ref)
		return event.Result(map[string]any{"checked": false})
	}
	if err := a.checkPushes(ctx, event, ref); err != nil {
		return event.Fail(PublicError(err).Code, PublicError(err).Message)
	}
	return event.Result(map[string]any{"checked": true})
}
