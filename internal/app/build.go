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
func findBuildCharacter(engine *reference.Engine, panel CharacterPanel) (reference.Character, error) {
	record, err := findReferenceCharacter(engine, panel)
	if err == nil && record.Script == "" {
		return reference.Character{}, buildUnavailable("character")
	}
	return record, err
}

func findReferenceCharacter(engine *reference.Engine, panel CharacterPanel) (reference.Character, error) {
	wanted := canonicalBuildElement(panel.Element)
	for _, record := range engine.Metadata().Characters {
		if record.ID == panel.ID && canonicalBuildElement(record.Element) == wanted {
			return record, nil
		}
	}
	return reference.Character{}, buildUnavailable("character")
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
	record, err := findBuildCharacter(a.Game.Calc, panel)
	if err != nil {
		return nil, err
	}
	profile, err := buildProfile(a.Game.Calc, panel, record)
	if err != nil {
		return nil, err
	}
	if action == "build.prepare" {
		calculated, calcErr := referenceBuild(ctx, a.Game.Calc, record, profile)
		if calcErr != nil {
			return nil, calcErr
		}
		applyBuildIdentity(&calculated, panel)
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
		maxPromote, maxLevel := 5, 60
		if value == nil || decodeObject(value, &w) != nil || w.Promote == nil || *w.Promote < 0 || *w.Promote > maxPromote || w.Refinement < 1 || w.Refinement > 5 || (w.ID != "" && (w.Level < 1 || w.Level > maxLevel)) || (w.ID == "" && (w.Level != 0 || *w.Promote != 0 || w.Refinement != 1)) {
			return nil, gameError("build_input_invalid", "请填写有效的音擎、等级、突破阶段和星级。")
		}
		profile.CandidateWeapon = &w
		if w.ID != "" {
			metadata := a.Game.Calc.Metadata()
			valid := false
			for _, weapon := range metadata.Weapons {
				if weapon.ID == w.ID && weapon.Type == record.WeaponType {
					valid = true
					break
				}
			}
			steps := []int{1, 10, 20, 30, 40, 50, 60}
			if !valid || w.Level < steps[*w.Promote] || w.Level > steps[*w.Promote+1] {
				return nil, gameError("build_input_invalid", "候选音擎类型或突破阶段与所选角色、等级不符。")
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
		gear, err := buildGearSet(source)
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
	applyBuildIdentity(&result, panel)
	result.EquipmentFrom = equipmentSource
	result.Conditions = profile.Conditions
	return map[string]any{"build": result}, nil
}
func applyBuildIdentity(result *BuildResult, panel CharacterPanel) {
	result.CharacterID = panel.ID
	result.Character = panel.Name
}

// panelDamage calculates the reference damage of a panel already read from the
// account, so a panel reply does not query it twice.
func (a *App) panelDamage(ctx context.Context, panel CharacterPanel) (BuildResult, error) {
	record, err := findBuildCharacter(a.Game.Calc, panel)
	if err != nil {
		return BuildResult{}, err
	}
	profile, err := buildProfile(a.Game.Calc, panel, record)
	if err != nil {
		return BuildResult{}, err
	}
	result, err := referenceBuild(ctx, a.Game.Calc, record, profile)
	if err != nil {
		return BuildResult{}, err
	}
	applyBuildIdentity(&result, panel)
	return result, nil
}

// fullPanelView answers a single-character panel the way upstream's
// <角色>面板 does: attributes and equipment with their scores, then the
// reference damage. A part that cannot be calculated is named in the note
// instead of failing the whole reply. A panel viewed in a group also enters
// the group ranking.
func (a *App) fullPanelView(ctx context.Context, event *rayleabot.EventContext, panel CharacterPanel, uid string) View {
	missing := []string{}
	if scored, err := a.scorePanel(ctx, panel); err == nil {
		panel = scored
	} else {
		missing = append(missing, "评分："+friendlyError(err))
	}
	view := PanelView(a.Game, []CharacterPanel{panel}, uid)
	image := PanelImage{Panel: panel, UID: uid}
	if result, err := a.panelDamage(ctx, panel); err == nil {
		damage := BuildView(a.Game, result)
		view.Sections = append(view.Sections, Section{Title: "参考伤害 · " + result.Version, Rows: damage.Rows})
		image.Damage = &result
	} else {
		missing = append(missing, "伤害："+friendlyError(err))
	}
	a.recordRank(event, uid, panel, image.Damage)
	if len(missing) > 0 {
		view.Note += "\n" + strings.Join(missing, "\n")
	}
	if a.panel != nil {
		if a.Game.Calc != nil {
			image.Record, _ = findReferenceCharacter(a.Game.Calc, panel)
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
	return View{Title: result.Character + " · 参考伤害", Subtitle: game.Name + " · " + result.Version, Rows: rows, Note: "按固定参考列出的战斗情境自动应用角色、武器和套装增益。"}
}

// PanelTalent is a skill level as miao's panel shows it: Level includes
// constellation or eidolon bonuses, Original does not.
type PanelTalent struct {
	Level    int
	Original int
}

func buildGearSet(panel CharacterPanel) ([]BuildGear, error) {
	if !panel.EquipmentKnown {
		return nil, buildUnavailable("equipment.missing")
	}
	out := []BuildGear{}
	slots := map[int]bool{}
	maxSlot := 6
	for _, gear := range panel.Equipment {
		if !gear.Complete || len(gear.Main) != 1 || gear.Slot < 1 || gear.Slot > maxSlot || slots[gear.Slot] || gear.SetName == "" {
			return nil, buildUnavailable("equipment")
		}
		slots[gear.Slot] = true
		piece := BuildGear{Slot: gear.Slot, SetName: gear.SetName, Sub: []BuildStat{}}
		for i, stat := range append(append([]PanelStat{}, gear.Main...), gear.Sub...) {
			value, ok := buildStatNumber(stat.Value)
			if !ok || stat.ID == "" || stat.Key == "" {
				return nil, buildUnavailable("equipment.stat")
			}
			s := BuildStat{ID: stat.ID, Key: stat.Key, Value: value, Percent: strings.HasSuffix(strings.TrimSpace(stat.Value), "%")}
			if i == 0 {
				piece.Main = s
			} else {
				piece.Sub = append(piece.Sub, s)
			}
		}
		out = append(out, piece)
	}
	return out, nil
}

func buildProfile(engine *reference.Engine, panel CharacterPanel, record reference.Character) (BuildProfile, error) {
	p := BuildProfile{Level: panel.Level, Promote: panel.Promote, Rank: panel.Rank, Talents: map[string]int{}, Trees: []string{}, Attributes: map[string]float64{}, SubElement: panel.SubElement, EnemyLevel: panel.Level}
	if !panel.RankKnown || p.Level < 1 || p.Level > 60 || p.Rank < 0 || p.Rank > 6 || panel.Profession < 1 || panel.Profession > 7 || (!panel.WeaponKnown && panel.Weapon == nil) {
		return p, buildUnavailable("identity")
	}
	if strconv.Itoa(panel.Profession) != record.WeaponType {
		return p, buildUnavailable("profession")
	}
	names := map[string]string{"生命值": "HP", "攻击力": "ATK", "防御力": "DEF", "冲击力": "Impact", "暴击率": "CRITRate", "暴击伤害": "CRITDMG", "异常掌控": "AnomalyMastery", "异常精通": "AnomalyProficiency", "穿透率": "PenRatio", "能量自动回复": "EnergyRegen", "贯穿力": "SheerForce"}
	ids := map[string]string{"1": "HP", "2": "ATK", "3": "DEF", "4": "Impact", "5": "CRITRate", "6": "CRITDMG", "7": "AnomalyMastery", "8": "AnomalyProficiency", "9": "PenRatio", "11": "EnergyRegen", "19": "SheerForce", "11103": "HP", "12103": "ATK", "13103": "DEF", "12202": "Impact", "20103": "CRITRate", "21103": "CRITDMG", "31402": "AnomalyMastery", "31203": "AnomalyProficiency", "23103": "PenRatio", "30502": "EnergyRegen"}
	for _, stat := range panel.Stats {
		key := names[stat.Name]
		if key == "" {
			key = ids[stat.ID]
		}
		if key == "" {
			continue
		}
		value, ok := buildStatNumber(stat.Value)
		if !ok {
			continue
		}
		percent := strings.HasSuffix(strings.TrimSpace(stat.Value), "%")
		if slices.Contains([]string{"CRITRate", "CRITDMG", "PenRatio"}, key) && !percent {
			return p, buildUnavailable("attributes.units")
		}
		if percent {
			value /= 100
		}
		p.Attributes[key] = value
		if n, ok := buildStatNumber(stat.Base); ok {
			if strings.HasSuffix(stat.Base, "%") {
				n /= 100
			}
			p.Attributes[key+"Base"] = n
		}
	}
	for _, key := range []string{"HP", "ATK", "DEF", "HPBase", "ATKBase", "DEFBase", "Impact", "CRITRate", "CRITDMG", "AnomalyMastery", "AnomalyProficiency"} {
		if _, ok := p.Attributes[key]; !ok {
			return p, buildUnavailable("attributes." + key)
		}
	}
	for _, skill := range panel.Skills {
		if !skill.HasSkillType {
			return p, buildUnavailable("talent.type")
		}
		if !slices.Contains([]int{0, 1, 2, 3, 5, 6}, skill.SkillType) {
			continue
		}
		maxLevel := 16
		if skill.SkillType == 5 {
			maxLevel = 7
		}
		if skill.Level < 1 || skill.Level > maxLevel {
			return p, buildUnavailable("talent.level")
		}
		key := strconv.Itoa(skill.SkillType)
		if _, exists := p.Talents[key]; exists {
			return p, buildUnavailable("talent.duplicate")
		}
		p.Talents[key] = skill.Level
	}
	for _, key := range []string{"0", "1", "2", "3", "5", "6"} {
		if p.Talents[key] == 0 {
			return p, buildUnavailable("talent." + key)
		}
	}
	if panel.Weapon != nil {
		w := panel.Weapon
		if w.ID == "" || w.Level < 1 || w.Level > 60 || w.Refinement < 1 || w.Refinement > 5 {
			return p, buildUnavailable("weapon")
		}
		p.Weapon = BuildWeapon{ID: w.ID, Name: w.Name, Level: w.Level, Promote: w.Promote, Refinement: w.Refinement}
		metadata := engine.Metadata()
		found := false
		for _, candidate := range metadata.Weapons {
			if candidate.ID == w.ID {
				found = true
				break
			}
		}
		if !found {
			err := gameError("build_unavailable", "固定参考尚未提供此音擎的被动规则，请使用通用试算或其他已适配配装。")
			err.Details = map[string]any{"reason": "weapon_rule_missing"}
			return p, err
		}
		if len(w.Main) == 0 {
			return p, buildUnavailable("weapon.attributes")
		}
		for _, stat := range w.Main {
			value, ok := buildStatNumber(stat.Value)
			if !ok {
				return p, buildUnavailable("weapon.attributes")
			}
			p.WeaponMain = append(p.WeaponMain, BuildStat{ID: stat.ID, Key: stat.Key, Value: value})
		}
	} else {
		zero := 0
		p.Weapon = BuildWeapon{Promote: &zero, Refinement: 1}
	}
	var err error
	p.Equipment, err = buildGearSet(panel)
	return p, err
}
