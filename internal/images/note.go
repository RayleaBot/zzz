// Package images draws this plugin's replies with its own templates, following
// the ZZZ-Plugin images for the same commands.
package images

import (
	"fmt"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/app"
)

// Builders lists the image builders by the operation they draw.
func Builders() map[string]app.ImageBuilder {
	return map[string]app.ImageBuilder{"zzz.note": Note, "zzz.challenge": Abyss, "zzz.deadly": Deadly, "zzz.holo_boss": HoloBoss, "zzz.void_front": VoidFront, "zzz.tower": Tower, "zzz.profile": Card, "zzz.characters": Card, "zzz.training": Training, "zzz.monthly": Monthly,
		"zzz.hollow_zero": HollowZero, "zzz.lost_void": LostVoid, "zzz.zenkov": Zenkov, "zzz.zenkov_detail": ZenkovDetail,
		"zzz.exploration": Exploration}
}

// Queries lists the commands that run another operation's query: 练度统计
// draws on the agent list.
func Queries() map[string]func(time.Time) string {
	return map[string]func(time.Time) string{"zzz.training": func(time.Time) string { return "zzz.characters" }}
}

// regionNames are the server names the official role list shows.
var regionNames = map[string]string{"prod_gf_cn": "新艾利都", "prod_gf_us": "America", "prod_gf_eu": "Europe", "prod_gf_jp": "Asia", "prod_gf_sg": "TW/HK/MO"}

// noteArtwork maps resource IDs to ZZZ-Plugin files used by the note.
var noteArtwork = [][2]string{
	{"page-bg", "resources/common/images/bg.jpg"},
	{"avatar-frame", "resources/common/images/SuitBg.png"},
	{"uid-frame", "resources/common/images/UIDBg.png"},
	{"card-frame", "resources/note/images/BgFrame.png"},
	{"battery-bg", "resources/note/images/batteryBg.png"},
	{"battery-icon", "resources/note/images/IconStamina.png"},
	{"battery-icon-frame", "resources/note/images/PetSelectBG.png"},
	{"battery-bar-frame", "resources/note/images/ActivityGeneralBtnBg02.png"},
	{"status-done", "resources/note/images/yes.png"},
	{"status-open", "resources/note/images/no.png"},
	{"zzz", "resources/common/fonts/inpinhongmengti.ttf"},
}

// Note draws the real-time note the way ZZZ-Plugin's note does: the player
// card, battery charge with the remaining recovery time, and the daily
// activity, video store and scratch card states.
func Note(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	data := result.Data
	energy, _ := data["energy"].(map[string]any)
	progress, _ := energy["progress"].(map[string]any)
	if progress == nil {
		return app.Image{}, false
	}
	current, maximum := app.Int(progress["current"]), app.Int(progress["max"])
	percent := 0
	if maximum > 0 {
		percent = current * 100 / maximum
	}
	// Upstream reads the restore seconds as a UTC clock, so it wraps at a day.
	restore := app.Int(energy["restore"])
	vitality, _ := data["vitality"].(map[string]any)
	sale, _ := data["vhs_sale"].(map[string]any)
	selling := strings.Contains(app.Text(sale["sale_state"]), "Doing")
	signed := strings.Contains(app.Text(data["card_sign"]), "Done")

	resources := []rayleabot.RenderImageResource{}
	for _, item := range noteArtwork {
		if resource, ok := context.ArtworkResource(item[0], "zzz-plugin", item[1]); ok {
			resources = append(resources, resource)
		}
	}
	region := regionNames[result.Role.Region]
	if region == "" {
		region = result.Role.Region
	}
	return app.Image{
		Template: "note",
		Data: map[string]any{
			"player": map[string]any{"nickname": result.Role.Nickname, "level": result.Role.Level, "region": region, "uid": result.Role.UID},
			"energy": map[string]any{"current": current, "max": maximum, "percent": percent, "rest": fmt.Sprintf("%d小时%d分钟", restore/3600%24, restore/60%60)},
			"activities": []any{
				map[string]any{"title": "今日活跃度", "finished": app.Int(vitality["current"]) == app.Int(vitality["max"]), "value": app.Text(vitality["current"]), "sub": "/" + app.Text(vitality["max"])},
				map[string]any{"title": "录像店经营", "finished": selling, "value": map[bool]string{true: "正在营业", false: "尚未营业"}[selling], "sub": ""},
				map[string]any{"title": "饼铺盲盒/刮刮卡/占卜", "finished": signed, "value": map[bool]string{true: "已完成", false: "未完成"}[signed], "sub": ""},
			},
		},
		Resources: resources,
	}, true
}
