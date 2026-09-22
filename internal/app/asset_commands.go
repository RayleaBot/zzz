package app

import (
	"context"
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func (a *App) assetCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	if command == "codes" {
		data, err := a.Content.codes(ctx)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return a.sendView(ctx, event, a.codesView(data))
	}
	if len(args) < 1 || len(args) > 2 {
		return event.SendText("使用“" + a.Game.Prefix + "兑换 代码 [UID]”确认兑换。")
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
	input := map[string]any{"account_ref": choice.AccountRef, "role_ref": choice.RoleRef, "confirm": true, "code": args[0]}
	result, err := a.redeem(ctx, client, input)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return a.sendView(ctx, event, result["view"].(View))
}
