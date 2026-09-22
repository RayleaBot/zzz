package app

import (
	"context"
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func (a *App) birthdayAction(ctx context.Context, client AccountsClient, action string, input map[string]any) (map[string]any, error) {
	if a.Game.ID != "genshin" {
		return nil, gameError("operation_denied", "留影叙佳期仅适用于原神。")
	}
	choice := Selection{asText(input["account_ref"]), asText(input["role_ref"])}
	var result QueryResult
	var err error
	if action == "birthday.claim" {
		if input["confirm"] != true {
			return nil, gameError("input_invalid", "请明确领取所选生日留影。")
		}
		result, err = client.ExecuteConfirmed(ctx, choice, "genshin.birthday_claim", map[string]any{"role_id": asText(input["role_id"])})
	} else {
		result, err = client.Execute(ctx, choice, "genshin.birthday_list", nil)
	}
	if err != nil {
		return nil, err
	}
	view := View{Title: "留影叙佳期", Subtitle: result.Role.UID, Rows: []Row{}}
	if action == "birthday.claim" {
		view.Rows = append(view.Rows, Row{"领取请求", "官方已接受"})
	} else {
		for _, v := range asList(result.Data["items"]) {
			row := asObject(v)
			view.Rows = append(view.Rows, Row{asText(row["name"]), "角色编号 " + asText(row["role_id"])})
		}
		if len(view.Rows) == 0 {
			view.Note = "今天没有可读取的生日角色。"
		} else {
			view.Note = "在游戏插件管理页查看官方图片并明确领取。"
		}
	}
	return map[string]any{"view": view, "data": result.Data}, nil
}
func (a *App) birthdayCommand(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	if len(args) > 1 {
		return event.SendText("使用“" + a.Game.Prefix + "生日留影 [本人UID]”，领取在管理页逐项确认。")
	}
	uid := ""
	if len(args) == 1 {
		uid = args[0]
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
	out, err := a.birthdayAction(ctx, client, "birthday.list", map[string]any{"account_ref": choice.AccountRef, "role_ref": choice.RoleRef})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return a.sendView(ctx, event, out["view"].(View))
}
