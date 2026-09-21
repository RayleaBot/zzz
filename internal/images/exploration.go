package images

import gamekit "github.com/RayleaBot/game-plugin-kit"

// Exploration draws 区域收集 the way ZZZ-Plugin's explorationDetail page
// does: the player card, then each area's completion with its sub-areas, each
// with its icon, completion and at most four collection counts. An answer
// without areas shows upstream's empty notice.
func Exploration(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	if len(result.Data) == 0 {
		return gamekit.Image{}, false
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
				collections = append(collections, map[string]any{"name": gamekit.Text(collection["name"]), "icon": resources.official(collection["icon"]),
					"count": gamekit.Text(collection["num"]) + "/" + gamekit.Text(collection["total"])})
			}
			maps = append(maps, map[string]any{"name": gamekit.Text(item["name"]), "icon": resources.official(item["icon"]),
				"progress": gamekit.Text(item["collection_progress"]), "collections": collections})
		}
		areas = append(areas, map[string]any{"name": gamekit.Text(area["name"]), "progress": gamekit.Text(area["collection_progress"]), "maps": maps})
	}
	return gamekit.Image{Template: "exploration", Data: map[string]any{"player": playerCard(result.Role), "areas": areas}, Resources: resources.List}, true
}
