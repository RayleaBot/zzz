package images

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/RayleaBot/plugin-zzz/internal/app"
)

// rankPage is one of ZZZ-Plugin's rank pages: what it keeps of a record and
// its sort keys (larger first), a line for text replies, and its row.
type rankPage struct {
	artwork [][2]string
	score   func(now time.Time, data map[string]any) (keys []float64, summary string, ok bool)
	row     func(resources *recordResources, data map[string]any) map[string]any
}

// rankRatings orders Shiyu Defense ratings as upstream does.
var rankRatings = map[string]float64{"S+": 5, "S": 4, "A": 3, "B": 2, "C": 1}

var rankPages = map[string]rankPage{
	"abyss": {artwork: [][2]string{{"rank-abyss-images-BgFrame01", "resources/rank/abyss/images/BgFrame01.png"}}, score: abyssRankScore, row: abyssRankRow},
	"deadly": {artwork: [][2]string{{"rank-deadly-images-BgFrame01", "resources/rank/deadly/images/BgFrame01.png"}, {"rank-deadly-images-PetSelectBG", "resources/rank/deadly/images/PetSelectBG.png"},
		{"rank-deadly-images-star-icon-dark", "resources/rank/deadly/images/star-icon-dark.png"}, {"rank-deadly-images-star-icon-light", "resources/rank/deadly/images/star-icon-light.png"}},
		score: deadlyRankScore, row: deadlyRankRow},
	"deadly-hard": {artwork: [][2]string{{"rank-deadlyHard-images-BgFrame01", "resources/rank/deadlyHard/images/BgFrame01.png"}, {"rank-deadlyHard-images-PetSelectBG", "resources/rank/deadlyHard/images/PetSelectBG.png"},
		{"rank-deadlyHard-images-hard-module-inner-bg-m-83b7d1a1", "resources/rank/deadlyHard/images/hard-module-inner-bg-m.83b7d1a1.png"},
		{"rank-deadlyHard-images-star-icon-dark", "resources/rank/deadlyHard/images/star-icon-dark.png"}, {"rank-deadlyHard-images-star-icon-hard-13c9ee51", "resources/rank/deadlyHard/images/star-icon-hard.13c9ee51.png"},
		{"rank-deadlyHard-images-star-icon-light", "resources/rank/deadlyHard/images/star-icon-light.png"}},
		score: deadlyHardRankScore, row: deadlyHardRankRow},
	"holo-boss": {artwork: [][2]string{{"rank-holoBoss-images-BgFrame01", "resources/rank/holoBoss/images/BgFrame01.png"}, {"rank-holoBoss-images-PetSelectBG", "resources/rank/holoBoss/images/PetSelectBG.png"},
		{"rank-holoBoss-images-star-gray-53e3ee51", "resources/rank/holoBoss/images/star-gray.53e3ee51.png"}, {"rank-holoBoss-images-star-light-d591d065", "resources/rank/holoBoss/images/star-light.d591d065.png"}},
		score: holoRankScore, row: holoRankRow},
	"void-front": {artwork: [][2]string{{"rank-voidFrontBattle-images-BgFrame01", "resources/rank/voidFrontBattle/images/BgFrame01.png"}, {"rank-voidFrontBattle-images-PetSelectBG", "resources/rank/voidFrontBattle/images/PetSelectBG.png"}},
		score: voidRankScore, row: voidRankRow},
	"tower-s1": towerRankPage("s1"),
	"tower-s2": towerRankPage("s2"),
	"tower-s3": towerRankPage("s3"),
	"tower-s4": towerRankPage("s4"),
}

// QueryRank draws a group ranking the way ZZZ-Plugin's rank pages do: the
// records of the current period only, ordered by the page's rules, the first
// fifteen, each with the member's avatar and the player card beside the
// result upstream shows for that mode.
func QueryRank(context app.ImageContext, rank app.QueryRankImage) (app.Image, []app.QueryRankRecord, bool) {
	page, ok := rankPages[rank.Type.Page]
	if !ok {
		return app.Image{}, nil, false
	}
	type scored struct {
		record app.QueryRankRecord
		keys   []float64
	}
	list := []scored{}
	for _, record := range rank.Records {
		if keys, summary, ok := page.score(context.Now, record.Data); ok {
			record.Summary = summary
			list = append(list, scored{record, keys})
		}
	}
	if len(list) == 0 {
		return app.Image{}, nil, false
	}
	slices.SortStableFunc(list, func(a, b scored) int {
		for index := range a.keys {
			if order := cmp.Compare(b.keys[index], a.keys[index]); order != 0 {
				return order
			}
		}
		return 0
	})
	list = list[:min(15, len(list))]
	resources := newRecordResources(context, commonArtwork, page.artwork)
	rows, ranked := []any{}, []app.QueryRankRecord{}
	for index, item := range list {
		row := page.row(resources, item.record.Data)
		player := playerCard(item.record.Role)
		if item.record.Avatar != "" {
			player["avatar"] = resources.Remote("avatar-"+strconv.Itoa(index), item.record.Avatar)
		}
		row["player"] = player
		rows = append(rows, row)
		ranked = append(ranked, item.record)
	}
	data := map[string]any{"rows": rows, "prefix": context.Game.Prefix, "bars": make([]int, 8)}
	return app.Image{Template: "rank-" + rank.Type.Page, Data: data, Resources: resources.List}, ranked, true
}

// rankTime reads the official {year, month, …} time of the China servers.
func rankTime(value any) (time.Time, bool) {
	fields, _ := value.(map[string]any)
	if app.Int(fields["year"]) == 0 {
		return time.Time{}, false
	}
	return time.Date(app.Int(fields["year"]), time.Month(app.Int(fields["month"])), app.Int(fields["day"]),
		app.Int(fields["hour"]), app.Int(fields["minute"]), app.Int(fields["second"]), 0, time.FixedZone("UTC+8", 8*3600)), true
}

// rankInPeriod is whether now falls in the record's start and end time.
func rankInPeriod(now time.Time, data map[string]any) bool {
	start, ok := rankTime(data["start_time"])
	end, ok2 := rankTime(data["end_time"])
	return ok && ok2 && !now.Before(start) && !now.After(end)
}

// rankLatest is the latest challenge time among the records, or now when none
// has one, as upstream breaks ties by it.
func rankLatest(now time.Time, list []any, field string) float64 {
	latest := int64(0)
	for _, raw := range list {
		item, _ := raw.(map[string]any)
		if at, ok := rankTime(item[field]); ok {
			latest = max(latest, at.Unix())
		}
	}
	if latest == 0 {
		latest = now.Unix()
	}
	return float64(latest)
}

func abyssRankScore(now time.Time, data map[string]any) ([]float64, string, bool) {
	info, _ := data["hadal_info_v2"].(map[string]any)
	if app.Text(data["hadal_ver"]) != "v2" || info == nil || now.Unix() < int64(app.Int(info["begin_time"])) || now.Unix() > int64(app.Int(info["end_time"])) {
		return nil, "", false
	}
	brief, _ := info["brief"].(map[string]any)
	detail, _ := info["fitfh_layer_detail"].(map[string]any)
	layers, _ := detail["layer_challenge_info_list"].([]any)
	latest := int64(0)
	for _, raw := range layers {
		latest = max(latest, int64(app.Int(raw.(map[string]any)["challenge_time"])))
	}
	if latest == 0 {
		latest = now.Unix()
	}
	rating := app.Text(brief["rating"])
	if rating == "" {
		rating = "C"
	}
	score := app.Int(brief["score"])
	return []float64{float64(score), rankRatings[rating], -float64(latest)}, fmt.Sprintf("%d · %s", score, rating), true
}

func abyssRankRow(resources *recordResources, data map[string]any) map[string]any {
	info, _ := data["hadal_info_v2"].(map[string]any)
	brief, _ := info["brief"].(map[string]any)
	detail, _ := info["fitfh_layer_detail"].(map[string]any)
	list, _ := detail["layer_challenge_info_list"].([]any)
	layers := []any{}
	for index := range 3 {
		if index >= len(list) {
			layers = append(layers, nil)
			continue
		}
		layer, _ := list[index].(map[string]any)
		buffer, _ := layer["buffer"].(map[string]any)
		score := app.Int(layer["score"])
		layers = append(layers, map[string]any{"name": app.Text(buffer["title"]), "score": score, "max": score == 50000,
			"rating": strings.ReplaceAll(app.Text(layer["rating"]), "+", "P"), "team": bangbooTeam(resources, layer)})
	}
	score := app.Int(brief["score"])
	rating := app.Text(brief["rating"])
	if rating == "" {
		rating = "C"
	}
	return map[string]any{"score": score, "max": score == 150000, "rating": strings.ReplaceAll(rating, "+", "P"), "layers": layers}
}

// bangbooTeam is a team with the Bangboo slot upstream's rank pages keep even
// when empty.
func bangbooTeam(resources *recordResources, item map[string]any) map[string]any {
	team := resources.team(item)
	team["bangboo"] = true
	return team
}

func deadlyRankScore(now time.Time, data map[string]any) ([]float64, string, bool) {
	if !rankInPeriod(now, data) || data["has_data"] != true {
		return nil, "", false
	}
	list, _ := data["list"].([]any)
	stars, score := app.Int(data["total_star"]), app.Int(data["total_score"])
	return []float64{float64(stars), float64(score), -rankLatest(now, list, "challenge_time")}, fmt.Sprintf("%d 星 · %d", stars, score), true
}

// deadlyTeam is a Deadly Assault boss team: its buff, boss, score, stars
// and team.
func deadlyTeam(resources *recordResources, item map[string]any) map[string]any {
	buffers, _ := item["buffer"].([]any)
	bosses, _ := item["boss"].([]any)
	var buffer, boss map[string]any
	if len(buffers) > 0 {
		buffer, _ = buffers[0].(map[string]any)
	}
	if len(bosses) > 0 {
		boss, _ = bosses[0].(map[string]any)
	}
	star, total := app.Int(item["star"]), app.Int(item["total_star"])
	stars := []any{}
	for index := range max(total, star) {
		stars = append(stars, index < star)
	}
	team := map[string]any{"name": app.Text(boss["name"]), "score": app.Int(item["score"]), "stars": stars, "team": bangbooTeam(resources, item)}
	if icon := app.Text(buffer["icon"]); icon != "" {
		team["pop"] = resources.official(icon)
	}
	return team
}

func deadlyRankRow(resources *recordResources, data map[string]any) map[string]any {
	list, _ := data["list"].([]any)
	teams := []any{}
	for index := range 3 {
		if index >= len(list) {
			teams = append(teams, nil)
			continue
		}
		item, _ := list[index].(map[string]any)
		teams = append(teams, deadlyTeam(resources, item))
	}
	return map[string]any{"score": app.Int(data["total_score"]), "stars": app.Int(data["total_star"]), "teams": teams}
}

func deadlyHardRankScore(now time.Time, data map[string]any) ([]float64, string, bool) {
	hard, _ := data["hard_list"].([]any)
	if !rankInPeriod(now, data) || data["has_data"] != true || data["has_hard"] != true || len(hard) == 0 {
		return nil, "", false
	}
	score := 0
	for _, raw := range hard {
		score += app.Int(raw.(map[string]any)["score"])
	}
	return []float64{float64(score), -rankLatest(now, hard, "challenge_time")}, strconv.Itoa(score), true
}

func deadlyHardRankRow(resources *recordResources, data map[string]any) map[string]any {
	hard, _ := data["hard_list"].([]any)
	item, _ := hard[0].(map[string]any)
	return map[string]any{"team": deadlyTeam(resources, item)}
}

func holoRankScore(now time.Time, data map[string]any) ([]float64, string, bool) {
	list, ok := data["list"].([]any)
	if !rankInPeriod(now, data) || data["unlock"] != true || !ok {
		return nil, "", false
	}
	stars, seconds, flawless := holoTotals(list)
	// Upstream's records carry no update time, so every record ties on it.
	return []float64{float64(stars), -float64(seconds), float64(flawless)}, fmt.Sprintf("%d 星 · %s · 无伤 %d", stars, holoClock(seconds), flawless), true
}

func holoTotals(list []any) (stars, seconds, flawless int) {
	for _, raw := range list {
		item, _ := raw.(map[string]any)
		spent, _ := item["challenge_time"].(map[string]any)
		boss, _ := item["boss"].(map[string]any)
		medal, _ := boss["medal"].(map[string]any)
		stars += app.Int(item["star"])
		seconds += app.Int(spent["minute"])*60 + app.Int(spent["second"])
		if medal["is_no_injured"] == true {
			flawless++
		}
	}
	return
}

func holoClock(seconds int) string { return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60) }

func holoRankRow(resources *recordResources, data map[string]any) map[string]any {
	list, _ := data["list"].([]any)
	stars, seconds, flawless := holoTotals(list)
	teams := []any{}
	for index := range 3 {
		if index >= len(list) {
			teams = append(teams, nil)
			continue
		}
		item, _ := list[index].(map[string]any)
		boss, _ := item["boss"].(map[string]any)
		medal, _ := boss["medal"].(map[string]any)
		spent, _ := item["challenge_time"].(map[string]any)
		star := app.Int(item["star"])
		row := []any{}
		for slot := range 4 {
			row = append(row, slot < star)
		}
		team := map[string]any{"name": app.Text(boss["name"]), "flawless": medal["is_no_injured"] == true, "stars": row,
			"time": holoClock(app.Int(spent["minute"])*60 + app.Int(spent["second"])), "team": resources.team(item)}
		if icon := app.Text(medal["medal_icon"]); icon != "" {
			team["pop"] = resources.official(icon)
		}
		teams = append(teams, team)
	}
	return map[string]any{"time": holoClock(seconds), "stars": stars, "flawless": flawless, "teams": teams}
}

func voidRankScore(now time.Time, data map[string]any) ([]float64, string, bool) {
	brief, _ := data["void_front_battle_abstract_info_brief"].(map[string]any)
	if brief == nil || now.Unix() > int64(app.Int(brief["end_ts"])) || brief["has_ending_record"] != true {
		return nil, "", false
	}
	boss, _ := data["boss_challenge_record"].(map[string]any)
	main, _ := boss["main_challenge_record"].(map[string]any)
	list, _ := data["main_challenge_record_list"].([]any)
	latest := rankLatest(now, append([]any{main}, list...), "challenge_time")
	score := app.Int(brief["total_score"])
	return []float64{float64(score), -latest}, strconv.Itoa(score), true
}

// voidStage is one Critical Node stage: its buff, name, score, rating and
// team; full is the score upstream shows as its full-score badge.
func voidStage(resources *recordResources, item map[string]any, name string, full int) map[string]any {
	buffer, _ := item["buffer"].(map[string]any)
	score := app.Int(item["score"])
	stage := map[string]any{"name": name, "score": score, "max": score == full, "full": full,
		"rating": strings.ReplaceAll(app.Text(item["star"]), "+", "P"), "team": bangbooTeam(resources, item)}
	if icon := app.Text(buffer["icon"]); icon != "" {
		stage["pop"] = resources.official(icon)
	}
	return stage
}

func voidRankRow(resources *recordResources, data map[string]any) map[string]any {
	brief, _ := data["void_front_battle_abstract_info_brief"].(map[string]any)
	record, _ := data["boss_challenge_record"].(map[string]any)
	info, _ := record["boss_info"].(map[string]any)
	main, _ := record["main_challenge_record"].(map[string]any)
	list, _ := data["main_challenge_record_list"].([]any)
	stages := []any{}
	for index := range 3 {
		if index >= len(list) {
			stages = append(stages, nil)
			continue
		}
		item, _ := list[index].(map[string]any)
		full := 149500
		if index == 0 {
			full = 182000
		}
		stages = append(stages, voidStage(resources, item, app.Text(item["name"]), full))
	}
	score := app.Int(brief["total_score"])
	return map[string]any{"score": score, "max": score == 663000, "boss": voidStage(resources, main, app.Text(info["name"]), 182000), "stages": stages}
}

// towerRankPage is a Simulated Battle Trial season's page: S1 ranks by floor,
// S2 by floor and flawless floors, S3 and S4 by score, floor and flawless
// floors.
func towerRankPage(season string) rankPage {
	fields := func(data map[string]any) (layer, mvp map[string]any, ok bool) {
		value, ok := data["climbing_tower_"+season].(map[string]any)
		if !ok {
			return nil, nil, false
		}
		// The first two seasons keep their fields at the top.
		layer, mvp = value, value
		if season == "s3" || season == "s4" {
			layer, _ = value["layer_info"].(map[string]any)
			mvp, _ = value["mvp_info"].(map[string]any)
		}
		return layer, mvp, true
	}
	return rankPage{
		score: func(now time.Time, data map[string]any) ([]float64, string, bool) {
			layer, mvp, ok := fields(data)
			if !ok {
				return nil, "", false
			}
			floor, flawless, score := app.Int(layer["climbing_tower_layer"]), app.Int(mvp["floor_mvp_num"]), app.Int(layer["total_score"])
			switch season {
			case "s1":
				return []float64{float64(floor)}, fmt.Sprintf("%d 层", floor), true
			case "s2":
				return []float64{float64(floor), float64(flawless)}, fmt.Sprintf("%d 层 · 无伤 %d", floor, flawless), true
			}
			return []float64{float64(score), float64(floor), float64(flawless)}, fmt.Sprintf("%d · %d 层 · 无伤 %d", score, floor, flawless), true
		},
		row: func(resources *recordResources, data map[string]any) map[string]any {
			layer, mvp, _ := fields(data)
			return map[string]any{"floor": app.Int(layer["climbing_tower_layer"]), "flawless": app.Int(mvp["floor_mvp_num"]),
				"score": app.Int(layer["total_score"]), "medal": resources.official(layer["medal_icon"])}
		},
	}
}
