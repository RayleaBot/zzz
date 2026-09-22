package app

import (
	"context"
	"errors"
	"github.com/RayleaBot/plugin-zzz/internal/localdata"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPublicActivityUsesActualWindowNotAnnouncementDisplayTime(t *testing.T) {
	for _, tc := range []struct {
		text       string
		start, end bool
	}{{`<p>〓活动时间〓</p><p>&lt;t&gt;2026/09/14 04:00&lt;/t&gt; ~ &lt;t&gt;2026/09/21 03:59&lt;/t&gt;</p>〓参与条件〓`, true, true}, {`活动时间：7.0版本更新后~2026/09/21 03:59`, false, true}, {`〓活动时间〓永久开放。〓参与条件〓2026/1/1 00:00 ~ 2026/2/1 00:00`, false, false}, {`活动时间：2026/10/3 12:00 ~ 2026/9/3 12:00`, false, false}} {
		a, b := activityWindow(publicText(tc.text))
		if !a.IsZero() != tc.start || !b.IsZero() != tc.end {
			t.Fatalf("incorrect window: %s %s %s", tc.text, a, b)
		}
	}
}
func TestPublicPostsUseOfficialCursorAndSanitizeContent(t *testing.T) {
	var last string
	c := PublicContentClient{HTTP: cloudDoer(func(r *http.Request) (*http.Response, error) {
		last = r.URL.Query().Get("last_id")
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"retcode":0,"data":{"posts":[{"post":{"post_id":"123","game_id":2,"subject":"a"}}],"is_last":true,"last_id":"3"}}`))}, nil
	})}
	result, err := c.posts(t.Context(), "genshin", ContentQuery{Kind: "search", Query: "角色", LastID: "2"})
	if err != nil || last != "2" || len(result["items"].([]PublicPost)) != 1 {
		t.Fatal(result, err, last)
	}
	p, err := normalizePublicPost("genshin", map[string]any{"post": map[string]any{"post_id": "123", "game_id": 0, "subject": "<b>标题</b>", "content": "<script>secret()</script><p>内容</p>", "images": []any{"https://upload-bbs.miyoushe.com/test.png?x-oss-process=foo", "http://other/a.png"}}}, true)
	if err != nil || p.Title != "标题" || len(p.Parts) != 1 || p.Parts[0] != "内容" || len(p.Images) != 1 || strings.Contains(p.Images[0], "?") {
		t.Fatal(p, err)
	}
	if _, err = publicPostID("starrail", "https://www.miyoushe.com/ys/article/123"); err == nil {
		t.Fatal("wrong game route accepted")
	}
	if _, err = publicPostID("genshin", "https://www.miyoushe.com/ys/article/123?cookie=bad"); err == nil {
		t.Fatal("query accepted")
	}
}
func TestPublicSubscriptionBaselineFailureDedupRestartAndExpiry(t *testing.T) {
	now := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC).UnixMilli()
	s := ContentSubscriptions{Path: filepath.Join(t.TempDir(), "subscriptions.json")}
	task := ContentSubscription{Ref: "task", Kind: "news", Enabled: true, ExpiresMS: now + int64(time.Hour/time.Millisecond)}
	if err := localdata.Write(s.Path, []ContentSubscription{task}); err != nil {
		t.Fatal(err)
	}
	rows := []PublicPost{{ID: "1", Title: "old"}}
	fetch := func(ContentSubscription) (map[string]any, error) { return map[string]any{"items": rows}, nil }
	sent := 0
	send := func(ContentSubscription, string) error { sent++; return errors.New("delivery failed") }
	if err := s.Tick(t.Context(), "task", now, fetch, send, Game{Name: "原神"}); err != nil || sent != 0 {
		t.Fatal("historical notification", err)
	}
	rows = append(rows, PublicPost{ID: "2", Title: "new"})
	now += int64(16 * time.Minute / time.Millisecond)
	_ = s.Tick(t.Context(), "task", now, fetch, send, Game{Name: "原神"})
	if sent != 1 {
		t.Fatal(sent)
	}
	s = ContentSubscriptions{Path: s.Path}
	now += int64(16 * time.Minute / time.Millisecond)
	_ = s.Tick(t.Context(), "task", now, fetch, send, Game{Name: "原神"})
	if sent != 1 {
		t.Fatal("failed send replayed")
	}
	now += int64(time.Hour / time.Millisecond)
	_ = s.Tick(t.Context(), "task", now, fetch, send, Game{Name: "原神"})
	_ = s.Tick(t.Context(), "task", now+int64(time.Hour/time.Millisecond), fetch, send, Game{Name: "原神"})
	if sent != 2 {
		t.Fatal("expiry replayed")
	}
}
func TestPublicExpirySkipsUnknownAndPastWindows(t *testing.T) {
	now := time.Now().UnixMilli()
	s := ContentSubscriptions{Path: filepath.Join(t.TempDir(), "s.json")}
	_ = localdata.Write(s.Path, []ContentSubscription{{Ref: "e", Kind: "expiry", Enabled: true, ExpiresMS: now + int64(2*time.Hour/time.Millisecond)}})
	sent := ""
	err := s.Tick(t.Context(), "e", now, func(ContentSubscription) (map[string]any, error) {
		return map[string]any{"items": []PublicActivity{{ID: "1", Title: "unknown", EndMS: now + 1000, TimeStatus: "unknown"}, {ID: "2", Title: "past", EndMS: now - 1000, TimeStatus: "explicit"}, {ID: "3", Title: "known", EndMS: now + 1000, TimeStatus: "explicit_end"}}}, nil
	}, func(_ ContentSubscription, text string) error { sent = text; return nil }, Game{})
	if err != nil || !strings.Contains(sent, "known") || strings.Contains(sent, "unknown") || strings.Contains(sent, "past") {
		t.Fatal(sent, err)
	}
}
func TestPublicLiveAnonymousSources(t *testing.T) {
	if os.Getenv("RAYLEA_PUBLIC_SMOKE") != "1" {
		t.Skip("explicit anonymous network smoke")
	}
	for _, g := range []string{"genshin", "starrail", "zzz"} {
		t.Run(g, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 25*time.Second)
			defer cancel()
			client := PublicContentClient{}
			cal, err := client.calendar(ctx, g)
			if err != nil {
				t.Fatal(err)
			}
			items := cal["items"].([]PublicActivity)
			known := 0
			for _, v := range items {
				if v.EndMS > 0 {
					known++
				}
			}
			news, err := client.posts(ctx, g, ContentQuery{Kind: "news", Type: 1})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("announcements=%d dated=%d news=%d", len(items), known, len(news["items"].([]PublicPost)))
		})
	}
}
