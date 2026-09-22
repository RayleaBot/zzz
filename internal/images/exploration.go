package images

import "github.com/RayleaBot/plugin-zzz/internal/app"

// Exploration draws 区域收集 the way ZZZ-Plugin's explorationDetail page
// does: the player card, then each area's completion with its sub-areas, each
// with its icon, completion and at most four collection counts. An answer
// without areas shows upstream's empty notice.
func Exploration(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	if len(result.Data) == 0 {
		return app.Image{}, false
	}
	resources := newRecordResources(context, commonArtwork)
	areas := []any{}
	list, _ := result.Data["area_collections"].([]any)
	for _, raw := range list {
		area, _ := raw.(map[string]any)
		maps := []any{}
		mapList, _ := area["map_collections"].([]any)
		for _, rawMap := range mapList {
			item, _ := rawMap.(map[string]any)
			collections := []any{}
			collectionList, _ := item["collections"].([]any)
			for index, rawCollection := range collectionList {
				if index == 4 {
					break
				}
				collection, _ := rawCollection.(map[string]any)
				collections = append(collections, map[string]any{"name": app.Text(collection["name"]), "icon": resources.official(collection["icon"]),
					"count": app.Text(collection["num"]) + "/" + app.Text(collection["total"])})
			}
			maps = append(maps, map[string]any{"name": app.Text(item["name"]), "icon": resources.official(item["icon"]),
				"progress": app.Text(item["collection_progress"]), "collections": collections})
		}
		areas = append(areas, map[string]any{"name": app.Text(area["name"]), "progress": app.Text(area["collection_progress"]), "maps": maps})
	}
	return app.Image{Template: "exploration", Data: map[string]any{"player": playerCard(result.Role), "areas": areas}, Resources: resources.List}, true
}
