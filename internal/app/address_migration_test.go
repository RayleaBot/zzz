package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/zzz/internal/localdata"
)

func legacyAddressRecord(t *testing.T, value any, field, adapter string) map[string]any {
	t.Helper()
	var record map[string]any
	if err := decodeObject(value, &record); err != nil {
		t.Fatal(err)
	}
	asObject(record[field])["source_adapter"] = adapter
	return record
}

func writeAddressRecord(t *testing.T, path string, value any) {
	t.Helper()
	if err := localdata.Write(path, value); err != nil {
		t.Fatal(err)
	}
}

func TestAccountIdentityIgnoresAdapter(t *testing.T) {
	old := Subject{"onebot11", "removed", "123", "user"}
	current := Subject{"onebot11", "replacement", "123", "user"}
	if old.identity() != current.identity() || interactionRef(old) != interactionRef(current) || contentRef("zzz", old, "g") != contentRef("zzz", current, "g") {
		t.Fatal("adapter still participates in identity")
	}
	bots := []rayleabot.Bot{{SourceProtocol: "onebot11", SourceAdapter: "replacement", ID: "123"}, {SourceProtocol: "onebot11", SourceAdapter: "second", ID: "123"}}
	if !accountOnline(old.SourceProtocol, old.BotID, bots) || accountOnline("qqofficial", "123", bots) || accountOnline("onebot11", "", bots) {
		t.Fatal("online check did not select the protocol and account")
	}
	store := &InteractionStore{Path: filepath.Join(t.TempDir(), "interactions.json")}
	writeAddressRecord(t, store.Path, []any{legacyAddressRecord(t, InteractionProfile{Owner: old, LastImage: "image"}, "owner", old.SourceAdapter)})
	if got, err := store.Get(current); err != nil || got.LastImage != "image" {
		t.Fatal(got, err)
	}
	groups := &GroupStore{Directory: t.TempDir()}
	scope := GroupScope{"onebot11", "removed", "123", "g"}
	if err := groups.Update(scope, func(d *GroupData) error { off := false; d.Config.Enabled = &off; return nil }); err != nil {
		t.Fatal(err)
	}
	other := scope
	other.Adapter = "replacement"
	if groups.file(scope) != groups.file(other) {
		t.Fatal("adapter changed group filename")
	}
	if config, err := groups.Config(other); err != nil || config.Enabled == nil || *config.Enabled {
		t.Fatal(config, err)
	}
	for _, value := range []any{old, scope} {
		raw, err := json.Marshal(value)
		if err != nil || strings.Contains(string(raw), "source_adapter") {
			t.Fatalf("new record carries adapter: %s %v", raw, err)
		}
	}
	other.BotID = "another"
	if groups.file(scope) == groups.file(other) {
		t.Fatal("different bot accounts share settings")
	}
}

func TestGroupMigrationMergesAndRetainsSources(t *testing.T) {
	s := &GroupStore{Directory: t.TempDir()}
	off, on := false, true
	scope := GroupScope{Protocol: "onebot11", BotID: "123", GroupID: "g"}
	newer := GroupData{Scope: scope, Revision: 8, Config: GroupConfig{Enabled: &off, Aliases: map[string]string{"same": "new", "first": "1"}}, Rank: []RankEntry{{UID: "u1", CharacterID: "c", Score: 9}}}
	older := GroupData{Scope: scope, Revision: 2, Config: GroupConfig{Enabled: &on, ImageReplies: &off, Aliases: map[string]string{"same": "old", "second": "2"}}, Rank: []RankEntry{{UID: "u2", CharacterID: "c", Score: 8}}, QueryRanks: map[string]map[string]QueryRankMember{"q": {"u2": {Hidden: true}}}}
	writeAddressRecord(t, filepath.Join(s.Directory, "old-a.json"), legacyAddressRecord(t, older, "scope", "a"))
	writeAddressRecord(t, filepath.Join(s.Directory, "old-b.json"), legacyAddressRecord(t, newer, "scope", "b"))
	unknown := GroupData{Scope: GroupScope{Protocol: "qqofficial", GroupID: "unknown", Adapter: "a"}, Config: GroupConfig{Enabled: &off}}
	writeAddressRecord(t, filepath.Join(s.Directory, "unknown.json"), unknown)
	warnings := []string{}
	bots := []rayleabot.Bot{{SourceProtocol: "onebot11", SourceAdapter: "a", ID: "123"}}
	if err := s.migrate(bots, func(ref string) { warnings = append(warnings, ref) }); err != nil {
		t.Fatal(err)
	}
	got, err := s.Read(scope)
	if err != nil || *got.Config.Enabled || *got.Config.ImageReplies || got.Config.Aliases["same"] != "new" || len(got.Config.Aliases) != 3 || len(got.Rank) != 2 || !got.QueryRanks["q"]["u2"].Hidden {
		t.Fatal(got, err)
	}
	for _, file := range []string{"old-a.json.legacy", "old-b.json.legacy", "unknown.json"} {
		if _, err := os.Stat(filepath.Join(s.Directory, file)); err != nil {
			t.Fatal(err)
		}
	}
	if !slices.Contains(warnings, "unknown.json") {
		t.Fatal("missing bot was not reported")
	}
	if err := s.migrate(bots, func(string) {}); err != nil {
		t.Fatal(err)
	}
	again, _ := s.Read(scope)
	if !sameTask(got, again) {
		t.Fatal("repeated migration changed group data")
	}
	bots = append(bots, rayleabot.Bot{SourceProtocol: "qqofficial", SourceAdapter: "a", ID: "official"})
	if err := s.migrate(bots, func(string) {}); err != nil {
		t.Fatal(err)
	}
	recovered, err := s.Read(GroupScope{Protocol: "qqofficial", BotID: "official", GroupID: "unknown"})
	if err != nil || recovered.Config.Enabled != nil {
		t.Fatal("QQ official openids were reassigned through a reused connection", recovered, err)
	}
	if _, err := os.Stat(filepath.Join(s.Directory, "unknown.json")); err != nil {
		t.Fatal("unassigned QQ official record was lost", err)
	}
}

func TestGroupMigrationWriteFailureKeepsLegacyFile(t *testing.T) {
	s := &GroupStore{Directory: t.TempDir()}
	scope := GroupScope{Protocol: "onebot11", BotID: "123", GroupID: "g"}
	old := filepath.Join(s.Directory, "old.json")
	writeAddressRecord(t, old, legacyAddressRecord(t, GroupData{Scope: scope, Config: GroupConfig{Aliases: map[string]string{"kept": "1"}}}, "scope", "removed"))
	if err := os.Mkdir(s.file(scope), 0700); err != nil {
		t.Fatal(err)
	}
	if err := s.migrate(nil, func(string) {}); err == nil {
		t.Fatal("failed migration reported success")
	}
	if _, err := os.Stat(old); err != nil {
		t.Fatal("legacy file lost", err)
	}
	if err := os.Remove(s.file(scope)); err != nil {
		t.Fatal(err)
	}
	if err := s.migrate(nil, func(string) {}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Read(scope)
	if err != nil || got.Config.Aliases["kept"] != "1" {
		t.Fatal(got, err)
	}
}

func TestSubscriptionMigrationPreservesDedupAcrossSchedulerFailure(t *testing.T) {
	s := &ContentSubscriptions{Path: filepath.Join(t.TempDir(), "content-subscriptions.json")}
	owner := Subject{SourceProtocol: "onebot11", BotID: "123", ActorID: "user"}
	now := time.Now()
	first := ContentSubscription{Ref: "old-a", Owner: owner, GroupID: "g", Kinds: []string{"announce"}, Sent: map[string]int64{"post": now.UnixMilli() - 10}}
	second := ContentSubscription{Ref: "old-b", Owner: owner, GroupID: "g", Kinds: []string{"info"}, Sent: map[string]int64{"post": now.UnixMilli(), "another": now.UnixMilli()}}
	unknown := ContentSubscription{Ref: "unknown", Owner: Subject{SourceProtocol: "onebot11", SourceAdapter: "offline", ActorID: "user"}, GroupID: "other"}
	writeAddressRecord(t, s.Path, []any{legacyAddressRecord(t, first, "owner", "a"), legacyAddressRecord(t, second, "owner", "b"), unknown})
	created, removed := []string{}, []string{}
	create := func(ref string) error { created = append(created, ref); return nil }
	warn := func(string) {}
	err := s.migrate("zzz", nil, warn, create, func(string) error { return errors.New("scheduler unavailable") })
	if err == nil {
		t.Fatal("scheduler error swallowed")
	}
	items, _ := s.List()
	if len(items) != 2 || len(items[0].Kinds) != 2 || len(items[0].Sent) != 2 || items[0].Sent["post"] != now.UnixMilli() || len(items[0].RetiredRefs) != 2 || items[1].Owner.SourceAdapter != "offline" {
		t.Fatal(items)
	}
	if _, ok := postToPush([]pushPost{{id: "post", kind: "announce", created: now.Unix()}}, items[0], now); ok {
		t.Fatal("migration lost sent state")
	}
	remove := func(ref string) error { removed = append(removed, ref); return nil }
	if err := s.migrate("zzz", nil, warn, create, remove); err != nil {
		t.Fatal(err)
	}
	items, _ = s.List()
	if len(items[0].RetiredRefs) != 0 || !slices.Equal(removed, []string{"old-a", "old-b"}) {
		t.Fatal(items, removed)
	}
	if err := s.migrate("zzz", nil, warn, create, remove); err != nil || len(created) != 2 {
		t.Fatal("settled migration changed scheduler", created, err)
	}
	bots := []rayleabot.Bot{{SourceProtocol: "onebot11", SourceAdapter: "offline", ID: "123"}}
	if err := s.migrate("zzz", bots, warn, create, remove); err != nil {
		t.Fatal(err)
	}
	items, _ = s.List()
	if items[1].Owner.BotID != "123" || items[1].Owner.SourceAdapter != "" {
		t.Fatal(items[1])
	}
}

func TestTaskOwnerRecoveryKeepsNotificationState(t *testing.T) {
	directory := t.TempDir()
	reminders, daily := reminderStore(directory), dailySyncStore(directory)
	owner := Subject{SourceProtocol: "onebot11", SourceAdapter: "old", ActorID: "user"}
	reminder := Reminder{Ref: "reminder", Owner: owner, Armed: false, LastAttemptMS: 1234, LastCode: "notification_failed", LastSignDay: "2026-10-01"}
	sync := DailySync{Ref: "daily", Owner: owner, RunDay: "2026-10-01", LastNotificationMS: 1234, LastCode: "sync_completed.notification_failed"}
	writeAddressRecord(t, reminders.file(reminder.Ref), reminder)
	writeAddressRecord(t, daily.file(sync.Ref), sync)
	warnings := 0
	warn := func(string) { warnings++ }
	if err := migrateTaskOwners(&reminders.taskFiles, nil, func(r *Reminder) *Subject { return &r.Owner }, warn); err != nil || warnings != 1 {
		t.Fatal(err, warnings)
	}
	kept, _ := reminders.List()
	if kept[0].Owner.SourceAdapter != "old" {
		t.Fatal("unresolved adapter was lost")
	}
	bots := []rayleabot.Bot{{SourceProtocol: "qqofficial", SourceAdapter: "old", ID: "wrong"}, {SourceProtocol: "onebot11", SourceAdapter: "old", ID: "123"}}
	if err := migrateTaskOwners(&reminders.taskFiles, bots, func(r *Reminder) *Subject { return &r.Owner }, warn); err != nil {
		t.Fatal(err)
	}
	if err := migrateTaskOwners(&daily.taskFiles, bots, func(d *DailySync) *Subject { return &d.Owner }, warn); err != nil {
		t.Fatal(err)
	}
	reminder.Owner.BotID, reminder.Owner.SourceAdapter = "123", ""
	sync.Owner = reminder.Owner
	kept, _ = reminders.List()
	syncs, _ := daily.List()
	if !sameTask(kept[0], reminder) || !sameTask(syncs[0], sync) {
		t.Fatal("migration changed delivery state", kept, syncs)
	}
}

func assertAccountMessages(t *testing.T, actions []hostAction) {
	t.Helper()
	count := 0
	for _, action := range actions {
		if action.Name != "message.send" {
			continue
		}
		count++
		_, adapter := action.Data["source_adapter"]
		if adapter || action.Data["source_protocol"] != "onebot11" || action.Data["bot_id"] != "bot" {
			t.Fatalf("message used an instance route: %v", action.Data)
		}
	}
	if count == 0 {
		t.Fatal("no account message sent")
	}
}

func TestReminderSendsThroughReplacementAccount(t *testing.T) {
	a := pluginApp(t)
	accounts := &taskAccounts{t: t}
	host := newSDKHost(t, a, func(call hostCall) (map[string]any, string) {
		if call.Method == "execute" {
			return map[string]any{"data": map[string]any{"energy": map[string]any{"progress": map[string]any{"current": 90, "max": 100}}}}, ""
		}
		return accounts.answer(call)
	})
	end, _ := host.manage("reminder.create", map[string]any{"account_ref": "account", "role_ref": "role", "threshold": 80, "days": 30})
	if end["type"] != "result" {
		t.Fatal(end)
	}
	items, err := a.Reminders.List()
	if err != nil || len(items) != 1 {
		t.Fatal(items, err)
	}
	writeAddressRecord(t, a.Reminders.file(items[0].Ref), legacyAddressRecord(t, items[0], "owner", "removed"))
	end, actions := host.trigger(items[0].Ref)
	if end["type"] != "result" || len(host.sent) != 1 || host.sent[0].TargetType != "private" || host.sent[0].TargetID != "u" {
		t.Fatal(end, host.sent)
	}
	assertAccountMessages(t, actions)
}
