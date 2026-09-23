package images

import (
	"strconv"

	"github.com/RayleaBot/plugin-zzz/internal/app"
)

// buddyArtwork are the number font and pictures the converted Yunzai buddy
// page names.
var buddyArtwork = [][3]string{
	{"tttgbnumber", "yunzai-genshin", "resources/font/tttgbnumber.ttf"},
	{"other-S", "yunzai-genshin", "resources/ZZZero/img/other/S.png"},
	{"other-A", "yunzai-genshin", "resources/ZZZero/img/other/A.png"},
	{"other-fill", "yunzai-genshin", "resources/ZZZero/img/other/fill.png"},
}

// Buddies draws 邦布 the way the Yunzai 原神插件's ZZZero/html/buddy does:
// the UID, then each owned Bangboo with its rarity strip, star badge above
// one, level and name, on the narrower page for eight or fewer. Pictures are upstream's by name; a Bangboo newer than
// those takes ZZZ-Plugin's square portrait instead of a blank.
func Buddies(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	list, _ := result.Data["list"].([]any)
	if len(list) == 0 {
		return app.Image{}, false
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range buddyArtwork {
		resources.Artwork(item[0], item[1], item[2])
	}
	buddies := []any{}
	for index, raw := range list {
		buddy, _ := raw.(map[string]any)
		name, id := app.Text(buddy["name"]), app.Text(buddy["id"])
		image := resources.Artwork("buddy-"+strconv.Itoa(index), "yunzai-genshin", "resources/ZZZero/img/buddy/"+name+".png")
		if image == "" && id != "" {
			if resource, ok := context.FetchArtworkResource("buddy-"+strconv.Itoa(index), "zzzerouid", "square_bangbo/bangboo_rectangle_avatar_"+id+".png"); ok {
				resources.List = append(resources.List, resource)
				image = resource.ID
			}
		}
		// Upstream shows the star badge above one star.
		star := app.Int(buddy["star"])
		if star <= 1 {
			star = 0
		}
		buddies = append(buddies, map[string]any{"name": name, "rarity": app.Text(buddy["rarity"]), "level": app.Text(buddy["level"]), "star": star, "image": image})
	}
	return app.Image{Template: "buddy", Data: map[string]any{"uid": result.Role.UID, "list8": len(list) <= 8, "buddies": buddies}, Resources: resources.List}, true
}
