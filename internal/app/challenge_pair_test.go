package app

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestChallengePairFollowsZZZPluginCounts(t *testing.T) {
	china := time.FixedZone("UTC+8", 8*3600)
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, china)
	abyss := func(raw string) map[string]any {
		var data map[string]any
		if err := json.Unmarshal([]byte(`{"hadal_info_v2":`+raw+`}`), &data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	period := `"schedule_id":62,"hadal_begin_time":{"year":2026,"month":9,"day":15,"hour":4,"minute":0,"second":0},"hadal_end_time":{"year":2026,"month":9,"day":29,"hour":3,"minute":59,"second":59}`
	// Starting on the fourth floor with S counts the three below it.
	skipped := abyss(`{` + period + `,"brief":{"rating":"S","score":30000},"fourth_layer_detail":{"rating":"S"}}`)
	if count, fifth := abyssSLayers(asObject(skipped["hadal_info_v2"])); count != 5 || fifth != "S" {
		t.Fatalf("skipped floors counted %d %s", count, fifth)
	}
	// Starting on the second floor with S counts the first.
	if count, _ := abyssSLayers(asObject(abyss(`{"brief":{"rating":"A"},"second_layer_detail":{"rating":"S"},"third_layer_detail":{"rating":"A"},"fourth_layer_detail":{"rating":"S"}}`)["hadal_info_v2"])); count != 3 {
		t.Fatalf("second floor start counted %d", count)
	}
	// 5 is met by five S floors; 6 also wants S+ on the fifth.
	if lines := challengePairLines("challenge", 5, skipped, false, now); len(lines) != 0 {
		t.Fatalf("met target reminded: %v", lines)
	}
	want := []string{"式舆防卫战S评级: 5/5", "第五层评价: S", "统计周期：2026/9/15 - 2026/9/29", "刷新剩余: 8天15小时"}
	if lines := challengePairLines("challenge", 6, skipped, false, now); !reflect.DeepEqual(lines, want) {
		t.Fatalf("threshold 6 lines = %q", lines)
	}
	if lines := challengePairLines("challenge", 5, skipped, true, now); lines[0] != "式舆防卫战S评级: 5/5 ✓" {
		t.Fatalf("status lines = %q", lines)
	}
	if lines := challengePairLines("challenge", 3, map[string]any{}, false, now); !reflect.DeepEqual(lines, []string{"式舆防卫战S评级: 0/5"}) {
		t.Fatalf("missing record = %q", lines)
	}
	deadly := map[string]any{"has_data": true, "total_star": json.Number("7"), "start_time": map[string]any{"year": 2026, "month": 9, "day": 15}, "end_time": map[string]any{"year": 2026, "month": 9, "day": 20, "hour": 11}}
	if lines := challengePairLines("deadly", 6, deadly, false, now); len(lines) != 0 {
		t.Fatalf("met stars reminded: %v", lines)
	}
	if lines := challengePairLines("deadly", 9, deadly, false, now); !reflect.DeepEqual(lines, []string{"危局强袭战星数: 7/9", "统计周期：2026/9/15 - 2026/9/20", "刷新剩余: 0天0小时"}) {
		t.Fatalf("deadly lines = %q", lines)
	}
	if lines := challengePairLines("deadly", 0, deadly, true, now); lines[0] != "危局强袭战星数: 7/9" {
		t.Fatalf("threshold 0 status = %q", lines)
	}
	// The detailed reminder's 6 means the same as the pair's.
	kind, _ := challengeKind("challenge")
	if _, met, known, err := challengeTarget(kind, "s_layers", 6, skipped); err != nil || !known || met {
		t.Fatalf("s_layers 6 met=%v known=%v err=%v", met, known, err)
	}
	if challengePairDescription(6, 0) != "防卫战S评级<5层或第五层<S+评价进行提醒，危局强袭战不提醒" {
		t.Fatal(challengePairDescription(6, 0))
	}
}

func TestChallengePairTicksAndFollowsSettings(t *testing.T) {
	directory := t.TempDir()
	a := &App{Reminders: reminderStore(directory), ChallengePrefs: challengePreferences(directory)}
	china := time.FixedZone("UTC+8", 8*3600)
	now := time.Date(2026, 9, 21, 20, 0, 0, 0, china).UnixMilli()
	owner, other := Subject{"onebot11", "a", "bot", "1"}, Subject{"onebot11", "a", "bot", "2"}
	expires := now + int64(24*time.Hour/time.Millisecond)
	tasks := []Reminder{
		{Ref: "abyss", Kind: "challenge", ChallengeKind: "challenge", Metric: "s_layers", Threshold: 5, Pair: true, Owner: owner, Enabled: true, Hour: 20, ExpiresAtMS: expires},
		{Ref: "deadly", Kind: "challenge", ChallengeKind: "deadly", Metric: "star", Threshold: 0, Pair: true, Owner: owner, Enabled: true, Hour: 20, ExpiresAtMS: expires},
		{Ref: "other", Kind: "challenge", ChallengeKind: "deadly", Metric: "star", Threshold: 6, Pair: true, Owner: other, Enabled: true, Hour: 20, ExpiresAtMS: expires},
		{Ref: "detail", Kind: "challenge", ChallengeKind: "deadly", Metric: "star", Threshold: 9, Owner: owner, Enabled: true, Hour: 20, ExpiresAtMS: expires},
	}
	if err := seedReminders(a.Reminders, tasks); err != nil {
		t.Fatal(err)
	}
	queries, sent := 0, []string{}
	query := func(Reminder) (QueryResult, error) {
		queries++
		return QueryResult{Data: map[string]any{"hadal_info_v2": map[string]any{"brief": map[string]any{"rating": "A"}, "fourth_layer_detail": map[string]any{"rating": "S"}}}}, nil
	}
	send := func(_ Reminder, text string) error { sent = append(sent, text); return nil }
	for _, ref := range []string{"abyss", "deadly"} {
		if err := a.Reminders.Tick(ref, now, query, send, Game{ID: "zzz"}); err != nil {
			t.Fatal(err)
		}
	}
	// A threshold of 0 checks nothing; the unmet one sends upstream's lines.
	if queries != 1 || len(sent) != 1 || !strings.HasPrefix(sent[0], "【式舆/危局挑战提醒】\n式舆防卫战S评级: 4/5") {
		t.Fatalf("queries %d sent %q", queries, sent)
	}

	// The user's own time and threshold move only that user's pair.
	nine := 9
	if _, err := a.ChallengePrefs.Edit(owner, func(p *ChallengePreference) bool {
		p.DeadlyStars, p.RemindTime = &nine, "每周一8时30分"
		return true
	}); err != nil {
		t.Fatal(err)
	}
	global := Settings{ChallengeRemind: true, ChallengeRemindTime: "每日21时", ChallengeAbyssLevel: 6, ChallengeDeadlyStars: 3}.challengeGlobals()
	if err := a.syncChallengePairs(global, &owner); err != nil {
		t.Fatal(err)
	}
	items, _ := a.Reminders.List()
	byRef := map[string]Reminder{}
	for _, task := range items {
		byRef[task.Ref] = task
	}
	if task := byRef["deadly"]; task.Threshold != 9 || task.Hour != 8 || task.Minute != 30 || task.Weekday != 1 || task.NextCheckMS != nextChallengeCheck(time.Now().UnixMilli(), 8, 30, 1) {
		t.Fatalf("own settings not applied: %+v", task)
	}
	if task := byRef["abyss"]; task.Threshold != 6 {
		t.Fatalf("global threshold not applied: %+v", task)
	}
	if byRef["other"].Threshold != 6 || byRef["other"].Hour != 20 || byRef["detail"].Threshold != 9 || byRef["detail"].Hour != 20 {
		t.Fatalf("other tasks moved: %+v %+v", byRef["other"], byRef["detail"])
	}
	// A global change reaches users without their own values.
	if err := a.syncChallengePairs(global, nil); err != nil {
		t.Fatal(err)
	}
	items, _ = a.Reminders.List()
	for _, task := range items {
		if task.Ref == "other" && (task.Threshold != 3 || task.Hour != 21) {
			t.Fatalf("global settings not applied: %+v", task)
		}
	}
	// Values out of range fall back to upstream's defaults.
	if global := (Settings{ChallengeRemindTime: "25时", ChallengeAbyssLevel: 7, ChallengeDeadlyStars: -1}).challengeGlobals(); global.Time != "每日20时" || global.Abyss != 5 || global.Deadly != 6 {
		t.Fatalf("defaults = %+v", global)
	}
}
