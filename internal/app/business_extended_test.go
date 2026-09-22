package app

import (
	"encoding/json"
	"strings"
	"testing"
)

func decoded(t *testing.T, source string) map[string]any {
	t.Helper()
	var result map[string]any
	decoder := json.NewDecoder(strings.NewReader(source))
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestMonthlyAndChallengeViewsUseBusinessFields(t *testing.T) {
	for _, tc := range []struct {
		game, op, data string
		want           []string
	}{
		{"genshin", "monthly", `{"data_month":9,"month_data":{"current_primogems":1200,"current_mora":30000,"group_by":[{"action":"活动奖励","num":800,"percent":60}]} }`, []string{"本月原石：1200", "本月摩拉：30000", "活动奖励：800"}},
		{"starrail", "currency_wars", `{"grid_fight_brief":{"weekly_score_cur":"80","weekly_score_max":"100","division":{"name":"黄金"}},"grid_fight_archive_list":[{"brief":{"remain_hp":"42","archive_rank":"1"},"lineup":{"front_roles":[{"name":"测试角色"}]}}]}`, []string{"每周积分：80 / 100", "名称：黄金", "前排：测试角色", "剩余生命：42"}},
		{"starrail", "peak", `{"challenge_peak_best_record_brief":{"boss_stars":3,"mob_stars":9},"challenge_peak_records":[{"boss_record":{"round_num":2,"avatars":[{"name":"测试角色"}]}}]}`, []string{"首领星数：3", "关卡星数：9", "轮数：2", "队伍：测试角色"}},
		{"zzz", "deadly", `{"total_score":10000,"list":[{"score":3000,"avatar_list":[{"name":"测试角色"}]}],"hard_list":[{"score":7000}]}`, []string{"总分：10000", "得分：3000", "困难记录 1", "得分：7000"}},
		{"zzz", "exploration", `{"area_collections":[{"name":"六分街","collection_progress":80,"map_collections":[{"name":"街区","collection_progress":50}]}]}`, []string{"六分街", "探索进度：80", "街区"}},
	} {
		t.Run(tc.game+tc.op, func(t *testing.T) {
			view := BusinessView(Game{ID: tc.game}, Operation{Name: tc.game + "." + tc.op, Label: "测试"}, QueryResult{Data: decoded(t, tc.data)}, Catalog{})
			for _, text := range tc.want {
				if !strings.Contains(view.Text(), text) {
					t.Fatalf("missing business output %q", text)
				}
			}
			if strings.Contains(view.Text(), "avatar_list") || strings.Contains(view.Text(), "grid_fight_brief") {
				t.Fatal("raw protocol key shown to user")
			}
		})
	}
}
func TestGachaSummaryNeverClaimsCompleteHistory(t *testing.T) {
	view := BusinessView(Game{ID: "starrail"}, Operation{Name: "starrail.gacha_summary"}, QueryResult{Data: decoded(t, `{"gacha_type":"11","pool":{"cards":[{"name":"池一","total_count":90,"up_count":1}]},"five_star":{"list":[{"gacha_count":10,"item":null},{"gacha_count":80,"item":{"name":"测试五星"}}],"has_more":true}}`)}, Catalog{})
	for _, text := range []string{"总抽数 90", "当前未出五星抽数：10", "测试五星：80 抽", "不是完整逐抽历史", "首批"} {
		if !strings.Contains(view.Text(), text) {
			t.Fatalf("missing boundary %q", text)
		}
	}
}
func TestExtendedCommandInputs(t *testing.T) {
	app := &App{}
	for _, tc := range []struct{ mode, valid, invalid, key string }{{"month", "9", "13", "month"}, {"year_month", "202609", "202613", "month"}, {"rogue_period", "3", "4", "schedule_type"}, {"peak_period", "往期", "4", "schedule_type"}, {"starrail_pool", "21", "100", "gacha_type"}} {
		input, uid, err := app.commandInput(Operation{Input: tc.mode}, []string{tc.valid, "100000001"}, nil)
		if err != nil || uid != "100000001" || input[tc.key] == nil {
			t.Fatal("valid selector failed")
		}
		if _, _, err := app.commandInput(Operation{Input: tc.mode}, []string{tc.invalid}, nil); err == nil {
			t.Fatal("invalid selector accepted")
		}
	}
	// 往期 is the last period elsewhere, and the recent runs for Anomaly Arbitration.
	for mode, want := range map[string]int{"period": 2, "peak_period": 3} {
		if input, _, err := app.commandInput(Operation{Input: mode}, []string{"往期"}, nil); err != nil || input["schedule_type"] != want {
			t.Errorf("%s 往期 = %v, %v", mode, input["schedule_type"], err)
		}
	}
}
