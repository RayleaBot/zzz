package images

import (
	"encoding/json"
	"testing"

	"github.com/RayleaBot/plugin-zzz/internal/app"
)

func TestDeadlyFollowsZZZPlugin(t *testing.T) {
	boss := func(name string) map[string]any {
		return map[string]any{"score": json.Number("25000"), "star": json.Number("2"), "total_star": json.Number("3"), "challenge_time": zzzTime(12),
			"boss":        []any{map[string]any{"name": name, "icon": "http://insecure.example/boss.png"}},
			"buffer":      []any{map[string]any{"icon": "http://insecure.example/buff.png"}},
			"avatar_list": []any{map[string]any{"rank": json.Number("0"), "role_square_url": ""}}}
	}
	data := map[string]any{"has_data": true, "nick_name": "绳匠", "total_score": json.Number("65000"), "rank_percent": json.Number("4999"), "total_star": json.Number("8"),
		"start_time": zzzTime(1), "end_time": zzzTime(15), "list": []any{boss("冥宁芙·双子"), boss("未知复合侵蚀体")},
		"has_hard": true, "hard_rank_percent": json.Number("80"), "hard_list": []any{boss("绝境首领")}}
	image, ok := Deadly(app.ImageContext{Game: app.Game{Prefix: "%"}}, app.QueryResult{Data: data})
	if !ok || image.Data["rank_bg"] != 4 || image.Data["rank"] != "49.99%" || image.Data["begin"] != "2026.09.01" || image.Data["rank_note"].(map[string]any)["state"] != "" {
		t.Fatalf("image = %v", image.Data)
	}
	list := image.Data["list"].([]any)
	first, hard := list[0].(map[string]any), list[2].(map[string]any)
	if len(list) != 3 || first["name"] != "冥宁芙·双子" || first["time"] != "2026.09.12 20:05:09" || first["hard"] != false {
		t.Errorf("first = %v", first)
	}
	if stars := first["stars"].([]any); len(stars) != 3 || stars[1] != true || stars[2] != false {
		t.Errorf("stars = %v", stars)
	}
	// Upstream reads a missing rarity as A.
	if slot := first["team"].(map[string]any)["slots"].([]any)[0].(map[string]any); slot["rarity"] != "A" {
		t.Errorf("slot = %v", slot)
	}
	if hard["hard"] != true || hard["rank_bg"] != 1 || hard["rank"] != "0.80%" {
		t.Errorf("hard = %v", hard)
	}
	data["has_data"] = false
	if _, ok := Deadly(app.ImageContext{}, app.QueryResult{Data: data}); ok {
		t.Error("a period without data answers in text like upstream")
	}
}
