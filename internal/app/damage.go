package app

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/zzz/internal/reference"
)

// DamageResult is ZZZ-Plugin's damage calculation of one panel, as the
// runner's runDamage returns it: every skill's damage, and for the panel card
// the buffs the agent's rule shows there. For 伤害 it also has the chosen
// skill (an index into Damages) with its damage areas and useful buffs, the
// change of its expected damage when one sub or main stat is swapped for
// another, and the scoring weights that pick those stats and colour the
// property labels.
type DamageResult struct {
	Damages    []DamageRow        `json:"damages"`
	PanelBuffs []DamageBuff       `json:"panel_buffs"`
	Skill      int                `json:"skill"`
	Anomaly    bool               `json:"anomaly"`
	Sheer      bool               `json:"sheer"`
	Areas      map[string]float64 `json:"areas"`
	Buffs      []DamageBuff       `json:"buffs"`
	Sub        DamageTable        `json:"sub"`
	Main       DamageTable        `json:"main"`
	Weights    map[string]float64 `json:"weights"`
}

// DamageRow is one skill's damage; Critical is 0 for anomaly damage that
// cannot crit.
type DamageRow struct {
	Name     string  `json:"name"`
	Critical float64 `json:"critical"`
	Expected float64 `json:"expected"`
}

// DamageBuff is a buff with its value. The panel card names no source but
// shows the most a buff can reach, Max (0 for none).
type DamageBuff struct {
	Name   string  `json:"name"`
	Source string  `json:"source"`
	Type   string  `json:"type"`
	Value  float64 `json:"value"`
	Max    float64 `json:"max"`
}

// DamageTable compares stats: Columns are the stat added, and each row is a
// stat removed with the change of expected damage under every column.
type DamageTable struct {
	Columns []DamageStat          `json:"columns"`
	Rows    []DamageDifferenceRow `json:"rows"`
}

// DamageStat is a stat by upstream's short name and the value of one roll.
type DamageStat struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type DamageDifferenceRow struct {
	Name        string    `json:"name"`
	Value       string    `json:"value"`
	Differences []float64 `json:"differences"`
}

// DamageImage is what the 伤害 page draws: the panel with its official
// entry, the UID, a custom portrait as a path in the plugin data directory
// ("" for the default one) and the calculation.
type DamageImage struct {
	Panel    CharacterPanel
	UID      string
	Portrait string
	Result   DamageResult
}

// DamageImageBuilder draws the 伤害 page, or returns false to answer in text.
type DamageImageBuilder func(ImageContext, DamageImage) (Image, bool)

// damageSkill is the number ZZZ-Plugin reads after 伤害.
var damageSkill = regexp.MustCompile(`伤害([0-9]*)$`)

// damageCommand is ZZZ-Plugin's <角色>伤害[序号]: the panel is read as 面板
// reads it, and the page lists every skill's damage with the chosen one's
// areas, buffs and stat comparisons. As upstream, 原图 then sends the
// portrait the page shows.
func (a *App) damageCommand(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	if len(args) > 2 {
		return event.Result(map[string]any{"handled": false})
	}
	panel, uid, err := a.commandPanel(ctx, event, args)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	name := args[0]
	skill := ""
	if match := damageSkill.FindStringSubmatch(event.Event.Command()); match != nil {
		skill = match[1]
	}
	result, err := a.panelDamages(ctx, panel, &skill)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if len(result.Damages) == 0 {
		return event.SendText("暂无角色" + name + "的伤害计算")
	}
	view := DamageView(panel, uid, result)
	if a.damage != nil {
		image := DamageImage{Panel: panel, UID: uid, Portrait: a.PanelImages.Random(panel.ID), Result: result}
		if drawn, ok := a.damage(a.imageContext(ctx), image); ok {
			view.Image = &drawn
			if portrait := panelPortrait(drawn); portrait != "" {
				_ = a.rememberImage(event, portrait)
			}
		}
	}
	return a.sendView(ctx, event, view)
}

// panelDamages runs ZZZ-Plugin's damage calculation on a panel's official
// entry: for 伤害 with skill, the number written after it ("" for none), and
// for the panel card with none. A character the pinned calculation does not
// know has no damages, as upstream.
func (a *App) panelDamages(ctx context.Context, panel CharacterPanel, skill *string) (DamageResult, error) {
	var record reference.Character
	for _, candidate := range a.Game.Calc.Metadata().Characters {
		if candidate.ID == panel.ID {
			record = candidate
		}
	}
	if record.ID == "" {
		return DamageResult{}, nil
	}
	input := map[string]any{"avatar": panel.Official}
	if skill != nil {
		input["skill"] = *skill
	}
	raw, err := a.Game.Calc.Damage(ctx, record, input)
	var result DamageResult
	if err != nil || json.Unmarshal(raw, &result) != nil || len(result.Damages) > 0 && (result.Skill < 0 || result.Skill >= len(result.Damages)) {
		return DamageResult{}, buildUnavailable("reference_calculation")
	}
	return result, nil
}

// DamageView is the 伤害 reply in text: each skill's damage, and the skill
// the comparisons are for.
func DamageView(panel CharacterPanel, uid string, result DamageResult) View {
	return View{Title: panel.Name + " · 伤害统计", Subtitle: "UID " + uid, Rows: damageRows(result.Damages), Note: "以 " + result.Damages[result.Skill].Name + " 为计算目标"}
}

// damageRows lists each skill's damage in text, numbered as 伤害 takes them.
func damageRows(damages []DamageRow) []Row {
	rows := []Row{}
	for index, damage := range damages {
		value := "期望伤害 " + strconv.FormatFloat(damage.Expected, 'f', 0, 64)
		if damage.Critical != 0 {
			value = "暴击伤害 " + strconv.FormatFloat(damage.Critical, 'f', 0, 64) + " · " + value
		}
		rows = append(rows, Row{Label: strconv.Itoa(index+1) + ". " + damage.Name, Value: value})
	}
	return rows
}
