package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestGachaLinkParsesZenlessLinksOnly(t *testing.T) {
	link, ok, err := parseGachaLink("https://webstatic.mihoyo.com/nap/event/e20230424gacha/index.html?authkey_ver=1&authkey=abcdefgh%2Bij&game_biz=nap_cn#/log")
	if !ok || err != nil || link.key != "abcdefgh+ij" || link.region != "prod_gf_cn" || link.biz != "nap_cn" {
		t.Fatal(link, ok, err)
	}
	for _, text := range []string{
		"https://webstatic.mihoyo.com/hk4e/event/e20190909gacha-v3/index.html?authkey=abcdefghij",
		"https://webstatic.mihoyo.com/hkrpg/event/e20211215gacha-v2/index.html?authkey=abcdefghij&game_biz=hkrpg_cn",
		"抽卡分析",
	} {
		if _, ok, _ := parseGachaLink(text); ok {
			t.Fatal("claimed:", text)
		}
	}
	if _, ok, err := parseGachaLink("nap_cn authkey="); !ok || err == nil || !strings.Contains(friendlyError(err), "抽卡链接格式错误") {
		t.Fatal("a broken link was accepted")
	}
}

// signalHistory serves the official signal search for UID 10000001: 21
// exclusive channel and one Bangboo channel records.
type signalHistory struct{}

func (signalHistory) RoundTrip(request *http.Request) (*http.Response, error) {
	query := request.URL.Query()
	pool := query.Get("gacha_type")
	body := map[string]any{"retcode": 0, "message": ""}
	if query.Get("authkey") != "abcdefghij" || request.URL.Host != "public-operation-common.mihoyo.com" || query.Get("real_gacha_type") != map[string]string{"1001": "1", "2001": "2", "3001": "3", "5001": "5", "12001": "102", "13001": "103"}[pool] {
		body["retcode"] = -101
	} else {
		records := []any{}
		count, base := map[string]int{"2001": 21, "5001": 1}[pool], map[string]int{"2001": 5000, "5001": 4000}[pool]
		first := 1
		if end, _ := strconv.Atoi(query.Get("end_id")); end != 0 {
			first = base - end + 1
		}
		for n := first; n <= count && len(records) < 20; n++ {
			records = append(records, map[string]any{"uid": "10000001", "gacha_id": "0", "gacha_type": query.Get("real_gacha_type"), "item_id": "1191", "count": "1", "time": fmt.Sprintf("2024-01-01 00:00:%02d", 59-n), "name": "艾莲", "item_type": "代理人", "rank_type": "4", "id": strconv.Itoa(base - n)})
		}
		body["data"] = map[string]any{"list": records, "region": "prod_gf_cn", "region_time_zone": 0}
	}
	raw, _ := json.Marshal(body)
	recorder := httptest.NewRecorder()
	_, _ = recorder.Write(raw)
	return recorder.Result(), nil
}

// roundTripFunc answers a client's requests with a function.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// A link sent in private chat, run through the SDK as the host runs it: its
// event moves to the background, says that the link was read and fetches
// every page of the signal search, ten seconds a page, far beyond the
// event's time, then answers ZZZ-Plugin's report of the channels.
func TestGachaLinkReadsTheWholeHistoryInOneDetachedEvent(t *testing.T) {
	a := pluginApp(t)
	clock := &fakeClock{at: time.Unix(1_800_000_000, 0)}
	a.clock = clock
	pages := 0
	a.LinkHTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		pages++
		clock.set(clock.Now().Add(10 * time.Second))
		return signalHistory{}.RoundTrip(r)
	})}
	host := newSDKHost(t, a, func(hostCall) (map[string]any, string) {
		t.Error("a link asked the account service")
		return nil, "plugin.service_unavailable"
	})
	link := "https://webstatic.mihoyo.com/nap/event/e20230424gacha/index.html?authkey_ver=1&authkey=abcdefghij&game_biz=nap_cn#/log"
	end, actions := host.message(link, "")
	if end["type"] != "result" || len(host.detached) != 1 || actionAt(actions, "event.detach") > actionAt(actions, "message.send") || len(host.created) != 0 {
		t.Fatalf("the link event ended with %v, actions %v", end, actions)
	}
	if clock.Now().Sub(time.Unix(1_800_000_000, 0)) <= hostEventTimeout || len(host.sent) != 2 || sentText(host.sent[0].Message) != "抽卡链接解析成功，正在查询抽卡记录，可能耗费一段时间，请勿重复发送" {
		t.Fatalf("%d pages until %v, sent %v", pages, clock.Now(), host.sent)
	}
	summary := sentText(host.sent[1].Message)
	if host.sent[1].TargetType != "private" || host.sent[1].TargetID != "u" || !strings.HasPrefix(summary, "抽卡记录更新成功，共6个卡池") || !strings.Contains(summary, "独家频段新增21条记录，一共21条记录") || !strings.Contains(summary, "邦布频段新增1条记录，一共1条记录") {
		t.Fatalf("answered %+v", host.sent[1])
	}
	if archive, err := a.Gacha.Read("10000001", "prod_gf_cn"); err != nil || len(archive.Records) != 22 {
		t.Fatalf("kept %d records, %v", len(archive.Records), err)
	}
	// A link whose authkey the official service refuses finds no UID.
	host.message(strings.Replace(link, "abcdefghij", "expired-key", 1), "")
	if len(host.sent) != 4 || sentText(host.sent[3].Message) != "未查询到uid，请检查链接是否正确" {
		t.Fatalf("an expired link answered %v", host.sent[2:])
	}
}
