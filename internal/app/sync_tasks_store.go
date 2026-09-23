package app

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"path/filepath"
	"sync"
	"time"

	"github.com/RayleaBot/plugin-zzz/internal/gacha"
)

type SyncTask struct {
	Ref string `json:"ref"`
	Selection
	Owner         Subject `json:"owner"`
	Role          Role    `json:"role"`
	Provider      string  `json:"provider"`
	DelegationRef string  `json:"delegation_ref"`
	ExpiresAtMS   int64   `json:"expires_at_ms"`
	Kind          string  `json:"kind"`
	Hour          int     `json:"hour"`
	Full          bool    `json:"full"`
	Notify        bool    `json:"notify"`
	// ReplyType and ReplyID are the chat a 刷新抽卡记录 answers in when the
	// round completes, and Before the records each channel held then.
	ReplyType          string         `json:"reply_type,omitempty"`
	ReplyID            string         `json:"reply_id,omitempty"`
	Before             map[string]int `json:"before,omitempty"`
	State              string         `json:"state"`
	NextCheckMS        int64          `json:"next_check_ms"`
	LastCheckedMS      int64          `json:"last_checked_ms"`
	LastFinishedMS     int64          `json:"last_finished_ms"`
	LastNotificationMS int64          `json:"last_notification_ms"`
	RunDay             string         `json:"run_day"`
	Failures           int            `json:"failures"`
	Restarts           int            `json:"restarts"`
	LastCode           string         `json:"last_code"`
	Progress           gacha.SyncInfo `json:"progress"`
}

// SyncTaskStore keeps each background sync task in its own file. steps is
// held while a trigger advances a sync, so removing an archive waits for a
// page being merged instead of letting it revive the archive.
type SyncTaskStore struct {
	taskFiles[SyncTask]
	steps sync.RWMutex
}

func syncTaskStore(directory string) *SyncTaskStore {
	return &SyncTaskStore{taskFiles: taskFiles[SyncTask]{Directory: filepath.Join(directory, "sync-tasks"), Legacy: filepath.Join(directory, "sync-tasks.json")}}
}

func syncTaskTime(now int64, hour int) (string, int64, int64) {
	china := time.UnixMilli(now).In(time.FixedZone("UTC+8", 8*3600))
	at := time.Date(china.Year(), china.Month(), china.Day(), hour, 0, 0, 0, china.Location())
	return china.Format("2006-01-02"), at.UnixMilli(), at.AddDate(0, 0, 1).UnixMilli()
}
func syncTaskError(err error) string {
	if errors.Is(err, gacha.ErrConflict) {
		return "plugin.game_sync_conflict"
	}
	if errors.Is(err, gacha.ErrInvalid) {
		return "plugin.game_sync_invalid"
	}
	if errors.Is(err, gacha.ErrSync) {
		return "plugin.game_sync_busy"
	}
	return PublicError(err).Code
}
func syncTaskFailure(task *SyncTask, err error, now int64) {
	task.LastCode = syncTaskError(err)
	task.Failures++
	task.NextCheckMS = now + 5*60*1000
	switch task.LastCode {
	case "plugin.account_delegation_denied", "plugin.account_caller_denied", "plugin.account_not_found", "plugin.account_role_denied", "plugin.account_subject_denied", "plugin.upstream_auth_invalid", "plugin.upstream_device_required", "plugin.upstream_challenge_required", "plugin.upstream_gacha_unavailable", "plugin.game_sync_conflict", "plugin.game_sync_invalid":
		task.State = "paused"
	default:
		if task.Failures >= 3 {
			if task.Kind == "daily" {
				_, _, next := syncTaskTime(now, task.Hour)
				task.State = "waiting"
				task.NextCheckMS = next
			} else {
				task.State = "paused"
			}
		}
	}
}

type SyncTaskFetch func(context.Context, SyncTask, string, string, int) (gacha.RemotePage, error)
type SyncTaskSend func(context.Context, SyncTask, string) error

// Tick runs one trigger of a task: it advances the sync for pages until the
// event deadline nears and writes the task once at the end. Only the current
// scheduler event may supply fetch; it is never retained.
func (s *SyncTaskStore) Tick(ctx context.Context, ref string, now int64, jobs *gacha.Syncs, archive *gacha.Store, fetch SyncTaskFetch, send SyncTaskSend) error {
	claimed, ok, err := s.claim(ref)
	if err != nil || !ok {
		return err
	}
	defer s.release(ref)
	if err = s.run(ctx, &claimed, now, jobs, archive, fetch, send); errors.Is(err, errTaskChanged) {
		return nil
	}
	return err
}

func (s *SyncTaskStore) run(ctx context.Context, task *SyncTask, now int64, jobs *gacha.Syncs, archive *gacha.Store, fetch SyncTaskFetch, send SyncTaskSend) error {
	if task.State != "running" && task.State != "waiting" || task.NextCheckMS > now {
		return nil
	}
	if task.ExpiresAtMS <= now {
		task.State = "expired"
		task.LastCode = "expired"
		jobs.Forget(task.Progress.Ref)
		return s.save(*task)
	}
	day, at, nextDay := syncTaskTime(now, task.Hour)
	if task.State == "waiting" && task.Kind == "daily" && now < at {
		task.NextCheckMS = at
		return s.save(*task)
	}
	if task.State == "waiting" && task.RunDay == day && task.Kind == "daily" {
		task.NextCheckMS = nextDay
		return s.save(*task)
	}
	if task.State == "waiting" {
		jobs.Forget(task.Progress.Ref)
		task.Progress = gacha.SyncInfo{}
		task.RunDay = day
		task.Failures = 0
		task.Restarts = 0
		task.State = "running"
	}
	task.NextCheckMS = now + 60000
	task.LastCheckedMS = now
	s.steps.RLock()
	defer s.steps.RUnlock()
	info, err := jobs.Info(task.Progress.Ref)
	if err != nil {
		if task.Progress.Ref != "" {
			task.Restarts++
			jobs.Forget(task.Progress.Ref)
		}
		info, err = jobs.Start(archive, gacha.SyncChoice{AccountRef: task.AccountRef, RoleRef: task.RoleRef}, task.Role.UID, task.Role.Region, task.Full)
		if err != nil {
			syncTaskFailure(task, err, now)
			return s.save(*task)
		}
	}
	task.Progress = info
	// Pull pages back to back, 300 to 500 ms apart as upstream does, until
	// the event deadline is near; the next tick continues from there.
	for page := 0; task.Progress.State == "running"; page++ {
		if page > 0 {
			if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < 1500*time.Millisecond {
				break
			}
			timer := time.NewTimer(time.Duration(300+rand.IntN(201)) * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return s.save(*task)
			case <-timer.C:
			}
		}
		info, err = jobs.Step(ctx, archive, task.Progress.Ref, task.Progress.Sequence, func(ctx context.Context, pool, end string, page int) (gacha.RemotePage, error) {
			return fetch(ctx, *task, pool, end, page)
		})
		if err != nil {
			syncTaskFailure(task, err, now)
			if task.State != "running" {
				jobs.Forget(task.Progress.Ref)
			}
			return s.save(*task)
		}
		task.Progress = info
		task.Failures = 0
		task.LastCode = "sync_running"
	}
	if task.Progress.State != "completed" {
		return s.save(*task)
	}
	task.State = "completed"
	task.LastCode = "sync_completed"
	task.LastFinishedMS = now
	if task.Kind == "daily" {
		task.State = "waiting"
		task.NextCheckMS = nextDay
		task.RunDay = day
	}
	if task.Notify || task.ReplyType != "" {
		task.LastNotificationMS = now
	}
	// The notification is marked before it is sent, so a restart does not
	// send it again.
	answer := *task
	task.ReplyType, task.ReplyID, task.Before = "", "", nil
	if err = s.save(*task); err != nil {
		return err
	}
	jobs.Forget(task.Progress.Ref)
	if answer.ReplyType != "" && send != nil {
		// ZZZ-Plugin's report after 刷新抽卡记录.
		archived, _ := archive.Read(task.Role.UID, task.Role.Region)
		if err = send(ctx, answer, gachaLinkSummary(answer.Before, gachaPoolCounts(archived))); err != nil {
			task.LastCode = "sync_completed.notification_failed"
			return s.save(*task)
		}
	} else if task.Notify && send != nil {
		result := task.Progress.Result
		message := fmt.Sprintf("抽卡后台同步完成\n%s · %s\n新增 %d 条，档案共 %d 条。", task.Role.Nickname, task.Role.UID, result.Added, result.Total)
		if err = send(ctx, *task, message); err != nil {
			task.LastCode = "sync_completed.notification_failed"
			return s.save(*task)
		}
	}
	return nil
}

// RemoveArchive pauses the tasks of an archive, waiting for any page being
// merged, then removes the archive.
func (s *SyncTaskStore) RemoveArchive(jobs *gacha.Syncs, archive *gacha.Store, uid, region string) error {
	s.steps.Lock()
	defer s.steps.Unlock()
	forget := []string{}
	err := s.edit("", func(items *[]SyncTask, _ int) error {
		for i := range *items {
			t := &(*items)[i]
			if t.Role.UID == uid && t.Role.Region == region {
				t.State = "paused"
				t.LastCode = "archive_removed"
				forget = append(forget, t.Progress.Ref)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, ref := range forget {
		jobs.Forget(ref)
	}
	return archive.Remove(uid, region)
}
