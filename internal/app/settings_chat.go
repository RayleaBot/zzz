package app

import (
	"context"
	"strconv"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// panelIntervalCommand is ZZZ-Plugin's 刷新面板间隔: the seconds between two
// refreshes of a UID's panels, 0–1000, kept in the plugin settings.
func (a *App) panelIntervalCommand(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	value := -1
	if len(args) > 0 {
		if parsed, err := strconv.Atoi(args[0]); err == nil {
			value = parsed
		}
	}
	switch {
	case len(args) == 0:
		return event.SendText("请在命令后写上秒数，如“" + a.Game.Prefix + "设置刷新面板间隔60”。")
	case value < 0:
		return event.SendText("刷新面板间隔不能小于0秒")
	case value > 1000:
		return event.SendText("刷新面板间隔不能大于1000秒")
	}
	if _, err := event.Actions().ConfigWrite(ctx, map[string]any{"panel_refresh_interval": value}); err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText(a.Game.Name + "刷新面板间隔已设置为: " + strconv.Itoa(value))
}
