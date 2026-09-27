package app

import (
	"context"
	"sync"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// fakeClock is a clock tests move by hand; sleeping moves it on. Events the
// SDK runs read it from goroutines of their own, so it is locked.
type fakeClock struct {
	mu sync.Mutex
	at time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.at
}

func (c *fakeClock) Sleep(ctx context.Context, d time.Duration) error {
	c.mu.Lock()
	if d > 0 {
		c.at = c.at.Add(d)
	}
	c.mu.Unlock()
	return ctx.Err()
}

// set moves the clock to at.
func (c *fakeClock) set(at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.at = at
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
type countdown struct{ left int }

func (c *countdown) step(context.Context, *App, taskHost, time.Time) ([]rayleabot.Segment, bool) {
	c.left--
	return []rayleabot.Segment{rayleabot.Text("完成")}, c.left <= 0
}

// chatEvent is a group message event of user u.
func chatEvent() *rayleabot.EventContext {
	return &rayleabot.EventContext{Bot: rayleabot.Bot{ID: "bot"}, Event: rayleabot.Event{EventType: "message.group", SourceProtocol: "onebot11", SourceAdapter: "a", Actor: rayleabot.Actor{ID: "u"}, Target: rayleabot.Target{Type: "group", ID: "g"}}}
}

func TestChatTaskContinuesOnItsScheduledTask(t *testing.T) {
	a := &App{clock: &fakeClock{at: time.Unix(1_800_000_000, 0)}}
	host := &fakeHost{}
	task := a.beginChatTask(chatEvent(), "game.link.x", "记录", "gacha_link", time.Minute, &countdown{left: 3})
	if task == nil || a.beginChatTask(chatEvent(), "game.link.x", "记录", "gacha_link", time.Minute, &countdown{}) != nil {
		t.Fatal("a running task was started twice")
	}
	if _, done, err := a.stepChatTask(t.Context(), host, task, a.now()); done || err != nil {
		t.Fatal("the event finished work that remains", done, err)
	}
	if len(host.scheduled) != 1 || host.scheduled[0].TaskID != "game.link.x" || host.scheduled[0].Cron != "* * * * *" || host.scheduled[0].Payload["kind"] != "gacha_link" {
		t.Fatalf("scheduled = %+v", host.scheduled)
	}
	// A trigger while an event works on the task keeps its scheduled task.
	a.ChatTasks.tasks["game.link.x"].busy = true
	a.continueChatTask(t.Context(), host, "game.link.x")
	a.ChatTasks.release("game.link.x")
	a.continueChatTask(t.Context(), host, "game.link.x")
	if len(host.deleted) != 0 || len(host.sent) != 0 {
		t.Fatal("unfinished work ended", host.deleted, host.sent)
	}
	a.continueChatTask(t.Context(), host, "game.link.x")
	if len(host.deleted) != 1 || len(host.sent) != 1 {
		t.Fatal("finished work did not answer once", host.deleted, host.sent)
	}
	if sent := host.sent[0]; sent.SourceProtocol != "onebot11" || sent.SourceAdapter != "a" || sent.TargetType != "group" || sent.TargetID != "g" {
		t.Fatalf("answered in %+v", sent)
	}
	// A task from before a restart, or past its lifetime, is removed without
	// an answer.
	a.continueChatTask(t.Context(), host, "game.link.gone")
	a.beginChatTask(chatEvent(), "game.link.old", "记录", "gacha_link", time.Minute, &countdown{left: 2})
	a.ChatTasks.release("game.link.old")
	a.clock.(*fakeClock).set(a.now().Add(2 * time.Minute))
	a.continueChatTask(t.Context(), host, "game.link.old")
	if len(host.deleted) != 3 || host.deleted[1] != "game.link.gone" || host.deleted[2] != "game.link.old" || len(host.sent) != 1 || len(a.ChatTasks.tasks) != 0 {
		t.Fatal("stale tasks were kept or answered", host.deleted, host.sent)
	}
}
