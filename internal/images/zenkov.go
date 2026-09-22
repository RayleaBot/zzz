package images

import (
	"fmt"
	"math"
	"strconv"

	"github.com/RayleaBot/plugin-zzz/internal/app"
)

// zenkovArtwork maps the images the zenkov pages name to ZZZ-Plugin's paths.
var zenkovArtwork = [][2]string{
	{"zenkov-images-icon-fund", "resources/zenkov/images/icon-fund.png"},
	{"zenkov-images-icon-season-coins", "resources/zenkov/images/icon-season-coins.png"},
	{"zenkov-images-icon-gain-bg", "resources/zenkov/images/icon-gain-bg.png"},
	{"zenkov-images-icon-gain-bg-purple", "resources/zenkov/images/icon-gain-bg-purple.png"},
	{"zenkov-images-icon-gain-bg-yellow", "resources/zenkov/images/icon-gain-bg-yellow.png"},
}

// zenkovDifficulties are upstream's names for the record difficulties.
var zenkovDifficulties = map[string]string{"Hell": "高危", "Hard": "困难"}

// Zenkov draws 迷宫诡域 the way ZZZ-Plugin's zenkov page does: the player
// card, the season level, commissions and coin cap, the weekly commissions
// with the time left and haul totals, each map's High-Risk extraction rate
// and best haul with the season's best rank, and the medal wall and
// collection.
func Zenkov(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	data := result.Data
	if len(data) == 0 {
		return app.Image{}, false
	}
	resources := newRecordResources(context, commonArtwork, zenkovArtwork)
	image := zenkovMaps(data, "迷宫诡域")
	image["player"] = playerCard(result.Role)
	season, _ := data["season_data"].(map[string]any)
	unlocked, _ := data["season_unlock"].(bool)
	// Upstream shows the first season while the official data has none.
	summary := map[string]any{"id": "1", "level": "1", "refresh": "0天"}
	if season != nil {
		quest, _ := season["season_quest"].(map[string]any)
		coin, _ := season["season_coin"].(map[string]any)
		summary = map[string]any{"id": app.Text(season["cur_season_id"]), "level": app.Text(season["season_level"]), "refresh": zenkovDays(season["refresh_time"]),
			"stage": app.Text(season["season_stage"]), "quest": zenkovCount(quest["cur_quest"]), "quest_max": zenkovCount(quest["max_quest"]),
			"coin": zenkovCount(coin["cur_coin"]) + "/" + zenkovCount(coin["max_coin"])}
		summary["unlocked"] = unlocked
	}
	duty, _ := data["abyss_duty"].(map[string]any)
	image["season"] = summary
	image["weekly"] = map[string]any{"refresh": zenkovDaysHours(data["refresh_time"]), "duty": zenkovCount(duty["cur_duty"]) + " / " + zenkovCount(duty["max_duty"]),
		"value": orText(data["collect_total_value"], "0"), "big_red": orText(data["big_red_num"], "-"), "millions": orText(data["millions_evacuations"], "-")}
	if collection, _ := data["collection_data"].(map[string]any); collection != nil {
		entry := map[string]any{}
		if medals, _ := collection["medal_data"].(map[string]any); medals != nil {
			list, _ := medals["list"].([]any)
			items := []any{}
			for _, raw := range list {
				item, _ := raw.(map[string]any)
				unlocked, _ := item["unlock"].(bool)
				items = append(items, map[string]any{"name": app.Text(item["name"]), "unlocked": unlocked, "icon": resources.official(item["medal_icon"])})
			}
			entry["medals"] = map[string]any{"progress": app.Text(medals["cur"]) + " / " + app.Text(medals["total"]), "list": items}
		}
		if goods, _ := collection["goods_data"].(map[string]any); goods != nil {
			list, _ := goods["list"].([]any)
			items := []any{}
			for _, raw := range list {
				item, _ := raw.(map[string]any)
				unlocked, _ := item["unlock"].(bool)
				number := ""
				if app.Int(item["number"]) > 0 {
					number = app.Text(item["number"])
				}
				items = append(items, map[string]any{"name": app.Text(item["name"]), "unlocked": unlocked, "icon": resources.official(item["goods_icon"]), "number": number})
			}
			entry["goods"] = map[string]any{"progress": app.Text(goods["cur"]) + " / " + app.Text(goods["total"]), "list": items}
		}
		image["collection"] = entry
	}
	return app.Image{Template: "zenkov", Data: image, Resources: resources.List}, true
}

// ZenkovDetail draws 迷宫诡域战绩 the way ZZZ-Plugin's zenkov detail page
// does: the maps and best rank, then each run with its map, start time,
// difficulty, result, duration, haul value, agents and the items found.
func ZenkovDetail(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	data := result.Data
	if len(data) == 0 {
		return app.Image{}, false
	}
	resources := newRecordResources(context, commonArtwork, zenkovArtwork)
	image := zenkovMaps(data, "迷宫诡域战绩")
	image["player"] = playerCard(result.Role)
	records := []any{}
	list, _ := data["record_list"].([]any)
	for _, raw := range list {
		record, _ := raw.(map[string]any)
		difficulty := app.Text(record["difficult"])
		name := zenkovDifficulties[difficulty]
		if name == "" {
			name = orText(record["difficult"], "普通")
		}
		success, _ := record["is_success"].(bool)
		agents := []any{}
		avatars, _ := record["avatar_list"].([]any)
		for _, rawAvatar := range avatars {
			avatar, _ := rawAvatar.(map[string]any)
			agents = append(agents, map[string]any{"icon": resources.official(avatar["role_square_url"]), "rank": app.Int(avatar["rank"])})
		}
		items := []any{}
		itemList, _ := record["item_list"].([]any)
		for _, rawItem := range itemList {
			item, _ := rawItem.(map[string]any)
			class := "no-bg"
			if rarity := app.Int(item["rarity"]); rarity >= 3 && rarity <= 5 {
				class = "rarity-" + strconv.Itoa(rarity)
			}
			items = append(items, map[string]any{"class": class, "title": app.Text(item["name"]) + " (" + app.Text(item["price"]) + ")", "icon": resources.official(item["icon_url"])})
		}
		records = append(records, map[string]any{"map": app.Text(record["map_name"]), "start": zenkovDateTime(record["start_time"]),
			"difficulty": name, "hell": difficulty == "Hell", "success": success, "duration": zenkovDuration(record["challenge_time"]),
			"value": zenkovThousands(record["material_total_value"]), "agents": agents, "items": items})
	}
	image["detail"], image["records"] = true, records
	return app.Image{Template: "zenkov", Data: image, Resources: resources.List}, true
}

// zenkovMaps is what both pages share: the title, each map's High-Risk
// extraction rate and best haul, and the season's best rank, as a top
// percentage over rank badge or an absolute place.
func zenkovMaps(data map[string]any, title string) map[string]any {
	maps := []any{}
	list, _ := data["map_list"].([]any)
	for _, raw := range list {
		item, _ := raw.(map[string]any)
		unlocked, _ := item["hell_unlock"].(bool)
		leave := "0%"
		if value, ok := item["leave_percent"]; ok && value != nil {
			if number, err := strconv.ParseFloat(app.Text(value), 64); err == nil {
				percent := number / 100
				digits := 2
				if percent == math.Trunc(percent) {
					digits = 0
				}
				leave = strconv.FormatFloat(percent, 'f', digits, 64) + "%"
			} else {
				leave = app.Text(value)
			}
		}
		maps = append(maps, map[string]any{"name": app.Text(item["map_name"]), "unlocked": unlocked, "leave": leave, "price": app.Text(item["max_price"])})
	}
	rank := map[string]any{}
	if number, err := strconv.ParseFloat(app.Text(data["max_rank"]), 64); err == nil && number != 0 {
		if percent, _ := data["is_show_percent"].(bool); percent {
			rank = map[string]any{"percent": rankText(int(number)), "background": rankBackground(int(number))}
		} else {
			rank = map[string]any{"top": "TOP " + strconv.Itoa(int(math.Floor(number)))}
		}
	}
	return map[string]any{"title": title, "maps": maps, "rank": rank}
}

// zenkovCount is a count upstream shows as 0 when missing.
func zenkovCount(value any) string { return orText(value, "0") }

// orText is the official value as text, or fallback when it is missing.
func orText(value any, fallback string) string {
	if text := app.Text(value); text != "" {
		return text
	}
	return fallback
}

// zenkovDaysHours is upstream's formatTimeDaysHours: seconds left as XX天YY时.
func zenkovDaysHours(value any) string {
	seconds := app.Int(value)
	if seconds <= 0 {
		return "00天00时"
	}
	return fmt.Sprintf("%02d天%02d时", seconds/86400, seconds%86400/3600)
}

// zenkovDays is upstream's formatTimeDays: whole days left.
func zenkovDays(value any) string {
	seconds := app.Int(value)
	if seconds <= 0 {
		return "0天"
	}
	return strconv.Itoa(seconds/86400) + "天"
}

// zenkovDateTime is upstream's formatDateTimeObj: YYYY-MM-DD HH:mm:ss, or -.
func zenkovDateTime(value any) string {
	fields, _ := value.(map[string]any)
	if fields == nil {
		return "-"
	}
	return fmt.Sprintf("%s-%02d-%02d %02d:%02d:%02d", app.Text(fields["year"]), app.Int(fields["month"]), app.Int(fields["day"]),
		app.Int(fields["hour"]), app.Int(fields["minute"]), app.Int(fields["second"]))
}

// zenkovDuration is upstream's formatDurationObj: MM:SS, with hours only when
// there are any.
func zenkovDuration(value any) string {
	fields, _ := value.(map[string]any)
	if fields == nil {
		return "00:00"
	}
	hours := ""
	if hour := app.Int(fields["hour"]); hour != 0 {
		hours = fmt.Sprintf("%02d:", hour)
	}
	return fmt.Sprintf("%s%02d:%02d", hours, app.Int(fields["minute"]), app.Int(fields["second"]))
}

// zenkovThousands is upstream's formatNumberCommas: the haul value with
// thousands separators.
func zenkovThousands(value any) string {
	text := app.Text(value)
	if text == "" {
		return "0"
	}
	number, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return text
	}
	digits := strconv.FormatInt(number, 10)
	sign := ""
	if number < 0 {
		sign, digits = "-", digits[1:]
	}
	for index := len(digits) - 3; index > 0; index -= 3 {
		digits = digits[:index] + "," + digits[index:]
	}
	return sign + digits
}
