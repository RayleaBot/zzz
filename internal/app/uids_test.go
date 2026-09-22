package app

import (
	"strings"
	"testing"
)

func TestUIDListNumbersAccountUIDsBeforeBoundOnes(t *testing.T) {
	a := &App{Game: Game{ID: "zzz", Name: "绝区零", Prefix: "#"}}
	listed := Accounts{
		Items:       []Account{{Ref: "a1", Roles: []Role{{Ref: "r1", Game: "zzz", UID: "100000001", Nickname: "绳匠"}, {Ref: "r2", Game: "starrail", UID: "800000001"}}}},
		Defaults:    map[string]Selection{"zzz": {AccountRef: "a1", RoleRef: "r1"}},
		UIDBindings: map[string]UIDBinding{"zzz": {UIDs: []string{"100000002", "100000001"}}},
	}
	list := a.uidList(listed)
	if len(list) != 2 || list[0].Role == nil || list[1].UID != "100000002" || list[1].Role != nil {
		t.Fatalf("list = %+v", list)
	}
	// Without a picked UID the default account role is in use.
	if a.currentUID(listed) != "100000001" {
		t.Fatal("default role not in use")
	}
	listed.UIDBindings["zzz"] = UIDBinding{UIDs: []string{"100000002"}, Current: "100000002"}
	// The image shows each UID's player from its saved panels, with the
	// first saved character's face as miao does.
	a.Profiles = &PanelStore{Directory: t.TempDir()}
	if _, err := a.Profiles.Keep("100000001", []CharacterPanel{{ID: "1191"}, {ID: "1041"}}, "enka", &ShowcaseProfile{Nickname: "绳匠", Level: 60}); err != nil {
		t.Fatal(err)
	}
	view, image := a.uidView(listed)
	if a.currentUID(listed) != "100000002" || !strings.Contains(view.Text(), "☑2：100000002 [绑定]") {
		t.Fatal(view.Text())
	}
	first, second := image.Entries[0], image.Entries[1]
	if !first.Account || first.Active || first.Nickname != "绳匠" || first.Level != 60 || first.Character != "1041" || second.Account || !second.Active || second.Character != "" {
		t.Fatalf("entries = %+v", image.Entries)
	}
}
