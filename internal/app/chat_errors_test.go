package app

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"context"
	"errors"
	"github.com/RayleaBot/plugin-zzz/internal/artwork"
	"net"
)

func TestArtworkStatusDoesNotSendLocalErrors(t *testing.T) {
	var payload bytes.Buffer
	compressed := gzip.NewWriter(&payload)
	archive := tar.NewWriter(compressed)
	if err := archive.WriteHeader(&tar.Header{Name: "fixture-main/assets/fixture.png", Mode: 0o600, Size: 3}); err != nil {
		t.Fatal(err)
	}
	if _, err := archive.Write([]byte("png")); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(payload.Bytes()) }))
	defer server.Close()

	blocked := filepath.Join(t.TempDir(), "private-fixture-root")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := pluginApp(t)
	a.Artwork = &artwork.Store{Root: blocked, Sources: []artwork.Source{{
		ID: "fixture", Name: "fixture", Archives: []string{server.URL}, Include: []string{"assets/"}, Extensions: []string{".png"},
	}}}
	if _, err := a.Artwork.Start("fixture"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(4 * time.Second)
	for {
		status := a.Artwork.Statuses()[0]
		if status.Error != "" {
			if !strings.Contains(status.Error, blocked) {
				t.Fatalf("expected filesystem diagnostic, got %q", status.Error)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("download did not fail")
		}
		time.Sleep(5 * time.Millisecond)
	}

	host := newSDKHost(t, a, nil)
	terminal, _ := host.groupMessage("素材状态", "素材状态")
	text := terminalText(terminal)
	if terminal["action"] != "message.send" || !strings.Contains(text, "fixture") || !strings.Contains(text, "失败") {
		t.Fatalf("missing source failure notice: %v", terminal)
	}
	for _, private := range []string{blocked, "private-fixture-root", server.URL} {
		if strings.Contains(text, private) {
			t.Fatalf("chat leaked a diagnostic: %q", text)
		}
	}
	if !strings.Contains(a.Artwork.Statuses()[0].Error, blocked) {
		t.Fatal("management diagnostics were lost")
	}
}

func TestPoolFailureLogsDiagnosticsWithoutSendingThem(t *testing.T) {
	a := pluginApp(t)
	failure := &net.OpError{Op: "proxyconnect", Net: "tcp", Addr: &net.TCPAddr{IP: net.IPv4(10, 0, 0, 8), Port: 7890}, Err: errors.New("fixture connection refused")}
	a.banners = func(context.Context) ([]PoolRecord, string, error) { return nil, "", failure }
	host := newSDKHost(t, a, nil)
	terminal, actions := host.groupMessage("卡池", "卡池")
	text := terminalText(terminal)
	if terminal["action"] != "message.send" || !strings.Contains(text, "失败") {
		t.Fatalf("missing failure notice: %v", terminal)
	}
	for _, private := range []string{"10.0.0.8", "7890", "proxyconnect", "fixture connection refused"} {
		if strings.Contains(text, private) {
			t.Fatalf("chat leaked network diagnostics: %q", text)
		}
	}
	logged := false
	for _, action := range actions {
		if action.Name == "logger.write" && asText(asObject(action.Data["fields"])["error"]) == failure.Error() {
			logged = true
		}
	}
	if !logged {
		t.Fatal("network diagnostics were not recorded in management logs")
	}
}
