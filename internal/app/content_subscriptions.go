package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/localdata"
)

type ContentSubscription struct {
	Ref         string   `json:"ref"`
	Owner       Subject  `json:"owner"`
	TargetType  string   `json:"target_type"`
	TargetID    string   `json:"target_id"`
	Kind        string   `json:"kind"`
	Enabled     bool     `json:"enabled"`
	ExpiresMS   int64    `json:"expires_ms"`
	NextMS      int64    `json:"next_ms"`
	Initialized bool     `json:"initialized"`
	Seen        []string `json:"seen"`
	LastCode    string   `json:"last_code"`
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
func (s *ContentSubscriptions) Tick(ref string, now int64, fetch func(ContentSubscription) (map[string]any, error), send func(ContentSubscription, string) error, game Game) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return err
	}
	i := slices.IndexFunc(items, func(v ContentSubscription) bool { return v.Ref == ref })
	if i < 0 {
		return nil
	}
	task := &items[i]
	if !task.Enabled || task.NextMS > now {
		return nil
	}
	task.NextMS = now + int64(15*time.Minute/time.Millisecond)
	if task.ExpiresMS <= now {
		task.Enabled = false
		task.LastCode = "expired"
		if err = localdata.Write(s.Path, items); err != nil {
			return err
		}
		if err = send(*task, game.Name+"订阅已到期，可重新发送订阅指令继续接收。"); err != nil {
			task.LastCode = "expiry_notification_failed"
		}
		return localdata.Write(s.Path, items)
	}
	if err = localdata.Write(s.Path, items); err != nil {
		return err
	}
	data, err := fetch(*task)
	if err != nil {
		task.LastCode = PublicError(err).Code
		return localdata.Write(s.Path, items)
	}
	keys := []string{}
	messages := []string{}
	if task.Kind == "expiry" {
		for _, v := range data["items"].([]PublicActivity) {
			key := v.ID + ":" + strconv.FormatInt(v.EndMS, 10)
			if (v.TimeStatus != "explicit" && v.TimeStatus != "explicit_end") || v.EndMS <= now || v.EndMS > now+int64(24*time.Hour/time.Millisecond) {
				continue
			}
			keys = append(keys, key)
			if !slices.Contains(task.Seen, key) {
				messages = append(messages, v.Title+"\n截止："+v.End)
			}
		}
	} else {
		for _, v := range data["items"].([]PublicPost) {
			keys = append(keys, v.ID)
			if task.Initialized && !slices.Contains(task.Seen, v.ID) {
				messages = append(messages, v.Title+"\n"+v.URL)
			}
		}
	}
	task.Initialized = true
	for _, key := range keys {
		if !slices.Contains(task.Seen, key) {
			task.Seen = append(task.Seen, key)
		}
	}
	if len(task.Seen) > 1000 {
		task.Seen = slices.Clone(task.Seen[len(task.Seen)-1000:])
	}
	task.LastCode = "checked"
	if len(messages) > 0 {
		task.LastCode = "notification_attempted"
	}
	if err = localdata.Write(s.Path, items); err != nil {
		return err
	}
	if len(messages) > 0 {
		body := game.Name + " · " + contentKindLabel(task.Kind) + "订阅\n" + strings.Join(messages, "\n\n")
		if err = send(*task, body); err != nil {
			task.LastCode = "notification_failed"
		} else {
			task.LastCode = "notified"
		}
		return localdata.Write(s.Path, items)
	}
	return nil
}
func contentKindLabel(kind string) string {
	return map[string]string{"news": "公告", "info": "资讯", "events": "活动", "expiry": "到期"}[kind]
}
func (a *App) subscriptionManage(action string, input map[string]any) (map[string]any, error) {
	if action == "content.subscription.list" {
		items, err := a.Subscriptions.List()
		return map[string]any{"items": items}, err
	}
	if action != "content.subscription.remove" || input["confirm"] != true {
		return nil, gameError("input_invalid", "请确认移除此订阅。")
	}
	err := a.Subscriptions.edit(asText(input["ref"]), func(items *[]ContentSubscription, i int) error {
		if i < 0 {
			return gameError("subscription_missing", "订阅已不存在。")
		}
		*items = slices.Delete(*items, i, i+1)
		return nil
	})
	return map[string]any{"removed": err == nil}, err
}
func (a *App) subscriptionCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	group := event.Event.EventType == "message.group"
	if group && !groupAdministrator(event) {
		return event.SendText("群订阅需要群管理员操作。")
	}
	kind := ""
	if len(args) > 0 {
		for _, k := range []string{"news", "info", "events", "expiry"} {
			if args[0] == contentKindLabel(k) {
				kind = k
			}
		}
	}
	remove := command == "unsubscribe"
	if !remove && kind == "" || len(args) > 2 || remove && len(args) > 1 || remove && len(args) == 1 && kind == "" {
		return event.SendText("使用“" + a.Game.Prefix + "订阅 公告/资讯/活动/到期 [1–90天]”或“" + a.Game.Prefix + "退订 [类型]”。首次检查不推送历史新闻，到期预警为未来24小时。")
	}
	owner := Subject{event.Event.SourceProtocol, event.Event.SourceAdapter, event.Bot.ID, event.Event.Actor.ID}
	targetType, targetID := "private", owner.ActorID
	if group {
		targetType, targetID = "group", event.Event.Target.ID
	}
	if remove {
		count := 0
		err := a.Subscriptions.edit("", func(items *[]ContentSubscription, _ int) error {
			*items = slices.DeleteFunc(*items, func(v ContentSubscription) bool {
				yes := v.Owner.SourceProtocol == owner.SourceProtocol && v.Owner.SourceAdapter == owner.SourceAdapter && v.Owner.BotID == owner.BotID && v.TargetType == targetType && v.TargetID == targetID && (kind == "" || v.Kind == kind)
				if yes {
					count++
				}
				return yes
			})
			return nil
		})
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.SendText(fmt.Sprintf("已移除 %d 个%s订阅。", count, a.Game.Name))
	}
	days := 30
	if len(args) == 2 {
		var err error
		days, err = strconv.Atoi(strings.TrimSuffix(args[1], "天"))
		if err != nil || days < 1 || days > 90 {
			return event.SendText("订阅期限应为1–90天。")
		}
	}
	identity, _ := json.Marshal([]string{owner.SourceProtocol, owner.SourceAdapter, owner.BotID, targetType, targetID, kind})
	sum := sha256.Sum256(identity)
	task := ContentSubscription{Ref: "game.content." + a.Game.ID + "." + hex.EncodeToString(sum[:]), Owner: owner, TargetType: targetType, TargetID: targetID, Kind: kind, ExpiresMS: time.Now().Add(time.Duration(days) * 24 * time.Hour).UnixMilli(), Seen: []string{}}
	err := a.Subscriptions.edit(task.Ref, func(items *[]ContentSubscription, i int) error {
		if i >= 0 {
			return gameError("subscription_exists", "已有此订阅，请先退订后重新开启。")
		}
		if len(*items) >= 256 {
			return gameError("subscription_limit", "订阅已达256项上限。")
		}
		*items = append(*items, task)
		return nil
	})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	_, err = event.Actions().SchedulerCreate(ctx, rayleabot.SchedulerCreateRequest{TaskID: task.Ref, Cron: "*/15 * * * *", LogLabel: a.Game.Name + "公开内容订阅", Payload: map[string]any{"kind": "public_content"}})
	if err != nil {
		_ = a.Subscriptions.edit(task.Ref, func(items *[]ContentSubscription, i int) error {
			if i >= 0 {
				*items = slices.Delete(*items, i, i+1)
			}
			return nil
		})
		return event.SendText(friendlyError(err))
	}
	err = a.Subscriptions.edit(task.Ref, func(items *[]ContentSubscription, i int) error {
		if i < 0 {
			return gameError("subscription_missing", "订阅已取消。")
		}
		(*items)[i].Enabled = true
		return nil
	})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText(fmt.Sprintf("已开启%s%s订阅，%d天后到期。首次检查建立新闻基线；到期提醒仅采用明确日期。", a.Game.Name, contentKindLabel(kind), days))
}
func (a *App) runContentSubscription(ctx context.Context, event *rayleabot.EventContext) error {
	if event.Event.SourceProtocol != "scheduler" || event.Event.SourceAdapter != "scheduler.internal" {
		return event.Fail("plugin.game_source_invalid", "任务来源无效。")
	}
	err := a.Subscriptions.Tick(asText(event.Event.Payload["task_id"]), time.Now().UnixMilli(), func(task ContentSubscription) (map[string]any, error) {
		if task.TargetType == "group" {
			config, err := a.Groups.Config(GroupScope{task.Owner.SourceProtocol, task.Owner.SourceAdapter, task.Owner.BotID, task.TargetID})
			if err != nil {
				return nil, err
			}
			if config.Enabled != nil && !*config.Enabled {
				return nil, gameError("group_disabled", "此群已暂停游戏功能。")
			}
		}
		if task.Kind == "expiry" {
			return a.Content.calendar(ctx)
		}
		return a.Content.posts(ctx, ContentQuery{Kind: "news", Type: map[string]int{"news": 1, "events": 2, "info": 3}[task.Kind]})
	}, func(task ContentSubscription, text string) error {
		found := false
		for _, bot := range event.Bots {
			if bot.ID == task.Owner.BotID && bot.SourceAdapter == task.Owner.SourceAdapter && bot.SourceProtocol == task.Owner.SourceProtocol {
				found = true
			}
		}
		if !found {
			return gameError("bot_missing", "订阅机器人当前不可用。")
		}
		if task.TargetType == "group" {
			config, err := a.Groups.Config(GroupScope{task.Owner.SourceProtocol, task.Owner.SourceAdapter, task.Owner.BotID, task.TargetID})
			if err != nil {
				return err
			}
			if config.Enabled != nil && !*config.Enabled {
				return gameError("group_disabled", "群功能已暂停。")
			}
		}
		_, err := event.Actions().MessageSend(ctx, rayleabot.MessageSendRequest{SourceProtocol: task.Owner.SourceProtocol, SourceAdapter: task.Owner.SourceAdapter, TargetType: task.TargetType, TargetID: task.TargetID, Message: rayleabot.MessageOut{Segments: []rayleabot.Segment{rayleabot.Text(text)}}})
		return err
	}, a.Game)
	if err != nil {
		return event.Fail(PublicError(err).Code, PublicError(err).Message)
	}
	return event.Result(map[string]any{"checked": true})
}
