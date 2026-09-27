package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
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
	// As ZZZ-Plugin's 式舆阈值, 6 asks for all five floors with S and S+ on
	// the fifth.
	if metric == "s_layers" {
		met = v >= min(threshold, 5) && (threshold < 6 || entry.Metrics["rating"] == 5)
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
	var q challengeRequest
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
	task, err := a.createChallengeReminder(ctx, client, eventScheduler(event), kind, q)
	if err != nil {
		return nil, err
	}
	return map[string]any{"task": task}, nil
}

// taskScheduler creates the host schedule of a task.
type taskScheduler func(context.Context, rayleabot.SchedulerCreateRequest) error

// eventScheduler schedules through the event's host actions.
func eventScheduler(event *rayleabot.EventContext) taskScheduler {
	return func(ctx context.Context, request rayleabot.SchedulerCreateRequest) error {
		_, err := event.Actions().SchedulerCreate(ctx, request)
		return err
	}
}

// challengeRequest is a challenge reminder to create or check: the account
// role, the kind and metric, the target, the Beijing time and the days it
// runs. Pair marks one of the two reminders 开启挑战提醒 creates.
type challengeRequest struct {
	Selection
	Kind      string   `json:"kind"`
	Metric    string   `json:"metric"`
	Threshold *float64 `json:"threshold"`
	Hour      int      `json:"hour"`
	Minute    int      `json:"minute"`
	Weekday   int      `json:"weekday"`
	Days      int      `json:"days"`
	Confirm   bool     `json:"confirm"`
	// Pair creates 开启挑战提醒's task, which also checks 危局强袭战 against
	// DeadlyThreshold.
	Pair            bool    `json:"-"`
	DeadlyThreshold float64 `json:"-"`
}

// createChallengeReminder keeps a checked request as a task with its own
// delegations and schedule: one delegation for the kind's operation, and
// for a pair a second one for 危局强袭战 under the same task. A failed step
// removes what was set up.
func (a *App) createChallengeReminder(ctx context.Context, client AccountsClient, schedule taskScheduler, kind ChallengeKind, q challengeRequest) (Reminder, error) {
	account, role, err := client.Authorize(ctx, q.Selection)
	if err != nil {
		return Reminder{}, err
	}
	key := client.Provider + "\x00" + q.AccountRef + "\x00" + q.RoleRef + "\x00" + kind.ID
	operations := []string{kind.Operation}
	if q.Pair {
		key = client.Provider + "\x00" + q.AccountRef + "\x00" + q.RoleRef + "\x00pair"
		operations = append(operations, a.Game.ID+".deadly")
	}
	hash := sha256.Sum256([]byte(key))
	task := Reminder{Ref: "game.challenge." + a.Game.ID + "." + hex.EncodeToString(hash[:]), Selection: q.Selection, Owner: account.Owner, Role: role, Provider: client.Provider, Kind: "challenge", ChallengeKind: kind.ID, Metric: q.Metric, Threshold: *q.Threshold, Pair: q.Pair, Hour: q.Hour, Minute: q.Minute, Weekday: q.Weekday, NextCheckMS: nextChallengeCheck(time.Now().UnixMilli(), q.Hour, q.Minute, q.Weekday)}
	if q.Pair {
		task.DeadlyThreshold = q.DeadlyThreshold
	}
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
		return Reminder{}, err
	}
	for index, operation := range operations {
		var grant struct {
			Delegation struct {
				Ref         string `json:"ref"`
				ExpiresAtMS int64  `json:"expires_at_ms"`
			} `json:"delegation"`
		}
		if err = client.call(ctx, "delegation.create", map[string]any{"account_ref": task.AccountRef, "role_ref": task.RoleRef, "task_id": task.Ref, "operation": operation, "days": q.Days}, &grant); err != nil {
			break
		}
		if index == 0 {
			task.DelegationRef, task.ExpiresAtMS = grant.Delegation.Ref, grant.Delegation.ExpiresAtMS
		} else {
			task.DeadlyDelegationRef, task.ExpiresAtMS = grant.Delegation.Ref, min(task.ExpiresAtMS, grant.Delegation.ExpiresAtMS)
		}
	}
	if err == nil {
		label := kind.Label
		if q.Pair {
			label = "式舆/危局"
		}
		err = schedule(ctx, rayleabot.SchedulerCreateRequest{TaskID: task.Ref, Cron: "*/5 * * * *", LogLabel: a.Game.Name + label + "挑战提醒", Payload: taskPayload("challenge_reminder", task.Ref)})
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
		for _, ref := range []string{task.DelegationRef, task.DeadlyDelegationRef} {
			if ref != "" {
				_ = client.call(ctx, "delegation.revoke", map[string]any{"account_ref": task.AccountRef, "delegation_ref": ref}, nil)
			}
		}
		return Reminder{}, err
	}
	return task, nil
}

// failCheck records a failed check: revoked access pauses the task, and an
// official verification waits at least a day.
func (task *Reminder) failCheck(now int64, err error) {
	task.LastCode = PublicError(err).Code
	switch task.LastCode {
	case "plugin.account_delegation_denied", "plugin.account_caller_denied", "plugin.account_not_found", "plugin.account_role_denied", "plugin.upstream_auth_invalid":
		task.Enabled = false
	case "plugin.upstream_device_required", "plugin.upstream_challenge_required":
		task.NextCheckMS = max(task.NextCheckMS, now+int64(24*time.Hour/time.Millisecond))
	}
}

func (s *ReminderStore) tickChallenge(task *Reminder, now int64, query func(Reminder) (QueryResult, error), send func(Reminder, string) error, game Game) error {
	if task.Pair {
		return s.tickChallengePair(task, now, query, send)
	}
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
		task.failCheck(now, err)
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
	owner := Subject{event.Event.SourceProtocol, event.Event.SourceAdapter, event.Bot.ID, event.Event.Actor.ID}
	if command == "challenge-stop" || command == "challenge-status" {
		if len(args) > 1 {
			return event.SendText("格式：" + a.Game.Prefix + map[string]string{"challenge-stop": "关闭挑战提醒", "challenge-status": "挑战提醒状态"}[command] + " [UID]。")
		}
		items, err := a.Reminders.List()
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		lines := []string{}
		for _, task := range items {
			if task.Kind != "challenge" || task.Owner != owner || len(args) == 1 && args[0] != task.Role.UID {
				continue
			}
			if command == "challenge-status" {
				lines = append(lines, challengeReminderLine(task))
				continue
			}
			if _, err = a.removeDelegatedTask(ctx, event, task.Ref, "challenge"); err != nil {
				return event.SendText(friendlyError(err))
			}
			lines = append(lines, task.Ref)
		}
		if command == "challenge-stop" {
			// Without a UID it is ZZZ-Plugin's 关闭挑战提醒.
			switch {
			case len(args) == 1:
				return event.SendText(fmt.Sprintf("已停止 %d 项挑战提醒。", len(lines)))
			case len(lines) == 0:
				return event.SendText("提醒功能尚未开启")
			}
			return event.SendText("提醒功能已关闭")
		}
		if len(lines) == 0 {
			return event.SendText("尚未开启挑战提醒。\n" + a.challengeUsage())
		}
		return event.SendText(a.Game.Name + "挑战提醒\n" + strings.Join(lines, "\n") + "\n发送“" + a.Game.Prefix + "关闭挑战提醒 [UID]”停止。")
	}
	if len(args) < 3 || len(args) > 5 {
		return event.SendText(a.challengeUsage())
	}
	kind, ok := challengeKind(args[0])
	if !ok {
		return event.SendText("挑战玩法不存在。\n" + a.challengeUsage())
	}
	metric, ok := challengeMetric(kind, args[1])
	if !ok {
		return event.SendText(kind.Label + "没有此指标。\n" + a.challengeUsage())
	}
	threshold, err := strconv.ParseFloat(args[2], 64)
	if err != nil {
		return event.SendText("阈值必须为数值。")
	}
	hour, minute, weekday, uid := 20, 0, 0, ""
	if len(args) >= 4 {
		if hour, minute, weekday, ok = parseChallengeTime(args[3]); !ok {
			return event.SendText("提醒时间请写作 HH:MM、每日20时30分或每周一20时。")
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
	_, err = a.challengeReminder(ctx, event, "challenge.reminder.create", map[string]any{"account_ref": choice.AccountRef, "role_ref": choice.RoleRef, "kind": kind.ID, "metric": metric.Key, "threshold": threshold, "hour": hour, "minute": minute, "weekday": weekday, "days": 30, "confirm": true})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText("挑战提醒已开启，" + challengeSchedule(hour, minute, weekday) + "检查，有效 30 天；未达目标时私聊通知。发送“" + a.Game.Prefix + "挑战提醒状态”查看，或“" + a.Game.Prefix + "关闭挑战提醒”停止。")
}

// challengeUsage lists the reminder's form with every kind and its metrics.
func (a *App) challengeUsage() string {
	lines := []string{"格式：" + a.Game.Prefix + "挑战提醒 玩法 指标 阈值 [时间] [UID]", "时间写作 HH:MM、每日20时30分或每周一20时，默认每日 20:00；有效 30 天。", "发送“" + a.Game.Prefix + "开启挑战提醒”则按个人或全局阈值同时检查式舆防卫战与危局强袭战。", "可选玩法与指标："}
	for _, kind := range challengeKinds() {
		names := []string{}
		for _, metric := range kind.Metrics {
			names = append(names, metric.Label)
		}
		lines = append(lines, kind.Label+"："+strings.Join(names, "、"))
	}
	return strings.Join(lines, "\n")
}

// challengeMetric finds a kind's metric by key, label, or the label without
// its note in brackets.
func challengeMetric(kind ChallengeKind, name string) (ChallengeMetric, bool) {
	for _, metric := range kind.Metrics {
		short, _, _ := strings.Cut(metric.Label, "（")
		if strings.EqualFold(name, metric.Key) || name == metric.Label || name == short {
			return metric, true
		}
	}
	return ChallengeMetric{}, false
}

var challengeTimes = regexp.MustCompile(`^(?:每日|每周([一二三四五六日天]))?(\d{1,2})(?::(\d{2})|时(?:(\d{1,2})分)?)$`)

// parseChallengeTime reads HH:MM or ZZZ-Plugin's 每日20时30分 and 每周一20时;
// weekday is 0 for every day, else 1 (Monday) to 7 (Sunday).
func parseChallengeTime(text string) (hour, minute, weekday int, ok bool) {
	match := challengeTimes.FindStringSubmatch(text)
	if match == nil {
		return 0, 0, 0, false
	}
	hour, _ = strconv.Atoi(match[2])
	minute, _ = strconv.Atoi(match[3] + match[4])
	switch match[1] {
	case "":
	case "天":
		weekday = 7
	default:
		weekday = strings.Index("一二三四五六日", match[1])/len("一") + 1
	}
	return hour, minute, weekday, hour <= 23 && minute <= 59
}

func challengeSchedule(hour, minute, weekday int) string {
	day := "每日"
	if weekday > 0 {
		day = "每周" + []string{"一", "二", "三", "四", "五", "六", "日"}[weekday-1]
	}
	return fmt.Sprintf("%s %02d:%02d", day, hour, minute)
}

// challengeReminderLine describes one reminder for 挑战提醒状态.
func challengeReminderLine(task Reminder) string {
	kind, _ := challengeKind(task.ChallengeKind)
	label, compare := task.Metric, "≥"
	for _, metric := range kind.Metrics {
		if metric.Key == task.Metric {
			label, _, _ = strings.Cut(metric.Label, "（")
			if metric.Lower {
				compare = "≤"
			}
		}
	}
	state := map[string]string{"": "尚未检查", "target_met": "上次已达标", "notified": "上次未达标，已私聊通知", "notification_failed": "上次通知未送达", "notification_attempted": "上次通知未确认送达", "metric_unknown": "上次未读到该指标", "expired": "已到期", "threshold_off": "阈值为 0，不提醒", "plugin.game_remind_disabled": "全局挑战提醒已关闭"}[task.LastCode]
	if state == "" {
		state = "上次检查失败"
	}
	if !task.Enabled && task.LastCode != "expired" {
		state = "已暂停（" + state + "）"
	}
	target := label + " " + compare + " " + strconv.FormatFloat(task.Threshold, 'f', -1, 64)
	name := kind.Label
	if task.Pair {
		name, target = "式舆/危局", challengePairDescription(int(task.Threshold), int(task.DeadlyThreshold))
	}
	expires := time.UnixMilli(task.ExpiresAtMS).In(time.FixedZone("UTC+8", 28800)).Format("01-02")
	return fmt.Sprintf("%s · UID %s · %s · %s · %s 到期 · %s", name, task.Role.UID, target, challengeSchedule(task.Hour, task.Minute, task.Weekday), expires, state)
}
