package images

import (
	"slices"
	"strconv"
	"testing"

	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/game-plugin-kit/gacha"
)

func TestGachaFollowsZZZPlugin(t *testing.T) {
	// Oldest first: 70 pulls, then Ellen (S), then ten pulls without an S.
	records := []gacha.Record{}
	add := func(count int, pool, id, name, rank string) {
		for range count {
			records = append(records, gacha.Record{ID: strconv.Itoa(9000 + len(records)), GachaType: pool, ItemID: id, Name: name, Rank: rank, ItemType: "代理人", Time: "2026-01-01 12:00:00"})
		}
	}
	add(70, "2", "1011", "安比", "3")
	add(1, "2", "1191", "艾莲", "4")
	add(10, "2", "1011", "安比", "3")
	add(1, "1", "1021", "猫又", "4")
	image, ok := Gacha(gamekit.ImageContext{}, gamekit.GachaImage{UID: "10000001", Role: gamekit.Role{Nickname: "绳匠", Level: 60, Region: "prod_gf_cn"}, Archive: gacha.Archive{Records: records}})
	if !ok || image.Data["player"].(map[string]any)["region"] != "新艾利都" {
		t.Fatalf("image = %+v", image.Data)
	}
	channels := image.Data["channels"].([]any)
	if len(channels) != 6 {
		t.Fatalf("channels = %d", len(channels))
	}
	exclusive := channels[0].(map[string]any)
	for key, want := range map[string]any{"name": "独家频段", "last": "10", "total": 81, "avg_five": "71.0", "avg_up": "71.0", "no_wai": "100%", "tag": "欧狗在此", "band": 1} {
		if exclusive[key] != want {
			t.Errorf("%s = %v, want %v", key, exclusive[key], want)
		}
	}
	if !slices.Contains([]int{6, 14, 7}, exclusive["emoji"].(int)) {
		t.Errorf("emoji %v is not from the luckiest set", exclusive["emoji"])
	}
	card := exclusive["cards"].([]any)[0].(map[string]any)
	if card["count"] != "71" || card["up"] != true || card["color"] != "white" {
		t.Errorf("card = %v", card)
	}
	standard := channels[4].(map[string]any)
	if standard["name"] != "常驻频段" || standard["cards"].([]any)[0].(map[string]any)["up"] != false || standard["show_no_wai"] != false || standard["band"] != 3 {
		t.Errorf("standard channel = %v", standard)
	}
	if empty := channels[2].(map[string]any); empty["time_range"] != "还没有抽卡" || empty["last"] != "-" {
		t.Errorf("empty channel = %v", empty)
	}
}
