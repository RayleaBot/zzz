package app

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestChallengeScoresSeparatePeriodsAndMissingMetrics(t *testing.T) {
	fixtures := []struct {
		game, kind, raw, key string
		value                float64
		period               int
	}{
		{"genshin", "abyss", `{"schedule_id":12,"max_floor":"12-3","floors":[{"index":11,"star":9},{"index":12,"star":7}],"total_battle_times":20}`, "star", 7, 1},
		{"genshin", "theater", `{"data":[{"schedule":{"schedule_id":11},"stat":{"difficulty_id":4,"get_medal_round_list":[1,0,1]},"detail":{"rounds_data":[{},{}]}}]}`, "medal", 2, 1},
		{"genshin", "hard_single", `{"data":[{"schedule":{"schedule_id":11},"single":{"best":{"difficulty":6,"second":132}}}]}`, "time", 132, 1},
		{"genshin", "hard_mp", `{"data":[{"schedule":{"schedule_id":11},"mp":{"difficulty":5,"second":100}}]}`, "difficulty", 5, 1},
		{"starrail", "challenge", `{"schedule_id":12,"max_floor":12,"all_floor_detail":[{"star_num":3,"round_num":4}]}`, "round", 4, 1},
		{"starrail", "story", `{"schedule_id":12,"max_floor":4,"all_floor_detail":[{"star_num":3,"score":80000}]}`, "score", 80000, 1},
		{"starrail", "boss", `{"schedule_id":12,"max_floor":4,"all_floor_detail":[{"star_num":3,"score":7600}]}`, "score", 7600, 1},
		{"starrail", "peak", `{"challenge_peak_records":[{"group":{"group_id":13},"boss_stars":3},{"group":{"group_id":12},"boss_stars":2,"mob_stars":9,"boss_record":{"round_num":7,"hard_mode":true}}]}`, "round", 7, 2},
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
			kind, _ := challengeKind(f.game, f.kind)
			r, e := challengeExtract(f.game, kind, data, f.period)
			if e != nil || r.Metrics[f.key] != f.value {
				t.Fatalf("%+v %v", r, e)
			}
			if f.kind == "peak" && r.Season != "12" {
				t.Fatal("previous period mixed", r)
			}
			if _, e = challengeExtract(f.game, kind, map[string]any{}, 1); e == nil {
				t.Fatal("missing data accepted")
			}
		})
	}
	kind, _ := challengeKind("starrail", "challenge")
	a := ChallengeEntry{Metrics: map[string]float64{"floor": 12, "star": 3, "round": 4}}
	b := ChallengeEntry{Metrics: map[string]float64{"floor": 12, "star": 3}}
	if challengeCompare(kind, "", a, b) >= 0 {
		t.Fatal("missing round ranked as zero")
	}
	b.Metrics["round"] = 3
	if challengeCompare(kind, "", a, b) <= 0 {
		t.Fatal("lower round must win")
	}
}
func TestChallengeGroupPaginationRestartWithdrawalAndRevision(t *testing.T) {
	dir := t.TempDir()
	s := &GroupStore{Directory: filepath.Join(dir, "groups")}
	scope := GroupScope{"onebot11", "a", "bot", "group"}
	now := time.Now().UnixMilli()
	for i := 0; i < 23; i++ {
		e := ChallengeEntry{Kind: "tower_s4", Season: "tower_s4", Region: "cn_gf01", ActorID: fmt.Sprint(i), UID: fmt.Sprint(10000000 + i), Metrics: map[string]float64{"score": float64(i)}, FirstMS: now, UpdatedMS: now}
		if err := s.SubmitChallenge(scope, e); err != nil {
			t.Fatal(err)
		}
	}
	a := App{Game: Game{ID: "zzz"}, Groups: &GroupStore{Directory: s.Directory}}
	kind, _ := challengeKind("zzz", "tower_s4")
	out, err := a.challengeList(scope, kind, "", "", 20)
	if err != nil {
		t.Fatal(err)
	}
	group := out["groups"].([]map[string]any)[0]
	if len(group["items"].([]ChallengeEntry)) != 3 || group["items"].([]ChallengeEntry)[0].ActorID != "2" {
		t.Fatal(group)
	}
	other := scope
	other.BotID = "other"
	empty, _ := a.challengeList(other, kind, "", "", 0)
	if len(empty["groups"].([]map[string]any)) != 0 {
		t.Fatal("cross bot leak")
	}
	_, err = a.manageChallenge("challenge.clear", map[string]any{"scope": scope, "confirmed": true, "revision": 0})
	if err == nil {
		t.Fatal("stale destructive clear")
	}
	_, err = a.manageChallenge("challenge.clear", map[string]any{"scope": scope, "confirmed": true, "revision": out["revision"], "kind": "tower_s4"})
	if err != nil {
		t.Fatal(err)
	}
	fresh, _ := s.Read(scope)
	if len(fresh.Challenges) != 0 {
		t.Fatal(fresh)
	}
}
