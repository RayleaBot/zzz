package images

import (
	"strings"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

// helpIcons gives each group the attribute icon of ZZZ-Plugin's help part on
// the same subject; the rest take the icon of its 其他 part.
var helpIcons = map[string]string{"信息查询": "Fire", "抽卡": "Ice", "角色面板": "Electric", "战绩": "Ether", "排名": "Fire", "提醒": "HonedEdge", "图鉴与养成": "Ice"}

// Help draws the help menu the way ZZZ-Plugin's help page does: the special
// title, then each group with its attribute icon and every command's name,
// usage and description.
func Help(context gamekit.ImageContext, help gamekit.HelpImage) (gamekit.Image, bool) {
	if len(help.Groups) == 0 {
		return gamekit.Image{}, false
	}
	resources := newRecordResources(context, commonArtwork)
	parts := []any{}
	for _, group := range help.Groups {
		icon := helpIcons[group.Title]
		if icon == "" {
			icon = "Fire"
		}
		items := []any{}
		for _, command := range group.Commands {
			// Upstream's stylesheet writes the % before each command.
			usage := strings.TrimPrefix(command.Usage, context.Game.Prefix)
			items = append(items, map[string]any{"name": command.Name, "command": usage, "desc": command.Description})
		}
		parts = append(parts, map[string]any{"title": group.Title, "icon": icon, "items": items})
	}
	return gamekit.Image{Template: "help", Data: map[string]any{"title": help.Title, "parts": parts, "bars": make([]int, 8)}, Resources: resources.List}, true
}
