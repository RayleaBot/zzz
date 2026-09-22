package app

import (
	"encoding/json"
	"testing"
)

func TestChallengeScoresSeparatePeriodsAndMissingMetrics(t *testing.T) {
	fixtures := []struct {
		game, kind, raw, key string
		value                float64
		period               int
	}{
		{"zzz", "challenge", `{"hadal_info_v2":{"begin_time":1760000000,"brief":{"score":12345,"rating":"S+"}}}`, "rating", 5, 1},
		{"zzz", "deadly", `{"start_time":{"year":2026,"month":9,"day":1},"total_star":9,"total_score":123456}`, "score", 123456, 1},
		{"zzz", "deadly_hard", `{"start_time":1760000000,"has_hard":true,"hard_list":[{"score":100,"star":2},{"score":200,"star":3}]}`, "score", 300, 1},
		{"zzz", "holo_boss", `{"start_time":1760000000,"list":[{"star":3,"challenge_time":{"minute":1,"second":20},"boss":{"medal":{"is_no_injured":true}}},{"star":2,"challenge_time":{"minute":0,"second":45}}]}`, "time", 125, 1},
		{"zzz", "void_front", `{"void_front_battle_detail":{"void_front_battle_abstract_info_brief":{"end_ts":1760000000,"total_score":10000}}}`, "score", 10000, 1},
		{"zzz", "tower_s1", `{"climbing_tower_s1":{"climbing_tower_layer":200}}`, "floor", 200, 1},
		{"zzz", "tower_s2", `{"climbing_tower_s2":{"climbing_tower_layer":200,"floor_mvp_num":150}}`, "flawless", 150, 1},
		{"zzz", "tower_s3", `{"climbing_tower_s3":{"layer_info":{"total_score":60000,"climbing_tower_layer":100},"mvp_info":{"floor_mvp_num":90}}}`, "score", 60000, 1},
		{"zzz", "tower_s4", `{"climbing_tower_s4":{"layer_info":{"total_score":60000,"climbing_tower_layer":100},"mvp_info":{"floor_mvp_num":90}}}`, "flawless", 90, 1},
	}
	for _, f := range fixtures {
		t.Run(f.game+"."+f.kind, func(t *testing.T) {
			var data map[string]any
			json.Unmarshal([]byte(f.raw), &data)
			kind, _ := challengeKind(f.kind)
			r, e := challengeExtract(kind, data)
			if e != nil || r.Metrics[f.key] != f.value {
				t.Fatalf("%+v %v", r, e)
			}
			if f.kind == "peak" && r.Season != "12" {
				t.Fatal("previous period mixed", r)
			}
			if _, e = challengeExtract(kind, map[string]any{}); e == nil {
				t.Fatal("missing data accepted")
			}
		})
	}
}
