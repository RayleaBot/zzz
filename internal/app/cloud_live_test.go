//go:build manual_smoke

package app

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestLiveAnonymousCloudUsage(t *testing.T) {
	if os.Getenv("RAYLEA_CLOUD_READONLY_SMOKE") != "1" {
		t.Skip("explicit anonymous check required")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	client := CloudClient{}
	route, body, _ := cloudRequest("genshin", CloudInput{Mode: "usage", Consent: true})
	view, err := client.fetch(ctx, Game{ID: "genshin"}, CloudInput{Mode: "usage", Consent: true}, route, body)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Rows) == 0 {
		t.Fatal("empty usage projection")
	}
	t.Logf("anonymous quota fields accepted: %d", len(view.Rows))
	input := CloudInput{Mode: "distribution", CharacterID: "10000046", Consent: true}
	route, body, _ = cloudRequest("genshin", input)
	view, err = client.fetch(ctx, Game{ID: "genshin"}, input, route, body)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Sections) == 0 {
		t.Fatal("missing distribution sections")
	}
	t.Logf("public character distribution accepted: sections=%d", len(view.Sections))
}

func TestLiveAnonymousCloudCustomRank(t *testing.T) {
	if os.Getenv("RAYLEA_CLOUD_CUSTOM_SMOKE") != "1" {
		t.Skip("explicit anonymous check required")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Second)
	defer cancel()
	c := CloudClient{}
	defer c.Close()
	input := CloudInput{Mode: "custom", CharacterID: "10000046", Consent: true, Limit: 2, Filters: []CloudFilter{{Op: "=", Value: 0}}}
	route, body, err := cloudRequest("genshin", input)
	if err != nil {
		t.Fatal(err)
	}
	r, err := c.fetchResult(ctx, Game{ID: "genshin"}, input, route, body)
	if err != nil {
		t.Fatal(err)
	}
	if r.Ranking == nil {
		t.Fatal("missing ranking")
	}
	t.Logf("anonymous custom query accepted: rows=%d", len(r.Ranking.Rows))
	if len(r.Ranking.Rows) > 0 {
		t.Logf("panel reference available=%t; public UID available=%t", r.queryID != "", r.Ranking.Rows[0].UID != "")
	}
	// No local account, QQ or UID is used. Do not log returned public identities.
	if len(r.Ranking.Rows) > 0 && r.Ranking.Rows[0].CanReadPanel {
		input.Mode = "rank_panel"
		input.UID = r.Ranking.Rows[0].UID
		panel, err := c.fetchResult(ctx, Game{ID: "genshin"}, input, "rank/custom/specific", map[string]any{"version": "0.1.0", "query_id": r.queryID, "index": 0})
		if err != nil {

			t.Fatal(err)
		}
		t.Logf("public ranking panel accepted: characters=%d", len(panel.Panels))
	}
}
