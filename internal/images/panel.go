package images

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/app"
)

// propertyClasses are ZZZ-Plugin's icon classes by the first three digits of
// a property ID.
var propertyClasses = map[string]string{
	"111": "hpmax", "121": "attack", "131": "def", "122": "breakstun", "201": "crit", "211": "critdam",
	"314": "elementabnormalpower", "312": "elementmystery", "231": "penratio", "232": "penvalue",
	"305": "sprecover", "310": "spgetratio", "115": "spmax", "315": "physdmg", "316": "fire", "317": "ice",
	"318": "thunder", "319": "dungeonbuffether", "323": "wind", "25": "sharpnessaccumulate", "28": "laceration",
}

func propertyClass(id any) string {
	text := app.Text(id)
	return propertyClasses[text[:min(3, len(text))]]
}

// panelRow is one character property row of the upstream card: its icon, the
// property name it reads, and the weight ID that colours its label.
type panelRow struct{ icon, name, weight string }

var (
	rowsBefore = []panelRow{{"hpmax", "生命值", "11102"}}
	rowsMiddle = []panelRow{
		{"def", "防御力", "13102"}, {"breakstun", "冲击力", "12202"}, {"crit", "暴击率", "20103"},
		{"critdam", "暴击伤害", "21103"}, {"elementabnormalpower", "异常掌控", "31402"}, {"elementmystery", "异常精通", "31203"},
	}
)

// Panel draws a single agent the way ZZZ-Plugin's panel card does: the agent
// portrait with skill levels, rank, level and Mindscape, the property list,
// the W-Engine, the drive disc rating with substat totals, each disc, and the
// buffs the agent's rule shows in the panel with every skill's damage.
func Panel(context app.ImageContext, image app.PanelImage) (app.Image, bool) {
	official := image.Panel.Official
	if official == nil {
		return app.Image{}, false
	}
	var detail zzzDetail
	if image.Panel.ScoreDetail != nil {
		_ = json.Unmarshal(image.Panel.ScoreDetail.Raw, &detail)
	}
	card := newAgentCard(context)
	data := card.basic(official, image.UID, image.Portrait, detail.Weights)
	if image.Panel.ScoreDetail != nil {
		data["rating"] = rating(detail, image.Panel.ScoreDetail)
	}
	data["discs"] = discs(official, detail, card.maps, card.fetch)
	if result := image.Damage; len(result.Damages) > 0 || len(result.PanelBuffs) > 0 {
		buffs := []any{}
		for _, buff := range result.PanelBuffs {
			value := buffValue(buff.Value)
			if buff.Max != 0 {
				value += "/" + buffValue(buff.Max)
			}
			buffs = append(buffs, map[string]any{"name": buff.Name, "type": buff.Type, "value": value})
		}
		rows := []any{}
		for index, damage := range result.Damages {
			rows = append(rows, damageRow(index, damage))
		}
		data["damage"] = map[string]any{"buffs": buffs, "rows": rows, "level": app.Int(official["level"]), "hint": context.Game.Prefix + app.Text(official["name_mi18n"]) + "伤害"}
	}
	return app.Image{Template: "panel", Data: data, Resources: card.resources}, true
}

// agentCard is what ZZZ-Plugin's panel card and damage page share: the
// upstream ID tables, the card's pictures and the agent's own ones fetched on
// demand.
type agentCard struct {
	context   app.ImageContext
	maps      zzzMaps
	resources []rayleabot.RenderImageResource
}

func newAgentCard(context app.ImageContext) *agentCard {
	card := &agentCard{context: context, maps: readMaps(context)}
	for _, item := range append(append([][2]string{}, commonArtwork...), panelArtwork...) {
		if resource, ok := context.ArtworkResource(item[0], "zzz-plugin", item[1]); ok {
			card.resources = append(card.resources, resource)
		}
	}
	card.resources = append(card.resources, fontResources(context)...)
	return card
}

func (c *agentCard) fetch(id, name string) bool {
	resource, ok := c.context.FetchArtworkResource(id, "zzzerouid", name)
	if ok {
		c.resources = append(c.resources, resource)
	}
	return ok
}

// basic is the top of the card: the UID, the portrait with the skill levels,
// rarity, element, name, level and Mindscape, the properties with labels
// coloured by the score weights, and the W-Engine. A custom picture uploaded
// for the character replaces the portrait, as ZZZ-Plugin's 上传面板图.
func (c *agentCard) basic(official map[string]any, uid, portrait string, weights map[string]float64) map[string]any {
	id := app.Text(official["id"])
	if portrait != "" {
		c.resources = append(c.resources, rayleabot.RenderImageResource{ID: "role-icon", Path: portrait})
	} else if sprite := c.maps.partners[id].SpriteID; sprite != "" {
		c.fetch("role-icon", "role/IconRole"+sprite+".png")
	}
	label := func(weightID string) string {
		switch w := weights[weightID]; {
		case w == 0:
			return ""
		case w == 1:
			return "yellow"
		case w >= 0.75:
			return "blue"
		case w >= 0.5:
			return "white"
		}
		return ""
	}

	skills, _ := official["skills"].([]any)
	levels := []any{}
	for _, index := range []int{0, 2, 5, 1, 3, 4} {
		level := ""
		if index < len(skills) {
			skill, _ := skills[index].(map[string]any)
			level = app.Text(skill["level"])
		}
		levels = append(levels, level)
	}

	data := map[string]any{
		"uid": uid, "rarity": app.Text(official["rarity"]), "name": app.Text(official["full_name_mi18n"]),
		"level": app.Int(official["level"]), "rank": app.Int(official["rank"]), "skills": levels,
		"sub_element": c.maps.elementName(official["element_type"], official["sub_element_type"]),
		"properties":  properties(official, c.maps, label),
	}
	if weapon, _ := official["weapon"].(map[string]any); weapon != nil {
		if code := c.maps.weapons[app.Text(weapon["id"])].CodeName; code != "" {
			c.fetch("weapon-icon", "weapon/"+code+"_High.png")
		}
		data["weapon"] = map[string]any{
			"rarity": app.Text(weapon["rarity"]), "name": app.Text(weapon["name"]), "star": app.Int(weapon["star"]), "level": app.Int(weapon["level"]),
			"main": weaponProperties(weapon["main_properties"]), "sub": weaponProperties(weapon["properties"]),
		}
	}
	return data
}

// properties lists the character property rows in upstream's order; the
// Rupture and Armorer professions swap some rows for their own properties.
func properties(official map[string]any, maps zzzMaps, label func(string) string) []any {
	byName := map[string]map[string]any{}
	list, _ := official["properties"].([]any)
	for _, raw := range list {
		property, _ := raw.(map[string]any)
		byName[app.Text(property["property_name"])] = property
	}
	profession := app.Int(official["avatar_profession"])
	rows := append([]panelRow{}, rowsBefore...)
	if profession == 7 {
		rows = append(rows, panelRow{"laceration", "锐暴伤害", "yellow"})
	} else {
		rows = append(rows, panelRow{"attack", "攻击力", "12102"})
	}
	rows = append(rows, rowsMiddle...)
	switch profession {
	case 6:
		rows = append(rows, panelRow{"sheerforce", "贯穿力", "yellow"}, panelRow{"adrenalineaccumulate", "闪能自动累积", "white"})
	case 7:
		rows = append(rows, panelRow{"penratio", "穿透率", "23103"}, panelRow{"sharpnessaccumulate", "锐能自动累积", "white"})
	default:
		rows = append(rows, panelRow{"penratio", "穿透率", "23103"}, panelRow{"sprecover", "能量自动回复", "30502"})
	}
	result := []any{}
	add := func(icon, iconClass, labelClass, name string, property map[string]any) {
		base, added := app.Text(property["base"]), app.Text(property["add"])
		final := app.Text(property["final"])
		if property == nil {
			final = "0"
		}
		result = append(result, map[string]any{"icon": icon, "icon_class": iconClass, "label": labelClass, "name": name,
			"detail": base != "" && added != "" && !strings.HasPrefix(added, "0"), "base": base, "add": added, "final": final})
	}
	for _, row := range rows {
		labelClass := row.weight
		if labelClass != "yellow" && labelClass != "white" {
			labelClass = label(row.weight)
		}
		add("prop-icon", row.icon, labelClass, row.name, byName[row.name])
	}
	// The element damage bonus row uses the element's own property.
	if propertyID := maps.elementProperty(official["element_type"]); propertyID != "" {
		for _, raw := range list {
			property, _ := raw.(map[string]any)
			if app.Text(property["property_id"]) == propertyID {
				add("element-icon", maps.elementName(official["element_type"], 0), label(propertyID+"03"), app.Text(property["property_name"]), property)
			}
		}
	}
	return result
}

func weaponProperties(value any) []any {
	list, _ := value.([]any)
	result := []any{}
	for _, raw := range list {
		property, _ := raw.(map[string]any)
		result = append(result, map[string]any{"class": propertyClass(property["property_id"]), "name": app.Text(property["property_name"]), "base": app.Text(property["base"])})
	}
	return result
}

// zzzDetail is the ZZZ-Plugin part of the scoring result.
type zzzDetail struct {
	Title   string             `json:"title"`
	Grade   string             `json:"grade"`
	Total   float64            `json:"total"`
	Weights map[string]float64 `json:"weights"`
	Stats   []struct {
		Name   string  `json:"name"`
		Weight float64 `json:"weight"`
		Value  string  `json:"value"`
		Count  int     `json:"count"`
	} `json:"stats"`
	Pieces []struct {
		Slot  int     `json:"slot"`
		Score float64 `json:"score"`
		Grade string  `json:"grade"`
		Props []struct {
			ID     int     `json:"id"`
			Count  int     `json:"count"`
			Weight float64 `json:"weight"`
		} `json:"props"`
	} `json:"pieces"`
}

// rating is the drive disc score block: the total and its grade, the rule,
// nine substat totals, and the useful and effective roll counts.
func rating(detail zzzDetail, score *app.ScoreDetail) map[string]any {
	stats := []any{}
	useful, effective := 0, 0.0
	for _, stat := range detail.Stats[:min(len(detail.Stats), 9)] {
		class := "useless"
		switch {
		case stat.Weight == 1:
			class = "great"
		case stat.Weight >= 0.75:
			class = "good"
		case stat.Weight > 0:
			class = "useful"
		}
		if stat.Weight > 0 {
			useful += stat.Count
		}
		effective += float64(stat.Count) * stat.Weight
		stats = append(stats, map[string]any{"class": class, "count": stat.Count, "name": stat.Name, "value": stat.Value})
	}
	for len(stats) < 9 {
		stats = append(stats, map[string]any{})
	}
	rule := score.Title
	if rule == "" {
		rule = "默认"
	}
	return map[string]any{"score": fmt.Sprintf("%.2f", detail.Total), "grade": detail.Grade, "rule": rule, "stats": stats,
		"useful": useful, "effective": fmt.Sprintf("%.2f", effective)}
}

// discs lists the six drive disc slots with their scores and properties;
// substat labels show the weight in quarters as upstream's hit classes do.
func discs(official map[string]any, detail zzzDetail, maps zzzMaps, fetch func(id, name string) bool) []any {
	bySlot := map[int]map[string]any{}
	equipment, _ := official["equip"].([]any)
	for _, raw := range equipment {
		disc, _ := raw.(map[string]any)
		bySlot[app.Int(disc["equipment_type"])] = disc
	}
	pieces := map[int]int{}
	for index, piece := range detail.Pieces {
		pieces[piece.Slot] = index
	}
	result := []any{}
	for slot := 1; slot <= 6; slot++ {
		disc := bySlot[slot]
		if disc == nil {
			result = append(result, map[string]any{"empty": true})
			continue
		}
		icon := ""
		if id := app.Text(disc["id"]); len(id) == 5 {
			if sprite := maps.suits[id[:3]+"00"].SpriteFile; sprite != "" {
				icon = "suit-" + strconv.Itoa(slot)
				if !fetch(icon, "suit/"+sprite+".png") {
					icon = ""
				}
			}
		}
		item := map[string]any{"icon": icon, "level": app.Int(disc["level"]), "rarity": app.Text(disc["rarity"]), "name": app.Text(disc["name"])}
		index, scored := pieces[slot]
		if scored {
			piece := detail.Pieces[index]
			item["score"], item["grade"] = fmt.Sprintf("%.2f", piece.Score), piece.Grade
		}
		mains := []any{}
		list, _ := disc["main_properties"].([]any)
		for _, raw := range list {
			property, _ := raw.(map[string]any)
			name := app.Text(property["property_name"])
			switch {
			case strings.Contains(name, "属性伤害加成"):
				name = strings.ReplaceAll(name, "属性伤害加成", "伤加成")
			case name == "能量自动回复":
				name = "能量回复"
			}
			mains = append(mains, map[string]any{"class": propertyClass(property["property_id"]), "name": name, "base": app.Text(property["base"])})
		}
		subs := []any{}
		list, _ = disc["properties"].([]any)
		for position, raw := range list {
			property, _ := raw.(map[string]any)
			sub := map[string]any{"class": propertyClass(property["property_id"]), "name": app.Text(property["property_name"]), "base": app.Text(property["base"])}
			if scored && position < len(detail.Pieces[index].Props) {
				prop := detail.Pieces[index].Props[position]
				sub["hit"] = "hit" + strconv.Itoa(int(prop.Weight*100)/25*25)
				sub["count"] = make([]struct{}, prop.Count)
			}
			subs = append(subs, sub)
		}
		item["main"], item["sub"] = mains, subs
		result = append(result, item)
	}
	return result
}

// zzzMaps are the ZZZ-Plugin ID tables downloaded with its resources.
type zzzMaps struct {
	partners map[string]struct {
		SpriteID string `json:"sprite_id"`
	}
	weapons map[string]struct {
		CodeName string `json:"CodeName"`
	}
	suits map[string]struct {
		SpriteFile string `json:"sprite_file"`
	}
	elements []struct {
		ElementType    int    `json:"element_type"`
		SubElementType int    `json:"sub_element_type"`
		Name           string `json:"en_sub"`
		PropertyID     int    `json:"property_id"`
	}
}

func readMaps(context app.ImageContext) zzzMaps {
	var maps zzzMaps
	if context.Artwork == nil {
		return maps
	}
	for name, target := range map[string]any{"PartnerId2Data": &maps.partners, "WeaponId2Data": &maps.weapons, "SuitData": &maps.suits, "ElementData": &maps.elements} {
		if raw, err := context.Artwork.Open("zzz-plugin", "resources/map/"+name+".json"); err == nil {
			_ = json.Unmarshal(raw, target)
		}
	}
	return maps
}

func (m zzzMaps) elementName(elementType, subType any) string {
	for _, element := range m.elements {
		if element.ElementType == app.Int(elementType) && element.SubElementType == app.Int(subType) {
			return element.Name
		}
	}
	return ""
}

func (m zzzMaps) elementProperty(elementType any) string {
	for _, element := range m.elements {
		if element.ElementType == app.Int(elementType) && element.SubElementType == 0 {
			return strconv.Itoa(element.PropertyID)
		}
	}
	return ""
}
