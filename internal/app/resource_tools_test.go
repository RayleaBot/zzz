package app

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnemyFormulaFactorsLevelAndUnknown(t *testing.T) {
	a := App{Game: testGame(t, "genshin")}
	plain, err := a.enemyAction("enemies.query", map[string]any{"name": "无相之火", "stat": "HP", "level": 1})
	if err != nil || math.Abs(plain["value"].(float64)-72.916996656*7) > 1e-8 {
		t.Fatal(plain, err)
	}
	multi, err := a.enemyAction("enemies.query", map[string]any{"name": "无相之火", "stat": "HP", "level": 1, "modifiers": []string{"SpecialHP:0"}})
	if err != nil || math.Abs(multi["value"].(float64)-plain["value"].(float64)*1.5) > 1e-8 {
		t.Fatal(multi, err)
	}
	preset, err := a.enemyAction("enemies.query", map[string]any{"name": "无相之火", "stat": "HP", "level": 0, "modifiers": []string{"DefaultHP:0"}})
	if err != nil || preset["level"] != 88 {
		t.Fatal(preset, err)
	}
	for _, q := range []map[string]any{{"name": "无相之火", "stat": "HP", "modifiers": []string{"SpecialHP:0", "SpecialHP:1"}}, {"name": "未收录原魔", "stat": "HP"}, {"name": "无相之火", "stat": "ATK", "modifiers": []string{"SpecialHP:0"}}} {
		if _, err = a.enemyAction("enemies.query", q); err == nil {
			t.Fatal("invalid factors accepted")
		}
	}
}
func TestGuidesFixedCollectionFullOriginalsAndVersionedDefault(t *testing.T) {
	s := GuideSettings{Path: filepath.Join(t.TempDir(), "guides.json")}
	if _, err := s.Manage("genshin", "guides.configure", map[string]any{"default_source": "2", "revision": 0}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Manage("genshin", "guides.configure", map[string]any{"default_source": "3", "revision": 0}); err == nil {
		t.Fatal("stale settings overwritten")
	}
	client := PublicContentClient{HTTP: cloudDoer(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "bbs-api.mihoyo.com" || r.URL.Query().Get("collection_id") != "813033" || r.Header.Get("Cookie") != "" {
			t.Fatal("wrong destination")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"retcode":0,"data":{"posts":[{"post":{"post_id":"1","game_id":2,"subject":"胡桃攻略","content":"<p>合成正文</p>","images":["https://upload-bbs.miyoushe.com/a.png"]}},{"post":{"post_id":"2","game_id":2,"subject":"其他"}}]}}`))}, nil
	})}
	out, err := client.guides(context.Background(), "genshin", ContentQuery{Source: "2", Query: "胡桃"})
	if err != nil {
		t.Fatal(err)
	}
	posts := out["items"].([]PublicPost)
	if len(posts) != 1 || len(posts[0].Parts) != 1 || len(posts[0].Images) != 1 || out["next_offset"] != -1 {
		t.Fatal(out)
	}
	a := App{Game: testGame(t, "genshin")}
	m, err := a.mapQuery(map[string]any{"query": "琉璃百合&cookie=none", "map_id": 7})
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(m)
	if strings.Contains(string(encoded), "&cookie=") {
		t.Fatal("map query injection")
	}
}
