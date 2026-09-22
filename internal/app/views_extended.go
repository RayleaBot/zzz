package app

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

func extendedBusinessView(v *View, game Game, operation Operation, data map[string]any, catalog Catalog) bool {
	if strings.HasSuffix(operation.Name, ".monthly") {
		monthlyView(v, data)
		return true
	}
	name := strings.TrimPrefix(operation.Name, game.ID+".")
	if !slices.Contains([]string{"deadly", "holo_boss", "tower", "void_front", "hollow_zero", "hollow_zero_detail", "lost_void", "zenkov", "zenkov_detail", "exploration"}, name) {
		return false
	}
	walkMetrics(v, data, operation.Label, catalog, 0)
	if len(v.Rows) == 0 && len(v.Sections) == 0 {
		v.Note = "暂无可展示的记录。"
	}
	for _, key := range []string{"is_unlock", "unlock", "abyss_unlock", "unlocked"} {
		if flag, ok := data[key].(bool); ok && !flag {
			v.Note = "此玩法尚未解锁。"
		}
	}
	if flag, ok := data["has_data"].(bool); ok && !flag {
		v.Note = "本期没有挑战记录。"
	}
	return true
}

func firstText(value map[string]any, keys ...string) string {
	for _, key := range keys {
		if text := asText(value[key]); text != "" {
			return text
		}
	}
	return ""
}

func calendarTime(value any) string {
	if obj := asObject(value); obj != nil {
		y, m, d := asText(obj["year"]), asText(obj["month"]), asText(obj["day"])
		if y != "" && m != "" && d != "" {
			return y + "-" + m + "-" + d
		}
	}
	raw := asText(value)
	stamp, err := strconv.ParseInt(raw, 10, 64)
	if err == nil && stamp > 1000000000 && stamp < 10000000000 {
		return time.Unix(stamp, 0).In(time.FixedZone("UTC+8", 8*3600)).Format("2006-01-02 15:04")
	}
	return ""
}

func monthlyView(v *View, data map[string]any) {
	month := asObject(data["month_data"])
	addScalar(v, "统计月份", data["data_month"])
	for _, raw := range asList(month["list"]) {
		item := asObject(raw)
		label := asText(item["data_name"])
		if label == "" {
			label = map[string]string{"PolychromesData": "菲林", "MatserTapeData": "母带", "BooponsData": "邦布券"}[asText(item["data_type"])]
		}
		if label != "" {
			addScalar(v, label, item["count"])
		}
	}
	groups := asList(month["group_by"])
	if len(groups) == 0 {
		groups = asList(month["income_components"])
	}
	rows := []Row{}
	for _, raw := range groups {
		item := asObject(raw)
		label := firstText(item, "action_name", "action")
		if translated := incomeNames[label]; translated != "" {
			label = translated
		}
		if label == "" {
			label = "其他来源"
		}
		value := asText(item["num"])
		if percent := asText(item["percent"]); percent != "" {
			value += " · " + percent + "%"
		}
		rows = append(rows, Row{Label: label, Value: value})
	}
	if len(rows) > 0 {
		v.Sections = append(v.Sections, Section{Title: "收入来源", Rows: rows})
	}
	available := []string{}
	for _, raw := range asList(data["optional_month"]) {
		if value := asText(raw); value != "" {
			available = append(available, value)
		}
	}
	if len(available) > 0 {
		v.Note = "可查询月份：" + strings.Join(available, "、") + "。数据以官方统计口径为准。"
	}
}

var incomeNames = map[string]string{"daily_activity_rewards": "日常活跃奖励", "growth_rewards": "成长奖励", "event_rewards": "活动奖励", "hollow_rewards": "零号空洞奖励", "shiyu_rewards": "式舆防卫战奖励", "mail_rewards": "邮件奖励", "other_rewards": "其他奖励"}

// Render only known business measures and grouping fields. Unknown upstream
// keys and image/link metadata are never dumped into a user's game report.
var measureNames = map[string]string{
	"level": "等级", "name": "名称", "name_mi18n": "名称", "season_name": "赛季", "title": "标题", "score": "得分", "total_score": "总分", "max_score": "满分", "star": "星数", "star_num": "星数", "total_star": "总星数", "rank": "级别", "rating": "评级", "round_num": "轮数", "battle_num": "战斗次数", "total_battle_num": "总战斗次数", "difficulty": "难度", "difficulty_id": "难度", "max_round_id": "最高幕数", "medal_num": "勋章", "coin_num": "幻剧之花", "rent_cnt": "助演次数", "heraldry": "徽记", "avatar_bonus_num": "增益角色", "tarot_finished_cnt": "月谕圣牌进度", "best_time": "最佳用时（秒）", "battle_time": "战斗用时（秒）", "total_use_time": "总用时（秒）", "max_name": "最高记录", "max_count": "最高层数", "heat_count": "热度", "high_difficulty": "难度", "boss_stars": "首领星数", "mob_stars": "关卡星数", "finish_cnt": "完成次数", "unlocked_buff_num": "已解锁祝福", "unlocked_miracle_num": "已解锁奇物", "unlocked_skill_points": "技能点", "unlocked": "已解锁", "unlock": "已解锁", "is_unlock": "已解锁", "is_lock": "锁定", "has_data": "已有记录", "exist_data": "已有记录", "has_challenge_record": "已有挑战记录", "has_played": "已参与", "finished_weekly": "本周已完成", "hard_mode": "困难模式", "finish_color_medal": "彩色勋章", "is_success": "成功撤离", "is_finish": "已完成", "active_nerve": "激活神经", "unlock_event": "已解锁事件", "unlock_miracle": "已解锁奇物", "season_level": "赛季等级", "weekly_score": "本周积分", "skill_tree_activated": "激活节点", "discover_secrets": "发现秘密", "linear_tree_num": "神经网络节点", "magic_compendium": "权杖图鉴", "additional_problems_num": "额外难题", "additional_rounds": "额外回合", "calculation_tendency": "演算倾向", "climbing_tower_layer": "最高层数", "floor_mvp_num": "层内最佳次数", "collect_total_value": "累计收集价值", "big_red_num": "大红数量", "millions_evacuations": "百万撤离次数", "max_rank": "最高级别", "collection_progress": "探索进度", "handbook_progress": "图鉴进度", "trait_progress": "羁绊进度", "archive_rank": "排名", "remain_hp": "剩余生命", "lineup_coin": "阵容价值", "total_coin": "总金币", "archive_time": "记录时间", "division_level": "段位", "num": "数量", "total": "总数", "cur_duty": "当前委托", "max_duty": "委托上限", "count": "数量", "use_count": "使用次数", "proficiency": "熟练度", "card_name": "卡牌", "deck_name": "牌组", "damage": "伤害", "total_time": "总用时（秒）",
}
var groupNames = map[string]string{
	"stat": "统计", "statistic": "统计", "stats": "统计", "basic": "概况", "basic_info": "基础资料", "detail": "详细记录", "detail_stat": "详细统计", "fight_statisic": "战斗统计", "single": "单人挑战", "mp": "多人挑战", "best": "最佳记录", "challenge": "挑战记录", "rounds_data": "分幕记录", "current_record": "本期宇宙", "last_record": "上期宇宙", "latest_record": "最近记录", "best_record": "最佳记录", "records": "记录", "record_list": "记录", "normal_record_list": "常规记录", "custom_record_list": "自定义记录", "normal_detail": "常规演算", "cur_week_detail": "本周演算", "last_week_detail": "上周演算", "normal_record_brief": "常规概况", "weekly_record_brief": "周期概况", "common_info": "挑战信息", "common_info_v2": "挑战信息", "magic_record": "演算记录", "cnt": "图鉴收集", "destiny": "命途", "challenge_peak_best_record_brief": "仲裁最佳记录", "challenge_peak_records": "仲裁记录", "boss_record": "首领记录", "mob_records": "关卡记录", "group": "赛季", "boss_info": "首领", "mob_infos": "关卡", "act_info_list": "周期活动", "grid_fight_brief": "货币战争概况", "grid_fight_archive_list": "货币战争记录", "brief": "概况", "division": "段位", "lineup": "阵容", "augment_list": "投资环境", "trait_list": "羁绊", "damage_list": "伤害", "list": "记录", "hard_list": "困难记录", "boss": "首领", "buffer": "增益", "buff": "增益", "climbing_tower_s1": "第一赛季", "climbing_tower_s2": "第二赛季", "climbing_tower_s3": "第三赛季", "climbing_tower_s4": "第四赛季", "layer_info": "层数", "mvp_info": "最佳表现", "void_front_battle_detail": "推演记录", "void_front_battle_abstract_info_brief": "推演概况", "boss_challenge_record": "首领关卡", "main_challenge_record": "主线战绩", "main_challenge_record_list": "主线关卡", "sub_challenge_record": "支线关卡", "abyss_level": "执照等级", "abyss_point": "积分", "abyss_duty": "委托", "abyss_talent": "天赋", "abyss_collect": "收集", "abyss_nest": "枯萎苗圃", "abyss_throne": "刀耕火焚", "abyss_throne_max": "最高挑战", "abyss_max": "最高记录", "abyss_task": "任务", "abyss_task_force_investigation_max": "强袭调查", "special_mission": "特殊任务", "season_data": "赛季", "season_quest": "赛季任务", "season_coin": "赛季货币", "map_list": "地图", "collection_data": "收集", "medal_data": "勋章", "goods_data": "藏品", "area_collections": "区域收集", "map_collections": "地图收集", "collections": "收集项目", "cat_notes": "喵吉笔记", "medal_list": "奖章", "card_list": "卡牌", "avatar_cards": "角色牌", "action_cards": "行动牌", "cards": "卡牌", "display_avatar_rank_list": "代理人表现", "persona_style": "演算风格",
}
var ratios = []struct{ current, maximum, label string }{
	{"current_rogue_score", "max_rogue_score", "宇宙积分"}, {"cur_level", "max_level", "等级"}, {"cur_point", "max_point", "积分"}, {"cur_talent", "max_talent", "天赋"}, {"cur_collect", "max_collect", "收集"}, {"cur_progress", "max_progress", "进度"}, {"cur_rolling", "max_rolling", "骰面"}, {"cur_task", "max_task", "任务"}, {"quest_cur", "quest_max", "任务"}, {"weekly_score_cur", "weekly_score_max", "每周积分"}, {"weekly_score", "weekly_score_max", "每周积分"}, {"season_task_finished", "season_task_total", "赛季任务"}, {"possibility_gallery_finished", "possibility_gallery_total", "可能性画廊"}, {"titan_current", "titan_total", "泰坦"}, {"challenge_task_current_num", "challenge_task_total_num", "挑战任务"}, {"schedule_current_floor_num", "schedule_total_floor_num", "层数"}, {"cur", "total", "收集"}, {"cur_quest", "max_quest", "赛季任务"}, {"cur_coin", "max_coin", "赛季货币"},
}

func walkMetrics(v *View, data map[string]any, title string, catalog Catalog, depth int) {
	if data == nil || depth > 6 {
		return
	}
	if len(v.Sections) >= 80 {
		v.Note = "记录较多，仅展示前 80 个分组。"
		return
	}
	rows := []Row{}
	handled := map[string]bool{}
	for _, ratio := range ratios {
		if value, ok := data[ratio.current]; ok {
			if maximum, ok := data[ratio.maximum]; ok {
				rows = append(rows, Row{Label: ratio.label, Value: asText(value) + " / " + asText(maximum)})
				handled[ratio.current] = true
				handled[ratio.maximum] = true
			}
		}
	}
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		if handled[key] {
			continue
		}
		label := measureNames[key]
		if label == "" {
			continue
		}
		value := asText(data[key])
		if flag, ok := data[key].(bool); ok {
			value = "否"
			if flag {
				value = "是"
			}
		}
		if value != "" {
			rows = append(rows, Row{Label: label, Value: value})
		}
	}
	for _, key := range []string{"avatar_list", "avatars", "final_lineup", "front_roles", "back_roles", "active_avatars"} {
		if list := asList(data[key]); len(list) > 0 {
			label := "队伍"
			if key == "front_roles" {
				label = "前排"
			}
			if key == "back_roles" {
				label = "后排"
			}
			rows = append(rows, Row{Label: label, Value: teamNames(list, catalog)})
		}
	}
	if len(rows) > 0 {
		v.Sections = append(v.Sections, Section{Title: title, Rows: rows})
	}
	for _, key := range keys {
		group := groupNames[key]
		if group == "" {
			continue
		}
		if object := asObject(data[key]); object != nil {
			walkMetrics(v, object, title+" · "+group, catalog, depth+1)
			continue
		}
		for i, raw := range asList(data[key]) {
			item := asObject(raw)
			if item == nil {
				continue
			}
			label := firstText(item, "name", "name_mi18n", "season_name", "title", "map_name")
			if label == "" {
				label = fmt.Sprintf("%s %d", group, i+1)
			}
			walkMetrics(v, item, title+" · "+label, catalog, depth+1)
		}
	}
}
