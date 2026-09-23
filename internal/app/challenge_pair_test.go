package app

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
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
	expires := now + int64(7*24*time.Hour/time.Millisecond)
	tasks := []Reminder{
		{Ref: "pair", Kind: "challenge", ChallengeKind: "challenge", Metric: "s_layers", Threshold: 5, DeadlyThreshold: 9, Pair: true, Owner: owner, Enabled: true, Hour: 20, ExpiresAtMS: expires},
		{Ref: "other", Kind: "challenge", ChallengeKind: "challenge", Metric: "s_layers", Pair: true, Owner: other, Enabled: true, Hour: 20, ExpiresAtMS: expires},
		{Ref: "detail", Kind: "challenge", ChallengeKind: "deadly", Metric: "star", Threshold: 9, Owner: owner, Enabled: true, Hour: 20, ExpiresAtMS: expires},
	}
	if err := seedReminders(a.Reminders, tasks); err != nil {
		t.Fatal(err)
	}
	abyss := map[string]any{"hadal_info_v2": map[string]any{"brief": map[string]any{"rating": "A"}, "fourth_layer_detail": map[string]any{"rating": "S"}}}
	data := map[string]any{"challenge": abyss, "deadly": map[string]any{"has_data": true, "total_star": json.Number("7")}}
	queries, sent := 0, []string{}
	query := func(Reminder) (QueryResult, error) { queries++; return QueryResult{Data: data}, nil }
	send := func(_ Reminder, text string) error { sent = append(sent, text); return nil }
	for _, ref := range []string{"pair", "other"} {
		if err := a.Reminders.Tick(ref, now, query, send, Game{ID: "zzz"}); err != nil {
			t.Fatal(err)
		}
	}
	// Both modes arrive in one message; thresholds of 0 check nothing.
	if queries != 1 || len(sent) != 1 || sent[0] != "【式舆/危局挑战提醒】\n式舆防卫战S评级: 4/5\n第五层评价: A\n危局强袭战星数: 7/9" {
		t.Fatalf("queries %d sent %q", queries, sent)
	}
	byRef := func() map[string]Reminder {
		items, _ := a.Reminders.List()
		out := map[string]Reminder{}
		for _, task := range items {
			out[task.Ref] = task
		}
		return out
	}
	// A mode that could not be read is named in the same message.
	data = map[string]any{"deadly": map[string]any{"has_data": true, "total_star": json.Number("9")}, "errors": map[string]any{"challenge": "账号插件未运行"}}
	if err := a.Reminders.Tick("pair", byRef()["pair"].NextCheckMS, query, send, Game{ID: "zzz"}); err != nil {
		t.Fatal(err)
	}
	if len(sent) != 2 || sent[1] != "【式舆/危局挑战提醒】\n式舆防卫战查询失败: 账号插件未运行" {
		t.Fatalf("sent %q", sent)
	}

	// The user's own time and threshold move only that user's task.
	three := 3
	if _, err := a.ChallengePrefs.Edit(owner, func(p *ChallengePreference) bool {
		p.DeadlyStars, p.RemindTime = &three, "每周一8时30分"
		return true
	}); err != nil {
		t.Fatal(err)
	}
	global := Settings{ChallengeRemind: true, ChallengeRemindTime: "每日21时", ChallengeAbyssLevel: 6, ChallengeDeadlyStars: 4}.challengeGlobals()
	if err := a.syncChallengePairs(global, &owner); err != nil {
		t.Fatal(err)
	}
	if task := byRef()["pair"]; task.Threshold != 6 || task.DeadlyThreshold != 3 || task.Hour != 8 || task.Minute != 30 || task.Weekday != 1 || task.NextCheckMS != nextChallengeCheck(time.Now().UnixMilli(), 8, 30, 1) {
		t.Fatalf("own settings not applied: %+v", task)
	}
	if task := byRef()["other"]; task.Threshold != 0 || task.Hour != 20 {
		t.Fatalf("another user's task moved: %+v", task)
	}
	// A global change reaches users without their own values; detailed
	// reminders keep theirs.
	if err := a.syncChallengePairs(global, nil); err != nil {
		t.Fatal(err)
	}
	if task := byRef()["other"]; task.Threshold != 6 || task.DeadlyThreshold != 4 || task.Hour != 21 {
		t.Fatalf("global settings not applied: %+v", task)
	}
	if task := byRef()["detail"]; task.Threshold != 9 || task.Hour != 20 {
		t.Fatalf("a detailed reminder moved: %+v", task)
	}
	// Values out of range fall back to upstream's defaults.
	if global := (Settings{ChallengeRemindTime: "25时", ChallengeAbyssLevel: 7, ChallengeDeadlyStars: -1}).challengeGlobals(); global.Time != "每日20时" || global.Abyss != 5 || global.Deadly != 6 {
		t.Fatalf("defaults = %+v", global)
	}
}

// pairCaller answers the account service for a pair: roles, the account,
// delegations (failing for failOperation) and reads.
type pairCaller struct {
	failOperation string
	revoked       []string
	reads         map[string]string
}

func (c *pairCaller) CallService(_ context.Context, req rayleabot.ServiceCallRequest, out any) error {
	role := Role{Ref: "role", Game: "zzz", UID: "10000001"}
	switch req.Method {
	case "roles":
		return decodeObject(map[string]any{"roles": []Role{role}}, out)
	case "list":
		return decodeObject(Accounts{Items: []Account{{Ref: "account", Owner: Subject{"onebot11", "a", "bot", "1"}, Roles: []Role{role}}}}, out)
	case "delegation.create":
		operation := asText(req.Params["operation"])
		if operation == c.failOperation {
			return gameError("operation_denied", "synthetic")
		}
		return decodeObject(map[string]any{"delegation": map[string]any{"ref": "d-" + operation, "expires_at_ms": 1}}, out)
	case "delegation.revoke":
		c.revoked = append(c.revoked, asText(req.Params["delegation_ref"]))
		return nil
	case "execute":
		operation := asText(req.Params["operation"])
		c.reads[operation] = asText(req.Params["delegation_ref"])
		if operation == c.failOperation {
			return gameError("upstream_unavailable", "官方暂不可用")
		}
		return decodeObject(QueryResult{Role: role, Data: map[string]any{"has_data": true}}, out)
	}
	return gameError("operation_denied", "unexpected")
}

func TestChallengePairTaskHoldsBothDelegations(t *testing.T) {
	a := &App{Game: Game{ID: "zzz", Name: "绝区零"}, Reminders: reminderStore(t.TempDir())}
	choice := Selection{"account", "role"}
	scheduled := 0
	schedule := func(context.Context, rayleabot.SchedulerCreateRequest) error { scheduled++; return nil }
	// A second delegation that fails takes the first and the task with it.
	caller := &pairCaller{failOperation: "zzz.deadly", reads: map[string]string{}}
	if _, err := a.createChallengePair(t.Context(), AccountsClient{Caller: caller, Provider: "p", Game: "zzz"}, schedule, choice, 5, 6, "每日20时"); err == nil {
		t.Fatal("a pair without its 危局 delegation was created")
	}
	if items, _ := a.Reminders.List(); len(items) != 0 || scheduled != 0 || !reflect.DeepEqual(caller.revoked, []string{"d-zzz.challenge"}) {
		t.Fatalf("left behind: tasks %v scheduled %d revoked %v", items, scheduled, caller.revoked)
	}
	caller = &pairCaller{reads: map[string]string{}}
	client := AccountsClient{Caller: caller, Provider: "p", Game: "zzz"}
	task, err := a.createChallengePair(t.Context(), client, schedule, choice, 5, 6, "每日20时")
	if err != nil || scheduled != 1 || task.DelegationRef != "d-zzz.challenge" || task.DeadlyDelegationRef != "d-zzz.deadly" || task.DeadlyThreshold != 6 || !task.Pair {
		t.Fatalf("task %+v scheduled %d err %v", task, scheduled, err)
	}
	if items, _ := a.Reminders.List(); len(items) != 1 || !items[0].Enabled {
		t.Fatalf("tasks = %+v", items)
	}
	// Each mode is read under its own delegation; a failed one is reported
	// beside the other's data.
	caller.failOperation = "zzz.deadly"
	result, err := a.queryChallengePair(t.Context(), client, task)
	if err != nil || caller.reads["zzz.challenge"] != "d-zzz.challenge" || caller.reads["zzz.deadly"] != "d-zzz.deadly" || result.Data["challenge"] == nil || asObject(result.Data["errors"])["deadly"] != "官方暂不可用" {
		t.Fatalf("result %+v reads %v err %v", result, caller.reads, err)
	}
	task.Threshold = 0
	if _, err = a.queryChallengePair(t.Context(), client, task); err == nil {
		t.Fatal("a pair whose only read failed succeeded")
	}
}
