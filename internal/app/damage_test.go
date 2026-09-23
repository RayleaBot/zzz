package app

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"testing"
)

// damagePanel is a fixture agent's official entry as a kept panel.
func damagePanel(t *testing.T, id string) CharacterPanel {
	t.Helper()
	raw, err := os.ReadFile("testdata/build-panels.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures map[string]map[string]any
	if json.Unmarshal(raw, &fixtures) != nil || fixtures[id] == nil {
		t.Fatal("missing fixture " + id)
	}
	return CharacterPanel{ID: id, Official: fixtures[id]}
}

func TestDamagePicksTheSkillAsZZZPlugin(t *testing.T) {
	a := &App{Game: testGame(t)}
	ctx := context.Background()
	panel := damagePanel(t, "1011")
	chosen, err := a.panelDamages(ctx, panel, new(""))
	if err != nil || len(chosen.Damages) < 3 {
		t.Fatalf("damages = %+v, %v", chosen, err)
	}
	// Without a number the page is the one of the skill the comparison was
	// made for, the same as writing that skill's number.
	written, err := a.panelDamages(ctx, panel, new(strconv.Itoa(chosen.Skill+1)))
	if err != nil || !reflect.DeepEqual(written, chosen) {
		t.Errorf("no number chose %d, unlike writing its number", chosen.Skill)
	}
	last := len(chosen.Damages) - 1
	// Upstream turns 0 into the first skill and a number past the list into
	// the last; the number right after the last, which fails upstream, too.
	for word, want := range map[string]int{"1": 0, "0": 0, "3": 2, strconv.Itoa(last + 1): last, strconv.Itoa(last + 2): last, "99999999999": last} {
		result, err := a.panelDamages(ctx, panel, &word)
		if err != nil || result.Skill != want {
			t.Errorf("伤害%s chose %d, want %d (%v)", word, result.Skill, want, err)
		}
	}

	sub := chosen.Sub
	if len(sub.Rows) != len(sub.Columns) || sub.Columns[len(sub.Columns)-1] != (DamageStat{Name: "对照组", Value: "0"}) {
		t.Fatalf("sub = %+v", sub)
	}
	for index, row := range sub.Rows {
		if row.Name != sub.Columns[index].Name || len(row.Differences) != len(sub.Columns) || row.Differences[index] != 0 {
			t.Errorf("sub row %d = %+v", index, row)
		}
	}
	// Main stats only compare the ones the discs in slots 4 to 6 carry, with
	// the control row.
	main := chosen.Main
	if len(main.Rows) < 2 || len(main.Rows) > 4 || main.Rows[len(main.Rows)-1].Name != "对照组" {
		t.Errorf("main rows = %+v", main.Rows)
	}
	if chosen.Weights["20103"] == 0 || chosen.Areas["BasicArea"] == 0 || len(chosen.Buffs) == 0 {
		t.Errorf("weights %v, areas %v, buffs %v", chosen.Weights, chosen.Areas, chosen.Buffs)
	}
}

func TestPanelCardCalculatesAsDamage(t *testing.T) {
	a := &App{Game: testGame(t)}
	ctx := context.Background()
	// 丽娜's rule shows the Penetration Ratio she gives the team in the panel.
	panel := damagePanel(t, "1211")
	card, err := a.panelDamages(ctx, panel, nil)
	if err != nil || len(card.PanelBuffs) != 1 || card.PanelBuffs[0].Type != "穿透率" || card.PanelBuffs[0].Value == 0 || card.PanelBuffs[0].Max != 0.3 {
		t.Fatalf("card = %+v, %v", card, err)
	}
	// The card lists the damages 伤害 draws.
	page, err := a.panelDamages(ctx, panel, new(""))
	if err != nil || len(card.Damages) == 0 || !reflect.DeepEqual(card.Damages, page.Damages) {
		t.Errorf("card damages %+v, 伤害 %+v", card.Damages, page.Damages)
	}
}

func TestDamageKeepsToUpstreamRules(t *testing.T) {
	a := &App{Game: testGame(t)}
	ctx := context.Background()
	// 咚哒回声 has an effect only among the plugin's own W-Engine supplements,
	// which upstream's 伤害 does not have.
	panel := damagePanel(t, "1011")
	official := map[string]any{}
	for key, value := range panel.Official {
		official[key] = value
	}
	official["weapon"] = map[string]any{"id": 13018, "name": "咚哒回声", "level": 60, "star": 5, "rarity": "S", "properties": []any{}, "main_properties": []any{}}
	panel.Official = official
	result, err := a.panelDamages(ctx, panel, new(""))
	if err != nil || len(result.Damages) == 0 {
		t.Fatal(err)
	}
	for _, buff := range result.Buffs {
		if buff.Source == "音擎" {
			t.Errorf("a supplement took effect: %+v", buff)
		}
	}
	// An agent the pinned calculation only scores has no damage.
	official = map[string]any{}
	for key, value := range panel.Official {
		official[key] = value
	}
	official["id"] = 1341
	if result, err = a.panelDamages(ctx, CharacterPanel{ID: "1341", Official: official}, new("")); err != nil || len(result.Damages) != 0 {
		t.Errorf("score-only agent = %+v, %v", result, err)
	}
}
