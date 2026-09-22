package app

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestChallengeReminderScheduleUnknownAndRestart(t *testing.T) {
	china := time.FixedZone("UTC+8", 28800)
	now := time.Date(2026, 9, 21, 20, 0, 0, 0, china).UnixMilli()
	if want := time.Date(2026, 9, 28, 20, 0, 0, 0, china).UnixMilli(); nextChallengeCheck(now, 20, 0, 1) != want {
		t.Fatal("weekly duplicate admission")
	}
	s := reminderStore(t.TempDir())
	task := Reminder{Ref: "task", Kind: "challenge", ChallengeKind: "deadly", Metric: "star", Threshold: 9, Enabled: true, Hour: 20, ExpiresAtMS: now + int64(7*24*time.Hour/time.Millisecond)}
	if err := seedReminders(s, []Reminder{task}); err != nil {
		t.Fatal(err)
	}
	var queries, sends atomic.Int32
	data := map[string]any{"start_time": 1760000000, "total_star": 6, "total_score": 200}
	query := func(Reminder) (QueryResult, error) { queries.Add(1); return QueryResult{Data: data}, nil }
	send := func(Reminder, string) error { sends.Add(1); return errors.New("synthetic uncertain result") }
	tick := func() {
		if err := s.Tick("task", now, query, send, Game{ID: "zzz"}); err != nil {
			t.Error(err)
		}
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(tick)
	}
	wg.Wait()
	if queries.Load() != 1 || sends.Load() != 1 {
		t.Fatal("repeated notification")
	}
	s = &ReminderStore{taskFiles[Reminder]{Directory: s.Directory}}
	tick()
	if sends.Load() != 1 {
		t.Fatal("restart resent")
	}
	items, _ := s.List()
	now = items[0].NextCheckMS
	data = map[string]any{"start_time": 1760000000, "total_star": 9}
	tick()
	items, _ = s.List()
	if items[0].LastCode != "target_met" || sends.Load() != 1 {
		t.Fatal(items)
	}
	now = items[0].NextCheckMS
	data = map[string]any{"start_time": 1760000000}
	tick()
	if sends.Load() != 1 {
		t.Fatal("missing treated as zero")
	}
	items, _ = s.List()
	now = items[0].NextCheckMS
	data = map[string]any{"has_data": false}
	tick()
	if sends.Load() != 2 {
		t.Fatal("explicitly unstarted challenge was not reminded")
	}
}
