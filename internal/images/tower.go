package images

import "github.com/RayleaBot/zzz/internal/app"

// Tower draws the Simulated Battle Trial the way ZZZ-Plugin's climbingTower
// page does: the player card and each season played with its medal, floor,
// score, flawless clears and rank, and the top three agents of the latest
// season. The frame image upstream's stylesheet names is not in its
// repository, so the page has none there either.
func Tower(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	data := result.Data
	resources := newRecordResources(context, commonArtwork)
	seasons := map[string]any{}
	for _, key := range []string{"s1", "s2", "s3", "s4"} {
		season, _ := data["climbing_tower_"+key].(map[string]any)
		if season == nil {
			continue
		}
		// The first two seasons keep their fields at the top.
		layer, mvp := season, season
		if key == "s3" || key == "s4" {
			layer, _ = season["layer_info"].(map[string]any)
			mvp, _ = season["mvp_info"].(map[string]any)
		}
		entry := map[string]any{"medal": resources.official(layer["medal_icon"]), "layer": app.Text(layer["climbing_tower_layer"]),
			"score": app.Text(layer["total_score"]), "mvp": app.Text(mvp["floor_mvp_num"]), "rank": rankText(app.Int(mvp["rank_percent"]))}
		agents := []any{}
		list, _ := season["display_avatar_rank_list"].([]any)
		for index, raw := range list {
			if index == 3 {
				break
			}
			agent, _ := raw.(map[string]any)
			agents = append(agents, map[string]any{"rarity": app.Text(agent["rarity"]), "icon": resources.official(agent["icon"]),
				"score": app.Text(agent["score"]), "rank": rankText(app.Int(agent["rank_percent"]))})
		}
		entry["agents"] = agents
		seasons[key] = entry
	}
	// Upstream draws whatever arrives; an answer with no season stays in text.
	if len(seasons) == 0 {
		return app.Image{}, false
	}
	return app.Image{Template: "tower", Data: map[string]any{
		"player": playerCard(result.Role), "seasons": seasons, "rank_note": rankNote(context),
	}, Resources: resources.List}, true
}
