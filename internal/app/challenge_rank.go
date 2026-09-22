package app

import (
	"strconv"
	"strings"
)

type ChallengeMetric struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Lower bool   `json:"lower"`
}
type ChallengeKind struct {
	ID        string            `json:"id"`
	Label     string            `json:"label"`
	Operation string            `json:"operation"`
	Metrics   []ChallengeMetric `json:"metrics"`
	Required  []string          `json:"required"`
}
type ChallengeEntry struct {
	Kind      string             `json:"kind"`
	Season    string             `json:"season"`
	Region    string             `json:"region"`
	ActorID   string             `json:"actor_id"`
	Nickname  string             `json:"nickname"`
	UID       string             `json:"uid"`
	Metrics   map[string]float64 `json:"metrics"`
	FirstMS   int64              `json:"first_ms"`
	UpdatedMS int64              `json:"updated_ms"`
}

func challengeKinds() []ChallengeKind {
	defs := [][4]string{{"challenge", "防卫战", "challenge", "score rating"}, {"deadly", "危局", "deadly", "star score"}, {"deadly_hard", "危局绝境", "deadly", "score star"}, {"holo_boss", "拟境", "holo_boss", "star time flawless"}, {"void_front", "临界", "void_front", "score"}, {"tower_s1", "爬塔S1", "tower", "floor"}, {"tower_s2", "爬塔S2", "tower", "floor flawless"}, {"tower_s3", "爬塔S3", "tower", "score floor flawless"}, {"tower_s4", "爬塔S4", "tower", "score floor flawless"}}
	labels := map[string]string{"floor": "最深层", "star": "星数", "time": "用时（秒）", "score": "得分", "rating": "评级（C=1 至 S+=5）", "flawless": "无伤/最佳次数"}
	out := []ChallengeKind{}
	for _, d := range defs {
		v := ChallengeKind{ID: d[0], Label: d[1], Operation: "zzz." + d[2]}
		for _, key := range strings.Fields(d[3]) {
			v.Metrics = append(v.Metrics, ChallengeMetric{key, labels[key], key == "time"})
		}
		v.Required = []string{v.Metrics[0].Key}
		out = append(out, v)
	}
	return out
}
func challengeKind(id string) (ChallengeKind, bool) {
	for _, k := range challengeKinds() {
		if strings.EqualFold(id, k.ID) || strings.EqualFold(id, k.Label) {
			return k, true
		}
	}
	return ChallengeKind{}, false
}
func fieldAt(data any, path string) any {
	value := data
	for _, key := range strings.Split(path, ".") {
		switch v := value.(type) {
		case map[string]any:
			value = v[key]
		case []any:
			i, err := strconv.Atoi(key)
			if err != nil || i < 0 || i >= len(v) {
				return nil
			}
			value = v[i]
		default:
			return nil
		}
	}
	return value
}
func challengeNumber(v any) (float64, bool) {
	if v == nil {
		return 0, false
	}
	n, e := strconv.ParseFloat(asText(v), 64)
	return n, e == nil && finiteRange(n, 0, 1e14)
}
func challengeSeason(data map[string]any) string {
	for _, p := range []string{"schedule_id", "schedule.schedule_id", "group.group_id", "groups.0.group_id", "void_front_id"} {
		if id := asText(fieldAt(data, p)); id != "" && len(id) <= 128 {
			return id
		}
	}
	for _, p := range []string{"start_time", "begin_time", "begin_ts", "schedule.start_time", "schedule.start_date_time", "group.begin_time", "groups.0.begin_time"} {
		v := fieldAt(data, p)
		if date := calendarTime(v); date != "" {
			return strings.ReplaceAll(date, " ", "T")
		}
	}
	return ""
}
func challengeExtract(kind ChallengeKind, data map[string]any) (ChallengeEntry, error) {
	out := ChallengeEntry{Kind: kind.ID, Metrics: map[string]float64{}}
	put := func(key string, paths ...string) {
		for _, p := range paths {
			if v, ok := challengeNumber(fieldAt(data, p)); ok {
				out.Metrics[key] = v
				return
			}
		}
	}
	sum := func(key, path, field string) {
		list, ok := fieldAt(data, path).([]any)
		if !ok || len(list) == 0 {
			return
		}
		total := 0.0
		for _, item := range list {
			v, valid := challengeNumber(fieldAt(item, field))
			if !valid {
				return
			}
			total += v
		}
		out.Metrics[key] = total
	}
	out.Season = challengeSeason(data)
	switch kind.ID {
	case "challenge":
		data = asObject(data["hadal_info_v2"])
		out.Season = challengeSeason(data)
		put("score", "brief.score")
		if v, ok := map[string]float64{"C": 1, "B": 2, "A": 3, "S": 4, "S+": 5}[asText(fieldAt(data, "brief.rating"))]; ok {
			out.Metrics["rating"] = v
		}
	case "deadly":
		put("star", "total_star")
		put("score", "total_score")
	case "deadly_hard":
		if data["has_hard"] != true {
			return out, gameError("challenge_unavailable", "此期没有绝境成绩。")
		}
		sum("score", "hard_list", "score")
		sum("star", "hard_list", "star")
	case "holo_boss":
		sum("star", "list", "star")
		valid := true
		total, flawless := 0.0, 0.0
		for _, item := range asList(data["list"]) {
			m, ok := challengeNumber(fieldAt(item, "challenge_time.minute"))
			sec, ok2 := challengeNumber(fieldAt(item, "challenge_time.second"))
			if !ok || !ok2 {
				valid = false
			}
			total += m*60 + sec
			if fieldAt(item, "boss.medal.is_no_injured") == true {
				flawless++
			}
		}
		if valid {
			out.Metrics["time"] = total
		}
		out.Metrics["flawless"] = flawless
	case "void_front":
		data = asObject(fieldAt(data, "void_front_battle_detail.void_front_battle_abstract_info_brief"))
		out.Season = challengeSeason(data)
		if out.Season == "" {
			if v := asText(data["end_ts"]); v != "" {
				out.Season = "end:" + v
			}
		}
		put("score", "total_score")
	default:
		if strings.HasPrefix(kind.ID, "tower_s") {
			out.Season = kind.ID
			data = asObject(data["climbing_"+kind.ID])
			put("floor", "climbing_tower_layer", "layer_info.climbing_tower_layer")
			put("score", "layer_info.total_score")
			put("flawless", "floor_mvp_num", "mvp_info.floor_mvp_num")
		}
	}
	if out.Season == "" {
		return out, gameError("challenge_unavailable", "官方未提供可区分的期次，未加入群榜。")
	}
	for _, key := range kind.Required {
		if _, ok := out.Metrics[key]; !ok {
			return out, gameError("challenge_unavailable", "官方未提供此玩法的必要成绩，未加入群榜。")
		}
	}
	return out, nil
}
