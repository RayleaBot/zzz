package images_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/RayleaBot/plugin-zzz/internal/app"
	"github.com/RayleaBot/plugin-zzz/internal/images"
)

func TestDamageShowsTheCalculationAsZZZPlugin(t *testing.T) {
	official := map[string]any{"id": 1011, "name_mi18n": "安比", "full_name_mi18n": "安比·德玛拉", "rarity": "A", "level": 60, "rank": 2, "avatar_profession": 2,
		"properties": []any{map[string]any{"property_name": "生命值", "base": "7689", "add": "2200", "final": "9889"}}}
	table := app.DamageTable{
		Columns: []app.DamageStat{{Name: "暴击率", Value: "2.4%"}, {Name: "对照组", Value: "0"}},
		Rows: []app.DamageDifferenceRow{
			{Name: "暴击率", Value: "2.4%", Differences: []float64{0, -372.6}},
			{Name: "对照组", Value: "0", Differences: []float64{372.5, -0.2}},
		},
	}
	result := app.DamageResult{
		Damages: []app.DamageRow{{Name: "感电每段", Expected: 6551.8}, {Name: "终结技", Critical: 96464.2, Expected: 75802.7}},
		Skill:   1, Anomaly: true, Areas: map[string]float64{"BasicArea": 66815.8, "CriticalArea": 1.405, "DefenceArea": 0.5176},
		Buffs: []app.DamageBuff{{Name: "驱动盘5号位", Source: "套装", Type: "增伤", Value: 0.3}, {Name: "啄木鸟电音4", Source: "套装", Type: "攻击力", Value: 305.914}, {Name: "固定", Type: "攻击力", Value: 500}},
		Sub:   table, Main: app.DamageTable{Columns: table.Columns, Rows: table.Rows[1:]},
		Weights: map[string]float64{"11102": 1},
	}
	drawn, ok := images.Damage(app.ImageContext{Game: app.Game{Prefix: "%"}, Now: time.Now()}, app.DamageImage{Panel: app.CharacterPanel{Official: official}, UID: "10000001", Result: result})
	if !ok || drawn.Template != "damage" || drawn.Data["uid"] != "10000001" {
		t.Fatalf("drawn = %+v", drawn)
	}
	if hp := drawn.Data["properties"].([]any)[0].(map[string]any); hp["label"] != "yellow" {
		t.Errorf("labels follow the weights: %v", hp)
	}
	damage := drawn.Data["damage"].(map[string]any)
	rows := damage["rows"].([]any)
	// Anomaly damage that cannot crit has no crit column.
	if first := rows[0].(map[string]any); first["crit"] != nil || first["expect"] != "6552" || first["current"] != false {
		t.Errorf("first row = %v", first)
	}
	if second := rows[1].(map[string]any); second["crit"] != "96464" || second["current"] != true {
		t.Errorf("chosen row = %v", second)
	}
	if damage["command"] != "%安比伤害2" || damage["hint"] != "%安比伤害123" || damage["skill"] != "终结技" || damage["level"] != 60 {
		t.Errorf("damage = %v", damage)
	}
	values := []string{}
	for _, cell := range damage["areas"].([]any) {
		values = append(values, cell.(map[string]any)["value"].(string))
	}
	// A missing area shows 1; defence keeps four digits.
	if want := []string{"66816", "1.41", "1.00", "1.00", "1.00", "0.5176"}; !reflect.DeepEqual(values, want) {
		t.Errorf("areas = %v, want %v", values, want)
	}
	if anomaly := damage["anomaly"].([]any); len(anomaly) != 3 || anomaly[2].(map[string]any)["label"] != "等级区" {
		t.Errorf("anomaly areas = %v", anomaly)
	}
	buffs := []string{}
	for _, buff := range damage["buffs"].([]any) {
		buffs = append(buffs, buff.(map[string]any)["value"].(string))
	}
	if want := []string{"30%", "305.91", "500"}; !reflect.DeepEqual(buffs, want) {
		t.Errorf("buff values = %v, want %v", buffs, want)
	}
	sub := drawn.Data["sub"].(map[string]any)
	cells := sub["rows"].([]any)[1].(map[string]any)["cells"].([]any)
	if cells[0].(map[string]any)["text"] != "+373" || cells[0].(map[string]any)["class"] != "positive" || cells[1].(map[string]any)["text"] != "-0" || cells[1].(map[string]any)["class"] != "negative" {
		t.Errorf("cells = %v", cells)
	}
	// A comparison with only its control row is left out, as upstream.
	if _, shown := drawn.Data["main"]; shown {
		t.Error("main comparison with one row shown")
	}

	result.Sheer, result.Areas["SheerBoostArea"] = true, 1.25
	drawn, _ = images.Damage(app.ImageContext{Game: app.Game{Prefix: "%"}}, app.DamageImage{Panel: app.CharacterPanel{Official: official}, Result: result})
	areas := drawn.Data["damage"].(map[string]any)["areas"].([]any)
	if last := areas[len(areas)-1].(map[string]any); last["label"] != "贯穿增伤区" || last["value"] != "1.25" {
		t.Errorf("sheer damage shows %v", last)
	}
}
