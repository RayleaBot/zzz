package images

import (
	"strings"

	"github.com/RayleaBot/plugin-zzz/internal/app"
)

// helpIcons gives each group, by id, the attribute icon of ZZZ-Plugin's help
// part on the same subject; the rest take the icon of its 其他 part.
var helpIcons = map[string]string{"info": "Fire", "gacha": "Ice", "panel": "Electric", "records": "Ether", "group-rank": "Fire", "banners": "Frost", "remind": "HonedEdge", "guides": "Ice"}

// Help draws the help menu the way ZZZ-Plugin's help page does: the special
// title, then each group with its attribute icon and every command's name,
// usage and description.
func Help(context app.ImageContext, help app.HelpImage) (app.Image, bool) {
	if len(help.Groups) == 0 {
		return app.Image{}, false
	}
	resources := newRecordResources(context, commonArtwork)
	parts := []any{}
	for _, group := range help.Groups {
		icon := helpIcons[group.ID]
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
	return app.Image{Template: "help", Data: map[string]any{"title": help.Title, "parts": parts, "bars": make([]int, 8)}, Resources: resources.List}, true
}
