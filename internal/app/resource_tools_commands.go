package app

import (
	"context"
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"strconv"
)

func (a *App) resourceToolsCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	switch command {
	case "guides", "guide-help", "guide-default":
		return a.guideCommand(ctx, event, command, args)
	case "map":
		return a.mapCommand(ctx, event)
	case "enemy":
		if len(args) < 2 || len(args) > 3 {
			return event.SendText("使用“" + a.Game.Prefix + "原魔 名称 生命值/攻击力 [等级]”。修饰因子在攻略与工具页选择。")
		}
		stat := map[string]string{"生命值": "HP", "攻击力": "ATK", "HP": "HP", "ATK": "ATK"}[args[1]]
		level := 90
		if len(args) == 3 {
			var err error
			level, err = strconv.Atoi(args[2])
			if err != nil {
				return event.SendText("等级应为1–200。")
			}
		}
		out, err := a.enemyAction("enemies.query", map[string]any{"name": args[0], "stat": stat, "level": level})
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return a.sendView(ctx, event, out["view"].(View))
	case "blueprint":
		if len(args) < 1 || len(args) > 2 {
			return event.SendText("使用“" + a.Game.Prefix + "摹本 分享码 [UID]”。选择制作数量及完整材料计算见攻略与工具页。")
		}
		uid := ""
		if len(args) == 2 {
			uid = args[1]
		}
		client := a.accountClient(event)
		list, err := client.List(ctx, 0)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		choice, _, err := Choose(list, a.Game.ID, uid)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		out, err := a.blueprintAction(ctx, client, "blueprint.read", map[string]any{"account_ref": choice.AccountRef, "role_ref": choice.RoleRef, "share_code": args[0]})
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return a.sendView(ctx, event, out["view"].(View))
	}
	return event.Result(map[string]any{"handled": false})
}
