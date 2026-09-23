package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RayleaBot/plugin-zzz/internal/artwork"
)

func TestDownloadAllCountsEveryFileAndResumes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "missing") {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("png"))
	}))
	defer server.Close()
	store := &artwork.Store{Root: t.TempDir(), Sources: []artwork.Source{{ID: "mirror", Mirrors: []string{server.URL + "/"}}}}
	groups := []ArtworkGroup{{Label: "角色图", Source: "mirror", Files: []string{"a.png", "missing-b.png", "c.png"}}, {Label: "武器图", Source: "mirror", Files: []string{"w.png"}}}
	job := &artworkJob{groups: groups, success: make([]int, 2), failed: make([]int, 2)}
	for index, group := range groups {
		for file := range group.Files {
			job.items = append(job.items, [2]int{index, file})
		}
	}
	// A slice that has already run out fetches nothing, and the next resumes.
	if job.step(t.Context(), store, time.Now().Add(-time.Second)) {
		t.Fatal("a spent slice fetched files")
	}
	if !job.step(t.Context(), store, time.Now().Add(time.Minute)) {
		t.Fatal("job did not finish")
	}
	want := "资源下载完成（成功包含已下载资源）\n角色图：总数3，成功2，失败1\n武器图：总数1，成功1，失败0\n注：下载失败可能缘于该资源尚处于内测中"
	if job.summary() != want {
		t.Fatalf("summary = %q", job.summary())
	}
}
