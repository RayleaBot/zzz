package images

import (
	"os"
	"path/filepath"
	"testing"

	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/game-plugin-kit/artwork"
)

func TestCalendarShowsTheOfficialCalendarPicture(t *testing.T) {
	root := t.TempDir()
	cached := filepath.Join(root, "mihoyo", "example.invalid", "calendar.png")
	if err := os.MkdirAll(filepath.Dir(cached), 0o755); err != nil {
		t.Fatal(err)
	}
	// A 1x1 PNG stands in for the cached picture.
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\rIDATx\x9cc\xf8\x0f\x00\x00\x01\x01\x00\x05\x18\xd8N\x00\x00\x00\x00IEND\xaeB`\x82")
	if err := os.WriteFile(cached, png, 0o644); err != nil {
		t.Fatal(err)
	}
	context := gamekit.ImageContext{Artwork: &artwork.Store{Root: root, Sources: []artwork.Source{{ID: "mihoyo", Mirrors: []string{"https://"}}}}}
	calendar := gamekit.CalendarImage{Announcements: gamekit.Announcements{Contents: []any{
		map[string]any{"title": "版本日历", "subtitle": "版本说明", "content": `<img src="https://example.invalid/other.png">`},
		map[string]any{"title": "9月日历", "subtitle": "活动日历", "content": `<p><img class="x" src="https://example.invalid/calendar.png" /></p>`},
	}}}
	image, ok := Calendar(context, calendar)
	if !ok || image.Template != "calendar" || len(image.Resources) != 1 {
		t.Fatalf("image = %v, %v", image, ok)
	}
	// Without the calendar announcement the text list answers.
	calendar.Announcements.Contents = calendar.Announcements.Contents[:1]
	if _, ok := Calendar(context, calendar); ok {
		t.Error("drew a calendar from another announcement")
	}
}
