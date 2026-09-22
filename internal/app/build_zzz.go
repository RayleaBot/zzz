package app

import (
	"slices"
	"strconv"
	"strings"

	"github.com/RayleaBot/plugin-zzz/internal/reference"
)

func buildGearSet(game string, panel CharacterPanel) ([]BuildGear, error) {
	if !panel.EquipmentKnown {
		return nil, buildUnavailable("equipment.missing")
	}
	out := []BuildGear{}
	slots := map[int]bool{}
	maxSlot := 6
	if game == "genshin" {
		maxSlot = 5
	}
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

func buildZZZProfile(engine *reference.Engine, panel CharacterPanel, record reference.Character) (BuildProfile, error) {
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
			if candidate.Game == "zzz" && candidate.ID == w.ID {
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
	p.Equipment, err = buildGearSet("zzz", panel)
	return p, err
}
