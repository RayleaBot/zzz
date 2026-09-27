package app

import (
	"strconv"
	"strings"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// characterRead is one zzz.character request a refresh sent.
type characterRead struct {
	at         time.Time
	id         string
	delegation string
}

// characterHost answers the account service as a role with any character:
// each zzz.character request is recorded, delegation.create grants "d1",
// and fail fails the request of that character.
func characterHost(t *testing.T, clock *fakeClock, fail string) (*fakeHost, *[]characterRead, *[]map[string]any) {
	reads, delegations := &[]characterRead{}, &[]map[string]any{}
	host := &fakeHost{call: func(request rayleabot.ServiceCallRequest, out any) error {
		switch request.Method {
		case "delegation.create":
			*delegations = append(*delegations, request.Params)
			return decodeObject(map[string]any{"delegation": map[string]any{"ref": "d1"}}, out)
		case "execute":
			list, _ := asObject(request.Params["input"])["id_list"].([]any)
			if request.Params["operation"] != "zzz.character" || len(list) != 1 {
				t.Fatalf("read %v %v", request.Params["operation"], request.Params["input"])
			}
			id := asText(list[0])
			*reads = append(*reads, characterRead{at: clock.Now(), id: id, delegation: asText(request.Params["delegation_ref"])})
			if id == fail {
				return gameError("upstream_rejected", "米游社拒绝了本次请求，请稍后重试。")
			}
			return decodeObject(QueryResult{Role: Role{UID: "10000001"}, Data: map[string]any{"avatar_list": []any{map[string]any{"id": id, "level": 60}}}}, out)
		}
		t.Fatalf("unexpected %s", request.Method)
		return nil
	}}
	return host, reads, delegations
}

func characterIDs(n int) []string {
	ids := []string{}
	for i := range n {
		ids = append(ids, strconv.Itoa(1011+10*i))
	}
	return ids
}

func TestRoleIntervalFollowsZZZPlugin(t *testing.T) {
	if got := settings(&rayleabot.EventContext{}).roleInterval(); got != 3*time.Second {
		t.Fatalf("default = %v", got)
	}
	for value, want := range map[int]time.Duration{0: 100 * time.Millisecond, 20: 100 * time.Millisecond, 1500: 1500 * time.Millisecond} {
		if got := (Settings{PanelRoleInterval: value}).roleInterval(); got != want {
			t.Errorf("%d ms = %v", value, got)
		}
	}
}

// ZZZ-Plugin reads each character in a request of its own and waits
// roleInterval after it; what the event cannot read continues on the role's
// scheduled task with an account delegation, spaced the same across the
// handover, and answers in the chat the command came from.
func TestPanelRefreshReadsOneCharacterAtATimeAndContinuesOnItsTask(t *testing.T) {
	a := pluginApp(t)
	clock := &fakeClock{at: time.Unix(1_800_000_000, 0)}
	a.clock = clock
	start := clock.Now()
	host, reads, delegations := characterHost(t, clock, "")
	ids := characterIDs(30)
	refresh := &panelRefresh{uid: "10000001", choice: Selection{AccountRef: "account", RoleRef: "role"}, provider: "p", player: ShowcaseProfile{Nickname: "绳匠", Level: 50}, ids: ids, interval: 3 * time.Second}
	ref := panelTaskID("zzz", "p", syncOwner(chatEvent()), refresh.choice)
	if other := (Subject{SourceProtocol: "onebot11", SourceAdapter: "a", BotID: "bot", ActorID: "v"}); panelTaskID("zzz", "p", other, refresh.choice) == ref || len(ref) > 128 {
		t.Fatal("users share a refresh task, or its ID is too long for a delegation")
	}
	task := a.beginChatTask(chatEvent(), ref, "绝区零更新面板", "panel_refresh", time.Hour, refresh)
	if task == nil || a.beginChatTask(chatEvent(), ref, "绝区零更新面板", "panel_refresh", time.Hour, &panelRefresh{}) != nil {
		t.Fatal("a role's refresh started while another was reading")
	}
	if _, done, err := a.stepChatTask(t.Context(), host, task, start.Add(chatTaskBudget)); done || err != nil {
		t.Fatal("the event did not hand over", done, err)
	}
	// The event reads until 40 seconds after the command: at 0, 3 … 39 s.
	if len(*reads) != 14 || (*reads)[13].at != start.Add(39*time.Second) || (*reads)[13].delegation != "" {
		t.Fatalf("event reads %d, last %+v", len(*reads), (*reads)[len(*reads)-1])
	}
	grant := (*delegations)[0]
	if len(*delegations) != 1 || grant["task_id"] != ref || grant["operation"] != "zzz.character" || grant["days"] != 1 || grant["account_ref"] != "account" || grant["role_ref"] != "role" {
		t.Fatalf("delegations = %v", *delegations)
	}
	if len(host.scheduled) != 1 || host.scheduled[0].TaskID != ref || host.scheduled[0].Cron != "* * * * *" {
		t.Fatalf("scheduled = %+v", host.scheduled)
	}
	// A trigger a second after the handover still waits the interval.
	clock.set(start.Add(40 * time.Second))
	a.continueChatTask(t.Context(), host, ref)
	if next := (*reads)[14]; next.at != start.Add(42*time.Second) || next.delegation != "d1" {
		t.Fatalf("first task read %+v", next)
	}
	if len(host.sent) != 0 || len(host.deleted) != 0 {
		t.Fatal("the refresh ended early")
	}
	clock.set(start.Add(100 * time.Second))
	a.continueChatTask(t.Context(), host, ref)
	if len(*reads) != len(ids) {
		t.Fatalf("read %d of %d", len(*reads), len(ids))
	}
	for i, read := range *reads {
		if read.id != ids[i] {
			t.Fatalf("read %d is %s", i, read.id)
		}
		if i > 0 && read.at.Sub((*reads)[i-1].at) < 3*time.Second {
			t.Fatalf("reads %d and %d are %v apart", i-1, i, read.at.Sub((*reads)[i-1].at))
		}
		if i >= 14 && read.delegation != "d1" {
			t.Fatalf("task read %d without the delegation", i)
		}
	}
	if len(host.deleted) != 1 || host.deleted[0] != ref || len(host.sent) != 1 {
		t.Fatal("the finished refresh did not answer once", host.deleted, host.sent)
	}
	sent := host.sent[0]
	if sent.SourceProtocol != "onebot11" || sent.SourceAdapter != "a" || sent.TargetType != "group" || sent.TargetID != "g" || !strings.Contains(asText(sent.Message.Segments[0].Data["text"]), "面板列表") {
		t.Fatalf("answered %+v", sent)
	}
	saved, err := a.Profiles.Read("10000001")
	if err != nil || len(saved.Panels) != len(ids) || saved.Nickname != "绳匠" || saved.RefreshedAtMS == 0 {
		t.Fatalf("kept %d panels, %+v", len(saved.Panels), err)
	}
}

// As upstream, a failed read ends the refresh without keeping what was read.
func TestPanelRefreshFailureKeepsNothing(t *testing.T) {
	a := pluginApp(t)
	clock := &fakeClock{at: time.Unix(1_800_000_000, 0)}
	a.clock = clock
	host, reads, _ := characterHost(t, clock, "1021")
	refresh := &panelRefresh{uid: "10000001", choice: Selection{AccountRef: "account", RoleRef: "role"}, provider: "p", ids: characterIDs(3), interval: 100 * time.Millisecond}
	reply, done := refresh.step(t.Context(), a, host, clock.Now().Add(chatTaskBudget))
	if !done || len(*reads) != 2 || !strings.HasPrefix(asText(reply[0].Data["text"]), "面板列表更新失败") {
		t.Fatalf("done %v after %d reads: %v", done, len(*reads), reply)
	}
	if saved, _ := a.Profiles.Read("10000001"); len(saved.Panels) != 0 {
		t.Fatal("a failed refresh kept panels")
	}
}

// The management page asks an agent's details one agent a request.
func TestManagementCharacterQueryAsksOneAgent(t *testing.T) {
	a := pluginApp(t)
	for _, input := range []map[string]any{{}, {"id_list": []any{"1011", "1021"}}} {
		_, err := a.Manage(t.Context(), &rayleabot.EventContext{}, "query", map[string]any{"operation": "zzz.character", "account_ref": "account", "role_ref": "role", "input": input})
		if PublicError(err).Code != "plugin.game_input_invalid" {
			t.Fatalf("%v: %v", input, err)
		}
	}
}
