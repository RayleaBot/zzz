package app

import (
	"context"
	"encoding/json"
	"os"
	"testing"
)

func TestZZZOfficialShapeBuild(t *testing.T) {
	raw, err := os.ReadFile("testdata/build-panels.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures map[string]map[string]any
	if json.Unmarshal(raw, &fixtures) != nil {
		t.Fatal("invalid fixtures")
	}
	if len(fixtures) != 44 {
		t.Fatal("missing character fixtures")
	}
	engine := calcEngine(t)
	for id, data := range fixtures {
		t.Run(id, func(t *testing.T) {
			panels := NormalizePanels(QueryResult{Data: map[string]any{"avatar_list": []any{data}}}, Catalog{})
			if len(panels) != 1 {
				t.Fatal("panel missing")
			}
			panel := panels[0]
			record, err := findBuildCharacter(engine, panel)
			if err != nil {
				t.Fatal(err)
			}
			profile, err := buildProfile(engine, panel, record)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = referenceBuild(context.Background(), engine, record, profile); err != nil {
				t.Fatal(err)
			}
			panel.Skills[0].HasSkillType = false
			if _, err = buildProfile(engine, panel, record); err == nil {
				t.Fatal("unknown skill type became normal attack")
			}
		})
	}
}

// The management page's simulation has no upstream counterpart and fails on
// the rule errors ZZZ-Plugin's calculation logs and skips.
func TestBuildFailsOnRuleErrors(t *testing.T) {
	panel := NormalizePanels(QueryResult{Data: map[string]any{"avatar_list": []any{damagePanel(t, "1011").Official}}}, Catalog{})[0]
	for name, rule := range map[string]string{"clean": cleanRule, "faulty": faultyRule} {
		engine := ruleGame(t, rule).Calc
		record, err := findBuildCharacter(engine, panel)
		if err != nil {
			t.Fatal(err)
		}
		profile, err := buildProfile(engine, panel, record)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = referenceBuild(context.Background(), engine, record, profile); (err != nil) != (name == "faulty") {
			t.Errorf("%s rule built with %v", name, err)
		}
	}
}
func TestGearTransferRejectsUnknownState(t *testing.T) {
	if _, err := buildGearSet(CharacterPanel{}); err == nil {
		t.Fatal("unknown gear treated as empty")
	}
	if gear, err := buildGearSet(CharacterPanel{EquipmentKnown: true}); err != nil || len(gear) != 0 {
		t.Fatal("explicit unequipped state rejected")
	}
}
