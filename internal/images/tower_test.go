package images

import (
	"encoding/json"
	"testing"

	"github.com/RayleaBot/zzz/internal/app"
)

func TestTowerFollowsZZZPlugin(t *testing.T) {
	agent := func(score string) map[string]any {
		return map[string]any{"rarity": "S", "score": json.Number(score), "rank_percent": json.Number("123")}
	}
	data := map[string]any{
		"climbing_tower_s2": map[string]any{"climbing_tower_layer": json.Number("30"), "floor_mvp_num": json.Number("12")},
		"climbing_tower_s4": map[string]any{
			"layer_info":               map[string]any{"climbing_tower_layer": json.Number("50"), "total_score": json.Number("98765")},
			"mvp_info":                 map[string]any{"floor_mvp_num": json.Number("40"), "rank_percent": json.Number("512")},
			"display_avatar_rank_list": []any{agent("1"), agent("2"), agent("3"), agent("4")},
		},
	}
	image, ok := Tower(app.ImageContext{Game: app.Game{Prefix: "%"}}, app.QueryResult{Data: data})
	if !ok || image.Data["rank_note"].(map[string]any)["state"] != "" {
		t.Fatalf("image = %v", image.Data)
	}
	seasons := image.Data["seasons"].(map[string]any)
	s2, s4 := seasons["s2"].(map[string]any), seasons["s4"].(map[string]any)
	if seasons["s1"] != nil || s2["layer"] != "30" || s2["mvp"] != "12" {
		t.Errorf("s2 = %v", s2)
	}
	// Later seasons nest their floor and flawless counts; three agents show.
	if s4["layer"] != "50" || s4["score"] != "98765" || s4["mvp"] != "40" || s4["rank"] != "5.12%" || len(s4["agents"].([]any)) != 3 {
		t.Errorf("s4 = %v", s4)
	}
	if _, ok := Tower(app.ImageContext{}, app.QueryResult{Data: map[string]any{}}); ok {
		t.Error("no season answers in text")
	}
}
