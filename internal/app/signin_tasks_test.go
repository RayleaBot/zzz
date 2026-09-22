package app

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestScheduledSignUsesChinaDayAndSurvivesRestart(t *testing.T) {
	s := reminderStore(t.TempDir())
	now := time.Date(2026, 9, 19, 23, 59, 0, 0, time.UTC)
	task := Reminder{Ref: "sign", Kind: "signin", Hour: 8, Enabled: true, ExpiresAtMS: now.Add(72 * time.Hour).UnixMilli()}
	if err := seedReminders(s, []Reminder{task}); err != nil {
		t.Fatal(err)
	}
	var calls, notices atomic.Int32
	query := func(Reminder) (QueryResult, error) {
		calls.Add(1)
		return QueryResult{Data: map[string]any{"signed": true, "outcome": "signed"}}, nil
	}
	send := func(Reminder, string) error { notices.Add(1); return nil }
	tick := func(at time.Time) {
		t.Helper()
		if err := s.Tick(t.Context(), "sign", at.UnixMilli(), query, send, Game{ID: "genshin"}); err != nil {
			t.Fatal(err)
		}
	}
	tick(now)
	if calls.Load() != 0 {
		t.Fatal("ran before Beijing hour")
	}
	tick(now.Add(time.Minute))
	if calls.Load() != 1 || notices.Load() != 0 {
		t.Fatal("scheduled sign or default notification incorrect")
	}
	s = &ReminderStore{taskFiles[Reminder]{Directory: s.Directory}}
	tick(now.Add(time.Hour))
	if calls.Load() != 1 {
		t.Fatal("restart repeated today's sign")
	}
	tick(now.Add(24*time.Hour + time.Minute))
	if calls.Load() != 2 {
		t.Fatal("next Beijing day not checked")
	}
}
func TestScheduledSignFailureIsNotReplayedWithinDay(t *testing.T) {
	s := reminderStore(t.TempDir())
	now := time.Date(2026, 9, 19, 2, 0, 0, 0, time.UTC)
	_ = seedReminders(s, []Reminder{{Ref: "sign", Kind: "signin", Hour: 0, Enabled: true, Notify: true, ExpiresAtMS: now.Add(48 * time.Hour).UnixMilli()}})
	calls, notices := 0, 0
	query := func(Reminder) (QueryResult, error) {
		calls++
		return QueryResult{}, gameError("sign_unknown", "synthetic sign response unknown")
	}
	send := func(Reminder, string) error { notices++; return nil }
	for _, at := range []time.Time{now, now.Add(11 * time.Minute), now.Add(time.Hour)} {
		if err := s.Tick(t.Context(), "sign", at.UnixMilli(), query, send, Game{ID: "genshin"}); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 || notices != 1 {
		t.Fatal("ambiguous write or notification repeated")
	}
}
