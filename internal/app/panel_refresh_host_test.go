package app

import (
	"strings"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// accountsService answers the account plugin's service for a role with the
// characters ids: list, the character list, each character's details and
// delegation.create, which grants "d1". A trigger's reads must carry the
// delegation and a chat event's must not. fail, when set, answers a read
// with a failure code.
type accountsService struct {
	t      *testing.T
	clock  *fakeClock
	ids    []string
	reads  []characterRead
	grants []map[string]any
	fail   func(id string, scheduled bool) string
}

func (s *accountsService) answer(request rayleabot.ServiceCallRequest, scheduled bool) (map[string]any, string) {
	if request.TargetPluginID != "raylea.mihoyo-accounts" || request.Service != "accounts" || request.ServiceVersion != 1 {
		s.t.Errorf("call %+v", request)
		return nil, "plugin.service_not_found"
	}
	role := map[string]any{"ref": "role", "game": "zzz", "uid": "10000001", "nickname": "绳匠", "level": 50}
	params := request.Params
	switch request.Method {
	case "list":
		return map[string]any{"items": []any{map[string]any{"ref": "account", "roles": []any{role}, "status": "valid"}}, "default_roles": map[string]any{}, "uid_bindings": map[string]any{}}, ""
	case "delegation.create":
		s.grants = append(s.grants, params)
		return map[string]any{"delegation": map[string]any{"ref": "d1"}}, ""
	case "execute":
		switch params["operation"] {
		case "zzz.characters":
			list := []any{}
			for _, id := range s.ids {
				list = append(list, map[string]any{"id": id})
			}
			return map[string]any{"operation": "zzz.characters", "role": role, "data": map[string]any{"avatar_list": list}}, ""
		case "zzz.character":
			list, _ := asObject(params["input"])["id_list"].([]any)
			if len(list) != 1 {
				s.t.Errorf("read %v", params["input"])
				return nil, "plugin.account_input_invalid"
			}
			id, delegation := asText(list[0]), asText(params["delegation_ref"])
			if scheduled != (delegation == "d1") {
				s.t.Errorf("read of %s with delegation %q, scheduled %v", id, delegation, scheduled)
			}
			s.reads = append(s.reads, characterRead{at: s.clock.Now(), id: id, delegation: delegation})
			if s.fail != nil {
				if code := s.fail(id, scheduled); code != "" {
					return nil, code
				}
			}
			return map[string]any{"operation": "zzz.character", "role": role, "data": map[string]any{"avatar_list": []any{map[string]any{"id": id, "level": 60}}}}, ""
		}
	}
	s.t.Errorf("unexpected %s %v", request.Method, params["operation"])
	return nil, "plugin.method_not_found"
}

// refreshOnHost sends 更新面板 for a role with n characters through the SDK
// and checks that the event read what it could and handed the rest to a
// scheduler job, whose ID it returns.
func refreshOnHost(t *testing.T, n int) (*App, *fakeClock, *accountsService, *sdkHost, string) {
	t.Helper()
	a := pluginApp(t)
	clock := &fakeClock{at: time.Unix(1_800_000_000, 0)}
	a.clock = clock
	accounts := &accountsService{t: t, clock: clock, ids: characterIDs(n)}
	host := newSDKHost(t, a, accounts.answer)
	end, _ := host.message("%更新面板", "更新面板")
	if end["type"] != "result" || len(host.sent) != 1 || sentText(host.sent[0].Message) != "正在更新面板列表，请稍候..." {
		t.Fatalf("the command ended with %v after %d messages", end, len(host.sent))
	}
	// The event reads until 40 seconds after the command: at 0, 3 … 39 s.
	if len(accounts.reads) != 14 || len(accounts.grants) != 1 {
		t.Fatalf("the event read %d characters with %d grants", len(accounts.reads), len(accounts.grants))
	}
	return a, clock, accounts, host, host.job(panelTask)
}

// triggerUntilDone runs the job each minute after start, as the host's
// scheduler does, until the plugin deletes it.
func triggerUntilDone(t *testing.T, clock *fakeClock, host *sdkHost, ref string, start time.Time, limit int) {
	t.Helper()
	for minute := 1; len(host.deleted) == 0; minute++ {
		if minute > limit {
			t.Fatalf("%d triggers of %s did not finish it", limit, ref)
		}
		clock.set(start.Add(time.Duration(minute) * time.Minute))
		end, _ := host.trigger(ref)
		if end["type"] != "result" {
			t.Fatalf("trigger %d ended with %v", minute, end)
		}
	}
}

// 更新面板 of a role with 40 characters, run through the SDK as the host
// runs it: the host's per-minute triggers of the job carry the job's payload
// but not its task ID, and they read the remaining characters, one a request
// spaced as upstream, until the panels are kept and the list is answered in
// the chat the command came from.
func TestPanelRefreshFinishesOnTheHostsTriggers(t *testing.T) {
	a, clock, accounts, host, ref := refreshOnHost(t, 40)
	triggerUntilDone(t, clock, host, ref, time.Unix(1_800_000_000, 0), 5)
	if len(accounts.reads) != 40 {
		t.Fatalf("read %d of 40 characters", len(accounts.reads))
	}
	for i, read := range accounts.reads {
		if read.id != accounts.ids[i] || i > 0 && read.at.Sub(accounts.reads[i-1].at) < 3*time.Second {
			t.Fatalf("read %d is %s at %v", i, read.id, read.at)
		}
	}
	if len(host.deleted) != 1 || host.deleted[0] != ref || len(host.jobs) != 0 || len(host.sent) != 2 {
		t.Fatalf("deleted %v, jobs %v, %d messages", host.deleted, host.jobs, len(host.sent))
	}
	if sent := host.sent[1]; sent.SourceProtocol != "onebot11" || sent.SourceAdapter != "a" || sent.TargetType != "private" || sent.TargetID != "u" || !strings.Contains(sentText(sent.Message), "面板列表") {
		t.Fatalf("answered %+v", sent)
	}
	if saved, err := a.Profiles.Read("10000001"); err != nil || len(saved.Panels) != 40 || saved.Nickname != "绳匠" {
		t.Fatalf("kept %d panels, %v", len(saved.Panels), err)
	}
}

// countReads is how often the character id was asked for.
func countReads(s *accountsService, id string) int {
	count := 0
	for _, read := range s.reads {
		if read.id == id {
			count++
		}
	}
	return count
}

// A trigger whose read is still open when the trigger runs out of time (the
// account service answers only once the host has timed the event out) ends
// its event in time and leaves the task to the next trigger, which asks for
// that character again and finishes the refresh.
func TestPanelRefreshRecoversFromATriggerThatRanOutOfTime(t *testing.T) {
	a, clock, accounts, host, ref := refreshOnHost(t, 20)
	start := time.Unix(1_800_000_000, 0)
	stuck := accounts.ids[16]
	accounts.fail = func(id string, scheduled bool) string {
		if id != stuck || countReads(accounts, id) > 1 {
			return ""
		}
		clock.set(clock.Now().Add(time.Minute))
		return "plugin.event_timeout"
	}
	clock.set(start.Add(time.Minute))
	if end, _ := host.trigger(ref); end["type"] != "result" {
		t.Fatalf("the trigger ended with %v", end)
	}
	if len(accounts.reads) != 17 || len(host.sent) != 1 || len(host.deleted) != 0 {
		t.Fatalf("after the trigger ran out: %d reads, %d messages, deleted %v", len(accounts.reads), len(host.sent), host.deleted)
	}
	clock.set(start.Add(3 * time.Minute))
	if end, _ := host.trigger(ref); end["type"] != "result" {
		t.Fatalf("the next trigger ended with %v", end)
	}
	if len(accounts.reads) != 21 || accounts.reads[17].id != stuck || countReads(accounts, stuck) != 2 {
		t.Fatalf("read %d times, %v", len(accounts.reads), accounts.reads)
	}
	if len(host.deleted) != 1 || host.deleted[0] != ref || len(host.sent) != 2 || !strings.Contains(sentText(host.sent[1].Message), "面板列表") {
		t.Fatalf("deleted %v, answered %v", host.deleted, host.sent)
	}
	if saved, _ := a.Profiles.Read("10000001"); len(saved.Panels) != 20 {
		t.Fatalf("kept %d panels", len(saved.Panels))
	}
}

// A read a trigger fails ends the refresh as upstream does: the failure is
// answered in the chat, the job is removed and nothing is kept, and the role
// can be refreshed again.
func TestPanelRefreshFailedInATriggerAnswersAndEnds(t *testing.T) {
	a, clock, accounts, host, ref := refreshOnHost(t, 20)
	accounts.fail = func(id string, scheduled bool) string {
		if id == accounts.ids[15] {
			return "plugin.game_upstream_rejected"
		}
		return ""
	}
	clock.set(time.Unix(1_800_000_000, 0).Add(time.Minute))
	if end, _ := host.trigger(ref); end["type"] != "result" {
		t.Fatalf("the trigger ended with %v", end)
	}
	if len(host.deleted) != 1 || host.deleted[0] != ref || len(host.sent) != 2 || !strings.HasPrefix(sentText(host.sent[1].Message), "面板列表更新失败") {
		t.Fatalf("deleted %v, answered %v", host.deleted, host.sent)
	}
	if saved, _ := a.Profiles.Read("10000001"); len(saved.Panels) != 0 {
		t.Fatal("a failed refresh kept panels")
	}
	accounts.fail = nil
	if host.message("%更新面板", "更新面板"); len(host.sent) != 3 || sentText(host.sent[2].Message) != "正在更新面板列表，请稍候..." {
		t.Fatalf("the role could not be refreshed again: %v", host.sent)
	}
}
