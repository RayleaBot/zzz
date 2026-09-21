package images

import gamekit "github.com/RayleaBot/game-plugin-kit"

// voidFrontArtwork is what voidFrontBattle adds.
var voidFrontArtwork = [][2]string{
	{"voidFrontBattle-images-BgFrame01", "resources/voidFrontBattle/images/BgFrame01.png"},
	{"voidFrontBattle-images-bg", "resources/voidFrontBattle/images/bg.png"},
	{"voidFrontBattle-images-m-deduce-detail-block-mask", "resources/voidFrontBattle/images/m-deduce-detail-block-mask.png"},
}

// VoidFront draws Threshold Simulation the way ZZZ-Plugin's voidFrontBattle
// page does: the player card and period, the total score with its rank and
// ending, the final boss stage with its score, ratio, clear time, team, rating
// and sub-stages, then every earlier stage the same way.
func VoidFront(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	detail, _ := result.Data["void_front_battle_detail"].(map[string]any)
	// Upstream answers in text while the period has no data.
	if detail == nil {
		return gamekit.Image{}, false
	}
	resources := newRecordResources(context, commonArtwork, voidFrontArtwork)
	const clock = "2006-01-02 15:04:05"
	score := func(item map[string]any) map[string]any {
		value, maximum := gamekit.Int(item["score"]), gamekit.Int(item["max_score"])
		return map[string]any{"value": value, "max": maximum, "full": value == maximum}
	}
	buff := func(item map[string]any) map[string]any {
		buffer, _ := item["buffer"].(map[string]any)
		return map[string]any{"icon": resources.official(buffer["icon"]), "name": gamekit.Text(buffer["name"])}
	}
	stage := func(item map[string]any) map[string]any {
		subs := []any{}
		records, _ := item["sub_challenge_record"].([]any)
		for _, raw := range records {
			sub, _ := raw.(map[string]any)
			subs = append(subs, map[string]any{"rating": gamekit.Text(sub["star"]), "name": gamekit.Text(sub["name"]), "buff": buff(sub), "team": resources.team(sub)})
		}
		return map[string]any{"name": gamekit.Text(item["name"]), "buff": buff(item), "score": score(item), "ratio": gamekit.Text(item["score_ratio"]),
			"time": recordTime(item["challenge_time"], clock), "team": resources.team(item), "rating": gamekit.Text(item["star"]), "subs": subs}
	}

	brief, _ := detail["void_front_battle_abstract_info_brief"].(map[string]any)
	end := recordTime(brief["end_time"], clock)
	if end == "" {
		end = "超过42天"
	}
	var total, boss any
	bossChallenge, _ := detail["boss_challenge_record"].(map[string]any)
	if brief != nil && bossChallenge != nil {
		percent := gamekit.Int(brief["rank_percent"])
		totalScore := score(map[string]any{"score": brief["total_score"], "max_score": brief["max_score"]})
		total = map[string]any{"score": totalScore, "rank_bg": rankBackground(percent), "rank": rankText(percent),
			"ending": gamekit.Text(brief["ending_record_name"]), "ending_bg": resources.official(brief["ending_record_bg_pic"])}
		main, _ := bossChallenge["main_challenge_record"].(map[string]any)
		info, _ := bossChallenge["boss_info"].(map[string]any)
		record := stage(main)
		record["name"], record["icon"] = gamekit.Text(info["name"]), resources.official(info["icon"])
		boss = record
	}
	stages := []any{}
	list, _ := detail["main_challenge_record_list"].([]any)
	for _, raw := range list {
		item, _ := raw.(map[string]any)
		stages = append(stages, stage(item))
	}
	// Upstream notes the group ranking only for the current period.
	var note map[string]any
	if gamekit.Int(context.Input["schedule_type"]) != 2 {
		note = rankNote(context)
	}
	return gamekit.Image{Template: "void-front", Data: map[string]any{
		"player": playerCard(result.Role), "begin": recordTime(brief["start_time"], clock), "end": end,
		"total": total, "boss": boss, "stages": stages, "rank_note": note,
	}, Resources: resources.List}, true
}
