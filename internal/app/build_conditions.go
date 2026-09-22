package app

import (
	"crypto/rand"
	"github.com/RayleaBot/plugin-zzz/internal/localdata"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

type BuildConditions struct {
	DisableCharacter bool               `json:"disable_character"`
	DisableWeapon    bool               `json:"disable_weapon"`
	DisableEquipment bool               `json:"disable_equipment"`
	EnemyResistance  *float64           `json:"enemy_resistance,omitempty"`
	Bonuses          map[string]float64 `json:"bonuses"`
	Team             []BuildTeammate    `json:"team"`
}
type BuildTeammate struct {
	CharacterID string             `json:"character_id"`
	Name        string             `json:"name"`
	Bonuses     map[string]float64 `json:"bonuses"`
}
type BuildConditionField struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Unit  string  `json:"unit"`
	Max   float64 `json:"max"`
}

func conditionFields(game string) []BuildConditionField {
	fields := []BuildConditionField{{"atkPct", "攻击力百分比", "%", 1000}, {"atkPlus", "固定攻击力", "点", 100000}, {"hpPct", "生命值百分比", "%", 1000}, {"hpPlus", "固定生命值", "点", 100000}, {"defPct", "防御力百分比", "%", 1000}, {"defPlus", "固定防御力", "点", 100000}, {"cpct", "暴击率", "%", 100}, {"cdmg", "暴击伤害", "%", 500}, {"dmg", "伤害加成", "%", 500}, {"enemyDmg", "目标受到伤害增加", "%", 100}, {"ignore", "无视防御", "%", 100}, {"resistance", "抗性降低/穿透", "%", 100}}
	if game == "zzz" {
		return append(fields, BuildConditionField{"proficiency", "异常精通", "点", 1000}, BuildConditionField{"anomaly", "异常增伤", "%", 200}, BuildConditionField{"sheer", "贯穿增伤", "%", 200}, BuildConditionField{"impact", "固定冲击力", "点", 1000}, BuildConditionField{"stun", "失衡易伤增量", "%", 400})
	}
	fields = append(fields, BuildConditionField{"enemyDef", "降低防御", "%", 100}, BuildConditionField{"recharge", "充能效率", "%", 500}, BuildConditionField{"aDmg", "普攻增伤", "%", 500}, BuildConditionField{"eDmg", "战技增伤", "%", 500}, BuildConditionField{"qDmg", "终结技/爆发增伤", "%", 500})
	if game == "genshin" {
		fields = append(fields, BuildConditionField{"mastery", "元素精通", "点", 5000})
	} else {
		fields = append(fields, BuildConditionField{"speed", "固定速度", "点", 500}, BuildConditionField{"stance", "击破特攻", "%", 500}, BuildConditionField{"dotDmg", "持续伤害增伤", "%", 500})
	}
	return fields
}
func (a *App) validateConditions(characterID string, value any) (*BuildConditions, error) {
	var raw map[string]any
	if decodeObject(value, &raw) != nil {
		return nil, gameError("build_input_invalid", "条件格式无效。")
	}
	checkValues := func(v any) bool {
		if v == nil {
			return true
		}
		m, ok := v.(map[string]any)
		if !ok {
			return false
		}
		for _, n := range m {
			if _, ok := n.(float64); !ok {
				return false
			}
		}
		return true
	}
	if !checkValues(raw["bonuses"]) {
		return nil, gameError("build_input_invalid", "增益必须是数值，不能使用空值。")
	}
	for _, t := range asList(raw["team"]) {
		if !checkValues(asObject(t)["bonuses"]) {
			return nil, gameError("build_input_invalid", "队友增益必须是数值。")
		}
	}
	var c BuildConditions
	if value == nil || decodeObject(value, &c) != nil {
		return nil, gameError("build_input_invalid", "自定义条件格式不正确。")
	}
	if c.EnemyResistance != nil && !finiteRange(*c.EnemyResistance, -100, 100) {
		return nil, gameError("build_input_invalid", "敌人抗性应为 -100% 至 100%。")
	}
	maxTeam := 3
	if a.Game.ID == "zzz" {
		maxTeam = 2
	}
	if len(c.Team) > maxTeam {
		return nil, gameError("build_input_invalid", "队友数量超过游戏队伍上限。")
	}
	fields := map[string]BuildConditionField{}
	for _, f := range conditionFields(a.Game.ID) {
		fields[f.Key] = f
	}
	valid := func(values map[string]float64) bool {
		if len(values) > len(fields) {
			return false
		}
		for key, v := range values {
			f, ok := fields[key]
			if !ok || !finiteRange(v, 0, f.Max) {
				return false
			}
		}
		return true
	}
	if !valid(c.Bonuses) {
		return nil, gameError("build_input_invalid", "请按字段范围填写数值增益。")
	}
	seen := map[string]bool{characterID: true}
	for i := range c.Team {
		t := &c.Team[i]
		entry, ok := a.Catalog.Get(t.CharacterID)
		if !ok || entry.Kind != "character" || seen[t.CharacterID] || !valid(t.Bonuses) {
			return nil, gameError("build_input_invalid", "队友必须来自本游戏图鉴、不可重复，增益需在所列范围内。")
		}
		seen[t.CharacterID] = true
		t.Name = entry.Name
	}
	return &c, nil
}

type BuildPreset struct {
	Ref         string          `json:"ref"`
	Revision    uint64          `json:"revision"`
	Name        string          `json:"name"`
	CharacterID string          `json:"character_id"`
	Conditions  BuildConditions `json:"conditions"`
}
type BuildPresetStore struct {
	mu   sync.Mutex
	Path string
}

func (s *BuildPresetStore) List() ([]BuildPreset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := []BuildPreset{}
	err := localdata.Read(s.Path, &items)
	return items, err
}
func (a *App) buildPresetAction(action string, input map[string]any) (map[string]any, error) {
	if action == "build.conditions" {
		return map[string]any{"fields": conditionFields(a.Game.ID)}, nil
	}
	s := a.BuildPresets
	if action == "build.presets.list" {
		items, err := s.List()
		return map[string]any{"items": items}, err
	}
	var q BuildPreset
	if decodeObject(input, &q) != nil {
		return nil, gameError("build_input_invalid", "方案输入不正确。")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items := []BuildPreset{}
	if err := localdata.Read(s.Path, &items); err != nil {
		return nil, err
	}
	i := slices.IndexFunc(items, func(p BuildPreset) bool { return p.Ref == q.Ref })
	if q.Ref != "" && (i < 0 || items[i].Revision != q.Revision) {
		return nil, gameError("preset_changed", "方案已变化，请刷新列表。")
	}
	if action == "build.presets.remove" {
		if i < 0 {
			return nil, gameError("preset_missing", "方案不存在。")
		}
		items = slices.Delete(items, i, i+1)
	} else if action == "build.presets.save" {
		q.Name = strings.TrimSpace(q.Name)
		if q.Name == "" || len([]rune(q.Name)) > 40 {
			return nil, gameError("build_input_invalid", "方案名称为 1–40 个字符。")
		}
		entry, ok := a.Catalog.Get(q.CharacterID)
		if !ok || entry.Kind != "character" {
			return nil, gameError("build_input_invalid", "请选择有效角色。")
		}
		conditions, err := a.validateConditions(q.CharacterID, q.Conditions)
		if err != nil {
			return nil, err
		}
		q.Conditions = *conditions
		for n, p := range items {
			if n != i && p.Name == q.Name {
				return nil, gameError("preset_changed", "此方案名称已存在。")
			}
		}
		q.Revision++
		if i < 0 {
			if len(items) >= 64 {
				return nil, gameError("preset_limit", "最多保存 64 个方案。")
			}
			q.Ref = rand.Text()
			q.Revision = 1
			items = append(items, q)
		} else {
			items[i] = q
		}
	} else {
		return nil, gameError("operation_denied", "方案操作不存在。")
	}
	err := localdata.Write(s.Path, items)
	return map[string]any{"items": items, "preset": q}, err
}
func buildPresetPath(directory string) string { return filepath.Join(directory, "build-presets.json") }
