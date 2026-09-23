package images

import (
	"encoding/json"
	"maps"
	"slices"

	"github.com/RayleaBot/plugin-zzz/internal/app"
)

// Downloads lists the pictures the templates fetch on demand, for every
// agent, W-Engine, drive disc set and Bangboo in ZZZ-Plugin's maps, as its
// 下载全部资源 fetches each of them. Upstream's round portraits and its
// Nanoka data files have no template here that reads them; the Mindscape art
// the cinema page reads takes their place.
func Downloads(context app.ImageContext) []app.ArtworkGroup {
	tables := readMaps(context)
	role := app.ArtworkGroup{Label: "角色图", Source: "zzzerouid"}
	square := app.ArtworkGroup{Label: "角色头像图", Source: "mys-zzz"}
	general := app.ArtworkGroup{Label: "角色头像图(练度统计)", Source: "zzzerouid"}
	suits := app.ArtworkGroup{Label: "驱动盘套装图", Source: "zzzerouid"}
	weapons := app.ArtworkGroup{Label: "武器图", Source: "zzzerouid"}
	bangboo := app.ArtworkGroup{Label: "邦布图", Source: "zzzerouid"}
	cinema := app.ArtworkGroup{Label: "意象影画图", Source: "nanoka"}
	for _, id := range slices.Sorted(maps.Keys(tables.partners)) {
		if sprite := tables.partners[id].SpriteID; sprite != "" {
			role.Files = append(role.Files, "role/IconRole"+sprite+".png")
			general.Files = append(general.Files, "role_general/IconRoleGeneral"+sprite+".png")
		}
		square.Files = append(square.Files, "role_square_avatar/role_square_avatar_"+id+".png")
		cinema.Files = append(cinema.Files, "assets/zzz/Mindscape_"+id+"_3.webp")
	}
	for _, id := range slices.Sorted(maps.Keys(tables.suits)) {
		if sprite := tables.suits[id].SpriteFile; sprite != "" {
			suits.Files = append(suits.Files, "suit/"+sprite+".png")
		}
	}
	for _, id := range slices.Sorted(maps.Keys(tables.weapons)) {
		if code := tables.weapons[id].CodeName; code != "" {
			weapons.Files = append(weapons.Files, "weapon/"+code+"_High.png")
		}
	}
	if context.Artwork != nil {
		var buddies map[string]json.RawMessage
		if raw, err := context.Artwork.Open("zzz-plugin", "resources/map/BangbooId2Data.json"); err == nil && json.Unmarshal(raw, &buddies) == nil {
			for _, id := range slices.Sorted(maps.Keys(buddies)) {
				bangboo.Files = append(bangboo.Files, "square_bangbo/bangboo_rectangle_avatar_"+id+".png")
			}
		}
	}
	return []app.ArtworkGroup{role, square, general, suits, weapons, bangboo, cinema}
}
