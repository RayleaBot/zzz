package images

import (
	"encoding/json"
	"strconv"
	"testing"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

func TestMonthlyFollowsZZZPlugin(t *testing.T) {
	data := map[string]any{"data_month": "202609", "month_data": map[string]any{
		"list":              []any{map[string]any{"data_type": "PolychromesData", "count": json.Number("12000")}, map[string]any{"data_type": "BooponsData", "count": json.Number("8")}},
		"income_components": []any{map[string]any{"action": "daily_activity_rewards", "num": json.Number("6000"), "percent": json.Number("50")}, map[string]any{"action": "new_rewards", "num": json.Number("10"), "percent": json.Number("0")}},
	}}
	image, ok := Monthly(gamekit.ImageContext{}, gamekit.QueryResult{Role: gamekit.Role{UID: "10000001", Region: "prod_gf_cn"}, Data: data})
	if !ok || image.Data["month"] != "9月" || image.Data["poly"] != "12000" || image.Data["tape"] != "0" || image.Data["boopon"] != "8" {
		t.Fatalf("image = %v", image.Data)
	}
	// Sources take upstream's names, and unknown ones its fallback.
	sources := image.Data["sources"].([]any)
	if sources[0].(map[string]any)["name"] != "日常活跃奖励" || sources[1].(map[string]any)["name"] != "未知奖励" {
		t.Errorf("sources = %v", sources)
	}
	if _, ok := Monthly(gamekit.ImageContext{}, gamekit.QueryResult{Data: map[string]any{}}); ok {
		t.Error("an empty month answers in text like upstream")
	}
}

func TestMonthlyCollectFollowsZZZPlugin(t *testing.T) {
	month := func(key string, poly int) gamekit.SavedMonth {
		return gamekit.SavedMonth{Month: key, Data: map[string]any{"month_data": map[string]any{
			"list": []any{map[string]any{"data_type": "PolychromesData", "count": json.Number(strconv.Itoa(poly))}, map[string]any{"data_type": "BooponsData", "count": json.Number("3")}}}}}
	}
	stats := gamekit.MonthlyStats{Months: []gamekit.SavedMonth{month("2026-07", 1000), month("2026-08", 2000), month("2026-09", 500)}}
	image, ok := MonthlyCollect(gamekit.ImageContext{}, stats)
	months := image.Data["months"].([]any)
	if !ok || image.Data["range"] != "2026年7月～2026年9月" || image.Data["poly"] != "3500" || image.Data["tape"] != "0" || image.Data["boopon"] != "9" ||
		months[0].(map[string]any)["date"] != "2026年9月" || months[0].(map[string]any)["poly"] != "500" {
		t.Errorf("image = %v", image.Data)
	}
	if _, ok := MonthlyCollect(gamekit.ImageContext{}, gamekit.MonthlyStats{}); ok {
		t.Error("no saved month drew a page")
	}
}
