package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/gacha"
)

func syncOwner(event *rayleabot.EventContext) Subject {
	return Subject{SourceProtocol: event.Event.SourceProtocol, SourceAdapter: event.Event.SourceAdapter, BotID: event.Bot.ID, ActorID: event.Event.Actor.ID}
}
func syncTaskID(game, provider string, choice Selection) string {
	sum := sha256.Sum256([]byte(provider + "\x00" + choice.AccountRef + "\x00" + choice.RoleRef))
	return "game.sync." + game + "." + hex.EncodeToString(sum[:])
}
func (a *App) syncTaskAction(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	if action == "gacha.task.list" {
		items, err := a.SyncTasks.List()
		if event.Event.EventType != "management.action" {
			items = slices.DeleteFunc(items, func(t SyncTask) bool { return t.Owner != syncOwner(event) })
		}
		return map[string]any{"items": items}, err
	}
	if action == "gacha.task.remove" {
		var t SyncTask
		err := a.SyncTasks.edit(asText(input["ref"]), func(items *[]SyncTask, i int) error {
			if i < 0 {
				return gameError("sync_task_missing", "同步任务不存在。")
			}
			t = (*items)[i]
			if event.Event.EventType != "management.action" && t.Owner != syncOwner(event) {
				return gameError("source_invalid", "只能停止自己的同步任务。")
			}
			*items = slices.Delete(*items, i, i+1)
			return nil
		})
		if err != nil {
			return map[string]any{"removed": false, "delegation_revoked": false}, err
		}
		a.Syncs.Forget(t.Progress.Ref)
		client := AccountsClient{Caller: event.Actions(), Provider: t.Provider, Game: a.Game.ID}
		revoked := client.call(ctx, "delegation.revoke", map[string]any{"account_ref": t.AccountRef, "delegation_ref": t.DelegationRef}, nil) == nil
		return map[string]any{"removed": true, "delegation_revoked": revoked}, nil
	}
	if input["confirm"] != true {
		return nil, gameError("input_invalid", "请明确开启或重新运行后台同步。")
	}
	if action == "gacha.task.run" {
		ref := asText(input["ref"])
		items, err := a.SyncTasks.List()
		if err != nil {
			return nil, err
		}
		i := slices.IndexFunc(items, func(t SyncTask) bool { return t.Ref == ref })
		if i < 0 {
			return nil, gameError("sync_task_missing", "同步任务不存在。")
		}
		if event.Event.EventType != "management.action" && items[i].Owner != syncOwner(event) {
			return nil, gameError("source_invalid", "只能操作自己的同步任务。")
		}
		// The account check runs before the task is edited, outside the
		// store's lock.
		client := AccountsClient{Caller: event.Actions(), Provider: items[i].Provider, Game: a.Game.ID}
		_, role, err := client.Authorize(ctx, items[i].Selection)
		if err != nil {
			return nil, err
		}
		err = a.SyncTasks.edit(ref, func(items *[]SyncTask, i int) error {
			if i < 0 {
				return gameError("sync_task_missing", "同步任务不存在。")
			}
			t := &(*items)[i]
			if t.State == "running" || t.State == "creating" {
				return gameError("sync_task_running", "此任务正在处理，请查看进度。")
			}
			if t.ExpiresAtMS <= time.Now().UnixMilli() || t.DelegationRef == "" {
				return gameError("sync_task_expired", "委托已过期，请移除后重新创建任务。")
			}
			if role.UID != t.Role.UID || role.Region != t.Role.Region {
				return gameError("role_missing", "账号角色已变更，请重新创建任务。")
			}
			if !syncRegionAllowed(role.Region) {
				return gameError("sync_unavailable", "此区服暂不可同步完整记录。")
			}
			a.Syncs.Forget(t.Progress.Ref)
			t.Progress = gacha.SyncInfo{}
			t.State = "running"
			t.Failures = 0
			t.Restarts = 0
			t.NextCheckMS = 0
			t.LastCode = "queued"
			t.RunDay, _, _ = syncTaskTime(time.Now().UnixMilli(), t.Hour)
			return nil
		})
		return map[string]any{"queued": err == nil}, err
	}
	var q struct {
		Selection
		Kind   string `json:"kind"`
		Hour   int    `json:"hour"`
		Days   int    `json:"days"`
		Full   bool   `json:"full"`
		Notify bool   `json:"notify"`
	}
	if action != "gacha.task.create" || decodeObject(input, &q) != nil || (q.Kind != "once" && q.Kind != "daily") || q.Days < 1 || q.Days > 90 || q.Hour < 0 || q.Hour > 23 {
		return nil, gameError("input_invalid", "请选择单次/每日同步、1–90 天有效期和 0–23 时。")
	}
	client := a.accountClient(event)
	account, role, err := client.Authorize(ctx, q.Selection)
	if err != nil {
		return nil, err
	}
	if !syncRegionAllowed(role.Region) {
		return nil, gameError("region_unsupported", "此区服暂未适配后台同步。")
	}
	task := SyncTask{Ref: syncTaskID(a.Game.ID, client.Provider, q.Selection), Selection: q.Selection, Owner: account.Owner, Role: role, Provider: client.Provider, Kind: q.Kind, Hour: q.Hour, Full: q.Full, Notify: q.Notify, State: "creating"}
	// The task is kept as "creating" while the delegation and schedule are
	// made outside the store's lock, then enabled or dropped.
	err = a.SyncTasks.edit(task.Ref, func(items *[]SyncTask, i int) error {
		if len(*items) >= 256 {
			return gameError("sync_task_limit", "后台同步任务已达上限。")
		}
		if i >= 0 {
			return gameError("sync_task_exists", "此角色已有后台同步，请重新运行已有任务，或移除后调整设置。")
		}
		*items = append(*items, task)
		return nil
	})
	if err != nil {
		return nil, err
	}
	var grant struct {
		Delegation struct {
			Ref         string `json:"ref"`
			ExpiresAtMS int64  `json:"expires_at_ms"`
		} `json:"delegation"`
	}
	err = client.call(ctx, "delegation.create", map[string]any{"account_ref": task.AccountRef, "role_ref": task.RoleRef, "task_id": task.Ref, "operation": a.Game.ID + ".gacha", "days": q.Days}, &grant)
	if err == nil {
		task.DelegationRef = grant.Delegation.Ref
		task.ExpiresAtMS = grant.Delegation.ExpiresAtMS
		_, err = event.Actions().SchedulerCreate(ctx, rayleabot.SchedulerCreateRequest{TaskID: task.Ref, Cron: "* * * * *", LogLabel: a.Game.Name + "抽卡后台同步", Payload: map[string]any{"kind": "gacha_sync"}})
	}
	if err == nil {
		task.State = "waiting"
		task.LastCode = "queued"
		err = a.SyncTasks.edit(task.Ref, func(items *[]SyncTask, i int) error {
			if i < 0 {
				return gameError("sync_task_missing", "同步任务创建已取消。")
			}
			(*items)[i] = task
			return nil
		})
	}
	if err != nil {
		_ = a.SyncTasks.edit(task.Ref, func(items *[]SyncTask, i int) error {
			if i >= 0 {
				*items = slices.Delete(*items, i, i+1)
			}
			return nil
		})
		if task.DelegationRef != "" {
			_ = client.call(ctx, "delegation.revoke", map[string]any{"account_ref": task.AccountRef, "delegation_ref": task.DelegationRef}, nil)
		}
		return nil, err
	}
	return map[string]any{"task": task}, nil
}
func (a *App) runSyncTask(ctx context.Context, event *rayleabot.EventContext) error {
	if event.Event.SourceProtocol != "scheduler" || event.Event.SourceAdapter != "scheduler.internal" {
		return event.Fail("plugin.game_source_invalid", "任务来源无效。")
	}
	err := a.SyncTasks.Tick(ctx, asText(event.Event.Payload["task_id"]), time.Now().UnixMilli(), &a.Syncs, a.Gacha, func(ctx context.Context, task SyncTask, pool, end string, page int) (gacha.RemotePage, error) {
		client := AccountsClient{Caller: event.Actions(), Provider: task.Provider, Game: a.Game.ID}
		var response QueryResult
		err := client.call(ctx, "execute", map[string]any{"account_ref": task.AccountRef, "role_ref": task.RoleRef, "operation": a.Game.ID + ".gacha", "delegation_ref": task.DelegationRef, "input": map[string]any{"gacha_type": pool, "end_id": end, "page": page}}, &response)
		if err != nil {
			return gacha.RemotePage{}, err
		}
		if response.Role.UID != task.Role.UID || response.Role.Region != task.Role.Region || response.Role.Ref != task.Role.Ref {
			return gacha.RemotePage{}, gacha.ErrInvalid
		}
		return gacha.ParsePage(response.Role.UID, response.Role.Region, pool, end, response.Data)
	}, func(ctx context.Context, task SyncTask, text string) error {
		if !slices.ContainsFunc(event.Bots, func(bot rayleabot.Bot) bool {
			return bot.ID == task.Owner.BotID && bot.SourceProtocol == task.Owner.SourceProtocol && bot.SourceAdapter == task.Owner.SourceAdapter
		}) {
			return gameError("bot_missing", "所属机器人暂不可用。")
		}
		targetType, targetID := "private", task.Owner.ActorID
		if task.ReplyType != "" {
			targetType, targetID = task.ReplyType, task.ReplyID
		}
		_, err := event.Actions().MessageSend(ctx, rayleabot.MessageSendRequest{SourceProtocol: task.Owner.SourceProtocol, SourceAdapter: task.Owner.SourceAdapter, TargetType: targetType, TargetID: targetID, Message: rayleabot.MessageOut{Segments: []rayleabot.Segment{rayleabot.Text(text)}}})
		return err
	})
	if err != nil {
		e := PublicError(err)
		return event.Fail(e.Code, e.Message)
	}
	return event.Result(map[string]any{"checked": true})
}

// syncTaskCommand is ZZZ-Plugin's 刷新抽卡记录: the account's records are
// read on the role's background task, run again when it exists, and the
// chat it was sent in gets upstream's report when the round completes.
func (a *App) syncTaskCommand(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	uid := ""
	if len(args) > 1 {
		return event.SendText("请指定一个 UID，或省略以使用默认角色。")
	}
	if len(args) == 1 {
		uid = args[0]
	}
	client := a.accountClient(event)
	accounts, err := client.List(ctx, 0)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	choice, role, err := Choose(accounts, a.Game.ID, uid)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	ref := syncTaskID(a.Game.ID, client.Provider, choice)
	items, err := a.SyncTasks.List()
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if slices.ContainsFunc(items, func(t SyncTask) bool { return t.Ref == ref }) {
		_, err = a.syncTaskAction(ctx, event, "gacha.task.run", map[string]any{"ref": ref, "confirm": true})
		// A round already running answers here too.
		if err != nil && PublicError(err).Code != "plugin.game_sync_task_running" {
			return event.SendText(friendlyError(err))
		}
	} else if _, err = a.syncTaskAction(ctx, event, "gacha.task.create", map[string]any{"account_ref": choice.AccountRef, "role_ref": choice.RoleRef, "kind": "once", "hour": 8, "days": 7, "notify": false, "confirm": true}); err != nil {
		return event.SendText(friendlyError(err))
	}
	before := gachaPoolCounts(a.archiveOrEmpty(role.UID, role.Region))
	err = a.SyncTasks.edit(ref, func(items *[]SyncTask, i int) error {
		if i < 0 {
			return gameError("sync_task_missing", "同步任务不存在。")
		}
		(*items)[i].ReplyType, (*items)[i].ReplyID, (*items)[i].Before = event.Event.Target.Type, event.Event.Target.ID, before
		return nil
	})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText("抽卡记录获取中请稍等...可能需要一段时间，请耐心等待")
}
