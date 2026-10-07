package app

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/zzz/internal/gacha"
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
// two of them, and then forgets it. proceed, when set, is asked before each
// page with the progress so far and stops the sync by returning false.
func (a *App) runSync(ctx context.Context, info gacha.SyncInfo, gap time.Duration, fetch gacha.FetchPage, proceed func(gacha.SyncInfo) bool) (gacha.SyncInfo, error) {
	defer a.Syncs.Forget(info.Ref)
	for info.State == "running" {
		if info.Pages > 0 {
			if err := a.sleep(ctx, gap); err != nil {
				return info, err
			}
		}
		if proceed != nil && !proceed(info) {
			return info, nil
		}
		next, err := a.Syncs.Step(ctx, a.Gacha, info.Ref, info.Sequence, fetch)
		if err != nil {
			return info, err
		}
		info = next
	}
	return info, nil
}

// syncPageGap is the least time between two pages an account sync reads,
// which keeps the reads under 米游社's rate limits.
const syncPageGap = time.Second

// accountPages reads role's pages with the account, under delegation when a
// scheduler trigger reads them; a page of another role is invalid.
func (a *App) accountPages(client AccountsClient, choice Selection, role Role, delegation string) gacha.FetchPage {
	return func(ctx context.Context, pool, endID string, page int) (gacha.RemotePage, error) {
		params := map[string]any{"account_ref": choice.AccountRef, "role_ref": choice.RoleRef, "operation": a.Game.ID + ".gacha", "input": map[string]any{"gacha_type": pool, "end_id": endID, "page": page}}
		if delegation != "" {
			params["delegation_ref"] = delegation
		}
		var response QueryResult
		if err := client.call(ctx, "execute", params, &response); err != nil {
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
	info, err = a.runSync(ctx, info, syncPageGap, a.accountPages(client, choice, role, ""), nil)
	if err != nil {
		return event.SendText(friendlyError(syncError(err)))
	}
	return event.SendText(a.gachaReply(before, *info.Result))
}

// A background sync is a sync the management page starts and leaves: the
// management action moves to the background with the page's answer and
// reads every channel in that one event, calling the account plugin as the
// page's user did. The page lists the latest ones and may cancel one that
// runs.

// backgroundSyncsKept is how many background syncs the page can list.
const backgroundSyncsKept = 16

// BackgroundSync is a background sync as the page lists it: the role it
// reads, the chat user told of its result, its state (running, completed,
// failed or canceled), the code of its last outcome and its progress.
type BackgroundSync struct {
	Ref        string         `json:"ref"`
	Role       Role           `json:"role"`
	Owner      Subject        `json:"owner"`
	Full       bool           `json:"full"`
	Notify     bool           `json:"notify"`
	State      string         `json:"state"`
	LastCode   string         `json:"last_code"`
	StartedMS  int64          `json:"started_ms"`
	FinishedMS int64          `json:"finished_ms,omitempty"`
	Progress   gacha.SyncInfo `json:"progress"`
	// Task is the 每日同步 a round belongs to.
	Task string `json:"task,omitempty"`
}

// backgroundSyncs are the latest background syncs of this process, newest
// first. They live in memory only: a sync ends with the event running it.
type backgroundSyncs struct {
	mu    sync.Mutex
	items []BackgroundSync
}

// errSyncRunning refuses a second background sync of a role.
var errSyncRunning = gameError("sync_task_running", "此角色正在后台同步，请查看进度。")

// running reports whether a sync of role is running.
func (s *backgroundSyncs) running(role Role) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.runningLocked(role)
}

func (s *backgroundSyncs) runningLocked(role Role) bool {
	return slices.ContainsFunc(s.items, func(item BackgroundSync) bool {
		return item.State == "running" && item.Role.UID == role.UID && item.Role.Region == role.Region
	})
}

// begin lists a starting sync, unless one of the same role is running, and
// drops the oldest finished syncs past backgroundSyncsKept.
func (s *backgroundSyncs) begin(item BackgroundSync) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runningLocked(item.Role) {
		return false
	}
	s.items = append([]BackgroundSync{item}, s.items...)
	for i := len(s.items) - 1; i >= 0 && len(s.items) > backgroundSyncsKept; i-- {
		if s.items[i].State != "running" {
			s.items = slices.Delete(s.items, i, i+1)
		}
	}
	return true
}

// update records a running sync's progress; false once it no longer runs.
func (s *backgroundSyncs) update(ref string, progress gacha.SyncInfo) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.items, func(item BackgroundSync) bool { return item.Ref == ref })
	if i < 0 || s.items[i].State != "running" {
		return false
	}
	s.items[i].Progress = progress
	return true
}

// finish records how a sync ended: its final progress and, unless the page
// or an archive removal canceled it meanwhile, its state and code. It
// returns the sync as listed.
func (s *backgroundSyncs) finish(ref string, progress gacha.SyncInfo, state, code string) BackgroundSync {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.items, func(item BackgroundSync) bool { return item.Ref == ref })
	if i < 0 {
		return BackgroundSync{}
	}
	item := &s.items[i]
	if item.State == "running" {
		item.State, item.LastCode, item.FinishedMS = state, code, time.Now().UnixMilli()
	}
	item.Progress = progress
	return *item
}

// drop removes a sync that never started.
func (s *backgroundSyncs) drop(ref string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = slices.DeleteFunc(s.items, func(item BackgroundSync) bool { return item.Ref == ref })
}

func (s *backgroundSyncs) list() []BackgroundSync {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.items)
}

// cancel marks the running syncs match picks as canceled with code, each to
// stop before its next page; false when none matched.
func (s *backgroundSyncs) cancel(match func(BackgroundSync) bool, code string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	canceled := false
	for i := range s.items {
		if item := &s.items[i]; item.State == "running" && match(*item) {
			item.State, item.LastCode, item.FinishedMS = "canceled", code, time.Now().UnixMilli()
			canceled = true
		}
	}
	return canceled
}

// cancelArchiveSyncs cancels the background syncs of an archive being
// removed; a page merged meanwhile finds the archive changed.
func (a *App) cancelArchiveSyncs(uid, region string) {
	a.BackgroundSyncs.cancel(func(item BackgroundSync) bool { return item.Role.UID == uid && item.Role.Region == region }, "archive_removed")
}

func (a *App) backgroundSyncAction(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	switch action {
	case "gacha.task.list":
		daily, err := a.DailySyncs.List()
		return map[string]any{"items": a.BackgroundSyncs.list(), "daily": daily}, err
	case "gacha.task.cancel":
		ref := asText(input["ref"])
		if !a.BackgroundSyncs.cancel(func(item BackgroundSync) bool { return item.Ref == ref }, "sync_canceled") {
			return nil, gameError("sync_task_missing", "没有正在进行的这项后台同步。")
		}
		return map[string]any{"canceled": true}, nil
	case "gacha.task.start":
		return a.startBackgroundSync(ctx, event, input)
	case "gacha.task.create":
		return a.createDailySync(ctx, event, input)
	case "gacha.task.run":
		return a.runDailySyncNow(ctx, event, input)
	case "gacha.task.remove":
		return a.removeDailySync(ctx, event, input)
	}
	return nil, gameError("operation_denied", "操作不存在。")
}

// startBackgroundSync checks the role, starts its sync and moves the
// management action to the background with the listed sync, then reads the
// pages one after another, syncPageGap apart, until the sync completes,
// fails, the page cancels it or the background deadline passes. A completed
// sync tells the account's chat user when asked.
func (a *App) startBackgroundSync(ctx context.Context, event *rayleabot.EventContext, input map[string]any) (map[string]any, error) {
	var q struct {
		Selection
		Full   bool `json:"full"`
		Notify bool `json:"notify"`
	}
	if input["confirm"] != true || decodeObject(input, &q) != nil {
		return nil, gameError("input_invalid", "请确认开始后台同步。")
	}
	client := a.accountClient(event)
	account, role, err := client.Authorize(ctx, q.Selection)
	if err != nil {
		return nil, err
	}
	if !syncRegionAllowed(role.Region) {
		return nil, gameError("region_unsupported", "此区服暂未适配官方同步。")
	}
	if a.BackgroundSyncs.running(role) {
		return nil, errSyncRunning
	}
	info, err := a.Syncs.Start(a.Gacha, gacha.SyncChoice{AccountRef: q.AccountRef, RoleRef: q.RoleRef}, role.UID, role.Region, q.Full)
	if err != nil {
		return nil, syncError(err)
	}
	item := BackgroundSync{Ref: info.Ref, Role: role, Owner: account.Owner, Full: q.Full, Notify: q.Notify, State: "running", LastCode: "sync_running", StartedMS: time.Now().UnixMilli(), Progress: info}
	if !a.BackgroundSyncs.begin(item) {
		a.Syncs.Forget(info.Ref)
		return nil, errSyncRunning
	}
	if _, err := event.Detach(ctx, map[string]any{"task": item}); err != nil {
		a.BackgroundSyncs.drop(item.Ref)
		a.Syncs.Forget(info.Ref)
		return nil, detachFailure(err)
	}
	item, err = a.readBackgroundSync(ctx, item, info, a.accountPages(client, q.Selection, role, ""), func(done gacha.SyncInfo) error {
		if !item.Notify {
			return nil
		}
		return a.notifySync(ctx, event, item, *done.Result)
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"task": item}, nil
}

// readBackgroundSync reads a listed sync to the end and lists how it ended.
// completed follows a completed sync with its final progress; its error is a
// notification that failed.
func (a *App) readBackgroundSync(ctx context.Context, item BackgroundSync, info gacha.SyncInfo, fetch gacha.FetchPage, completed func(gacha.SyncInfo) error) (BackgroundSync, error) {
	info, err := a.runSync(ctx, info, syncPageGap, fetch, func(progress gacha.SyncInfo) bool {
		return a.BackgroundSyncs.update(item.Ref, progress)
	})
	state, code := "completed", "sync_completed"
	switch {
	case ctx.Err() != nil:
		state, code = "failed", "sync_timeout"
	case err != nil:
		err = syncError(err)
		state, code = "failed", PublicError(err).Code
	case info.State != "completed":
		// Canceled: the listed sync already says so.
	case completed(info) != nil:
		code = "sync_completed.notification_failed"
	}
	return a.BackgroundSyncs.finish(item.Ref, info, state, code), err
}

// notifySync tells the account's chat user, through the bot the account
// belongs to, what a completed background sync added.
func (a *App) notifySync(ctx context.Context, event *rayleabot.EventContext, item BackgroundSync, result gacha.ImportResult) error {
	owner := item.Owner
	if !slices.ContainsFunc(event.Bots, func(bot rayleabot.Bot) bool {
		return bot.ID == owner.BotID && bot.SourceProtocol == owner.SourceProtocol && bot.SourceAdapter == owner.SourceAdapter
	}) {
		return gameError("bot_missing", "所属机器人暂不可用。")
	}
	text := fmt.Sprintf("抽卡后台同步完成\n%s · %s\n新增 %d 条，档案共 %d 条。", item.Role.Nickname, item.Role.UID, result.Added, result.Total)
	_, err := event.Actions().MessageSend(ctx, rayleabot.MessageSendRequest{SourceProtocol: owner.SourceProtocol, SourceAdapter: owner.SourceAdapter, TargetType: "private", TargetID: owner.ActorID, Message: rayleabot.MessageOut{Segments: []rayleabot.Segment{rayleabot.Text(text)}}})
	return err
}
