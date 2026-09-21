package images

import (
	"fmt"
	"strings"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

// abyssArtwork is what abyss adds.
var abyssArtwork = [][2]string{
	{"abyss-images-BgFrame01", "resources/abyss/images/BgFrame01.png"},
	{"abyss-images-bg", "resources/abyss/images/bg.png"},
	{"abyss-images-boss_bg_4", "resources/abyss/images/boss_bg_4.png"},
}

// Abyss draws Shiyu Defense the way ZZZ-Plugin's abyss page does: the player
// card and period, the fifth frontier with its score, rank, rating and clear
// times, each of its rooms with the monster, score, buff and team, then
// frontiers four to one with their ratings and teams.
func Abyss(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	// Upstream answers in text for an older version or an empty period.
	data, _ := result.Data["hadal_info_v2"].(map[string]any)
	if gamekit.Text(result.Data["hadal_ver"]) != "v2" || data == nil {
		return gamekit.Image{}, false
	}
	names := []string{"first", "second", "third", "fourth", "fitfh"}
	empty := true
	for _, name := range names {
		if data[name+"_layer_detail"] != nil {
			empty = false
		}
	}
	if empty {
		return gamekit.Image{}, false
	}
	resources := newRecordResources(context, commonArtwork, abyssArtwork)
	const clock = "2006-01-02 15:04:05"

	var fifth any
	brief, _ := data["brief"].(map[string]any)
	detail, _ := data["fitfh_layer_detail"].(map[string]any)
	if list, _ := detail["layer_challenge_info_list"].([]any); len(list) > 0 && brief != nil {
		rooms := []any{}
		for index, raw := range list {
			item, _ := raw.(map[string]any)
			buffer, _ := item["buffer"].(map[string]any)
			score := gamekit.Int(item["score"])
			rooms = append(rooms, map[string]any{"no": index + 1, "rating": gamekit.Text(item["rating"]), "monster": resources.official(item["monster_pic"]),
				"score": score, "max_score": score == 50000, "time": recordTime(item["challenge_time"], clock),
				"buff": gamekit.Text(buffer["title"]), "team": resources.team(item)})
		}
		score, percent := gamekit.Int(brief["score"]), gamekit.Int(brief["rank_percent"])
		battle := ""
		if seconds := gamekit.Int(brief["battle_time"]); seconds > 0 {
			battle = fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
		}
		fifth = map[string]any{"score": score, "max_score": score == 150000, "rank_bg": rankBackground(percent), "rank": rankText(percent),
			"rating": strings.ReplaceAll(gamekit.Text(brief["rating"]), "+", "P"), "time": recordTime(brief["challenge_time"], clock),
			"battle_time": battle, "rooms": rooms}
	}
	lower := []any{}
	for index := 3; index >= 0; index-- {
		detail, _ := data[names[index]+"_layer_detail"].(map[string]any)
		list, _ := detail["layer_challenge_info_list"].([]any)
		if detail == nil || list == nil {
			continue
		}
		teams := []any{}
		for number, raw := range list {
			item, _ := raw.(map[string]any)
			teams = append(teams, map[string]any{"no": number + 1, "team": resources.team(item)})
		}
		lower = append(lower, map[string]any{"name": "剧变节点第" + []string{"一", "二", "三", "四"}[index] + "防线",
			"rating": gamekit.Text(detail["rating"]), "time": recordTime(detail["challenge_time"], clock), "teams": teams})
	}
	return gamekit.Image{Template: "abyss", Data: map[string]any{
		"player": playerCard(result.Role), "begin": recordTime(data["hadal_begin_time"], clock), "end": recordTime(data["hadal_end_time"], clock),
		"fifth": fifth, "lower": lower, "rank_command": rankCommand(context, "防卫战"),
	}, Resources: resources.List}, true
}
