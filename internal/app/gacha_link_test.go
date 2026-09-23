package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/RayleaBot/plugin-zzz/internal/gacha"
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

func TestGachaLinkFetchesTheWholeSignalHistory(t *testing.T) {
	a := pluginApp(t)
	a.LinkHTTP = &http.Client{Transport: signalHistory{}}
	link := gachaLink{key: "abcdefghij", region: "prod_gf_cn", biz: "nap_cn"}
	info, err := a.Syncs.Start(a.Gacha, gacha.SyncChoice{Link: true}, "10000001", link.region, false)
	if err != nil {
		t.Fatal(err)
	}
	job := &gachaLinkJob{link: link, uid: "10000001", sync: info.Ref, before: map[string]int{}}
	result, err := a.stepGachaLink(context.Background(), job, time.Now().Add(time.Minute))
	if err != nil || result == nil || result.Added != 22 {
		t.Fatal(result, err)
	}
	archive, _ := a.Gacha.Read("10000001", link.region)
	summary := gachaLinkSummary(job.before, gachaPoolCounts(archive))
	if !strings.Contains(summary, "抽卡记录更新成功，共6个卡池\n") || !strings.Contains(summary, "独家频段新增21条记录，一共21条记录") || !strings.Contains(summary, "邦布频段新增1条记录，一共1条记录") {
		t.Fatal(summary)
	}
	if _, err := a.gachaLinkPage(context.Background(), gachaLink{key: "expired-key", region: "prod_gf_cn", biz: "nap_cn"}, "2001", "2", "0", 1); err == nil || friendlyError(err) != "未查询到uid，请检查链接是否正确" {
		t.Fatal(err)
	}
}
