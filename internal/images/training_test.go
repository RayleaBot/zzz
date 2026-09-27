package images

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/RayleaBot/plugin-zzz/internal/app"
)

func TestTrainingFollowsZZZPlugin(t *testing.T) {
	agent := func(id, rarity string, rank int, weapon string) map[string]any {
		entry := map[string]any{"id": json.Number(id), "rarity": rarity, "level": json.Number("60"), "rank": json.Number(strconv.Itoa(rank)),
			"skills": []any{map[string]any{"level": json.Number("12")}, map[string]any{"level": json.Number("12")}}}
		if weapon != "" {
			entry["weapon"] = map[string]any{"id": json.Number("14102"), "rarity": weapon, "level": json.Number("60"), "star": json.Number("1"), "name": "音擎"}
		}
		return entry
	}
	context := app.ImageContext{
		SavedPanels: func(uid string) []app.CharacterPanel {
			if uid != "10000001" {
				return nil
			}
			return app.NormalizePanels(app.QueryResult{Data: map[string]any{"avatar_list": []any{agent("1011", "A", 6, "A"), agent("1191", "S", 0, "S"), agent("1021", "S", 5, "")}}}, app.Catalog{})
		},
		Score: func(panel app.CharacterPanel) (app.CharacterPanel, error) {
			raw, _ := json.Marshal(map[string]any{"pieces": []any{map[string]any{"slot": 1, "score": 60, "grade": "SSS"}, map[string]any{"slot": 2, "score": 45, "grade": "A"}}})
			panel.ScoreDetail = &app.ScoreDetail{Raw: raw}
			return panel, nil
		},
	}
	image, ok := Training(context, app.QueryResult{Role: app.Role{UID: "10000001", Region: "prod_gf_cn"}})
	if !ok {
		t.Fatal("training")
	}
	if _, ok := Training(context, app.QueryResult{Role: app.Role{UID: "10000002"}}); ok {
		t.Fatal("drawn without kept panels")
	}
	general := image.Data["general"].([]any)
	want := []string{"2/3", "50.0%", "2/3", "3"}
	for index, value := range want {
		if general[index].(map[string]any)["value"] != value {
			t.Errorf("general[%d] = %v, want %s", index, general[index], value)
		}
	}
	// Proficiency: discs 105*2 + level 120 + skills 24*base + Mindscape
	// 2*base*rank + W-Engine level*2 + refinement*2*base: the A-rank at M6
	// (602) outranks the S-rank at M0 (580) and the S-rank at M5 without an
	// engine (500).
	list2 := image.Data["list"].([]any)
	first, last := list2[0].(map[string]any), list2[2].(map[string]any)
	if first["rank"] != 6 || list2[1].(map[string]any)["rank"] != 0 || last["rank"] != 5 || last["weapon"] != nil || first["comment"] != "B" {
		t.Errorf("list = %v", list2)
	}
	if skills := first["skills"].([]any); skills[0] != "12" || skills[1] != "" {
		t.Errorf("skills = %v", skills)
	}
}
