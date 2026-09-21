package images

import (
	"strconv"

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
	counts := map[string]string{}
	list, _ := data["list"].([]any)
	for _, raw := range list {
		item, _ := raw.(map[string]any)
		counts[gamekit.Text(item["data_type"])] = gamekit.Text(item["count"])
	}
	count := func(kind string) string {
		if value := counts[kind]; value != "" {
			return value
		}
		return "0"
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
	month := gamekit.Text(result.Data["data_month"])
	if len(month) >= 2 {
		if number, err := strconv.Atoi(month[len(month)-2:]); err == nil {
			month = strconv.Itoa(number)
		}
	}
	return gamekit.Image{Template: "monthly", Data: map[string]any{
		"month": month + "月", "player": playerCard(result.Role),
		"poly": count("PolychromesData"), "tape": count("MatserTapeData"), "boopon": count("BooponsData"), "sources": sources,
	}, Resources: resources.List}, true
}
