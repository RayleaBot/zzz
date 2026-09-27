package app

import (
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func TestReminderPersistsAdmissionAndThresholdRearming(t *testing.T) {
	store := reminderStore(t.TempDir())
	now := time.Now().UnixMilli()
	task := Reminder{Ref: "task", Enabled: true, Armed: true, Threshold: 80, ExpiresAtMS: now + int64(24*time.Hour/time.Millisecond)}
	if err := seedReminders(store, []Reminder{task}); err != nil {
		t.Fatal(err)
	}
	var checks, sent atomic.Int32
	current := 180.0
	query := func(Reminder) (QueryResult, error) {
		checks.Add(1)
		return QueryResult{Data: map[string]any{"energy": map[string]any{"progress": map[string]any{"current": current, "max": 200}}}}, nil
	}
	send := func(Reminder, string) error { sent.Add(1); return nil }
	tick := func(at int64) {
		t.Helper()
		if err := store.Tick("task", at, query, send, Game{ID: "zzz"}); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() { tick(now) })
	}
	wg.Wait()
	if checks.Load() != 1 || sent.Load() != 1 {
		t.Fatal("duplicate concurrent notification")
	}
	store = &ReminderStore{taskFiles[Reminder]{Directory: store.Directory}}
	tick(now + int64(11*time.Minute/time.Millisecond))
	if sent.Load() != 1 {
		t.Fatal("restart repeated high-level notification")
	}
	current = 80
	tick(now + int64(22*time.Minute/time.Millisecond))
	current = 190
	tick(now + int64(33*time.Minute/time.Millisecond))
	if sent.Load() != 2 {
		t.Fatal("threshold did not rearm")
	}
	tick(task.ExpiresAtMS)
	items, _ := store.List()
	if items[0].Enabled || items[0].LastCode != "expired" {
		t.Fatal(items)
	}
}
func TestReminderVerificationBackoffAndFailedSendNotRepeated(t *testing.T) {
	store := reminderStore(t.TempDir())
	now := time.Now().UnixMilli()
	task := Reminder{Ref: "task", Enabled: true, Armed: true, Threshold: 80, ExpiresAtMS: now + int64(72*time.Hour/time.Millisecond)}
	_ = seedReminders(store, []Reminder{task})
	sent := 0
	send := func(Reminder, string) error { sent++; return errors.New("synthetic send failure") }
	query := func(Reminder) (QueryResult, error) {
		return QueryResult{}, &rayleabot.ActionError{Code: "plugin.upstream_device_required"}
	}
	if err := store.Tick("task", now, query, send, Game{ID: "zzz"}); err != nil {
		t.Fatal(err)
	}
	items, _ := store.List()
	if items[0].NextCheckMS < now+int64(24*time.Hour/time.Millisecond) || sent != 0 {
		t.Fatal(items)
	}
	query = func(Reminder) (QueryResult, error) {
		return QueryResult{Data: map[string]any{"energy": map[string]any{"progress": map[string]any{"current": 240, "max": 240}}}}, nil
	}
	_ = store.Tick("task", items[0].NextCheckMS, query, send, Game{ID: "zzz"})
	items, _ = store.List()
	if items[0].LastCode != "notification_failed" || items[0].Armed {
		t.Fatal(items)
	}
	_ = store.Tick("task", items[0].NextCheckMS, query, send, Game{ID: "zzz"})
	if sent != 1 {
		t.Fatal("ambiguous send retried")
	}
}

// seedReminders stores tasks as the management actions would.
func seedReminders(s *ReminderStore, tasks []Reminder) error {
	return s.edit("", func(items *[]Reminder, _ int) error {
		*items = append(*items, tasks...)
		return nil
	})
}

// A trigger holding its task does not block edits of others, and a task
// removed meanwhile is not written back.
func TestReminderStoreKeepsTasksApart(t *testing.T) {
	directory := t.TempDir()
	store := reminderStore(directory)
	if err := seedReminders(store, []Reminder{{Ref: "a", Enabled: true}, {Ref: "b"}}); err != nil {
		t.Fatal(err)
	}
	items, err := store.List()
	if err != nil || len(items) != 2 || items[0].Ref != "a" {
		t.Fatal(items, err)
	}
	claimed, ok, err := store.claim("a")
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	if _, again, _ := store.claim("a"); again {
		t.Fatal("a running task was claimed twice")
	}
	if err = store.edit("a", func(items *[]Reminder, i int) error { *items = slices.Delete(*items, i, i+1); return nil }); err != nil {
		t.Fatal(err)
	}
	claimed.LastCode = "late"
	if err = store.save(claimed); !errors.Is(err, errTaskChanged) {
		t.Fatal("a removed task was written back", err)
	}
	store.release("a")
	if items, _ = reminderStore(directory).List(); len(items) != 1 || items[0].Ref != "b" {
		t.Fatal(items)
	}
}
