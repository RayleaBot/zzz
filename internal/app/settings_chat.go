package app

import (
	"context"
	"strconv"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// intervalSettings are ZZZ-Plugin's 刷新抽卡间隔, 刷新面板间隔 and 刷新角色间隔
// by command: the setting, its name, unit and range.
var intervalSettings = map[string]struct {
	key, name, unit string
	low, high       int
}{
	"setting-gacha-interval": {"gacha_refresh_interval", "刷新抽卡间隔", "秒", 0, 1000},
	"setting-panel-interval": {"panel_refresh_interval", "刷新面板间隔", "秒", 0, 1000},
	"setting-role-interval":  {"panel_role_interval", "刷新角色间隔", "毫秒", 100, 10000},
}

// settingCommand changes one of the intervals in the plugin settings, with
// upstream's replies.
func (a *App) settingCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	setting, ok := intervalSettings[command]
	if !ok {
		return event.Result(map[string]any{"handled": false})
	}
	value := -1
	if len(args) > 0 {
		if parsed, err := strconv.Atoi(args[0]); err == nil {
			value = parsed
		}
	}
	switch {
	case len(args) == 0:
		return event.SendText("请在命令后写上" + setting.unit + "数，如“" + a.Game.Prefix + "设置" + setting.name + strconv.Itoa(max(setting.low, 60)) + "”。")
	case value < setting.low:
		return event.SendText(setting.name + "不能小于" + strconv.Itoa(setting.low) + setting.unit)
	case value > setting.high:
		return event.SendText(setting.name + "不能大于" + strconv.Itoa(setting.high) + setting.unit)
	}
	if _, err := event.Actions().ConfigWrite(ctx, map[string]any{setting.key: value}); err != nil {
		return event.SendText(friendlyError(err))
	}
	text := a.Game.Name + setting.name + "已设置为: " + strconv.Itoa(value)
	if command == "setting-role-interval" {
		text += "毫秒"
	}
	return event.SendText(text)
}
