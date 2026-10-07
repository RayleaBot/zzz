package images

import "github.com/RayleaBot/zzz/internal/app"

// Card draws 卡片 and 角色, one command upstream, the way ZZZ-Plugin's card
// page does: the player card with the world level as its label, days active,
// agents, Bangboo and this Shiyu Defense period's floors, then every agent
// with rank, attribute, square portrait, Mindscape and level, and every
// Bangboo with rank, portrait, star and level. Either command's query is one
// of the three upstream reads; the builder adds the others.
func Card(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	if context.Query == nil {
		return app.Image{}, false
	}
	read := func(operation string) (map[string]any, bool) {
		if result.Operation == operation {
			return result.Data, true
		}
		// Upstream answers nothing when one of its reads fails.
		other, err := context.Query(operation, nil)
		return other.Data, err == nil
	}
	index, okIndex := read("zzz.profile")
	characters, okCharacters := read("zzz.characters")
	buddies, okBuddies := read("zzz.buddies")
	list, _ := characters["avatar_list"].([]any)
	if !okIndex || !okCharacters || !okBuddies || len(list) == 0 {
		return app.Image{}, false
	}
	resources := newRecordResources(context, commonArtwork, [][2]string{{"card-images-status", "resources/card/images/status.png"}})
	maps := readMaps(context)
	square := func(id, source, name string) string {
		resource, ok := context.FetchArtworkResource(id, source, name)
		if !ok {
			return ""
		}
		resources.List = append(resources.List, resource)
		return id
	}
	agents := []any{}
	for _, raw := range list {
		agent, _ := raw.(map[string]any)
		id := app.Text(agent["id"])
		agents = append(agents, map[string]any{"rarity": app.Text(agent["rarity"]), "element": maps.elementName(agent["element_type"], agent["sub_element_type"]),
			"icon": square("agent-"+id, "mys-zzz", "role_square_avatar/role_square_avatar_"+id+".png"), "rank": app.Int(agent["rank"]), "level": app.Text(agent["level"])})
	}
	bangboo := []any{}
	entries, _ := buddies["list"].([]any)
	for _, raw := range entries {
		buddy, _ := raw.(map[string]any)
		id := app.Text(buddy["id"])
		bangboo = append(bangboo, map[string]any{"rarity": app.Text(buddy["rarity"]), "star": app.Int(buddy["star"]), "level": app.Text(buddy["level"]),
			"icon": square("bangboo-"+id, "zzzerouid", "square_bangbo/bangboo_rectangle_avatar_"+id+".png")})
	}
	stats, _ := index["stats"].(map[string]any)
	player := playerCard(result.Role)
	if world := app.Text(stats["world_level_name"]); world != "" {
		player["region"] = world
	}
	return app.Image{Template: "card", Data: map[string]any{
		"player": player, "stats": []any{
			map[string]any{"value": app.Text(stats["active_days"]), "label": "活跃天数"},
			map[string]any{"value": app.Text(stats["avatar_num"]), "label": "获得代理人"},
			map[string]any{"value": app.Text(stats["buddy_num"]), "label": "获得邦布"},
			map[string]any{"value": app.Text(stats["cur_period_zone_layer_count"]), "label": "式舆防卫战"},
		},
		"agents": agents, "bangboo": bangboo, "bars": make([]int, 8),
	}, Resources: resources.List}, true
}
