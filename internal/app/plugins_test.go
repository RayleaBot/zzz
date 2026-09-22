package app

import (
	"os"
	"testing"

	"github.com/RayleaBot/plugin-zzz/internal/reference"
	"github.com/RayleaBot/plugin-zzz/internal/reference/miao"
)

// Tests that need real game data read the sibling plugin checkouts, the same
// workspace layout the plugins are built from.
func pluginFile(t *testing.T, game, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile("../../../plugin-" + game + "/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func calcProfile(game string) reference.Profile {
	files := os.DirFS("../../../plugin-" + game + "/internal/assets/calc")
	if game != "zzz" {
		return miao.Profile(files)
	}
	// Mirrors plugin-zzz/internal/assets, which owns this runtime.
	script := func(path string) reference.Script { return reference.Script{Files: files, Path: path} }
	return reference.Profile{
		Files:   files,
		Prelude: []reference.Script{reference.Lodash(), script("data.js"), script("bootstrap.js"), script("common.js"), script("buffs.js")},
		Runner:  script("runner.js"),
	}
}

func calcEngine(t *testing.T, game string) *reference.Engine {
	t.Helper()
	engine, err := reference.New(calcProfile(game))
	if err != nil {
		t.Fatal(err)
	}
	return engine
}

// pluginAssets mirrors what the game plugin embeds.
func pluginAssets(t *testing.T, game string) Assets {
	t.Helper()
	optional := func(name string) []byte {
		raw, err := os.ReadFile("../../../plugin-" + game + "/internal/assets/data/" + name)
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		return raw
	}
	return Assets{
		Game:        pluginFile(t, game, "internal/assets/game.json"),
		Catalog:     pluginFile(t, game, "internal/assets/catalog.json"),
		Manifest:    pluginFile(t, game, "info.json"),
		Calc:        calcProfile(game),
		Resources:   pluginFile(t, game, "internal/assets/data/resources.json"),
		Simulation:  optional("simulation.json"),
		CloudPanels: optional("cloud-panels.json"),
		Enemies:     optional("enemies.json"),
	}
}

// pluginApp builds the app exactly as the game plugin does at start.
func pluginApp(t *testing.T, game string) *App {
	t.Helper()
	a, err := New(pluginAssets(t, game), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.Close)
	return a
}

// testGame is a game descriptor with its real calculation engine, for tests
// that reach panel or build calculations without a full plugin app.
func testGame(t *testing.T, game string) Game {
	t.Helper()
	data, err := parseGameData(pluginAssets(t, game))
	if err != nil {
		t.Fatal(err)
	}
	return Game{ID: game, Calc: calcEngine(t, game), Data: data}
}
