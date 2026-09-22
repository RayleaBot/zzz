//go:build manual_smoke

package artwork

import (
	"os"
	"strings"
	"testing"
	"time"
)

// TestLiveDownload downloads one real upstream archive, for example
// ARTWORK_ARCHIVE=https://codeload.github.com/ZZZure/ZZZ-Plugin/tar.gz/HEAD
// ARTWORK_INCLUDE=resources/ go test -tags manual_smoke ./artwork -run Live -v
func TestLiveDownload(t *testing.T) {
	archive := os.Getenv("ARTWORK_ARCHIVE")
	if archive == "" {
		t.Skip("set ARTWORK_ARCHIVE to a repository tar.gz URL")
	}
	var include []string
	if value := os.Getenv("ARTWORK_INCLUDE"); value != "" {
		include = strings.Split(value, ",")
	}
	store := &Store{Root: t.TempDir(), Sources: []Source{{ID: "live", Name: "live", Archives: []string{archive}, Include: include, Extensions: []string{".webp", ".png", ".jpg", ".jpeg", ".gif", ".json", ".yaml"}}}}
	defer store.Close()
	started := time.Now()
	if _, err := store.Start("live"); err != nil {
		t.Fatal(err)
	}
	for {
		status := store.Statuses()[0]
		if status.State != "downloading" {
			t.Logf("%+v in %s", status, time.Since(started).Round(time.Second))
			if status.State != "ready" || status.Error != "" {
				t.Fatal("download failed")
			}
			return
		}
		time.Sleep(time.Second)
	}
}
