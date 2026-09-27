package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"slices"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/gacha"
)

// A 每日同步 is a role's periodic background sync the management page
// creates. The account plugin grants it a read-only zzz.gacha delegation for
// the days chosen; every day after its hour (Beijing time) a trigger of its
// scheduler job moves to the background and reads every channel under the
// delegation, listed like a background sync the page started. A failed round
// is tried again five minutes later, three times a day at most; a revoked
// delegation, a verification the official app asks for, a conflicting
// archive or an invalid page pause the task until the page runs it again.

// dailySyncTask prefixes the scheduler jobs of the 每日同步.
const dailySyncTask = "game.sync."

// dailyRetry is how long a failed round waits before it is tried again.
const dailyRetry = 5 * time.Minute

// DailySync is a stored 每日同步. State is creating, waiting, running (a
// round is due or reading), paused or expired; Progress is the last round's.
type DailySync struct {
	Ref string `json:"ref"`
	Selection
	Owner              Subject        `json:"owner"`
	Role               Role           `json:"role"`
	Provider           string         `json:"provider"`
	DelegationRef      string         `json:"delegation_ref"`
	ExpiresAtMS        int64          `json:"expires_at_ms"`
	Hour               int            `json:"hour"`
	Full               bool           `json:"full"`
	Notify             bool           `json:"notify"`
	State              string         `json:"state"`
	NextCheckMS        int64          `json:"next_check_ms"`
	LastCheckedMS      int64          `json:"last_checked_ms"`
	LastFinishedMS     int64          `json:"last_finished_ms"`
	LastNotificationMS int64          `json:"last_notification_ms"`
	RunDay             string         `json:"run_day"`
	Failures           int            `json:"failures"`
	LastCode           string         `json:"last_code"`
	Progress           gacha.SyncInfo `json:"progress"`
}

func (t DailySync) taskRef() string { return t.Ref }

// DailySyncStore keeps each 每日同步 in its own file.
type DailySyncStore struct {
	taskFiles[DailySync]
}

func dailySyncStore(directory string) *DailySyncStore {
	return &DailySyncStore{taskFiles[DailySync]{Directory: filepath.Join(directory, "gacha-daily")}}
}

// dailySyncID is the task of a provider's role.
func dailySyncID(game, provider string, choice Selection) string {
	sum := sha256.Sum256([]byte(provider + "\x00" + choice.AccountRef + "\x00" + choice.RoleRef))
	return dailySyncTask + game + "." + hex.EncodeToString(sum[:])
}

// dailyTime is the Beijing day of now, the task's hour on it and on the next
// day.
func dailyTime(now int64, hour int) (day string, at, next int64) {
	china := time.UnixMilli(now).In(time.FixedZone("UTC+8", 8*3600))
	start := time.Date(china.Year(), china.Month(), china.Day(), hour, 0, 0, 0, china.Location())
	return china.Format("2006-01-02"), start.UnixMilli(), start.AddDate(0, 0, 1).UnixMilli()
}

// dailyJob is the scheduler job of a 每日同步.
func (a *App) dailyJob(task DailySync) rayleabot.SchedulerCreateRequest {
	return rayleabot.SchedulerCreateRequest{TaskID: task.Ref, Cron: "*/5 * * * *", LogLabel: a.Game.Name + "每日抽卡同步", Payload: map[string]any{"kind": "gacha_daily"}}
}

// due decides a trigger at now: it moves the task on and reports whether a
// round reads now and whether the task changed.
func (t *DailySync) due(now int64) (round, changed bool) {
	if t.State != "running" && t.State != "waiting" || t.NextCheckMS > now {
		return false, false
	}
	if t.ExpiresAtMS <= now {
		t.State, t.LastCode = "expired", "expired"
		return false, true
	}
	day, at, next := dailyTime(now, t.Hour)
	switch {
	case t.State == "waiting" && now < at:
		t.NextCheckMS = at
		return false, true
	case t.State == "waiting" && t.RunDay == day:
		t.NextCheckMS = next
		return false, true
	case t.State == "waiting":
		t.State, t.RunDay, t.Failures = "running", day, 0
	}
	// A round cut short by a restart is read again once this has passed.
	t.NextCheckMS = now + dailyRetry.Milliseconds()
	t.LastCheckedMS, t.LastCode = now, "sync_running"
	return true, true
}

// dailyPauses are the failures a round cannot get past by itself.
var dailyPauses = []string{"plugin.account_delegation_denied", "plugin.account_caller_denied", "plugin.account_not_found", "plugin.account_role_denied", "plugin.account_subject_denied", "plugin.upstream_auth_invalid", "plugin.upstream_device_required", "plugin.upstream_challenge_required", "plugin.upstream_gacha_unavailable", "plugin.game_sync_conflict", "plugin.game_sync_invalid", "archive_removed"}

// fail records a round that ended with code at now: it pauses the task, waits
// for the next day after a cancel or a third failure, or tries again after
// dailyRetry.
func (t *DailySync) fail(code string, now int64) {
	t.LastCode = code
	_, _, next := dailyTime(now, t.Hour)
	switch {
	case slices.Contains(dailyPauses, code):
		t.State = "paused"
	case code == "sync_canceled":
		t.State, t.NextCheckMS = "waiting", next
	default:
		t.Failures++
		t.NextCheckMS = now + dailyRetry.Milliseconds()
		if t.Failures >= 3 {
			t.State, t.NextCheckMS = "waiting", next
		}
	}
}

// pauseArchive pauses the 每日同步 of an archive being removed, so none of
// them brings it back.
func (s *DailySyncStore) pauseArchive(uid, region string) error {
	return s.edit("", func(items *[]DailySync, _ int) error {
		for i := range *items {
			if t := &(*items)[i]; t.Role.UID == uid && t.Role.Region == region {
				t.State, t.LastCode = "paused", "archive_removed"
			}
		}
		return nil
	})
}

// runDailySync is a trigger of a 每日同步's job. A due round moves the
// trigger to the background, since reading every page takes longer than the
// event; a trigger the host keeps in the foreground reads within its
// deadline, and what it cannot read is tried again. A trigger whose task is
// not stored deletes its job.
func (a *App) runDailySync(ctx context.Context, event *rayleabot.EventContext) error {
	if event.Event.SourceProtocol != "scheduler" || event.Event.SourceAdapter != "scheduler.internal" {
		return event.Fail("plugin.game_source_invalid", "任务来源无效。")
	}
	ref := event.Event.TaskID()
	task, claimed, err := a.DailySyncs.claim(ref)
	switch {
	case errors.Is(err, errTaskMissing):
		_, _ = event.Actions().SchedulerDelete(ctx, ref)
		return event.Result(map[string]any{"checked": false})
	case err != nil:
		return manageFailure(event, err)
	case !claimed:
		// Another trigger is reading it.
		return event.Result(map[string]any{"checked": false})
	}
	defer a.DailySyncs.release(ref)
	now := a.now().UnixMilli()
	round, changed := task.due(now)
	if changed {
		if err = a.DailySyncs.save(task); err != nil {
			return a.dailyResult(event, err)
		}
	}
	if !round {
		return event.Result(map[string]any{"checked": true})
	}
	_, _ = event.Detach(ctx, nil)
	return a.dailyResult(event, a.readDailySync(ctx, event, task))
}

// dailyResult ends a trigger; a task edited or removed meanwhile keeps what
// the page did.
func (a *App) dailyResult(event *rayleabot.EventContext, err error) error {
	if err != nil && !errors.Is(err, errTaskChanged) {
		return manageFailure(event, err)
	}
	return event.Result(map[string]any{"checked": true})
}

// readDailySync reads a due round of task and saves how it ended. A
// completed round is saved before its notification is sent, so a restart
// does not send it again.
func (a *App) readDailySync(ctx context.Context, event *rayleabot.EventContext, task DailySync) error {
	client := AccountsClient{Caller: event.Actions(), Provider: task.Provider, Game: a.Game.ID}
	info, err := a.Syncs.Start(a.Gacha, gacha.SyncChoice{AccountRef: task.AccountRef, RoleRef: task.RoleRef}, task.Role.UID, task.Role.Region, task.Full)
	if err != nil {
		task.fail(PublicError(syncError(err)).Code, a.now().UnixMilli())
		return a.DailySyncs.save(task)
	}
	item := BackgroundSync{Ref: info.Ref, Task: task.Ref, Role: task.Role, Owner: task.Owner, Full: task.Full, Notify: task.Notify, State: "running", LastCode: "sync_running", StartedMS: a.now().UnixMilli(), Progress: info}
	if !a.BackgroundSyncs.begin(item) {
		a.Syncs.Forget(info.Ref)
		task.fail(errSyncRunning.Code, a.now().UnixMilli())
		return a.DailySyncs.save(task)
	}
	var saved error
	item, _ = a.readBackgroundSync(ctx, item, info, a.accountPages(client, task.Selection, task.Role, task.DelegationRef), func(done gacha.SyncInfo) error {
		now := a.now().UnixMilli()
		_, _, next := dailyTime(now, task.Hour)
		task.State, task.NextCheckMS, task.LastFinishedMS, task.LastCode, task.Progress = "waiting", next, now, "sync_completed", done
		if task.Notify {
			task.LastNotificationMS = now
		}
		if saved = a.DailySyncs.save(task); saved != nil || !task.Notify {
			return nil
		}
		return a.notifySync(ctx, event, item, *done.Result)
	})
	switch {
	case saved != nil:
		return saved
	case item.State == "completed" && item.LastCode == "sync_completed":
		return nil
	case item.State == "completed":
		task.LastCode = item.LastCode
	default:
		task.fail(item.LastCode, a.now().UnixMilli())
	}
	task.Progress = item.Progress
	return a.DailySyncs.save(task)
}

// createDailySync creates a role's 每日同步: its delegation, its stored task
// and its scheduler job. A task that cannot get all three is dropped, and
// its delegation revoked.
func (a *App) createDailySync(ctx context.Context, event *rayleabot.EventContext, input map[string]any) (map[string]any, error) {
	var q struct {
		Selection
		Hour   int  `json:"hour"`
		Days   int  `json:"days"`
		Full   bool `json:"full"`
		Notify bool `json:"notify"`
	}
	if input["confirm"] != true || decodeObject(input, &q) != nil || q.Days < 1 || q.Days > 90 || q.Hour < 0 || q.Hour > 23 {
		return nil, gameError("input_invalid", "请选择 1–90 天有效期和 0–23 时。")
	}
	client := a.accountClient(event)
	account, role, err := client.Authorize(ctx, q.Selection)
	if err != nil {
		return nil, err
	}
	if !syncRegionAllowed(role.Region) {
		return nil, gameError("region_unsupported", "此区服暂未适配每日同步。")
	}
	task := DailySync{Ref: dailySyncID(a.Game.ID, client.Provider, q.Selection), Selection: q.Selection, Owner: account.Owner, Role: role, Provider: client.Provider, Hour: q.Hour, Full: q.Full, Notify: q.Notify, State: "creating"}
	// The task is kept as creating while the delegation and job are made
	// outside the store's lock, then enabled or dropped.
	err = a.DailySyncs.edit(task.Ref, func(items *[]DailySync, i int) error {
		if len(*items) >= 256 {
			return gameError("sync_task_limit", "每日同步任务已达上限。")
		}
		if i >= 0 {
			return gameError("sync_task_exists", "此角色已有每日同步，请重新运行已有任务，或移除后调整设置。")
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
		task.DelegationRef, task.ExpiresAtMS = grant.Delegation.Ref, grant.Delegation.ExpiresAtMS
		_, err = event.Actions().SchedulerCreate(ctx, a.dailyJob(task))
	}
	if err == nil {
		task.State, task.LastCode = "waiting", "queued"
		err = a.DailySyncs.edit(task.Ref, func(items *[]DailySync, i int) error {
			if i < 0 {
				return gameError("sync_task_missing", "每日同步创建已取消。")
			}
			(*items)[i] = task
			return nil
		})
	}
	if err != nil {
		_ = a.DailySyncs.edit(task.Ref, func(items *[]DailySync, i int) error {
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

// runDailySyncNow has a 每日同步 read a round at its next trigger, after the
// account check, whatever the day and hour.
func (a *App) runDailySyncNow(ctx context.Context, event *rayleabot.EventContext, input map[string]any) (map[string]any, error) {
	if input["confirm"] != true {
		return nil, gameError("input_invalid", "请明确重新运行每日同步。")
	}
	ref := asText(input["ref"])
	items, err := a.DailySyncs.List()
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(items, func(t DailySync) bool { return t.Ref == ref })
	if i < 0 {
		return nil, gameError("sync_task_missing", "每日同步不存在。")
	}
	// The account check runs before the task is edited, outside the store's
	// lock.
	client := AccountsClient{Caller: event.Actions(), Provider: items[i].Provider, Game: a.Game.ID}
	_, role, err := client.Authorize(ctx, items[i].Selection)
	if err != nil {
		return nil, err
	}
	now := a.now().UnixMilli()
	err = a.DailySyncs.edit(ref, func(items *[]DailySync, i int) error {
		if i < 0 {
			return gameError("sync_task_missing", "每日同步不存在。")
		}
		t := &(*items)[i]
		switch {
		case t.State == "running" || t.State == "creating":
			return gameError("sync_task_running", "此任务正在处理，请查看进度。")
		case t.ExpiresAtMS <= now || t.DelegationRef == "":
			return gameError("sync_task_expired", "委托已过期，请移除后重新创建任务。")
		case role.UID != t.Role.UID || role.Region != t.Role.Region:
			return gameError("role_missing", "账号角色已变更，请重新创建任务。")
		}
		t.State, t.NextCheckMS, t.Failures, t.LastCode = "running", 0, 0, "queued"
		t.RunDay, _, _ = dailyTime(now, t.Hour)
		return nil
	})
	return map[string]any{"queued": err == nil}, err
}

// removeDailySync stops a 每日同步: its stored task, its job, a round it is
// reading and, as far as the account plugin answers, its delegation.
func (a *App) removeDailySync(ctx context.Context, event *rayleabot.EventContext, input map[string]any) (map[string]any, error) {
	ref := asText(input["ref"])
	var task DailySync
	err := a.DailySyncs.edit(ref, func(items *[]DailySync, i int) error {
		if i < 0 {
			return gameError("sync_task_missing", "每日同步不存在。")
		}
		task = (*items)[i]
		*items = slices.Delete(*items, i, i+1)
		return nil
	})
	if err != nil {
		return map[string]any{"removed": false, "delegation_revoked": false}, err
	}
	_, _ = event.Actions().SchedulerDelete(ctx, ref)
	a.BackgroundSyncs.cancel(func(item BackgroundSync) bool { return item.Task == ref }, "sync_canceled")
	client := AccountsClient{Caller: event.Actions(), Provider: task.Provider, Game: a.Game.ID}
	revoked := client.call(ctx, "delegation.revoke", map[string]any{"account_ref": task.AccountRef, "delegation_ref": task.DelegationRef}, nil) == nil
	return map[string]any{"removed": true, "delegation_revoked": revoked}, nil
}
