package app

import (
	"context"
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"slices"
)

type OwnedTeam struct {
	Team     PublicTeam `json:"team"`
	Owned    bool       `json:"owned"`
	Weight   float64    `json:"weight"`
	Fallback bool       `json:"fallback"`
}
type TeamPair struct {
	Up     OwnedTeam `json:"up"`
	Down   OwnedTeam `json:"down"`
	Weight float64   `json:"weight"`
}

func pairStatisticsTeams(teams []PublicTeam, weights map[string]float64, fallback map[string]bool) []TeamPair {
	type candidate struct {
		owned OwnedTeam
		half  string
		rate  float64
		used  bool
	}
	candidates := []candidate{}
	for _, team := range teams {
		v := OwnedTeam{Team: team, Owned: true}
		for _, id := range team.IDs {
			score, ok := weights[id]
			if id == "" || !ok {
				v.Owned = false
			}
			v.Weight += score
			v.Fallback = v.Fallback || fallback[id]
		}
		if !v.Owned {
			v.Weight = 1
		}
		for _, half := range []string{"up", "down"} {
			rate := team.Up
			if half == "down" {
				rate = team.Down
			}
			if rate != nil && *rate > 0 {
				copy := v
				copy.Weight *= *rate
				candidates = append(candidates, candidate{copy, half, *rate, false})
			}
		}
	}
	slices.SortStableFunc(candidates, func(a, b candidate) int {
		if a.owned.Weight > b.owned.Weight {
			return -1
		}
		if a.owned.Weight < b.owned.Weight {
			return 1
		}
		return 0
	})
	pairs := []TeamPair{}
	for i := range candidates {
		if candidates[i].used {
			continue
		}
		for j := range candidates {
			if i == j || candidates[j].used || candidates[i].half == candidates[j].half {
				continue
			}
			left, right := candidates[i], candidates[j]
			overlap := false
			for _, id := range left.owned.Team.IDs {
				if id == "" || slices.Contains(right.owned.Team.IDs, id) {
					overlap = true
				}
			}
			if overlap {
				continue
			}
			up, down := left.owned, right.owned
			if left.half == "down" {
				up, down = right.owned, left.owned
			}
			weight := up.Weight + down.Weight
			if !up.Owned || !down.Owned {
				weight = left.rate + right.rate
			}
			pairs = append(pairs, TeamPair{up, down, weight})
			candidates[i].used = true
			candidates[j].used = true
			break
		}
		if len(pairs) >= 20 {
			break
		}
	}
	slices.SortStableFunc(pairs, func(a, b TeamPair) int {
		if a.Weight > b.Weight {
			return -1
		}
		if a.Weight < b.Weight {
			return 1
		}
		return 0
	})
	return pairs[:min(4, len(pairs))]
}

func (a *App) statisticsTeams(ctx context.Context, event *rayleabot.EventContext, input map[string]any) (map[string]any, error) {
	if a.Game.ID != "genshin" {
		return nil, gameError("operation_denied", "此配队来源仅适用于原神。")
	}
	job, err := a.ContentJobs.Poll(asText(input["ref"]), false)
	if err != nil {
		return nil, err
	}
	if job.State != "completed" || job.Result["source"] != "abyss" {
		return nil, gameError("statistics_missing", "请先读取深渊统计。")
	}
	teams, ok := job.Result["teams"].([]PublicTeam)
	if !ok {
		return nil, gameError("statistics_invalid", "统计队伍格式无效。")
	}
	choice := Selection{asText(input["account_ref"]), asText(input["role_ref"])}
	client := a.accountClient(event)
	owned, err := client.Execute(ctx, choice, "genshin.characters", nil)
	if err != nil {
		return nil, err
	}
	roster := asList(owned.Data["list"])
	if roster == nil {
		roster = asList(owned.Data["avatars"])
	}
	available := map[string]bool{}
	for _, v := range roster {
		id := firstText(asObject(v), "id", "avatar_id")
		if id != "" {
			available[id] = true
		}
	}
	ids := []string{}
	for _, team := range teams {
		for _, id := range team.IDs {
			if available[id] && !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
	}
	if len(ids) > 150 {
		return nil, gameError("statistics_limit", "候选角色超过150位，请缩小统计范围。")
	}
	weights := map[string]float64{}
	fallback := map[string]bool{}
	for at := 0; at < len(ids); at += 50 {
		values := []any{}
		for _, id := range ids[at:min(at+50, len(ids))] {
			values = append(values, id)
		}
		details, err := client.Execute(ctx, choice, "genshin.character", map[string]any{"character_ids": values})
		if err != nil {
			return nil, err
		}
		list := asList(details.Data["list"])
		if list == nil {
			list = asList(details.Data["avatars"])
		}
		for _, raw := range list {
			v := asObject(raw)
			id := firstText(v, "id", "avatar_id")
			level, ok := challengeNumber(v["level"])
			if !available[id] || !ok {
				continue
			}
			weapon, hasWeapon := challengeNumber(asObject(v["weapon"])["level"])
			if !hasWeapon {
				weapon = 1
			}
			maxTalent := 1.0
			hasTalent := false
			for _, field := range []string{"skills", "skill_list", "talents"} {
				for _, r := range asList(v[field]) {
					m := asObject(r)
					n, ok := challengeNumber(m["level"])
					if !ok {
						n, ok = challengeNumber(m["level_current"])
					}
					if ok {
						hasTalent = true
						maxTalent = max(maxTalent, n)
					}
				}
			}
			weights[id] = min(level, weapon)*100 + maxTalent*1000
			fallback[id] = !hasWeapon || !hasTalent
		}
	}
	pairs := pairStatisticsTeams(teams, weights, fallback)
	return map[string]any{"pairs": pairs, "uid": owned.Role.UID, "note": "按固定参考的角色/武器等级与最高天赋加权，再以样本使用数量排序；缺失武器或天赋按参考默认1并标记，不是实战伤害预测。", "source": "Yshelper"}, nil
}
