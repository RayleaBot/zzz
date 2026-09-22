package app

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type MaterialInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Kind        string   `json:"kind"`
	Rarity      int      `json:"rarity"`
	Family      string   `json:"family"`
	Days        []int    `json:"days"`
	City        string   `json:"city"`
	Sources     []string `json:"sources"`
	Description string   `json:"description"`
	Related     []string `json:"related"`
}

func resourceView(game Game, action string, result map[string]any) View {
	v := View{Title: game.Name + "固定资料", Rows: []Row{}, Note: "来源：" + asText(result["version"]) + "；固定快照，不代表已在线校准的当前活动。完整条目可在材料与卡池页面查询。"}
	if action == "materials.query" {
		items := result["materials"].([]MaterialInfo)
		v.Title = game.Name + "材料查询"
		for _, m := range items[:min(10, len(items))] {
			value := fmt.Sprintf("%d 星 · %s", m.Rarity, m.Family)
			if m.City != "" {
				value += " · " + m.City
			}
			if len(m.Days) > 0 {
				days := []string{}
				for _, d := range m.Days {
					days = append(days, []string{"", "周一", "周二", "周三", "周四", "周五", "周六", "周日"}[d])
				}
				value += " · " + strings.Join(days, "/")
			}
			if len(m.Sources) > 0 {
				value += "\n" + strings.Join(m.Sources, "；")
			}
			v.Rows = append(v.Rows, Row{Label: m.Name, Value: value})
		}
		if len(items) == 0 {
			v.Note = "固定资料中没有匹配材料。"
		}
		return v
	}
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
type Birthday struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Date string `json:"date"`
}
type GameResources struct {
	Version   string         `json:"version,omitempty"`
	Materials []MaterialInfo `json:"materials"`
	Pools     []PoolInfo     `json:"pools"`
	Birthdays []Birthday     `json:"birthdays"`
}

func (a *App) resourceQuery(action string, input map[string]any) (map[string]any, error) {
	data := a.Game.Data.Resources
	if action == "calendar.query" {
		data = a.bannerGame().Data.Resources
	}
	if action == "materials.query" {
		var q struct {
			Query   string `json:"query"`
			Weekday int    `json:"weekday"`
			Offset  int    `json:"offset"`
		}
		if decodeObject(input, &q) != nil || len(q.Query) > 128 || q.Offset < 0 || q.Weekday < 0 || q.Weekday > 7 || a.Game.ID != "genshin" && q.Weekday != 0 {
			return nil, gameError("input_invalid", "材料筛选条件无效。")
		}
		matches := []MaterialInfo{}
		query := strings.TrimSpace(strings.ToLower(q.Query))
		for _, m := range data.Materials {
			if query != "" && !strings.Contains(strings.ToLower(m.Name+" "+m.Family+" "+m.City+" "+m.ID), query) {
				continue
			}
			if q.Weekday != 0 && !slices.Contains(m.Days, q.Weekday) {
				continue
			}
			m.Related = []string{}
			for _, entry := range a.Catalog.Entries {
				for _, material := range entry.Materials {
					if material == m.Name || material == m.Family {
						m.Related = append(m.Related, entry.Name)
						break
					}
				}
			}
			m.Description = plainGameText(m.Description)
			matches = append(matches, m)
		}
		total := len(matches)
		offset := min(q.Offset, total)
		end := min(offset+100, total)
		var next *int
		if end < total {
			next = &end
		}
		return map[string]any{"materials": matches[offset:end], "total": total, "next_offset": next, "version": resourceVersion(a.Game)}, nil
	}
	if action != "calendar.query" {
		return nil, gameError("operation_denied", "资料操作不存在。")
	}
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
	pools = pools[offset:end]
	date := q.Date
	if date == "" {
		date = time.Now().In(time.FixedZone("UTC+8", 8*60*60)).Format("2006-01-02")
	}
	birthdays := []Birthday{}
	for _, b := range data.Birthdays {
		if b.Date == date[5:] {
			birthdays = append(birthdays, b)
		}
	}
	return map[string]any{"pools": pools, "total": total, "next_offset": next, "birthdays": birthdays, "birthday_date": date, "version": resourceVersion(a.Game), "timezone": "UTC+8", "source": "fixed_reference"}, nil
}
