package images

import "github.com/RayleaBot/plugin-zzz/internal/app"

// panelListArtwork is what panel/refresh adds.
var panelListArtwork = [][2]string{
	{"panel-images-CurseBG04", "resources/panel/images/CurseBG04.png"},
	{"panel-images-avatar_box", "resources/panel/images/avatar_box.png"},
	{"panel-images-bg1", "resources/panel/images/bg1.jpg"},
	{"panel-images-refresh_title", "resources/panel/images/refresh_title.png"},
}

// uidRegions are the servers ZZZ-Plugin's getGameRoles reads from a UID's
// leading digits; other UIDs are on 新艾利都.
var uidRegions = map[string]string{"10": "prod_gf_us", "15": "prod_gf_eu", "13": "prod_gf_jp", "17": "prod_gf_sg"}

// PanelList draws 面板列表 the way ZZZ-Plugin's panel/list does, and the
// reply to 更新面板 as its panel/refresh: the player card with the number of
// agents kept or updated, then every agent's square avatar, Mindscape, name
// and rarity, the updated ones marked.
func PanelList(context app.ImageContext, list app.PanelListImage) (app.Image, bool) {
	resources := newRecordResources(context, commonArtwork, panelListArtwork)
	agents := []any{}
	for _, saved := range list.Panels {
		panel := saved.Panel
		// Upstream shows the short name.
		name := app.Text(saved.Official["name_mi18n"])
		if name == "" {
			name = panel.Name
		}
		agent := map[string]any{"name": name, "rank": panel.Rank, "is_new": list.Updated[panel.ID]}
		if entry, ok := context.Catalog.Get(panel.ID); ok {
			agent["rarity"] = map[int]string{4: "S", 3: "A", 2: "B"}[entry.Rarity]
		}
		if rarity := app.Text(saved.Official["rarity"]); rarity != "" {
			agent["rarity"] = rarity
		}
		if avatar, ok := context.FetchArtworkResource("square-"+panel.ID, "mys-zzz", "role_square_avatar/role_square_avatar_"+panel.ID+".png"); ok {
			resources.List = append(resources.List, avatar)
			agent["square_icon"] = avatar.ID
		}
		agents = append(agents, agent)
	}
	region := "prod_gf_cn"
	if len(list.UID) > 8 {
		if code := uidRegions[list.UID[:len(list.UID)-8]]; code != "" {
			region = code
		}
	}
	nickname, level := list.Profiles.Nickname, list.Profiles.Level
	if nickname == "" {
		// Upstream's placeholder for a player it has no card for.
		nickname, level = "Fairy", 60
	}
	player := playerCard(app.Role{UID: list.UID, Nickname: nickname, Level: level, Region: region})
	data := map[string]any{"player": player, "refresh": list.Updated != nil, "count": len(agents), "new_count": len(list.Updated), "list": agents}
	return app.Image{Template: "panel-list", Data: data, Resources: resources.List}, true
}
