package app

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// gachaRead is one zzz.gacha page a sync asked for, and the event it came
// from.
type gachaRead struct {
	at        time.Time
	pool, end string
	page      int
	parent    string
}

// gachaAccounts answers the account plugin's service for one account of user
// "u" with a mainland role whose exclusive channel holds records records and
// whose other channels are empty: roles, list and each zzz.gacha page, which
// only an event already in the background may ask for, without a
// delegation. A page takes took of the fake clock.
type gachaAccounts struct {
	t       *testing.T
	clock   *fakeClock
	records int
	took    time.Duration
	reads   []gachaRead
}

func (s *gachaAccounts) answer(call hostCall) (map[string]any, string) {
	role := map[string]any{"ref": "role", "game": "zzz", "uid": "10000001", "region": "prod_gf_cn", "nickname": "绳匠", "level": 50}
	switch call.Method {
	case "roles":
		return map[string]any{"account_ref": "account", "roles": []any{role}}, ""
	case "list":
		owner := map[string]any{"source_protocol": "onebot11", "source_adapter": "a", "bot_id": "bot", "actor_id": "u"}
		return map[string]any{"items": []any{map[string]any{"ref": "account", "owner": owner, "roles": []any{role}, "status": "valid"}}, "default_roles": map[string]any{}, "uid_bindings": map[string]any{}}, ""
	case "execute":
		input := asObject(call.Params["input"])
		if call.Scheduled || !call.Detached || call.Params["delegation_ref"] != nil || call.Params["operation"] != "zzz.gacha" {
			s.t.Errorf("%v was requested by %s, detached %v", call.Params, call.Parent, call.Detached)
		}
		page, _ := input["page"].(float64)
		read := gachaRead{at: s.clock.Now(), pool: asText(input["gacha_type"]), end: asText(input["end_id"]), page: int(page), parent: call.Parent}
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
	s.t.Errorf("unexpected %s", call.Method)
	return nil, "plugin.method_not_found"
}

// syncHost runs the plugin through the SDK for a role with 45 records in its
// exclusive channel, each page taking thirty seconds of a fake clock.
func syncHost(t *testing.T) (*App, *fakeClock, *gachaAccounts, *sdkHost) {
	t.Helper()
	a := pluginApp(t)
	clock := &fakeClock{at: time.Date(2026, 9, 20, 9, 0, 0, 0, time.UTC)}
	a.clock = clock
	accounts := &gachaAccounts{t: t, clock: clock, records: 45, took: 30 * time.Second}
	return a, clock, accounts, newSDKHost(t, a, accounts.answer)
}

// checkSyncReads checks that every page came from the event parent, a
// second or more apart, and that the whole archive was kept.
func checkSyncReads(t *testing.T, a *App, accounts *gachaAccounts, parent string) {
	t.Helper()
	if len(accounts.reads) != 8 {
		t.Fatalf("read %d pages: %+v", len(accounts.reads), accounts.reads)
	}
	for i, read := range accounts.reads {
		if read.parent != parent || i > 0 && read.at.Sub(accounts.reads[i-1].at) < accounts.took+syncPageGap {
			t.Fatalf("page %d is %+v", i, read)
		}
	}
	if archive, err := a.Gacha.Read("10000001", "prod_gf_cn"); err != nil || len(archive.Records) != 45 {
		t.Fatalf("kept %d records, %v", len(archive.Records), err)
	}
}

// 更新抽卡记录, run through the SDK as the host runs it: the command's event
// moves to the background, says that the records are being read, reads
// every page with the user's account, four minutes in all, and answers
// ZZZ-Plugin's report of the channels.
func TestGachaRefreshReadsEveryPageInOneDetachedEvent(t *testing.T) {
	a, clock, accounts, host := syncHost(t)
	start := clock.Now()
	end, actions := host.groupMessage("%更新抽卡记录", "更新抽卡记录")
	if end["type"] != "result" || !slices.Equal(host.detached, []string{"chat-1"}) || actionAt(actions, "event.detach") > actionAt(actions, "message.send") || len(host.created) != 0 {
		t.Fatalf("the command ended with %v, actions %v", end, actions)
	}
	checkSyncReads(t, a, accounts, "chat-1")
	if clock.Now().Sub(start) <= hostEventTimeout || len(host.sent) != 2 || sentText(host.sent[0].Message) != "抽卡记录获取中请稍等...可能需要一段时间，请耐心等待" {
		t.Fatalf("until %v sent %v", clock.Now(), host.sent)
	}
	if sent := host.sent[1]; sent.TargetType != "group" || sent.TargetID != "g" || !strings.HasPrefix(sentText(sent.Message), "抽卡记录更新成功，共6个卡池") || !strings.Contains(sentText(sent.Message), "独家频段新增45条记录，一共45条记录") {
		t.Fatalf("answered %+v", sent)
	}
}

// The management page's 后台同步: the action answers the page with the
// started sync as it moves to the background, then reads every page and
// tells the account's owner in private chat.
func TestBackgroundSyncAnswersThePageAndFinishesInTheBackground(t *testing.T) {
	a, _, accounts, host := syncHost(t)
	end, _ := host.manage("gacha.sync.background", map[string]any{"account_ref": "account", "role_ref": "role", "notify": true, "confirm": true})
	if end["type"] != "result" || !slices.Equal(host.detached, []string{"manage-1"}) || asObject(host.delivered["manage-1"]["sync"])["state"] != "running" {
		t.Fatalf("the action ended with %v, delivered %v", end, host.delivered)
	}
	checkSyncReads(t, a, accounts, "manage-1")
	if len(host.sent) != 1 || host.sent[0].TargetType != "private" || host.sent[0].TargetID != "u" || sentText(host.sent[0].Message) != "抽卡后台同步完成\n绳匠 · 10000001\n新增 45 条，档案共 45 条。" {
		t.Fatalf("notified %v", host.sent)
	}
}

// When the host holds as many background events of the plugin as it allows,
// the page is told so and nothing is read; later the sync runs.
func TestBackgroundSyncRefusedDetachFailsThePage(t *testing.T) {
	a, _, accounts, host := syncHost(t)
	host.refuse = true
	input := map[string]any{"account_ref": "account", "role_ref": "role", "confirm": true}
	if end, _ := host.manage("gacha.sync.background", input); end["type"] != "error" || end["code"] != "plugin.game_background_busy" || end["message"] != busyReply || len(accounts.reads) != 0 {
		t.Fatalf("the refused action ended with %v after %d reads", end, len(accounts.reads))
	}
	host.refuse = false
	if end, _ := host.manage("gacha.sync.background", input); end["type"] != "result" || len(host.sent) != 0 {
		t.Fatalf("the later action ended with %v, sent %v", end, host.sent)
	}
	checkSyncReads(t, a, accounts, "manage-2")
}
