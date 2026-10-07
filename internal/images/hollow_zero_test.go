package images

import (
	"encoding/json"
	"testing"

	"github.com/RayleaBot/zzz/internal/app"
)

func TestHollowZeroReadsTheThroneOnlyOnceDamaged(t *testing.T) {
	queried := 0
	context := app.ImageContext{Query: func(operation string, _ map[string]any) (app.QueryResult, error) {
		queried++
		if operation != "zzz.hollow_zero_detail" {
			t.Errorf("queried %s", operation)
		}
		return app.QueryResult{Data: map[string]any{"abyss_throne_max": map[string]any{"time": json.Number("3725"), "max_damage": json.Number("120000"),
			"avatar_list": []any{map[string]any{"rarity": "S", "rank": json.Number("0"), "level": json.Number("60"), "damage_rate": "55.5"}},
			"buddy_list":  []any{map[string]any{"rarity": "A", "level": json.Number("60")}}}}}, nil
	}}
	data := map[string]any{"abyss_level": map[string]any{"cur_level": json.Number("50"), "max_level": json.Number("60")},
		"abyss_collect": []any{map[string]any{"cur_collect": json.Number("3"), "max_collect": json.Number("9")}}, "abyss_throne": map[string]any{"max_damage": json.Number("0")}}
	image, ok := HollowZero(context, app.QueryResult{Data: data})
	collections := image.Data["collections"].([]any)
	if !ok || queried != 0 || image.Data["throne"] != nil || image.Data["level"] != "50 / 60" || len(collections) != 5 ||
		collections[0].(map[string]any)["value"] != "3 / 9" || collections[4].(map[string]any)["name"] != "战术棱镜方案" {
		t.Fatalf("undamaged throne: %v, queried %d", image.Data, queried)
	}
	data["abyss_throne"] = map[string]any{"max_damage": json.Number("120000")}
	image, _ = HollowZero(context, app.QueryResult{Data: data})
	throne := image.Data["throne"].(map[string]any)
	slots := throne["slots"].([]any)
	if queried != 1 || throne["time"] != "01:02:05" || len(slots) != 3 || slots[1] != nil || slots[0].(map[string]any)["damage"] != "55.5" || throne["buddy"] == nil {
		t.Errorf("throne = %v", throne)
	}
}

func TestLostVoidFollowsZZZPlugin(t *testing.T) {
	data := map[string]any{"abyss_duty": map[string]any{"cur_duty": json.Number("4"), "max_duty": json.Number("10")},
		"abyss_max": map[string]any{"max_name": "高危", "heat_count": json.Number("80"), "max_count": json.Number("2"), "best_time": json.Number("59")}}
	image, ok := LostVoid(app.ImageContext{}, app.QueryResult{Data: data})
	matrix := image.Data["matrix"].(map[string]any)
	if !ok || image.Data["duty"] != "4 / 10" || len(image.Data["collections"].([]any)) != 7 || matrix["time"] != "00:00:59" || matrix["heat"] != "80" {
		t.Errorf("image = %v", image.Data)
	}
	if _, ok := LostVoid(app.ImageContext{}, app.QueryResult{Data: map[string]any{}}); ok {
		t.Error("no record answers in text")
	}
}
