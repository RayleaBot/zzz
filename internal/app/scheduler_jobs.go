package app

import (
	"context"
	"sync"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// timedTaskLabels name the timed tasks signinTask creates, by kind.
var timedTaskLabels = map[string]string{"signin": "每日签到", "monthly": "每日月报收集", "community": "米游社任务", "cloudgame": "云游戏签到"}

// reminderJob is the scheduler job of a stored reminder or timed task.
func (a *App) reminderJob(task Reminder) rayleabot.SchedulerCreateRequest {
	kind, cron, label := "stamina_reminder", "*/10 * * * *", "体力提醒"
	switch task.Kind {
	case "challenge":
		challenge, _ := challengeKind(task.ChallengeKind)
		label = challenge.Label
		if task.Pair {
			label = "式舆/危局"
		}
		kind, cron, label = "challenge_reminder", "*/5 * * * *", label+"挑战提醒"
	case "signin", "monthly":
		kind, label = task.Kind, timedTaskLabels[task.Kind]
	case "community", "cloudgame":
		kind, cron, label = task.Kind, "* * * * *", timedTaskLabels[task.Kind]
	}
	return rayleabot.SchedulerCreateRequest{TaskID: task.Ref, Cron: cron, LogLabel: a.Game.Name + label, Payload: taskPayload(kind, task.Ref)}
}

// syncJob is the scheduler job of a background gacha sync.
func (a *App) syncJob(task SyncTask) rayleabot.SchedulerCreateRequest {
	return rayleabot.SchedulerCreateRequest{TaskID: task.Ref, Cron: "* * * * *", LogLabel: a.Game.Name + "抽卡后台同步", Payload: taskPayload("gacha_sync", task.Ref)}
}

// contentJob is the scheduler job of a group's 米游社 pushes.
func (a *App) contentJob(ref string) rayleabot.SchedulerCreateRequest {
	return rayleabot.SchedulerCreateRequest{TaskID: ref, Cron: "*/5 * * * *", LogLabel: a.Game.Name + "米游社推送", Payload: taskPayload("public_content", ref)}
}

// Jobs created before their payloads carried task IDs trigger without
// saying which task they run. The first such trigger of a process creates
// the job of every stored task again: scheduler.create replaces a job with
// the same task ID, keeping its schedule, so each job's later triggers carry
// its task ID. A job whose task is not stored, such as an earlier chat
// task's, stays unknown; its triggers do nothing.
type legacyJobs struct {
	mu      sync.Mutex
	running bool
	done    bool
}

// begin claims the re-creation; false while another trigger runs it or
// after it is done.
func (l *legacyJobs) begin() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.running || l.done {
		return false
	}
	l.running = true
	return true
}

// finish ends a claimed re-creation, done unless it failed.
func (l *legacyJobs) finish(done bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.running, l.done = false, done
}

// runLegacyTrigger answers a trigger whose job's payload has no task ID by
// creating the stored tasks' jobs again, within the event's time.
func (a *App) runLegacyTrigger(ctx context.Context, event *rayleabot.EventContext) error {
	if event.Event.SourceProtocol != "scheduler" || event.Event.SourceAdapter != "scheduler.internal" {
		return event.Fail("plugin.game_source_invalid", "任务来源无效。")
	}
	if !a.legacyJobs.begin() {
		return event.Result(map[string]any{"handled": true})
	}
	ctx, cancel := a.eventWork(ctx, a.now())
	defer cancel()
	err := a.recreateJobs(ctx, event.Actions())
	a.legacyJobs.finish(err == nil)
	if err != nil {
		failure := PublicError(err)
		return event.Fail(failure.Code, failure.Message)
	}
	return event.Result(map[string]any{"handled": true})
}

// recreateJobs creates the job of every stored task again.
func (a *App) recreateJobs(ctx context.Context, actions *rayleabot.Actions) error {
	requests := []rayleabot.SchedulerCreateRequest{}
	reminders, err := a.Reminders.List()
	if err != nil {
		return err
	}
	for _, task := range reminders {
		requests = append(requests, a.reminderJob(task))
	}
	syncs, err := a.SyncTasks.List()
	if err != nil {
		return err
	}
	for _, task := range syncs {
		requests = append(requests, a.syncJob(task))
	}
	subscriptions, err := a.Subscriptions.List()
	if err != nil {
		return err
	}
	for _, item := range subscriptions {
		requests = append(requests, a.contentJob(item.Ref))
	}
	for _, request := range requests {
		if _, err := actions.SchedulerCreate(ctx, request); err != nil {
			return err
		}
	}
	return nil
}
