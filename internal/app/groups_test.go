package app

import (
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func TestGroupSettingsDoNotChangeGlobalOrOtherBots(t *testing.T) {
	store := &GroupStore{Directory: t.TempDir()}
	scope := GroupScope{Protocol: "onebot11", Adapter: "adapter", BotID: "bot", GroupID: "group"}
	off := false
	if err := store.Update(scope, func(d *GroupData) error {
		d.Config.ImageReplies = &off
		d.Config.Aliases = map[string]string{"name": "1001"}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	other := scope
	other.BotID = "other"
	value, err := store.Read(other)
	if err != nil || value.Config.ImageReplies != nil || len(value.Config.Aliases) != 0 {
		t.Fatal("cross-bot group settings")
	}
	global := map[string]any{"image_replies": true, "custom_aliases": map[string]string{"global": "1002", "name": "1003"}}
	event := &rayleabot.EventContext{Config: global}
	saved, _ := store.Read(scope)
	applyGroupConfig(event, saved.Config)
	if global["image_replies"] != true || global["custom_aliases"].(map[string]string)["name"] != "1003" {
		t.Fatal("global snapshot mutated")
	}
	settings := settings(event)
	if settings.ImageReplies || settings.CustomAliases["global"] != "1002" || settings.CustomAliases["name"] != "1001" {
		t.Fatal(settings)
	}
}
func TestGroupAdministratorMatchesHostRoles(t *testing.T) {
	for _, role := range []string{"member", "admin", "owner"} {
		event := &rayleabot.EventContext{Event: rayleabot.Event{Actor: rayleabot.Actor{ID: "u", Role: role}}}
		if groupAdministrator(event) != (role != "member") {
			t.Fatal(role)
		}
	}
}

func TestGroupManagementRejectsStaleRevision(t *testing.T) {
	store := &GroupStore{Directory: t.TempDir()}
	scope := GroupScope{Protocol: "p", Adapter: "a", BotID: "b", GroupID: "g"}
	app := &App{Groups: store}
	input := map[string]any{"scope": scope, "revision": 0, "config": map[string]any{"enabled": false}}
	if _, err := app.manageGroups("groups.set", input); err != nil {
		t.Fatal(err)
	}
	if _, err := app.manageGroups("groups.clear", input); err == nil || PublicError(err).Code != "plugin.game_group_changed" {
		t.Fatal("stale clear changed group", err)
	}
	data, _ := store.Read(scope)
	if data.Config.Enabled == nil || *data.Config.Enabled {
		t.Fatal("failed clear changed settings")
	}
}

func TestGroupConfigFollowsEveryWrite(t *testing.T) {
	s := &GroupStore{Directory: t.TempDir()}
	scope := GroupScope{"onebot11", "adapter", "bot", "group"}
	if config, err := s.Config(scope); err != nil || config.Enabled != nil {
		t.Fatal(config, err)
	}
	off := false
	if err := s.Update(scope, func(data *GroupData) error { data.Config.Enabled = &off; return nil }); err != nil {
		t.Fatal(err)
	}
	if config, _ := s.Config(scope); config.Enabled == nil || *config.Enabled {
		t.Fatal("the kept settings missed a write", config)
	}
	// Another process start reads the file again.
	if config, _ := (&GroupStore{Directory: s.Directory}).Config(scope); config.Enabled == nil || *config.Enabled {
		t.Fatal(config)
	}
}
