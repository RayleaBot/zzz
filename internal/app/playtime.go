package app

import (
	"context"
	"slices"
	"time"
)

type PlaytimeDay struct {
	Date    string   `json:"date"`
	Minutes int      `json:"estimated_minutes"`
	Periods [][2]int `json:"periods"`
}

func estimatePlaytime(rows []BillingRow, action string) []PlaytimeDay {
	days := map[string][][2]int{}
	zone := time.FixedZone("UTC+8", 28800)
	for _, row := range rows {
		if row.Action != action || row.Change != "1" {
			continue
		}
		end, err := time.ParseInLocation("2006-01-02 15:04:05", row.Time, zone)
		if err != nil {
			continue
		}
		start := end.Add(-6 * time.Minute)
		for start.Before(end) {
			midnight := time.Date(start.Year(), start.Month(), start.Day()+1, 0, 0, 0, 0, zone)
			until := end
			if midnight.Before(end) {
				until = midnight
			}
			lo := start.Hour()*60 + start.Minute()
			hi := until.Hour()*60 + until.Minute()
			if !until.Before(midnight) {
				hi = 1440
			}
			if hi > lo {
				day := start.Format("2006-01-02")
				days[day] = append(days[day], [2]int{lo, hi})
			}
			start = until
		}
	}
	out := []PlaytimeDay{}
	for date, parts := range days {
		slices.SortFunc(parts, func(a, b [2]int) int { return a[0] - b[0] })
		merged := [][2]int{}
		for _, p := range parts {
			if len(merged) > 0 && p[0] <= merged[len(merged)-1][1] {
				merged[len(merged)-1][1] = max(merged[len(merged)-1][1], p[1])
			} else {
				merged = append(merged, p)
			}
		}
		minutes := 0
		for _, p := range merged {
			minutes += p[1] - p[0]
		}
		out = append(out, PlaytimeDay{date, minutes, merged})
	}
	slices.SortFunc(out, func(a, b PlaytimeDay) int {
		if a.Date > b.Date {
			return -1
		}
		if a.Date < b.Date {
			return 1
		}
		return 0
	})
	return out
}
func (a *App) playtimeAction(ctx context.Context, client AccountsClient, input map[string]any) (map[string]any, error) {
	if a.Game.ID != "starrail" {
		return nil, gameError("operation_denied", "开拓力时长估算仅适用于星铁。")
	}
	choice := Selection{asText(input["account_ref"]), asText(input["role_ref"])}
	if _, err := authorizeCloudRole(ctx, client, choice); err != nil {
		return nil, err
	}
	archive, err := a.Billing.Read(client.Provider, choice, "power", "all")
	if err != nil {
		return nil, err
	}
	actions := []string{}
	for _, row := range archive.Rows {
		if row.Action != "" && !slices.Contains(actions, row.Action) {
			actions = append(actions, row.Action)
		}
	}
	slices.Sort(actions)
	action := asText(input["recovery_action"])
	if action == "" {
		return map[string]any{"actions": actions, "days": []PlaytimeDay{}, "saved_ms": archive.SavedMS}, nil
	}
	if !slices.Contains(actions, action) {
		return nil, gameError("input_invalid", "请选择已保存记录中的自然恢复分类。")
	}
	return map[string]any{"actions": actions, "days": estimatePlaytime(archive.Rows, action), "saved_ms": archive.SavedMS, "note": "采用参考规则：每条增加1点开拓力的所选恢复记录，估计此前6分钟在线；跨午夜拆分、重叠区间去重。记录缺失、体力已满或更新节奏都会影响估算，空白时段并非已确认离线。"}, nil
}
