package app

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RayleaBot/plugin-zzz/internal/reference"
)

func TestConditionScopeAndPresetRevision(t *testing.T) {
	a := pluginApp(t)
	for _, v := range []any{map[string]any{"bonuses": map[string]any{"atkPct": nil}}, map[string]any{"bonuses": map[string]any{"script": 1}}, map[string]any{"enemy_resistance": 101}, map[string]any{"team": []any{map[string]any{"character_id": "1011"}}}, map[string]any{"team": []any{map[string]any{"character_id": "1021"}, map[string]any{"character_id": "1021"}}}} {
		if _, err := a.validateConditions("1011", v); err == nil {
			t.Fatal("invalid condition accepted", v)
		}
	}
	conditions := map[string]any{"team": []any{map[string]any{"character_id": "1021", "name": "untrusted", "bonuses": map[string]any{"atkPct": 25}}}, "bonuses": map[string]any{"cpct": 10}}
	out, err := a.buildPresetAction("build.presets.save", map[string]any{"name": "日常队伍", "character_id": "1011", "conditions": conditions, "account_ref": "must-not-save", "uid": "must-not-save"})
	if err != nil {
		t.Fatal(err)
	}
	p := out["preset"].(BuildPreset)
	if p.Conditions.Team[0].Name == "untrusted" {
		t.Fatal("caller supplied team identity used")
	}
	reloaded := &BuildPresetStore{Path: a.BuildPresets.Path}
	list, err := reloaded.List()
	if err != nil || len(list) != 1 {
		t.Fatal(err)
	}
	if _, err = a.buildPresetAction("build.presets.remove", map[string]any{"ref": p.Ref, "revision": p.Revision + 1}); err == nil {
		t.Fatal("stale remove accepted")
	}
	bytes, _ := os.ReadFile(filepath.Clean(a.BuildPresets.Path))
	if string(bytes) == "" || strings.Contains(string(bytes), "must-not-save") {
		t.Fatal("credential scope retained")
	}
	if _, err = a.buildPresetAction("build.presets.remove", map[string]any{"ref": p.Ref, "revision": p.Revision}); err != nil {
		t.Fatal(err)
	}
}
func TestCustomConditionsUseBaseAttributePercent(t *testing.T) {
	for _, spec := range []struct{ game, key, file, attr, base string }{{"zzz", "zzz_1011", "", "ATK", "ATKBase"}} {
		t.Run(spec.game, func(t *testing.T) {
			engine := calcEngine(t)
			metadata := engine.Metadata()
			raw := pluginFile(t, "internal/assets/testdata/calc-vectors.json")
			var cases []struct {
				Key   string
				Input map[string]any
			}
			if json.Unmarshal(raw, &cases) != nil {
				t.Fatal("invalid vector")
			}
			var input map[string]any
			var record reference.Character
			for _, c := range cases {
				if c.Key == spec.key {
					input = c.Input
					break
				}
			}
			for _, c := range metadata.Characters {
				if c.Key == spec.key {
					record = c
					break
				}
			}
			if input == nil || record.Key == "" {
				t.Fatal("fixture missing")
			}
			conditions := map[string]any{"disable_character": true, "disable_weapon": true, "disable_equipment": true, "bonuses": map[string]any{}}
			input["conditions"] = conditions
			calculate := func() BuildResult {
				raw, err := engine.Run(t.Context(), record, input)
				if err != nil {
					t.Fatal(err)
				}
				var out BuildResult
				if json.Unmarshal(raw, &out) != nil || out.Candidate == nil {
					t.Fatal("missing candidate")
				}
				return out
			}
			before := calculate()
			conditions["bonuses"] = map[string]any{"atkPct": 100}
			after := calculate()
			attrs := input["attributes"].(map[string]any)
			ratio := (attrs[spec.attr].(float64) + attrs[spec.base].(float64)) / attrs[spec.attr].(float64)
			a, b := before.Candidate.Results[0].Expected, after.Candidate.Results[0].Expected
			if a == nil || b == nil || *a <= 0 || math.Abs(*b / *a - ratio) > 1e-7 {
				t.Fatalf("base percent mismatch: %v %v ratio %v", a, b, ratio)
			}
		})
	}
}
