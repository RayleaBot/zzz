package images

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/RayleaBot/plugin-zzz/internal/app"
)

func TestCardFollowsZZZPlugin(t *testing.T) {
	asked := []string{}
	context := app.ImageContext{Query: func(operation string, input map[string]any) (app.QueryResult, error) {
		asked = append(asked, operation)
		if operation == "zzz.profile" {
			return app.QueryResult{Data: map[string]any{"stats": map[string]any{"active_days": json.Number("300"), "avatar_num": json.Number("30"), "buddy_num": json.Number("20"),
				"cur_period_zone_layer_count": json.Number("7"), "world_level_name": "等级 5"}}}, nil
		}
		return app.QueryResult{Data: map[string]any{"list": []any{map[string]any{"id": json.Number("53001"), "rarity": "S", "star": json.Number("2"), "level": json.Number("60")}}}}, nil
	}}
	result := app.QueryResult{Operation: "zzz.characters", Role: app.Role{UID: "10000001", Nickname: "绳匠", Level: 60, Region: "prod_gf_cn"},
		Data: map[string]any{"avatar_list": []any{map[string]any{"id": json.Number("1191"), "rarity": "S", "rank": json.Number("1"), "level": json.Number("60"), "element_type": json.Number("202"), "sub_element_type": json.Number("0")}}}}
	image, ok := Card(context, result)
	// 角色 reads the agents itself and adds the index and Bangboo.
	if !ok || strings.Join(asked, ",") != "zzz.profile,zzz.buddies" {
		t.Fatalf("image = %v asked = %v", image.Data, asked)
	}
	// Upstream labels the player card with the world level.
	if player := image.Data["player"].(map[string]any); player["region"] != "等级 5" {
		t.Errorf("player = %v", player)
	}
	stats := image.Data["stats"].([]any)
	if stats[0].(map[string]any)["value"] != "300" || stats[3].(map[string]any)["label"] != "式舆防卫战" {
		t.Errorf("stats = %v", stats)
	}
	agent := image.Data["agents"].([]any)[0].(map[string]any)
	bangboo := image.Data["bangboo"].([]any)[0].(map[string]any)
	if agent["rank"] != 1 || agent["rarity"] != "S" || bangboo["star"] != 2 || len(image.Data["bars"].([]int)) != 8 {
		t.Errorf("agent = %v bangboo = %v", agent, bangboo)
	}
}
