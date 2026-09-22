// Package miao is the calculation runtime shared by the Genshin Impact and
// Honkai: Star Rail plugins. common.js bundles the damage and attribute models of
// miao-plugin 7f6f1c84 (MIT); bootstrap.js and runner.js adapt them to the
// engine's runBuild entry point.
package miao

import (
	"embed"
	"io/fs"

	"github.com/RayleaBot/plugin-zzz/internal/reference"
)

//go:embed bootstrap.js common.js runner.js
var runtime embed.FS

// Profile runs a game directory holding catalog.json, game.js (the artifact and
// weapon buffs of that game) and characters/*.js.
func Profile(files fs.FS) reference.Profile {
	return reference.Profile{
		Files: files,
		Prelude: []reference.Script{
			reference.Lodash(),
			{Files: runtime, Path: "bootstrap.js"},
			{Files: runtime, Path: "common.js"},
			{Files: files, Path: "game.js"},
		},
		Runner:      reference.Script{Files: runtime, Path: "runner.js"},
		PassWeapons: true,
	}
}
