package assets

import (
	"testing"

	"github.com/RayleaBot/plugin-zzz/internal/app"
)

// The plugin starts exactly this way; a manifest or data change that the shared
// library rejects must fail here rather than when the host launches it.
func TestPluginStartsFromEmbeddedAssets(t *testing.T) {
	application, err := app.New(Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	application.Close()
}
