package images_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/RayleaBot/plugin-zzz/internal/app"
	"github.com/RayleaBot/plugin-zzz/internal/images"
)

func TestPanelFollowsZZZPluginRules(t *testing.T) {
	property := func(name string, id int, base string) map[string]any {
		return map[string]any{"property_name": name, "property_id": id, "base": base}
	}
	official := map[string]any{
		"id": 1011, "name_mi18n": "安比", "full_name_mi18n": "安比·德玛拉", "rarity": "A", "level": 60, "rank": 2, "avatar_profession": 7,
		"skills":     []any{map[string]any{"level": 1}, map[string]any{"level": 2}, map[string]any{"level": 3}, map[string]any{"level": 4}, map[string]any{"level": 5}, map[string]any{"level": 6}},
		"properties": []any{map[string]any{"property_name": "生命值", "base": "7689", "add": "2200", "final": "9889"}, map[string]any{"property_name": "防御力", "base": "612", "add": "0", "final": "612"}},
		"equip": []any{map[string]any{"equipment_type": 2, "id": 31021, "level": 15, "rarity": "S", "name": "盘",
			"main_properties": []any{property("电属性伤害加成", 31803, "30%")},
			"properties":      []any{property("暴击率", 20103, "7.2%"), property("攻击力", 12102, "3%")}}},
	}
	raw, _ := json.Marshal(map[string]any{"total": 30.5, "grade": "S", "weights": map[string]float64{"11102": 1, "12102": 0.75},
		"stats":  []any{map[string]any{"name": "暴击", "weight": 1, "value": "7.2%", "count": 3}, map[string]any{"name": "攻击", "weight": 0.5, "value": "3.0%", "count": 1}, map[string]any{"name": "防御", "weight": 0, "value": "4.8%", "count": 1}},
		"pieces": []any{map[string]any{"slot": 2, "score": 30.5, "grade": "S", "props": []any{map[string]any{"id": 20103, "count": 2, "weight": 1}, map[string]any{"id": 12102, "count": 0, "weight": 0.75}}}}})
	panel := app.CharacterPanel{Official: official, ScoreDetail: &app.ScoreDetail{Raw: raw}}
	damage := app.DamageResult{
		Damages:    []app.DamageRow{{Name: "感电每段", Expected: 6551.5}, {Name: "终结技", Critical: 96464.2, Expected: 75802.7}},
		PanelBuffs: []app.DamageBuff{{Name: "核心被动：迷你毁灭拍档", Type: "穿透率", Value: 0.156, Max: 0.3}, {Name: "技能：加油！", Type: "攻击力", Value: 568.928}},
	}
	drawn, ok := images.Panel(app.ImageContext{Game: app.Game{Prefix: "%"}, Now: time.Now()}, app.PanelImage{Panel: panel, UID: "10000001", Damage: damage})
	if !ok || drawn.Template != "panel" {
		t.Fatalf("drawn = %+v", drawn)
	}
	// Skill levels follow upstream's order of the official list.
	if skills := drawn.Data["skills"].([]any); skills[0] != "1" || skills[1] != "3" || skills[2] != "6" {
		t.Errorf("skills = %v", skills)
	}
	rows := drawn.Data["properties"].([]any)
	first, second := rows[0].(map[string]any), rows[1].(map[string]any)
	if first["label"] != "yellow" || first["detail"] != true || first["final"] != "9889" {
		t.Errorf("hp row = %v", first)
	}
	// Armorer agents show Laceration in place of attack.
	if second["name"] != "锐暴伤害" || second["final"] != "0" {
		t.Errorf("second row = %v", second)
	}
	if defence := rows[2].(map[string]any); defence["detail"] != false {
		t.Errorf("a zero bonus hides the detail: %v", defence)
	}
	rating := drawn.Data["rating"].(map[string]any)
	if rating["useful"] != 4 || rating["effective"] != "3.50" || rating["score"] != "30.50" || len(rating["stats"].([]any)) != 9 {
		t.Errorf("rating = %v", rating)
	}
	discs := drawn.Data["discs"].([]any)
	if discs[0].(map[string]any)["empty"] != true {
		t.Errorf("slot 1 should be empty: %v", discs[0])
	}
	disc := discs[1].(map[string]any)
	subs := disc["sub"].([]any)
	if main := disc["main"].([]any)[0].(map[string]any); main["name"] != "电伤加成" {
		t.Errorf("main = %v", main)
	}
	if subs[0].(map[string]any)["hit"] != "hit100" || len(subs[0].(map[string]any)["count"].([]struct{})) != 2 || subs[1].(map[string]any)["hit"] != "hit75" {
		t.Errorf("subs = %v", subs)
	}
	// Damages and buffs read as upstream's card shows them: anomaly damage
	// has no crit, and a buff's maximum follows its value.
	table := drawn.Data["damage"].(map[string]any)
	rows, buffs := table["rows"].([]any), table["buffs"].([]any)
	if first := rows[0].(map[string]any); first["crit"] != nil || first["expect"] != "6552" || rows[1].(map[string]any)["crit"] != "96464" {
		t.Errorf("rows = %v", rows)
	}
	if buffs[0].(map[string]any)["value"] != "16%/30%" || buffs[1].(map[string]any)["value"] != "568.93" || table["hint"] != "%安比伤害" {
		t.Errorf("damage = %v", table)
	}
	if drawn, _ = images.Panel(app.ImageContext{}, app.PanelImage{Panel: panel}); drawn.Data["damage"] != nil {
		t.Errorf("no damage calculated still shows %v", drawn.Data["damage"])
	}
	if _, ok := images.Panel(app.ImageContext{}, app.PanelImage{}); ok {
		t.Error("a panel without the official entry should keep the summary card")
	}
}
