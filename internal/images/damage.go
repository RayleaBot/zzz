package images

import (
	"math"
	"math/big"
	"strconv"

	"github.com/RayleaBot/plugin-zzz/internal/app"
)

// damageFonts are the fonts ZZZ-Plugin's common style loads besides its own:
// the number and Chinese fonts of the Yunzai 原神插件 beside it, which the
// damage tables use for most of their text.
var damageFonts = [][2]string{
	{"tttgbnumber", "resources/font/tttgbnumber.ttf"},
	{"HYWenHei-55W", "resources/font/HYWenHei-55W.ttf"},
}

// damageArea is one column of the damage area table: upstream's label, the
// area's key in the calculation and the digits it shows.
type damageArea struct {
	label, key string
	digits     int
}

var (
	damageAreas = []damageArea{
		{"基础区", "BasicArea", 0}, {"暴击区", "CriticalArea", 2}, {"增伤区", "BoostArea", 2},
		{"易伤区", "VulnerabilityArea", 2}, {"抗性区", "ResistanceArea", 2},
	}
	defenceArea  = damageArea{"防御区", "DefenceArea", 4}
	sheerArea    = damageArea{"贯穿增伤区", "SheerBoostArea", 2}
	anomalyAreas = []damageArea{{"异常精通区", "AnomalyProficiencyArea", 2}, {"异常增伤区", "AnomalyBoostArea", 2}, {"等级区", "LevelArea", 2}}
)

// Damage draws 伤害 the way ZZZ-Plugin's panel/damage page does: the panel
// card's agent, properties and W-Engine, every skill's damage with the chosen
// one marked, the chosen skill's damage areas and useful buffs, and the
// expected damage change of swapping sub and main stats.
func Damage(context app.ImageContext, image app.DamageImage) (app.Image, bool) {
	official, result := image.Panel.Official, image.Result
	if official == nil || result.Skill < 0 || result.Skill >= len(result.Damages) {
		return app.Image{}, false
	}
	card := newAgentCard(context)
	for _, font := range damageFonts {
		if resource, ok := context.ArtworkResource(font[0], "yunzai-genshin", font[1]); ok {
			card.resources = append(card.resources, resource)
		}
	}
	data := card.basic(official, image.UID, image.Portrait, result.Weights)
	name := app.Text(official["name_mi18n"])
	damages := []any{}
	for index, damage := range result.Damages {
		row := map[string]any{"index": index + 1, "name": damage.Name, "expect": fixed(damage.Expected, 0), "current": index == result.Skill}
		if damage.Critical != 0 {
			row["crit"] = fixed(damage.Critical, 0)
		}
		damages = append(damages, row)
	}
	areas := append([]damageArea{}, damageAreas...)
	if result.Sheer {
		areas = append(areas, sheerArea)
	} else {
		areas = append(areas, defenceArea)
	}
	buffs := []any{}
	for index, buff := range result.Buffs {
		buffs = append(buffs, map[string]any{"index": index + 1, "name": buff.Name, "source": buff.Source, "type": buff.Type, "value": buffValue(buff.Value)})
	}
	damage := map[string]any{
		"level": app.Int(official["level"]), "skill": result.Damages[result.Skill].Name, "rows": damages,
		"command": context.Game.Prefix + name + "伤害" + strconv.Itoa(result.Skill+1), "hint": context.Game.Prefix + name + "伤害123",
		"areas": areaCells(result.Areas, areas), "buffs": buffs,
	}
	if result.Anomaly {
		damage["anomaly"] = areaCells(result.Areas, anomalyAreas)
	}
	data["damage"] = damage
	// Upstream shows a comparison only when it has more than its control row.
	if len(result.Sub.Rows) > 1 {
		data["sub"] = differenceTable(result.Sub)
	}
	if len(result.Main.Rows) > 1 {
		data["main"] = differenceTable(result.Main)
	}
	return app.Image{Template: "damage", Data: data, Resources: card.resources}, true
}

// areaCells shows the areas as upstream does, a missing or zero area as 1.
func areaCells(values map[string]float64, areas []damageArea) []any {
	cells := []any{}
	for _, area := range areas {
		value := values[area.key]
		if value == 0 {
			value = 1
		}
		cells = append(cells, map[string]any{"label": area.label, "value": fixed(value, area.digits)})
	}
	return cells
}

// buffValue shows a buff as upstream does: below 2 as a whole percentage,
// else as the number with at most two decimals.
func buffValue(value float64) string {
	switch {
	case value < 2:
		return fixed(value*100, 0) + "%"
	case math.Mod(value, 1) == 0:
		return strconv.FormatFloat(value, 'f', -1, 64)
	}
	return fixed(value, 2)
}

// differenceTable is a stat comparison with each change signed and classed
// as upstream colours it.
func differenceTable(table app.DamageTable) map[string]any {
	columns := []any{}
	for _, column := range table.Columns {
		columns = append(columns, map[string]any{"name": column.Name, "value": column.Value})
	}
	rows := []any{}
	for _, row := range table.Rows {
		cells := []any{}
		for _, difference := range row.Differences {
			cell := map[string]any{"class": "zero", "text": fixed(difference, 0)}
			if difference > 0 {
				cell["class"], cell["text"] = "positive", "+"+fixed(difference, 0)
			} else if difference < 0 {
				cell["class"] = "negative"
			}
			cells = append(cells, cell)
		}
		rows = append(rows, map[string]any{"name": row.Name, "value": row.Value, "cells": cells})
	}
	return map[string]any{"columns": columns, "rows": rows}
}

// fixed formats as JavaScript's toFixed, which rounds an exact tie away from
// zero where strconv rounds it to even.
func fixed(value float64, digits int) string {
	scaled := new(big.Float).SetPrec(256).SetFloat64(value)
	scaled.Mul(scaled, big.NewFloat(math.Pow10(digits)))
	whole, _ := scaled.Int(nil)
	rest := new(big.Float).Sub(scaled, new(big.Float).SetInt(whole))
	if rest.Abs(rest).Cmp(big.NewFloat(0.5)) == 0 {
		value = math.Nextafter(value, math.Copysign(math.Inf(1), value))
	}
	return strconv.FormatFloat(value, 'f', digits, 64)
}
