package app

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

func (a *App) assetAccountAction(ctx context.Context, client AccountsClient, action string, input map[string]any) (map[string]any, error) {
	choice := Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}
	if action == "redeem.run" {
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
	if a.Game.ID == "zzz" {
		return nil, gameError("operation_denied", "当前参考未提供绝区零客服资产记录接口。")
	}
	params := map[string]any{}
	for _, key := range []string{"category", "direction", "end_id", "page"} {
		if v, ok := input[key]; ok {
			params[key] = v
		}
	}
	result, err := client.Execute(ctx, choice, a.Game.ID+".billing", params)
	if err != nil {
		return nil, err
	}
	view := billingPageView(a.Game, result)
	return map[string]any{"page": result.Data, "view": view}, nil
}
func billingPageView(game Game, result QueryResult) View {
	view := View{Title: game.Name + "官方资产变动记录", Subtitle: result.Role.UID, Rows: []Row{}, Note: "这是官方当前可提供的游戏资产变化记录，不是实际支付账单；金额未由官方明确提供时不推算人民币。"}
	for _, v := range asList(result.Data["items"]) {
		item := asObject(v)
		date := firstText(item, "datetime", "time")
		text := firstText(item, "action", "reason", "item_name")
		parts := []string{}
		for _, key := range []string{"add_num", "sub_num", "change_num"} {
			if value := asText(item[key]); value != "" {
				parts = append(parts, map[string]string{"add_num": "变动", "sub_num": "扣除", "change_num": "变化"}[key]+" "+value)
			}
		}
		view.Rows = append(view.Rows, Row{Label: date + " · " + text, Value: strings.Join(parts, " · ")})
	}
	if result.Data["has_more"] == true {
		view.Note += " 后面仍有记录，可继续翻页。"
	}
	if len(view.Rows) == 0 {
		view.Note += " 当前页没有符合筛选的记录。"
	}
	return view
}
func billingCategories(game string) []map[string]string {
	if game == "genshin" {
		return []map[string]string{{"id": "crystal", "label": "创世结晶"}, {"id": "primogem", "label": "原石"}}
	}
	if game == "starrail" {
		return []map[string]string{{"id": "dreams", "label": "古老梦华"}, {"id": "stellar", "label": "星琼"}, {"id": "power", "label": "开拓力"}, {"id": "relic", "label": "遗器"}, {"id": "cone", "label": "光锥"}}
	}
	return []map[string]string{}
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
func billingSelectionLabel(game Game, category string) string {
	for _, v := range billingCategories(game.ID) {
		if v["id"] == category {
			return v["label"]
		}
	}
	return category
}
func validBillingCategory(game, id string) bool {
	return slices.ContainsFunc(billingCategories(game), func(v map[string]string) bool { return v["id"] == id })
}
func assetSummaryTitle(game Game, category string) string {
	return fmt.Sprintf("%s · %s记录", game.Name, billingSelectionLabel(game, category))
}
