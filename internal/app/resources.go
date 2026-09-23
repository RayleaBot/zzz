package app

import (
	"slices"
	"strings"
	"time"
)

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
