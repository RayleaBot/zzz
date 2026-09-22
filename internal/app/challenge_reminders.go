package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func nextChallengeCheck(now int64, hour, minute, weekday int) int64 {
	current := time.UnixMilli(now).In(time.FixedZone("UTC+8", 28800))
	at := time.Date(current.Year(), current.Month(), current.Day(), hour, minute, 0, 0, current.Location())
	for !at.After(current) || (weekday != 0 && (int(at.Weekday())+6)%7+1 != weekday) {
		at = at.AddDate(0, 0, 1)
	}
	return at.UnixMilli()
}

func challengeUnstarted(data map[string]any) bool {
	for _, key := range []string{"has_data", "has_detail_record"} {
		if v, ok := data[key].(bool); ok && !v {
			return true
		}
	}
	return false
}

func challengeTarget(kind ChallengeKind, metric string, threshold float64, data map[string]any) (float64, bool, bool, error) {
	index := slices.IndexFunc(kind.Metrics, func(m ChallengeMetric) bool { return m.Key == metric })
	if index < 0 {
		return 0, false, false, gameError("input_invalid", "挑战指标不存在。")
	}
	entry, err := challengeExtract(kind, data)
	if err != nil {
		return 0, false, false, err
	}
	v, ok := entry.Metrics[metric]
	if !ok {
		return 0, false, false, nil
	}
	met := v >= threshold
	if kind.Metrics[index].Lower {
		met = v <= threshold
	}
	return v, met, true, nil
}
func (a *App) challengeReminder(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	if action == "challenge.reminder.list" {
		items, err := a.Reminders.List()
		items = slices.DeleteFunc(items, func(r Reminder) bool { return r.Kind != "challenge" })
		return map[string]any{"items": items}, err
	}
	if action == "challenge.reminder.remove" {
		return a.removeDelegatedTask(ctx, event, asText(input["ref"]), "challenge")
	}
	var q struct {
		Selection
		Kind      string   `json:"kind"`
		Metric    string   `json:"metric"`
		Threshold *float64 `json:"threshold"`
		Hour      int      `json:"hour"`
		Minute    int      `json:"minute"`
		Weekday   int      `json:"weekday"`
		Days      int      `json:"days"`
		Confirm   bool     `json:"confirm"`
	}
	if decodeObject(input, &q) != nil || q.Threshold == nil || !finiteRange(*q.Threshold, 0, 1e14) {
		return nil, gameError("input_invalid", "请提供明确的挑战指标和阈值。")
	}
	kind, ok := challengeKind(q.Kind)
	if !ok || !slices.ContainsFunc(kind.Metrics, func(m ChallengeMetric) bool { return m.Key == q.Metric }) {
		return nil, gameError("input_invalid", "挑战玩法或指标不存在。")
	}
	client := a.accountClient(event)
	if action == "challenge.reminder.check" {
		result, err := client.Execute(ctx, q.Selection, kind.Operation, nil)
		if err != nil {
			return nil, err
		}
		if challengeUnstarted(result.Data) {
			return map[string]any{"state": "not_started", "known": true, "met": false, "value": nil, "role": result.Role}, nil
		}
		value, met, known, err := challengeTarget(kind, q.Metric, *q.Threshold, result.Data)
		return map[string]any{"value": value, "met": met, "known": known, "role": result.Role}, err
	}
	if action != "challenge.reminder.create" || !q.Confirm || q.Days < 1 || q.Days > 90 || q.Hour < 0 || q.Hour > 23 || q.Minute < 0 || q.Minute > 59 || q.Weekday < 0 || q.Weekday > 7 {
		return nil, gameError("input_invalid", "请确认挑战提醒，并设置有效时间和 1–90 天期限。")
	}
	account, role, err := client.Authorize(ctx, q.Selection)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256([]byte(client.Provider + "\x00" + q.AccountRef + "\x00" + q.RoleRef + "\x00" + kind.ID))
	task := Reminder{Ref: "game.challenge." + a.Game.ID + "." + hex.EncodeToString(hash[:]), Selection: q.Selection, Owner: account.Owner, Role: role, Provider: client.Provider, Kind: "challenge", ChallengeKind: kind.ID, Metric: q.Metric, Threshold: *q.Threshold, Hour: q.Hour, Minute: q.Minute, Weekday: q.Weekday, NextCheckMS: nextChallengeCheck(time.Now().UnixMilli(), q.Hour, q.Minute, q.Weekday)}
	err = a.Reminders.edit(task.Ref, func(items *[]Reminder, i int) error {
		if i >= 0 {
			return gameError("reminder_exists", "此角色已有该玩法提醒，请先停止旧任务。")
		}
		if len(*items) >= 256 {
			return gameError("reminder_limit", "提醒数量已满。")
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
	err = client.call(ctx, "delegation.create", map[string]any{"account_ref": task.AccountRef, "role_ref": task.RoleRef, "task_id": task.Ref, "operation": kind.Operation, "days": q.Days}, &grant)
	if err == nil {
		task.DelegationRef = grant.Delegation.Ref
		task.ExpiresAtMS = grant.Delegation.ExpiresAtMS
		_, err = event.Actions().SchedulerCreate(ctx, rayleabot.SchedulerCreateRequest{TaskID: task.Ref, Cron: "*/5 * * * *", LogLabel: a.Game.Name + kind.Label + "挑战提醒", Payload: map[string]any{"kind": "challenge_reminder"}})
	}
	if err == nil {
		task.Enabled = true
		err = a.Reminders.edit(task.Ref, func(items *[]Reminder, i int) error {
			if i < 0 {
				return gameError("reminder_missing", "提醒创建已取消。")
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
func (s *ReminderStore) tickChallenge(task *Reminder, now int64, query func(Reminder) (QueryResult, error), send func(Reminder, string) error, game Game) error {
	kind, ok := challengeKind(task.ChallengeKind)
	if !ok {
		task.Enabled = false
		task.LastCode = "plugin.game_input_invalid"
		return s.save(*task)
	}
	task.LastCheckedMS = now
	task.NextCheckMS = nextChallengeCheck(now, task.Hour, task.Minute, task.Weekday)
	result, err := query(*task)
	if err != nil {
		task.LastCode = PublicError(err).Code
		switch task.LastCode {
		case "plugin.account_delegation_denied", "plugin.account_caller_denied", "plugin.account_not_found", "plugin.account_role_denied", "plugin.upstream_auth_invalid":
			task.Enabled = false
		case "plugin.upstream_device_required", "plugin.upstream_challenge_required":
			task.NextCheckMS = max(task.NextCheckMS, now+int64(24*time.Hour/time.Millisecond))
		}
		return s.save(*task)
	}
	value, met, known, err := challengeTarget(kind, task.Metric, task.Threshold, result.Data)
	if challengeUnstarted(result.Data) {
		err = nil
		known = true
		met = false
	}
	if err != nil {
		task.LastCode = PublicError(err).Code
		return s.save(*task)
	}
	if !known {
		task.LastCode = "metric_unknown"
		return s.save(*task)
	}
	if met {
		task.LastCode = "target_met"
		return s.save(*task)
	}
	task.LastAttemptMS = now
	task.LastCode = "notification_attempted"
	if err = s.save(*task); err != nil {
		return err
	}
	label := task.Metric
	for _, m := range kind.Metrics {
		if m.Key == task.Metric {
			label = m.Label
		}
	}
	message := fmt.Sprintf("%s%s挑战提醒\n%s · UID %s\n%s：%.2f，尚未达到你设置的 %.2f 目标。\n发送“%s关闭挑战提醒”可停止。", game.Name, kind.Label, task.Role.Nickname, task.Role.UID, label, value, task.Threshold, game.Prefix)
	if challengeUnstarted(result.Data) {
		message = fmt.Sprintf("%s%s挑战提醒\n%s · UID %s\n本期尚无完成记录。发送“%s关闭挑战提醒”可停止。", game.Name, kind.Label, task.Role.Nickname, task.Role.UID, game.Prefix)
	}
	if err = send(*task, message); err != nil {
		task.LastCode = "notification_failed"
	} else {
		task.LastCode = "notified"
	}
	return s.save(*task)
}
func (a *App) challengeReminderCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	if command == "challenge-stop" {
		if len(args) > 1 {
			return event.SendText("格式：关闭挑战提醒 [UID]。")
		}
		items, err := a.Reminders.List()
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		owner := Subject{event.Event.SourceProtocol, event.Event.SourceAdapter, event.Bot.ID, event.Event.Actor.ID}
		n := 0
		for _, task := range items {
			if task.Kind == "challenge" && task.Owner == owner && (len(args) == 0 || args[0] == task.Role.UID) {
				if _, err = a.removeDelegatedTask(ctx, event, task.Ref, "challenge"); err != nil {
					return event.SendText(friendlyError(err))
				}
				n++
			}
		}
		return event.SendText(fmt.Sprintf("已停止 %d 项挑战提醒。", n))
	}
	if len(args) < 3 || len(args) > 5 {
		return event.SendText("格式：" + a.Game.Prefix + "挑战提醒 玩法 指标 阈值 [HH:MM] [UID]。默认每日北京时间 20:00，30 天。玩法及指标可在管理页查看。")
	}
	hour, minute, uid := 20, 0, ""
	threshold, err := strconv.ParseFloat(args[2], 64)
	if err != nil {
		return event.SendText("阈值必须为数值。")
	}
	if len(args) >= 4 {
		parts := strings.Split(args[3], ":")
		if len(parts) != 2 {
			return event.SendText("提醒时间请使用 HH:MM。")
		}
		hour, err = strconv.Atoi(parts[0])
		if err != nil {
			return event.SendText("提醒时间无效。")
		}
		minute, err = strconv.Atoi(parts[1])
		if err != nil {
			return event.SendText("提醒时间无效。")
		}
	}
	if len(args) == 5 {
		uid = args[4]
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
	_, err = a.challengeReminder(ctx, event, "challenge.reminder.create", map[string]any{"account_ref": choice.AccountRef, "role_ref": choice.RoleRef, "kind": args[0], "metric": args[1], "threshold": threshold, "hour": hour, "minute": minute, "days": 30, "confirm": true})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText("挑战提醒已开启，有效 30 天；未达目标时私聊通知。可在管理页调整每日/每周时间，或发送“" + a.Game.Prefix + "关闭挑战提醒”停止。")
}
