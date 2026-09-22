package app

import (
	"fmt"
	"slices"
	"strings"
)

type EnemyInfo struct {
	Name     string   `json:"name"`
	Group    string   `json:"group"`
	Aliases  []string `json:"aliases"`
	HPCurve  string   `json:"hp_curve"`
	HPBase   *float64 `json:"hp_base"`
	ATKCurve string   `json:"atk_curve"`
	ATKBase  *float64 `json:"atk_base"`
}
type EnemyModifier struct {
	ID    string   `json:"id"`
	Group string   `json:"group"`
	Names []string `json:"names"`
	Level int      `json:"level"`
	Value any      `json:"value"`
}

// EnemyTable is the Genshin monster data from the pinned Atlas snapshot.
type EnemyTable struct {
	Version   string               `json:"version"`
	Enemies   []EnemyInfo          `json:"enemies"`
	Curves    []map[string]float64 `json:"curves"`
	Modifiers []EnemyModifier      `json:"modifiers"`
}

func (a *App) enemyAction(action string, input map[string]any) (map[string]any, error) {
	enemyData := a.Game.Data.Enemies
	if enemyData == nil {
		return nil, gameError("operation_denied", "此游戏没有固定原魔资料。")
	}
	if action == "enemies.schema" {
		return map[string]any{"enemies": enemyData.Enemies, "modifiers": enemyData.Modifiers, "version": enemyData.Version, "max_level": len(enemyData.Curves)}, nil
	}
	var q struct {
		Name      string   `json:"name"`
		Level     int      `json:"level"`
		Stat      string   `json:"stat"`
		Modifiers []string `json:"modifiers"`
	}
	if decodeObject(input, &q) != nil || len(q.Modifiers) > 5 || len(q.Name) > 128 || (q.Stat != "HP" && q.Stat != "ATK") || q.Level < 0 || q.Level > len(enemyData.Curves) {
		return nil, gameError("input_invalid", "请选择原魔、属性、有效等级和因子。")
	}
	matches := []EnemyInfo{}
	for _, e := range enemyData.Enemies {
		if e.Name == q.Name || slices.Contains(e.Aliases, q.Name) {
			matches = append(matches, e)
		}
	}
	if len(matches) != 1 {
		return nil, gameError("enemy_ambiguous", "原魔名称未唯一匹配，请从列表选择完整名称。")
	}
	enemy := matches[0]
	curve, base := enemy.HPCurve, enemy.HPBase
	if q.Stat == "ATK" {
		curve, base = enemy.ATKCurve, enemy.ATKBase
	}
	if base == nil || !finiteRange(*base, 0, 1e15) || curve == "" {
		return nil, gameError("enemy_missing", "固定资料缺少此原魔属性。")
	}
	level := q.Level
	groups := map[string]bool{}
	ratio := 1.0
	rows := []Row{}
	for _, id := range q.Modifiers {
		i := slices.IndexFunc(enemyData.Modifiers, func(m EnemyModifier) bool { return m.ID == id })
		if i < 0 {
			return nil, gameError("input_invalid", "原魔因子不存在。")
		}
		m := enemyData.Modifiers[i]
		if !strings.HasSuffix(m.Group, q.Stat) || groups[m.Group] {
			return nil, gameError("input_invalid", "每个属性因子类别只能选择一项。")
		}
		groups[m.Group] = true
		if q.Level == 0 && m.Level > 0 {
			level = m.Level
		}
		value, ok := m.Value.(float64)
		if !ok {
			for _, raw := range asList(m.Value) {
				v := asObject(raw)
				names := []string{}
				for _, n := range asList(v["enemy"]) {
					names = append(names, asText(n))
				}
				if len(names) == 0 || slices.Contains(names, enemy.Name) {
					value, ok = challengeNumber(v["value"])
					break
				}
			}
		}
		if !ok || !finiteRange(value, 0, 1000) {
			return nil, gameError("enemy_missing", "因子没有此原魔的有效倍率。")
		}
		ratio *= value
		rows = append(rows, Row{strings.Join(m.Names, " / "), fmt.Sprintf("%.4g 倍", value)})
	}
	if level == 0 {
		level = 90
	}
	var scale float64
	known := false
	for _, c := range enemyData.Curves {
		if int(c["Level"]) == level {
			scale, known = c[curve]
			break
		}
	}
	if !known || !finiteRange(scale, 0, 1e15) {
		return nil, gameError("enemy_missing", "固定资料没有此等级的属性曲线。")
	}
	value := scale * *base * ratio
	if !finiteRange(value, 0, 1e30) {
		return nil, gameError("enemy_missing", "计算结果超出有效范围。")
	}
	view := View{Title: fmt.Sprintf("%d级 %s · %s", level, enemy.Name, q.Stat), Rows: []Row{{"计算结果", fmt.Sprintf("%.1f", value)}, {"计算过程", fmt.Sprintf("曲线 %.6g × 基础 %.6g × 因子 %.6g", scale, *base, ratio)}}, Sections: []Section{{Title: "所选修饰因子", Rows: rows}}, Note: "来源：" + enemyData.Version + "。固定数据，未校准快照之外的新版本。"}
	return map[string]any{"value": value, "level": level, "base": *base, "curve": scale, "ratio": ratio, "view": view}, nil
}
