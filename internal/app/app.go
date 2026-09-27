package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/artwork"
	"github.com/RayleaBot/plugin-zzz/internal/gacha"
	"github.com/RayleaBot/plugin-zzz/internal/pluginmeta"
	"github.com/RayleaBot/plugin-zzz/internal/reference"
)

type Operation struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	Command string `json:"command"`
	Input   string `json:"input"`
}
type Game struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Prefix     string      `json:"prefix"`
	Region     string      `json:"region"`
	Operations []Operation `json:"operations"`
	// Hints are the replies of commands that upstream answers only by naming
	// the commands that query, by command ID; {prefix} is the reply prefix.
	Hints map[string]string `json:"hints"`
	// QueryRanks are the group rankings fed by members' own record queries.
	QueryRanks []QueryRankType `json:"query_ranks"`
	// Panels are the 更新面板 rules and replies.
	Panels PanelSettings `json:"panels"`
	// Artwork lists the upstream image repositories an administrator can
	// download into the data directory.
	Artwork []artwork.Source `json:"artwork"`
	// Pictures are where 图鉴 finds downloaded images.
	Pictures Pictures `json:"pictures"`
	// Calc runs this game's pinned upstream calculation scripts.
	Calc *reference.Engine `json:"-"`
	// Data is this game's fixed reference data.
	Data *GameData `json:"-"`
}

// Assets is the data a game plugin compiles in and hands to the shared library.
type Assets struct {
	Game     []byte // game.json
	Catalog  []byte // catalog.json
	Manifest []byte // info.json
	Calc     reference.Profile
	// Resources holds the banners.
	Resources []byte
	// Images holds the plugin's own image builders by operation name; Panel
	// draws single-character panels.
	Images map[string]ImageBuilder
	Panel  PanelImageBuilder
	// Damage draws 伤害.
	Damage DamageImageBuilder
	Gacha  GachaImageBuilder
	Help   HelpImageBuilder
	// MonthlyStats draws 统计 from the saved months.
	MonthlyStats MonthlyStatsImageBuilder
	// Calendar draws 日历 from the official announcements.
	Calendar CalendarImageBuilder
	// Entry draws reference pages for a catalog entry, as 天赋 and 图鉴.
	Entry EntryImageBuilder
	// QueryRank orders and draws the rankings of game.json's query_ranks.
	QueryRank QueryRankImageBuilder
	// Showcase is the public service 更新面板 reads without an account;
	// PanelList draws 面板列表.
	Showcase  ShowcaseSource
	PanelList PanelListImageBuilder
	// UIDList draws 我的uid; Banners reads the banners online.
	UIDList UIDListImageBuilder
	Banners BannerSource
	// Downloads lists the on-demand pictures 下载全部资源 fetches.
	Downloads ArtworkGroupsBuilder
	// Queries picks the official query, by operation name, for commands that
	// draw on another operation's data, as 最新深渊 runs whichever record
	// opened last.
	Queries map[string]func(now time.Time) string
}
type Settings struct {
	AccountProvider string            `json:"account_provider"`
	ImageReplies    bool              `json:"image_replies"`
	CustomAliases   map[string]string `json:"custom_aliases"`
	// ChallengeRemind and the three after it are ZZZ-Plugin's remind.yaml:
	// the switch of the reminders 开启挑战提醒 creates, and the time and
	// thresholds they use for users who set none.
	ChallengeRemind      bool   `json:"challenge_remind_enabled"`
	ChallengeRemindTime  string `json:"challenge_remind_time"`
	ChallengeAbyssLevel  int    `json:"challenge_abyss_level"`
	ChallengeDeadlyStars int    `json:"challenge_deadly_stars"`
	// GroupRank is ZZZ-Plugin's rank.allow_group: whether the group
	// challenge rankings answer and keep queried records.
	GroupRank bool `json:"group_rank_enabled"`
	// PanelInterval is ZZZ-Plugin's panel.interval: the seconds between two
	// refreshes of a UID's panels.
	PanelInterval int `json:"panel_refresh_interval"`
	// PanelRoleInterval is ZZZ-Plugin's panel.roleInterval: the milliseconds
	// 更新面板 waits between two characters' details.
	PanelRoleInterval int `json:"panel_role_interval"`
	// DeviceURL is ZZZ-Plugin's config.url: the download of the device
	// information tool 绑定设备帮助 links.
	DeviceURL string `json:"device_download_url"`
}
type App struct {
	Manifest      pluginmeta.Manifest
	Artwork       *artwork.Store
	Interactions  *InteractionStore
	GuideSettings *GuideSettings
	Subscriptions *ContentSubscriptions
	pushLists     pushLists
	ContentJobs   ContentJobs
	Content       PublicContentClient
	Monthly       *MonthlyStore
	Game          Game
	Catalog       Catalog
	Gacha         *gacha.Store
	Transfers     gacha.Transfers
	Syncs         gacha.Syncs
	// fileImports are the senders 导入记录 is waiting on for a file.
	fileImports fileImports
	// ChatTasks is chat work that continues past its event, as gacha links
	// whose records are still being fetched; LinkHTTP reads the official
	// signal search (nil uses a default client).
	ChatTasks chatTasks
	LinkHTTP  *http.Client
	// flows are the long flows running now.
	flows        flows
	Showcase     ShowcaseClient
	Profiles     *PanelStore
	Reminders    *ReminderStore
	PanelHistory *PanelHistoryStore
	Groups       *GroupStore
	QueryRanks   *QueryRankStore
	BuildPresets *BuildPresetStore

	// ChallengePrefs are users' own 挑战提醒 thresholds and times.
	ChallengePrefs *ChallengePreferences
	// PanelImages are the custom pictures panels show.
	PanelImages *PanelImages

	commands       commandSet
	images         map[string]ImageBuilder
	queries        map[string]func(now time.Time) string
	panel          PanelImageBuilder
	damage         DamageImageBuilder
	gacha          GachaImageBuilder
	helpImage      HelpImageBuilder
	monthlyStats   MonthlyStatsImageBuilder
	calendarImage  CalendarImageBuilder
	entryPage      EntryImageBuilder
	queryRankImage QueryRankImageBuilder
	showcase       ShowcaseSource
	panelList      PanelListImageBuilder
	uidListImage   UIDListImageBuilder
	atlases        atlasIndexes
	guides         guideCache
	emoticons      newsEmoticons
	aliases        customAliases
	banners        BannerSource
	downloads      ArtworkGroupsBuilder
	usageOnce      sync.Once
	// clock is nil for the wall clock.
	clock clock
}

func New(assets Assets, directory string) (*App, error) {
	var game Game
	if err := json.Unmarshal(assets.Game, &game); err != nil || game.ID == "" {
		return nil, fmt.Errorf("game description is invalid")
	}
	catalog, err := ParseCatalog(assets.Catalog)
	if err != nil {
		return nil, err
	}
	manifest, err := pluginmeta.Read(assets.Manifest)
	if err != nil {
		return nil, err
	}
	if manifest.ID != "raylea."+game.ID {
		return nil, fmt.Errorf("manifest %s does not belong to game %s", manifest.ID, game.ID)
	}
	commands, err := newCommandSet(manifest)
	if err != nil {
		return nil, err
	}
	if game.Calc, err = reference.New(assets.Calc); err != nil {
		return nil, err
	}
	if game.Data, err = parseGameData(assets); err != nil {
		return nil, err
	}
	if directory == "" {
		return nil, fmt.Errorf("plugin data directory is required")
	}
	return &App{commands: commands, images: assets.Images, queries: assets.Queries, panel: assets.Panel, damage: assets.Damage, gacha: assets.Gacha, helpImage: assets.Help, monthlyStats: assets.MonthlyStats, calendarImage: assets.Calendar, entryPage: assets.Entry, queryRankImage: assets.QueryRank, showcase: assets.Showcase, panelList: assets.PanelList, uidListImage: assets.UIDList, banners: assets.Banners, downloads: assets.Downloads, Profiles: &PanelStore{Directory: filepath.Join(directory, "profiles")}, QueryRanks: &QueryRankStore{Directory: filepath.Join(directory, "query-ranks")}, Manifest: manifest, Artwork: &artwork.Store{Root: filepath.Join(directory, "assets"), Sources: game.Artwork}, Interactions: &InteractionStore{Path: filepath.Join(directory, "interactions.json")}, GuideSettings: &GuideSettings{Path: filepath.Join(directory, "guides.json")}, Subscriptions: &ContentSubscriptions{Path: filepath.Join(directory, "content-subscriptions.json")}, Monthly: &MonthlyStore{Directory: filepath.Join(directory, "monthly")}, Game: game, Catalog: catalog, BuildPresets: &BuildPresetStore{Path: buildPresetPath(directory)}, Gacha: &gacha.Store{Directory: filepath.Join(directory, "gacha"), Game: game.ID}, Reminders: reminderStore(directory), ChallengePrefs: challengePreferences(directory), PanelImages: &PanelImages{Directory: filepath.Join(directory, panelImagesDir)}, PanelHistory: &PanelHistoryStore{Directory: filepath.Join(directory, "panels")}, Groups: &GroupStore{Directory: filepath.Join(directory, "groups")}}, nil
}
func settings(event *rayleabot.EventContext) Settings {
	value := Settings{AccountProvider: "raylea.mihoyo-accounts", ImageReplies: true, CustomAliases: map[string]string{}, ChallengeRemind: true, ChallengeRemindTime: "每日20时", ChallengeAbyssLevel: 5, ChallengeDeadlyStars: 6, GroupRank: true, PanelInterval: 60, PanelRoleInterval: 3000, DeviceURL: defaultDeviceURL}
	_ = decodeObject(event.Config, &value)
	return value
}

// Close stops the plugin's background work: artwork downloads and public
// content jobs.
func (a *App) Close() {
	if a.Artwork != nil {
		a.Artwork.Close()
	}
	a.ContentJobs.Close()
}

func decodeObject(value any, target any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

// roleInterval is the wait between two characters' details; ZZZ-Plugin
// waits at least 100 ms whatever the setting.
func (s Settings) roleInterval() time.Duration {
	return time.Duration(max(s.PanelRoleInterval, 100)) * time.Millisecond
}

func (a *App) accountClient(event *rayleabot.EventContext) AccountsClient {
	return AccountsClient{Caller: event.Actions(), Provider: settings(event).AccountProvider, Game: a.Game.ID}
}
func (a *App) operation(name string) (Operation, bool) {
	for _, item := range a.Game.Operations {
		if item.Name == name {
			return item, true
		}
	}
	return Operation{}, false
}

// routedQuery is the official query a command runs, query unless the command
// draws on another operation's query, and the operation its text reply reads
// as.
func (a *App) routedQuery(operation Operation, query string) (string, Operation) {
	if route := a.queries[operation.Name]; route != nil {
		query = route(time.Now())
		if queried, ok := a.operation(query); ok {
			return query, queried
		}
	}
	return query, operation
}

func (a *App) Handle(ctx context.Context, event *rayleabot.EventContext) error {
	// Replies name commands with the first prefix the host gives this plugin;
	// the list is fixed for the process session.
	a.usageOnce.Do(func() {
		if len(event.CommandPrefixes) > 0 {
			a.Game.Prefix = event.CommandPrefixes[0]
		}
	})
	if event.Event.EventType == "scheduler.trigger" {
		// A trigger is dispatched by the task ID the host gives it; one whose
		// task is not stored deletes its job.
		task := event.Event.TaskID()
		switch {
		case strings.HasPrefix(task, "game.content."):
			return a.runContentSubscription(ctx, event)
		case strings.HasPrefix(task, artworkTask):
			return a.runChatTask(ctx, event)
		}
		return a.runReminder(ctx, event)
	}
	if event.Event.EventType == "management.action" {
		if event.Event.SourceProtocol != "management" || event.Event.SourceAdapter != "management.ui" {
			return event.Fail("plugin.game_source_invalid", "管理动作来源无效。")
		}
		action, input := asText(event.Event.Payload["action"]), asObject(event.Event.Payload["payload"])
		if action == "gacha.sync.background" {
			return a.backgroundSync(ctx, event, input)
		}
		result, err := a.Manage(ctx, event, action, input)
		if err != nil {
			return manageFailure(event, err)
		}
		return event.Result(result)
	}
	if event.Event.EventType == "config.changed" {
		a.aliases.observe(settings(event).CustomAliases)
		// The global 挑战提醒 time and thresholds may have changed.
		_ = a.syncChallengePairs(settings(event).challengeGlobals(), nil)
	}
	if event.Event.EventType != "message.private" && event.Event.EventType != "message.group" {
		return event.Result(map[string]any{"handled": false})
	}
	if handled, err := a.gachaLinkMessage(ctx, event); handled {
		return err
	}
	if handled, err := a.gachaFileMessage(ctx, event); handled {
		return err
	}
	command, args, known := a.commands.resolve(event.Event.Command(), event.Event.Args())
	if !known {
		return event.Result(map[string]any{"handled": false})
	}
	prefix := a.Game.Prefix
	if event.Event.EventType == "message.group" {
		if command == "group-settings" {
			return a.groupSettings(event)
		}
		config, readErr := a.Groups.Config(groupScope(event))
		if readErr != nil {
			return event.SendText(friendlyError(readErr))
		}
		applyGroupConfig(event, config)
		if config.Enabled != nil && !*config.Enabled && command != "help" && command != "version" && command != "unsubscribe" && command != "challenge-stop" && command != "challenge-status" && command != "community-stop" && command != "community-progress" && command != "cloud-game-stop" && !(command == "signin-task" && len(args) > 0 && args[0] == "关闭") {
			return event.Result(map[string]any{"handled": false})
		}
	}
	if hint, ok := a.Game.Hints[command]; ok {
		return event.SendText(strings.ReplaceAll(hint, "{prefix}", prefix))
	}
	var view View
	var err error
	switch command {
	case "help", "version":
		return a.helpCommand(ctx, event, command, args)
	case "gacha-export", "gacha-import":
		return a.gachaFileCommand(ctx, event, command, args)
	case "device-bind", "device-unbind", "device-help", "device-default":
		return a.deviceCommand(ctx, event, command)
	case "artwork", "artwork-status":
		return a.artworkCommand(event, command, args)
	case "setting-panel-interval":
		return a.panelIntervalCommand(ctx, event, args)
	case "setting-role-interval":
		return a.roleIntervalCommand(ctx, event, args)
	case "artwork-all":
		return a.artworkAll(ctx, event)
	case "artwork-delete":
		return a.artworkDeleteAll(ctx, event)
	case "original-image":
		return a.originalImage(event)
	case "guides", "guide-help", "guide-default", "guide-forward-count":
		return a.guideCommand(ctx, event, command, args)
	case "subscribe", "unsubscribe", "content-push":
		return a.subscriptionCommand(ctx, event, command, args)
	case "news", "info", "events", "search", "estimate":
		return a.newsCommand(ctx, event, command, args)
	case "live-calendar":
		return a.calendarCommand(ctx, event)
	case "alias-set", "alias-remove", "aliases":
		return a.aliasCommand(ctx, event, command, args)
	case "codes", "redeem":
		return a.assetCommand(ctx, event, command, args)
	case "monthly-history":
		return a.monthlyCommand(ctx, event, command, args)
	case "challenge-remind", "challenge-stop", "challenge-status":
		return a.challengeReminderCommand(ctx, event, command, args)
	case "challenge-enable", "challenge-check", "challenge-threshold", "challenge-time", "challenge-time-reset", "challenge-time-status", "challenge-global-switch", "challenge-global-threshold", "challenge-global-time", "challenge-global-time-status":
		return a.challengePairCommand(ctx, event, command, args)
	case "gacha-background":
		return a.gachaRefresh(ctx, event, args)
	case "community-progress":
		return a.communityProgress(ctx, event)
	case "community-task", "community-stop", "cloud-game-task", "cloud-game-stop":
		return a.accountTaskCommand(ctx, event, command, args)
	case "community-status", "community-sign", "cloud-game-status", "cloud-game-sign":
		return a.communityCommand(ctx, event, command, args)
	case "signin-task":
		return a.signinTaskCommand(ctx, event, args)
	case "signin", "signin-status":
		return a.signinCommand(ctx, event, command, args)
	case "rank":
		return a.rankCommand(ctx, event, args)
	case "query-rank", "query-rank-switch", "query-rank-reset", "group-rank-switch":
		return a.queryRankCommand(ctx, event, command)
	case "panel-refresh", "panel-list":
		return a.panelCommand(ctx, event, command, args)
	case "panel-image-upload", "panel-image-list", "panel-image-remove":
		return a.panelImageCommand(ctx, event, command, args)
	case "banner-history", "banner-current", "banner-all", "banner-version":
		return a.poolCommand(ctx, event, command, args)
	case "build":
		return a.damageCommand(ctx, event, args)
	case "score":
		if len(args) < 1 || len(args) > 2 {
			return event.SendText("使用“" + prefix + "评分 角色 [UID]”。")
		}
		panel, uid, panelErr := a.commandPanel(ctx, event, args)
		if panelErr != nil {
			err = panelErr
			break
		}
		panel, err = a.scorePanel(ctx, panel)
		if err == nil {
			view = PanelView(a.Game, []CharacterPanel{panel}, uid)
		}
	case "talent-wiki":
		cinema := !talentLevels.MatchString(event.Event.Command())
		if cinema && len(args) > 1 {
			// ZZZ-Plugin's cinema rule ends with the word, so text after it is
			// left to other plugins.
			return event.Result(map[string]any{"handled": false})
		}
		name, word := talentWords(event.Event.Command(), args)
		if _, legal := SkillLevels(word); !cinema && !legal {
			// ZZZ-Plugin checks the levels first and leaves a message whose
			// levels no skill has.
			return event.Result(map[string]any{"handled": false})
		}
		entry, ok := a.Catalog.Resolve(name, "character", a.aliasMap(event))
		if !ok && cinema {
			return event.SendText("未找到" + name + "的数据")
		}
		if !ok {
			return event.SendText("暂无" + name + "角色数据")
		}
		view = EntryView(a.Game, entry)
		view.Image = a.entryImage(ctx, command, word, entry)
	case "catalog":
		query := strings.Join(args, " ")
		entries := a.Catalog.Search(query, "", 20, a.aliasMap(event))
		if len(entries) == 1 {
			view = EntryView(a.Game, entries[0])
			view.Image = a.entryImage(ctx, command, event.Event.Command(), entries[0])
		}
		// Without a drawn page, the downloaded 图鉴 libraries answer as Atlas
		// does.
		if view.Image == nil {
			if file, ok := a.atlasPicture(event.Event.Command(), a.aliasMap(event)); ok {
				return a.sendArtwork(event, file)
			}
		}
		if len(entries) == 1 {
			if hint := a.pictureHint(a.Game.Pictures.Atlas); view.Image == nil && hint != "" {
				view.Note = strings.TrimSpace(view.Note + " " + hint)
			}
		} else {
			view = View{Title: a.Game.Name + "图鉴", Subtitle: "资料版本 " + a.Catalog.Version, Rows: []Row{}}
			for _, entry := range entries {
				view.Rows = append(view.Rows, Row{Label: entry.Name, Value: kindLabel(entry.Kind) + " · " + entry.ID})
			}
			if len(entries) == 0 {
				view.Note = "未找到匹配资料，请尝试角色、装备全名或 ID。"
			}
		}
	case "atlas":
		// Atlas answers the words no other command takes, as it does any word
		// after the prefix.
		if file, ok := a.atlasPicture(strings.Join(append([]string{event.Event.Command()}, args...), " "), a.aliasMap(event)); ok {
			return a.sendArtwork(event, file)
		}
		return event.Result(map[string]any{"handled": false})
	case "materials":
		// Atlas's material for role: the character's ascension materials.
		_, known := a.Catalog.Resolve(strings.Join(args, ""), "character", a.aliasMap(event))
		if file, ok := a.atlasPicture(event.Event.Command(), a.aliasMap(event)); ok {
			return a.sendArtwork(event, file)
		}
		if hint := a.pictureHint(a.Game.Pictures.Atlas); known && hint != "" {
			return event.SendText(hint)
		}
		return event.Result(map[string]any{"handled": false})
	case "character":
		if len(args) < 1 || len(args) > 2 {
			return event.Result(map[string]any{"handled": false})
		}
		panel, uid, panelErr := a.commandPanel(ctx, event, args)
		if panelErr != nil {
			err = panelErr
			break
		}
		view = a.fullPanelView(ctx, event, panel, uid)
	case "accounts", "select", "uid-remove":
		return a.uidCommand(ctx, event, command, args)
	case "gacha-link":
		return a.gachaLinkCommand(event)
	case "gacha-link-get":
		return a.gachaLinkGet(ctx, event)
	case "gacha":
		uid := ""
		if len(args) > 0 {
			uid = args[0]
		}
		archive, role, readErr := a.chatArchive(ctx, event, uid)
		if readErr != nil {
			err = readErr
			break
		}
		game := a.bannerGame()
		view = GachaView(game, archive)
		if a.gacha != nil {
			if drawn, ok := a.gacha(a.imageContext(ctx), GachaImage{UID: role.UID, Role: role, Word: event.Event.Command(), Archive: archive}); ok {
				view.Image = &drawn
			}
		}
	default:
		var operation Operation
		matched := false
		for _, item := range a.Game.Operations {
			// Operation names use underscores where command IDs use hyphens.
			if item.Name == a.Game.ID+"."+strings.ReplaceAll(command, "-", "_") {
				operation = item
				matched = true
				break
			}
		}
		if !matched {
			return event.Result(map[string]any{"handled": false})
		}
		input, uid, parseErr := commandInput(operation, strings.TrimSpace(event.Event.Command()), args)
		if parseErr != nil {
			err = parseErr
			break
		}
		if uid != "" && !uidPattern.MatchString(uid) {
			// Upstream's rules end with the command word. What follows it here
			// is a UID; anything else is not this command, as upstream.
			return event.Result(map[string]any{"handled": false})
		}
		listed, listErr := a.accountClient(event).List(ctx, 0)
		if listErr != nil {
			err = listErr
			break
		}
		choice, role, chooseErr := Choose(listed, a.Game.ID, uid)
		if chooseErr != nil {
			err = chooseErr
			break
		}
		// 练度统计 draws the panels kept for the UID, as upstream's
		// proficiency; without them it asks for 更新面板 first.
		if operation.Name == a.Game.ID+".training" {
			if saved, readErr := a.Profiles.Read(role.UID); readErr == nil && len(saved.Panels) == 0 {
				return event.SendText(a.panelReply("training_empty", nil))
			}
		}
		// A routed command's text reply reads as the query that ran; its image
		// is the command's own.
		query, textOperation := a.routedQuery(operation, operation.Name)
		result, queryErr := a.accountClient(event).Execute(ctx, choice, query, input)
		if queryErr != nil {
			err = queryErr
			break
		}
		shown := a.recordQueryRank(event, operation.Name, result)
		if query == a.Game.ID+".monthly" {
			// Upstream keeps every month it reads; a month that cannot be kept
			// still answers.
			_ = a.Monthly.Keep(a.accountClient(event).Provider, choice, result.Data, time.Now())
		}
		view = BusinessView(a.Game, textOperation, result, a.Catalog)
		view.Image = a.featureImage(ctx, a.accountClient(event), choice, operation.Name, event.Event.Command(), input, result, shown)
	}
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return a.sendView(ctx, event, view)
}

// commandInput reads a command's query input and UID from its arguments.
// Upstream writes the period and the month only in the command word, so they
// are read only from the arguments its trigger took from word.
func commandInput(operation Operation, word string, args []string) (map[string]any, string, error) {
	input := map[string]any{}
	uid := ""
	switch operation.Input {
	case "period":
		// Upstream reads the last period for 上期 or 往期 before the command,
		// as in "上期深渊", and the current one otherwise.
		if len(args) > 0 && strings.HasPrefix(word, args[0]) && (args[0] == "上期" || args[0] == "往期") {
			input["schedule_type"] = 2
			args = args[1:]
		}
	case "year_month":
		if len(args) > 0 && strings.HasSuffix(word, args[0]) && strings.ContainsAny(args[0], "年月") {
			// ZZZ-Plugin's 月报2025年3月 and 月报上月; a month it cannot use
			// reads the default month.
			month, valid := monthlyWord(args[0], time.Now())
			if !valid {
				return nil, "", gameError("input_invalid", "获取月报数据失败，请检查日期是否正确")
			}
			if month != 0 {
				input["month"] = month
			}
			args = args[1:]
		}
	}
	if len(args) > 0 {
		uid = args[0]
	}
	return input, uid, nil
}

func (a *App) Manage(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	switch action {
	case "public.query":
		result, err := a.accountClient(event).PublicProfile(ctx, asText(input["uid"]), asText(input["region"]))
		if err != nil {
			return nil, err
		}
		view := BusinessView(a.Game, Operation{Name: a.Game.ID + ".profile", Label: a.Game.Name + "公开资料"}, result, a.Catalog)
		view.Note = "通过已获所有者授权的公共查询池读取官方公开资料，不代表目标 UID 已绑定。"
		return map[string]any{"view": view}, nil
	case "signin.status", "signin.rewards", "signin.run":
		return a.signinAction(ctx, a.accountClient(event), action, input)
	case "content.start", "content.poll", "content.cancel":
		return a.contentAction(action, input)
	case "help.query":
		out, _, err := a.help(event, asText(input["query"]))
		return out, err
	case "aliases.validate":
		return a.aliasAction(input)
	case "guides.schema", "guides.settings", "guides.configure":
		return a.GuideSettings.Manage(action, input)
	case "content.subscription.list", "content.subscription.remove":
		return a.subscriptionManage(ctx, event, action, input)
	case "codes.query":
		return a.Content.codes(ctx)
	case "redeem.run":
		return a.redeem(ctx, a.accountClient(event), input)
	case "monthly.list", "monthly.fetch", "monthly.get", "monthly.remove":
		return a.monthlyAction(ctx, a.accountClient(event), action, input)
	case "challenge.reminder.create", "challenge.reminder.list", "challenge.reminder.remove", "challenge.reminder.check":
		return a.challengeReminder(ctx, event, action, input)
	case "challenge.schema":
		return map[string]any{"kinds": challengeKinds()}, nil
	case "groups.list", "groups.clear", "groups.set":
		return a.manageGroups(action, input)
	case "calendar.query":
		return a.calendarQuery(input)
	case "banners.query":
		return a.bannerQuery(input)
	case "build.prepare", "build.compare":
		return a.buildAction(ctx, a.accountClient(event), action, input)
	case "build.conditions", "build.presets.list", "build.presets.save", "build.presets.remove":
		return a.buildPresetAction(action, input)
	case "status":
		return map[string]any{"game": a.Game, "catalog_version": a.Catalog.Version, "catalog_entries": len(a.Catalog.Entries), "settings": settings(event), "bots": event.Bots}, nil
	case "accounts.list":
		result, err := a.accountClient(event).List(ctx, number(input["page"]))
		return map[string]any{"accounts": result}, err
	case "accounts.select":
		choice := Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}
		err := a.accountClient(event).Select(ctx, choice)
		return map[string]any{"selected": choice}, err
	case "query":
		operation, ok := a.operation(asText(input["operation"]))
		if !ok {
			return nil, gameError("operation_denied", "查询操作不存在。")
		}
		parameters := asObject(input["input"])
		// As ZZZ-Plugin, an agent's details are asked one agent a request.
		if strings.HasSuffix(operation.Name, ".character") && len(asList(parameters["id_list"])) != 1 {
			return nil, gameError("input_invalid", "请选择一个角色。")
		}
		query, textOperation := a.routedQuery(operation, operation.Name)
		result, err := a.accountClient(event).Execute(ctx, Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}, query, parameters)
		if err != nil {
			return nil, err
		}
		output := map[string]any{"view": BusinessView(a.Game, textOperation, result, a.Catalog), "result": result}
		if strings.HasSuffix(operation.Name, ".character") {
			output["panels"] = NormalizePanels(result, a.Catalog)
			delete(output, "result")
		}
		return output, nil
	case "panel.score":
		panel, err := a.queryScoredPanel(ctx, a.accountClient(event), Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}, asText(input["character_id"]))
		if err != nil {
			return nil, err
		}
		return map[string]any{"panels": []CharacterPanel{panel}, "view": PanelView(a.Game, []CharacterPanel{panel}, "")}, nil
	case "showcase":
		uid := asText(input["uid"])
		saved, panels, err := a.refreshShowcase(ctx, event, uid)
		if err != nil {
			return nil, gameError("showcase_failed", a.showcaseReply(err, uid))
		}
		view := View{Title: a.Game.Name + "公开展柜", Subtitle: uid, Rows: []Row{}, Note: "已保存到此 UID 的面板，聊天中可直接查看这些角色的面板、评分与伤害。"}
		for _, panel := range panels {
			view.Rows = append(view.Rows, Row{Label: panel.Name, Value: "等级 " + strconv.Itoa(panel.Level) + " · " + strconv.Itoa(panel.Rank)})
		}
		if saved.Nickname != "" {
			view.Subtitle = saved.Nickname + " · " + uid
		}
		return map[string]any{"view": view}, nil
	case "catalog.search":
		entries := a.Catalog.Search(asText(input["query"]), asText(input["kind"]), 50, a.aliasMap(event))
		return map[string]any{"entries": entries, "version": a.Catalog.Version}, nil
	case "catalog.get":
		entry, ok := a.Catalog.Get(asText(input["id"]))
		if !ok {
			return nil, gameError("entry_missing", "未找到这份资料。")
		}
		return map[string]any{"entry": entry, "view": EntryView(a.Game, entry)}, nil
	default:
		if strings.HasPrefix(action, "artwork.") {
			return a.artworkAction(action, input)
		}
		if strings.HasPrefix(action, "community.task.") || strings.HasPrefix(action, "cloudgame.task.") {
			return a.signinTask(ctx, event, action, input)
		}
		if strings.HasPrefix(action, "community.") || strings.HasPrefix(action, "cloudgame.") {
			return a.communityAction(ctx, a.accountClient(event), action, input)
		}
		if strings.HasPrefix(action, "signin.task.") || strings.HasPrefix(action, "monthly.task.") {
			return a.signinTask(ctx, event, action, input)
		}
		if strings.HasPrefix(action, "panel.history.") {
			return a.panelHistory(ctx, a.accountClient(event), action, input)
		}
		if strings.HasPrefix(action, "reminder.") {
			return a.manageReminder(ctx, event, action, input)
		}
		if strings.HasPrefix(action, "gacha.sync.") {
			return a.manageSync(ctx, event, action, input)
		}
		if strings.HasPrefix(action, "gacha.") {
			return a.manageGacha(action, input)
		}
		return nil, gameError("operation_denied", "操作不存在。")
	}
}

func (a *App) manageGacha(action string, input map[string]any) (map[string]any, error) {
	uid, region, ref := asText(input["uid"]), asText(input["region"]), asText(input["ref"])
	switch action {
	case "gacha.list":
		items, err := a.Gacha.List()
		return map[string]any{"items": items}, err
	case "gacha.history":
		var filter gacha.HistoryFilter
		if decodeObject(input, &filter) != nil || filter.Validate() != nil {
			return nil, gameError("input_invalid", "抽卡筛选条件无效。")
		}
		archive, err := a.Gacha.Read(uid, region)
		if err != nil {
			return nil, gameError("archive_missing", "未找到此抽卡档案。")
		}
		history, err := gacha.Browse(archive, filter)
		if errors.Is(err, gacha.ErrConflict) {
			return nil, gameError("archive_changed", "档案已更新，请从第一页重新查询。")
		}
		if err != nil {
			return nil, gameError("input_invalid", "抽卡记录或筛选条件无效。")
		}
		return map[string]any{"history": history}, nil
	case "gacha.summary":
		archive, err := a.Gacha.Read(uid, region)
		if err != nil {
			return nil, gameError("archive_missing", "未找到此抽卡档案。")
		}
		summary := gacha.Summarize(archive)
		for i := range summary {
			if len(summary[i].Rare) > 100 {
				summary[i].Rare = summary[i].Rare[len(summary[i].Rare)-100:]
			}
		}
		return map[string]any{"summary": summary, "view": GachaView(a.bannerGame(), archive)}, nil
	case "gacha.versions":
		archive, err := a.Gacha.Read(uid, region)
		if err != nil {
			return nil, gameError("archive_missing", "未找到此抽卡档案。")
		}
		return versionDraws(a.bannerGame(), archive), nil
	case "gacha.remove":
		err := a.Gacha.Remove(uid, region)
		return map[string]any{"removed": err == nil}, err
	case "gacha.import.start":
		var archive gacha.Archive
		if decodeObject(input, &archive) != nil {
			return nil, gameError("input_invalid", "抽卡档案信息无效。")
		}
		archive.Records = nil
		if gacha.Validate(archive) != nil {
			return nil, gameError("input_invalid", "请选择有效 UID、区服与时区。")
		}
		transfer, err := a.Transfers.Start(archive, false)
		return map[string]any{"transfer": transfer}, err
	case "gacha.import.append":
		var payload struct {
			Offset  int            `json:"offset"`
			Records []gacha.Record `json:"records"`
		}
		if decodeObject(input, &payload) != nil {
			return nil, gameError("input_invalid", "记录批次格式无效。")
		}
		for i := range payload.Records {
			record := &payload.Records[i]
			if entry, ok := a.Catalog.Get(record.ItemID); ok {
				if record.Name == "" {
					record.Name = entry.Name
				}
				if record.Rank == "" && entry.Rarity > 0 {
					record.Rank = strconv.Itoa(entry.Rarity)
				}
			}
		}
		accepted, err := a.Transfers.Append(ref, payload.Offset, payload.Records)
		return map[string]any{"accepted": accepted}, err
	case "gacha.import.finish":
		result, err := a.Transfers.Finish(ref, a.Gacha)
		if err != nil {
			return nil, gameError("import_failed", "记录存在格式或内容冲突，原档案已保留。")
		}
		return map[string]any{"added": result.Added, "total": result.Total}, nil
	case "gacha.import.cancel", "gacha.export.close":
		a.Transfers.Close(ref)
		return map[string]any{"closed": true}, nil
	case "gacha.export.start":
		archive, err := a.Gacha.Read(uid, region)
		if err != nil {
			return nil, gameError("archive_missing", "未找到此抽卡档案。")
		}
		transfer, err := a.Transfers.Start(archive, true)
		return map[string]any{"transfer": transfer}, err
	case "gacha.export.read":
		records, more, err := a.Transfers.Read(ref, number(input["offset"]), 500)
		return map[string]any{"records": records, "more": more}, err
	}
	return nil, gameError("operation_denied", "抽卡操作不存在。")
}

// manageFailure ends a management action with its failure.
func manageFailure(event *rayleabot.EventContext, err error) error {
	failure := PublicError(err)
	return event.FailDetails(failure.Code, failure.Message, failure.Details)
}

func friendlyError(err error) string {
	failure := PublicError(err)
	switch failure.Code {
	case "plugin.service_unavailable":
		return "米游社账号插件未运行，请先启用并扫码登录。"
	case "plugin.vault_locked":
		return "账号库已锁定，请联系机器人管理员解锁。"
	case "plugin.account_caller_denied":
		return "此游戏未获准使用米游社账号，请联系机器人管理员授权。"
	}
	return failure.Message
}

// sendView replies with the view's own template when it has one, then with
// the generic summary card, and with text when image replies are off or both
// renders fail.
func (a *App) sendView(ctx context.Context, event *rayleabot.EventContext, view View) error {
	if image, ok := renderView(ctx, event.Actions(), settings(event).ImageReplies, view); ok {
		return event.Send(event.Event.Target.Type, event.Event.Target.ID, image)
	}
	return event.SendText(view.Text())
}

type imageRenderer interface {
	RenderImage(context.Context, rayleabot.RenderImageRequest) (rayleabot.ActionResult, error)
}

// renderView draws a view with its own template when it has one, then as the
// generic summary card; ok is false when image replies are off or both
// renders fail.
func renderView(ctx context.Context, host imageRenderer, images bool, view View) (rayleabot.Segment, bool) {
	if !images {
		return rayleabot.Segment{}, false
	}
	text := view.Text()
	requests := []rayleabot.RenderImageRequest{}
	if view.Image != nil {
		requests = append(requests, rayleabot.RenderImageRequest{Template: view.Image.Template, Output: "png", FallbackText: text, Data: view.Image.Data, Resources: view.Image.Resources})
	}
	data := map[string]any{}
	_ = decodeObject(view, &data)
	requests = append(requests, rayleabot.RenderImageRequest{Template: "summary", Output: "png", FallbackText: text, Data: data})
	for _, request := range requests {
		result, err := host.RenderImage(ctx, request)
		if err == nil {
			if path := asText(result["image_path"]); path != "" {
				return rayleabot.Image(path), true
			}
		}
	}
	return rayleabot.Segment{}, false
}
