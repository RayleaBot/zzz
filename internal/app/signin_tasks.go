package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"slices"
	"strings"
	"time"
)

func (a *App) signinTask(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	kind, root, operation, label, taskPrefix := "signin", "signin.task.", a.Game.ID+".sign", "每日签到", "game.sign."
	if strings.HasPrefix(action, "monthly.task.") {
		kind, root, operation, label, taskPrefix = "monthly", "monthly.task.", a.Game.ID+".monthly", "每日月报收集", "game.monthly."
	}
	if strings.HasPrefix(action, "community.task.") {
		kind, root, operation, label, taskPrefix = "community", "community.task.", a.Game.ID+".community_run", "米游社任务", "game.community."
	}
	if strings.HasPrefix(action, "cloudgame.task.") {
		if a.Game.ID == "starrail" {
			return nil, gameError("operation_denied", "当前没有星铁云游戏任务。")
		}
		kind, root, operation, label, taskPrefix = "cloudgame", "cloudgame.task.", a.Game.ID+".cloud_sign", "云游戏签到", "game.cloudgame."
	}
	if action == root+"list" {
		items, err := a.Reminders.List()
		items = slices.DeleteFunc(items, func(task Reminder) bool { return task.Kind != kind })
		return map[string]any{"items": items}, err
	}
	if action == root+"remove" {
		return a.removeDelegatedTask(ctx, event, asText(input["ref"]), kind)
	}
	var q struct {
		Selection
		Days    int  `json:"days"`
		Once    bool `json:"once"`
		Read    bool `json:"read"`
		Like    bool `json:"like"`
		Share   bool `json:"share"`
		Unlike  bool `json:"unlike"`
		Hour    int  `json:"hour"`
		Minute  int  `json:"minute"`
		Notify  bool `json:"notify"`
		Confirm bool `json:"confirm"`
	}
	if action != root+"create" || decodeObject(input, &q) != nil || !q.Confirm || q.Days < 1 || q.Days > 90 || q.Hour < 0 || q.Hour > 23 || q.Minute < 0 || q.Minute > 59 {
		return nil, gameError("input_invalid", label+"需要明确开启，并选择有效时间和 1–90 天。")
	}
	client := a.accountClient(event)
	var account Account
	var role Role
	var err error
	if kind == "community" || kind == "cloudgame" {
		q.RoleRef = ""
		account, err = client.AuthorizeAccount(ctx, q.AccountRef)
		role = Role{Game: a.Game.ID, Nickname: account.AccountLabel}
	} else {
		account, role, err = client.Authorize(ctx, q.Selection)
	}
	if err != nil {
		return nil, err
	}
	if kind == "cloudgame" && !slices.Contains(account.CloudConfigured, a.Game.ID) {
		return nil, gameError("input_invalid", "请先在账号管理中配置对应云游戏授权。")
	}
	sum := sha256.Sum256([]byte(client.Provider + "\x00" + q.AccountRef + "\x00" + q.RoleRef))
	task := Reminder{Ref: taskPrefix + a.Game.ID + "." + hex.EncodeToString(sum[:]), Selection: q.Selection, Owner: account.Owner, Role: role, Provider: client.Provider, Kind: kind, Once: q.Once, Community: CommunityPlan{Read: q.Read, Like: q.Like, Share: q.Share, Unlike: q.Unlike}, Hour: q.Hour, Minute: q.Minute, Notify: q.Notify}
	err = a.Reminders.edit(task.Ref, func(items *[]Reminder, _ int) error {
		if len(*items) >= 256 {
			return gameError("reminder_limit", "任务数量已达上限。")
		}
		for _, old := range *items {
			if old.Selection == task.Selection && old.Provider == task.Provider && old.Kind == kind {
				return gameError("reminder_exists", "此角色已有"+label+"，请先停止旧任务。")
			}
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
	err = client.call(ctx, "delegation.create", map[string]any{"account_ref": task.AccountRef, "role_ref": task.RoleRef, "task_id": task.Ref, "operation": operation, "days": q.Days, "write_confirmed": kind != "monthly"}, &grant)
	if err == nil {
		task.DelegationRef = grant.Delegation.Ref
		task.ExpiresAtMS = grant.Delegation.ExpiresAtMS
		cron := "*/10 * * * *"
		if kind == "community" || kind == "cloudgame" {
			cron = "* * * * *"
		}
		_, err = event.Actions().SchedulerCreate(ctx, rayleabot.SchedulerCreateRequest{TaskID: task.Ref, Cron: cron, LogLabel: a.Game.Name + label, Payload: map[string]any{"kind": kind}})
	}
	if err == nil {
		task.Enabled = true
		if kind == "monthly" || ((kind == "community" || kind == "cloudgame") && !q.Once) {
			task.NextCheckMS = nextChallengeCheck(time.Now().UnixMilli(), q.Hour, q.Minute, 0)
		}
		err = a.Reminders.edit(task.Ref, func(items *[]Reminder, i int) error {
			if i < 0 {
				return gameError("reminder_missing", "自动签到创建已取消。")
			}
			(*items)[i] = task
			return nil
		})
	}
	if err != nil {
		_ = a.Reminders.edit(task.Ref, func(items *[]Reminder, i int) error {
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
func (s *ReminderStore) tickSignin(task *Reminder, now int64, query func(Reminder) (QueryResult, error), send func(Reminder, string) error, game Game) error {
	china := time.UnixMilli(now).In(time.FixedZone("UTC+8", 8*60*60))
	today := china.Format("2006-01-02")
	at := time.Date(china.Year(), china.Month(), china.Day(), task.Hour, 0, 0, 0, china.Location())
	if china.Before(at) {
		task.NextCheckMS = at.UnixMilli()
		return s.save(*task)
	}
	tomorrow := at.AddDate(0, 0, 1).UnixMilli()
	if task.LastSignDay == today {
		task.NextCheckMS = tomorrow
		return s.save(*task)
	}
	task.LastSignDay = today
	task.LastCheckedMS = now
	task.NextCheckMS = tomorrow
	task.LastCode = "sign_attempted"
	if err := s.save(*task); err != nil {
		return err
	}
	result, err := query(*task)
	message := ""
	if err != nil {
		task.LastCode = PublicError(err).Code
		message = friendlyError(err)
		switch task.LastCode {
		case "plugin.service_unavailable", "plugin.vault_locked", "plugin.vault_missing":
			task.LastSignDay = ""
			task.NextCheckMS = now + int64(time.Hour/time.Millisecond)
		case "plugin.account_delegation_throttled":
			task.LastSignDay = ""
			task.NextCheckMS = now + int64(10*time.Minute/time.Millisecond)
		case "plugin.account_delegation_denied", "plugin.account_caller_denied", "plugin.account_not_found", "plugin.account_role_denied", "plugin.upstream_auth_invalid", "plugin.upstream_first_sign_required":
			task.Enabled = false
		case "plugin.upstream_device_required", "plugin.upstream_challenge_required":
			task.NextCheckMS = max(task.NextCheckMS, now+int64(24*time.Hour/time.Millisecond))
		}
	} else if result.Data["signed"] == true {
		task.LastCode = asText(result.Data["outcome"])
		if task.LastCode == "already_signed" {
			message = "今日已签到。"
		} else {
			task.LastCode = "signed"
			message = "签到成功。"
		}
	} else {
		task.LastCode = "plugin.game_sign_unknown"
		message = "本次签到结果未能确认，请手动查询状态。"
	}
	if task.Notify {
		task.LastAttemptMS = now
	}
	if err := s.save(*task); err != nil {
		return err
	}
	if task.Notify {
		if err := send(*task, game.Name+"自动签到\n"+task.Role.Nickname+" · "+task.Role.UID+"\n"+message); err != nil {
			task.LastCode += ".notification_failed"
			return s.save(*task)
		}
	}
	return nil
}
func (a *App) signinTaskCommand(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	if len(args) < 1 || len(args) > 2 || (args[0] != "开启" && args[0] != "关闭") {
		return event.SendText("使用“" + a.Game.Prefix + "自动签到 开启/关闭 [UID]”。默认每天北京时间 08:00 后检查，有效 30 天，不发送私聊结果。")
	}
	uid := ""
	if len(args) == 2 {
		uid = args[1]
	}
	if args[0] == "关闭" {
		items, err := a.Reminders.List()
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		owner := Subject{SourceProtocol: event.Event.SourceProtocol, SourceAdapter: event.Event.SourceAdapter, BotID: event.Bot.ID, ActorID: event.Event.Actor.ID}
		count := 0
		for _, task := range items {
			if task.Kind == "signin" && task.Owner == owner && (uid == "" || uid == task.Role.UID) {
				if _, err = a.signinTask(ctx, event, "signin.task.remove", map[string]any{"ref": task.Ref}); err != nil {
					return event.SendText(friendlyError(err))
				}
				count++
			}
		}
		return event.SendText(fmt.Sprintf("已停止 %d 个%s自动签到任务。", count, a.Game.Name))
	}
	client := a.accountClient(event)
	accounts, err := client.List(ctx, 0)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	choice, _, err := Choose(accounts, a.Game.ID, uid)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	_, err = a.signinTask(ctx, event, "signin.task.create", map[string]any{"account_ref": choice.AccountRef, "role_ref": choice.RoleRef, "days": 30, "hour": 8, "confirm": true, "notify": false})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText("已开启" + a.Game.Name + "每日自动签到，默认北京时间 08:00 后检查，有效 30 天。可在管理页查看结果，或发送“" + a.Game.Prefix + "自动签到 关闭”停止。")
}
