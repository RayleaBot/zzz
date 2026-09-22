package app

import (
	"math"
	"slices"
	"strconv"
	"strings"
)

type CloudFilter struct {
	Op    string `json:"op"`
	Value int    `json:"value"`
}
type CloudRankRow struct {
	Index        int    `json:"index"`
	UID          string `json:"uid"`
	Level        string `json:"level"`
	Cons         string `json:"cons"`
	Weapon       string `json:"weapon"`
	Damage       string `json:"damage"`
	Score        string `json:"score"`
	View         View   `json:"view"`
	CanReadPanel bool   `json:"can_read_panel"`
}
type CloudRanking struct {
	Rows []CloudRankRow `json:"rows"`
	Sort string         `json:"sort"`
}

func cloudGame(game string) string {
	if game == "genshin" {
		return "gs"
	}
	return "sr"
}
func customCloudRequest(game string, input CloudInput, id int) (string, map[string]any, error) {
	if input.Sort == "" {
		input.Sort = "dmg_avg"
	}
	if input.Limit == 0 {
		input.Limit = 20
	}
	if (input.Sort != "dmg_avg" && input.Sort != "mark_score") || input.Limit < 1 || input.Limit > 50 || len(input.Filters) > 2 {
		return "", nil, gameError("input_invalid", "请选择伤害或装备评分排序，数量为 1–50，最多两项命座筛选。")
	}
	rules := map[string]int{">=": 0, "=": 1, "<=": 2, ">": 3, "<": 4, "!=": 5}
	filters := []any{}
	for _, f := range input.Filters {
		rule, ok := rules[f.Op]
		if !ok || f.Value < 0 || f.Value > 6 {
			return "", nil, gameError("input_invalid", "命座/星魂筛选必须使用有效比较符和 0–6 的整数。")
		}
		filters = append(filters, map[string]any{"type": "cons", "rule": rule, "value": f.Value})
	}
	advanced, err := advancedCloudFilters(game, input)
	if err != nil {
		return "", nil, err
	}
	filters = append(filters, advanced...)
	return "rank/custom", map[string]any{"version": "0.1.0", "charId": id, "game": cloudGame(game), "data": map[string]any{"rank": map[string]any{"col": input.Sort, "order": "desc"}, "filter": filters, "nums": input.Limit}}, nil
}
func cloudText(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	r := []rune(plainGameText(s))
	if len(r) > 200 {
		r = r[:200]
	}
	return string(r)
}
func cloudNumber(v any) string {
	s := asText(v)
	n, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsInf(n, 0) || math.IsNaN(n) || math.Abs(n) > 1e18 {
		return ""
	}
	return strconv.FormatFloat(n, 'f', -1, 64)
}
func cloudNumericRows(raw map[string]any, fields ...string) []Row {
	rows := []Row{}
	for i := 0; i < len(fields); i += 2 {
		if n := cloudNumber(raw[fields[i]]); n != "" {
			rows = append(rows, Row{Label: fields[i+1], Value: n})
		}
	}
	return rows
}
func customCloudResult(game Game, input CloudInput, result map[string]any) (CloudResult, error) {
	data := asObject(result["data"])
	rows, ok := data["rows"].([]any)
	if !ok || len(rows) > 50 {
		return CloudResult{}, gameError("cloud_invalid", "云榜单的数据格式暂不兼容。")
	}
	queryID := firstText(data, "query_id", "queryId")
	if len(queryID) > 256 {
		return CloudResult{}, gameError("cloud_invalid", "云榜单引用格式暂不兼容。")
	}
	sort := input.Sort
	if sort == "" {
		sort = "dmg_avg"
	}
	ranking := &CloudRanking{Rows: []CloudRankRow{}, Sort: sort}
	for i, raw := range rows {
		r := asObject(raw)
		if r == nil {
			return CloudResult{}, gameError("cloud_invalid", "云榜单行格式暂不兼容。")
		}
		uid := asText(r["uid"])
		if !uidPattern.MatchString(uid) {
			uid = ""
		}
		weapon := cloudText(r["weapon_name"])
		if weapon == "" {
			weapon = cloudNumber(r["weapon_id"])
		}
		v := View{Title: "UID " + uid, Rows: cloudNumericRows(r, "level", "等级", "promote", "突破", "cons", "命座 / 星魂", "weapon_level", "装备等级", "weapon_promote", "装备突破", "weapon_affix", "精炼 / 叠影", "talent_a", "普攻", "talent_e", "战技", "talent_q", "爆发 / 终结技", "talent_t", "天赋", "talent_me", "忆灵技", "talent_mt", "忆灵天赋", "dmg_avg", "服务伤害评分", "mark_score", "服务装备评分")}
		if weapon != "" {
			v.Rows = append(v.Rows, Row{Label: "武器 / 光锥", Value: weapon})
		}
		ranking.Rows = append(ranking.Rows, CloudRankRow{Index: i, UID: uid, Level: cloudNumber(r["level"]), Cons: cloudNumber(r["cons"]), Weapon: weapon, Damage: cloudNumber(r["dmg_avg"]), Score: cloudNumber(r["mark_score"]), View: v, CanReadPanel: queryID != "" && i < 20})
	}
	v := View{Title: game.Name + "自定义云排名", Subtitle: cloudText(data["dmgTitle"]), Rows: []Row{{Label: "返回条目", Value: strconv.Itoa(len(rows))}}, Note: "来源：ark.ivny.cn；仅代表服务收录的数据与评分。榜单前 20 项可在查询后五分钟内读取其公开云面板。"}
	if len(rows) == 0 {
		v.Note = "云服务暂无符合筛选条件的记录。"
	}
	return CloudResult{View: &v, Ranking: ranking, queryID: queryID}, nil
}

// Called under c.mu. The upstream query ID is owned by the result, never the UI.
func (c *CloudClient) rankPanelRequest(game string, input CloudInput) (string, map[string]any, CloudInput, error) {
	if !input.Consent || (game != "genshin" && game != "starrail") {
		return "", nil, input, gameError("cloud_consent_required", "请确认向 ark 云服务读取所选公开面板。")
	}
	j := c.jobs[input.Ref]
	if j == nil || j.game != game || j.State != "completed" || j.Ranking == nil || input.Index == nil || *input.Index < 0 || *input.Index >= len(j.Ranking.Rows) || !j.Ranking.Rows[*input.Index].CanReadPanel {
		return "", nil, input, gameError("cloud_missing", "此榜单条目不可读取或查询已失效，请重新查询排名。")
	}
	input.UID = j.Ranking.Rows[*input.Index].UID
	input.CharacterID = j.characterID
	return "rank/custom/specific", map[string]any{"version": "0.1.0", "query_id": j.queryID, "index": *input.Index}, input, nil
}

type CloudAdvancedFilter struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value any    `json:"value"`
}
type CloudFilterField struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Kind    string   `json:"kind"`
	Options []string `json:"options,omitempty"`
}

func cloudFilterSchema(game string) []CloudFilterField {
	out := []CloudFilterField{}
	for _, v := range [][2]string{{"level", "角色等级"}, {"promote", "角色突破"}, {"cons", "命座/星魂"}, {"talent_a", "普攻等级"}, {"talent_e", "战技等级"}, {"talent_q", "爆发/终结技等级"}, {"weapon_level", "装备等级"}, {"weapon_promote", "装备突破"}, {"weapon_affix", "精炼/叠影"}, {"dmg_avg", "伤害均值"}, {"mark_score", "装备评分"}, {"data_time", "数据时间戳"}} {
		out = append(out, CloudFilterField{Key: v[0], Label: v[1], Kind: "number"})
	}
	out = append(out, CloudFilterField{Key: "artis_sets", Label: "装备套装名称或编号", Kind: "text"})
	if game == "starrail" {
		for i, names := range [][]string{{"生命值"}, {"攻击力"}, {"生命百分比", "攻击百分比", "防御百分比", "暴击率", "暴击伤害", "治疗加成", "效果命中"}, {"生命百分比", "攻击百分比", "防御百分比", "速度"}, {"生命百分比", "攻击百分比", "防御百分比", "物理伤害", "火伤害", "冰伤害", "雷伤害", "风伤害", "量子伤害", "虚数伤害"}, {"击破特攻", "能量恢复效率", "生命百分比", "攻击百分比", "防御百分比"}} {
			out = append(out, CloudFilterField{Key: fmtPosition(i + 1), Label: "部位 " + strconv.Itoa(i+1) + " 主词条", Kind: "choice", Options: names})
		}
	}
	return out
}
func fmtPosition(n int) string { return "pos" + strconv.Itoa(n) }
func advancedCloudFilters(game string, input CloudInput) ([]any, error) {
	if len(input.AdvancedFilters) == 0 {
		return nil, nil
	}
	if !input.Authenticated || input.proxy == nil {
		return nil, gameError("cloud_credential_missing", "高级筛选需要启用授权查询，并在账号插件管理页配置 ark 令牌。")
	}
	if len(input.AdvancedFilters) > 16 {
		return nil, gameError("input_invalid", "高级筛选最多 16 项。")
	}
	fields := cloudFilterSchema(game)
	rules := map[string]int{">=": 0, "=": 1, "<=": 2, ">": 3, "<": 4, "!=": 5}
	out := []any{}
	for _, f := range input.AdvancedFilters {
		idx := slices.IndexFunc(fields, func(v CloudFilterField) bool { return v.Key == f.Field })
		rule, ok := rules[f.Op]
		if idx < 0 || !ok {
			return nil, gameError("input_invalid", "云筛选字段或比较符无效。")
		}
		field := fields[idx]
		var value any
		switch field.Kind {
		case "text":
			text := strings.TrimSpace(asText(f.Value))
			if text == "" || len([]rune(text)) > 64 || strings.ContainsAny(text, "\r\n\x00") {
				return nil, gameError("input_invalid", "套装名称或编号无效。")
			}
			value = text
		case "choice":
			n, ok := cloudInteger(f.Value, 1, len(field.Options))
			if !ok {
				return nil, gameError("input_invalid", "此部位不支持所选主词条。")
			}
			value = n
		default:
			n, err := strconv.ParseFloat(asText(f.Value), 64)
			if err != nil || !finiteRange(n, 0, 1e14) || (f.Field == "cons" && (n > 6 || math.Trunc(n) != n)) {
				return nil, gameError("input_invalid", "云筛选数值无效。")
			}
			value = n
		}
		out = append(out, map[string]any{"type": field.Key, "rule": rule, "value": value})
	}
	return out, nil
}
