package app

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// characterRead is one zzz.character request of a refresh: when, of which
// character and in which event.
type characterRead struct {
	at     time.Time
	id     string
	parent string
}

// accountsService answers the account plugin's service for a role with the
// characters ids: list, the character list and each character's details.
// The reads must come from a chat event already in the background, without
// a delegation. during, when set, runs as the nth read arrives, and fail
// answers a read with a failure code.
type accountsService struct {
	t      *testing.T
	clock  *fakeClock
	ids    []string
	reads  []characterRead
	during func(n int)
	fail   func(id string) string
}

func (s *accountsService) answer(call hostCall) (map[string]any, string) {
	if call.TargetPluginID != "raylea.mihoyo-accounts" || call.Service != "accounts" || call.ServiceVersion != 1 {
		s.t.Errorf("call %+v", call)
		return nil, "plugin.service_not_found"
	}
	role := map[string]any{"ref": "role", "game": "zzz", "uid": "10000001", "nickname": "绳匠", "level": 50}
	params := call.Params
	switch call.Method {
	case "list":
		return map[string]any{"items": []any{map[string]any{"ref": "account", "roles": []any{role}, "status": "valid"}}, "default_roles": map[string]any{}, "uid_bindings": map[string]any{}}, ""
	case "execute":
		if call.Scheduled || !call.Detached || params["delegation_ref"] != nil {
			s.t.Errorf("%v read by %s, detached %v", params, call.Parent, call.Detached)
		}
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
			id := asText(list[0])
			s.reads = append(s.reads, characterRead{at: s.clock.Now(), id: id, parent: call.Parent})
			if s.during != nil {
				s.during(len(s.reads))
			}
			if s.fail != nil {
				if code := s.fail(id); code != "" {
					return nil, code
				}
			}
			return map[string]any{"operation": "zzz.character", "role": role, "data": map[string]any{"avatar_list": []any{map[string]any{"id": id, "level": 60}}}}, ""
		}
	}
	s.t.Errorf("unexpected %s %v", call.Method, params["operation"])
	return nil, "plugin.method_not_found"
}

// refreshHost runs the plugin through the SDK for a role with n characters,
// on a fake clock.
func refreshHost(t *testing.T, n int) (*App, *accountsService, *sdkHost) {
	t.Helper()
	a := pluginApp(t)
	clock := &fakeClock{at: time.Unix(1_800_000_000, 0)}
	a.clock = clock
	accounts := &accountsService{t: t, clock: clock, ids: characterIDs(n)}
	return a, accounts, newSDKHost(t, a, accounts.answer)
}

// actionAt is the position of the first action named name, or -1.
func actionAt(actions []hostAction, name string) int {
	return slices.IndexFunc(actions, func(action hostAction) bool { return action.Name == name })
}

// 更新面板 of a role with 40 characters, run through the SDK as the host runs
// it: the command's event moves to the background, says that it is updating
// and reads every character in a request of its own, spaced as upstream,
// then keeps the panels and answers 面板列表 in the chat. Meanwhile the chat
// is free, and another 更新面板 of the UID answers that one is running.
func TestPanelRefreshReadsEveryCharacterInOneDetachedEvent(t *testing.T) {
	a, accounts, host := refreshHost(t, 40)
	var again map[string]any
	accounts.during = func(n int) {
		if n == 20 {
			again, _ = host.message("%更新面板", "更新面板")
		}
	}
	end, actions := host.message("%更新面板", "更新面板")
	if end["type"] != "result" || !slices.Equal(host.detached, []string{"chat-1"}) {
		t.Fatalf("the command ended with %v, detached %v", end, host.detached)
	}
	if detached, noticed := actionAt(actions, "event.detach"), actionAt(actions, "message.send"); detached < 0 || noticed < detached {
		t.Fatalf("actions %v", actions)
	}
	if terminalText(again) != "面板列表正在更新中，请稍后再试" {
		t.Fatalf("the second 更新面板 ended with %v", again)
	}
	if len(accounts.reads) != 40 {
		t.Fatalf("read %d of 40 characters", len(accounts.reads))
	}
	for i, read := range accounts.reads {
		if read.id != accounts.ids[i] || read.parent != "chat-1" || i > 0 && read.at.Sub(accounts.reads[i-1].at) != 3*time.Second {
			t.Fatalf("read %d is %+v", i, read)
		}
	}
	if len(host.sent) != 2 || sentText(host.sent[0].Message) != "正在更新面板列表，请稍候..." || len(host.created) != 0 {
		t.Fatalf("sent %v, created %v", host.sent, host.created)
	}
	if sent := host.sent[1]; sent.TargetType != "private" || sent.TargetID != "u" || !strings.Contains(sentText(sent.Message), "面板列表") {
		t.Fatalf("answered %+v", sent)
	}
	if saved, err := a.Profiles.Read("10000001"); err != nil || len(saved.Panels) != 40 || saved.Nickname != "绳匠" || saved.RefreshedAtMS == 0 {
		t.Fatalf("kept %d panels, %v", len(saved.Panels), err)
	}
}

// As upstream, a failed read ends the refresh with its failure and keeps
// nothing, and the UID can be refreshed again.
func TestPanelRefreshFailureAnswersAndKeepsNothing(t *testing.T) {
	a, accounts, host := refreshHost(t, 20)
	accounts.fail = func(id string) string {
		if id == accounts.ids[15] {
			return "plugin.game_upstream_rejected"
		}
		return ""
	}
	if end, _ := host.message("%更新面板", "更新面板"); end["type"] != "result" {
		t.Fatalf("the command ended with %v", end)
	}
	if len(accounts.reads) != 16 || len(host.sent) != 2 || !strings.HasPrefix(sentText(host.sent[1].Message), "面板列表更新失败") {
		t.Fatalf("%d reads, answered %v", len(accounts.reads), host.sent)
	}
	if saved, _ := a.Profiles.Read("10000001"); len(saved.Panels) != 0 {
		t.Fatal("a failed refresh kept panels")
	}
	accounts.fail = nil
	if host.message("%更新面板", "更新面板"); len(host.sent) != 4 || sentText(host.sent[2].Message) != "正在更新面板列表，请稍候..." {
		t.Fatalf("the UID could not be refreshed again: %v", host.sent)
	}
	if saved, _ := a.Profiles.Read("10000001"); len(saved.Panels) != 20 {
		t.Fatalf("kept %d panels", len(saved.Panels))
	}
}

// When the host holds as many background events of the plugin as it
// allows, 更新面板 says so and reads nothing; later it runs.
func TestPanelRefreshRefusedDetachAnswersBusy(t *testing.T) {
	_, accounts, host := refreshHost(t, 3)
	host.refuse = true
	if end, _ := host.message("%更新面板", "更新面板"); terminalText(end) != busyReply || len(accounts.reads) != 0 || len(host.sent) != 0 {
		t.Fatalf("the refused command ended with %v after %d reads, sent %v", end, len(accounts.reads), host.sent)
	}
	host.refuse = false
	if host.message("%更新面板", "更新面板"); len(accounts.reads) != 3 || len(host.sent) != 2 {
		t.Fatalf("the later command read %d characters, sent %v", len(accounts.reads), host.sent)
	}
}
