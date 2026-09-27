package app

import (
	"context"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// Long flows, such as 更新面板 reading an account's characters one by one, run
// to the end in the event that asks for them. They move the event to the
// background with event.detach, which ends its delivery at once and frees
// the chat, and keep its request ID and origin, so the account plugin still
// sees the user who asked, until the host's background deadline.

// busyReply answers a flow the host would not move to the background because
// this plugin already holds as many background events as it allows.
const busyReply = "后台任务较多，请稍后再试"

// detachFailure is why an event could not move to the background.
func detachFailure(err error) *rayleabot.ActionError {
	if PublicError(err).Code == "platform.rate_limited" {
		return gameError("background_busy", busyReply)
	}
	return PublicError(err)
}

// detachChat moves a chat event to the background; false once the chat has
// been told why it could not.
func detachChat(ctx context.Context, event *rayleabot.EventContext) (bool, error) {
	if _, err := event.Detach(ctx, nil); err != nil {
		return false, event.SendText(detachFailure(err).Message)
	}
	return true, nil
}

// flows are the long flows running now, by a key each runs under once at a
// time.
type flows struct {
	mu      sync.Mutex
	running map[string]bool
}

// begin claims key; false while another flow holds it.
func (f *flows) begin(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.running[key] {
		return false
	}
	if f.running == nil {
		f.running = map[string]bool{}
	}
	f.running[key] = true
	return true
}

// end releases key.
func (f *flows) end(key string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.running, key)
}

// clock is the time flows wait between their requests by and daily syncs
// are due by; tests set their own.
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
