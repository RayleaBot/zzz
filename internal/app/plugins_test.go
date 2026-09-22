package app

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/RayleaBot/plugin-zzz/internal/reference"
)

// pluginFile reads a file of this plugin, for tests that need the shipped
// data.
func pluginFile(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile("../../" + name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func calcProfile() reference.Profile {
	files := os.DirFS("../../internal/assets/calc")
	// Mirrors internal/assets, which owns this runtime.
	script := func(path string) reference.Script { return reference.Script{Files: files, Path: path} }
	return reference.Profile{
		Files:   files,
		Prelude: []reference.Script{reference.Lodash(), script("data.js"), script("bootstrap.js"), script("common.js"), script("buffs.js")},
		Runner:  script("runner.js"),
	}
}

func calcEngine(t *testing.T) *reference.Engine {
	t.Helper()
	engine, err := reference.New(calcProfile())
	if err != nil {
		t.Fatal(err)
	}
	return engine
}

// pluginAssets mirrors what the plugin embeds.
func pluginAssets(t *testing.T) Assets {
	t.Helper()
	return Assets{
		Game:      pluginFile(t, "internal/assets/game.json"),
		Catalog:   pluginFile(t, "internal/assets/catalog.json"),
		Manifest:  pluginFile(t, "info.json"),
		Calc:      calcProfile(),
		Resources: pluginFile(t, "internal/assets/data/resources.json"),
	}
}

// pluginApp builds the app exactly as the plugin does at start.
func pluginApp(t *testing.T) *App {
	t.Helper()
	a, err := New(pluginAssets(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.Close)
	return a
}

// testGame is the game descriptor with its real calculation engine, for tests
// that reach panel or build calculations without a full app.
func testGame(t *testing.T) Game {
	t.Helper()
	data, err := parseGameData(pluginAssets(t))
	if err != nil {
		t.Fatal(err)
	}
	return Game{ID: "zzz", Calc: calcEngine(t), Data: data}
}

// httpDoer answers a client's requests with a function.
type httpDoer func(*http.Request) (*http.Response, error)

func (f httpDoer) Do(r *http.Request) (*http.Response, error) { return f(r) }

// jsonObject decodes a JSON object, keeping numbers as json.Number as the
// official answers are read.
func jsonObject(t *testing.T, s string) map[string]any {
	t.Helper()
	var v map[string]any
	d := json.NewDecoder(strings.NewReader(s))
	d.UseNumber()
	if err := d.Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}
