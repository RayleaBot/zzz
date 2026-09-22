package app

import (
	"context"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/localdata"
)

// Panels are kept per UID as the upstream plugins keep a player's data:
// 更新面板 merges what the showcase service or the account returned, a
// character read from the account is kept when it is viewed, and 面板,
// 评分 and 伤害 read the kept panel first.

// PanelSettings are a game's 更新面板 rules and upstream's wording for its
// replies.
type PanelSettings struct {
	// Cooldown is the wait in seconds between two refreshes of a UID.
	Cooldown int `json:"cooldown"`
	// Replies are upstream's replies by key: failed, unreachable, empty,
	// none, slow, account_start, account_failed, cooldown, list_empty and
	// missing. {prefix}, {uid}, {name}, {service}, {status}, {seconds} and
	// {error} are filled in.
	Replies map[string]string `json:"replies"`
}

// SavedPanel is a character panel kept for a UID, with the official entry it
// was read from, which some images draw on.
type SavedPanel struct {
	Panel       CharacterPanel `json:"panel"`
	Official    map[string]any `json:"official,omitempty"`
	Service     string         `json:"service"`
	UpdatedAtMS int64          `json:"updated_at_ms"`
}

// SavedProfiles are the panels kept for one UID and the last 更新面板.
type SavedProfiles struct {
	UID           string                `json:"uid"`
	Nickname      string                `json:"nickname,omitempty"`
	Level         int                   `json:"level,omitempty"`
	RefreshedAtMS int64                 `json:"refreshed_at_ms,omitempty"`
	Service       string                `json:"service,omitempty"`
	Panels        map[string]SavedPanel `json:"panels"`
}

// Sorted lists the kept panels the way upstream's 面板列表 orders them:
// updated characters first, then by rarity, level and ID, all descending.
func (p SavedProfiles) Sorted(catalog Catalog, updated map[string]bool) []SavedPanel {
	list := slices.Collect(maps.Values(p.Panels))
	key := func(panel SavedPanel) []int {
		entry, _ := catalog.Get(panel.Panel.ID)
		id, _ := strconv.Atoi(panel.Panel.ID)
		isNew := 0
		if updated[panel.Panel.ID] {
			isNew = 1
		}
		return []int{isNew, entry.Rarity, panel.Panel.Level, id}
	}
	slices.SortFunc(list, func(x, y SavedPanel) int { return slices.Compare(key(y), key(x)) })
	return list
}

type PanelStore struct {
	mu        sync.Mutex
	Directory string
}

func (s *PanelStore) file(uid string) (string, error) {
	if !uidPattern.MatchString(uid) {
		return "", gameError("input_invalid", "UID 格式不正确。")
	}
	return filepath.Join(s.Directory, uid+".json"), nil
}

func (s *PanelStore) read(uid string) (SavedProfiles, error) {
	value := SavedProfiles{UID: uid, Panels: map[string]SavedPanel{}}
	file, err := s.file(uid)
	if err == nil {
		err = localdata.Read(file, &value)
	}
	if value.Panels == nil {
		value.Panels = map[string]SavedPanel{}
	}
	return value, err
}

func (s *PanelStore) Read(uid string) (SavedProfiles, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read(uid)
}

// Keep merges panels into a UID's kept panels, replacing each character's
// older one; a player marks a 更新面板 and names the player.
func (s *PanelStore) Keep(uid string, panels []CharacterPanel, service string, player *ShowcaseProfile) (SavedProfiles, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.read(uid)
	if err != nil {
		return value, err
	}
	now := time.Now().UnixMilli()
	for _, panel := range panels {
		value.Panels[panel.ID] = SavedPanel{Panel: panel, Official: panel.Official, Service: service, UpdatedAtMS: now}
	}
	if player != nil {
		value.RefreshedAtMS, value.Service = now, service
		if player.Nickname != "" {
			value.Nickname, value.Level = player.Nickname, player.Level
		}
	}
	file, _ := s.file(uid)
	return value, localdata.Write(file, value)
}

// Delete removes a UID's kept panels.
func (s *PanelStore) Delete(uid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, err := s.file(uid)
	if err != nil {
		return err
	}
	if err = os.Remove(file); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// panel returns the kept panel with its official entry.
func (s SavedPanel) panel() CharacterPanel {
	panel := s.Panel
	panel.Official, panel.UpdatedAtMS = s.Official, s.UpdatedAtMS
	return panel
}

// PanelListImage is 面板列表: a UID's kept panels, and after 更新面板 the
// characters it updated.
type PanelListImage struct {
	UID      string
	Profiles SavedProfiles
	Panels   []SavedPanel
	Updated  map[string]bool
	// Service is the service of this refresh, or of the last one.
	Service string
}

type PanelListImageBuilder func(ImageContext, PanelListImage) (Image, bool)

func (a *App) panelReply(key string, values map[string]string) string {
	text := a.Game.Panels.Replies[key]
	if text == "" {
		return ""
	}
	pairs := []string{"{prefix}", a.Game.Prefix}
	for name, value := range values {
		pairs = append(pairs, "{"+name+"}", value)
	}
	return strings.NewReplacer(pairs...).Replace(text)
}

// notice sends a message to the conversation while a command is still
// working.
func notice(ctx context.Context, event *rayleabot.EventContext, text string) {
	if text == "" {
		return
	}
	_, _ = event.Actions().MessageSend(ctx, rayleabot.MessageSendRequest{SourceProtocol: event.Event.SourceProtocol, SourceAdapter: event.Event.SourceAdapter, TargetType: event.Event.Target.Type, TargetID: event.Event.Target.ID,
		Message: rayleabot.MessageOut{Segments: []rayleabot.Segment{rayleabot.Text(text)}}})
}

// panelOwner is the UID a panel command is about: the one it names, or the
// UID in use, bound with or without an account. Owned marks a UID of the
// user's account, whose official data can be read.
type panelOwner struct {
	UID    string
	Choice Selection
	Role   Role
	Owned  bool
}

func (a *App) panelOwner(ctx context.Context, event *rayleabot.EventContext, uid string) (panelOwner, error) {
	listed, err := a.accountClient(event).List(ctx, 0)
	if err != nil && uid == "" {
		return panelOwner{}, err
	}
	if uid == "" {
		// The UID in use may be one bound without an account.
		uid = listed.UIDBindings[a.Game.ID].Current
	}
	choice, role, err := Choose(listed, a.Game.ID, uid)
	if err != nil {
		if uid == "" {
			return panelOwner{}, err
		}
		return panelOwner{UID: uid}, nil
	}
	return panelOwner{UID: role.UID, Choice: choice, Role: role, Owned: true}, nil
}

// showcaseReply turns a showcase error into upstream's reply.
func (a *App) showcaseReply(err error, uid string) string {
	values := map[string]string{"uid": uid, "service": a.showcase.Name}
	var failure *ShowcaseFailure
	switch {
	case errors.Is(err, ErrShowcaseEmpty):
		if text := a.panelReply("empty", values); text != "" {
			return text
		}
	case errors.As(err, &failure):
		values["status"] = strconv.Itoa(failure.Status)
		if failure.Status == 0 {
			if text := a.panelReply("unreachable", values); text != "" {
				return text
			}
		}
		if text := a.panelReply("failed", values); text != "" {
			return text
		}
	}
	return friendlyError(err)
}

// refreshShowcase reads the UID's showcase into its kept panels, telling the
// user when the service is slow as miao does.
func (a *App) refreshShowcase(ctx context.Context, event *rayleabot.EventContext, uid string) (SavedProfiles, []CharacterPanel, error) {
	slow := time.AfterFunc(2*time.Second, func() { notice(ctx, event, a.panelReply("slow", map[string]string{"uid": uid})) })
	profile, err := a.Showcase.Fetch(ctx, a.showcase, a.Game, a.Catalog, uid)
	slow.Stop()
	if err != nil {
		return SavedProfiles{}, nil, err
	}
	saved, err := a.Profiles.Keep(uid, profile.Panels, a.showcase.Name, &profile)
	return saved, profile.Panels, err
}

// accountPanels reads every character of the user's own UID from the
// account's official data, fifty characters a request.
func (a *App) accountPanels(ctx context.Context, client AccountsClient, choice Selection) ([]CharacterPanel, error) {
	listed, err := client.Execute(ctx, choice, a.Game.ID+".characters", map[string]any{})
	if err != nil {
		return nil, err
	}
	ids := []any{}
	for _, field := range []string{"list", "avatar_list", "avatars"} {
		for _, raw := range asList(listed.Data[field]) {
			item := asObject(raw)
			if base := asObject(item["base"]); base != nil {
				item = base
			}
			if id := asText(item["id"]); id != "" {
				ids = append(ids, id)
			}
		}
	}
	key := "id_list"
	panels := []CharacterPanel{}
	for batch := range slices.Chunk(ids, 50) {
		result, err := client.Execute(ctx, choice, a.Game.ID+".character", map[string]any{key: batch})
		if err != nil {
			return nil, err
		}
		panels = append(panels, NormalizePanels(result, a.Catalog)...)
	}
	return panels, nil
}

// characterPanel is the panel 面板, 评分 and 伤害 read: the kept one, else
// the account's official one when the UID is the user's own.
func (a *App) characterPanel(ctx context.Context, event *rayleabot.EventContext, owner panelOwner, id string) (CharacterPanel, error) {
	saved, err := a.Profiles.Read(owner.UID)
	if err != nil {
		return CharacterPanel{}, err
	}
	if kept, ok := saved.Panels[id]; ok {
		return kept.panel(), nil
	}
	if owner.Owned {
		panel, err := a.queryCharacterPanel(ctx, a.accountClient(event), owner.Choice, id)
		if err == nil {
			_, _ = a.Profiles.Keep(owner.UID, []CharacterPanel{panel}, "米游社", nil)
			return panel, nil
		}
	}
	name := id
	if entry, ok := a.Catalog.Get(id); ok {
		name = entry.Name
	}
	return CharacterPanel{}, gameError("panel_missing", a.panelReply("missing", map[string]string{"uid": owner.UID, "name": name}))
}

// commandPanel reads the panel a 面板, 评分 or 伤害 command names by its
// character and optional UID.
func (a *App) commandPanel(ctx context.Context, event *rayleabot.EventContext, args []string) (CharacterPanel, string, error) {
	input, uid, err := a.commandInput(Operation{Input: "characters"}, args, a.aliasMap(event))
	if err != nil {
		return CharacterPanel{}, "", err
	}
	owner, err := a.panelOwner(ctx, event, uid)
	if err != nil {
		return CharacterPanel{}, "", err
	}
	panel, err := a.characterPanel(ctx, event, owner, asText(input["character_ids"].([]any)[0]))
	return panel, owner.UID, err
}

// panelCommand answers 更新面板 and 面板列表. 更新面板 reads a user's own UID
// from the account, as ZZZ-Plugin does unless the word names the showcase
// (展柜), and other UIDs from the showcase.
func (a *App) panelCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	uid := ""
	if len(args) > 0 {
		uid = args[0]
	}
	owner, err := a.panelOwner(ctx, event, uid)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if command == "panel-list" {
		saved, err := a.Profiles.Read(owner.UID)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		if len(saved.Panels) == 0 {
			return event.SendText(a.panelReply("list_empty", map[string]string{"uid": owner.UID}))
		}
		return a.sendPanelList(ctx, event, saved, nil, saved.Service)
	}
	account := owner.Owned && !strings.Contains(event.Event.Command(), "展柜")
	saved, err := a.Profiles.Read(owner.UID)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if wait := time.Duration(a.Game.Panels.Cooldown)*time.Second - time.Since(time.UnixMilli(saved.RefreshedAtMS)); a.Game.Panels.Cooldown > 0 && wait > 0 {
		return event.SendText(a.panelReply("cooldown", map[string]string{"seconds": strconv.Itoa(a.Game.Panels.Cooldown)}))
	}
	var panels []CharacterPanel
	service := a.showcase.Name
	if account {
		service = "米游社"
		notice(ctx, event, a.panelReply("account_start", nil))
		panels, err = a.accountPanels(ctx, a.accountClient(event), owner.Choice)
		if err != nil {
			if text := a.panelReply("account_failed", map[string]string{"error": friendlyError(err)}); text != "" {
				return event.SendText(text)
			}
			return event.SendText(friendlyError(err))
		}
		saved, err = a.Profiles.Keep(owner.UID, panels, service, &ShowcaseProfile{Nickname: owner.Role.Nickname, Level: owner.Role.Level})
	} else {
		saved, panels, err = a.refreshShowcase(ctx, event, owner.UID)
		if err != nil {
			return event.SendText(a.showcaseReply(err, owner.UID))
		}
	}
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if len(panels) == 0 {
		return event.SendText(a.panelReply("none", map[string]string{"uid": owner.UID, "service": service}))
	}
	updated := map[string]bool{}
	for _, panel := range panels {
		updated[panel.ID] = true
	}
	return a.sendPanelList(ctx, event, saved, updated, service)
}

// sendPanelList answers with 面板列表, marking the characters a refresh
// updated.
func (a *App) sendPanelList(ctx context.Context, event *rayleabot.EventContext, saved SavedProfiles, updated map[string]bool, service string) error {
	list := saved.Sorted(a.Catalog, updated)
	view := View{Title: a.Game.Name + "面板列表", Subtitle: "UID " + saved.UID, Rows: []Row{}, Note: "当前更新服务：" + service}
	if saved.RefreshedAtMS > 0 && updated == nil {
		view.Note = "更新时间：" + time.UnixMilli(saved.RefreshedAtMS).In(time.FixedZone("UTC+8", 8*3600)).Format("01-02 15:04") + "\n" + view.Note
	}
	for _, item := range list {
		value := "等级 " + strconv.Itoa(item.Panel.Level) + " · " + strconv.Itoa(item.Panel.Rank)
		if updated[item.Panel.ID] {
			value += " · 本次更新"
		}
		view.Rows = append(view.Rows, Row{Label: item.Panel.Name, Value: value})
	}
	if a.panelList != nil {
		image := PanelListImage{UID: saved.UID, Profiles: saved, Panels: list, Updated: updated, Service: service}
		if drawn, ok := a.panelList(a.imageContext(ctx), image); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
}
