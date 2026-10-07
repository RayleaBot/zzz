package images

import (
	"strings"
	"testing"
	"time"

	"github.com/RayleaBot/zzz/internal/app"
)

func TestQueryRankFollowsZZZPluginOrder(t *testing.T) {
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	context := app.ImageContext{Now: now}
	uids := func(ranked []app.QueryRankRecord) string {
		out := []string{}
		for _, record := range ranked {
			out = append(out, record.UID)
		}
		return strings.Join(out, ",")
	}
	abyss := func(uid string, score int, rating string, end int64, played int64) app.QueryRankRecord {
		return app.QueryRankRecord{UID: uid, Data: map[string]any{"hadal_ver": "v2", "hadal_info_v2": map[string]any{
			"begin_time": now.Unix() - 3600, "end_time": end, "brief": map[string]any{"score": score, "rating": rating},
			"fitfh_layer_detail": map[string]any{"layer_challenge_info_list": []any{map[string]any{"challenge_time": played}}}}}}
	}
	// Score, then rating, then the earlier record; past periods leave.
	records := []app.QueryRankRecord{abyss("a", 140000, "S", now.Unix()+60, 10), abyss("b", 150000, "S+", now.Unix()+60, 10),
		abyss("c", 140000, "S+", now.Unix()+60, 20), abyss("d", 140000, "S+", now.Unix()+60, 5), abyss("e", 150000, "S+", now.Unix()-60, 1)}
	_, ranked, ok := QueryRank(context, app.QueryRankImage{Type: app.QueryRankType{Page: "abyss"}, Records: records})
	if !ok || uids(ranked) != "b,d,c,a" || ranked[0].Summary != "150000 · S+" {
		t.Fatalf("abyss = %s %v", uids(ranked), ranked)
	}
	// Holo Boss ranks fewer seconds first among equal stars.
	holo := func(uid string, minutes int) app.QueryRankRecord {
		period := func(day int) map[string]any { return map[string]any{"year": 2026, "month": 9, "day": day} }
		return app.QueryRankRecord{UID: uid, Data: map[string]any{"unlock": true, "start_time": period(12), "end_time": period(28),
			"list": []any{map[string]any{"star": 3, "challenge_time": map[string]any{"minute": minutes}}}}}
	}
	_, ranked, _ = QueryRank(context, app.QueryRankImage{Type: app.QueryRankType{Page: "holo-boss"}, Records: []app.QueryRankRecord{holo("slow", 5), holo("fast", 2)}})
	if uids(ranked) != "fast,slow" {
		t.Errorf("holo = %s", uids(ranked))
	}
	// Tower S2 breaks equal floors by flawless floors; S1 ignores them.
	tower := func(uid string, floor, flawless int) app.QueryRankRecord {
		season := map[string]any{"climbing_tower_layer": floor, "floor_mvp_num": flawless}
		return app.QueryRankRecord{UID: uid, Data: map[string]any{"climbing_tower_s1": season, "climbing_tower_s2": season}}
	}
	list := []app.QueryRankRecord{tower("a", 50, 10), tower("b", 50, 30), tower("c", 60, 0)}
	if _, ranked, _ = QueryRank(context, app.QueryRankImage{Type: app.QueryRankType{Page: "tower-s2"}, Records: list}); uids(ranked) != "c,b,a" {
		t.Errorf("tower s2 = %s", uids(ranked))
	}
	if _, ranked, _ = QueryRank(context, app.QueryRankImage{Type: app.QueryRankType{Page: "tower-s1"}, Records: list}); uids(ranked) != "c,a,b" {
		t.Errorf("tower s1 = %s", uids(ranked))
	}
	if _, _, ok := QueryRank(context, app.QueryRankImage{Type: app.QueryRankType{Page: "deadly"}, Records: list}); ok {
		t.Error("records without the mode's data do not rank")
	}
}
