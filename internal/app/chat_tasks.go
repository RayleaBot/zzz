package app

import (
	"context"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// Work a chat command starts may take longer than its event: the host gives
// an event 60 seconds by default and refuses actions once the event starts
// finishing. Such work is a chat task. It works within the event until
// chatTaskBudget, then continues on a temporary task the scheduler triggers
// each minute, each trigger working for the same budget, and answers in the
// chat the command came from. Tasks are held in memory only: a task left from
// before a restart finds nothing and removes itself.

// chatTaskBudget is how long an event works on a chat task, leaving the rest
// of the host's default 60 seconds for the reply.
const chatTaskBudget = 40 * time.Second

// taskHost is what a chat task asks of the host: an event's actions.
type taskHost interface {
	ServiceCaller
	imageRenderer
	SchedulerCreate(context.Context, rayleabot.SchedulerCreateRequest) (rayleabot.ActionResult, error)
	SchedulerDelete(context.Context, string) (rayleabot.ActionResult, error)
	MessageSend(context.Context, rayleabot.MessageSendRequest) (rayleabot.ActionResult, error)
}

// chatWork is the work of a chat task.
type chatWork interface {
	// step works until stop and returns the reply once the work is done;
	// done is false while work remains.
	step(ctx context.Context, a *App, host taskHost, stop time.Time) (reply []rayleabot.Segment, done bool)
}

// chatHandover is work that prepares, within its chat event, to continue on
// its scheduled task.
type chatHandover interface {
	handover(ctx context.Context, a *App, host taskHost, ref string) error
}

// chatTask is a chat task: its scheduled task's ID, log label and payload
// kind, its work and the chat to answer in.
type chatTask struct {
	ref     string
	work    chatWork
	owner   Subject
	target  rayleabot.Target
	expires time.Time
	label   string
	kind    string
	// busy marks a task an event is working on; a trigger meanwhile leaves
	// it and keeps its scheduled task.
	busy bool
}

// chatTasks holds the running chat tasks by their scheduled task ID.
type chatTasks struct {
	mu    sync.Mutex
	tasks map[string]*chatTask
}

// begin holds a task for the event that starts it; false when a task with
// its ID is still running.
func (t *chatTasks) begin(task *chatTask) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, exists := t.tasks[task.ref]; exists {
		return false
	}
	if t.tasks == nil {
		t.tasks = map[string]*chatTask{}
	}
	task.busy = true
	t.tasks[task.ref] = task
	return true
}

// claim hands a waiting task to a trigger. exists is true for a task another
// event is still working on, which claim leaves.
func (t *chatTasks) claim(ref string) (task *chatTask, exists bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	task, exists = t.tasks[ref]
	if !exists || task.busy {
		return nil, exists
	}
	task.busy = true
	return task, true
}

// release returns a task to wait for its next trigger.
func (t *chatTasks) release(ref string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if task := t.tasks[ref]; task != nil {
		task.busy = false
	}
}

func (t *chatTasks) end(ref string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.tasks, ref)
}

// beginChatTask holds work under its scheduled task's ID for the chat event
// that starts it, to answer in that chat within lifetime. It is nil when a
// task with the ID is still running.
func (a *App) beginChatTask(event *rayleabot.EventContext, ref, label, kind string, lifetime time.Duration, work chatWork) *chatTask {
	task := &chatTask{ref: ref, work: work, owner: syncOwner(event), target: event.Event.Target, expires: a.now().Add(lifetime), label: label, kind: kind}
	if !a.ChatTasks.begin(task) {
		return nil
	}
	return task
}

// stepChatTask works on a begun task within its chat event until stop. A
// finished task ends with its reply and done. Otherwise the task is handed to
// its scheduled task; err reports a handover that failed, which ends the
// task.
func (a *App) stepChatTask(ctx context.Context, host taskHost, task *chatTask, stop time.Time) (reply []rayleabot.Segment, done bool, err error) {
	if reply, done = task.work.step(ctx, a, host, stop); done {
		a.ChatTasks.end(task.ref)
		return reply, true, nil
	}
	if prepare, ok := task.work.(chatHandover); ok {
		err = prepare.handover(ctx, a, host, task.ref)
	}
	if err == nil {
		_, err = host.SchedulerCreate(ctx, rayleabot.SchedulerCreateRequest{TaskID: task.ref, Cron: "* * * * *", LogLabel: task.label, Payload: map[string]any{"kind": task.kind}})
	}
	if err != nil {
		a.ChatTasks.end(task.ref)
		return nil, false, err
	}
	a.ChatTasks.release(task.ref)
	return nil, false, nil
}

// continueChatTask is a trigger of a chat task's scheduled task: it works on
// the task for the budget and, once the work is done, answers in the chat the
// task came from and removes the scheduled task. An expired task, or one
// from before a restart, is removed without an answer.
func (a *App) continueChatTask(ctx context.Context, host taskHost, ref string) {
	task, exists := a.ChatTasks.claim(ref)
	if task == nil {
		if !exists {
			_, _ = host.SchedulerDelete(ctx, ref)
		}
		return
	}
	if a.now().After(task.expires) {
		a.ChatTasks.end(ref)
		_, _ = host.SchedulerDelete(ctx, ref)
		return
	}
	reply, done := task.work.step(ctx, a, host, a.now().Add(chatTaskBudget))
	if !done {
		a.ChatTasks.release(ref)
		return
	}
	_, _ = host.SchedulerDelete(ctx, ref)
	a.ChatTasks.end(ref)
	_, _ = host.MessageSend(ctx, rayleabot.MessageSendRequest{SourceProtocol: task.owner.SourceProtocol, SourceAdapter: task.owner.SourceAdapter, TargetType: task.target.Type, TargetID: task.target.ID, Message: rayleabot.MessageOut{Segments: reply}})
}

// runChatTask answers the scheduler's trigger of a chat task.
func (a *App) runChatTask(ctx context.Context, event *rayleabot.EventContext) error {
	if event.Event.SourceProtocol != "scheduler" || event.Event.SourceAdapter != "scheduler.internal" {
		return event.Fail("plugin.game_source_invalid", "任务来源无效。")
	}
	a.continueChatTask(ctx, event.Actions(), asText(event.Event.Payload["task_id"]))
	return event.Result(map[string]any{"handled": true})
}

// clock is the time chat tasks count and wait by; tests set their own.
type clock interface {
	Now() time.Time
	Sleep(context.Context, time.Duration) error
}

func (a *App) now() time.Time {
	if a.clock != nil {
		return a.clock.Now()
	}
	return time.Now()
}

// sleep waits for d or until ctx ends.
func (a *App) sleep(ctx context.Context, d time.Duration) error {
	if a.clock != nil {
		return a.clock.Sleep(ctx, d)
	}
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
