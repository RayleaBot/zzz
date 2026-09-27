package app

import (
	"context"
	"sync"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// fakeClock is a clock tests move by hand; sleeping moves it on, and the
// contexts Until gives end when it passes their deadline. Events the SDK
// runs read it from goroutines of their own, so it is locked.
type fakeClock struct {
	mu     sync.Mutex
	at     time.Time
	timers []fakeTimer
}

// fakeTimer ends a context Until gave at its deadline.
type fakeTimer struct {
	at     time.Time
	cancel context.CancelCauseFunc
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.at
}

func (c *fakeClock) Sleep(ctx context.Context, d time.Duration) error {
	if d > 0 {
		c.move(func(at time.Time) time.Time { return at.Add(d) })
	}
	return ctx.Err()
}

func (c *fakeClock) Until(ctx context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancelCause(ctx)
	c.mu.Lock()
	due := !c.at.Before(deadline)
	if !due {
		c.timers = append(c.timers, fakeTimer{at: deadline, cancel: cancel})
	}
	c.mu.Unlock()
	if due {
		cancel(context.DeadlineExceeded)
	}
	return ctx, func() { cancel(context.Canceled) }
}

// set moves the clock to at.
func (c *fakeClock) set(at time.Time) {
	c.move(func(time.Time) time.Time { return at })
}

// move sets the time to next of the current one and ends the contexts whose
// deadline it reached.
func (c *fakeClock) move(next func(time.Time) time.Time) {
	c.mu.Lock()
	c.at = next(c.at)
	var due []context.CancelCauseFunc
	kept := c.timers[:0]
	for _, timer := range c.timers {
		if timer.at.After(c.at) {
			kept = append(kept, timer)
		} else {
			due = append(due, timer.cancel)
		}
	}
	c.timers = kept
	c.mu.Unlock()
	for _, cancel := range due {
		cancel(context.DeadlineExceeded)
	}
}

// fakeHost records the host actions chat tasks ask for and answers service
// calls with call. It renders no images.
type fakeHost struct {
	call      func(rayleabot.ServiceCallRequest, any) error
	scheduled []rayleabot.SchedulerCreateRequest
	deleted   []string
	sent      []rayleabot.MessageSendRequest
}

func (h *fakeHost) CallService(_ context.Context, request rayleabot.ServiceCallRequest, out any) error {
	return h.call(request, out)
}

func (h *fakeHost) RenderImage(context.Context, rayleabot.RenderImageRequest) (rayleabot.ActionResult, error) {
	return nil, gameError("render_unavailable", "synthetic")
}

func (h *fakeHost) SchedulerCreate(_ context.Context, request rayleabot.SchedulerCreateRequest) (rayleabot.ActionResult, error) {
	h.scheduled = append(h.scheduled, request)
	return rayleabot.ActionResult{}, nil
}

func (h *fakeHost) SchedulerDelete(_ context.Context, id string) (rayleabot.ActionResult, error) {
	h.deleted = append(h.deleted, id)
	return rayleabot.ActionResult{}, nil
}

func (h *fakeHost) MessageSend(_ context.Context, request rayleabot.MessageSendRequest) (rayleabot.ActionResult, error) {
	h.sent = append(h.sent, request)
	return rayleabot.ActionResult{}, nil
}

// countdown is work finished after a number of steps.
type countdown struct{ left, steps int }

func (c *countdown) step(context.Context, *App, taskHost, time.Time) ([]rayleabot.Segment, bool) {
	c.left--
	c.steps++
	return []rayleabot.Segment{rayleabot.Text("完成")}, c.left <= 0
}

func (c *countdown) fail(_ *App, err error) []rayleabot.Segment {
	return []rayleabot.Segment{rayleabot.Text("失败：" + friendlyError(err))}
}

// chatEvent is a group message event of user u.
func chatEvent() *rayleabot.EventContext {
	return &rayleabot.EventContext{Bot: rayleabot.Bot{ID: "bot"}, Event: rayleabot.Event{EventType: "message.group", SourceProtocol: "onebot11", SourceAdapter: "a", Actor: rayleabot.Actor{ID: "u"}, Target: rayleabot.Target{Type: "group", ID: "g"}}}
}

func TestChatTaskContinuesOnItsScheduledTask(t *testing.T) {
	clock := &fakeClock{at: time.Unix(1_800_000_000, 0)}
	a := &App{clock: clock}
	host := &fakeHost{}
	task := a.beginChatTask(chatEvent(), "game.link.x", "记录", "gacha_link", time.Hour, &countdown{left: 3})
	if task == nil || a.beginChatTask(chatEvent(), "game.link.x", "记录", "gacha_link", time.Hour, &countdown{}) != nil {
		t.Fatal("a running task was started twice")
	}
	if _, done, err := a.stepChatTask(t.Context(), host, task, a.now()); done || err != nil {
		t.Fatal("the event finished work that remains", done, err)
	}
	if len(host.scheduled) != 1 || host.scheduled[0].TaskID != "game.link.x" || host.scheduled[0].Cron != "* * * * *" || host.scheduled[0].Payload["kind"] != "gacha_link" || host.scheduled[0].Payload["task_id"] != "game.link.x" {
		t.Fatalf("scheduled = %+v", host.scheduled)
	}
	// A trigger while another event works on the task keeps its scheduled
	// task.
	held, _, _ := a.ChatTasks.claim("game.link.x", a.now())
	a.continueChatTask(t.Context(), host, "game.link.x")
	a.ChatTasks.release(held, nil)
	a.continueChatTask(t.Context(), host, "game.link.x")
	if len(host.deleted) != 0 || len(host.sent) != 0 || task.work.(*countdown).steps != 2 {
		t.Fatal("unfinished work ended", host.deleted, host.sent)
	}
	a.continueChatTask(t.Context(), host, "game.link.x")
	if len(host.deleted) != 1 || len(host.sent) != 1 || sentText(host.sent[0].Message) != "完成" {
		t.Fatal("finished work did not answer once", host.deleted, host.sent)
	}
	if sent := host.sent[0]; sent.SourceProtocol != "onebot11" || sent.SourceAdapter != "a" || sent.TargetType != "group" || sent.TargetID != "g" {
		t.Fatalf("answered in %+v", sent)
	}
	// A task from before a restart is removed without an answer; one past
	// its lifetime answers with its failure.
	a.continueChatTask(t.Context(), host, "game.link.gone")
	old := a.beginChatTask(chatEvent(), "game.link.old", "记录", "gacha_link", time.Minute, &countdown{left: 2})
	a.ChatTasks.release(old, nil)
	clock.set(a.now().Add(2 * time.Minute))
	a.continueChatTask(t.Context(), host, "game.link.old")
	if len(host.deleted) != 3 || host.deleted[1] != "game.link.gone" || host.deleted[2] != "game.link.old" || len(host.sent) != 2 || sentText(host.sent[1].Message) != "失败：后台读取超时，未能完成。" || len(a.ChatTasks.tasks) != 0 {
		t.Fatal("stale tasks were kept or answered wrongly", host.deleted, host.sent)
	}
}

// An event that holds a task past its lease has outlived the host's event: a
// later trigger ends the task with its failure, and the event, should it
// ever finish, neither answers nor holds the task again.
func TestChatTaskTakenFromAnEventThatOverranAnswersItsFailure(t *testing.T) {
	clock := &fakeClock{at: time.Unix(1_800_000_000, 0)}
	a := &App{clock: clock}
	host := &fakeHost{}
	work := &countdown{left: 1}
	task := a.beginChatTask(chatEvent(), "game.panel.x", "更新面板", "panel_refresh", time.Hour, work)
	clock.set(clock.Now().Add(time.Minute))
	a.continueChatTask(t.Context(), host, "game.panel.x")
	if len(host.deleted) != 0 || len(host.sent) != 0 {
		t.Fatal("a trigger took the task from an event within its lease")
	}
	clock.set(clock.Now().Add(2 * time.Minute))
	a.continueChatTask(t.Context(), host, "game.panel.x")
	if len(host.deleted) != 1 || len(host.sent) != 1 || sentText(host.sent[0].Message) != "失败：后台读取超时，未能完成。" || work.steps != 0 {
		t.Fatal("the overrun task did not answer its failure", host.deleted, host.sent)
	}
	if reply, done, err := a.stepChatTask(t.Context(), host, task, clock.Now()); done || err != nil || len(host.sent) != 1 {
		t.Fatal("the overrunning event answered as well", reply, done, err)
	}
	if a.beginChatTask(chatEvent(), "game.panel.x", "更新面板", "panel_refresh", time.Hour, &countdown{left: 1}) == nil {
		t.Fatal("the overrun task still blocks the command")
	}
}

// Work that finishes after its trigger ran out of time to answer keeps its
// reply for the next trigger, which answers without working again.
func TestChatTaskReplyTheEventCouldNotSendWaitsForTheNextTrigger(t *testing.T) {
	a := &App{clock: &fakeClock{at: time.Unix(1_800_000_000, 0)}}
	host := &fakeHost{}
	work := &countdown{left: 2}
	task := a.beginChatTask(chatEvent(), "game.link.x", "记录", "gacha_link", time.Hour, work)
	if _, done, err := a.stepChatTask(t.Context(), host, task, a.now()); done || err != nil {
		t.Fatal(done, err)
	}
	ended, cancel := context.WithCancel(t.Context())
	cancel()
	a.continueChatTask(ended, host, "game.link.x")
	if len(host.sent) != 0 || len(host.deleted) != 0 || work.steps != 2 {
		t.Fatal("a trigger past its time answered", host.sent, host.deleted)
	}
	a.continueChatTask(t.Context(), host, "game.link.x")
	if len(host.sent) != 1 || sentText(host.sent[0].Message) != "完成" || len(host.deleted) != 1 || work.steps != 2 {
		t.Fatal("the kept reply was not answered once", host.sent, host.deleted, work.steps)
	}
}
