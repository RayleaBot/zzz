package app

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/RayleaBot/plugin-zzz/internal/reference"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

type BuildWeapon struct {
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	Level      int    `json:"level"`
	Promote    *int   `json:"promote"`
	Refinement int    `json:"refinement"`
}
type BuildProfile struct {
	Conditions         *BuildConditions   `json:"conditions,omitempty"`
	SubElement         *int               `json:"sub_element,omitempty"`
	WeaponMain         []BuildStat        `json:"weapon_main,omitempty"`
	CandidateEquipment *[]BuildGear       `json:"candidate_equipment,omitempty"`
	Level              int                `json:"level"`
	Promote            *int               `json:"promote"`
	Rank               int                `json:"rank"`
	Talents            map[string]int     `json:"talents"`
	Trees              []string           `json:"trees"`
	Attributes         map[string]float64 `json:"attributes"`
	Weapon             BuildWeapon        `json:"weapon"`
	Equipment          []BuildGear        `json:"equipment"`
	EnemyLevel         int                `json:"enemy_level"`
	CandidateWeapon    *BuildWeapon       `json:"candidate_weapon,omitempty"`
}
type BuildStat struct {
	ID      string  `json:"id,omitempty"`
	Percent bool    `json:"percent,omitempty"`
	Key     string  `json:"key"`
	Value   float64 `json:"value"`
	Times   int     `json:"times,omitempty"`
}
type BuildGear struct {
	Slot    int         `json:"slot,omitempty"`
	SetID   string      `json:"set_id,omitempty"`
	SetName string      `json:"set_name"`
	Main    BuildStat   `json:"main"`
	Sub     []BuildStat `json:"sub"`
}
type BuildResult struct {
	Conditions    *BuildConditions      `json:"conditions,omitempty"`
	EquipmentFrom *BuildEquipmentSource `json:"equipment_from,omitempty"`
	Variant       string                `json:"variant,omitempty"`
	Source        string                `json:"source"`
	Version       string                `json:"version"`
	CharacterID   string                `json:"character_id"`
	Character     string                `json:"character"`
	EnemyLevel    int                   `json:"enemy_level"`
	Baseline      BuildScenario         `json:"baseline"`
	Candidate     *BuildScenario        `json:"candidate"`
}
type BuildEquipmentSource struct {
	CharacterID string `json:"character_id"`
	Name        string `json:"name"`
}
type BuildScenario struct {
	Weapon     BuildWeapon        `json:"weapon"`
	Attributes map[string]float64 `json:"attributes"`
	Results    []BuildSkillResult `json:"results"`
}
type BuildSkillResult struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Expected *float64 `json:"expected"`
	Text     string   `json:"text,omitempty"`
	// Default marks the detail upstream ranks the group by.
	Default  bool     `json:"default,omitempty"`
	Critical *float64 `json:"critical"`
	Buffs    []string `json:"buffs"`
	Kind     string   `json:"kind"`
}
type BuildWeaponChoice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// findBuildCharacter returns the reference character only when it has a damage
// rule; some characters are catalogued for scoring alone.
func findBuildCharacter(engine *reference.Engine, game string, panel CharacterPanel) (reference.Character, error) {
	record, err := findReferenceCharacter(engine, game, panel)
	if err == nil && record.Script == "" {
		return reference.Character{}, buildUnavailable("character")
	}
	return record, err
}

func findReferenceCharacter(engine *reference.Engine, game string, panel CharacterPanel) (reference.Character, error) {
	metadata := engine.Metadata()
	short := map[string]string{"genshin": "gs", "starrail": "sr", "zzz": "zzz"}[game]
	if short == "" {
		return reference.Character{}, buildUnavailable("game")
	}
	wanted := canonicalBuildElement(panel.Element)
	ruleID := referenceCharacterID(game, panel)
	for _, record := range metadata.Characters {
		match := record.ID == ruleID
		if game == "genshin" && slices.Contains([]string{"10000005", "10000007"}, panel.ID) && slices.Contains([]string{"10000005", "10000007", "20000000"}, record.ID) {
			match = true
		}
		if record.Game == short && match && canonicalBuildElement(record.Element) == wanted {
			if game == "genshin" {
				record.ID = panel.ID
			}
			return record, nil
		}
	}
	if ruleID != panel.ID {
		err := gameError("build_unavailable", "固定参考尚无此强化形态的自动计算规则，可使用手动伤害试算。")
		err.Details = map[string]any{"reason": "enhanced_rule_missing"}
		return reference.Character{}, err
	}
	return reference.Character{}, buildUnavailable("character")
}

// Pinned MysPanelHSRApi selects enhanced rules from the actual skill prefix,
// while account queries continue to use the official, original character ID.
func referenceCharacterID(game string, panel CharacterPanel) string {
	if game != "starrail" || !slices.Contains([]string{"1212", "1205", "1005", "1006", "1004", "1102", "1217", "1310", "1306", "1307"}, panel.ID) {
		return panel.ID
	}
	for _, skill := range panel.Skills {
		if strings.HasPrefix(skill.ID, "1"+panel.ID) {
			return "2" + panel.ID[1:]
		}
	}
	return panel.ID
}
func canonicalBuildElement(value string) string {
	value = normalizedElement(value)
	if mapped := map[string]string{"fire": "pyro", "ice": "cryo", "lightning": "electro", "wind": "anemo", "雷": "electro"}[value]; mapped != "" {
		return mapped
	}
	return value
}
func buildUnavailable(reason string) error {
	err := gameError("build_unavailable", "当前面板缺少自动计算所需的数据，或固定参考尚未支持此角色。")
	err.Details = map[string]any{"reason": reason}
	return err
}
func buildStatNumber(raw string) (float64, bool) {
	raw = strings.ReplaceAll(strings.TrimSpace(raw), ",", "")
	raw = strings.TrimSuffix(raw, "%")
	value, err := strconv.ParseFloat(raw, 64)
	return value, err == nil && !math.IsNaN(value) && !math.IsInf(value, 0) && math.Abs(value) <= 1e8
}
func buildProfile(engine *reference.Engine, game string, panel CharacterPanel, record reference.Character) (BuildProfile, error) {
	if game == "zzz" {
		return buildZZZProfile(engine, panel, record)
	}
	if !panel.RankKnown || panel.Level < 1 || panel.Rank < 0 || panel.Rank > 6 || (panel.Weapon == nil && !panel.WeaponKnown) {
		return BuildProfile{}, buildUnavailable("identity")
	}
	maxLevel := 100
	if game == "starrail" {
		maxLevel = 80
	}
	if panel.Level > maxLevel {
		return BuildProfile{}, buildUnavailable("level")
	}
	w := panel.Weapon
	if w == nil {
		zero := 0
		w = &PanelEquipment{Promote: &zero, Refinement: 1}
	}
	p := BuildProfile{Level: panel.Level, Promote: panel.Promote, Rank: panel.Rank, Weapon: BuildWeapon{ID: w.ID, Name: w.Name, Level: w.Level, Promote: w.Promote, Refinement: w.Refinement}, Talents: map[string]int{}, Trees: []string{}, Attributes: map[string]float64{}, Equipment: []BuildGear{}, EnemyLevel: 103}
	if game == "starrail" {
		// miao profile-detail passes enemy level 80 for Star Rail.
		p.EnemyLevel = 80
	}
	if w.ID != "" && (w.Level < 1 || w.Level > 90 || w.Refinement < 1 || w.Refinement > 5) {
		return p, buildUnavailable("weapon")
	}
	keys := map[string]string{"2000": "hp", "2001": "atk", "2002": "def", "20": "cpct", "22": "cdmg", "23": "recharge", "28": "mastery", "26": "heal", "30": "phy"}
	if game == "starrail" {
		keys = map[string]string{"1": "hp", "2": "atk", "3": "def", "4": "speed", "5": "cpct", "6": "cdmg", "7": "heal", "9": "recharge", "10": "effPct", "11": "effDef", "58": "stance"}
	}
	if value, ok := elementDamageBonus(game, panel); ok {
		p.Attributes["dmg"] = value
	}
	for _, stat := range panel.Stats {
		key := keys[stat.ID]
		if key == "" {
			continue
		}
		if value, ok := buildStatNumber(stat.Value); ok {
			p.Attributes[key] = value
		}
		if slices.Contains([]string{"hp", "atk", "def", "speed"}, key) {
			if value, ok := buildStatNumber(stat.Base); ok {
				p.Attributes[key+"Base"] = value
			}
		}
	}
	for _, key := range []string{"hp", "atk", "def", "hpBase", "atkBase", "defBase", "cpct", "cdmg", "dmg"} {
		if _, ok := p.Attributes[key]; !ok {
			return p, buildUnavailable("attributes." + key)
		}
	}
	talents := asObject(record.Data["talent"])
	for key, talent := range PanelTalents(game, panel, record) {
		p.Talents[key] = talent.Level
	}
	if game == "starrail" {
		for _, skill := range panel.Skills {
			if skill.Active && skill.PointType != 2 && skill.ID != "" {
				p.Trees = append(p.Trees, RecordSkillID(game, panel, record, skill))
			}
		}
	}
	for key, raw := range talents {
		if !slices.Contains([]string{"a", "e", "q", "t", "me", "mt", "xe"}, key) {
			continue
		}
		if len(asObject(raw)) > 0 && p.Talents[key] == 0 {
			return p, buildUnavailable("talent." + key)
		}
	}
	var err error
	p.Equipment, err = buildGearSet(game, panel)
	return p, err
}
func referenceBuild(ctx context.Context, engine *reference.Engine, record reference.Character, input BuildProfile) (BuildResult, error) {
	var raw map[string]any
	if decodeObject(input, &raw) != nil {
		return BuildResult{}, buildUnavailable("input")
	}
	result, err := engine.Run(ctx, record, raw)
	if err != nil {
		return BuildResult{}, buildUnavailable("reference_calculation")
	}
	var output BuildResult
	if json.Unmarshal(result, &output) != nil {
		return output, buildUnavailable("result")
	}
	return output, nil
}
func (a *App) buildAction(ctx context.Context, client AccountsClient, action string, input map[string]any) (map[string]any, error) {
	panel, err := a.queryCharacterPanel(ctx, client, Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}, asText(input["character_id"]))
	if err != nil {
		return nil, err
	}
	record, err := findBuildCharacter(a.Game.Calc, a.Game.ID, panel)
	if err != nil {
		return nil, err
	}
	profile, err := buildProfile(a.Game.Calc, a.Game.ID, panel, record)
	if err != nil {
		return nil, err
	}
	if action == "build.prepare" {
		calculated, calcErr := referenceBuild(ctx, a.Game.Calc, record, profile)
		if calcErr != nil {
			return nil, calcErr
		}
		applyBuildIdentity(&calculated, panel, record)
		metadata := a.Game.Calc.Metadata()
		choices := []BuildWeaponChoice{}
		for _, weapon := range metadata.Weapons {
			if weapon.Game == record.Game && weapon.Type == record.WeaponType {
				choices = append(choices, BuildWeaponChoice{ID: weapon.ID, Name: weapon.Name})
			}
		}
		slices.SortFunc(choices, func(a, b BuildWeaponChoice) int { return strings.Compare(a.Name, b.Name) })
		return map[string]any{"character": panel.Name, "weapon": calculated.Baseline.Weapon, "weapons": choices, "talents": profile.Talents, "enemy_level": profile.EnemyLevel, "version": calculated.Version, "build": calculated}, nil
	}
	if action != "build.compare" {
		return nil, gameError("operation_denied", "操作不存在。")
	}
	if raw, exists := input["conditions"]; exists {
		profile.Conditions, err = a.validateConditions(panel.ID, raw)
		if err != nil {
			return nil, err
		}
	}
	if value, ok := input["enemy_level"]; ok {
		var level float64
		if decodeObject(value, &level) != nil || level < 1 || level > 200 || math.Trunc(level) != level {
			return nil, gameError("build_input_invalid", "敌人等级应为 1–200 的整数。")
		}
		profile.EnemyLevel = int(level)
	}
	if value, exists := input["candidate_weapon"]; exists {
		var w BuildWeapon
		maxPromote, maxLevel := 6, 90
		if record.Game == "zzz" {
			maxPromote, maxLevel = 5, 60
		}
		if value == nil || decodeObject(value, &w) != nil || w.Promote == nil || *w.Promote < 0 || *w.Promote > maxPromote || w.Refinement < 1 || w.Refinement > 5 || (w.ID != "" && (w.Level < 1 || w.Level > maxLevel)) || (w.ID == "" && (w.Level != 0 || *w.Promote != 0 || w.Refinement != 1)) {
			return nil, gameError("build_input_invalid", "请填写有效的武器、等级、突破阶段和精炼。")
		}
		profile.CandidateWeapon = &w
		if w.ID != "" {
			metadata := a.Game.Calc.Metadata()
			valid := false
			for _, weapon := range metadata.Weapons {
				if weapon.Game == record.Game && weapon.ID == w.ID && weapon.Type == record.WeaponType {
					valid = true
					break
				}
			}
			steps := []int{1, 20, 40, 50, 60, 70, 80, 90}
			if record.Game == "sr" {
				steps = []int{1, 20, 30, 40, 50, 60, 70, 80}
			}
			if record.Game == "zzz" {
				steps = []int{1, 10, 20, 30, 40, 50, 60}
			}
			if !valid || w.Level < steps[*w.Promote] || w.Level > steps[*w.Promote+1] {
				return nil, gameError("build_input_invalid", "候选武器类型或突破阶段与所选角色、等级不符。")
			}
		}
	}
	var equipmentSource *BuildEquipmentSource
	if value, exists := input["equipment_from_character_id"]; exists {
		id := asText(value)
		number, err := strconv.Atoi(id)
		if err != nil || number <= 0 || number > 1000000000 {
			return nil, gameError("build_input_invalid", "请选择有效的装备来源角色。")
		}
		source := panel
		if id != panel.ID {
			source, err = a.queryCharacterPanel(ctx, client, Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}, id)
			if err != nil {
				return nil, err
			}
		}
		gear, err := buildGearSet(a.Game.ID, source)
		if err != nil {
			return nil, err
		}
		profile.CandidateEquipment = &gear
		equipmentSource = &BuildEquipmentSource{CharacterID: source.ID, Name: source.Name}
	}
	result, err := referenceBuild(ctx, a.Game.Calc, record, profile)
	if err != nil {
		return nil, err
	}
	applyBuildIdentity(&result, panel, record)
	result.EquipmentFrom = equipmentSource
	result.Conditions = profile.Conditions
	return map[string]any{"build": result}, nil
}
func applyBuildIdentity(result *BuildResult, panel CharacterPanel, record reference.Character) {
	result.CharacterID = panel.ID
	result.Character = panel.Name
	if record.Game == "sr" && record.ID != panel.ID {
		result.Variant = "enhanced"
		result.Character += "（强化形态）"
	}
}

// panelDamage calculates the reference damage of a panel already read from the
// account, so a panel reply does not query it twice.
func (a *App) panelDamage(ctx context.Context, panel CharacterPanel) (BuildResult, error) {
	record, err := findBuildCharacter(a.Game.Calc, a.Game.ID, panel)
	if err != nil {
		return BuildResult{}, err
	}
	profile, err := buildProfile(a.Game.Calc, a.Game.ID, panel, record)
	if err != nil {
		return BuildResult{}, err
	}
	result, err := referenceBuild(ctx, a.Game.Calc, record, profile)
	if err != nil {
		return BuildResult{}, err
	}
	applyBuildIdentity(&result, panel, record)
	return result, nil
}

// fullPanelView answers a single-character panel the way upstream's
// <角色>面板 does: attributes and equipment with their scores, then the
// reference damage. A part that cannot be calculated is named in the note
// instead of failing the whole reply. When record is set, a panel viewed in a
// group also enters the group ranking; change is the 面板换装 word of a
// changed panel.
func (a *App) fullPanelView(ctx context.Context, event *rayleabot.EventContext, panel CharacterPanel, uid string, record bool, change string) View {
	missing := []string{}
	if scored, err := a.scorePanel(ctx, panel); err == nil {
		panel = scored
	} else {
		missing = append(missing, "评分："+friendlyError(err))
	}
	view := PanelView(a.Game, []CharacterPanel{panel}, uid)
	image := PanelImage{Panel: panel, UID: uid, Change: change}
	if change != "" {
		view.Note = "该面板为非实际数据。当前替换命令：" + change
	}
	if result, err := a.panelDamage(ctx, panel); err == nil {
		damage := BuildView(a.Game, result)
		view.Sections = append(view.Sections, Section{Title: "参考伤害 · " + result.Version, Rows: damage.Rows})
		image.Damage = &result
	} else {
		missing = append(missing, "伤害："+friendlyError(err))
	}
	if record {
		a.recordRank(event, uid, panel, image.Damage)
	}
	if len(missing) > 0 {
		view.Note += "\n" + strings.Join(missing, "\n")
	}
	if a.panel != nil {
		if a.Game.Calc != nil {
			image.Record, _ = findReferenceCharacter(a.Game.Calc, a.Game.ID, panel)
		}
		if drawn, ok := a.panel(a.imageContext(ctx), image); ok {
			view.Image = &drawn
		}
	}
	return view
}

func BuildView(game Game, result BuildResult) View {
	rows := []Row{}
	for _, item := range result.Baseline.Results {
		value := item.Text
		if item.Expected != nil {
			value = fmt.Sprintf("期望 %.1f", *item.Expected)
		}
		if item.Critical != nil {
			value += fmt.Sprintf(" · 暴击 %.1f", *item.Critical)
		}
		rows = append(rows, Row{Label: item.Title, Value: value})
	}
	return View{Title: result.Character + " · 参考伤害", Subtitle: game.Name + " · " + result.Version, Rows: rows, Note: "按固定参考列出的战斗情境自动应用角色、武器和套装增益。条件明细与武器换装可在管理页查看。"}
}

// elementDamageBonus reads the damage bonus of the character's own element from
// the official panel. Only displayed percentages are accepted, so a fractional
// number from an unknown endpoint revision is never mistaken for one.
func elementDamageBonus(game string, panel CharacterPanel) (float64, bool) {
	element := normalizedElement(panel.Element)
	if mapped := map[string]string{"fire": "pyro", "ice": "cryo", "lightning": "electro", "wind": "anemo", "以太": "ether", "电": "electro"}[element]; mapped != "" {
		element = mapped
	}
	statID := map[string]string{"pyro": "40", "electro": "41", "hydro": "42", "dendro": "43", "anemo": "44", "geo": "45", "cryo": "46"}[element]
	statName := ""
	switch game {
	case "starrail":
		statID = map[string]string{"physical": "12", "pyro": "14", "cryo": "16", "electro": "18", "anemo": "20", "quantum": "22", "imaginary": "24"}[element]
	case "zzz":
		// Official property names disambiguate element-specific short IDs; an
		// unknown element is deliberately not assigned another element's bonus.
		statID = map[string]string{"physical": "31503", "pyro": "31603", "cryo": "31703", "electro": "31803", "anemo": "32303", "ether": "31903"}[element]
		if label := map[string]string{"physical": "物理", "pyro": "火", "cryo": "冰", "electro": "电", "anemo": "风", "ether": "以太"}[element]; label != "" {
			statName = label + "属性伤害加成"
		}
	}
	for _, stat := range panel.Stats {
		if (statID == "" || stat.ID != statID) && (statName == "" || stat.Name != statName) {
			continue
		}
		raw := strings.ReplaceAll(strings.TrimSpace(stat.Value), ",", "")
		if !strings.HasSuffix(raw, "%") {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSuffix(raw, "%"), 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		return value, true
	}
	return 0, false
}

// PanelTalent is a skill level as miao's panel shows it: Level includes
// constellation or eidolon bonuses, Original does not.
type PanelTalent struct {
	Level    int
	Original int
}

// PanelTalents maps the panel's skills to miao talent keys (a, e and q, plus
// t, me, mt and xe in Star Rail) with the calculation record.
func PanelTalents(game string, panel CharacterPanel, record reference.Character) map[string]PanelTalent {
	result := map[string]PanelTalent{}
	talents := asObject(record.Data["talent"])
	talentIDs := asObject(record.Data["talentId"])
	skillKinds := map[string]string{"普攻": "a", "战技": "e", "终结技": "q", "天赋": "t", "秘技": "z", "欢愉技": "xe", "忆灵技": "me", "忆灵天赋": "mt"}
	activeIndex := 0
	for _, skill := range panel.Skills {
		key := asText(talentIDs[RecordSkillID(game, panel, record, skill)])
		if game == "starrail" && skillKinds[skill.Kind] != "" {
			key = skillKinds[skill.Kind]
		}
		if key == "" {
			for k, raw := range talents {
				if firstText(asObject(raw), "name") == skill.Name {
					key = strings.TrimRight(k, "123")
					break
				}
			}
		}
		if game == "genshin" && skill.SkillType == 1 {
			if key == "" && activeIndex < 3 {
				key = []string{"a", "e", "q"}[activeIndex]
			}
			activeIndex++
		}
		// The official level already includes constellation/eidolon bonuses.
		if key != "" && skill.Level > 0 && skill.Level <= 20 {
			result[key] = PanelTalent{Level: skill.Level, Original: skill.Level - skill.ExtraLevel}
		}
	}
	return result
}

// RecordSkillID translates a Star Rail skill ID of a character variant (such
// as the Trailblazer paths) to the record's own IDs.
func RecordSkillID(game string, panel CharacterPanel, record reference.Character, skill PanelSkill) string {
	skillID := skill.ID
	if game != "starrail" || record.ID == panel.ID {
		return skillID
	}
	if strings.HasPrefix(skillID, "1"+panel.ID) {
		skillID = record.ID + strings.TrimPrefix(skillID, "1"+panel.ID)
	}
	traceData, traceDetails := asObject(record.Data["tree"]), asObject(record.Data["treeData"])
	for _, candidate := range []string{skill.ID, skillID, "1" + skillID} {
		if traceData[candidate] != nil || traceDetails[candidate] != nil {
			return candidate
		}
	}
	return skillID
}
