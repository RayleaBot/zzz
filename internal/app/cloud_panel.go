package app

import (
	_ "embed"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type cloudAttribute struct {
	Key    string
	Title  string
	Format string
	Value  float64
	Base   float64
	Step   float64
}
type cloudItem struct {
	Name string
	Set  string
	Slot int
	Star int
}
type cloudGearData struct {
	AttrMap   map[string]cloudAttribute
	MainIDMap map[string]string
	AttrIDMap map[string]cloudAttribute
	Items     map[string]cloudItem
	Meta      struct {
		MainIdx  map[string]map[string]string
		StarData map[string]struct {
			Main map[string]cloudAttribute
			Sub  map[string]cloudAttribute
		}
	}
}

type CloudPanel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	View View   `json:"view"`
}

func cloudPanelResult(game Game, input CloudInput, result map[string]any) (CloudResult, error) {
	if game.Data == nil || game.Data.CloudGear == nil {
		return CloudResult{}, gameError("cloud_invalid", "本地面板解释资料无法读取。")
	}
	d := asObject(result["data"])
	if nested := asObject(d["info"]); nested != nil {
		d = nested
	} else if nested := asObject(d["playerData"]); nested != nil {
		d = nested
	}
	uid := asText(d["uid"])
	if input.UID != "" && (!uidPattern.MatchString(uid) || uid != input.UID) || input.Mode != "rank_panel" && !uidPattern.MatchString(uid) {
		return CloudResult{}, gameError("cloud_invalid", "云面板的 UID 与请求不一致。")
	}
	if !uidPattern.MatchString(uid) {
		uid = "服务未公开"
	}
	input.UID = uid
	list := []any{}
	switch avatars := d["avatars"].(type) {
	case map[string]any:
		if len(avatars) > 250 {
			return CloudResult{}, gameError("cloud_invalid", "云面板角色数量超过上限。")
		}
		keys := []string{}
		for key := range avatars {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			list = append(list, avatars[key])
		}
	case []any:
		list = avatars
	default:
		return CloudResult{}, gameError("cloud_invalid", "云面板未返回可识别角色资料。")
	}
	if len(list) > 250 {
		return CloudResult{}, gameError("cloud_invalid", "云面板角色数量超过上限。")
	}
	if game.Calc == nil {
		return CloudResult{}, gameError("cloud_invalid", "本地角色资料不可用。")
	}
	metadata := game.Calc.Metadata()
	names := map[string]string{}
	weapons := map[string]string{}
	for _, ch := range metadata.Characters {
		if ch.Game == cloudGame(game.ID) {
			names[ch.ID] = ch.Name
		}
	}
	for _, w := range metadata.Weapons {
		if w.Game == cloudGame(game.ID) {
			weapons[w.ID] = w.Name
		}
	}
	panels := []CloudPanel{}
	seen := map[string]bool{}
	for _, raw := range list {
		avatar := asObject(raw)
		id := cloudNumber(avatar["id"])
		if id == "" || seen[id] {
			return CloudResult{}, gameError("cloud_invalid", "云角色编号缺失或重复。")
		}
		seen[id] = true
		if input.Mode == "rank_panel" && input.CharacterID != "" && id != input.CharacterID {
			continue
		}
		name := names[id]
		if name == "" {
			name = "角色 " + id
		}
		v := View{Title: name + " · 云面板", Subtitle: "UID " + input.UID, Rows: cloudNumericRows(avatar, "level", "等级", "promote", "突破", "cons", "命座 / 星魂"), Note: "第三方保存的历史面板，仅作资料查看；缺少字段不补零，不代替本人官方实时面板。装备词条按固定 miao 资料解释。"}
		w := asObject(avatar["weapon"])
		if w != nil {
			weapon := weapons[asText(w["id"])]
			if weapon == "" {
				weapon = cloudText(w["name"])
			}
			if weapon == "" {
				weapon = "未识别装备"
			}
			v.Sections = append(v.Sections, Section{Title: weapon, Rows: cloudNumericRows(w, "level", "等级", "promote", "突破", "affix", "精炼 / 叠影")})
		}
		talents := asObject(avatar["talent"])
		rows := []Row{}
		for _, pair := range [][2]string{{"a", "普攻"}, {"e", "战技"}, {"q", "爆发 / 终结技"}, {"t", "天赋"}, {"me", "忆灵技"}, {"mt", "忆灵天赋"}} {
			value := talents[pair[0]]
			if obj := asObject(value); obj != nil {
				value = obj["level"]
			}
			if n := cloudNumber(value); n != "" {
				rows = append(rows, Row{Label: pair[1], Value: n})
			}
		}
		if len(rows) > 0 {
			v.Sections = append(v.Sections, Section{Title: "技能等级（云存档值）", Rows: rows})
		}
		artis := asObject(avatar["artis"])
		if artis == nil {
			v.Sections = append(v.Sections, Section{Title: "装备词条", Text: "此云面板未提供完整装备词条。"})
		}
		for slot := 1; slot <= 6; slot++ {
			if game.ID == "genshin" && slot == 6 {
				break
			}
			gear := asObject(artis[strconv.Itoa(slot)])
			if gear == nil {
				gear = asObject(artis["arti"+strconv.Itoa(slot)])
			}
			if gear != nil {
				v.Sections = append(v.Sections, cloudGearSection(game, slot, gear))
			}
		}
		panels = append(panels, CloudPanel{ID: id, Name: name, View: v})
	}
	if input.Mode == "rank_panel" && len(panels) == 0 {
		return CloudResult{}, gameError("cloud_invalid", "榜单面板没有返回所选角色。")
	}
	v := View{Title: game.Name + "公开云面板", Subtitle: "UID " + input.UID, Rows: []Row{{Label: "角色数量", Value: strconv.Itoa(len(panels))}}, Note: "来源：ark.ivny.cn。仅在当前查询中展示，不覆盖本地面板或账号资料。"}
	return CloudResult{View: &v, Panels: panels}, nil
}
func cloudGearSection(game Game, slot int, gear map[string]any) Section {
	section, _ := decodeCloudGear(game, slot, gear)
	return section
}
func decodeCloudGear(g Game, slot int, gear map[string]any) (Section, *PanelEquipment) {
	if g.Data == nil || g.Data.CloudGear == nil {
		return Section{Text: "装备解释资料不可用。"}, nil
	}
	game, d := g.ID, *g.Data.CloudGear
	key := asText(gear["name"])
	if game == "starrail" {
		key = asText(gear["id"])
	}
	item, known := d.Items[key]
	title := item.Name
	if title == "" {
		title = "未识别装备"
	}
	section := Section{Title: fmt.Sprintf("部位 %d · %s", slot, title), Rows: cloudNumericRows(gear, "level", "等级")}
	if !known || slot != item.Slot {
		section.Text = "固定资料未收录此装备，暂不能解释词条。"
		return section, nil
	}
	section.Rows = append(section.Rows, Row{Label: "套装", Value: item.Set})
	level, levelOK := cloudInteger(gear["level"], 0, 20)
	star := item.Star
	if game == "genshin" {
		star, _ = cloudInteger(gear["star"], 1, 5)
	}
	if !levelOK || star < 1 || star > 5 || (game == "genshin" && level > map[int]int{1: 4, 2: 4, 3: 12, 4: 16, 5: 20}[star]) || (game == "starrail" && (star < 2 || level > 3*star)) {
		section.Text = "装备等级或星级资料缺失，暂不能解释词条。"
		return section, nil
	}
	mainID := asText(gear["mainId"])
	mainKey := ""
	mainValue := 0.0
	validMain := false
	values := map[string]float64{}
	validSub := true
	subs, ok := gear["attrIds"].([]any)
	if !ok || len(subs) > 32 {
		validSub = false
		subs = nil
	}
	if game == "genshin" {
		mainKey = d.MainIDMap[mainID]
		attrKey := mainKey
		if slices.Contains([]string{"pyro", "hydro", "cryo", "electro", "anemo", "geo", "dendro"}, mainKey) {
			attrKey = "dmg"
		}
		attr, ok := d.AttrMap[attrKey]
		validMain = ok && mainKey != ""
		factor := 1.0
		if strings.HasSuffix(mainKey, "Plus") {
			factor = 2
		}
		mainValue = attr.Value * (1.2 + 0.34*float64(level)) * factor * map[int]float64{1: .21, 2: .36, 3: .6, 4: .9, 5: 1}[star]
		for _, raw := range subs {
			attr, ok := d.AttrIDMap[asText(raw)]
			if !ok {
				validSub = false
				continue
			}
			factor := 1.0
			if d.AttrMap[attr.Key].Format == "pct" {
				factor = 100
			}
			values[attr.Key] += attr.Value * factor
		}
	} else {
		mainKey = d.Meta.MainIdx[strconv.Itoa(slot)][mainID]
		table := d.Meta.StarData[strconv.Itoa(star)]
		attr, ok := table.Main[mainKey]
		validMain = ok
		mainValue = attr.Base + attr.Step*float64(level)
		for _, raw := range subs {
			obj := asObject(raw)
			if text, ok := raw.(string); ok {
				parts := strings.Split(text, ",")
				if len(parts) == 3 {
					obj = map[string]any{"id": parts[0], "count": parts[1], "step": parts[2]}
				}
			}
			attr, ok := table.Sub[asText(obj["id"])]
			count, cOK := cloudInteger(obj["count"], 1, 6)
			step, sOK := cloudInteger(obj["step"], 0, 12)
			if !ok || !cOK || !sOK || step > count*2 {
				validSub = false
				continue
			}
			values[attr.Key] += attr.Base*float64(count) + attr.Step*float64(step)
		}
	}
	if validMain {
		section.Rows = append(section.Rows, cloudStatRow(d, mainKey, mainValue, "主词条 · "))
	} else {
		section.Text = "主词条未收录，无法完整解释此装备。"
	}
	if validSub {
		keys := []string{}
		for key := range values {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			section.Rows = append(section.Rows, cloudStatRow(d, key, values[key], "副词条 · "))
		}
	} else {
		section.Text += " 副词条缺失或存在未知档位，未输出不完整合计。"
	}
	if !validMain || !validSub {
		return section, nil
	}
	makeStat := func(key, id string, value float64) PanelStat {
		row := cloudStatRow(d, key, value, "")
		text := strconv.FormatFloat(value, 'f', -1, 64)
		if strings.HasSuffix(row.Value, "%") {
			text += "%"
		}
		return PanelStat{ID: id, Key: key, Name: row.Label, Value: text}
	}
	panel := &PanelEquipment{ID: asText(gear["id"]), Name: item.Name, Slot: slot, Level: level, Rarity: strconv.Itoa(star), SetName: item.Set, Complete: true, Main: []PanelStat{makeStat(mainKey, "cloud-main:"+mainID, mainValue)}, Sub: []PanelStat{}}
	keys := []string{}
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		panel.Sub = append(panel.Sub, makeStat(key, "cloud-sub:"+key, values[key]))
	}
	return section, panel
}
func cloudInteger(v any, min, max int) (int, bool) {
	s := cloudNumber(v)
	n, err := strconv.Atoi(s)
	return n, err == nil && n >= min && n <= max
}
func cloudStatRow(d cloudGearData, key string, value float64, prefix string) Row {
	attr := d.AttrMap[key]
	label := attr.Title
	format := attr.Format
	if label == "" {
		label = map[string]string{"pyro": "火伤加成", "fire": "火伤加成", "hydro": "水伤加成", "cryo": "冰伤加成", "ice": "冰伤加成", "electro": "雷伤加成", "elec": "雷伤加成", "anemo": "风伤加成", "wind": "风伤加成", "geo": "岩伤加成", "dendro": "草伤加成", "phy": "物伤加成", "quantum": "量子伤害", "imaginary": "虚数伤害"}[key]
		format = "pct"
	}
	if label == "" {
		label = key
	}
	display := strconv.FormatFloat(value, 'f', 2, 64)
	if format == "pct" {
		display += "%"
	}
	return Row{Label: prefix + label, Value: display}
}
