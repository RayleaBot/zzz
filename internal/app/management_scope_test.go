package app

import (
	"fmt"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/zzz/internal/gacha"
	"github.com/RayleaBot/zzz/internal/localdata"
)

func managementEvent(protocol, id string) *rayleabot.EventContext {
	return &rayleabot.EventContext{Event: rayleabot.Event{EventType: "management.action", SourceProtocol: "management", SourceAdapter: "management.ui", Payload: map[string]any{"page_id": "overview", "bot": map[string]any{"source_protocol": protocol, "bot_id": id}}}}
}

func TestAccountManagementRequiresHostAccount(t *testing.T) {
	a := &App{}
	for _, action := range []string{"accounts.list", "accounts.select", "query", "groups.list", "groups.set", "groups.clear", "reminder.list", "reminder.create", "reminder.remove", "signin.status", "signin.run", "signin.task.list", "monthly.list", "monthly.task.list", "challenge.reminder.list", "challenge.reminder.check", "panel.history.list", "panel.score", "build.prepare", "community.status", "community.task.list", "cloudgame.task.list", "redeem.run", "content.subscription.list", "content.subscription.remove", "gacha.sync.start", "gacha.sync.step", "gacha.sync.cancel", "gacha.task.list", "gacha.task.run", "gacha.task.cancel", "history.assign"} {
		t.Run(action, func(t *testing.T) {
			event := managementEvent("onebot11", "bot")
			delete(event.Event.Payload, "bot")
			_, err := a.Manage(t.Context(), event, action, map[string]any{"bot": map[string]any{"source_protocol": "onebot11", "bot_id": "bot"}})
			if err == nil || PublicError(err).Code != "platform.invalid_request" {
				t.Fatalf("accepted missing account: %v", err)
			}
		})
	}
}

func TestAccountManagementRejectsConflictingInput(t *testing.T) {
	for _, field := range []string{"owner", "subject", "scope", "bot"} {
		for _, conflict := range []map[string]any{{"source_protocol": "qqofficial"}, {"bot_id": "other"}} {
			_, err := (&App{}).Manage(t.Context(), managementEvent("onebot11", "bot"), "groups.set", map[string]any{field: conflict})
			if err == nil || PublicError(err).Code != "plugin.game_account_mismatch" {
				t.Fatalf("%s: %v", field, err)
			}
		}
	}
}

func TestAccountGroupPaginationAndConcurrentWrites(t *testing.T) {
	a := pluginApp(t)
	for _, bot := range []string{"a", "b"} {
		for i := range 53 {
			scope := GroupScope{Protocol: "onebot11", BotID: bot, GroupID: fmt.Sprint(i)}
			if err := a.Groups.Update(scope, func(g *GroupData) error { return nil }); err != nil {
				t.Fatal(err)
			}
		}
	}
	for page, want := range []int{50, 3} {
		out, err := a.Manage(t.Context(), managementEvent("onebot11", "a"), "groups.list", map[string]any{"page": page})
		if err != nil {
			t.Fatal(err)
		}
		items := out["items"].([]map[string]any)
		if len(items) != want {
			t.Fatalf("page %d got %d", page, len(items))
		}
		for _, item := range items {
			if item["scope"].(GroupScope).BotID != "a" {
				t.Fatal("cross-account group")
			}
		}
	}
	var wg sync.WaitGroup
	for _, bot := range []string{"a", "b"} {
		wg.Go(func() {
			event := managementEvent("onebot11", bot)
			event.Config = map[string]any{"image_replies": true}
			_, err := a.Manage(t.Context(), event, "groups.set", map[string]any{"scope": map[string]any{"group_id": "new"}, "config": map[string]any{"enabled": false}})
			if err != nil || event.Config["image_replies"] != true {
				t.Errorf("write %s: %v", bot, err)
			}
		})
	}
	wg.Wait()
	for _, bot := range []string{"a", "b"} {
		group, err := a.Groups.Read(GroupScope{Protocol: "onebot11", BotID: bot, GroupID: "new"})
		if err != nil || group.Config.Enabled == nil || *group.Config.Enabled {
			t.Fatalf("lost %s: %+v %v", bot, group, err)
		}
	}
	_, err := a.Manage(t.Context(), managementEvent("onebot11", "a"), "groups.clear", map[string]any{"scope": map[string]any{"source_protocol": "onebot11", "bot_id": "b", "group_id": "new"}, "revision": 1})
	if err == nil {
		t.Fatal("cleared another bot's group")
	}
}

func TestAccountTaskListsAndRecordChecks(t *testing.T) {
	a := pluginApp(t)
	owners := []Subject{{SourceProtocol: "onebot11", BotID: "a", ActorID: "u"}, {SourceProtocol: "onebot11", BotID: "b", ActorID: "u"}, {SourceProtocol: "qqofficial", BotID: "a", ActorID: "u"}, {SourceProtocol: "onebot11", ActorID: "u"}}
	for n, owner := range owners {
		for _, kind := range []string{"", "signin", "monthly", "challenge", "community", "cloudgame"} {
			task := Reminder{Ref: fmt.Sprintf("%s-%d", kind, n), Owner: owner, Kind: kind}
			if err := a.Reminders.edit(task.Ref, func(items *[]Reminder, _ int) error { *items = append(*items, task); return nil }); err != nil {
				t.Fatal(err)
			}
		}
		if err := a.DailySyncs.edit("", func(items *[]DailySync, _ int) error {
			*items = append(*items, DailySync{Ref: fmt.Sprint(n), Owner: owner})
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if err := a.Subscriptions.edit("", func(items *[]ContentSubscription, _ int) error {
			*items = append(*items, ContentSubscription{Ref: fmt.Sprint(n), Owner: owner})
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		a.BackgroundSyncs.items = append(a.BackgroundSyncs.items, BackgroundSync{Ref: fmt.Sprint(n), Owner: owner, State: "running"})
	}
	event := managementEvent("onebot11", "a")
	for _, action := range []string{"reminder.list", "signin.task.list", "monthly.task.list", "challenge.reminder.list", "community.task.list", "cloudgame.task.list"} {
		out, err := a.Manage(t.Context(), event, action, nil)
		if err != nil {
			t.Fatal(err)
		}
		items := out["items"].([]Reminder)
		if len(items) != 1 || items[0].Owner != owners[0] {
			t.Fatalf("%s: %+v", action, items)
		}
	}
	out, err := a.Manage(t.Context(), event, "gacha.task.list", nil)
	if err != nil || len(out["items"].([]BackgroundSync)) != 1 || len(out["daily"].([]DailySync)) != 1 {
		t.Fatal(out, err)
	}
	out, err = a.Manage(t.Context(), event, "content.subscription.list", nil)
	if err != nil || len(out["items"].([]ContentSubscription)) != 1 {
		t.Fatal(out, err)
	}
	for action, ref := range map[string]string{"reminder.remove": "-1", "signin.task.remove": "signin-1", "monthly.task.remove": "monthly-1", "challenge.reminder.remove": "challenge-1", "community.task.remove": "community-1", "cloudgame.task.remove": "cloudgame-1", "gacha.task.remove": "1", "gacha.task.run": "1", "gacha.task.cancel": "1", "content.subscription.remove": "1"} {
		_, err := a.Manage(t.Context(), event, action, map[string]any{"ref": ref, "confirm": true})
		if err == nil || PublicError(err).Code != "plugin.game_account_mismatch" {
			t.Fatalf("%s accepted %s: %v", action, ref, err)
		}
	}
	if items, _ := a.Reminders.List(); len(items) != 24 {
		t.Fatal("removed another bot's reminder")
	}
	if items, _ := a.DailySyncs.List(); len(items) != 4 {
		t.Fatal("removed another bot's daily sync")
	}
	if items, _ := a.Subscriptions.List(); len(items) != 4 {
		t.Fatal("removed another bot's subscription")
	}
	if a.BackgroundSyncs.list()[1].State != "running" {
		t.Fatal("canceled another bot's background sync")
	}
}

func TestAccountCannotUseAnotherPagesSyncReference(t *testing.T) {
	a := pluginApp(t)
	info, err := a.Syncs.Start(a.Gacha, gacha.SyncChoice{AccountRef: "account", RoleRef: "role", SourceProtocol: "onebot11", BotID: "b"}, "10000001", "prod_gf_cn", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"gacha.sync.step", "gacha.sync.cancel"} {
		_, err := a.Manage(t.Context(), managementEvent("onebot11", "a"), action, map[string]any{"ref": info.Ref, "sequence": 0})
		if err == nil || PublicError(err).Code != "plugin.game_account_mismatch" {
			t.Fatal(action, err)
		}
	}
}

func TestHistoricalGroupsOnlyAssignMissingOneBot(t *testing.T) {
	a := pluginApp(t)
	global := &rayleabot.EventContext{Event: rayleabot.Event{EventType: "management.action", SourceProtocol: "management"}}
	for n, scope := range []GroupScope{{Protocol: "onebot11", Adapter: "gone", GroupID: "legacy"}, {Protocol: "qqofficial", Adapter: "gone", GroupID: "openid"}, {Protocol: "onebot11", BotID: "removed", GroupID: "kept"}} {
		path := filepath.Join(a.Groups.Directory, fmt.Sprintf("old-%d.json", n))
		if err := localdata.Write(path, GroupData{Scope: scope, Config: GroupConfig{Aliases: map[string]string{}}, Revision: 2}); err != nil {
			t.Fatal(err)
		}
	}
	out, err := a.Manage(t.Context(), global, "history.list", nil)
	if err != nil {
		t.Fatal(err)
	}
	rows := out["items"].([]historyRecord)
	if len(rows) != 3 || out["total"] != 3 {
		t.Fatal(out)
	}
	for _, row := range rows {
		input := map[string]any{"kind": row.Kind, "ref": row.Ref, "revision": row.Revision, "confirm": true}
		_, err := a.Manage(t.Context(), managementEvent("onebot11", "target"), "history.assign", input)
		if row.Owner.SourceProtocol == "onebot11" && row.Owner.BotID == "" {
			if err != nil {
				t.Fatal(err)
			}
			data, err := a.Groups.Read(GroupScope{Protocol: "onebot11", BotID: "target", GroupID: "legacy"})
			if err != nil || data.Revision != 3 {
				t.Fatal(data, err)
			}
		} else {
			if err == nil {
				t.Fatal("reassigned complete or QQ official owner")
			}
			if _, err := a.Manage(t.Context(), global, "history.get", input); err != nil {
				t.Fatal(err)
			}
			if _, err := a.Manage(t.Context(), global, "history.remove", input); err != nil {
				t.Fatal(err)
			}
		}
	}
	out, err = a.Manage(t.Context(), global, "history.list", nil)
	if err != nil || len(out["items"].([]historyRecord)) != 1 {
		t.Fatal(out, err)
	}
}

func TestHistoricalTaskMergeAndStaleGuard(t *testing.T) {
	a := pluginApp(t)
	tasks := []Reminder{{Ref: "legacy", Owner: Subject{SourceProtocol: "onebot11", ActorID: "u"}}, {Ref: "other", Owner: Subject{SourceProtocol: "onebot11", BotID: "b", ActorID: "u"}}}
	if err := a.Reminders.edit("", func(items *[]Reminder, _ int) error { *items = tasks; return nil }); err != nil {
		t.Fatal(err)
	}
	input := map[string]any{"kind": "reminders", "ref": "legacy", "revision": historyRevision(tasks[0]), "confirm": true}
	if _, err := a.Manage(t.Context(), managementEvent("qqofficial", "opaque"), "history.assign", input); err == nil {
		t.Fatal("assigned OneBot record to QQ official")
	}
	if _, err := a.Manage(t.Context(), managementEvent("onebot11", "a"), "history.assign", input); err != nil {
		t.Fatal(err)
	}
	items, _ := a.Reminders.List()
	if len(items) != 2 || !slices.ContainsFunc(items, func(t Reminder) bool { return t.Ref == "other" && t.Owner.BotID == "b" }) {
		t.Fatal(items)
	}
	if _, err := a.Manage(t.Context(), managementEvent("onebot11", "c"), "history.assign", input); err == nil {
		t.Fatal("stale assignment accepted")
	}
}

func TestAccountServiceUsesManagementOriginAndPropagatesDenials(t *testing.T) {
	a := pluginApp(t)
	const denial = "plugin.account_subject_denied"
	host := newSDKHost(t, a, func(call hostCall) (map[string]any, string) {
		if call.Bot.SourceProtocol != "onebot11" || call.Bot.BotID != "a" {
			t.Fatalf("lost source account: %+v", call)
		}
		for _, field := range []string{"bot", "bot_id", "source_protocol", "subject", "owner", "origin"} {
			if _, exists := call.Params[field]; exists {
				t.Fatalf("plugin supplied %s to provider", field)
			}
		}
		if call.Method == "list" {
			return map[string]any{"items": []Account{{Ref: "a", Owner: Subject{SourceProtocol: "onebot11", BotID: "a"}}}}, ""
		}
		return nil, denial
	})
	out, _ := host.manageAccount("accounts.list", nil, "onebot11", "a")
	if out["type"] != "result" {
		t.Fatal(out)
	}
	for _, action := range []string{"query", "accounts.select", "panel.history.list", "monthly.list", "reminder.create", "gacha.sync.start", "signin.status"} {
		input := map[string]any{"account_ref": "other", "role_ref": "role", "character_id": "1001", "operation": "zzz.note", "threshold": 80, "days": 7}
		out, _ := host.manageAccount(action, input, "onebot11", "a")
		if out["type"] != "error" || out["code"] != denial {
			t.Fatalf("%s hid provider denial: %v", action, out)
		}
	}
	if tasks, _ := a.Reminders.List(); len(tasks) != 0 {
		t.Fatal("denied account created reminder")
	}
}

func TestBackgroundSyncKeepsAccountServiceOriginAfterDetach(t *testing.T) {
	_, _, accounts, host := syncHost(t)
	detached := 0
	host.service = func(call hostCall) (map[string]any, string) {
		if call.Bot.SourceProtocol != "onebot11" || call.Bot.BotID != "bot" {
			t.Fatalf("lost account: %+v", call)
		}
		if call.Detached {
			detached++
		}
		if _, provided := call.Params["bot"]; provided {
			t.Fatal("bot provided as service parameter")
		}
		return accounts.answer(call)
	}
	out, _ := host.manage("gacha.task.start", map[string]any{"account_ref": "account", "role_ref": "role", "confirm": true})
	if out["type"] != "result" || detached == 0 {
		t.Fatal(out, detached)
	}
}

func TestArchivesRecordProvenanceForHistory(t *testing.T) {
	a := pluginApp(t)
	choice := Selection{AccountRef: "account", RoleRef: "role"}
	owner := Subject{SourceProtocol: "onebot11", BotID: "a"}
	for _, saver := range []Subject{{}, owner} {
		if _, err := a.PanelHistory.Save("p", choice, "v1", CharacterPanel{ID: "1001", Name: "角色"}, saver); err != nil {
			t.Fatal(err)
		}
	}
	panels, _ := a.PanelHistory.Read("p", choice, "1001")
	if len(panels) != 2 || panels[0].Owner != owner || panels[1].Owner != owner {
		t.Fatalf("legacy snapshot kept no provenance: %+v", panels)
	}
	month := map[string]any{"data_month": 202609, "month_data": map[string]any{}}
	for _, saver := range []Subject{owner, {}} {
		if err := a.Monthly.Keep("p", choice, month, time.Now(), saver); err != nil {
			t.Fatal(err)
		}
	}
	if monthly, _ := a.Monthly.Read("p", choice); monthly.Owner != owner {
		t.Fatalf("scheduled read dropped provenance: %+v", monthly.Owner)
	}
}
