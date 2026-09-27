package app

import (
	"context"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// panelTask prefixes the scheduled tasks that finish an account 更新面板.
const panelTask = "game.panel."

// panelTaskID is the scheduled task of a user's refreshes of a role. Its
// reads use a delegation, which the account plugin lets only the user who
// created it renew, so each user has a task per role and renews, rather than
// adds to, its delegation.
func panelTaskID(game, provider string, user Subject, choice Selection) string {
	return roleTaskID(panelTask, game, strings.Join([]string{provider, user.SourceProtocol, user.SourceAdapter, user.BotID, user.ActorID}, "\x00"), choice)
}

// panelRefresh is an account 更新面板 of a role, read as ZZZ-Plugin's
// getAvatarInfoList reads it: each character's details in a request of its
// own, roleInterval after the previous one. The panels are kept once all
// are read; a failed read ends the refresh without keeping any, as upstream.
type panelRefresh struct {
	uid      string
	choice   Selection
	provider string
	player   ShowcaseProfile
	// ids are the characters still to read, panels those read so far.
	ids    []string
	panels []CharacterPanel
	// interval is the wait between two requests, last when the last one was
	// sent.
	interval time.Duration
	last     time.Time
	// delegation lets the reads of the scheduled task use the account.
	delegation string
	images     bool
}

// refreshAccountPanels answers 更新面板 of the user's own role from the
// account: its character list, then each character's details. What the
// event cannot read before the chat task budget, counted from start, runs
// out continues as a chat task that answers in this chat.
func (a *App) refreshAccountPanels(ctx context.Context, event *rayleabot.EventContext, owner panelOwner, start time.Time) error {
	ctx, cancel := a.eventWork(ctx, start)
	defer cancel()
	client := a.accountClient(event)
	config := settings(event)
	refresh := &panelRefresh{uid: owner.UID, choice: owner.Choice, provider: client.Provider, player: ShowcaseProfile{Nickname: owner.Role.Nickname, Level: owner.Role.Level}, interval: config.roleInterval(), images: config.ImageReplies}
	task := a.beginChatTask(event, panelTaskID(a.Game.ID, client.Provider, syncOwner(event), owner.Choice), a.Game.Name+"更新面板", "panel_refresh", time.Hour, refresh)
	if task == nil {
		return event.SendText(a.panelReply("running", nil))
	}
	notice(ctx, event, a.panelReply("account_start", nil))
	listed, err := client.Execute(ctx, owner.Choice, a.Game.ID+".characters", map[string]any{})
	if err != nil {
		a.ChatTasks.end(task)
		return event.SendText(a.panelFailure(err))
	}
	for _, raw := range asList(listed.Data["avatar_list"]) {
		if id := asText(asObject(raw)["id"]); id != "" {
			refresh.ids = append(refresh.ids, id)
		}
	}
	reply, done, err := a.stepChatTask(ctx, event.Actions(), task, start.Add(chatTaskBudget))
	switch {
	case err != nil:
		return event.SendText(a.panelFailure(err))
	case !done:
		return event.Result(map[string]any{"handled": true})
	}
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, reply...)
}

// step reads until stop and, once every character is read, keeps the panels
// and answers with 面板列表 marking them. Keeping and drawing the list has
// the rest of an event: when the reads end at stop, the next trigger does
// it.
func (r *panelRefresh) step(ctx context.Context, a *App, host taskHost, stop time.Time) ([]rayleabot.Segment, bool) {
	if err := r.read(ctx, a, AccountsClient{Caller: host, Provider: r.provider, Game: a.Game.ID}, stop); err != nil {
		return r.fail(a, err), true
	}
	if len(r.ids) > 0 || !a.now().Before(stop) {
		return nil, false
	}
	saved, err := a.Profiles.Keep(r.uid, r.panels, "米游社", &r.player)
	if err != nil {
		return []rayleabot.Segment{rayleabot.Text(friendlyError(err))}, true
	}
	if len(r.panels) == 0 {
		return []rayleabot.Segment{rayleabot.Text(a.panelReply("none", map[string]string{"uid": r.uid, "service": "米游社"}))}, true
	}
	updated := map[string]bool{}
	for _, panel := range r.panels {
		updated[panel.ID] = true
	}
	return viewReply(ctx, host, r.images, a.panelListView(ctx, saved, updated, "米游社")), true
}

// fail is upstream's reply when the refresh fails.
func (r *panelRefresh) fail(a *App, err error) []rayleabot.Segment {
	return []rayleabot.Segment{rayleabot.Text(a.panelFailure(err))}
}

// read asks the remaining characters' details one a request, each the
// interval after the previous request, until the next would start at stop.
// A request open when ctx ends is asked again by the next trigger.
func (r *panelRefresh) read(ctx context.Context, a *App, client AccountsClient, stop time.Time) error {
	for len(r.ids) > 0 {
		at := a.now()
		if next := r.last.Add(r.interval); !r.last.IsZero() && next.After(at) {
			at = next
		}
		if !at.Before(stop) {
			return nil
		}
		if a.sleep(ctx, at.Sub(a.now())) != nil {
			return nil
		}
		r.last = a.now()
		params := map[string]any{"account_ref": r.choice.AccountRef, "role_ref": r.choice.RoleRef, "operation": a.Game.ID + ".character", "input": map[string]any{"id_list": []any{r.ids[0]}}}
		if r.delegation != "" {
			params["delegation_ref"] = r.delegation
		}
		var result QueryResult
		if err := client.call(ctx, "execute", params, &result); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		r.panels = append(r.panels, NormalizePanels(result, a.Catalog)...)
		r.ids = r.ids[1:]
	}
	return nil
}

// handover has the account delegate the remaining reads to the scheduled
// task, whose triggers have no chat user. The delegation lasts a day, the
// shortest the account plugin grants; the role's next refresh renews it.
func (r *panelRefresh) handover(ctx context.Context, a *App, host taskHost, ref string) error {
	var grant struct {
		Delegation struct {
			Ref string `json:"ref"`
		} `json:"delegation"`
	}
	client := AccountsClient{Caller: host, Provider: r.provider, Game: a.Game.ID}
	if err := client.call(ctx, "delegation.create", map[string]any{"account_ref": r.choice.AccountRef, "role_ref": r.choice.RoleRef, "task_id": ref, "operation": a.Game.ID + ".character", "days": 1}, &grant); err != nil {
		return err
	}
	r.delegation = grant.Delegation.Ref
	return nil
}

// panelFailure is upstream's reply when the account's panels cannot be read.
func (a *App) panelFailure(err error) string {
	if text := a.panelReply("account_failed", map[string]string{"error": friendlyError(err)}); text != "" {
		return text
	}
	return friendlyError(err)
}
