package app

import (
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
)

// taskAccounts answers the account plugin's service for one account of user
// "u" with one mainland role and the cloud game configured: the role and
// account checks, delegation.create and delegation.revoke, and the requests
// of the jobs, which it records by operation, with whether their event was
// in the background, and refuses. Only a trigger may make a job's request,
// and with a delegation.
type taskAccounts struct {
	t        *testing.T
	executed []string
	detached []bool
	revoked  []string
}

func (s *taskAccounts) answer(call hostCall) (map[string]any, string) {
	role := map[string]any{"ref": "role", "game": "zzz", "uid": "10000001", "region": "prod_gf_cn", "nickname": "绳匠", "level": 50}
	switch call.Method {
	case "roles":
		return map[string]any{"account_ref": "account", "roles": []any{role}}, ""
	case "list":
		owner := map[string]any{"source_protocol": "onebot11", "source_adapter": "a", "bot_id": "bot", "actor_id": "u"}
		return map[string]any{"items": []any{map[string]any{"ref": "account", "owner": owner, "account_label": "账号", "roles": []any{role}, "status": "valid", "cloud_configured": []any{"zzz"}}}, "default_roles": map[string]any{}, "uid_bindings": map[string]any{}}, ""
	case "delegation.create":
		ref := "d-" + asText(call.Params["operation"])
		return map[string]any{"delegation": map[string]any{"ref": ref, "expires_at_ms": time.Now().Add(30 * 24 * time.Hour).UnixMilli()}}, ""
	case "delegation.revoke":
		s.revoked = append(s.revoked, asText(call.Params["delegation_ref"]))
		return map[string]any{}, ""
	case "execute":
		operation := asText(call.Params["operation"])
		if !call.Scheduled || call.Params["delegation_ref"] != "d-"+operation {
			s.t.Errorf("%s was requested outside its job", operation)
		}
		s.executed = append(s.executed, operation)
		s.detached = append(s.detached, call.Detached)
		return nil, "plugin.upstream_unavailable"
	}
	s.t.Errorf("unexpected %s", call.Method)
	return nil, "plugin.method_not_found"
}

// The 开启挑战提醒 pair reads 式舆防卫战 and 危局强袭战 one after the other,
// each read allowed a call's whole time, so its trigger moves to the
// background before reading.
func TestChallengePairTriggerReadsInTheBackground(t *testing.T) {
	a := pluginApp(t)
	accounts := &taskAccounts{t: t}
	host := newSDKHost(t, a, accounts.answer)
	if end, _ := host.message("%开启挑战提醒", "开启挑战提醒"); !strings.HasPrefix(terminalText(end), "提醒功能已开启") {
		t.Fatalf("开启挑战提醒 ended with %v", end)
	}
	ref := host.job("game.challenge.")
	_ = a.Reminders.edit(ref, func(items *[]Reminder, i int) error {
		(*items)[i].NextCheckMS = 0
		return nil
	})
	if end, _ := host.trigger(ref); end["type"] != "result" && end["type"] != "error" {
		t.Fatalf("the trigger ended with %v", end)
	}
	if !slices.Equal(host.detached, []string{"scheduler-2"}) || !slices.Equal(accounts.executed, []string{"zzz.challenge", "zzz.deadly"}) || !slices.Equal(accounts.detached, []bool{true, true}) {
		t.Fatalf("detached %v, read %v %v", host.detached, accounts.executed, accounts.detached)
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
// and cloud game tasks and a 每日同步, and in chat 开启挑战提醒 and a group
// push.
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
		{"gacha.task.create", "gacha.task.remove", role(map[string]any{"hour": 0})},
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
// account for nothing. So does one of a job left in the host's database by
// an earlier version, as a 更新面板's.
func TestATriggerWhoseTaskIsGoneDeletesItsJob(t *testing.T) {
	a := pluginApp(t)
	accounts := &taskAccounts{t: t}
	host := newSDKHost(t, a, accounts.answer)
	jobs := createJobs(t, host)
	if err := a.Reminders.edit("", func(items *[]Reminder, _ int) error { *items = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := a.Subscriptions.edit("", func(items *[]ContentSubscription, _ int) error { *items = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := a.DailySyncs.edit("", func(items *[]DailySync, _ int) error { *items = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	for _, job := range jobs {
		end, actions := host.trigger(job.ref)
		if end["type"] != "result" || !deletedIn(actions, job.ref) {
			t.Fatalf("the trigger of %s ended with %v, actions %v", job.ref, end, actions)
		}
	}
	for _, ref := range []string{"game.panel.zzz.OLD", "game.link.OLD", "game.artwork.all", "game.sync.zzz.OLD", "game.reminder.zzz.OLD"} {
		host.jobs[ref] = map[string]any{"kind": "old", "task_id": ref}
		if end, actions := host.trigger(ref); end["type"] != "result" || !deletedIn(actions, ref) {
			t.Fatalf("the trigger of %s ended with %v, actions %v", ref, end, actions)
		}
	}
	if len(host.jobs) != 0 || len(accounts.executed) != 0 || len(host.sent) != 0 {
		t.Fatalf("jobs %v, requests %v, sent %v", host.jobs, accounts.executed, host.sent)
	}
}

// runJob triggers a job once its task is due and checks that the task ran:
// it asked the account for operation or, for a push, read the news.
func runJob(t *testing.T, a *App, host *sdkHost, accounts *taskAccounts, news *int, ref, operation string) {
	t.Helper()
	_ = a.Reminders.edit(ref, func(items *[]Reminder, i int) error {
		if i >= 0 {
			(*items)[i].NextCheckMS = 0
		}
		return nil
	})
	executed, read := len(accounts.executed), *news
	if end, _ := host.trigger(ref); end["type"] != "result" && end["type"] != "error" {
		t.Fatalf("the trigger of %s ended with %v", ref, end)
	}
	if operation == "" {
		if *news != read+1 {
			t.Fatalf("the trigger of %s read the news %d times", ref, *news-read)
		}
		return
	}
	if len(accounts.executed) == executed || accounts.executed[executed] != operation {
		t.Fatalf("the trigger of %s requested %v", ref, accounts.executed[executed:])
	}
}

// Each trigger runs the task the host names in it: the reminders and timed
// tasks ask the account for their operation and a group push reads the news.
func TestTriggersRunTheirTasks(t *testing.T) {
	a := pluginApp(t)
	news := 0
	a.Content = PublicContentClient{HTTP: httpDoer(func(r *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(r.URL.Path, "/getNewsList") {
			t.Errorf("content request %s", r.URL)
		}
		news++
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"retcode":0,"data":{"list":[]}}`))}, nil
	})}
	accounts := &taskAccounts{t: t}
	host := newSDKHost(t, a, accounts.answer)
	jobs := createJobs(t, host)
	for _, job := range jobs {
		if _, carried := host.jobs[job.ref]["task_id"]; carried {
			t.Fatalf("%s has payload %v", job.ref, host.jobs[job.ref])
		}
	}
	operations := []string{"zzz.note", "zzz.deadly", "zzz.sign", "zzz.monthly", "zzz.community_run", "zzz.cloud_sign", "zzz.gacha", "zzz.challenge", ""}
	for i, job := range jobs {
		runJob(t, a, host, accounts, &news, job.ref, operations[i])
	}
	// Only the 每日同步, which reads every page, and the 开启挑战提醒 pair,
	// which reads twice, run in the background.
	if len(host.detached) != 2 || accounts.executed[len(accounts.executed)-1] != "zzz.deadly" || !accounts.detached[len(accounts.detached)-1] {
		t.Fatalf("detached %v, read %v %v", host.detached, accounts.executed, accounts.detached)
	}
}

// A group push reads, draws and sends a new post in the foreground of its
// trigger's event, and the next trigger does not send it again. 推送公告
// checks every group in the background.
func TestGroupPushStaysInTheForeground(t *testing.T) {
	a := pluginApp(t)
	post := 1
	a.Content = PublicContentClient{HTTP: httpDoer(func(r *http.Request) (*http.Response, error) {
		id := fmt.Sprint(900 + post)
		body := `{"retcode":0,"data":{"list":[]}}`
		switch {
		case strings.HasSuffix(r.URL.Path, "/getNewsList") && r.URL.Query().Get("type") == "1":
			body = fmt.Sprintf(`{"retcode":0,"data":{"list":[{"post":{"post_id":"%s","subject":"公告%d","created_at":"%d"}}]}}`, id, post, time.Now().Unix())
		case strings.HasSuffix(r.URL.Path, "/getPostFull"):
			body = fmt.Sprintf(`{"retcode":0,"data":{"post":{"post":{"post_id":"%s","subject":"公告%d","content":""},"user":{},"stat":{}}}}`, r.URL.Query().Get("post_id"), post)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	host := newSDKHost(t, a, (&taskAccounts{t: t}).answer)
	host.groupMessage("%开启公告推送", "开启公告推送")
	ref := host.job("game.content.")
	end, _ := host.trigger(ref)
	if end["type"] != "result" || len(host.detached) != 0 || len(host.sent) != 1 || host.sent[0].TargetType != "group" || host.sent[0].TargetID != "g" || !strings.HasPrefix(sentText(host.sent[0].Message), "绝区零公告推送：公告1") {
		t.Fatalf("the trigger ended with %v, detached %v, sent %v", end, host.detached, host.sent)
	}
	if host.trigger(ref); len(host.detached) != 0 || len(host.sent) != 1 {
		t.Fatalf("a trigger without a new post detached %v, sent %v", host.detached, host.sent)
	}
	// A new post, once the news lists groups share for a minute are stale.
	post = 2
	a.pushLists.mu.Lock()
	a.pushLists.at = time.Time{}
	a.pushLists.mu.Unlock()
	end, _ = host.message("%推送公告", "推送公告")
	if end["type"] != "result" || !slices.Equal(host.detached, []string{"chat-4"}) || len(host.sent) != 2 || !strings.HasPrefix(sentText(host.sent[1].Message), "绝区零公告推送：公告2") {
		t.Fatalf("推送公告 ended with %v, detached %v, sent %v", end, host.detached, host.sent)
	}
}
