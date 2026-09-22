package app

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/RayleaBot/plugin-zzz/internal/gacha"
)

type Row struct {
	Label string `json:"label"`
	Value string `json:"value"`
}
type Section struct {
	Title string `json:"title"`
	Rows  []Row  `json:"rows"`
	Text  string `json:"text,omitempty"`
}
type View struct {
	Title    string    `json:"title"`
	Subtitle string    `json:"subtitle"`
	Rows     []Row     `json:"rows"`
	Sections []Section `json:"sections"`
	Note     string    `json:"note"`
	// Image draws the view with a plugin template instead of the summary card.
	Image *Image `json:"-"`
}

func (v View) Text() string {
	lines := []string{v.Title}
	if v.Subtitle != "" {
		lines = append(lines, v.Subtitle)
	}
	for _, row := range v.Rows {
		lines = append(lines, row.Label+"："+row.Value)
	}
	for _, section := range v.Sections {
		lines = append(lines, "\n"+section.Title)
		for _, row := range section.Rows {
			lines = append(lines, row.Label+"："+row.Value)
		}
		if section.Text != "" {
			lines = append(lines, section.Text)
		}
	}
	if v.Note != "" {
		lines = append(lines, "\n"+v.Note)
	}
	return strings.Join(lines, "\n")
}
func kindLabel(kind string) string {
	switch kind {
	case "character":
		return "角色"
	case "weapon":
		return "装备"
	case "buddy":
		return "邦布"
	}
	return "资料"
}
func EntryView(game Game, entry Entry) View {
	v := View{Title: entry.Name, Subtitle: game.Name + " · " + kindLabel(entry.Kind), Rows: []Row{}, Sections: []Section{}}
	if entry.Rarity > 0 {
		rarity := strconv.Itoa(entry.Rarity) + " 星"
		if game.ID == "zzz" {
			rarity = map[int]string{4: "S", 3: "A", 2: "B"}[entry.Rarity]
		}
		v.Rows = append(v.Rows, Row{Label: "稀有度", Value: rarity})
	}
	if entry.Element != "" {
		v.Rows = append(v.Rows, Row{Label: "属性", Value: entry.Element})
	}
	if entry.Weapon != "" {
		v.Rows = append(v.Rows, Row{Label: "类型", Value: entry.Weapon})
	}
	if entry.Description != "" {
		v.Sections = append(v.Sections, Section{Title: "介绍", Text: entry.Description})
	}
	if len(entry.Materials) > 0 {
		rows := []Row{}
		keys := []string{}
		for key := range entry.Materials {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			rows = append(rows, Row{Label: key, Value: entry.Materials[key]})
		}
		v.Sections = append(v.Sections, Section{Title: "养成材料", Rows: rows})
	}
	for _, talent := range entry.Talents {
		v.Sections = append(v.Sections, Section{Title: talent.Name, Text: strings.Join(talent.Description, "\n")})
	}
	return v
}
func GachaView(game Game, archive gacha.Archive) View {
	v := View{Title: game.Name + "抽卡记录", Subtitle: archive.UID + " · " + archive.Region, Rows: []Row{{Label: "记录总数", Value: strconv.Itoa(len(archive.Records))}}, Sections: []Section{}, Note: "统计基于已保存记录；缺少更早历史时不能确定首次出货前的完整抽数。"}
	fours := gachaFours(game.ID, archive)
	for _, pool := range gacha.Summarize(game.ID, archive) {
		label := poolName(game.ID, pool.Pool)
		pity := strconv.Itoa(pool.CurrentPity)
		if pool.PityLowerBound {
			pity = "至少 " + pity
		}
		if pool.PityUncertain {
			pity = "稀有度资料不全，暂不计算"
		}
		top := strconv.Itoa(pool.Top)
		if pool.UnknownRank > 0 {
			top = "至少 " + top
		}
		rows := []Row{{Label: "抽数", Value: strconv.Itoa(pool.Total)}, {Label: "最高稀有度", Value: top}, {Label: "当前未出货", Value: pity}}
		// Each top-rarity pull with its count and whether it was featured, as
		// the upstream analyses list them, and the averages they show.
		names, pulls, counted, featured, upPulls := []string{}, 0, 0, 0, 0
		for _, rare := range pool.Rare {
			text := rare.Name + " " + strconv.Itoa(rare.Pulls)
			if rare.LowerBound {
				text += "+"
			}
			if up := gachaFeatured(game, pool.Pool, rare); up {
				text += " UP"
				featured++
			}
			names = append(names, text)
			if !rare.LowerBound && !rare.Uncertain {
				pulls += rare.Pulls
				counted++
			}
		}
		if len(names) > 0 {
			rows = append(rows, Row{Label: "出货", Value: strings.Join(names, "、")})
		}
		if counted > 0 {
			rows = append(rows, Row{Label: "平均出货", Value: strconv.FormatFloat(float64(pulls)/float64(counted), 'f', 1, 64) + " 抽"})
		}
		if featured > 0 {
			upPulls = pool.Total - pool.CurrentPity
			rows = append(rows, Row{Label: "UP 平均", Value: strconv.FormatFloat(float64(upPulls)/float64(featured), 'f', 1, 64) + " 抽"})
		}
		if four := fours[pool.Pool]; four > 0 {
			rows = append(rows, Row{Label: "次高稀有度", Value: strconv.Itoa(four) + " 个 · 平均 " + strconv.FormatFloat(float64(pool.Total)/float64(four), 'f', 1, 64) + " 抽"})
		}
		v.Sections = append(v.Sections, Section{Title: label, Rows: rows})
	}
	return v
}

// gachaFours counts each pool's second-rarity pulls: four-star, or A-rank in
// ZZZ.
func gachaFours(game string, archive gacha.Archive) map[string]int {
	rank := "4"
	if game == "zzz" {
		rank = "3"
	}
	counts := map[string]int{}
	for _, record := range archive.Records {
		if record.Rank == rank {
			counts[gacha.Pool(game, record.GachaType)]++
		}
	}
	return counts
}

// gachaFeatured tells whether a top-rarity pull was featured in an event
// banner running when it was pulled.
func gachaFeatured(game Game, pool string, rare gacha.Rare) bool {
	if game.Data == nil || slices.Contains([]string{"100", "200", "1", "2", "5"}, pool) {
		return false
	}
	stamp := rare.Time
	for _, period := range game.Data.Resources.Pools {
		if period.From == "" || stamp < period.From || stamp > period.To {
			continue
		}
		if slices.Contains(period.Characters5, rare.Name) || slices.Contains(period.Weapons5, rare.Name) {
			return true
		}
	}
	return false
}

func poolName(game, pool string) string {
	names := map[string]map[string]string{"genshin": {"100": "新手祈愿", "200": "常驻祈愿", "301": "角色活动祈愿", "302": "武器活动祈愿", "500": "集录祈愿"}, "starrail": {"1": "常驻跃迁", "2": "始发跃迁", "11": "角色活动跃迁", "12": "光锥活动跃迁", "21": "联动角色跃迁", "22": "联动光锥跃迁"}, "zzz": {"1": "常驻频道", "2": "独家频道", "3": "音擎频道", "5": "邦布频道", "102": "独家重映", "103": "音擎回响"}}
	if name := names[game][pool]; name != "" {
		return name
	}
	return "卡池 " + pool
}

func BusinessView(game Game, operation Operation, result QueryResult, catalog Catalog) View {
	v := View{Title: operation.Label, Subtitle: result.Role.Nickname + " · " + result.Role.UID, Rows: []Row{}, Sections: []Section{}}
	data := result.Data
	if strings.HasSuffix(operation.Name, ".character") {
		return PanelView(game, NormalizePanels(game.ID, result, catalog), result.Role.UID)
	}
	if extendedBusinessView(&v, game, operation, data, catalog) {
		return v
	}
	if strings.HasSuffix(operation.Name, ".note") {
		switch game.ID {
		case "genshin":
			addRatio(&v, "原粹树脂", data, "current_resin", "max_resin")
			addSeconds(&v, "树脂回满", data["resin_recovery_time"])
			addRatio(&v, "每日委托", data, "finished_task_num", "total_task_num")
			addRatio(&v, "洞天宝钱", data, "current_home_coin", "max_home_coin")
			addRatio(&v, "探索派遣", data, "current_expedition_num", "max_expedition_num")
		case "starrail":
			addRatio(&v, "开拓力", data, "current_stamina", "max_stamina")
			addSeconds(&v, "开拓力回满", data["stamina_recover_time"])
			addScalar(&v, "后备开拓力", data["current_reserve_stamina"])
			addRatio(&v, "每日实训", data, "current_train_score", "max_train_score")
			addRatio(&v, "模拟宇宙积分", data, "current_rogue_score", "max_rogue_score")
			addRatio(&v, "委托", data, "accepted_epedition_num", "total_expedition_num")
		case "zzz":
			energy := asObject(data["energy"])
			progress := asObject(energy["progress"])
			addRatio(&v, "电量", progress, "current", "max")
			addSeconds(&v, "电量回满", energy["restore"])
			if len(v.Rows) == 0 {
				addRatio(&v, "电量", data, "current_energy", "max_energy")
			}
			addRatio(&v, "活跃度", asObject(data["vitality"]), "current", "max")
			addRatio(&v, "悬赏委托", asObject(data["bounty_commission"]), "num", "total")
		}
	} else if strings.HasSuffix(operation.Name, ".profile") {
		stats := asObject(data["stats"])
		for _, field := range [][2]string{{"active_day_number", "活跃天数"}, {"active_days", "活跃天数"}, {"achievement_number", "成就"}, {"achievement_num", "成就"}, {"achievement_count", "成就"}, {"avatar_number", "角色"}, {"avatar_num", "角色"}, {"buddy_num", "邦布"}, {"spiral_abyss", "深境螺旋"}, {"way_point_number", "传送点"}, {"chest_num", "宝箱"}, {"world_level_name", "称号"}} {
			addScalar(&v, field[1], stats[field[0]])
		}
	} else {
		for _, field := range [][2]string{{"total_star", "星数"}, {"star_num", "星数"}, {"max_floor", "最高层数"}, {"total_battle_times", "战斗次数"}, {"battle_num", "战斗次数"}, {"highest_floor", "最高层数"}, {"max_layer", "最高层数"}, {"schedule_id", "期数"}} {
			addScalar(&v, field[1], data[field[0]])
		}
		for _, key := range []string{"list", "avatars", "avatar_list", "buddy_list"} {
			list := asList(data[key])
			if len(list) == 0 {
				continue
			}
			rows := []Row{}
			for _, raw := range list {
				item := asObject(raw)
				basic := asObject(item["base"])
				if basic == nil {
					basic = item
				}
				id := asText(basic["id"])
				name := asText(basic["name"])
				if name == "" {
					name = asText(basic["full_name"])
				}
				if name == "" {
					name = asText(basic["full_name_mi18n"])
				}
				if name == "" {
					name = asText(basic["name_mi18n"])
				}
				if name == "" {
					if entry, ok := catalog.Get(id); ok {
						name = entry.Name
					} else {
						name = "角色 " + id
					}
				}
				level := asText(basic["level"])
				if level == "" {
					level = asText(item["level"])
				}
				value := "等级 " + level
				rows = append(rows, Row{Label: name, Value: value})
				if len(rows) >= 100 {
					break
				}
			}
			v.Sections = append(v.Sections, Section{Title: "角色与装备", Rows: rows})
			break
		}
		appendChallenge(&v, game, data, catalog)
	}
	if len(v.Rows) == 0 && len(v.Sections) == 0 {
		v.Note = "本期没有可展示的记录，或数据结构尚未适配。"
	}
	if unlocked, ok := data["is_unlock"].(bool); ok && !unlocked {
		v.Note = "此挑战尚未解锁。"
	}
	if hasData, ok := data["has_data"].(bool); ok && !hasData {
		v.Note = "本期没有挑战记录。"
	}
	return v
}

func appendChallenge(view *View, game Game, data map[string]any, catalog Catalog) {
	if game.ID == "genshin" {
		for _, raw := range asList(data["floors"]) {
			floor := asObject(raw)
			rows := []Row{{Label: "星数", Value: asText(floor["star"]) + " / " + asText(floor["max_star"])}}
			for _, raw := range asList(floor["levels"]) {
				level := asObject(raw)
				rows = append(rows, Row{Label: "第 " + asText(level["index"]) + " 间", Value: asText(level["star"]) + " 星"})
				for index, raw := range asList(level["battles"]) {
					battle := asObject(raw)
					rows = append(rows, Row{Label: fmt.Sprintf("队伍 %d", index+1), Value: teamNames(asList(battle["avatars"]), catalog)})
				}
			}
			view.Sections = append(view.Sections, Section{Title: "第 " + asText(floor["index"]) + " 层", Rows: rows})
		}
	}
	if game.ID == "starrail" {
		for _, raw := range asList(data["all_floor_detail"]) {
			floor := asObject(raw)
			rows := []Row{}
			for _, field := range [][2]string{{"star_num", "星数"}, {"round_num", "轮数"}, {"score", "得分"}} {
				if value := asText(floor[field[0]]); value != "" {
					rows = append(rows, Row{Label: field[1], Value: value})
				}
			}
			for index, key := range []string{"node_1", "node_2"} {
				node := asObject(floor[key])
				if list := asList(node["avatars"]); len(list) > 0 {
					rows = append(rows, Row{Label: fmt.Sprintf("队伍 %d", index+1), Value: teamNames(list, catalog)})
				}
				if score := asText(node["score"]); score != "" {
					rows = append(rows, Row{Label: fmt.Sprintf("队伍 %d 得分", index+1), Value: score})
				}
			}
			view.Sections = append(view.Sections, Section{Title: "关卡 " + asText(floor["name"]) + asText(floor["maze_id"]), Rows: rows})
		}
	}
	if game.ID == "zzz" {
		info := asObject(data["hadal_info_v2"])
		brief := asObject(info["brief"])
		for _, field := range [][2]string{{"score", "得分"}, {"max_score", "满分"}, {"rating", "评级"}, {"cur_period_zone_layer_count", "完成层数"}} {
			addScalar(view, field[1], brief[field[0]])
		}
		for index, key := range []string{"first_layer_detail", "second_layer_detail", "third_layer_detail", "fourth_layer_detail", "fitfh_layer_detail"} {
			detail := asObject(info[key])
			if detail == nil {
				continue
			}
			rows := []Row{}
			if rating := asText(detail["rating"]); rating != "" {
				rows = append(rows, Row{Label: "评级", Value: rating})
			}
			for team, raw := range asList(detail["layer_challenge_info_list"]) {
				battle := asObject(raw)
				rows = append(rows, Row{Label: fmt.Sprintf("队伍 %d", team+1), Value: teamNames(asList(battle["avatar_list"]), catalog)})
				for _, field := range [][2]string{{"rating", "评级"}, {"score", "得分"}, {"battle_time", "战斗用时（秒）"}} {
					if value := asText(battle[field[0]]); value != "" {
						rows = append(rows, Row{Label: field[1], Value: value})
					}
				}
			}
			view.Sections = append(view.Sections, Section{Title: fmt.Sprintf("第 %d 层", index+1), Rows: rows})
		}
	}
}

func teamNames(list []any, catalog Catalog) string {
	names := []string{}
	for _, raw := range list {
		item := asObject(raw)
		id := asText(item["id"])
		if id == "" {
			id = asText(item["avatar_id"])
		}
		name := asText(item["name"])
		if name == "" {
			if entry, ok := catalog.Get(id); ok {
				name = entry.Name
			} else {
				name = id
			}
		}
		if name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, "、")
}
func addScalar(v *View, label string, value any) {
	if text := asText(value); text != "" {
		v.Rows = append(v.Rows, Row{Label: label, Value: text})
	}
}
func addRatio(v *View, label string, data map[string]any, current, maximum string) {
	if value, ok := data[current]; ok {
		v.Rows = append(v.Rows, Row{Label: label, Value: asText(value) + " / " + asText(data[maximum])})
	}
}
func addSeconds(v *View, label string, value any) {
	if text := asText(value); text != "" {
		seconds, _ := strconv.Atoi(text)
		v.Rows = append(v.Rows, Row{Label: label, Value: fmt.Sprintf("%d 小时 %d 分钟", seconds/3600, seconds%3600/60)})
	}
}
