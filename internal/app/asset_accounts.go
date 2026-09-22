package app

import (
	"context"
	"strings"
)

// redeem uses a 兑换码 for the chosen role after the user confirmed it.
func (a *App) redeem(ctx context.Context, client AccountsClient, input map[string]any) (map[string]any, error) {
	choice := Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}
	{
		if input["confirm"] != true {
			return nil, gameError("input_invalid", "请确认兑换所选代码。")
		}
		code := strings.TrimSpace(asText(input["code"]))
		if !publicCodePattern.MatchString(code) {
			return nil, gameError("input_invalid", "兑换码格式无效。")
		}
		result, err := client.ExecuteConfirmed(ctx, choice, a.Game.ID+".redeem", map[string]any{"code": code})
		if err != nil {
			return nil, err
		}
		state := "兑换成功"
		if result.Data["outcome"] == "already_redeemed" {
			state = "此代码已经兑换过"
		}
		return map[string]any{"view": View{Title: a.Game.Name + "兑换码", Subtitle: result.Role.UID, Rows: []Row{{Label: code, Value: state}}, Note: "结果由官方兑换接口返回，请在游戏内核对奖励。"}, "result": result}, nil
	}
}
func (a *App) codesView(result map[string]any) View {
	view := View{Title: a.Game.Name + "官方前瞻兑换码", Subtitle: asText(result["title"]), Rows: []Row{}, Note: "来源：官方前瞻活动。请以官方有效期与实际兑换结果为准。"}
	for _, v := range result["items"].([]map[string]any) {
		text := asText(v["reward"])
		if expiry := asText(v["expires_at"]); expiry != "" {
			label := "有效期"
			if v["expiry_estimated"] == true {
				label = "参考估计期限"
			}
			text += " · " + label + " " + expiry
		}
		view.Rows = append(view.Rows, Row{Label: asText(v["code"]), Value: text})
	}
	if len(view.Rows) == 0 {
		view.Note = "当前官方入口没有可读取的兑换码；不代表其他渠道没有发布。"
	}
	return view
}
