package app

import (
	"slices"
	"testing"
	"time"
)

// dailyTasks are the 每日同步 and background syncs gacha.task.list returns.
func dailyTasks(t *testing.T, host *sdkHost) ([]DailySync, []BackgroundSync) {
	t.Helper()
	end, _ := host.manage("gacha.task.list", map[string]any{})
	var listed struct {
		Items []BackgroundSync `json:"items"`
		Daily []DailySync      `json:"daily"`
	}
	if end["type"] != "result" || decodeObject(end["data"], &listed) != nil || len(listed.Daily) != 1 {
		t.Fatalf("gacha.task.list ended with %v", end)
	}
	return listed.Daily, listed.Items
}

// createDaily creates the role's 每日同步 at 08:00 Beijing time, with
// notify, and returns its job.
func createDaily(t *testing.T, host *sdkHost) string {
	t.Helper()
	if end, _ := host.manage("gacha.task.create", map[string]any{"account_ref": "account", "role_ref": "role", "hour": 8, "days": 7, "notify": true, "confirm": true}); end["type"] != "result" {
		t.Fatalf("gacha.task.create ended with %v", end)
	}
	return host.job(dailySyncTask)
}

// beijing is a time on the Beijing clock.
func beijing(day, hour, minute int) time.Time {
	return time.Date(2026, 9, day, hour, minute, 0, 0, time.FixedZone("UTC+8", 8*3600))
}

// A 每日同步 is a scheduler job with a zzz.gacha delegation. After its hour,
// a trigger moves to the background and reads every page under the
// delegation, keeps the records, tells the account's owner and waits for the
// next day; later triggers of the day read nothing and stay in the
// foreground, and the next day's round reads what is new.
func TestDailySyncReadsEveryPageInADetachedTrigger(t *testing.T) {
	a, clock, accounts, host := syncHost(t)
	clock.set(beijing(20, 17, 0))
	ref := createDaily(t, host)
	grant := accounts.grants[0]
	if len(accounts.grants) != 1 || grant["operation"] != "zzz.gacha" || grant["task_id"] != ref || grant["days"] != float64(7) || host.created[0]["cron"] != "*/5 * * * *" {
		t.Fatalf("granted %v, created %v", accounts.grants, host.created)
	}
	if end, _ := host.trigger(ref); end["type"] != "result" || !slices.Equal(host.detached, []string{"scheduler-2"}) {
		t.Fatalf("the trigger ended with %v, detached %v", end, host.detached)
	}
	checkSyncReads(t, a, accounts, "scheduler-2")
	if len(host.sent) != 1 || host.sent[0].TargetType != "private" || host.sent[0].TargetID != "u" || sentText(host.sent[0].Message) != "抽卡后台同步完成\n绳匠 · 10000001\n新增 45 条，档案共 45 条。" {
		t.Fatalf("notified %v", host.sent)
	}
	daily, items := dailyTasks(t, host)
	if task := daily[0]; task.State != "waiting" || task.LastCode != "sync_completed" || task.RunDay != "2026-09-20" || task.NextCheckMS != beijing(21, 8, 0).UnixMilli() || task.LastNotificationMS == 0 || task.Progress.Result == nil || task.Progress.Result.Added != 45 {
		t.Fatalf("the task is %+v", task)
	}
	if len(items) != 1 || items[0].Task != ref || items[0].State != "completed" {
		t.Fatalf("listed %+v", items)
	}
	clock.set(beijing(20, 23, 0))
	if host.trigger(ref); len(accounts.reads) != 8 || len(host.detached) != 1 {
		t.Fatalf("a later trigger of the day read %d pages, detached %v", len(accounts.reads), host.detached)
	}
	clock.set(beijing(21, 8, 5))
	if host.trigger(ref); len(accounts.reads) != 14 || len(host.detached) != 2 || len(host.sent) != 2 || sentText(host.sent[1].Message) != "抽卡后台同步完成\n绳匠 · 10000001\n新增 0 条，档案共 45 条。" {
		t.Fatalf("the next day read %d pages, sent %v", len(accounts.reads), host.sent)
	}
}

// A failed round is tried again five minutes later and, after the third
// failure of the day, waits for the next day; a revoked delegation pauses
// the task, and the page runs it again at once.
func TestDailySyncRetriesPausesAndRunsAgain(t *testing.T) {
	a, clock, accounts, host := syncHost(t)
	clock.set(beijing(20, 17, 0))
	ref := createDaily(t, host)
	accounts.fail = "plugin.upstream_unavailable"
	for round := 1; round <= 3; round++ {
		host.trigger(ref)
		daily, items := dailyTasks(t, host)
		if len(accounts.reads) != round || daily[0].Failures != round || items[0].State != "failed" || items[0].LastCode != "plugin.upstream_unavailable" {
			t.Fatalf("round %d: %d reads, task %+v, listed %+v", round, len(accounts.reads), daily[0], items[0])
		}
		// The retry is due only after five minutes.
		if host.trigger(ref); len(accounts.reads) != round {
			t.Fatalf("round %d was tried again at once", round)
		}
		clock.set(clock.Now().Add(dailyRetry))
	}
	if daily, _ := dailyTasks(t, host); daily[0].State != "waiting" || daily[0].NextCheckMS != beijing(21, 8, 0).UnixMilli() {
		t.Fatalf("after three failures the task is %+v", daily[0])
	}
	accounts.fail = "plugin.account_delegation_denied"
	if end, _ := host.manage("gacha.task.run", map[string]any{"ref": ref, "confirm": true}); end["type"] != "result" {
		t.Fatalf("gacha.task.run ended with %v", end)
	}
	host.trigger(ref)
	if daily, _ := dailyTasks(t, host); daily[0].State != "paused" || daily[0].LastCode != "plugin.account_delegation_denied" {
		t.Fatalf("a revoked delegation left the task %+v", daily[0])
	}
	accounts.fail = ""
	host.manage("gacha.task.run", map[string]any{"ref": ref, "confirm": true})
	host.trigger(ref)
	if daily, _ := dailyTasks(t, host); daily[0].State != "waiting" || daily[0].LastCode != "sync_completed" {
		t.Fatalf("the task run again is %+v", daily[0])
	}
	if archive, err := a.Gacha.Read("10000001", "prod_gf_cn"); err != nil || len(archive.Records) != 45 {
		t.Fatalf("kept %d records, %v", len(archive.Records), err)
	}
}

// Removing the archive pauses its 每日同步, so no round brings it back;
// removing the task deletes its job and revokes its delegation.
func TestDailySyncArchiveRemovalAndRemoval(t *testing.T) {
	_, clock, accounts, host := syncHost(t)
	clock.set(beijing(20, 17, 0))
	ref := createDaily(t, host)
	if end, _ := host.manage("gacha.remove", map[string]any{"uid": "10000001", "region": "prod_gf_cn"}); end["type"] != "result" {
		t.Fatalf("gacha.remove ended with %v", end)
	}
	if host.trigger(ref); len(accounts.reads) != 0 {
		t.Fatal("a round read the removed archive's role")
	}
	if daily, _ := dailyTasks(t, host); daily[0].State != "paused" || daily[0].LastCode != "archive_removed" {
		t.Fatalf("the task is %+v", daily[0])
	}
	end, actions := host.manage("gacha.task.remove", map[string]any{"ref": ref, "confirm": true})
	if end["type"] != "result" || !deletedIn(actions, ref) || !slices.Equal(accounts.revoked, []string{"d1"}) {
		t.Fatalf("gacha.task.remove ended with %v, actions %v, revoked %v", end, actions, accounts.revoked)
	}
}
