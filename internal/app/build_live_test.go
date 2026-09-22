//go:build manual_smoke

package app

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestLiveReferenceBuild(t *testing.T) {
	directory := os.Getenv("RAYLEA_PANEL_SMOKE_INPUT_DIR")
	if directory == "" {
		t.Skip("explicit credential-free business DTO directory required")
	}
	for _, game := range []string{"genshin", "starrail"} {
		t.Run(game, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(directory, game+"-panel-business.json"))
			if err != nil {
				t.Fatal("missing business DTO")
			}
			var data map[string]any
			if json.Unmarshal(raw, &data) != nil {
				t.Fatal("invalid business DTO")
			}
			raw, err = os.ReadFile(filepath.Join("../../..", "plugin-"+game, "internal/assets/catalog.json"))
			if err != nil {
				t.Fatal(err)
			}
			catalog, err := ParseCatalog(raw)
			if err != nil {
				t.Fatal(err)
			}
			engine := calcEngine(t, game)
			counts := map[string]int{}
			ready, unchanged := 0, 0
			for _, panel := range NormalizePanels(game, QueryResult{Data: data}, catalog) {
				record, err := findBuildCharacter(engine, game, panel)
				if err != nil {
					counts["no_rule"]++
					continue
				}
				profile, err := buildProfile(engine, game, panel, record)
				if err != nil {
					counts[PublicError(err).Details["reason"].(string)]++
					continue
				}
				var input map[string]any
				_ = decodeObject(profile, &input)
				result, err := engine.Run(context.Background(), record, input)
				if err != nil {
					counts["reference_failure"]++
					t.Logf("reference error: %v", err)
					continue
				}
				var baseline BuildResult
				if json.Unmarshal(result, &baseline) != nil {
					t.Fatal("invalid result")
				}
				ready++
				profile.CandidateWeapon = &baseline.Baseline.Weapon
				compared, err := referenceBuild(context.Background(), engine, record, profile)
				if err != nil {
					t.Fatal("same weapon calculation failed")
				}
				if compared.Candidate == nil || len(compared.Baseline.Results) != len(compared.Candidate.Results) {
					t.Fatal("same weapon changed results")
				}
				for i, a := range compared.Baseline.Results {
					b := compared.Candidate.Results[i]
					if a.Expected != nil && (b.Expected == nil || math.Abs(*a.Expected-*b.Expected) > 1e-7) {
						t.Fatal("same weapon changed expected damage")
					}
				}
				unchanged++
			}
			t.Logf("automatic_ready=%d same_weapon_verified=%d unavailable=%v", ready, unchanged, counts)
			if ready == 0 {
				t.Fatal("no live character ready for automatic calculation")
			}
		})
	}
}
