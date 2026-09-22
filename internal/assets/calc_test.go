package assets

import (
	"context"
	"encoding/json"
	"github.com/RayleaBot/plugin-zzz/internal/reference"
	"math"
	"os"
	"testing"
)

type zzzResult struct {
	Baseline struct {
		Results []struct {
			ID       string   `json:"id"`
			Expected *float64 `json:"expected"`
			Critical *float64 `json:"critical"`
		} `json:"results"`
	} `json:"baseline"`
	Candidate *struct {
		Results []struct {
			ID       string   `json:"id"`
			Expected *float64 `json:"expected"`
			Critical *float64 `json:"critical"`
		} `json:"results"`
	} `json:"candidate"`
}

func zzzVectors(t *testing.T) []vector {
	t.Helper()
	raw, err := os.ReadFile("testdata/calc-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []vector
	if json.Unmarshal(raw, &cases) != nil || len(cases) != 176 {
		t.Fatal("incomplete ZZZ vectors")
	}
	return cases
}
func TestZZZReferenceProfiles(t *testing.T) {
	metadata := calcEngine(t).Metadata()
	records := map[string]reference.Character{}
	for _, r := range metadata.Characters {
		records[r.Key] = r
	}
	for _, v := range zzzVectors(t) {
		t.Run(v.Key, func(t *testing.T) {
			raw, err := calcEngine(t).Run(context.Background(), records[v.Key], v.Input)
			if err != nil {
				t.Fatal(err)
			}
			var result zzzResult
			if json.Unmarshal(raw, &result) != nil || len(result.Baseline.Results) != len(v.Results) {
				t.Fatal("result mismatch")
			}
			eq := func(a, b *float64) bool {
				if a == nil || b == nil {
					return a == nil && b == nil
				}
				return math.Abs(*a-*b) < 1e-8*math.Max(1, math.Abs(*b))
			}
			for i, a := range result.Baseline.Results {
				if !eq(a.Expected, v.Results[i].Expected) || !eq(a.Critical, v.Results[i].Critical) {
					t.Fatalf("scenario %d differs", i)
				}
			}
		})
	}
}
func TestZZZWeaponsAndIdenticalEquipment(t *testing.T) {
	metadata := calcEngine(t).Metadata()
	records := map[string]reference.Character{}
	for _, r := range metadata.Characters {
		records[r.Key] = r
	}
	byType := map[string]vector{}
	for _, v := range zzzVectors(t) {
		r := records[v.Key]
		if v.Input["level"].(float64) == 60 {
			byType[r.WeaponType] = v
		}
	}
	for _, weapon := range metadata.Weapons {
		if weapon.Game != "zzz" {
			continue
		}
		t.Run(weapon.ID, func(t *testing.T) {
			v, ok := byType[weapon.Type]
			if !ok {
				if weapon.Type == "7" {
					t.Skip("fixed source has no profession 7 character; passive is verified by isolated numerical tests")
				}
				t.Fatal("no representative character")
			}
			input := map[string]any{}
			for k, value := range v.Input {
				input[k] = value
			}
			input["candidate_weapon"] = map[string]any{"id": weapon.ID, "level": 60, "promote": 5, "refinement": 5}
			raw, err := calcEngine(t).Run(context.Background(), records[v.Key], input)
			if err != nil {
				t.Fatal(err)
			}
			var result zzzResult
			_ = json.Unmarshal(raw, &result)
			if result.Candidate == nil || len(result.Candidate.Results) == 0 {
				t.Fatal("candidate absent")
			}
		})
	}
	for _, v := range zzzVectors(t) {
		if v.Input["level"].(float64) != 60 {
			continue
		}
		t.Run("unchanged/"+v.Key, func(t *testing.T) {
			input := map[string]any{}
			for k, value := range v.Input {
				input[k] = value
			}
			input["candidate_weapon"] = input["weapon"]
			input["candidate_equipment"] = input["equipment"]
			raw, err := calcEngine(t).Run(context.Background(), records[v.Key], input)
			if err != nil {
				t.Fatal(err)
			}
			var result zzzResult
			_ = json.Unmarshal(raw, &result)
			before, _ := json.Marshal(result.Baseline.Results)
			after, _ := json.Marshal(result.Candidate.Results)
			if string(before) != string(after) {
				t.Fatal("same build changed damage, likely mutable skill cache")
			}
		})
	}
}

type vector struct {
	Key     string         `json:"key"`
	Input   map[string]any `json:"input"`
	Results []struct {
		Expected *float64 `json:"expected"`
		Text     string   `json:"text"`
		Critical *float64 `json:"critical"`
	} `json:"results"`
}

// calcEngine runs this plugin's embedded scripts exactly as the plugin does.
func calcEngine(t *testing.T) *reference.Engine {
	t.Helper()
	engine, err := reference.New(Load().Calc)
	if err != nil {
		t.Fatal(err)
	}
	return engine
}
