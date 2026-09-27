package app

import (
	"context"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// Work a chat command starts may take longer than its event: the host gives
// an event 60 seconds by default, refuses actions once the event starts
// finishing, and ignores what the plugin asks after it ends the event. Such
// work is a chat task. It works within the event until chatTaskBudget, then
// continues on a temporary task the scheduler triggers each minute, each
// trigger working for the same budget, and answers in the chat the command
// came from. An event's work on a task, its actions included, ends by
// chatTaskLimit; what the event could not finish waits for the next trigger.
// A task past its lifetime, or one an event held past chatTaskLease, answers
// with its failure. Tasks are held in memory only: a task left from before a
// restart finds nothing and removes itself.

const (
	// chatTaskBudget is how long an event starts new work on a chat task.
	chatTaskBudget = 40 * time.Second
	// chatTaskLimit is when an event's work on a chat task ends, leaving the
	// rest of the host's default 60 seconds for the event's own reply.
	chatTaskLimit = 55 * time.Second
	// chatTaskLease is how long an event may hold a chat task; an event
	// holding it longer has outlived the host's event.
	chatTaskLease = 2 * time.Minute
)

// errChatTaskTimeout is the failure of a task that ran out of time.
var errChatTaskTimeout = gameError("task_timeout", "后台读取超时，未能完成。")

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
	// done is false while work remains, including work ctx ended before it
	// could finish.
	step(ctx context.Context, a *App, host taskHost, stop time.Time) (reply []rayleabot.Segment, done bool)
	// fail is the reply when the work cannot finish. It may run while an
	// overrunning step still works, so it reads nothing step changes.
	fail(a *App, err error) []rayleabot.Segment
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
	// held is when the lease of the event working on the task runs out,
	// zero while the task waits for its next trigger.
	held time.Time
	// reply is the answer of work that finished after its event could no
	// longer send it.
	reply []rayleabot.Segment
	// ended marks a task no longer running; an event still holding it
	// leaves it alone.
	ended bool
}

// chatTasks holds the running chat tasks by their scheduled task ID.
type chatTasks struct {
	mu    sync.Mutex
	tasks map[string]*chatTask
}

// begin holds a task at now for the event that starts it; false when a task
// with its ID is still running.
func (t *chatTasks) begin(task *chatTask, now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, exists := t.tasks[task.ref]; exists {
		return false
	}
	if t.tasks == nil {
		t.tasks = map[string]*chatTask{}
	}
	task.held = now.Add(chatTaskLease)
	t.tasks[task.ref] = task
	return true
}

// claim hands a waiting task to a trigger at now; exists is false when no
// task has the ID. A task another event holds is left to it, returning nil,
// until the event's lease runs out: the task then ends and is returned with
// overdue set, for the trigger to answer with its failure.
func (t *chatTasks) claim(ref string, now time.Time) (task *chatTask, overdue, exists bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	task, exists = t.tasks[ref]
	switch {
	case !exists:
		return nil, false, false
	case task.held.IsZero():
		task.held = now.Add(chatTaskLease)
		return task, false, true
	case now.Before(task.held):
		return nil, false, true
	}
	t.endLocked(task)
	return task, true, true
}

// release returns a held task to wait for its next trigger, which answers
// with reply when it is set.
func (t *chatTasks) release(task *chatTask, reply []rayleabot.Segment) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !task.ended {
		task.held, task.reply = time.Time{}, reply
	}
}

// end stops a task; false when it had already ended.
func (t *chatTasks) end(task *chatTask) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if task.ended {
		return false
	}
	t.endLocked(task)
	return true
}

func (t *chatTasks) endLocked(task *chatTask) {
	task.ended = true
	if t.tasks[task.ref] == task {
		delete(t.tasks, task.ref)
	}
}

// beginChatTask holds work under its scheduled task's ID for the chat event
// that starts it, to answer in that chat within lifetime. It is nil when a
// task with the ID is still running.
func (a *App) beginChatTask(event *rayleabot.EventContext, ref, label, kind string, lifetime time.Duration, work chatWork) *chatTask {
	now := a.now()
	task := &chatTask{ref: ref, work: work, owner: syncOwner(event), target: event.Event.Target, expires: now.Add(lifetime), label: label, kind: kind}
	if !a.ChatTasks.begin(task, now) {
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
		// A task a trigger took from this event has answered already.
		return reply, a.ChatTasks.end(task), nil
	}
	if prepare, ok := task.work.(chatHandover); ok {
		err = prepare.handover(ctx, a, host, task.ref)
	}
	if err == nil {
		_, err = host.SchedulerCreate(ctx, rayleabot.SchedulerCreateRequest{TaskID: task.ref, Cron: "* * * * *", LogLabel: task.label, Payload: taskPayload(task.kind, task.ref)})
	}
	if err != nil {
		a.ChatTasks.end(task)
		return nil, false, err
	}
	a.ChatTasks.release(task, nil)
	return nil, false, nil
}

// continueChatTask is a trigger of a chat task's scheduled task: it works on
// the task for the budget and, once the work is done, answers in the chat the
// task came from and removes the scheduled task. A task past its lifetime, or
// taken from an overrunning event, answers with its failure; one from before
// a restart is removed without an answer.
func (a *App) continueChatTask(ctx context.Context, host taskHost, ref string) {
	now := a.now()
	task, overdue, exists := a.ChatTasks.claim(ref, now)
	switch {
	case !exists:
		_, _ = host.SchedulerDelete(ctx, ref)
		return
	case task == nil:
		return
	case overdue:
		a.answerChatTask(ctx, host, task, task.work.fail(a, errChatTaskTimeout))
		return
	case task.reply != nil:
		a.finishChatTask(ctx, host, task, task.reply)
		return
	case now.After(task.expires):
		if a.ChatTasks.end(task) {
			a.answerChatTask(ctx, host, task, task.work.fail(a, errChatTaskTimeout))
		}
		return
	}
	reply, done := task.work.step(ctx, a, host, now.Add(chatTaskBudget))
	if !done {
		a.ChatTasks.release(task, nil)
		return
	}
	a.finishChatTask(ctx, host, task, reply)
}

// finishChatTask ends a held task with its reply. When the event has run out
// of time to send it, the reply waits for the next trigger.
func (a *App) finishChatTask(ctx context.Context, host taskHost, task *chatTask, reply []rayleabot.Segment) {
	if ctx.Err() != nil {
		a.ChatTasks.release(task, reply)
		return
	}
	if a.ChatTasks.end(task) {
		a.answerChatTask(ctx, host, task, reply)
	}
}

// answerChatTask removes an ended task's scheduled task and answers in the
// chat the task came from.
func (a *App) answerChatTask(ctx context.Context, host taskHost, task *chatTask, reply []rayleabot.Segment) {
	_, _ = host.SchedulerDelete(ctx, task.ref)
	_, _ = host.MessageSend(ctx, rayleabot.MessageSendRequest{SourceProtocol: task.owner.SourceProtocol, SourceAdapter: task.owner.SourceAdapter, TargetType: task.target.Type, TargetID: task.target.ID, Message: rayleabot.MessageOut{Segments: reply}})
}

// runChatTask answers the scheduler's trigger of a chat task.
func (a *App) runChatTask(ctx context.Context, event *rayleabot.EventContext) error {
	if event.Event.SourceProtocol != "scheduler" || event.Event.SourceAdapter != "scheduler.internal" {
		return event.Fail("plugin.game_source_invalid", "任务来源无效。")
	}
	ctx, cancel := a.eventWork(ctx, a.now())
	defer cancel()
	a.continueChatTask(ctx, event.Actions(), triggerTask(event))
	return event.Result(map[string]any{"handled": true})
}

// eventWork bounds the chat task work of an event that started at start: it
// ends, the actions it asked for included, chatTaskLimit after start.
func (a *App) eventWork(ctx context.Context, start time.Time) (context.Context, context.CancelFunc) {
	if a.clock != nil {
		return a.clock.Until(ctx, start.Add(chatTaskLimit))
	}
	return context.WithDeadline(ctx, start.Add(chatTaskLimit))
}

// clock is the time chat tasks count and wait by; tests set their own.
type clock interface {
	Now() time.Time
	Sleep(context.Context, time.Duration) error
	// Until is ctx ending at deadline.
	Until(ctx context.Context, deadline time.Time) (context.Context, context.CancelFunc)
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
