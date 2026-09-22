package app

import (
	"context"
	"fmt"
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func (a *App) signinAction(ctx context.Context, client AccountsClient, action string, input map[string]any) (map[string]any, error) {
	choice := Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}
	var result QueryResult
	var err error
	switch action {
	case "signin.status":
		result, err = client.Execute(ctx, choice, a.Game.ID+".sign_info", nil)
	case "signin.rewards":
		result, err = client.Execute(ctx, choice, a.Game.ID+".sign_rewards", nil)
	case "signin.run":
		if input["confirm"] != true {
			return nil, gameError("input_invalid", "请明确选择为当前角色签到。")
		}
		result, err = client.ExecuteConfirmed(ctx, choice, a.Game.ID+".sign", nil)
	default:
		return nil, gameError("operation_denied", "签到操作不存在。")
	}
	if err != nil {
		return nil, err
	}
	view := View{Title: a.Game.Name + "游戏签到", Subtitle: result.Role.Nickname + " · " + result.Role.UID, Rows: []Row{}, Note: "来源：米游社官方游戏签到；不包含社区米游币和云游戏任务。"}
	if action == "signin.rewards" {
		for i, raw := range asList(result.Data["awards"]) {
			award := asObject(raw)
			view.Rows = append(view.Rows, Row{Label: fmt.Sprintf("第 %d 天", i+1), Value: asText(award["name"]) + " × " + asText(award["count"])})
		}
	} else {
		state := "今日未签到"
		if result.Data["signed"] == true {
			state = "今日已签到"
		}
		if result.Data["outcome"] == "signed" {
			state = "签到成功"
		}
		view.Rows = append(view.Rows, Row{Label: "签到状态", Value: state})
		if value := asText(result.Data["total_days"]); value != "" {
			view.Rows = append(view.Rows, Row{Label: "本月签到天数", Value: value})
		}
		if result.Data["first_bind"] == true {
			view.Note = "请在米游社官方应用完成首次签到后再使用此功能。"
		}
	}
	return map[string]any{"result": result, "view": view}, nil
}
func (a *App) signinCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	if len(args) > 1 {
		return event.SendText("请使用“" + a.Game.Prefix + "签到 [UID]”或“" + a.Game.Prefix + "签到状态 [UID]”。")
	}
	uid := ""
	if len(args) == 1 {
		uid = args[0]
	}
	client := a.accountClient(event)
	accounts, err := client.List(ctx, 0)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	choice, _, err := Choose(accounts, a.Game.ID, uid)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	action := "signin.status"
	if command == "signin" {
		action = "signin.run"
	}
	result, err := a.signinAction(ctx, client, action, map[string]any{"account_ref": choice.AccountRef, "role_ref": choice.RoleRef, "confirm": action == "signin.run"})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return a.sendView(ctx, event, result["view"].(View))
}
