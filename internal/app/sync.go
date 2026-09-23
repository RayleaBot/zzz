package app

import (
	"context"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/gacha"
)

func (a *App) manageSync(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	return a.syncAction(ctx, a.accountClient(event), action, input)
}
func (a *App) syncAction(ctx context.Context, client AccountsClient, action string, input map[string]any) (result map[string]any, err error) {
	defer func() { err = syncError(err) }()
	ref := asText(input["ref"])
	switch action {
	case "gacha.sync.start":
		choice := Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}
		var response struct {
			Roles []Role `json:"roles"`
		}
		if err := client.call(ctx, "roles", map[string]any{"account_ref": choice.AccountRef, "game": a.Game.ID}, &response); err != nil {
			return nil, err
		}
		var role Role
		for _, candidate := range response.Roles {
			if candidate.Ref == choice.RoleRef && candidate.Game == a.Game.ID {
				role = candidate
				break
			}
		}
		if role.Ref == "" {
			return nil, gameError("role_missing", "请选择已授权的游戏角色。")
		}
		if !syncRegionAllowed(role.Region) {
			return nil, gameError("region_unsupported", "此区服暂未适配官方同步。")
		}
		full, ok := input["full"].(bool)
		if input["full"] != nil && !ok {
			return nil, gameError("input_invalid", "同步模式无效。")
		}
		info, err := a.Syncs.Start(a.Gacha, gacha.SyncChoice{AccountRef: choice.AccountRef, RoleRef: choice.RoleRef}, role.UID, role.Region, full)
		return map[string]any{"sync": info}, err
	case "gacha.sync.step":
		var payload struct {
			Sequence *int `json:"sequence"`
		}
		if decodeObject(input, &payload) != nil || payload.Sequence == nil || *payload.Sequence < 0 {
			return nil, gameError("input_invalid", "同步进度无效。")
		}
		choice, err := a.Syncs.Choice(ref)
		if err != nil || choice.Link {
			return nil, gacha.ErrSync
		}
		info, err := a.Syncs.Step(ctx, a.Gacha, ref, *payload.Sequence, func(ctx context.Context, pool, endID string, page int) (gacha.RemotePage, error) {
			response, err := client.Execute(ctx, Selection{AccountRef: choice.AccountRef, RoleRef: choice.RoleRef}, a.Game.ID+".gacha", map[string]any{"gacha_type": pool, "end_id": endID, "page": page})
			if err != nil {
				return gacha.RemotePage{}, err
			}
			return gacha.ParsePage(response.Role.UID, response.Role.Region, pool, endID, response.Data)
		})
		return map[string]any{"sync": info}, err
	case "gacha.sync.cancel":
		err := a.Syncs.Cancel(ref)
		return map[string]any{"canceled": err == nil}, err
	default:
		return nil, gameError("operation_denied", "操作不存在。")
	}
}
