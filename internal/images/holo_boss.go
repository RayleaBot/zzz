package images

import (
	"fmt"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

// holoBossArtwork is what holoBoss adds.
var holoBossArtwork = [][2]string{
	{"holoBoss-images-BgFrame01", "resources/holoBoss/images/BgFrame01.png"},
	{"holoBoss-images-PetSelectBG", "resources/holoBoss/images/PetSelectBG.png"},
	{"holoBoss-images-star-gray-53e3ee51", "resources/holoBoss/images/star-gray.53e3ee51.png"},
	{"holoBoss-images-star-light-d591d065", "resources/holoBoss/images/star-light.d591d065.png"},
}

// HoloBoss draws Holo Annihilation the way ZZZ-Plugin's holoBoss page does:
// the total time, stars and flawless clears, the period, then each boss with
// its medal, clear time, stars out of four, rank and team.
func HoloBoss(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	data := result.Data
	list, _ := data["list"].([]any)
	// Upstream answers in text while the mode is locked or the period empty.
	if unlocked, _ := data["unlock"].(bool); !unlocked || len(list) == 0 {
		return gamekit.Image{}, false
	}
	resources := newRecordResources(context, commonArtwork, holoBossArtwork)
	clock := func(seconds int) string { return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60) }
	items := []any{}
	stars, seconds, flawless := 0, 0, 0
	for _, raw := range list {
		item, _ := raw.(map[string]any)
		boss, _ := item["boss"].(map[string]any)
		medal, _ := boss["medal"].(map[string]any)
		spent, _ := item["challenge_time"].(map[string]any)
		used := gamekit.Int(spent["minute"])*60 + gamekit.Int(spent["second"])
		star := gamekit.Int(item["star"])
		noInjury, _ := medal["is_no_injured"].(bool)
		stars, seconds = stars+star, seconds+used
		if noInjury {
			flawless++
		}
		row := []any{}
		for index := range 4 {
			row = append(row, index < star)
		}
		// This mode's avatars default to S and show only with a portrait.
		slots := []any{}
		avatars, _ := item["avatar_list"].([]any)
		for index := range 3 {
			var slot any
			if index < len(avatars) {
				avatar, _ := avatars[index].(map[string]any)
				if url := gamekit.Text(avatar["role_square_url"]); url != "" {
					level := gamekit.Text(avatar["rarity"])
					if level == "" {
						level = "S"
					}
					slot = map[string]any{"rank": gamekit.Int(avatar["rank"]), "rarity": level, "icon": resources.official(url)}
				}
			}
			slots = append(slots, slot)
		}
		entry := map[string]any{"name": gamekit.Text(boss["name"]), "icon": resources.official(boss["icon"]), "flawless": noInjury,
			"time": clock(used), "stars": row, "team": map[string]any{"slots": slots}}
		if icon := gamekit.Text(medal["medal_icon"]); icon != "" {
			entry["medal"] = resources.official(icon)
		}
		// The rank comes in hundredths, or already in percent when small.
		if rank := gamekit.Int(item["rank"]); rank > 0 {
			value := float64(rank)
			if rank > 100 {
				value /= 100
			}
			entry["rank"], entry["rank_bg"] = fmt.Sprintf("%.2f%%", value), rankBackground(int(value*100))
		}
		items = append(items, entry)
	}
	return gamekit.Image{Template: "holo-boss", Data: map[string]any{
		"time": clock(seconds), "stars": stars, "flawless": flawless,
		"begin": recordTime(data["start_time"], "2006.01.02"), "end": recordTime(data["end_time"], "2006.01.02"),
		"list": items, "rank_command": rankCommand(context, "拟境"),
	}, Resources: resources.List}, true
}
