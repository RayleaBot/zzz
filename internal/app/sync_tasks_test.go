package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/gacha"
)

func syncTaskFixture(t *testing.T, kind string) (*SyncTaskStore, *gacha.Syncs, *gacha.Store, SyncTask, int64) {
	t.Helper()
	dir := t.TempDir()
	now := time.Date(2026, 9, 20, 8, 0, 0, 0, time.FixedZone("CN", 28800)).UnixMilli()
	s := syncTaskStore(dir)
	jobs := &gacha.Syncs{}
	archive := &gacha.Store{Directory: filepath.Join(dir, "gacha"), Game: "zzz"}
	task := SyncTask{Ref: "game.sync.fixture", Selection: Selection{"account", "role"}, Provider: "provider", Role: Role{Ref: "role", Game: "zzz", UID: "100000001", Region: "prod_gf_cn"}, Kind: kind, Hour: 8, State: "waiting", ExpiresAtMS: now + 30*86400000, DelegationRef: "grant", Owner: Subject{ActorID: "owner"}}
	if err := seedSyncTasks(s, task); err != nil {
		t.Fatal(err)
	}
	return s, jobs, archive, task, now
}
func syncPage(_ context.Context, _ SyncTask, pool, end string, page int) (gacha.RemotePage, error) {
	p := gacha.RemotePage{Records: []gacha.Record{}, Timezone: 8, Language: "zh-cn", NextID: end}
	if pool == "1001" {
		p.Records = []gacha.Record{{ID: "100", ItemID: "1001", GachaType: "1", UIGFType: "1", Time: "2026-09-01 08:00:00", Rank: "5"}}
		p.NextID = "100"
	}
	return p, nil
}

// onePage is a trigger that reads a single page.
var onePage = SyncTaskPace{Now: time.Now, Wait: func(ctx context.Context, _ time.Duration) error { return ctx.Err() }}

func tickOne(t *testing.T, s *SyncTaskStore, j *gacha.Syncs, a *gacha.Store, task SyncTask, now int64, fetch SyncTaskFetch, send SyncTaskSend) SyncTask {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := s.Tick(ctx, task.Ref, now, j, a, fetch, send, onePage); err != nil {
		t.Fatal(err)
	}
	items, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	return items[0]
}
func TestBackgroundSyncContinuesWithoutUIAndRestartsSafely(t *testing.T) {
	s, j, a, task, now := syncTaskFixture(t, "once")
	task = tickOne(t, s, j, a, task, now, syncPage, nil)
	if task.Progress.Pages != 1 || task.State != "running" {
		t.Fatal(task)
	}
	if _, err := a.Read(task.Role.UID, task.Role.Region); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("partial page written")
	}
	// Fresh process has only persisted task metadata. It reads again from the
	// current archive; no in-memory version or authorization is reconstructed.
	s = &SyncTaskStore{taskFiles: taskFiles[SyncTask]{Directory: s.Directory}}
	j = &gacha.Syncs{}
	a = &gacha.Store{Directory: a.Directory, Game: a.Game}
	for i := 1; i <= 6; i++ {
		task = tickOne(t, s, j, a, task, now+int64(i)*60000, syncPage, nil)
	}
	if task.State != "completed" || task.Progress.Result.Added != 1 || task.Restarts != 1 {
		t.Fatal(task)
	}
	actual, err := a.Read(task.Role.UID, task.Role.Region)
	if err != nil || len(actual.Records) != 1 {
		t.Fatal(actual, err)
	}
	tickOne(t, s, j, a, task, now+86400000, func(context.Context, SyncTask, string, string, int) (gacha.RemotePage, error) {
		t.Fatal("completed single task repeated")
		return gacha.RemotePage{}, nil
	}, nil)
}
func TestBackgroundSyncDeletionAndConcurrentAdmission(t *testing.T) {
	s, j, a, task, now := syncTaskFixture(t, "once")
	var wg sync.WaitGroup
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			if err := s.Tick(ctx, task.Ref, now, j, a, syncPage, nil, onePage); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	items, _ := s.List()
	if items[0].Progress.Pages != 1 {
		t.Fatal("duplicate tick advanced pages", items)
	}
	if err := s.RemoveArchive(j, a, task.Role.UID, task.Role.Region); err != nil {
		t.Fatal(err)
	}
	s = &SyncTaskStore{taskFiles: taskFiles[SyncTask]{Directory: s.Directory}}
	j = &gacha.Syncs{}
	task = tickOne(t, s, j, a, task, now+60000, func(context.Context, SyncTask, string, string, int) (gacha.RemotePage, error) {
		t.Fatal("deleted archive fetched again")
		return gacha.RemotePage{}, nil
	}, nil)
	if task.State != "paused" || task.LastCode != "archive_removed" {
		t.Fatal(task)
	}
	if _, err := a.Read(task.Role.UID, task.Role.Region); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("deleted archive revived")
	}
}
func TestBackgroundSyncDailyHourCompletionAndNotifyOnce(t *testing.T) {
	s, j, a, task, now := syncTaskFixture(t, "daily")
	task.Notify = true
	_ = s.edit(task.Ref, func(items *[]SyncTask, i int) error { (*items)[i] = task; return nil })
	sent := 0
	send := func(context.Context, SyncTask, string) error { sent++; return nil }
	task = tickOne(t, s, j, a, task, now-60000, func(context.Context, SyncTask, string, string, int) (gacha.RemotePage, error) {
		t.Fatal("ran before hour")
		return gacha.RemotePage{}, nil
	}, send)
	for i := 0; i < 6; i++ {
		task = tickOne(t, s, j, a, task, now+int64(i)*60000, syncPage, send)
	}
	if task.State != "waiting" || sent != 1 || task.LastNotificationMS == 0 {
		t.Fatal(task, sent)
	}
	s = &SyncTaskStore{taskFiles: taskFiles[SyncTask]{Directory: s.Directory}}
	task = tickOne(t, s, j, a, task, now+12*3600000, func(context.Context, SyncTask, string, string, int) (gacha.RemotePage, error) {
		t.Fatal("same day restarted")
		return gacha.RemotePage{}, nil
	}, send)
	task = tickOne(t, s, j, a, task, now+86400000, syncPage, send)
	if task.State != "running" || task.Progress.Pages != 1 || sent != 1 {
		t.Fatal(task, sent)
	}
}
func TestBackgroundSyncFailuresExpiryAndMergeConflict(t *testing.T) {
	for _, mode := range []string{"auth", "transient", "daily", "conflict", "expiry"} {
		t.Run(mode, func(t *testing.T) {
			kind := "once"
			if mode == "daily" {
				kind = "daily"
			}
			s, j, a, task, now := syncTaskFixture(t, kind)
			if mode == "expiry" {
				task.ExpiresAtMS = now
				_ = s.edit(task.Ref, func(items *[]SyncTask, i int) error { (*items)[i] = task; return nil })
				task = tickOne(t, s, j, a, task, now, syncPage, nil)
				if task.State != "expired" {
					t.Fatal(task)
				}
				return
			}
			if mode == "conflict" {
				task = tickOne(t, s, j, a, task, now, syncPage, nil)
				_, _, err := a.Import(gacha.Archive{UID: task.Role.UID, Region: task.Role.Region, Timezone: 8, Language: "zh-cn", Records: []gacha.Record{{ID: "99", ItemID: "1002", GachaType: "1", UIGFType: "1", Time: "2026-09-01 07:00:00", Rank: "3"}}})
				if err != nil {
					t.Fatal(err)
				}
				for i := 1; i <= 5; i++ {
					task = tickOne(t, s, j, a, task, now+int64(i)*60000, syncPage, nil)
				}
				if task.State != "paused" || task.LastCode != "plugin.game_sync_conflict" {
					t.Fatal(task)
				}
				saved, _ := a.Read(task.Role.UID, task.Role.Region)
				if len(saved.Records) != 1 || saved.Records[0].ID != "99" {
					t.Fatal("conflict modified archive")
				}
				return
			}
			code := "plugin.upstream_unavailable"
			if mode == "auth" {
				code = "plugin.account_delegation_denied"
			}
			fail := func(context.Context, SyncTask, string, string, int) (gacha.RemotePage, error) {
				return gacha.RemotePage{}, &rayleabot.ActionError{Code: code}
			}
			for i := 0; i < 3; i++ {
				task = tickOne(t, s, j, a, task, now+int64(i)*5*60000, fail, nil)
			}
			want := "paused"
			if mode == "daily" {
				want = "waiting"
			}
			if task.State != want {
				t.Fatal(task)
			}
		})
	}
}

func seedSyncTasks(s *SyncTaskStore, tasks ...SyncTask) error {
	return s.edit("", func(items *[]SyncTask, _ int) error {
		*items = append(*items, tasks...)
		return nil
	})
}

func TestChatSyncAnswersInItsChatOnce(t *testing.T) {
	s, j, a, task, now := syncTaskFixture(t, "once")
	task.ReplyType, task.ReplyID, task.Before = "group", "group-1", map[string]int{}
	_ = s.edit(task.Ref, func(items *[]SyncTask, i int) error { (*items)[i] = task; return nil })
	type sent struct{ target, text string }
	answers := []sent{}
	send := func(_ context.Context, task SyncTask, text string) error {
		answers = append(answers, sent{task.ReplyType + ":" + task.ReplyID, text})
		return nil
	}
	for i := 0; i < 6; i++ {
		task = tickOne(t, s, j, a, task, now+int64(i)*60000, syncPage, send)
	}
	// ZZZ-Plugin's report goes to the chat that asked, and only once.
	if len(answers) != 1 || answers[0].target != "group:group-1" || answers[0].text != gachaLinkSummary(map[string]int{}, map[string]int{"1": 1}) {
		t.Fatal(answers)
	}
	if task.ReplyType != "" || task.Before != nil {
		t.Fatal("reply target kept", task)
	}
}

// gachaRead is one zzz.gacha page a background sync asked for.
type gachaRead struct {
	at        time.Time
	pool, end string
	page      int
}

// gachaAccounts answers the account plugin's service for one account of user
// "u" with a mainland role whose exclusive channel holds records records and
// whose other channels are empty: roles, list, delegation.create, which
// grants "d1", and each zzz.gacha page, which only a trigger may ask for,
// with the delegation. A page takes took of the fake clock.
type gachaAccounts struct {
	t       *testing.T
	clock   *fakeClock
	records int
	took    time.Duration
	reads   []gachaRead
}

func (s *gachaAccounts) answer(request rayleabot.ServiceCallRequest, scheduled bool) (map[string]any, string) {
	role := map[string]any{"ref": "role", "game": "zzz", "uid": "10000001", "region": "prod_gf_cn", "nickname": "绳匠", "level": 50}
	switch request.Method {
	case "roles":
		return map[string]any{"account_ref": "account", "roles": []any{role}}, ""
	case "list":
		owner := map[string]any{"source_protocol": "onebot11", "source_adapter": "a", "bot_id": "bot", "actor_id": "u"}
		return map[string]any{"items": []any{map[string]any{"ref": "account", "owner": owner, "roles": []any{role}, "status": "valid"}}, "default_roles": map[string]any{}, "uid_bindings": map[string]any{}}, ""
	case "delegation.create":
		return map[string]any{"delegation": map[string]any{"ref": "d1", "expires_at_ms": s.clock.Now().Add(7 * 24 * time.Hour).UnixMilli()}}, ""
	case "execute":
		input := asObject(request.Params["input"])
		if !scheduled || request.Params["delegation_ref"] != "d1" || request.Params["operation"] != "zzz.gacha" {
			s.t.Errorf("%v was requested outside its job", request.Params)
		}
		page, _ := input["page"].(float64)
		read := gachaRead{at: s.clock.Now(), pool: asText(input["gacha_type"]), end: asText(input["end_id"]), page: int(page)}
		s.reads = append(s.reads, read)
		_ = s.clock.Sleep(context.Background(), s.took)
		list := []any{}
		if read.pool == "2001" {
			first := 1
			if end, _ := strconv.Atoi(read.end); end != 0 {
				first = 5000 - end + 1
			}
			for n := first; n <= s.records && len(list) < 20; n++ {
				at := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC).Add(-time.Duration(n) * time.Minute)
				list = append(list, map[string]any{"uid": "10000001", "gacha_id": "0", "gacha_type": "2", "item_id": "1191", "count": "1", "time": at.Format("2006-01-02 15:04:05"), "name": "艾莲", "item_type": "代理人", "rank_type": "4", "id": strconv.Itoa(5000 - n)})
			}
		}
		return map[string]any{"operation": "zzz.gacha", "role": role, "data": map[string]any{"list": list, "region": "prod_gf_cn", "region_time_zone": 0}}, ""
	}
	s.t.Errorf("unexpected %s", request.Method)
	return nil, "plugin.method_not_found"
}

// A background sync run through the SDK as the host runs it: each trigger
// reads page after page, a second apart, and stops when its event's time
// runs out, without counting the page it was cut off in as a failure; the
// next trigger, even one that comes a little early, reads that page again
// and continues until the sync completes.
func TestBackgroundSyncTriggersStopInTimeAndContinue(t *testing.T) {
	a := pluginApp(t)
	start := time.Date(2026, 9, 20, 9, 0, 0, 0, time.UTC)
	clock := &fakeClock{at: start}
	a.clock = clock
	accounts := &gachaAccounts{t: t, clock: clock, records: 45, took: 30 * time.Second}
	host := newSDKHost(t, a, accounts.answer)
	if end, _ := host.manage("gacha.task.create", map[string]any{"account_ref": "account", "role_ref": "role", "kind": "once", "hour": 8, "days": 7, "confirm": true}); end["type"] != "result" {
		t.Fatalf("the task was not created: %v", end)
	}
	ref := host.job("game.sync.")
	task := func() SyncTask {
		items, err := a.SyncTasks.List()
		if err != nil || len(items) != 1 {
			t.Fatal(items, err)
		}
		return items[0]
	}
	// Pages of 30 seconds: the trigger reads one, and the second is still
	// open when the event's time runs out.
	clock.set(start.Add(time.Minute))
	if end, _ := host.trigger(ref); end["type"] != "result" {
		t.Fatalf("the trigger ended with %v", end)
	}
	if got := task(); len(accounts.reads) != 2 || got.Progress.Pages != 1 || got.State != "running" || got.Failures != 0 || got.LastCode != "sync_running" {
		t.Fatalf("after %d reads the task is %+v", len(accounts.reads), got)
	}
	cut := accounts.reads[1]
	// The next trigger comes half a second early and reads the rest.
	accounts.took = 0
	clock.set(start.Add(2*time.Minute - 500*time.Millisecond))
	if end, _ := host.trigger(ref); end["type"] != "result" {
		t.Fatalf("the next trigger ended with %v", end)
	}
	if len(accounts.reads) < 3 {
		t.Fatalf("the next trigger read nothing: %+v", task())
	}
	if again := accounts.reads[2]; again.pool != cut.pool || again.end != cut.end || again.page != cut.page {
		t.Fatalf("the next trigger asked for %+v, not the cut-off %+v", again, cut)
	}
	for i := 3; i < len(accounts.reads); i++ {
		if gap := accounts.reads[i].at.Sub(accounts.reads[i-1].at); gap < time.Second {
			t.Fatalf("pages %d and %d are %v apart", i-1, i, gap)
		}
	}
	if got := task(); got.State != "completed" || len(accounts.reads) != 9 {
		t.Fatalf("after %d reads the task is %+v", len(accounts.reads), got)
	}
	if archive, err := a.Gacha.Read("10000001", "prod_gf_cn"); err != nil || len(archive.Records) != 45 {
		t.Fatalf("kept %d records, %v", len(archive.Records), err)
	}
}
