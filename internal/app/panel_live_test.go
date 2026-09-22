//go:build manual_smoke

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLivePanelBusinessDTO(t *testing.T) {
	directory := os.Getenv("RAYLEA_PANEL_SMOKE_INPUT_DIR")
	if directory == "" {
		t.Skip("explicit redacted business DTO directory required")
	}
	for _, game := range []string{"genshin", "starrail", "zzz"} {
		t.Run(game, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(directory, game+"-panel-business.json"))
			if err != nil {
				t.Fatal("could not read sanitized business data")
			}
			var data map[string]any
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.UseNumber()
			if decoder.Decode(&data) != nil {
				t.Fatal("invalid DTO")
			}
			catalogRaw, err := os.ReadFile(filepath.Join("../../..", "plugin-"+game, "internal/assets/catalog.json"))
			if err != nil {
				t.Fatal(err)
			}
			catalog, err := ParseCatalog(catalogRaw)
			if err != nil {
				t.Fatal(err)
			}
			app := pluginApp(t, game)
			panels := NormalizePanels(game, QueryResult{Data: data}, catalog)
			if len(panels) == 0 {
				t.Fatal("missing normalized panels")
			}
			scored := 0
			for _, panel := range panels {
				if len(panel.Stats) < 3 {
					t.Fatal("too few recognized live attributes")
				}
				for _, stat := range panel.Stats {
					if strings.HasPrefix(stat.Name, "属性 ") {
						t.Fatal("unmapped live property label")
					}
				}
				if _, err := app.scorePanel(context.Background(), panel); err == nil {
					scored++
				}
				if strings.Contains(PanelView(Game{ID: game}, []CharacterPanel{panel}, "").Text(), "暂无可") {
					t.Fatal("live panel disappeared from view")
				}
			}
			if scored == 0 {
				t.Fatal("no equipped live character could be scored")
			}
			t.Logf("normalized=%d scored=%d", len(panels), scored)
		})
	}
}
