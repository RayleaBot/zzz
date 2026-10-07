package images

import (
	"encoding/json"
	"testing"

	"github.com/RayleaBot/zzz/internal/app"
)

func TestHoloBossFollowsZZZPlugin(t *testing.T) {
	boss := func(star, minute, second, rank string, flawless bool) map[string]any {
		return map[string]any{"star": json.Number(star), "rank": json.Number(rank),
			"challenge_time": map[string]any{"minute": json.Number(minute), "second": json.Number(second)},
			"boss":           map[string]any{"name": "首领", "medal": map[string]any{"is_no_injured": flawless}},
			"avatar_list":    []any{map[string]any{"rank": json.Number("1"), "role_square_url": "http://insecure.example/a.png"}, map[string]any{"role_square_url": ""}}}
	}
	data := map[string]any{"unlock": true, "start_time": zzzTime(1), "end_time": zzzTime(15),
		"list": []any{boss("4", "1", "30", "312", true), boss("3", "2", "45", "8", false), boss("2", "0", "0", "0", false)}}
	image, ok := HoloBoss(app.ImageContext{Game: app.Game{Prefix: "%"}}, app.QueryResult{Data: data})
	if !ok || image.Data["time"] != "04:15" || image.Data["stars"] != 9 || image.Data["flawless"] != 1 || image.Data["rank_note"].(map[string]any)["state"] != "" {
		t.Fatalf("image = %v", image.Data)
	}
	list := image.Data["list"].([]any)
	first, second, third := list[0].(map[string]any), list[1].(map[string]any), list[2].(map[string]any)
	// Ranks come in hundredths, or in percent when small; none shows nothing.
	if first["rank"] != "3.12%" || first["rank_bg"] != 2 || second["rank"] != "8.00%" || second["rank_bg"] != 3 || third["rank"] != nil {
		t.Errorf("ranks = %v %v %v", first["rank"], second["rank"], third["rank"])
	}
	if stars := first["stars"].([]any); len(stars) != 4 || stars[3] != true || second["stars"].([]any)[3] != false {
		t.Errorf("stars = %v", first["stars"])
	}
	slots := first["team"].(map[string]any)["slots"].([]any)
	if slots[0].(map[string]any)["rarity"] != "S" || slots[1] != nil || slots[2] != nil {
		t.Errorf("slots = %v", slots)
	}
	data["unlock"] = false
	if _, ok := HoloBoss(app.ImageContext{}, app.QueryResult{Data: data}); ok {
		t.Error("a locked mode answers in text like upstream")
	}
}
