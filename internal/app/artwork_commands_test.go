package app

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/zzz/internal/artwork"
)

// artworkMirror serves every file of an on-demand source but those named
// missing.
func artworkMirror(t *testing.T) *artwork.Store {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "missing") {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("png"))
	}))
	t.Cleanup(server.Close)
	return &artwork.Store{Root: t.TempDir(), Sources: []artwork.Source{{ID: "mirror", Mirrors: []string{server.URL + "/"}}}}
}

func TestDownloadAllCountsEveryFile(t *testing.T) {
	a := &App{Artwork: artworkMirror(t)}
	groups := []ArtworkGroup{{Label: "角色图", Source: "mirror", Files: []string{"a.png", "missing-b.png", "c.png"}}, {Label: "武器图", Source: "mirror", Files: []string{"w.png"}}}
	success, failed := a.fetchArtwork(t.Context(), groups)
	want := "资源下载完成（成功包含已下载资源）\n角色图：总数3，成功2，失败1\n武器图：总数1，成功1，失败0\n注：下载失败可能缘于该资源尚处于内测中"
	if got := artworkSummary(groups, success, failed); got != want {
		t.Fatalf("summary %q", got)
	}
}

// 下载全部资源, run through the SDK as the host runs it: the command's event
// moves to the background, says that the download starts and answers the
// counts once every file is fetched. Meanwhile the chat is free, and another
// 下载全部资源 answers that one is running.
func TestDownloadAllFetchesEveryFileInOneDetachedEvent(t *testing.T) {
	a := pluginApp(t)
	a.Artwork = artworkMirror(t)
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
	var again map[string]any
	host.onSend = func(rayleabot.MessageSendRequest) {
		if again == nil {
			again, _ = host.message("%下载全部资源", "下载全部资源")
		}
	}
	end, actions := host.message("%下载全部资源", "下载全部资源")
	if end["type"] != "result" || !slices.Equal(host.detached, []string{"chat-1"}) || actionAt(actions, "event.detach") > actionAt(actions, "message.send") || len(host.created) != 0 {
		t.Fatalf("the command ended with %v, actions %v", end, actions)
	}
	if terminalText(again) != "下载任务正在进行中，请稍后再试" {
		t.Fatalf("a second download ended with %v", again)
	}
	if len(host.sent) != 2 || sentText(host.sent[0].Message) != "开始下载全部资源：代理人、音擎、驱动盘、邦布图片等，请耐心等待……" || !strings.Contains(sentText(host.sent[1].Message), "代理人：总数20，成功18，失败2") {
		t.Fatalf("sent %v", host.sent)
	}
}
