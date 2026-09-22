package app

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

func TestShowcaseKeepsAnswersForTheirTTLAndBacksOffWhenLimited(t *testing.T) {
	count, status, body := 0, http.StatusOK, `{"ttl":300,"n":1}`
	client := ShowcaseClient{HTTP: doerFunc(func(req *http.Request) (*http.Response, error) {
		count++
		if req.URL.String() != "https://showcase.test/"+strings.TrimPrefix(req.URL.Path, "/") || req.Header.Get("Cookie") != "" || req.Header.Get("User-Agent") == "" {
			t.Fatal("invalid public request")
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	source := ShowcaseSource{Name: "fixture", URL: func(uid string) string { return "https://showcase.test/" + uid },
		Parse: func(_ context.Context, _ Game, _ Catalog, raw []byte) (ShowcaseProfile, error) {
			if strings.Contains(string(raw), `"empty"`) {
				return ShowcaseProfile{}, ErrShowcaseEmpty
			}
			return ShowcaseProfile{Nickname: string(raw), Panels: []CharacterPanel{{ID: "1"}}, TTL: 5 * time.Minute}, nil
		}}
	first, err := client.Fetch(t.Context(), source, Game{}, Catalog{}, "100000001")
	if err != nil || len(first.Panels) != 1 {
		t.Fatal(first, err)
	}
	if _, err = client.Fetch(t.Context(), source, Game{}, Catalog{}, "100000001"); err != nil || count != 1 {
		t.Fatalf("answer within its TTL was requested again: %d calls, %v", count, err)
	}
	if _, err = client.Fetch(t.Context(), source, Game{}, Catalog{}, "100000001/?url=other"); err == nil || count != 1 {
		t.Fatal("invalid UID reached the network")
	}
	// An empty showcase is not kept, so the player can show characters and ask
	// again.
	body = `{"empty":true}`
	for range 2 {
		if _, err = client.Fetch(t.Context(), source, Game{}, Catalog{}, "100000002"); !errors.Is(err, ErrShowcaseEmpty) {
			t.Fatal(err)
		}
	}
	if count != 3 {
		t.Fatalf("empty answers = %d calls", count-1)
	}
	status = http.StatusTooManyRequests
	var failure *ShowcaseFailure
	if _, err = client.Fetch(t.Context(), source, Game{}, Catalog{}, "100000003"); !errors.As(err, &failure) || failure.Status != 429 {
		t.Fatal(err)
	}
	if _, err = client.Fetch(t.Context(), source, Game{}, Catalog{}, "100000004"); !errors.As(err, &failure) || count != 4 {
		t.Fatal("a limited service was asked again within the minute")
	}
}

func TestPanelStoreMergesRefreshesAndListsLikeUpstream(t *testing.T) {
	catalog, err := ParseCatalog([]byte(`{"version":"1","entries":[{"id":"10000046","name":"胡桃","kind":"character","rarity":5},{"id":"10000023","name":"香菱","kind":"character","rarity":4},{"id":"10000032","name":"班尼特","kind":"character","rarity":4}]}`))
	if err != nil {
		t.Fatal(err)
	}
	store := &PanelStore{Directory: t.TempDir()}
	if _, err = store.Keep("100000001", []CharacterPanel{{ID: "10000046", Level: 90}, {ID: "10000023", Level: 80}}, "Enka", &ShowcaseProfile{Nickname: "旅行者", Level: 60}); err != nil {
		t.Fatal(err)
	}
	// A later refresh replaces its characters and keeps the others.
	saved, err := store.Keep("100000001", []CharacterPanel{{ID: "10000023", Level: 90, Official: map[string]any{"id": 10000023}}, {ID: "10000032", Level: 70}}, "米游社", nil)
	if err != nil || len(saved.Panels) != 3 || saved.Panels["10000023"].Panel.Level != 90 || saved.Service != "Enka" || saved.Nickname != "旅行者" {
		t.Fatal(saved, err)
	}
	saved, _ = store.Read("100000001")
	if saved.Panels["10000023"].panel().Official == nil {
		t.Fatal("official entry was not kept")
	}
	order := []string{}
	for _, item := range saved.Sorted(catalog, map[string]bool{"10000032": true}) {
		order = append(order, item.Panel.ID)
	}
	if strings.Join(order, ",") != "10000032,10000046,10000023" {
		t.Fatalf("order = %v", order)
	}
	if _, err = store.Read("../x"); err == nil {
		t.Fatal("invalid UID accepted")
	}
	if err = store.Delete("100000001"); err != nil {
		t.Fatal(err)
	}
	if saved, _ = store.Read("100000001"); len(saved.Panels) != 0 {
		t.Fatal("deleted panels were kept")
	}
}

func TestRankPlacesTakeTheBetterOfDamageAndScore(t *testing.T) {
	entry := func(uid, id string, score, damage float64) RankEntry {
		e := RankEntry{UID: uid, CharacterID: id, Score: score, Panel: &CharacterPanel{ID: id}}
		if damage > 0 {
			e.Damage = &RankDamage{Value: damage}
		}
		return e
	}
	places := rankPlaces([]RankEntry{entry("1", "a", 100, 30000), entry("2", "a", 200, 20000), entry("3", "a", 300, 10000), entry("1", "b", 50, 0), entry("2", "b", 60, 0)}, "1")
	if places["a"] != (RankPlace{Rank: 1, Mode: "dmg"}) || places["b"] != (RankPlace{Rank: 2, Mode: "mark"}) {
		t.Fatal(places)
	}
}

func TestCatalogRejectsDuplicateFirstEntryAndKeepsAmbiguity(t *testing.T) {
	if _, err := ParseCatalog([]byte(`{"version":"fixture","entries":[{"id":"1","name":"a"},{"id":"1","name":"b"}]}`)); err == nil {
		t.Fatal("duplicate first index accepted")
	}
	catalog, err := ParseCatalog([]byte(`{"version":"fixture","entries":[{"id":"1","name":"test-a","kind":"character","aliases":["test"]},{"id":"2","name":"test-b","kind":"character","aliases":["test"]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Search("test", "character", 10, nil)) != 2 {
		t.Fatal("ambiguous alias silently selected a character")
	}
}
