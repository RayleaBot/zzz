//go:build manual_smoke

package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLiveGrowthBusinessDTO(t *testing.T) {
	directory := os.Getenv("RAYLEA_GROWTH_SMOKE_INPUT_DIR")
	if directory == "" {
		t.Skip("explicit business DTO directory required")
	}
	for _, game := range []string{"genshin", "starrail"} {
		t.Run(game, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(directory, game+"-growth-business.json"))
			if err != nil {
				t.Fatal("could not load business DTO")
			}
			var value struct {
				CharacterID string         `json:"character_id"`
				Detail      map[string]any `json:"detail"`
				Computed    map[string]any `json:"computed"`
			}
			if json.Unmarshal(raw, &value) != nil {
				t.Fatal("invalid DTO")
			}
			plan, err := prepareGrowth(game, value.CharacterID, value.Detail)
			if err != nil {
				t.Fatal(err)
			}
			view := growthView(Game{ID: game}, plan, QueryResult{Data: value.Computed})
			if len(view.Rows) == 0 {
				t.Fatal("actual materials not parsed")
			}
			t.Logf("growth preparation and materials accepted: skills=%d material_types=%d", len(plan.Skills), len(view.Rows))
		})
	}
}
