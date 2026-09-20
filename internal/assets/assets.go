package assets

import _ "embed"

//go:embed catalog.json
var Catalog []byte

//go:embed game.json
var Game []byte

//go:embed manifest.json
var Manifest []byte
