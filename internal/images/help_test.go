package images

import (
	"testing"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

func TestHelpFollowsZZZPlugin(t *testing.T) {
	help := gamekit.HelpImage{Title: "绝区零帮助", Groups: []gamekit.HelpGroup{
		{ID: "records", Title: "战绩查询", Commands: []gamekit.HelpCommand{{ID: "challenge", Name: "式舆防卫战", Usage: "%式舆防卫战", Description: "式舆防卫战"}}},
		{ID: "images", Title: "图片与互动", Commands: []gamekit.HelpCommand{{ID: "poke", Name: "戳一戳"}}},
	}}
	image, ok := Help(gamekit.ImageContext{Game: gamekit.Game{Prefix: "%"}}, help)
	if !ok {
		t.Fatal("help")
	}
	parts := image.Data["parts"].([]any)
	records, fun := parts[0].(map[string]any), parts[1].(map[string]any)
	item := records["items"].([]any)[0].(map[string]any)
	// Groups take the icon of upstream's part on the same subject, the rest
	// that of its 其他 part; the stylesheet adds the % before commands.
	if records["icon"] != "Ether" || fun["icon"] != "Fire" || item["name"] != "式舆防卫战" || item["command"] != "式舆防卫战" {
		t.Errorf("parts = %v", parts)
	}
}
