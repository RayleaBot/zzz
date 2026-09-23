package app

import (
	"context"
	"regexp"
	"strconv"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

var digits = regexp.MustCompile(`^[0-9]+$`)

// numberArg is the number a command ends with, as upstream's (\d+)$: the
// last of want arguments, written in digits. ok is false otherwise, and the
// command is left to other plugins as upstream's rule does not match.
func numberArg(args []string, want int) (value int, ok bool) {
	if len(args) != want || !digits.MatchString(args[want-1]) {
		return 0, false
	}
	value, err := strconv.Atoi(args[want-1])
	return value, err == nil
}

// panelIntervalCommand is ZZZ-Plugin's 刷新面板间隔: the seconds between two
// refreshes of a UID's panels, 0–1000, kept in the plugin settings.
func (a *App) panelIntervalCommand(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	value, ok := numberArg(args, 1)
	if !ok {
		return event.Result(map[string]any{"handled": false})
	}
	if value > 1000 {
		return event.SendText("刷新面板间隔不能大于1000秒")
	}
	if _, err := event.Actions().ConfigWrite(ctx, map[string]any{"panel_refresh_interval": value}); err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText(a.Game.Name + "刷新面板间隔已设置为: " + strconv.Itoa(value))
}
