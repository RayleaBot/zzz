package app

import (
	"context"
	"slices"
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

// createdJob is a job created as a user or the management page creates it:
// its task ID and how its task is removed, a management action or, when
// action is empty, a chat command.
type createdJob struct {
	ref, action, command string
}

// createJobs creates a job of each kind: through the management page the
// stamina and challenge reminders, sign-in, monthly collection, community
// and cloud game tasks and a background sync, and in chat 开启挑战提醒 and a
// group push.
func createJobs(t *testing.T, host *sdkHost) []createdJob {
	t.Helper()
	role := func(input map[string]any) map[string]any {
		input["account_ref"], input["role_ref"], input["days"], input["confirm"] = "account", "role", 30, true
		return input
	}
	jobs := []createdJob{}
	created := func(action, command string, end map[string]any) {
		t.Helper()
		for ref := range host.jobs {
			if !slices.ContainsFunc(jobs, func(job createdJob) bool { return job.ref == ref }) {
				if host.jobs[ref]["task_id"] != ref {
					t.Fatalf("%s has payload %v", ref, host.jobs[ref])
				}
				jobs = append(jobs, createdJob{ref: ref, action: action, command: command})
				return
			}
		}
		t.Fatalf("no job was created: %v", end)
	}
	for _, job := range []struct {
		action, remove string
		input          map[string]any
	}{
		{"reminder.create", "reminder.remove", role(map[string]any{"threshold": 80})},
		{"challenge.reminder.create", "challenge.reminder.remove", role(map[string]any{"kind": "deadly", "metric": "star", "threshold": 6, "hour": 20})},
		{"signin.task.create", "signin.task.remove", role(map[string]any{"hour": 0})},
		{"monthly.task.create", "monthly.task.remove", role(map[string]any{"hour": 0})},
		{"community.task.create", "community.task.remove", role(map[string]any{"once": true, "read": true})},
		{"cloudgame.task.create", "cloudgame.task.remove", role(map[string]any{"once": true})},
		{"gacha.task.create", "gacha.task.remove", role(map[string]any{"kind": "once", "hour": 8})},
	} {
		end, _ := host.manage(job.action, job.input)
		created(job.remove, "", end)
	}
	end, _ := host.message("%开启挑战提醒", "开启挑战提醒")
	created("", "关闭挑战提醒", end)
	end, _ = host.groupMessage("%开启公告推送", "开启公告推送")
	created("content.subscription.remove", "", end)
	return jobs
}

// deletedIn is whether an event's actions deleted the job ref.
func deletedIn(actions []hostAction, ref string) bool {
	return slices.ContainsFunc(actions, func(action hostAction) bool {
		return action.Name == "scheduler.delete" && action.Data["task_id"] == ref
	})
}

// Removing a task, from the management page or in chat, deletes its job in
// the same event.
func TestRemovingATaskDeletesItsJob(t *testing.T) {
	a := pluginApp(t)
	host := newSDKHost(t, a, (&taskAccounts{t: t}).answer)
	jobs := createJobs(t, host)
	if len(jobs) != 9 {
		t.Fatalf("created %d jobs", len(jobs))
	}
	for _, job := range jobs {
		var end map[string]any
		var actions []hostAction
		if job.action != "" {
			end, actions = host.manage(job.action, map[string]any{"ref": job.ref, "confirm": true})
		} else {
			end, actions = host.message("%"+job.command, job.command)
		}
		if end["type"] == "error" || !deletedIn(actions, job.ref) {
			t.Fatalf("removing %s ended with %v, actions %v", job.ref, end, actions)
		}
		if _, kept := host.jobs[job.ref]; kept {
			t.Fatalf("%s kept its job", job.ref)
		}
	}
}

// A trigger whose task is no longer stored deletes its own job and asks the
// account for nothing; that of a job whose payload has no task ID deletes
// nothing, since a delete without an ID would stop the plugin.
func TestATriggerWhoseTaskIsGoneDeletesItsJob(t *testing.T) {
	a := pluginApp(t)
	accounts := &taskAccounts{t: t}
	host := newSDKHost(t, a, accounts.answer)
	jobs := createJobs(t, host)
	if err := a.Reminders.edit("", func(items *[]Reminder, _ int) error { *items = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := a.SyncTasks.edit("", func(items *[]SyncTask, _ int) error { *items = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := a.Subscriptions.edit("", func(items *[]ContentSubscription, _ int) error { *items = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	for _, job := range jobs {
		end, actions := host.trigger(job.ref)
		if end["type"] != "result" || !deletedIn(actions, job.ref) {
			t.Fatalf("the trigger of %s ended with %v, actions %v", job.ref, end, actions)
		}
	}
	host.jobs["game.reminder.OLD"] = map[string]any{"kind": "stamina_reminder"}
	if end, actions := host.trigger("game.reminder.OLD"); end["type"] != "result" || len(actions) != 0 {
		t.Fatalf("the trigger without a task ID ended with %v, actions %v", end, actions)
	}
	delete(host.jobs, "game.reminder.OLD")
	if len(host.jobs) != 0 || len(accounts.executed) != 0 || len(host.sent) != 0 {
		t.Fatalf("jobs %v, requests %v, sent %v", host.jobs, accounts.executed, host.sent)
	}
}
