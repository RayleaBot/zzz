// Package assets holds the game data compiled into this plugin.
package assets

import (
	"embed"
	"io/fs"

	plugin "github.com/RayleaBot/plugin-zzz"
	"github.com/RayleaBot/plugin-zzz/internal/app"
	"github.com/RayleaBot/plugin-zzz/internal/banners"
	"github.com/RayleaBot/plugin-zzz/internal/images"
	"github.com/RayleaBot/plugin-zzz/internal/reference"
	"github.com/RayleaBot/plugin-zzz/internal/showcase"
)

//go:embed catalog.json
var catalog []byte

//go:embed game.json
var game []byte

// calc holds the pinned ZZZ-Plugin calculation runtime and character scripts,
// see calc/catalog.json.
//
//go:embed calc
var calc embed.FS

// CalcProfile runs the ZZZ calculator: data.js holds the upstream id maps,
// common.js the Calculator and BuffManager, buffs.js the weapon and drive disc
// effects, and bootstrap.js and runner.js adapt them to runBuild.
func CalcProfile() reference.Profile {
	files, err := fs.Sub(calc, "calc")
	if err != nil {
		panic(err) // the embedded directory name is fixed at build time
	}
	script := func(path string) reference.Script { return reference.Script{Files: files, Path: path} }
	return reference.Profile{
		Files:   files,
		Prelude: []reference.Script{reference.Lodash(), script("data.js"), script("bootstrap.js"), script("common.js"), script("buffs.js")},
		Runner:  script("runner.js"),
	}
}

// data holds this game's fixed reference data: materials, banners and
// birthdays, plus the optional feature data listed in Kit.
//
//go:embed data
var data embed.FS

func dataFile(name string) []byte {
	raw, err := data.ReadFile("data/" + name)
	if err != nil {
		panic(err) // the embedded file names are fixed at build time
	}
	return raw
}

// Load is the compiled-in game data and image builders the app runs with.
func Load() app.Assets {
	return app.Assets{Game: game, Catalog: catalog, Manifest: plugin.Info, Calc: CalcProfile(), Resources: dataFile("resources.json"), Images: images.Builders(), Queries: images.Queries(), Panel: images.Panel, Damage: images.Damage, Gacha: images.Gacha, Help: images.Help, MonthlyStats: images.MonthlyCollect, Calendar: images.Calendar, QueryRank: images.QueryRank, Entry: images.Entry, Showcase: showcase.Source, PanelList: images.PanelList, UIDList: images.UIDList, Banners: banners.Source, Downloads: images.Downloads}
}
