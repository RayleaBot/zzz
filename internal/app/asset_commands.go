package app

import (
	"context"
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func (a *App) assetCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	if command == "codes" {
		data, err := a.Content.codes(ctx, a.Game.ID)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return a.sendView(ctx, event, a.codesView(data))
	}
	if len(args) < 1 || len(args) > 2 {
		return event.SendText("使用“" + a.Game.Prefix + "兑换 代码 [UID]”确认兑换，或“" + a.Game.Prefix + "资产记录 类别 [UID]”查询；完整资产收集在插件管理页。")
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
	action := "billing.page"
	input := map[string]any{"account_ref": choice.AccountRef, "role_ref": choice.RoleRef, "category": args[0]}
	if command == "redeem" {
		action = "redeem.run"
		input["confirm"] = true
		input["code"] = args[0]
	} else {
		for _, c := range billingCategories(a.Game.ID) {
			if args[0] == c["label"] {
				input["category"] = c["id"]
			}
		}
	}
	result, err := a.assetAccountAction(ctx, client, action, input)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return a.sendView(ctx, event, result["view"].(View))
}
