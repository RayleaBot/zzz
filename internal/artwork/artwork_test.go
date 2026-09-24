package artwork

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func archive(t *testing.T, commit string, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	zipped := gzip.NewWriter(&buffer)
	entries := tar.NewWriter(zipped)
	if commit != "" {
		if err := entries.WriteHeader(&tar.Header{Typeflag: tar.TypeXGlobalHeader, Name: "pax_global_header", PAXRecords: map[string]string{"comment": commit}}); err != nil {
			t.Fatal(err)
		}
	}
	for name, content := range files {
		if err := entries.WriteHeader(&tar.Header{Typeflag: tar.TypeReg, Name: "repo-" + commit + "/" + name, Mode: 0o644, Size: int64(len(content))}); err != nil {
			t.Fatal(err)
		}
		if _, err := entries.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := entries.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zipped.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func wait(t *testing.T, store *Store, id string) Status {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, status := range store.Statuses() {
			if status.ID == id && status.State != "downloading" {
				return status
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("download did not finish")
	return Status{}
}

func TestDownloadKeepsIncludedImagesAndFallsBackToTheNextArchive(t *testing.T) {
	var current atomic.Pointer[[]byte]
	first := archive(t, "abc123", map[string]string{
		"resources/meta-gs/character/芙宁娜/imgs/face.webp": "face",
		"resources/meta-gs/character/芙宁娜/calc.js":        "script",
		"resources/meta-sr/character/花火/imgs/face.webp":  "other game",
		"resources/common/bg/bg-hydro.webp":              "bg",
		"resources/common/CON.webp":                      "device name",
	})
	current.Store(&first)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/mirror.tar.gz" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(*current.Load())
	}))
	defer server.Close()
	store := &Store{Root: filepath.Join(t.TempDir(), "assets"), Sources: []Source{{
		ID: "miao-plugin", Name: "喵喵插件素材",
		Archives:   []string{server.URL + "/mirror.tar.gz", server.URL + "/github.tar.gz"},
		Include:    []string{"resources/meta-gs/", "resources/common/"},
		Extensions: []string{".webp", ".png"},
	}}}
	defer store.Close()

	if status := store.Statuses()[0]; status.State != "missing" {
		t.Fatalf("before download: %+v", status)
	}
	if _, err := store.Start("miao-plugin"); err != nil {
		t.Fatal(err)
	}
	status := wait(t, store, "miao-plugin")
	if status.State != "ready" || status.Commit != "abc123" || status.Files != 2 || status.Error != "" {
		t.Fatalf("after download: %+v", status)
	}
	if name, ok := store.File("miao-plugin", "resources/meta-gs/character/芙宁娜/imgs/face.webp"); !ok || name != "assets/miao-plugin/resources/meta-gs/character/芙宁娜/imgs/face.webp" {
		t.Fatalf("File = %q %v", name, ok)
	}
	for _, name := range []string{"resources/meta-gs/character/芙宁娜/calc.js", "resources/meta-sr/character/花火/imgs/face.webp", "resources/common/CON.webp", "../miao-plugin.json"} {
		if _, ok := store.File("miao-plugin", name); ok {
			t.Fatalf("%s should not be available", name)
		}
	}

	// An update replaces the previous copy as a whole.
	second := archive(t, "def456", map[string]string{"resources/common/bg/bg-pyro.webp": "bg"})
	current.Store(&second)
	if _, err := store.Start("miao-plugin"); err != nil {
		t.Fatal(err)
	}
	if status := wait(t, store, "miao-plugin"); status.Commit != "def456" || status.Files != 1 {
		t.Fatalf("after update: %+v", status)
	}
	if _, ok := store.File("miao-plugin", "resources/common/bg/bg-hydro.webp"); ok {
		t.Fatal("files from the previous download remain")
	}

	if err := store.Delete("miao-plugin"); err != nil {
		t.Fatal(err)
	}
	if status := store.Statuses()[0]; status.State != "missing" || status.Files != 0 {
		t.Fatalf("after delete: %+v", status)
	}
	entries, _ := os.ReadDir(store.Root)
	if len(entries) != 0 {
		t.Fatalf("leftover entries: %v", entries)
	}
}

func TestFailedUpdateKeepsThePreviousDownload(t *testing.T) {
	var fail atomic.Bool
	body := archive(t, "abc123", map[string]string{"images/a.png": "a"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write(body)
	}))
	defer server.Close()
	store := &Store{Root: t.TempDir(), Sources: []Source{{ID: "atlas", Archives: []string{server.URL}}}}
	defer store.Close()
	if _, err := store.Start("atlas"); err != nil {
		t.Fatal(err)
	}
	wait(t, store, "atlas")
	fail.Store(true)
	if _, err := store.Start("atlas"); err != nil {
		t.Fatal(err)
	}
	status := wait(t, store, "atlas")
	if status.State != "ready" || status.Commit != "abc123" || status.Error == "" {
		t.Fatalf("after failed update: %+v", status)
	}
	if _, ok := store.File("atlas", "images/a.png"); !ok {
		t.Fatal("previous download was lost")
	}
}

func TestFetchDownloadsOnDemandFilesOnce(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path != "/second/role/IconRole01.png" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png"))
	}))
	defer server.Close()
	store := &Store{Root: t.TempDir(), Sources: []Source{{ID: "zzzerouid", Name: "绝区零角色图", Mirrors: []string{server.URL + "/first/", server.URL + "/second/"}}}}
	path, ok := store.Fetch(t.Context(), "zzzerouid", "role/IconRole01.png")
	if !ok || path != "assets/zzzerouid/role/IconRole01.png" || hits.Load() != 2 {
		t.Fatalf("Fetch = %q %v after %d requests", path, ok, hits.Load())
	}
	if _, ok := store.Fetch(t.Context(), "zzzerouid", "role/IconRole01.png"); !ok || hits.Load() != 2 {
		t.Fatalf("cached file fetched again: %d requests", hits.Load())
	}
	// A missing file is not asked for again right away.
	if _, ok := store.Fetch(t.Context(), "zzzerouid", "role/IconRole99.png"); ok {
		t.Fatal("missing file reported present")
	}
	before := hits.Load()
	if _, ok := store.Fetch(t.Context(), "zzzerouid", "role/IconRole99.png"); ok || hits.Load() != before {
		t.Fatalf("paused file requested again: %d -> %d", before, hits.Load())
	}
	status := store.Statuses()[0]
	if status.State != "on_demand" || status.Files != 1 {
		t.Fatalf("status = %+v", status)
	}
	if _, err := store.Start("zzzerouid"); err == nil {
		t.Fatal("on-demand source started a download")
	}
}

func TestDownloadedFilesTakePrecedenceOverTheShippedOnes(t *testing.T) {
	shipped := t.TempDir()
	for name, content := range map[string]string{"miao/images/a.png": "shipped a", "miao/images/b.png": "shipped b", "miao.json": `{"commit":"pinned","files":2,"bytes":18}`} {
		file := filepath.Join(shipped, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	body := archive(t, "newer", map[string]string{"images/a.png": "new a", "images/c.png": "new c"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) }))
	defer server.Close()
	store := &Store{Root: t.TempDir(), Package: shipped, Sources: []Source{{ID: "miao", Archives: []string{server.URL}}}}
	defer store.Close()
	read := func(name string) string {
		data, err := store.Open("miao", name)
		if err != nil {
			return ""
		}
		return string(data)
	}
	if status := store.Statuses()[0]; status.State != "bundled" || status.Commit != "pinned" || status.Files != 2 || !store.Ready("miao") || read("images/a.png") != "shipped a" {
		t.Fatalf("shipped source: %+v", status)
	}
	if _, err := store.Start("miao"); err != nil {
		t.Fatal(err)
	}
	if status := wait(t, store, "miao"); status.State != "ready" || status.Commit != "newer" {
		t.Fatalf("after download: %+v", status)
	}
	// A download replaces shipped files it carries and leaves the rest.
	if read("images/a.png") != "new a" || read("images/b.png") != "shipped b" || read("images/c.png") != "new c" {
		t.Fatalf("files %q %q %q", read("images/a.png"), read("images/b.png"), read("images/c.png"))
	}
	if err := store.Delete("miao"); err != nil {
		t.Fatal(err)
	}
	if status := store.Statuses()[0]; status.State != "bundled" || read("images/a.png") != "shipped a" {
		t.Fatalf("after delete: %+v", status)
	}
}
