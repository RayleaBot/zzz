package app

import (
	"context"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// refreshAccountPanels answers 更新面板 of the user's own role from the
// account. The command's event moves to the background and reads as
// ZZZ-Plugin's getAvatarInfoList does: the character list, then each
// character's details in a request of its own, roleInterval apart. The
// panels are kept once all are read and answered as 面板列表 marking them;
// a failed read ends the refresh without keeping any, as upstream. A UID is
// refreshed once at a time, and free again before the chat is answered.
func (a *App) refreshAccountPanels(ctx context.Context, event *rayleabot.EventContext, owner panelOwner) error {
	key := "panel:" + owner.UID
	if !a.flows.begin(key) {
		return event.SendText(a.panelReply("running", nil))
	}
	answer := a.refreshPanels(ctx, event, owner)
	a.flows.end(key)
	return answer()
}

// refreshPanels reads and keeps the panels in the background and returns
// the answer.
func (a *App) refreshPanels(ctx context.Context, event *rayleabot.EventContext, owner panelOwner) func() error {
	if _, err := event.Detach(ctx, nil); err != nil {
		return reply(event, detachFailure(err).Message)
	}
	notice(ctx, event, a.panelReply("account_start", nil))
	panels, err := a.readAccountPanels(ctx, a.accountClient(event), owner.Choice, settings(event).roleInterval())
	if err != nil {
		return reply(event, a.panelFailure(err))
	}
	saved, err := a.Profiles.Keep(owner.UID, panels, "米游社", &ShowcaseProfile{Nickname: owner.Role.Nickname, Level: owner.Role.Level})
	if err != nil {
		return reply(event, friendlyError(err))
	}
	if len(panels) == 0 {
		return reply(event, a.panelReply("none", map[string]string{"uid": owner.UID, "service": "米游社"}))
	}
	updated := map[string]bool{}
	for _, panel := range panels {
		updated[panel.ID] = true
	}
	view := a.panelListView(ctx, saved, updated, "米游社")
	return func() error { return a.sendView(ctx, event, view) }
}

// readAccountPanels reads the role's character list, then each character's
// details in a request of its own, interval after the previous one.
func (a *App) readAccountPanels(ctx context.Context, client AccountsClient, choice Selection, interval time.Duration) ([]CharacterPanel, error) {
	listed, err := client.Execute(ctx, choice, a.Game.ID+".characters", map[string]any{})
	if err != nil {
		return nil, err
	}
	panels := []CharacterPanel{}
	read := 0
	for _, raw := range asList(listed.Data["avatar_list"]) {
		id := asText(asObject(raw)["id"])
		if id == "" {
			continue
		}
		if read > 0 {
			if err := a.sleep(ctx, interval); err != nil {
				return nil, err
			}
		}
		read++
		result, err := client.Execute(ctx, choice, a.Game.ID+".character", map[string]any{"id_list": []any{id}})
		if err != nil {
			return nil, err
		}
		panels = append(panels, NormalizePanels(result, a.Catalog)...)
	}
	return panels, nil
}

// panelFailure is upstream's reply when the account's panels cannot be read.
func (a *App) panelFailure(err error) string {
	if text := a.panelReply("account_failed", map[string]string{"error": friendlyError(err)}); text != "" {
		return text
	}
	return friendlyError(err)
}
