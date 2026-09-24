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
func TestExtendedCommandInputs(t *testing.T) {
	app := &App{}
	for _, tc := range []struct{ mode, valid, key string }{{"year_month", "202609", "month"}, {"period", "上期", "schedule_type"}} {
		input, uid, err := app.commandInput(Operation{Input: tc.mode}, []string{tc.valid, "100000001"}, nil)
		if err != nil || uid != "100000001" || input[tc.key] == nil {
			t.Fatal("valid selector failed")
		}
	}
	if _, _, err := app.commandInput(Operation{Input: "year_month"}, []string{"202613"}, nil); err == nil {
		t.Fatal("invalid month accepted")
	}
	// 往期 is the last period.
	if input, _, err := app.commandInput(Operation{Input: "period"}, []string{"往期"}, nil); err != nil || input["schedule_type"] != 2 {
		t.Errorf("往期 = %v, %v", input["schedule_type"], err)
	}
	// Upstream names periods only in words; a number after the command is
	// not one.
	if input, uid, err := app.commandInput(Operation{Input: "period"}, []string{"2"}, nil); err != nil || input["schedule_type"] != nil || uid != "2" {
		t.Errorf("2 = %v, %q, %v", input, uid, err)
	}
}
