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
	SkillType    int    `json:"skill_type,omitempty"`
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

// zzzElements are the element keys of the official element_type codes.
var zzzElements = map[int]string{200: "physical", 201: "fire", 202: "ice", 203: "lightning", 204: "wind", 205: "ether", 300: "lumiflux"}

// NormalizePanels reads the agents of an official avatar_list, from the
// account's avatar info or from Enka converted to it, into panels.
func NormalizePanels(result QueryResult, catalog Catalog) []CharacterPanel {
	panels := []CharacterPanel{}
	for _, raw := range asList(result.Data["avatar_list"]) {
		item := asObject(raw)
		id := asText(item["id"])
		if id == "" {
			continue
		}
		entry, hasEntry := catalog.Get(id)
		name := firstText(item, "full_name_mi18n", "name_mi18n")
		if name == "" && hasEntry {
			name = entry.Name
		}
		if name == "" {
			name = "角色 " + id
		}
		panel := CharacterPanel{ID: id, Name: plainGameText(name), Level: number(item["level"]), Rank: number(item["rank"]), Profession: number(item["avatar_profession"]), Element: zzzElements[number(item["element_type"])], Source: "mihoyo", Stats: panelStats(asList(item["properties"]), "final"), Equipment: []PanelEquipment{}, Skills: []PanelSkill{}, Ranks: []PanelSkill{}, Official: item}
		if panel.Element == "" && hasEntry {
			panel.Element = entry.Element
		}
		if value, exists := item["promote_level"]; exists {
			n := number(value)
			panel.Promote = &n
		}
		if value, exists := item["sub_element_type"]; exists {
			n := number(value)
			panel.SubElement = &n
		}
		_, panel.RankKnown = item["rank"]
		_, panel.WeaponKnown = item["weapon"]
		_, panel.EquipmentKnown = item["equip"]
		if weapon := asObject(item["weapon"]); weapon != nil && firstText(weapon, "name", "id") != "" {
			w := PanelEquipment{ID: asText(weapon["id"]), Name: plainGameText(asText(weapon["name"])), Level: number(weapon["level"]), Rarity: asText(weapon["rarity"]), Refinement: number(weapon["star"]), Main: panelStats(asList(weapon["main_properties"]), "base"), Sub: panelStats(asList(weapon["properties"]), "base")}
			if value, exists := weapon["promote_level"]; exists {
				n := number(value)
				w.Promote = &n
			}
			panel.Weapon = &w
		}
		for _, raw := range asList(item["equip"]) {
			gear := asObject(raw)
			main := panelStats(asList(gear["main_properties"]), "base")
			_, subPresent := gear["properties"]
			panel.Equipment = append(panel.Equipment, PanelEquipment{ID: asText(gear["id"]), Name: plainGameText(asText(gear["name"])), Slot: number(gear["equipment_type"]), Level: number(gear["level"]), Rarity: asText(gear["rarity"]), SetName: asText(asObject(gear["equip_suit"])["name"]), Main: main, Sub: panelStats(asList(gear["properties"]), "base"), Complete: subPresent && len(main) > 0})
		}
		// A skill's name and description are in its items, as 普通攻击：… titles.
		for i, raw := range asList(item["skills"]) {
			skill := asObject(raw)
			name, description := "", ""
			for _, raw := range asList(skill["items"]) {
				stage := asObject(raw)
				if name == "" {
					name = asText(stage["title"])
				}
				if description == "" {
					description = asText(stage["text"])
				}
			}
			if name == "" {
				name = "技能 " + strconv.Itoa(i+1)
			}
			_, hasSkillType := skill["skill_type"]
			panel.Skills = append(panel.Skills, PanelSkill{HasSkillType: hasSkillType, SkillType: number(skill["skill_type"]), Name: plainGameText(name), Level: number(skill["level"]), Active: true, Description: plainGameText(description)})
		}
		for _, raw := range asList(item["ranks"]) {
			rank := asObject(raw)
			active, _ := rank["is_unlocked"].(bool)
			panel.Ranks = append(panel.Ranks, PanelSkill{Name: plainGameText(asText(rank["name"])), Active: active, Description: plainGameText(asText(rank["desc"]))})
		}
		if plan := asObject(item["equip_plan_info"]); plan != nil {
			panel.OfficialScore = asText(plan["equip_rating_score"])
			if rating := asText(plan["equip_rating"]); rating != "" {
				panel.OfficialScore += " · " + rating
			}
		}
		panels = append(panels, panel)
	}
	return panels
}

// panelStats reads official properties, showing each by its value key
// ("final" for the agent, "base" for W-Engines and drive discs). A property
// without that value is left out rather than shown as zero.
func panelStats(raw []any, key string) []PanelStat {
	result := []PanelStat{}
	for _, value := range raw {
		item := asObject(value)
		id := asText(item["property_id"])
		display := asText(item[key])
		if id == "" || display == "" {
			continue
		}
		name := asText(item["property_name"])
		if name == "" {
			name = "属性 " + id
		}
		result = append(result, PanelStat{ID: id, Key: id, Name: plainGameText(name), Value: display, Base: asText(item["base"]), Added: asText(item["add"])})
	}
	return result
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
	v.Rows = append(v.Rows, Row{Label: "等级", Value: strconv.Itoa(p.Level)}, Row{Label: "意象影画", Value: strconv.Itoa(p.Rank)})
	rows := []Row{}
	for _, stat := range p.Stats {
		rows = append(rows, Row{Label: stat.Name, Value: stat.Value})
	}
	v.Sections = append(v.Sections, Section{Title: "角色属性", Rows: rows})
	if p.Weapon != nil {
		w := p.Weapon
		rows = []Row{{Label: w.Name, Value: "等级 " + strconv.Itoa(w.Level) + " · 星级 " + strconv.Itoa(w.Refinement)}}
		for _, stat := range append(append([]PanelStat{}, w.Main...), w.Sub...) {
			rows = append(rows, Row{Label: stat.Name, Value: stat.Value})
		}
		v.Sections = append(v.Sections, Section{Title: "音擎", Rows: rows})
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
		v.Sections = append(v.Sections, Section{Title: "技能", Rows: rows})
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
		v.Sections = append(v.Sections, Section{Title: "影画", Rows: rows})
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
