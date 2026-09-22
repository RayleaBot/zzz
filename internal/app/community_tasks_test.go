package app

import (
	"fmt"
	"testing"
	"time"
)

func TestCommunityPlanRespectsSelectedActionsAndStopsOnce(t *testing.T) {
	store := reminderStore(t.TempDir())
	now := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC).UnixMilli()
	task := Reminder{Ref: "community", Kind: "community", Once: true, Enabled: true, Hour: 8, ExpiresAtMS: now + int64(24*time.Hour/time.Millisecond), Community: CommunityPlan{Read: true, Like: true, Unlike: true, Share: true}}
	if err := seedReminders(store, []Reminder{task}); err != nil {
		t.Fatal(err)
	}
	actions := []string{}
	reads := 0
	query := func(r Reminder) (QueryResult, error) {
		step := r.Community.Steps[r.Community.Cursor]
		actions = append(actions, step.Action)
		if step.Action == "status" {
			reads++
			remaining := 20
			if reads > 1 {
				remaining = 0
			}
			return QueryResult{Data: map[string]any{"can_get_points": remaining, "states": []any{map[string]any{"mission_id": 59, "happened_times": 1}, map[string]any{"mission_id": 60, "happened_times": 3}}}}, nil
		}
		if step.Action == "posts" {
			posts := []any{}
			for i := 0; i < 5; i++ {
				posts = append(posts, map[string]any{"post_id": fmt.Sprint(100 + i)})
			}
			return QueryResult{Data: map[string]any{"posts": posts}}, nil
		}
		return QueryResult{Data: map[string]any{"accepted": true}}, nil
	}
	sent := 0
	send := func(Reminder, string) error { sent++; return nil }
	for range 20 {
		if err := store.Tick(task.Ref, now, query, send, Game{ID: "zzz"}); err != nil {
			t.Fatal(err)
		}
		now += int64(time.Minute / time.Millisecond)
	}
	counts := map[string]int{}
	for _, action := range actions {
		counts[action]++
	}
	if counts["read"] != 2 || counts["like"] != 2 || counts["unlike"] != 2 || counts["share"] != 1 || counts["sign"] != 1 || sent != 0 {
		t.Fatal(actions, sent)
	}
	items, _ := store.List()
	if items[0].Enabled || items[0].Community.State != "completed" || items[0].LastCode != "completed" {
		t.Fatal(items)
	}
	store = &ReminderStore{taskFiles[Reminder]{Directory: store.Directory}}
	before := len(actions)
	_ = store.Tick(task.Ref, now, query, send, Game{ID: "zzz"})
	if len(actions) != before {
		t.Fatal("completed once task replayed after restart")
	}
}
func TestCommunityUncertainWriteIsNotReplayedAfterCrash(t *testing.T) {
	store := reminderStore(t.TempDir())
	now := time.Now().UnixMilli()
	task := Reminder{Ref: "community", Kind: "community", Once: true, Enabled: true, ExpiresAtMS: now + 100000, Community: CommunityPlan{State: "running", Steps: []CommunityStep{{Action: "sign"}}, Cursor: 1, Pending: true}}
	_ = seedReminders(store, []Reminder{task})
	queries := 0
	_ = store.Tick(task.Ref, now, func(Reminder) (QueryResult, error) {
		queries++
		return QueryResult{}, nil
	}, func(Reminder, string) error {
		return nil
	}, Game{ID: "zzz"})
	items, _ := store.List()
	if queries != 0 || items[0].Enabled || items[0].LastCode != "step_outcome_unknown" {
		t.Fatal(queries, items)
	}
}
