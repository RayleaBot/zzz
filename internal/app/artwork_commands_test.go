package app

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
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
	clock := &fakeClock{at: time.Unix(1_800_000_000, 0)}
	a := &App{clock: clock, Artwork: &artwork.Store{Root: t.TempDir(), Sources: []artwork.Source{{ID: "mirror", Mirrors: []string{server.URL + "/"}}}}}
	groups := []ArtworkGroup{{Label: "角色图", Source: "mirror", Files: []string{"a.png", "missing-b.png", "c.png"}}, {Label: "武器图", Source: "mirror", Files: []string{"w.png"}}}
	job := &artworkJob{groups: groups, success: make([]int, 2), failed: make([]int, 2)}
	for index, group := range groups {
		for file := range group.Files {
			job.items = append(job.items, [2]int{index, file})
		}
	}
	// A slice that has already run out fetches nothing, a batch whose event
	// ran out is not counted, and the next slice resumes.
	if _, done := job.step(t.Context(), a, nil, clock.Now()); done || job.next != 0 {
		t.Fatal("a spent slice fetched files")
	}
	ended, cancel := context.WithCancel(t.Context())
	cancel()
	if _, done := job.step(ended, a, nil, clock.Now().Add(time.Minute)); done || job.next != 0 {
		t.Fatal("a batch its event ran out on was counted")
	}
	reply, done := job.step(t.Context(), a, nil, clock.Now().Add(time.Minute))
	want := "资源下载完成（成功包含已下载资源）\n角色图：总数3，成功2，失败1\n武器图：总数1，成功1，失败0\n注：下载失败可能缘于该资源尚处于内测中"
	if !done || sentText(rayleabot.MessageOut{Segments: reply}) != want {
		t.Fatalf("done %v, reply %v", done, reply)
	}
}

// 下载全部资源 with more files than its event fetches, run through the SDK as
// the host runs it: the command meanwhile answers that a download is running,
// and the triggers of the job, which carry no task ID, fetch the rest and
// answer the counts in the chat.
func TestDownloadAllFinishesOnTheHostsTriggers(t *testing.T) {
	a := pluginApp(t)
	clock := &fakeClock{at: time.Unix(1_800_000_000, 0)}
	a.clock = clock
	// Each file takes three seconds, so the event fetches one batch of 16.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = clock.Sleep(context.Background(), 3*time.Second)
		if strings.Contains(r.URL.Path, "missing") {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("png"))
	}))
	defer server.Close()
	a.Artwork = &artwork.Store{Root: t.TempDir(), Sources: []artwork.Source{{ID: "mirror", Mirrors: []string{server.URL + "/"}}}}
	files := []string{"missing-0.png", "missing-1.png"}
	for len(files) < 20 {
		files = append(files, fmt.Sprintf("%d.png", len(files)))
	}
	a.downloads = func(ImageContext) []ArtworkGroup {
		return []ArtworkGroup{{Label: "代理人", Source: "mirror", Files: files}}
	}
	host := newSDKHost(t, a, func(hostCall) (map[string]any, string) {
		t.Error("下载全部资源 asked the account service")
		return nil, "plugin.service_unavailable"
	})
	if end, _ := host.message("%下载全部资源", "下载全部资源"); end["type"] != "result" || len(host.sent) != 1 {
		t.Fatalf("the command ended with %v", end)
	}
	if end, _ := host.message("%下载全部资源", "下载全部资源"); terminalText(end) != "下载任务正在进行中，请稍后再试" {
		t.Fatalf("a second download started: %v", end)
	}
	ref := host.job(artworkTask)
	triggerUntilDone(t, clock, host, ref, time.Unix(1_800_000_000, 0), 5)
	if len(host.deleted) != 1 || host.deleted[0] != ref || len(host.sent) != 2 || !strings.Contains(sentText(host.sent[1].Message), "代理人：总数20，成功18，失败2") {
		t.Fatalf("deleted %v, answered %v", host.deleted, host.sent)
	}
}
