package app

import (
	"context"
)

func (a *App) blueprintAction(ctx context.Context, client AccountsClient, action string, input map[string]any) (map[string]any, error) {
	if a.Game.ID != "genshin" {
		return nil, gameError("operation_denied", "尘歌壶功能仅适用于原神。")
	}
	params := map[string]any{}
	operation := "genshin.blueprint_read"
	title := "尘歌壶摹本摆设"
	if action == "blueprint.read" {
		params["share_code"] = input["share_code"]
	} else {
		operation = "genshin.blueprint_compute"
		title = "尘歌壶材料需求"
		params["list"] = input["list"]
	}
	result, err := client.Execute(ctx, Selection{asText(input["account_ref"]), asText(input["role_ref"])}, operation, params)
	if err != nil {
		return nil, err
	}
	view := View{Title: title, Subtitle: result.Role.UID, Rows: []Row{}, Note: "来源：官方养成计算器，仅查询所需数量，不读取本人摆设库存或执行游戏内放置。"}
	for _, v := range asList(result.Data["list"]) {
		item := asObject(v)
		view.Rows = append(view.Rows, Row{firstText(item, "name", "furniture_name", "id"), firstText(item, "num", "cnt")})
	}
	return map[string]any{"items": result.Data["list"], "view": view}, nil
}
