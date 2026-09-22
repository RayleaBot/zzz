package images

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/RayleaBot/plugin-zzz/internal/app"
)

func zzzTime(day int) map[string]any {
	return map[string]any{"year": json.Number("2026"), "month": json.Number("9"), "day": json.Number(strconv.Itoa(day)), "hour": json.Number("20"), "minute": json.Number("5"), "second": json.Number("9")}
}

func TestAbyssFollowsZZZPlugin(t *testing.T) {
	room := func(score string) map[string]any {
		return map[string]any{"rating": "S", "score": json.Number(score), "challenge_time": zzzTime(12), "buffer": map[string]any{"title": "强击"},
			"avatar_list": []any{map[string]any{"rank": json.Number("2"), "rarity": "S", "role_square_url": "http://insecure.example/a.png"}},
			"buddy":       map[string]any{"id": json.Number("53001"), "rarity": "A"}}
	}
	data := map[string]any{"hadal_ver": "v2", "hadal_info_v2": map[string]any{
		"hadal_begin_time": zzzTime(1), "hadal_end_time": zzzTime(15),
		"brief":               map[string]any{"score": json.Number("150000"), "rank_percent": json.Number("312"), "rating": "S+", "battle_time": json.Number("405"), "challenge_time": zzzTime(12)},
		"fitfh_layer_detail":  map[string]any{"layer_challenge_info_list": []any{room("50000"), room("48000"), room("47000")}},
		"fourth_layer_detail": map[string]any{"rating": "S", "challenge_time": zzzTime(10), "layer_challenge_info_list": []any{room("0"), room("0")}},
	}}
	context := app.ImageContext{Game: app.Game{Prefix: "%"}}
	image, ok := Abyss(context, app.QueryResult{Role: app.Role{UID: "10000001", Nickname: "绳匠", Level: 60, Region: "prod_gf_cn"}, Data: data})
	if !ok {
		t.Fatal("abyss")
	}
	fifth := image.Data["fifth"].(map[string]any)
	for key, want := range map[string]any{"max_score": true, "rank_bg": 2, "rank": "3.12%", "rating": "SP", "battle_time": "06:45", "time": "2026-09-12 20:05:09"} {
		if fifth[key] != want {
			t.Errorf("%s = %v, want %v", key, fifth[key], want)
		}
	}
	rooms := fifth["rooms"].([]any)
	first := rooms[0].(map[string]any)
	team := first["team"].(map[string]any)
	if first["max_score"] != true || rooms[1].(map[string]any)["max_score"] != false || first["buff"] != "强击" || first["no"] != 1 {
		t.Errorf("room = %v", first)
	}
	if slots := team["slots"].([]any); len(slots) != 3 || slots[1] != nil || team["buddy"] == nil {
		t.Errorf("team = %v", team)
	}
	lower := image.Data["lower"].([]any)
	if len(lower) != 1 || lower[0].(map[string]any)["name"] != "剧变节点第四防线" || len(lower[0].(map[string]any)["teams"].([]any)) != 2 {
		t.Errorf("lower = %v", lower)
	}
	if image.Data["rank_note"].(map[string]any)["state"] != "" || image.Data["begin"] != "2026-09-01 20:05:09" {
		t.Errorf("data = %v", image.Data)
	}
	data["hadal_ver"] = "v1"
	if _, ok := Abyss(context, app.QueryResult{Data: data}); ok {
		t.Error("an older version answers in text like upstream")
	}
}
