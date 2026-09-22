package app

import (
	"reflect"
	"testing"
)

func TestPanelChangeReadsMiaoWording(t *testing.T) {
	catalog, err := ParseCatalog([]byte(`{"version":"1","entries":[
{"id":"10000052","name":"雷电将军","kind":"character","aliases":["雷神"]},{"id":"10000042","name":"刻晴","kind":"character"},
{"id":"13501","name":"护摩之杖","kind":"weapon","aliases":["护摩"]}],
"artifact_sets":{"72544":"绝缘之旗印","71544":"追忆之注连"},"set_abbrs":{"绝缘之旗印":"绝缘"},"set_aliases":{"追忆之注连":["追忆"]}}`))
	if err != nil {
		t.Fatal(err)
	}
	gs := Game{ID: "genshin"}
	change, ok := parsePanelChange(gs, catalog, nil, "雷神换90级5精护摩换绝缘4换天赋10 10 10")
	if !ok || change.CharacterID != "10000052" || change.Weapon == nil || change.Weapon.ID != "13501" || change.Weapon.Level != 90 || change.Weapon.Refinement != 5 {
		t.Fatalf("weapon: %+v", change)
	}
	if !reflect.DeepEqual(change.Sets, []string{"绝缘之旗印", "绝缘之旗印"}) || !reflect.DeepEqual(change.Talents, map[string]int{"a": 10, "e": 10, "q": 10}) {
		t.Fatalf("sets and talents: %+v", change)
	}
	// Two sets fill the first two slots and the last three.
	if change, _ = parsePanelChange(gs, catalog, nil, "雷神换绝缘2+追忆2"); !reflect.DeepEqual(change.Sets, []string{"绝缘之旗印", "追忆之注连"}) {
		t.Fatalf("two sets: %v", change.Sets)
	}
	change, _ = parsePanelChange(gs, catalog, nil, "雷神换刻晴圣遗物+武器")
	if change.Artifacts == nil || change.Artifacts.CharacterID != "10000042" || change.WeaponFrom == nil || change.WeaponFrom.CharacterID != "10000042" {
		t.Fatalf("parts of another character: %+v", change)
	}
	change, _ = parsePanelChange(gs, catalog, nil, "100000001雷神换刻晴花")
	if change.UID != "100000001" || change.Pieces[1] != (changeSource{CharacterID: "10000042"}) {
		t.Fatalf("a piece: %+v", change)
	}
	change, _ = parsePanelChange(gs, catalog, nil, "雷神换满命换80级")
	if change.Cons == nil || *change.Cons != 6 || change.Level != 80 {
		t.Fatalf("constellation and level: %+v", change)
	}
	if change, _ = parsePanelChange(gs, catalog, nil, "雷神换刻晴"); change.Swap != "10000042" {
		t.Fatalf("another character: %+v", change)
	}
	if _, ok = parsePanelChange(gs, catalog, nil, "雷神换一个好心情"); ok {
		t.Fatal("a word without a change was read")
	}
	if _, ok = parsePanelChange(gs, catalog, nil, "兑换码"); ok {
		t.Fatal("a word without a character was read")
	}
	change, _ = parsePanelChange(Game{ID: "starrail"}, catalog, nil, "刻晴换满行迹换2魂")
	if len(change.Trees) != 15 || change.Cons == nil || *change.Cons != 2 {
		t.Fatalf("traces: %+v", change)
	}
}
