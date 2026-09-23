package app

import (
	"context"
	"io"
	"net/http"
	"os"
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
	c := PublicContentClient{HTTP: httpDoer(func(r *http.Request) (*http.Response, error) {
		last = r.URL.Query().Get("last_id")
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"retcode":0,"data":{"posts":[{"post":{"post_id":"123","game_id":8,"subject":"a"}}],"is_last":true,"last_id":"3"}}`))}, nil
	})}
	result, err := c.posts(t.Context(), ContentQuery{Kind: "search", Query: "角色", LastID: "2"})
	if err != nil || last != "2" || len(result["items"].([]PublicPost)) != 1 {
		t.Fatal(result, err, last)
	}
	p, err := normalizePublicPost(map[string]any{"post": map[string]any{"post_id": "123", "game_id": 0, "subject": "<b>标题</b>", "content": "<script>secret()</script><p>内容</p>", "images": []any{"https://upload-bbs.miyoushe.com/test.png?x-oss-process=foo", "http://other/a.png"}}}, true)
	if err != nil || p.Title != "标题" || len(p.Parts) != 1 || p.Parts[0] != "内容" || len(p.Images) != 1 || strings.Contains(p.Images[0], "?") {
		t.Fatal(p, err)
	}
	if _, err = publicPostID("https://www.miyoushe.com/ys/article/123"); err == nil {
		t.Fatal("wrong game route accepted")
	}
	if _, err = publicPostID("https://www.miyoushe.com/zzz/article/123?cookie=bad"); err == nil {
		t.Fatal("query accepted")
	}
}
func TestPushPicksOneRecentPostAsYunzai(t *testing.T) {
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	at := func(ago time.Duration) int64 { return now.Add(-ago).Unix() }
	posts := []pushPost{
		{"105", "新版本前瞻", "announce", at(time.Hour)},
		{"101", "旧公告", "announce", at(3 * time.Hour)},
		{"102", "作品展示合集", "announce", at(time.Hour)},
		{"103", "已推送的公告", "announce", at(time.Hour)},
		{"104", "资讯", "info", at(time.Hour)},
	}
	sub := ContentSubscription{Kinds: []string{"announce"}, Sent: map[string]int64{"103": now.Add(-time.Hour).UnixMilli()}}
	if post, ok := postToPush(posts, sub, now); !ok || post.id != "105" {
		t.Fatal(post, ok)
	}
	// Ten hours after a push the post may be pushed again.
	sub.Sent["103"] = now.Add(-10 * time.Hour).UnixMilli()
	if post, _ := postToPush(posts, sub, now); post.id != "103" {
		t.Fatal(post)
	}
}

func TestPublicLiveAnonymousSources(t *testing.T) {
	if os.Getenv("RAYLEA_PUBLIC_SMOKE") != "1" {
		t.Skip("explicit anonymous network smoke")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Second)
	defer cancel()
	client := PublicContentClient{}
	cal, err := client.calendar(ctx)
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
	news, err := client.posts(ctx, ContentQuery{Kind: "news", Type: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("announcements=%d dated=%d news=%d", len(items), known, len(news["items"].([]PublicPost)))
}
