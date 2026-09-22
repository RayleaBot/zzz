package app

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

type PanelStat struct {
	ID    string `json:"id"`
	Key   string `json:"key,omitempty"`
	Name  string `json:"name"`
	Value string `json:"value"`
	Base  string `json:"base,omitempty"`
	Added string `json:"added,omitempty"`
	// Times is how often an equipment substat was upgraded.
	Times int `json:"times,omitempty"`
}
type PanelEquipment struct {
	Promote    *int            `json:"promote,omitempty"`
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Slot       int             `json:"slot"`
	Level      int             `json:"level"`
	Rarity     string          `json:"rarity"`
	Refinement int             `json:"refinement,omitempty"`
	SetName    string          `json:"set_name,omitempty"`
	Main       []PanelStat     `json:"main"`
	Sub        []PanelStat     `json:"sub"`
	Complete   bool            `json:"complete"`
	Score      *EquipmentScore `json:"score,omitempty"`
}
type PanelSkill struct {
	HasSkillType bool   `json:"has_skill_type,omitempty"`
	Kind         string `json:"kind,omitempty"`
	PointType    int    `json:"point_type,omitempty"`
	SkillType    int    `json:"skill_type,omitempty"`
	ID           string `json:"id,omitempty"`
	ExtraLevel   int    `json:"extra_level,omitempty"`
	Name         string `json:"name"`
	Level        int    `json:"level,omitempty"`
	Active       bool   `json:"active"`
	Description  string `json:"description,omitempty"`
}
type CharacterPanel struct {
	EquipmentKnown  bool             `json:"equipment_known"`
	SubElement      *int             `json:"sub_element,omitempty"`
	WeaponKnown     bool             `json:"weapon_known"`
	Promote         *int             `json:"promote,omitempty"`
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Level           int              `json:"level"`
	Rank            int              `json:"rank"`
	RankKnown       bool             `json:"rank_known"`
	Profession      int              `json:"profession,omitempty"`
	Element         string           `json:"element"`
	Source          string           `json:"source"`
	Stats           []PanelStat      `json:"stats"`
	Weapon          *PanelEquipment  `json:"weapon,omitempty"`
	Equipment       []PanelEquipment `json:"equipment"`
	Skills          []PanelSkill     `json:"skills"`
	Ranks           []PanelSkill     `json:"ranks"`
	OfficialScore   string           `json:"official_score,omitempty"`
	ScoreRule       string           `json:"score_rule,omitempty"`
	ScoreNote       string           `json:"score_note,omitempty"`
	TotalScore      *float64         `json:"total_score,omitempty"`
	ScoredEquipment int              `json:"scored_equipment,omitempty"`
	// ScoreDetail is the upstream scoring detail for the panel image; it is
	// not stored with the panel.
	ScoreDetail *ScoreDetail `json:"-"`
	// Official is the official character entry the panel was read from, for
	// images that show fields the panel does not keep. It is not stored.
	Official map[string]any `json:"-"`
	// UpdatedAtMS is when the kept panel was saved; it is not stored with the
	// panel.
	UpdatedAtMS int64 `json:"-"`
}

var markup = regexp.MustCompile(`<[^>]*>`)

func plainGameText(value string) string {
	value = strings.ReplaceAll(strings.ReplaceAll(value, "<br>", "\n"), "<br/>", "\n")
	return html.UnescapeString(markup.ReplaceAllString(value, ""))
}

var gsStatKeys = map[string]string{"2": "hpPlus", "3": "hp", "5": "atkPlus", "6": "atk", "8": "defPlus", "9": "def", "20": "cpct", "22": "cdmg", "23": "recharge", "26": "heal", "28": "mastery", "30": "phy", "40": "pyro", "41": "electro", "42": "hydro", "43": "dendro", "44": "anemo", "45": "geo", "46": "cryo"}
var srStatKeys = map[string]string{"1": "hpPlus", "2": "atkPlus", "3": "defPlus", "4": "speed", "5": "cpct", "6": "cdmg", "7": "heal", "9": "recharge", "10": "effPct", "11": "effDef", "12": "physical", "14": "fire", "16": "ice", "18": "lightning", "20": "wind", "22": "quantum", "24": "imaginary", "27": "hp", "29": "atk", "31": "def", "32": "hpPlus", "33": "atkPlus", "34": "defPlus", "51": "speed", "52": "cpct", "53": "cdmg", "54": "recharge", "55": "heal", "56": "effPct", "57": "effDef", "58": "stance", "59": "stance"}

func panelStats(game string, raw []any, metadata map[string]any, defaultValue string) []PanelStat {
	result := []PanelStat{}
	for _, value := range raw {
		item := asObject(value)
		id := firstText(item, "property_type", "property_id")
		if id == "" {
			continue
		}
		name := firstText(item, "property_name", "name")
		if name == "" {
			name = firstText(asObject(metadata[id]), "name", "property_name_relic", "filter_name")
		}
		if name == "" {
			name = "属性 " + id
		}
		display := firstText(item, defaultValue, "value", "final", "base")
		if display == "" {
			continue
		}
		key := id
		if game == "genshin" {
			key = gsStatKeys[id]
		}
		if game == "starrail" {
			key = srStatKeys[id]
			// The record API has used both ID families for flat and percent
			// relic stats. The explicit unit keeps both representations correct.
			baseKey := strings.TrimSuffix(key, "Plus")
			if baseKey == "hp" || baseKey == "atk" || baseKey == "def" {
				key = baseKey
				if !strings.HasSuffix(strings.TrimSpace(display), "%") {
					key += "Plus"
				}
			}
		}
		result = append(result, PanelStat{ID: id, Key: key, Name: plainGameText(name), Value: display, Base: asText(item["base"]), Added: asText(item["add"]), Times: number(item["times"])})
	}
	return result
}
func oneStat(value any) []any {
	if item := asObject(value); item != nil {
		return []any{item}
	}
	return nil
}

func NormalizePanels(game string, result QueryResult, catalog Catalog) []CharacterPanel {
	data := result.Data
	metadata := asObject(data["property_map"])
	if game == "starrail" {
		metadata = asObject(data["property_info"])
	}
	list := asList(data["list"])
	if list == nil {
		list = asList(data["avatar_list"])
	}
	if list == nil {
		list = asList(data["avatars"])
	}
	panels := []CharacterPanel{}
	for _, raw := range list {
		item := asObject(raw)
		base := asObject(item["base"])
		if base == nil {
			base = item
		}
		id := asText(base["id"])
		if id == "" {
			continue
		}
		name := firstText(base, "name", "full_name_mi18n", "name_mi18n")
		entry, hasEntry := catalog.Get(id)
		if name == "" && hasEntry {
			name = entry.Name
		}
		if name == "" {
			name = "角色 " + id
		}
		panel := CharacterPanel{ID: id, Name: plainGameText(name), Level: number(base["level"]), Rank: number(base["rank"]), Element: asText(base["element"]), Source: "mihoyo", Stats: []PanelStat{}, Equipment: []PanelEquipment{}, Skills: []PanelSkill{}, Ranks: []PanelSkill{}, Official: item}
		_, panel.EquipmentKnown = item["relics"]
		if game == "starrail" {
			_, known := item["ornaments"]
			panel.EquipmentKnown = panel.EquipmentKnown && known
		}
		if value, exists := base["promote_level"]; exists {
			n := number(value)
			panel.Promote = &n
		}
		if game == "genshin" {
			panel.Rank = number(base["actived_constellation_num"])
			_, panel.RankKnown = base["actived_constellation_num"]
		} else {
			_, panel.RankKnown = base["rank"]
		}
		if panel.Element == "" && hasEntry {
			panel.Element = entry.Element
		}
		if game == "zzz" {
			panel.Profession = number(item["avatar_profession"])
			_, panel.EquipmentKnown = item["equip"]
			if v, ok := item["sub_element_type"]; ok {
				n := number(v)
				panel.SubElement = &n
			}
			if element := map[int]string{200: "physical", 201: "fire", 202: "ice", 203: "lightning", 204: "wind", 205: "ether", 300: "lumiflux"}[number(item["element_type"])]; element != "" {
				panel.Element = element
			}
		}
		seen := map[string]bool{}
		for _, field := range []string{"base_properties", "extra_properties", "element_properties", "properties"} {
			for _, stat := range panelStats(game, asList(item[field]), metadata, "final") {
				if !seen[stat.ID] {
					panel.Stats = append(panel.Stats, stat)
					seen[stat.ID] = true
				}
			}
		}
		weapon := asObject(item["weapon"])
		_, panel.WeaponKnown = item["weapon"]
		if game == "starrail" {
			weapon = asObject(item["equip"])
			_, panel.WeaponKnown = item["equip"]
		}
		if weapon == nil {
			weapon = asObject(base["weapon"])
			if _, ok := base["weapon"]; ok {
				panel.WeaponKnown = true
			}
		}
		if weapon != nil && firstText(weapon, "name", "id") != "" {
			w := PanelEquipment{ID: asText(weapon["id"]), Name: plainGameText(asText(weapon["name"])), Level: number(weapon["level"]), Rarity: asText(weapon["rarity"]), Refinement: number(weapon["affix_level"])}
			if value, exists := weapon["promote_level"]; exists {
				n := number(value)
				w.Promote = &n
			}
			if game == "starrail" {
				w.Refinement = number(weapon["rank"])
			}
			if game == "zzz" {
				w.Refinement = number(weapon["star"])
			}
			w.Main = panelStats(game, oneStat(weapon["main_property"]), metadata, "final")
			w.Sub = panelStats(game, oneStat(weapon["sub_property"]), metadata, "final")
			if game == "zzz" {
				w.Main = panelStats(game, asList(weapon["main_properties"]), metadata, "base")
				w.Sub = panelStats(game, asList(weapon["properties"]), metadata, "base")
			}
			panel.Weapon = &w
		}
		gearFields := []string{"relics", "ornaments"}
		if game == "zzz" {
			gearFields = []string{"equip"}
		}
		for _, field := range gearFields {
			for _, raw := range asList(item[field]) {
				gear := asObject(raw)
				equipment := PanelEquipment{ID: asText(gear["id"]), Name: plainGameText(asText(gear["name"])), Slot: number(gear["pos"]), Level: number(gear["level"]), Rarity: asText(gear["rarity"]), SetName: asText(asObject(gear["set"])["name"])}
				if equipment.SetName == "" {
					equipment.SetName = catalog.ArtifactSets[equipment.ID]
					if equipment.SetName == "" {
						equipment.SetName = catalog.ArtifactSets[equipment.Name]
					}
				}
				equipment.Main = panelStats(game, oneStat(gear["main_property"]), metadata, "value")
				subKey := "sub_property_list"
				if game == "starrail" {
					subKey = "properties"
				}
				equipment.Sub = panelStats(game, asList(gear[subKey]), metadata, "value")
				_, subPresent := gear[subKey]
				equipment.Complete = subPresent && len(equipment.Main) > 0
				if game == "zzz" {
					equipment.Slot = number(gear["equipment_type"])
					equipment.SetName = asText(asObject(gear["equip_suit"])["name"])
					equipment.Main = panelStats(game, asList(gear["main_properties"]), metadata, "base")
					equipment.Sub = panelStats(game, asList(gear["properties"]), metadata, "base")
					_, subPresent = gear["properties"]
					equipment.Complete = subPresent && len(equipment.Main) > 0
				}
				panel.Equipment = append(panel.Equipment, equipment)
			}
		}
		skills := append(append([]any{}, asList(item["skills"])...), asList(item["servant_skills"])...)
		skills = append(skills, asList(asObject(item["servant_detail"])["servant_skills"])...)
		for i, raw := range skills {
			skill := asObject(raw)
			name := asText(skill["name"])
			description := firstText(skill, "desc", "description")
			if name == "" {
				for _, raw := range append(asList(skill["items"]), asList(skill["skill_stages"])...) {
					stage := asObject(raw)
					if name == "" {
						name = firstText(stage, "name", "title")
					}
					if description == "" {
						description = firstText(stage, "desc", "description", "text")
					}
				}
			}
			if name == "" {
				name = fmtSkillLabel(game, skill, i)
			}
			active := true
			for _, key := range []string{"is_unlock", "is_activated"} {
				if flag, ok := skill[key].(bool); ok {
					active = flag
				}
			}
			_, hasSkillType := skill["skill_type"]
			panel.Skills = append(panel.Skills, PanelSkill{HasSkillType: hasSkillType, Kind: firstText(skill, "remake"), PointType: number(skill["point_type"]), SkillType: number(skill["skill_type"]), ID: firstText(skill, "skill_id", "point_id", "id"), ExtraLevel: number(skill["extra_level"]), Name: plainGameText(name), Level: number(skill["level"]), Active: active, Description: plainGameText(description)})
		}
		ranks := asList(item["ranks"])
		if game == "genshin" {
			ranks = asList(item["constellations"])
		}
		for _, raw := range ranks {
			rank := asObject(raw)
			active, _ := rank["is_unlocked"].(bool)
			if game == "genshin" {
				active, _ = rank["is_actived"].(bool)
			}
			panel.Ranks = append(panel.Ranks, PanelSkill{Name: plainGameText(asText(rank["name"])), Active: active, Description: plainGameText(firstText(rank, "desc", "effect"))})
		}
		if plan := asObject(item["equip_plan_info"]); plan != nil {
			panel.OfficialScore = firstText(plan, "equip_rating_score")
			if rating := asText(plan["equip_rating"]); rating != "" {
				panel.OfficialScore += " · " + rating
			}
		}
		panels = append(panels, panel)
	}
	return panels
}
func fmtSkillLabel(game string, item map[string]any, index int) string {
	if game == "starrail" {
		return "行迹 " + firstText(item, "point_id")
	}
	return "技能 " + strconv.Itoa(index+1)
}

func PanelView(game Game, panels []CharacterPanel, uid string) View {
	v := View{Title: game.Name + "角色面板", Subtitle: uid, Rows: []Row{}, Sections: []Section{}, Note: "属性与装备来自米游社；未返回的数值不作推算。"}
	if len(panels) == 0 {
		v.Note = "此账号未返回所选角色的面板。"
		return v
	}
	if len(panels) > 1 {
		for _, p := range panels {
			v.Rows = append(v.Rows, Row{Label: p.Name, Value: p.ID + " · 等级 " + strconv.Itoa(p.Level) + " · " + strconv.Itoa(p.Rank) + " 阶"})
		}
		v.Note = "请指定角色名称或 ID 查看完整面板。"
		return v
	}
	p := panels[0]
	v.Title = p.Name + " · 角色面板"
	v.Rows = append(v.Rows, Row{Label: "等级", Value: strconv.Itoa(p.Level)}, Row{Label: map[string]string{"genshin": "命之座", "starrail": "星魂", "zzz": "意象影画"}[game.ID], Value: strconv.Itoa(p.Rank)})
	rows := []Row{}
	for _, stat := range p.Stats {
		rows = append(rows, Row{Label: stat.Name, Value: stat.Value})
	}
	v.Sections = append(v.Sections, Section{Title: "角色属性", Rows: rows})
	if p.Weapon != nil {
		w := p.Weapon
		rows = []Row{{Label: w.Name, Value: "等级 " + strconv.Itoa(w.Level) + " · 精炼/叠影 " + strconv.Itoa(w.Refinement)}}
		for _, stat := range append(append([]PanelStat{}, w.Main...), w.Sub...) {
			rows = append(rows, Row{Label: stat.Name, Value: stat.Value})
		}
		v.Sections = append(v.Sections, Section{Title: map[string]string{"genshin": "武器", "starrail": "光锥", "zzz": "音擎"}[game.ID], Rows: rows})
	}
	for _, equipment := range p.Equipment {
		rows = []Row{{Label: "等级", Value: strconv.Itoa(equipment.Level)}}
		for _, stat := range equipment.Main {
			rows = append(rows, Row{Label: "主属性 · " + stat.Name, Value: stat.Value})
		}
		for _, stat := range equipment.Sub {
			rows = append(rows, Row{Label: stat.Name, Value: stat.Value})
		}
		if equipment.Score != nil {
			rows = append(rows, Row{Label: "评分", Value: strconv.FormatFloat(equipment.Score.Value, 'f', 1, 64) + " · " + equipment.Score.Grade})
		}
		v.Sections = append(v.Sections, Section{Title: strconv.Itoa(equipment.Slot) + "号位 · " + equipment.Name, Rows: rows})
	}
	rows = []Row{}
	for _, skill := range p.Skills {
		status := "未解锁"
		if skill.Active {
			status = "等级 " + strconv.Itoa(skill.Level)
		}
		rows = append(rows, Row{Label: skill.Name, Value: status})
	}
	if len(rows) > 0 {
		v.Sections = append(v.Sections, Section{Title: "技能与行迹", Rows: rows})
	}
	rows = []Row{}
	for _, rank := range p.Ranks {
		status := "未解锁"
		if rank.Active {
			status = "已解锁"
		}
		rows = append(rows, Row{Label: rank.Name, Value: status})
	}
	if len(rows) > 0 {
		v.Sections = append(v.Sections, Section{Title: "命座 / 星魂 / 影画", Rows: rows})
	}
	if p.OfficialScore != "" {
		v.Rows = append(v.Rows, Row{Label: "官方配装评分", Value: p.OfficialScore})
	}
	if p.TotalScore != nil {
		v.Rows = append(v.Rows, Row{Label: "装备评分合计", Value: strconv.FormatFloat(*p.TotalScore, 'f', 1, 64) + " · 已评分 " + strconv.Itoa(p.ScoredEquipment) + " / " + strconv.Itoa(len(p.Equipment)) + " 件"})
	}
	if p.ScoreRule != "" {
		v.Note += "\n评分规则：" + p.ScoreRule + "。"
	}
	if p.ScoreNote != "" {
		v.Note += "\n" + p.ScoreNote
	}
	return v
}
