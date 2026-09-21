package assets

import (
	"testing"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

// The plugin starts exactly this way; a manifest or data change that the shared
// library rejects must fail here rather than when the host launches it.
func TestPluginStartsFromEmbeddedAssets(t *testing.T) {
	app, err := gamekit.New(Kit(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	app.Close()
}
