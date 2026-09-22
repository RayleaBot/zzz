package images

import (
	"fmt"

	"github.com/RayleaBot/plugin-zzz/internal/app"
)

// talentIcon is the official image ZZZ-Plugin shows for the combat talents.
const talentIcon = "https://act-webstatic.mihoyo.com/darkmatter/nap/prod_gf_cn/item_icon_uc45id/3ae6457d965f876adb61a1bdfb2384b8.png"

// Withered Domain and Lost Void list their collections by position.
var (
	witheredCollections = []string{"鸣徽图鉴", "特殊区域记录", "哨站课题", "异域模块", "战术棱镜方案"}
	lostVoidCollections = []string{"探究勋证", "武备图鉴", "战术档案", "异域模块", "战术棱镜方案", "零号碎片", "空洞异闻"}
)

// HollowZero draws 枯萎苗圃 the way ZZZ-Plugin's hollowZero page does: the
// player card, license level and combat talents, the five collections, and,
// once the throne has been damaged, the best Slash-and-Burn run from the
// challenge record with its time, damage, agents and Bangboo. The frame image
// upstream's stylesheet names is not in its repository, so the page has none
// there either.
func HollowZero(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	data := result.Data
	if len(data) == 0 {
		return app.Image{}, false
	}
	resources := newRecordResources(context, commonArtwork)
	level, _ := data["abyss_level"].(map[string]any)
	talent, _ := data["abyss_talent"].(map[string]any)
	image := map[string]any{
		"player": playerCard(result.Role), "level": app.Text(level["cur_level"]) + " / " + app.Text(level["max_level"]), "level_icon": resources.official(level["icon"]),
		"talent": app.Text(talent["cur_talent"]) + " / " + app.Text(talent["max_talent"]), "talent_icon": resources.official(talentIcon),
		"collections": collections(data, witheredCollections),
	}
	if throne, _ := data["abyss_throne"].(map[string]any); app.Int(throne["max_damage"]) != 0 && context.Query != nil {
		challenge, err := context.Query("zzz.hollow_zero_detail", nil)
		if err != nil {
			return app.Image{}, false
		}
		if best, _ := challenge.Data["abyss_throne_max"].(map[string]any); best != nil {
			image["throne"] = throneRun(resources, best)
		}
	}
	return app.Image{Template: "hollow-zero", Data: image, Resources: resources.List}, true
}

// throneRun is the best Slash-and-Burn run: its time, the damage dealt to
// Nineveh, three agent slots with level and damage share, and the Bangboo.
func throneRun(resources *recordResources, best map[string]any) map[string]any {
	avatars, _ := best["avatar_list"].([]any)
	slots := []any{}
	for index := range 3 {
		if index >= len(avatars) {
			slots = append(slots, nil)
			continue
		}
		avatar, _ := avatars[index].(map[string]any)
		slots = append(slots, map[string]any{"rarity": app.Text(avatar["rarity"]), "rank": app.Int(avatar["rank"]), "icon": resources.official(avatar["role_square_url"]),
			"level": app.Text(avatar["level"]), "damage": app.Text(avatar["damage_rate"])})
	}
	var buddy any
	if buddies, _ := best["buddy_list"].([]any); len(buddies) > 0 {
		item, _ := buddies[0].(map[string]any)
		buddy = map[string]any{"rarity": app.Text(item["rarity"]), "icon": resources.official(item["bangboo_rectangle_url"]), "level": app.Text(item["level"])}
	}
	// Upstream formats the time only when there is one.
	time := ""
	if app.Int(best["time"]) != 0 {
		time = clockTime(best["time"])
	}
	return map[string]any{"time": time, "damage": app.Text(best["max_damage"]), "slots": slots, "buddy": buddy}
}

// LostVoid draws 迷失之地 the way ZZZ-Plugin's hollowZeroS2 page does: the
// player card, license level, bounty and exploration progress, the seven
// collections, and the hardest Matrix Operation and Task Force Investigation
// cleared with their clear counts and best times.
func LostVoid(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	data := result.Data
	if len(data) == 0 {
		return app.Image{}, false
	}
	resources := newRecordResources(context, commonArtwork)
	level, _ := data["abyss_level"].(map[string]any)
	duty, _ := data["abyss_duty"].(map[string]any)
	task, _ := data["abyss_task"].(map[string]any)
	best := func(key string) map[string]any {
		item, _ := data[key].(map[string]any)
		if item == nil {
			return map[string]any{}
		}
		return map[string]any{"name": app.Text(item["max_name"]), "heat": app.Text(item["heat_count"]), "count": app.Text(item["max_count"]), "time": clockTime(item["best_time"])}
	}
	return app.Image{Template: "lost-void", Data: map[string]any{
		"player": playerCard(result.Role), "level": app.Text(level["cur_level"]) + " / " + app.Text(level["max_level"]), "level_icon": resources.official(level["icon"]),
		"duty": app.Text(duty["cur_duty"]) + " / " + app.Text(duty["max_duty"]), "task": app.Text(task["cur_task"]) + " / " + app.Text(task["max_task"]),
		"collections": collections(data, lostVoidCollections), "matrix": best("abyss_max"), "investigation": best("abyss_task_force_investigation_max"),
	}, Resources: resources.List}, true
}

// collections pairs upstream's collection names with the official counts in
// the same order.
func collections(data map[string]any, names []string) []any {
	list, _ := data["abyss_collect"].([]any)
	out := []any{}
	for index, name := range names {
		item := map[string]any{}
		if index < len(list) {
			item, _ = list[index].(map[string]any)
		}
		out = append(out, map[string]any{"name": name, "value": app.Text(item["cur_collect"]) + " / " + app.Text(item["max_collect"])})
	}
	return out
}

// clockTime is upstream's formatTime: seconds as HH:MM:SS.
func clockTime(value any) string {
	seconds := app.Int(value)
	return fmt.Sprintf("%02d:%02d:%02d", seconds/3600, seconds%3600/60, seconds%60)
}
