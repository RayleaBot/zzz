package app

import (
	"context"
	"strings"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// taskAccounts answers the account plugin's service for one account of user
// "u" with one mainland role and the cloud game configured: the role and
// account checks, delegation.create and delegation.revoke, and the requests
// of the jobs, which it records by operation and refuses. Only a trigger may
// make a job's request, and with a delegation. took, when set, is how long a
// request takes on the fake clock, answered as the host answers a call its
// event outlived.
type taskAccounts struct {
	t        *testing.T
	clock    *fakeClock
	took     time.Duration
	executed []string
	revoked  []string
}

func (s *taskAccounts) answer(request rayleabot.ServiceCallRequest, scheduled bool) (map[string]any, string) {
	role := map[string]any{"ref": "role", "game": "zzz", "uid": "10000001", "region": "prod_gf_cn", "nickname": "绳匠", "level": 50}
	switch request.Method {
	case "roles":
		return map[string]any{"account_ref": "account", "roles": []any{role}}, ""
	case "list":
		owner := map[string]any{"source_protocol": "onebot11", "source_adapter": "a", "bot_id": "bot", "actor_id": "u"}
		return map[string]any{"items": []any{map[string]any{"ref": "account", "owner": owner, "account_label": "账号", "roles": []any{role}, "status": "valid", "cloud_configured": []any{"zzz"}}}, "default_roles": map[string]any{}, "uid_bindings": map[string]any{}}, ""
	case "delegation.create":
		ref := "d-" + asText(request.Params["operation"])
		return map[string]any{"delegation": map[string]any{"ref": ref, "expires_at_ms": time.Now().Add(30 * 24 * time.Hour).UnixMilli()}}, ""
	case "delegation.revoke":
		s.revoked = append(s.revoked, asText(request.Params["delegation_ref"]))
		return map[string]any{}, ""
	case "execute":
		operation := asText(request.Params["operation"])
		if !scheduled || request.Params["delegation_ref"] != "d-"+operation {
			s.t.Errorf("%s was requested outside its job", operation)
		}
		s.executed = append(s.executed, operation)
		if s.took > 0 {
			_ = s.clock.Sleep(context.Background(), s.took)
			return nil, "plugin.event_timeout"
		}
		return nil, "plugin.upstream_unavailable"
	}
	s.t.Errorf("unexpected %s", request.Method)
	return nil, "plugin.method_not_found"
}

// A reminder's trigger ends its work when its event's time runs out: the
// 开启挑战提醒 pair whose 式舆防卫战 read outlives the event does not go on
// to read 危局强袭战.
func TestReminderTriggerStopsWhenItsEventRunsOut(t *testing.T) {
	a := pluginApp(t)
	clock := &fakeClock{at: time.Now()}
	a.clock = clock
	accounts := &taskAccounts{t: t, clock: clock}
	host := newSDKHost(t, a, accounts.answer)
	if end, _ := host.message("%开启挑战提醒", "开启挑战提醒"); !strings.HasPrefix(terminalText(end), "提醒功能已开启") {
		t.Fatalf("开启挑战提醒 ended with %v", end)
	}
	ref := host.job("game.challenge.")
	_ = a.Reminders.edit(ref, func(items *[]Reminder, i int) error {
		(*items)[i].NextCheckMS = 0
		return nil
	})
	accounts.took = time.Minute
	if end, _ := host.trigger(ref); end["type"] != "result" && end["type"] != "error" {
		t.Fatalf("the trigger ended with %v", end)
	}
	if len(accounts.executed) != 1 || accounts.executed[0] != "zzz.challenge" || len(host.sent) != 0 {
		t.Fatalf("the trigger went on after its time: %v, sent %v", accounts.executed, host.sent)
	}
}
