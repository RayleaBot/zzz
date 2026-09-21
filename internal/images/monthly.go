package images

import (
	"strconv"
	"strings"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

// monthlyArtwork is what monthly adds.
var monthlyArtwork = [][2]string{
	{"monthly-images-bamboo", "resources/monthly/images/bamboo.png"},
	{"monthly-images-bg", "resources/monthly/images/bg.png"},
	{"monthly-images-bottom-deco", "resources/monthly/images/bottom-deco.gif"},
	{"monthly-images-container", "resources/monthly/images/container.png"},
	{"monthly-images-icon-bangboo", "resources/monthly/images/icon-bangboo.png"},
	{"monthly-images-icon-feilin", "resources/monthly/images/icon-feilin.png"},
	{"monthly-images-icon-matser", "resources/monthly/images/icon-matser.png"},
	{"monthly-images-icon-title-deco", "resources/monthly/images/icon-title-deco.png"},
	{"monthly-images-itembox", "resources/monthly/images/itembox.png"},
	{"monthly-images-list", "resources/monthly/images/list.png"},
}

// monthlySources are ZZZ-Plugin's names for the Polychrome sources.
var monthlySources = map[string]string{"daily_activity_rewards": "日常活跃奖励", "growth_rewards": "成长奖励", "event_rewards": "活动奖励", "hollow_rewards": "零号空洞奖励",
	"shiyu_rewards": "式舆防卫战奖励", "mail_rewards": "邮件奖励", "other_rewards": "其他奖励"}

// Monthly draws 月报 the way ZZZ-Plugin's monthly page does: the month, the
// player card, the Polychromes, Master Tapes and Boopons earned, and each
// Polychrome source with its share as a bar.
func Monthly(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	data, _ := result.Data["month_data"].(map[string]any)
	// Upstream answers in text for an empty month.
	if data == nil {
		return gamekit.Image{}, false
	}
	resources := newRecordResources(context, commonArtwork, monthlyArtwork)
	report, _ := monthlyReport(data)
	month := gamekit.Text(result.Data["data_month"])
	if len(month) >= 2 {
		if number, err := strconv.Atoi(month[len(month)-2:]); err == nil {
			month = strconv.Itoa(number)
		}
	}
	report["month"], report["player"] = month+"月", playerCard(result.Role)
	return gamekit.Image{Template: "monthly", Data: report, Resources: resources.List}, true
}

// monthlyReport is what ZZZ-Plugin's Monthly model shows of a month: the
// Polychromes, Master Tapes and Boopons earned, each 0 when missing, and
// every Polychrome source with its share; counts holds the three amounts.
func monthlyReport(data map[string]any) (report map[string]any, counts [3]int) {
	kinds := map[string]int{"PolychromesData": 0, "MatserTapeData": 1, "BooponsData": 2}
	list, _ := data["list"].([]any)
	for _, raw := range list {
		item, _ := raw.(map[string]any)
		if kind, ok := kinds[gamekit.Text(item["data_type"])]; ok {
			counts[kind] = gamekit.Int(item["count"])
		}
	}
	sources := []any{}
	components, _ := data["income_components"].([]any)
	for _, raw := range components {
		item, _ := raw.(map[string]any)
		name := monthlySources[gamekit.Text(item["action"])]
		if name == "" {
			name = "未知奖励"
		}
		sources = append(sources, map[string]any{"name": name, "percent": gamekit.Text(item["percent"]), "num": gamekit.Text(item["num"])})
	}
	return map[string]any{"poly": strconv.Itoa(counts[0]), "tape": strconv.Itoa(counts[1]), "boopon": strconv.Itoa(counts[2]), "sources": sources}, counts
}

// MonthlyCollect draws 月报统计 the way ZZZ-Plugin's monthly collect page
// does: the span of the saved months, the player card, their Polychromes,
// Master Tapes and Boopons together, then each month, newest first, with its
// counts and Polychrome sources.
func MonthlyCollect(context gamekit.ImageContext, stats gamekit.MonthlyStats) (gamekit.Image, bool) {
	months := []any{}
	var totals [3]int
	for index := len(stats.Months) - 1; index >= 0; index-- {
		data, _ := stats.Months[index].Data["month_data"].(map[string]any)
		if data == nil {
			continue
		}
		report, counts := monthlyReport(data)
		for kind, count := range counts {
			totals[kind] += count
		}
		year, month, _ := strings.Cut(stats.Months[index].Month, "-")
		number, _ := strconv.Atoi(month)
		report["date"] = year + "年" + strconv.Itoa(number) + "月"
		months = append(months, report)
	}
	if len(months) == 0 {
		return gamekit.Image{}, false
	}
	resources := newRecordResources(context, commonArtwork, monthlyArtwork)
	return gamekit.Image{Template: "monthly-collect", Data: map[string]any{
		"range":  months[len(months)-1].(map[string]any)["date"].(string) + "～" + months[0].(map[string]any)["date"].(string),
		"player": playerCard(stats.Role), "poly": strconv.Itoa(totals[0]), "tape": strconv.Itoa(totals[1]), "boopon": strconv.Itoa(totals[2]), "months": months,
	}, Resources: resources.List}, true
}
