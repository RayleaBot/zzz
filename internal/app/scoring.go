package app

import (
	"context"
	"encoding/json"
	"math"
	"slices"
	"strconv"
	"strings"
)

type EquipmentScore struct {
	Value float64 `json:"value"`
	Grade string  `json:"grade"`
}

// ScoreDetail is what miao's getMarkDetail gives its panel image: the total
// mark and grade, attribute weights, every piece's formatted main stat and
// substats, and the substats summed over all pieces. ZZZ scoring leaves it
// empty.
type ScoreDetail struct {
	Title    string              `json:"title"`
	Mark     string              `json:"mark"`
	Grade    string              `json:"grade"`
	Weights  map[string]float64  `json:"weights"`
	Titles   map[string]string   `json:"titles"`
	AllAttrs []ScoredAttr        `json:"all_attrs"`
	Pieces   map[int]ScoredPiece `json:"-"`
	// Raw is the whole scoring result, for game-specific fields.
	Raw json.RawMessage `json:"-"`
}

// ScoredPiece is one piece of equipment as upstream's panel shows it.
type ScoredPiece struct {
	Mark  string       `json:"mark"`
	Grade string       `json:"grade"`
	Main  ScoredAttr   `json:"main"`
	Attrs []ScoredAttr `json:"attrs"`
}

// ScoredAttr is a formatted stat: its display value, its mark under the
// character's weights, rolls, and efficiency in maximum rolls ("-" when none).
type ScoredAttr struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Mark  string `json:"mark,omitempty"`
	UpNum int    `json:"upNum"`
	Eff   string `json:"eff"`
}

const scoreNote = "装备评分用于比较角色专属权重下的词条，不代表实际伤害或队伍收益。"

func decimalStat(value string) (float64, bool) {
	value = strings.TrimSpace(strings.TrimSuffix(value, "%"))
	value = strings.ReplaceAll(value, ",", "")
	n, err := strconv.ParseFloat(value, 64)
	return n, err == nil && !math.IsNaN(n) && !math.IsInf(n, 0) && n >= 0
}
func normalizedElement(value string) string {
	value = strings.ToLower(value)
	for key, names := range map[string][]string{"pyro": {"pyro", "火"}, "electro": {"electro", "雷"}, "hydro": {"hydro", "水"}, "dendro": {"dendro", "草"}, "anemo": {"anemo", "风"}, "geo": {"geo", "岩"}, "cryo": {"cryo", "冰"}, "physical": {"physical", "物理"}, "fire": {"fire"}, "ice": {"ice"}, "lightning": {"lightning"}, "wind": {"wind"}, "quantum": {"quantum", "量子"}, "imaginary": {"imaginary", "虚数"}} {
		if slices.Contains(names, value) {
			return key
		}
	}
	return value
}

// scorePanel rates each complete piece of equipment with the character's
// upstream rule, run by the calculation engine: ZZZ-Plugin's Score.
func (a *App) scorePanel(ctx context.Context, panel CharacterPanel) (CharacterPanel, error) {
	engine := a.Game.Calc
	if engine == nil {
		return panel, gameError("score_unavailable", "当前资料没有评分规则。")
	}
	record, err := findReferenceCharacter(engine, panel)
	if err != nil {
		return panel, gameError("score_unavailable", "缺少角色基准资料，暂不计算评分。")
	}
	var input map[string]any
	input, err = zzzScoreInput(panel)

	if err != nil {
		return panel, err
	}
	raw, err := engine.Score(ctx, record, input)
	if err != nil {
		return panel, gameError("score_unavailable", "评分计算失败，暂不输出评分。")
	}
	var result struct {
		ScoreDetail
		Pieces []struct {
			ScoredPiece
			Slot  int     `json:"slot"`
			Score float64 `json:"score"`
		} `json:"pieces"`
		Total float64 `json:"total"`
	}
	if json.Unmarshal(raw, &result) != nil || len(result.Pieces) == 0 {
		return panel, gameError("score_unavailable", "评分计算结果无效。")
	}
	scores := map[int]EquipmentScore{}
	detail := result.ScoreDetail
	detail.Raw = raw
	detail.Pieces = map[int]ScoredPiece{}
	for _, piece := range result.Pieces {
		scores[piece.Slot] = EquipmentScore{Value: piece.Score, Grade: piece.Grade}
		detail.Pieces[piece.Slot] = piece.ScoredPiece
	}
	panel.Equipment = append([]PanelEquipment{}, panel.Equipment...)
	scored := 0
	for i := range panel.Equipment {
		panel.Equipment[i].Score = nil
		if score, ok := scores[panel.Equipment[i].Slot]; ok {
			panel.Equipment[i].Score = &score
			delete(scores, panel.Equipment[i].Slot)
			scored++
		}
	}
	panel.TotalScore = &result.Total
	panel.ScoredEquipment = scored
	panel.ScoreRule = result.Title
	panel.ScoreDetail = &detail
	panel.ScoreNote = scoreNote
	if scored != len(panel.Equipment) {
		panel.ScoreNote += "部分装备缺少必要字段，未计算分数。"
	}
	return panel, nil
}

// zzzScoreInput passes the official property list, Mindscape level and the
// complete drive discs that ZZZ-Plugin Score reads.
func zzzScoreInput(panel CharacterPanel) (map[string]any, error) {
	if !panel.RankKnown || panel.Rank < 0 || panel.Rank > 6 {
		return nil, gameError("score_unavailable", "面板缺少影画信息，暂不计算评分。")
	}
	properties := []map[string]any{}
	names := map[string]bool{}
	for _, stat := range panel.Stats {
		id, err := strconv.Atoi(stat.ID)
		if err != nil || stat.Name == "" || stat.Value == "" {
			continue
		}
		properties = append(properties, map[string]any{"property_id": id, "property_name": stat.Name, "base": strings.ReplaceAll(stat.Base, ",", ""), "final": strings.ReplaceAll(stat.Value, ",", "")})
		names[stat.Name] = true
	}
	for _, name := range []string{"生命值", "攻击力", "防御力", "暴击率", "暴击伤害", "异常掌控", "异常精通"} {
		if !names[name] {
			return nil, gameError("score_unavailable", "面板缺少评分所需的属性，暂不计算评分。")
		}
	}
	rarities := map[string]string{"S": "S", "A": "A", "B": "B", "5": "S", "4": "S", "3": "A", "2": "B"}
	discs := []map[string]any{}
	seen := map[int]bool{}
	for _, piece := range panel.Equipment {
		rarity := rarities[piece.Rarity]
		if !piece.Complete || len(piece.Main) != 1 || piece.Slot < 1 || piece.Slot > 6 || seen[piece.Slot] || piece.Level < 0 || piece.Level > 15 || rarity == "" {
			continue
		}
		if _, err := strconv.Atoi(piece.Main[0].ID); err != nil {
			continue
		}
		sub := []map[string]string{}
		valid := true
		for _, stat := range piece.Sub {
			if _, err := strconv.Atoi(stat.ID); err != nil {
				valid = false
				break
			}
			if _, ok := decimalStat(stat.Value); !ok {
				valid = false
				break
			}
			sub = append(sub, map[string]string{"id": stat.ID, "value": strings.TrimSpace(stat.Value)})
		}
		if valid {
			seen[piece.Slot] = true
			discs = append(discs, map[string]any{"slot": piece.Slot, "level": piece.Level, "rarity": rarity, "main": piece.Main[0].ID, "sub": sub})
		}
	}
	if len(discs) == 0 {
		return nil, gameError("score_unavailable", "缺少完整词条，暂不输出评分。")
	}
	return map[string]any{"rank": panel.Rank, "properties": properties, "equipment": discs}, nil
}
