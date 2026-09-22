package app

import (
	"slices"
	"strings"
	"time"
)

func calendarView(game Game, result map[string]any) View {
	v := View{Title: game.Name + "固定资料", Rows: []Row{}, Note: "来源：" + asText(result["version"]) + "；固定快照，不代表已在线校准的当前活动。完整条目可在材料与卡池页面查询。"}
	for _, p := range result["pools"].([]PoolInfo) {
		if len(v.Sections) >= 10 {
			break
		}
		v.Sections = append(v.Sections, Section{Title: p.Version + " · " + p.Half, Rows: []Row{{Label: "国服时间", Value: p.From + " 至 " + p.To}, {Label: "五星角色", Value: strings.Join(p.Characters5, "、")}, {Label: "四星角色", Value: strings.Join(p.Characters4, "、")}, {Label: "五星装备", Value: strings.Join(p.Weapons5, "、")}, {Label: "四星装备", Value: strings.Join(p.Weapons4, "、")}}})
	}
	if len(v.Sections) == 0 {
		v.Note = "固定资料中没有匹配卡池，未推断快照之外的日程。"
	}
	return v
}

type PoolInfo struct {
	EstimatedStart bool     `json:"estimated_start,omitempty"`
	Version        string   `json:"version"`
	Half           string   `json:"half"`
	From           string   `json:"from"`
	To             string   `json:"to"`
	Kind           string   `json:"kind"`
	Characters5    []string `json:"characters5"`
	Characters4    []string `json:"characters4"`
	Weapons5       []string `json:"weapons5"`
	Weapons4       []string `json:"weapons4"`
}
type GameResources struct {
	Version string     `json:"version,omitempty"`
	Pools   []PoolInfo `json:"pools"`
}

// calendarQuery lists the banners of a version or open on a date (UTC+8),
// newest first, 50 a page.
func (a *App) calendarQuery(input map[string]any) (map[string]any, error) {
	data := a.bannerGame().Data.Resources
	var q struct {
		Date    string `json:"date"`
		Version string `json:"version"`
		Offset  int    `json:"offset"`
	}
	if decodeObject(input, &q) != nil || len(q.Version) > 32 || q.Offset < 0 {
		return nil, gameError("input_invalid", "卡池筛选条件无效。")
	}
	if q.Date != "" {
		if _, err := time.Parse("2006-01-02", q.Date); err != nil {
			return nil, gameError("input_invalid", "日期请使用 YYYY-MM-DD。")
		}
	}
	pools := []PoolInfo{}
	for _, p := range data.Pools {
		if q.Version != "" && p.Version != q.Version {
			continue
		}
		if q.Date != "" && (len(p.From) < 10 || len(p.To) < 10 || q.Date < p.From[:10] || q.Date > p.To[:10]) {
			continue
		}
		pools = append(pools, p)
	}
	slices.SortFunc(pools, func(a, b PoolInfo) int { return strings.Compare(b.From, a.From) })
	total := len(pools)
	offset := min(q.Offset, total)
	end := min(offset+50, total)
	var next *int
	if end < total {
		next = &end
	}
	return map[string]any{"pools": pools[offset:end], "total": total, "next_offset": next, "version": resourceVersion(a.Game), "timezone": "UTC+8", "source": "fixed_reference"}, nil
}
