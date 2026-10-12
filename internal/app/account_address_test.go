package app

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/zzz/internal/localdata"
)

func TestAccountIdentity(t *testing.T) {
	owner := Subject{"onebot11", "123", "user"}
	bots := []rayleabot.Bot{{SourceProtocol: "onebot11", SourceAdapter: "replacement", ID: "123"}, {SourceProtocol: "onebot11", SourceAdapter: "second", ID: "123"}}
	if !accountOnline(owner.SourceProtocol, owner.BotID, bots) || accountOnline("qqofficial", "123", bots) || accountOnline("onebot11", "", bots) {
		t.Fatal("online check did not select the protocol and account")
	}
	store := &InteractionStore{Path: filepath.Join(t.TempDir(), "interactions.json")}
	if err := localdata.Write(store.Path, []InteractionProfile{{Owner: owner, LastImage: "image"}}); err != nil {
		t.Fatal(err)
	}
	if got, err := store.Get(owner); err != nil || got.LastImage != "image" {
		t.Fatal(got, err)
	}
	groups := &GroupStore{Directory: t.TempDir()}
	scope := GroupScope{"onebot11", "123", "g"}
	if err := groups.Update(scope, func(d *GroupData) error { off := false; d.Config.Enabled = &off; return nil }); err != nil {
		t.Fatal(err)
	}
	if config, err := groups.Config(scope); err != nil || config.Enabled == nil || *config.Enabled {
		t.Fatal(config, err)
	}
	for _, value := range []any{owner, scope} {
		raw, err := json.Marshal(value)
		if err != nil || strings.Contains(string(raw), "source_adapter") {
			t.Fatalf("new record carries adapter: %s %v", raw, err)
		}
	}
	other := scope
	other.BotID = "another"
	if groups.file(scope) == groups.file(other) {
		t.Fatal("different bot accounts share settings")
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

func TestReminderSendsThroughAccount(t *testing.T) {
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
	end, actions := host.trigger(items[0].Ref)
	if end["type"] != "result" || len(host.sent) != 1 || host.sent[0].TargetType != "private" || host.sent[0].TargetID != "u" {
		t.Fatal(end, host.sent)
	}
	assertAccountMessages(t, actions)
}
