package app

import (
	"context"
	"fmt"
	"slices"
	"time"

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

// runSync reads a started sync to the end, page after page with gap between
// two of them, and then forgets it.
func (a *App) runSync(ctx context.Context, ref string, gap time.Duration, fetch gacha.FetchPage) (gacha.ImportResult, error) {
	defer a.Syncs.Forget(ref)
	for sequence := 0; ; {
		info, err := a.Syncs.Step(ctx, a.Gacha, ref, sequence, fetch)
		if err != nil {
			return gacha.ImportResult{}, err
		}
		if info.State == "completed" {
			return *info.Result, nil
		}
		sequence = info.Sequence
		if err := a.sleep(ctx, gap); err != nil {
			return gacha.ImportResult{}, err
		}
	}
}

// syncPageGap is the least time between two pages an account sync reads,
// which keeps the reads under 米游社's rate limits.
const syncPageGap = time.Second

// accountPages reads role's pages with the account; a page of another role
// is invalid.
func (a *App) accountPages(client AccountsClient, choice Selection, role Role) gacha.FetchPage {
	return func(ctx context.Context, pool, endID string, page int) (gacha.RemotePage, error) {
		response, err := client.Execute(ctx, choice, a.Game.ID+".gacha", map[string]any{"gacha_type": pool, "end_id": endID, "page": page})
		if err != nil {
			return gacha.RemotePage{}, err
		}
		if response.Role.UID != role.UID || response.Role.Region != role.Region {
			return gacha.RemotePage{}, gacha.ErrInvalid
		}
		return gacha.ParsePage(role.UID, role.Region, pool, endID, response.Data)
	}
}

// gachaRefresh is ZZZ-Plugin's 更新抽卡记录 of the role named or in use. The
// event moves to the background, says that the records are being read,
// reads every channel with the account, page after page, and answers
// upstream's report of the channels.
func (a *App) gachaRefresh(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	if len(args) > 1 {
		return event.SendText("请指定一个 UID，或省略以使用默认角色。")
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
	choice, role, err := Choose(accounts, a.Game.ID, uid)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if !syncRegionAllowed(role.Region) {
		return event.SendText("此区服暂未适配官方同步。")
	}
	if detached, err := detachChat(ctx, event); !detached {
		return err
	}
	notice(ctx, event, "抽卡记录获取中请稍等...可能需要一段时间，请耐心等待")
	before := gachaPoolCounts(a.archiveOrEmpty(role.UID, role.Region))
	info, err := a.Syncs.Start(a.Gacha, gacha.SyncChoice{AccountRef: choice.AccountRef, RoleRef: choice.RoleRef}, role.UID, role.Region, false)
	if err != nil {
		return event.SendText(friendlyError(syncError(err)))
	}
	result, err := a.runSync(ctx, info.Ref, syncPageGap, a.accountPages(client, choice, role))
	if err != nil {
		return event.SendText(friendlyError(syncError(err)))
	}
	return event.SendText(a.gachaReply(before, result))
}

// backgroundSync is the management page's 后台同步 of a role. Once the sync
// has started, the action moves to the background, answering the page with
// the sync, and reads every channel page after page as 更新抽卡记录 does;
// the page may close meanwhile. With notify, the account's owner is told in
// private chat how it ended.
func (a *App) backgroundSync(ctx context.Context, event *rayleabot.EventContext, input map[string]any) error {
	var q struct {
		Selection
		Full   bool `json:"full"`
		Notify bool `json:"notify"`
	}
	if input["confirm"] != true || decodeObject(input, &q) != nil {
		return manageFailure(event, gameError("input_invalid", "请确认开始后台同步。"))
	}
	client := a.accountClient(event)
	account, role, err := client.Authorize(ctx, q.Selection)
	if err != nil {
		return manageFailure(event, err)
	}
	if !syncRegionAllowed(role.Region) {
		return manageFailure(event, gameError("region_unsupported", "此区服暂未适配官方同步。"))
	}
	info, err := a.Syncs.Start(a.Gacha, gacha.SyncChoice{AccountRef: q.AccountRef, RoleRef: q.RoleRef}, role.UID, role.Region, q.Full)
	if err != nil {
		return manageFailure(event, syncError(err))
	}
	if _, err := event.Detach(ctx, map[string]any{"sync": info}); err != nil {
		a.Syncs.Forget(info.Ref)
		return manageFailure(event, detachFailure(err))
	}
	result, err := a.runSync(ctx, info.Ref, syncPageGap, a.accountPages(client, q.Selection, role))
	if err != nil {
		err = syncError(err)
	}
	if q.Notify {
		text := fmt.Sprintf("抽卡后台同步完成\n%s · %s\n新增 %d 条，档案共 %d 条。", role.Nickname, role.UID, result.Added, result.Total)
		if err != nil {
			text = fmt.Sprintf("抽卡后台同步未完成\n%s · %s\n%s", role.Nickname, role.UID, friendlyError(err))
		}
		notifyOwner(ctx, event, account.Owner, text)
	}
	if err != nil {
		failure := PublicError(err)
		return event.Fail(failure.Code, failure.Message)
	}
	return event.Result(map[string]any{"added": result.Added, "total": result.Total})
}

// notifyOwner tells an account's owner text in private chat, through the
// owner's bot while it is online.
func notifyOwner(ctx context.Context, event *rayleabot.EventContext, owner Subject, text string) {
	if !slices.ContainsFunc(event.Bots, func(bot rayleabot.Bot) bool {
		return bot.ID == owner.BotID && bot.SourceProtocol == owner.SourceProtocol && bot.SourceAdapter == owner.SourceAdapter
	}) {
		return
	}
	_, _ = event.Actions().MessageSend(ctx, rayleabot.MessageSendRequest{SourceProtocol: owner.SourceProtocol, SourceAdapter: owner.SourceAdapter, TargetType: "private", TargetID: owner.ActorID, Message: rayleabot.MessageOut{Segments: []rayleabot.Segment{rayleabot.Text(text)}}})
}
