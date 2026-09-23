package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

type Reminder struct {
	Once          bool          `json:"once,omitempty"`
	Community     CommunityPlan `json:"community,omitempty"`
	ChallengeKind string        `json:"challenge_kind,omitempty"`
	Metric        string        `json:"metric,omitempty"`
	Pair          bool          `json:"pair,omitempty"` // 开启挑战提醒, checking 式舆 and 危局 together
	Minute        int           `json:"minute,omitempty"`
	Weekday       int           `json:"weekday,omitempty"`
	Kind          string        `json:"kind,omitempty"`
	Hour          int           `json:"hour,omitempty"`
	Notify        bool          `json:"notify,omitempty"`
	LastSignDay   string        `json:"last_sign_day,omitempty"`
	Ref           string        `json:"ref"`
	Selection
	Owner         Subject `json:"owner"`
	Role          Role    `json:"role"`
	Provider      string  `json:"provider"`
	DelegationRef string  `json:"delegation_ref"`
	ExpiresAtMS   int64   `json:"expires_at_ms"`
	Threshold     float64 `json:"threshold"`
	Enabled       bool    `json:"enabled"`
	Armed         bool    `json:"armed"`
	NextCheckMS   int64   `json:"next_check_ms"`
	LastCheckedMS int64   `json:"last_checked_ms"`
	LastAttemptMS int64   `json:"last_attempt_ms"`
	LastCode      string  `json:"last_code"`
	// DeadlyThreshold and DeadlyDelegationRef are the 危局强袭战 half of a
	// pair; Threshold and DelegationRef are its 式舆防卫战 half.
	DeadlyThreshold     float64 `json:"deadly_threshold,omitempty"`
	DeadlyDelegationRef string  `json:"deadly_delegation_ref,omitempty"`
}

func (c AccountsClient) Authorize(ctx context.Context, choice Selection) (Account, Role, error) {
	// roles performs the existing account/owner/caller check without an upstream request.
	var result struct {
		Roles []Role `json:"roles"`
	}
	if err := c.call(ctx, "roles", map[string]any{"account_ref": choice.AccountRef, "game": c.Game}, &result); err != nil {
		return Account{}, Role{}, err
	}
	for _, r := range result.Roles {
		if r.Ref == choice.RoleRef {
			page := 0
			for {
				listed, err := c.List(ctx, page)
				if err != nil {
					return Account{}, Role{}, err
				}
				for _, a := range listed.Items {
					if a.Ref == choice.AccountRef {
						return a, r, nil
					}
				}
				if listed.NextPage == nil || *listed.NextPage <= page {
					break
				}
				page = *listed.NextPage
			}
		}
	}
	return Account{}, Role{}, gameError("role_missing", "未找到已授权的账号角色。")
}
func (a *App) manageReminder(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	client := a.accountClient(event)
	switch action {
	case "reminder.list":
		items, err := a.Reminders.List()
		items = slices.DeleteFunc(items, func(task Reminder) bool { return task.Kind != "" })
		return map[string]any{"items": items}, err
	case "reminder.remove":
		return a.removeDelegatedTask(ctx, event, asText(input["ref"]), "")
	case "reminder.create":
		var inputValue struct {
			Selection
			Threshold float64 `json:"threshold"`
			Days      int     `json:"days"`
		}
		if decodeObject(input, &inputValue) != nil || !finiteRange(inputValue.Threshold, 1, 100) || inputValue.Days < 1 || inputValue.Days > 90 {
			return nil, gameError("input_invalid", "提醒阈值为 1–100%，期限为 1–90 天。")
		}
		account, role, err := client.Authorize(ctx, inputValue.Selection)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256([]byte(client.Provider + "\x00" + inputValue.AccountRef + "\x00" + inputValue.RoleRef))
		task := Reminder{Ref: "game.reminder." + a.Game.ID + "." + hex.EncodeToString(digest[:]), Selection: inputValue.Selection, Owner: account.Owner, Role: role, Provider: client.Provider, Threshold: inputValue.Threshold, Armed: true}
		err = a.Reminders.edit(task.Ref, func(items *[]Reminder, _ int) error {
			if len(*items) >= 256 {
				return gameError("reminder_limit", "提醒数量已达上限。")
			}
			for _, old := range *items {
				if old.Selection == task.Selection && old.Provider == task.Provider && old.Kind == "" {
					return gameError("reminder_exists", "此角色已有提醒，请先移除旧提醒再调整。")
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
		err = client.call(ctx, "delegation.create", map[string]any{"account_ref": task.AccountRef, "role_ref": task.RoleRef, "task_id": task.Ref, "operation": a.Game.ID + ".note", "days": inputValue.Days}, &grant)
		if err == nil {
			task.DelegationRef = grant.Delegation.Ref
			task.ExpiresAtMS = grant.Delegation.ExpiresAtMS
			_, err = event.Actions().SchedulerCreate(ctx, rayleabot.SchedulerCreateRequest{TaskID: task.Ref, Cron: "*/10 * * * *", LogLabel: a.Game.Name + "体力提醒", Payload: map[string]any{"kind": "stamina_reminder"}})
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
		return map[string]any{"reminder": task}, nil
	}
	return nil, gameError("operation_denied", "提醒操作不存在。")
}
func (a *App) removeDelegatedTask(ctx context.Context, event *rayleabot.EventContext, ref string, kind string) (map[string]any, error) {
	client := a.accountClient(event)
	var task Reminder
	err := a.Reminders.edit(ref, func(items *[]Reminder, i int) error {
		if i < 0 || (*items)[i].Kind != kind {
			return gameError("reminder_missing", "提醒不存在或已移除。")
		}
		task = (*items)[i]
		if event.Event.EventType != "management.action" && task.Owner != (Subject{SourceProtocol: event.Event.SourceProtocol, SourceAdapter: event.Event.SourceAdapter, BotID: event.Bot.ID, ActorID: event.Event.Actor.ID}) {
			return gameError("source_invalid", "只能移除自己的提醒。")
		}
		*items = slices.Delete(*items, i, i+1)
		return nil
	})
	if err != nil {
		return map[string]any{"removed": false, "delegation_revoked": false}, err
	}
	client.Provider = task.Provider
	revoked := true
	for _, ref := range []string{task.DelegationRef, task.DeadlyDelegationRef} {
		if ref != "" {
			revoked = client.call(ctx, "delegation.revoke", map[string]any{"account_ref": task.AccountRef, "delegation_ref": ref}, nil) == nil && revoked
		}
	}
	return map[string]any{"removed": true, "delegation_revoked": revoked}, nil
}
func finiteRange(v, low, high float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= low && v <= high
}
func stamina(data map[string]any) (float64, float64, bool) {
	progress := asObject(asObject(data["energy"])["progress"])
	var c, m float64
	if decodeObject(progress["current"], &c) != nil || decodeObject(progress["max"], &m) != nil || !finiteRange(c, 0, 100000) || !finiteRange(m, 1, 100000) || c > m {
		return 0, 0, false
	}
	return c, m, true
}

// Tick runs one trigger of a task. The task's own kind decides what it
// requests; the store's lock is not held while it does.
func (s *ReminderStore) Tick(id string, now int64, query func(Reminder) (QueryResult, error), send func(Reminder, string) error, game Game) error {
	claimed, ok, err := s.claim(id)
	if err != nil || !ok {
		return err
	}
	defer s.release(id)
	if err = s.run(&claimed, now, query, send, game); errors.Is(err, errTaskChanged) {
		return nil
	}
	return err
}

func (s *ReminderStore) run(task *Reminder, now int64, query func(Reminder) (QueryResult, error), send func(Reminder, string) error, game Game) error {
	if !task.Enabled || task.NextCheckMS > now {
		return nil
	}
	if task.ExpiresAtMS <= now {
		task.Enabled = false
		task.LastCode = "expired"
		return s.save(*task)
	}
	switch task.Kind {
	case "community":
		return s.tickCommunity(task, now, query, send, game)
	case "cloudgame":
		return s.tickCloudGame(task, now, query, send, game)
	case "monthly":
		return s.tickMonthly(task, now, query, send, game)
	case "challenge":
		return s.tickChallenge(task, now, query, send, game)
	case "signin":
		return s.tickSignin(task, now, query, send, game)
	}
	task.NextCheckMS = now + int64(10*time.Minute/time.Millisecond)
	task.LastCheckedMS = now
	result, err := query(*task)
	if err != nil {
		task.LastCode = PublicError(err).Code
		task.NextCheckMS = now + int64(time.Hour/time.Millisecond)
		switch task.LastCode {
		case "plugin.account_delegation_denied", "plugin.account_caller_denied", "plugin.account_not_found", "plugin.account_role_denied", "plugin.upstream_auth_invalid":
			task.Enabled = false
		case "plugin.upstream_device_required", "plugin.upstream_challenge_required":
			task.NextCheckMS = now + int64(24*time.Hour/time.Millisecond)
		}
		return s.save(*task)
	}
	c, m, ok := stamina(result.Data)
	if !ok {
		task.LastCode = "plugin.game_note_invalid"
		return s.save(*task)
	}
	task.LastCode = "checked"
	if c*100/m < task.Threshold {
		task.Armed = true
		return s.save(*task)
	}
	if !task.Armed {
		return s.save(*task)
	}
	task.Armed = false
	task.LastAttemptMS = now
	task.LastCode = "notification_attempted"
	if err = s.save(*task); err != nil {
		return err
	}
	text := fmt.Sprintf("%s体力提醒\n%s · %s\n当前 %.0f / %.0f，达到 %.0f%% 阈值。\n降低到阈值以下后会重新启用下一次提醒。", game.Name, task.Role.Nickname, task.Role.UID, c, m, task.Threshold)
	if err = send(*task, text); err != nil {
		task.LastCode = "notification_failed"
	} else {
		task.LastCode = "notified"
	}
	return s.save(*task)
}
func (a *App) runReminder(ctx context.Context, event *rayleabot.EventContext) error {
	if event.Event.SourceProtocol != "scheduler" || event.Event.SourceAdapter != "scheduler.internal" {
		return event.Fail("plugin.game_source_invalid", "任务来源无效。")
	}
	err := a.Reminders.Tick(asText(event.Event.Payload["task_id"]), time.Now().UnixMilli(), func(task Reminder) (QueryResult, error) {
		client := AccountsClient{Caller: event.Actions(), Provider: task.Provider, Game: a.Game.ID}
		var result QueryResult
		params := map[string]any{"account_ref": task.AccountRef, "role_ref": task.RoleRef, "operation": a.Game.ID + ".note", "input": map[string]any{}, "delegation_ref": task.DelegationRef}
		if task.Kind == "signin" {
			params["operation"] = a.Game.ID + ".sign"
			params["write_confirmed"] = true
		}
		if task.Kind == "challenge" && task.Pair {
			// ZZZ-Plugin's global switch stops the reminders of its own
			// 开启挑战提醒, not the detailed ones.
			if !settings(event).ChallengeRemind {
				return result, gameError("remind_disabled", challengeRemindOff)
			}
			return a.queryChallengePair(ctx, client, task)
		}
		if task.Kind == "challenge" {
			kind, ok := challengeKind(task.ChallengeKind)
			if !ok {
				return result, gameError("input_invalid", "挑战任务玩法无效。")
			}
			params["operation"] = kind.Operation
		}
		if task.Kind == "community" {
			params["operation"] = a.Game.ID + ".community_run"
			params["write_confirmed"] = true
			step := task.Community.Steps[task.Community.Cursor]
			value := map[string]any{"action": step.Action}
			if step.PostID != "" {
				value["post_id"] = step.PostID
			}
			params["input"] = value
		}
		if task.Kind == "cloudgame" {
			params["operation"] = a.Game.ID + ".cloud_sign"
			params["write_confirmed"] = true
		}
		if task.Kind == "monthly" {
			params["operation"] = a.Game.ID + ".monthly"
			archive, err := a.Monthly.Read(task.Provider, task.Selection)
			if err != nil {
				return result, err
			}
			if err = client.call(ctx, "execute", params, &result); err != nil {
				return result, err
			}
			_, err = a.Monthly.Save(task.Provider, task.Selection, archive.Revision, result.Data, time.Now())
			return result, err
		}
		err := client.call(ctx, "execute", params, &result)
		return result, err
	}, func(task Reminder, text string) error {
		found := false
		for _, bot := range event.Bots {
			if bot.ID == task.Owner.BotID && bot.SourceProtocol == task.Owner.SourceProtocol && bot.SourceAdapter == task.Owner.SourceAdapter {
				found = true
				break
			}
		}
		if !found {
			return gameError("bot_missing", "提醒所属机器人当前不可用。")
		}
		_, err := event.Actions().MessageSend(ctx, rayleabot.MessageSendRequest{SourceProtocol: task.Owner.SourceProtocol, SourceAdapter: task.Owner.SourceAdapter, TargetType: "private", TargetID: task.Owner.ActorID, Message: rayleabot.MessageOut{Segments: []rayleabot.Segment{rayleabot.Text(text)}}})
		return err
	}, a.Game)
	if err != nil {
		failure := PublicError(err)
		return event.Fail(failure.Code, failure.Message)
	}
	return event.Result(map[string]any{"checked": true})
}

func (a *App) chatReminder(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	uid := ""
	threshold := 80.0
	create := command == "reminder"
	if create {
		if len(args) < 1 || len(args) > 2 {
			return event.SendText("使用“" + a.Game.Prefix + "体力提醒 百分数 [UID]”开启 30 天私聊提醒，例如 80。")
		}
		v, err := strconv.ParseFloat(strings.TrimSuffix(args[0], "%"), 64)
		if err != nil || !finiteRange(v, 1, 100) {
			return event.SendText("阈值应为 1–100 的百分数。")
		}
		threshold = v
		if len(args) == 2 {
			uid = args[1]
		}
	} else if len(args) > 1 {
		return event.SendText("使用“" + a.Game.Prefix + "关闭提醒 [UID]”。")
	} else if len(args) == 1 {
		uid = args[0]
	}
	if !create {
		items, err := a.Reminders.List()
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		owner := Subject{SourceProtocol: event.Event.SourceProtocol, SourceAdapter: event.Event.SourceAdapter, BotID: event.Bot.ID, ActorID: event.Event.Actor.ID}
		count := 0
		for _, task := range items {
			if task.Kind == "" && task.Owner == owner && (uid == "" || uid == task.Role.UID) {
				if _, err = a.manageReminder(ctx, event, "reminder.remove", map[string]any{"ref": task.Ref}); err != nil {
					return event.SendText(friendlyError(err))
				}
				count++
			}
		}
		return event.SendText(fmt.Sprintf("已关闭 %d 个%s体力提醒。", count, a.Game.Name))
	}
	listed, err := a.accountClient(event).List(ctx, 0)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	choice, _, err := Choose(listed, a.Game.ID, uid)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	_, err = a.manageReminder(ctx, event, "reminder.create", map[string]any{"account_ref": choice.AccountRef, "role_ref": choice.RoleRef, "threshold": threshold, "days": 30})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText(fmt.Sprintf("已开启%s体力提醒：达到 %.0f%% 后私聊通知，有效 30 天，每十分钟检查一次。发送“%s关闭提醒”停止。", a.Game.Name, threshold, a.Game.Prefix))
}
