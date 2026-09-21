package images

import (
	"encoding/json"
	"testing"
	"time"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

func TestNoteFollowsZZZPlugin(t *testing.T) {
	data := map[string]any{
		"energy":    map[string]any{"progress": map[string]any{"current": json.Number("179"), "max": json.Number("240")}, "restore": json.Number("21960")},
		"vitality":  map[string]any{"current": json.Number("400"), "max": json.Number("400")},
		"vhs_sale":  map[string]any{"sale_state": "SaleStateDoing"},
		"card_sign": "CardSignNo",
	}
	image, ok := Note(gamekit.ImageContext{Now: time.Now()}, gamekit.QueryResult{Role: gamekit.Role{UID: "10000001", Nickname: "绳匠", Level: 60, Region: "prod_gf_cn"}, Data: data})
	if !ok || image.Template != "note" {
		t.Fatalf("image = %+v", image)
	}
	energy := image.Data["energy"].(map[string]any)
	if energy["percent"] != 74 || energy["rest"] != "6小时6分钟" {
		t.Errorf("energy = %v", energy)
	}
	if player := image.Data["player"].(map[string]any); player["region"] != "新艾利都" {
		t.Errorf("player = %v", player)
	}
	want := [][2]any{{true, "400"}, {true, "正在营业"}, {false, "未完成"}}
	for index, raw := range image.Data["activities"].([]any) {
		item := raw.(map[string]any)
		if item["finished"] != want[index][0] || item["value"] != want[index][1] {
			t.Errorf("activity %d = %v", index, item)
		}
	}
	if _, ok := Note(gamekit.ImageContext{}, gamekit.QueryResult{Data: map[string]any{}}); ok {
		t.Error("a result without energy should keep the summary card")
	}
}
