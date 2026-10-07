package images

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/RayleaBot/zzz/internal/app"
)

// trainingArtwork is what proficiency adds.
var trainingArtwork = [][2]string{
	{"proficiency-images-BgFrame", "resources/proficiency/images/BgFrame.png"},
	{"proficiency-images-skill_bg", "resources/proficiency/images/skill_bg.png"},
}

// trainingBase is ZZZ-Plugin's per-rarity weight in the proficiency score.
func trainingBase(rarity string) float64 {
	switch rarity {
	case "S":
		return 5
	case "A":
		return 4
	}
	return 3
}

// discComment is ZZZ-Plugin's grade for the total drive disc score.
func discComment(score float64) string {
	for _, step := range []struct {
		below float64
		grade string
	}{{80, "C"}, {120, "B"}, {160, "A"}, {180, "S"}, {200, "SS"}, {220, "SSS"}, {280, "ACE"}} {
		if score < step.below {
			return step.grade
		}
	}
	return "MAX"
}

// Training draws 练度统计 the way ZZZ-Plugin's proficiency page does, from
// the panels kept for the UID as upstream reads its saved panels: the player
// card, S-rank agents, the S-rank W-Engine rate, high-Mindscape agents and
// SSS-or-better discs, then every agent by ZZZ-Plugin's proficiency score
// with Mindscape, level, attribute, portrait, the six skill levels, the
// W-Engine and the disc grade.
func Training(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	if context.SavedPanels == nil {
		return app.Image{}, false
	}
	panels := context.SavedPanels(result.Role.UID)
	if len(panels) == 0 {
		return app.Image{}, false
	}
	resources := newRecordResources(context, commonArtwork, trainingArtwork)
	maps := readMaps(context)
	fetch := func(id, name string) string {
		resource, ok := context.FetchArtworkResource(id, "zzzerouid", name)
		if !ok {
			return ""
		}
		resources.List = append(resources.List, resource)
		return id
	}
	type row struct {
		score float64
		data  map[string]any
	}
	rows := []row{}
	agentsS, weapons, weaponsS, highRank, discsSSS := 0, 0, 0, 0, 0
	for _, panel := range panels {
		official := panel.Official
		if official == nil {
			continue
		}
		id, rarity := app.Text(official["id"]), app.Text(official["rarity"])
		level, rank := app.Int(official["level"]), app.Int(official["rank"])
		base := trainingBase(rarity)
		discScore := 0.0
		if context.Score != nil {
			if scored, err := context.Score(panel); err == nil && scored.ScoreDetail != nil {
				var detail zzzDetail
				if json.Unmarshal(scored.ScoreDetail.Raw, &detail) == nil {
					for _, piece := range detail.Pieces {
						discScore += piece.Score
						if piece.Grade == "SSS" || piece.Grade == "ACE" || piece.Grade == "MAX" {
							discsSSS++
						}
					}
				}
			}
		}
		score := discScore*2 + float64(level)*2 + float64(rank)*base*2
		skills, _ := official["skills"].([]any)
		for _, raw := range skills {
			skill, _ := raw.(map[string]any)
			score += float64(app.Int(skill["level"])) * base
		}
		levels := []any{}
		for _, index := range []int{0, 2, 5, 1, 3, 4} {
			level := ""
			if index < len(skills) {
				skill, _ := skills[index].(map[string]any)
				level = app.Text(skill["level"])
			}
			levels = append(levels, level)
		}
		data := map[string]any{"rarity": rarity, "rank": rank, "level": level, "level_rank": level / 10,
			"element": maps.elementName(official["element_type"], official["sub_element_type"]), "skills": levels}
		if sprite := maps.partners[id].SpriteID; sprite != "" {
			data["icon"] = fetch("agent-"+id, "role_general/IconRoleGeneral"+sprite+".png")
		}
		// Upstream grades every agent, C when nothing is scored.
		data["comment"] = discComment(discScore)
		if weapon, _ := official["weapon"].(map[string]any); weapon != nil {
			weapons++
			weaponRarity := app.Text(weapon["rarity"])
			if weaponRarity == "S" {
				weaponsS++
			}
			weaponLevel, star := app.Int(weapon["level"]), app.Int(weapon["star"])
			score += float64(weaponLevel)*2 + float64(star)*2*trainingBase(weaponRarity)
			engine := map[string]any{"rarity": weaponRarity, "name": app.Text(weapon["name"]), "star": star, "level": weaponLevel, "level_rank": weaponLevel / 10}
			if code := maps.weapons[app.Text(weapon["id"])].CodeName; code != "" {
				engine["icon"] = fetch("weapon-"+app.Text(weapon["id"]), "weapon/"+code+"_High.png")
			}
			data["weapon"] = engine
		}
		if rarity == "S" {
			agentsS++
		}
		if rank > 4 {
			highRank++
		}
		rows = append(rows, row{score: score, data: data})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].score > rows[j].score })
	items := []any{}
	for _, item := range rows {
		items = append(items, item.data)
	}
	rate := 0.0
	if weapons > 0 {
		rate = float64(weaponsS) / float64(weapons) * 100
	}
	total := len(items)
	return app.Image{Template: "training", Data: map[string]any{
		"player": playerCard(result.Role), "list": items, "bars": make([]int, 8),
		"general": []any{
			map[string]any{"value": fmt.Sprintf("%d/%d", agentsS, total), "label": "S级代理人"},
			map[string]any{"value": fmt.Sprintf("%.1f%%", rate), "label": "S级音擎率"},
			map[string]any{"value": fmt.Sprintf("%d/%d", highRank, total), "label": "高影代理人"},
			map[string]any{"value": fmt.Sprint(discsSSS), "label": "SSS+驱动盘"},
		},
	}, Resources: resources.List}, true
}
