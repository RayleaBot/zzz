package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/RayleaBot/zzz/internal/artwork"
)

func TestImageResourcesAddEachFileOnce(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"upstream/a.png", "official/example.com/icon.png"} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("png"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	store := &artwork.Store{Root: root, Sources: []artwork.Source{{ID: "official", Mirrors: []string{"https://"}}}}
	resources := &ImageResources{Context: ImageContext{Artwork: store}}
	if id := resources.Artwork("a", "upstream", "a.png"); id != "a" {
		t.Fatalf("artwork = %q", id)
	}
	if id := resources.Artwork("again", "upstream", "a.png"); id != "a" {
		t.Errorf("a file added twice keeps its first ID, got %q", id)
	}
	first := resources.URL("official", "https://example.com/icon.png?x=1")
	if first == "" || resources.URL("official", "https://example.com/icon.png?x=1") != first {
		t.Errorf("url = %q", first)
	}
	if resources.Artwork("missing", "upstream", "b.png") != "" || resources.URL("official", "http://example.com/icon.png") != "" {
		t.Error("missing or insecure images are left out")
	}
	if len(resources.List) != 2 || resources.List[1].Path != "assets/official/example.com/icon.png" {
		t.Errorf("list = %+v", resources.List)
	}
}

func TestPrefetchDownloadsSeveralAtOnce(t *testing.T) {
	var inFlight, most atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		now := inFlight.Add(1)
		for {
			seen := most.Load()
			if now <= seen || most.CompareAndSwap(seen, now) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		inFlight.Add(-1)
		_, _ = w.Write([]byte("png"))
	}))
	defer server.Close()
	store := &artwork.Store{Root: t.TempDir(), Sources: []artwork.Source{{ID: "official", Mirrors: []string{server.URL + "/"}}}}
	resources := &ImageResources{Context: ImageContext{Artwork: store}}
	urls := []any{}
	for index := range 6 {
		urls = append(urls, "https://example.com/"+strconv.Itoa(index)+".png")
	}
	resources.Prefetch("official", append(urls, urls[0], "http://example.com/plain.png")...)
	if most.Load() < 2 {
		t.Errorf("at most %d downloads ran at once", most.Load())
	}
	// Every URL now reads from the cache.
	for _, url := range urls {
		if id := resources.URL("official", url); id == "" {
			t.Errorf("%v was not cached", url)
		}
	}
}

// The host rejects an image with more than 512 resources or more than 16
// fetched from URLs; the extra pictures are left out instead.
func TestImageResourcesStayWithinHostLimits(t *testing.T) {
	resources := &ImageResources{}
	for index := range 20 {
		resources.Remote("avatar-"+strconv.Itoa(index), "https://example.com/"+strconv.Itoa(index))
	}
	if len(resources.List) != 16 || resources.Remote("late", "https://example.com/late") != "" {
		t.Fatalf("remote resources = %d", len(resources.List))
	}
	root := t.TempDir()
	for index := range 600 {
		name := strconv.Itoa(index) + ".png"
		if err := os.WriteFile(filepath.Join(root, name), []byte("png"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	resources.Context = ImageContext{Artwork: &artwork.Store{Root: filepath.Dir(root)}}
	for index := range 600 {
		resources.Artwork("file-"+strconv.Itoa(index), filepath.Base(root), strconv.Itoa(index)+".png")
	}
	if len(resources.List) != 512 {
		t.Fatalf("resources = %d", len(resources.List))
	}
}
