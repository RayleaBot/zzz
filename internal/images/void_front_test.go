package images

import (
	"encoding/json"
	"testing"

	"github.com/RayleaBot/zzz/internal/app"
)

func TestVoidFrontFollowsZZZPlugin(t *testing.T) {
	stage := func(name, score, maximum string) map[string]any {
		return map[string]any{"name": name, "score": json.Number(score), "max_score": json.Number(maximum), "score_ratio": "150%", "star": "S",
			"challenge_time": zzzTime(12), "buffer": map[string]any{"name": "增益"},
			"sub_challenge_record": []any{map[string]any{"name": name + "-1", "star": "A", "buffer": map[string]any{"name": "子增益"}}}}
	}
	data := map[string]any{"void_front_battle_detail": map[string]any{
		"void_front_battle_abstract_info_brief": map[string]any{"start_time": zzzTime(1), "total_score": json.Number("481000"), "max_score": json.Number("481000"),
			"rank_percent": json.Number("50"), "ending_record_name": "终局"},
		"boss_challenge_record":      map[string]any{"boss_info": map[string]any{"name": "首领"}, "main_challenge_record": stage("最终", "90000", "100000")},
		"main_challenge_record_list": []any{stage("STAGE 03", "65000", "65000"), stage("STAGE 02", "60000", "65000")},
	}}
	context := app.ImageContext{Game: app.Game{Prefix: "%"}}
	image, ok := VoidFront(context, app.QueryResult{Data: data})
	if !ok || image.Data["end"] != "超过42天" || image.Data["rank_note"].(map[string]any)["state"] != "" {
		t.Fatalf("image = %v", image.Data)
	}
	total := image.Data["total"].(map[string]any)
	if total["score"].(map[string]any)["full"] != true || total["rank_bg"] != 1 || total["ending"] != "终局" {
		t.Errorf("total = %v", total)
	}
	boss := image.Data["boss"].(map[string]any)
	if boss["name"] != "首领" || boss["ratio"] != "150%" || boss["score"].(map[string]any)["full"] != false || len(boss["subs"].([]any)) != 1 {
		t.Errorf("boss = %v", boss)
	}
	stages := image.Data["stages"].([]any)
	if len(stages) != 2 || stages[0].(map[string]any)["score"].(map[string]any)["full"] != true || stages[1].(map[string]any)["time"] != "2026-09-12 20:05:09" {
		t.Errorf("stages = %v", stages)
	}
	// Upstream notes the ranking only for the current period.
	context.Input = map[string]any{"schedule_type": 2}
	if image, _ := VoidFront(context, app.QueryResult{Data: data}); image.Data["rank_note"].(map[string]any) != nil {
		t.Errorf("last period = %v", image.Data["rank_note"])
	}
	if _, ok := VoidFront(context, app.QueryResult{Data: map[string]any{}}); ok {
		t.Error("a period without data answers in text like upstream")
	}
}
