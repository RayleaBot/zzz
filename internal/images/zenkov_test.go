package images

import (
	"encoding/json"
	"testing"

	"github.com/RayleaBot/plugin-zzz/internal/app"
)

func TestZenkovFollowsZZZPlugin(t *testing.T) {
	data := map[string]any{"refresh_time": json.Number("190800"), "season_unlock": true, "big_red_num": json.Number("0"),
		"season_data": map[string]any{"cur_season_id": json.Number("2"), "refresh_time": json.Number("1000000")},
		"map_list": []any{
			map[string]any{"map_name": "甲", "hell_unlock": true, "leave_percent": json.Number("10000")},
			map[string]any{"map_name": "乙", "hell_unlock": true, "leave_percent": json.Number("4567")},
			map[string]any{"map_name": "丙", "hell_unlock": false},
		},
		"max_rank": json.Number("345"), "is_show_percent": true}
	image, ok := Zenkov(app.ImageContext{}, app.QueryResult{Data: data})
	if !ok {
		t.Fatal("zenkov")
	}
	maps := image.Data["maps"].([]any)
	if maps[0].(map[string]any)["leave"] != "100%" || maps[1].(map[string]any)["leave"] != "45.67%" || maps[2].(map[string]any)["leave"] != "0%" {
		t.Errorf("maps = %v", maps)
	}
	weekly, season := image.Data["weekly"].(map[string]any), image.Data["season"].(map[string]any)
	if weekly["refresh"] != "02天05时" || weekly["big_red"] != "0" || weekly["millions"] != "-" || season["refresh"] != "11天" || season["unlocked"] != true {
		t.Errorf("weekly = %v, season = %v", weekly, season)
	}
	if rank := image.Data["rank"].(map[string]any); rank["percent"] != "3.45%" || rank["background"] != 2 {
		t.Errorf("rank = %v", rank)
	}
	data["is_show_percent"] = false
	image, _ = Zenkov(app.ImageContext{}, app.QueryResult{Data: data})
	if rank := image.Data["rank"].(map[string]any); rank["top"] != "TOP 345" {
		t.Errorf("absolute rank = %v", rank)
	}
}

func TestZenkovDetailFormatsRuns(t *testing.T) {
	data := map[string]any{"record_list": []any{
		map[string]any{"difficult": "Hell", "is_success": true, "material_total_value": json.Number("1234567"),
			"start_time":     map[string]any{"year": json.Number("2026"), "month": json.Number("9"), "day": json.Number("3"), "hour": json.Number("7"), "minute": json.Number("5"), "second": json.Number("9")},
			"challenge_time": map[string]any{"hour": json.Number("0"), "minute": json.Number("12"), "second": json.Number("3")},
			"item_list":      []any{map[string]any{"rarity": json.Number("5")}, map[string]any{"rarity": json.Number("2")}}},
		map[string]any{"difficult": "", "challenge_time": map[string]any{"hour": json.Number("1"), "minute": json.Number("2"), "second": json.Number("3")}},
	}}
	image, ok := ZenkovDetail(app.ImageContext{}, app.QueryResult{Data: data})
	records := image.Data["records"].([]any)
	first, second := records[0].(map[string]any), records[1].(map[string]any)
	items := first["items"].([]any)
	if !ok || image.Data["detail"] != true || first["difficulty"] != "高危" || first["hell"] != true || first["start"] != "2026-09-03 07:05:09" ||
		first["duration"] != "12:03" || first["value"] != "1,234,567" || items[0].(map[string]any)["class"] != "rarity-5" || items[1].(map[string]any)["class"] != "no-bg" {
		t.Errorf("first = %v", first)
	}
	if second["difficulty"] != "普通" || second["duration"] != "01:02:03" || second["start"] != "-" || second["value"] != "0" {
		t.Errorf("second = %v", second)
	}
}

func TestExplorationShowsFourCollectionsASubArea(t *testing.T) {
	collection := map[string]any{"num": json.Number("1"), "total": json.Number("2")}
	data := map[string]any{"area_collections": []any{map[string]any{"name": "区域", "collection_progress": json.Number("80"),
		"map_collections": []any{map[string]any{"collection_progress": json.Number("75"), "collections": []any{collection, collection, collection, collection, collection}}}}}}
	image, ok := Exploration(app.ImageContext{}, app.QueryResult{Data: data})
	area := image.Data["areas"].([]any)[0].(map[string]any)
	sub := area["maps"].([]any)[0].(map[string]any)
	if !ok || area["progress"] != "80" || len(sub["collections"].([]any)) != 4 || sub["collections"].([]any)[0].(map[string]any)["count"] != "1/2" {
		t.Errorf("image = %v", image.Data)
	}
}
