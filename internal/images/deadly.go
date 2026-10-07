package images

import "github.com/RayleaBot/zzz/internal/app"

// deadlyArtwork is what deadly adds.
var deadlyArtwork = [][2]string{
	{"deadly-images-BgFrame01", "resources/deadly/images/BgFrame01.png"},
	{"deadly-images-PetSelectBG", "resources/deadly/images/PetSelectBG.png"},
	{"deadly-images-block-bg-m", "resources/deadly/images/block-bg-m.png"},
	{"deadly-images-hard-module-inner-bg-m-83b7d1a1", "resources/deadly/images/hard-module-inner-bg-m.83b7d1a1.png"},
	{"deadly-images-star-icon-dark", "resources/deadly/images/star-icon-dark.png"},
	{"deadly-images-star-icon-hard-13c9ee51", "resources/deadly/images/star-icon-hard.13c9ee51.png"},
	{"deadly-images-star-icon-light", "resources/deadly/images/star-icon-light.png"},
}

// Deadly draws Deadly Assault the way ZZZ-Plugin's deadly page does: the
// player's icon and name with the total score, rank and stars, the period,
// then each boss fought with its buff, score, stars, clear time and team, and
// after them the hard-mode bosses with the hard-mode rank.
func Deadly(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	data := result.Data
	// Upstream answers in text while the period has no data.
	if has, _ := data["has_data"].(bool); !has {
		return app.Image{}, false
	}
	resources := newRecordResources(context, commonArtwork, deadlyArtwork)
	items := func(field string, hard bool) []any {
		list := []any{}
		entries, _ := data[field].([]any)
		for _, raw := range entries {
			item, _ := raw.(map[string]any)
			entry := map[string]any{"hard": hard, "score": app.Text(item["score"]), "time": recordTime(item["challenge_time"], "2006.01.02 15:04:05"), "team": resources.team(item)}
			if hard {
				percent := app.Int(data["hard_rank_percent"])
				entry["rank_bg"], entry["rank"] = rankBackground(percent), rankText(percent)
			}
			if buffers, _ := item["buffer"].([]any); len(buffers) > 0 {
				buffer, _ := buffers[0].(map[string]any)
				entry["buff"] = resources.official(buffer["icon"])
			}
			if bosses, _ := item["boss"].([]any); len(bosses) > 0 {
				boss, _ := bosses[0].(map[string]any)
				entry["name"], entry["bg"], entry["icon"] = app.Text(boss["name"]), resources.official(boss["bg_icon"]), resources.official(boss["icon"])
				if race := app.Text(boss["race_icon"]); race != "" {
					entry["race"] = resources.official(race)
				}
			}
			stars := []any{}
			star, total := app.Int(item["star"]), app.Int(item["total_star"])
			for index := range max(total, star) {
				stars = append(stars, index < star)
			}
			entry["stars"] = stars
			list = append(list, entry)
		}
		return list
	}
	var hard []any
	if has, _ := data["has_hard"].(bool); has {
		hard = items("hard_list", true)
	}
	percent := app.Int(data["rank_percent"])
	return app.Image{Template: "deadly", Data: map[string]any{
		"avatar": resources.official(data["avatar_icon"]), "nickname": app.Text(data["nick_name"]),
		"score": app.Text(data["total_score"]), "rank_bg": rankBackground(percent), "rank": rankText(percent), "stars": app.Text(data["total_star"]),
		"begin": recordTime(data["start_time"], "2006.01.02"), "end": recordTime(data["end_time"], "2006.01.02"),
		"list":      append(items("list", false), hard...),
		"rank_note": rankNote(context),
	}, Resources: resources.List}, true
}
