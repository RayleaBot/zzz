package app

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestPanelHistoryScopesRetentionAndRestart(t *testing.T) {
	store := &PanelHistoryStore{Directory: t.TempDir()}
	choice := Selection{AccountRef: "account", RoleRef: "role"}
	panel := CharacterPanel{ID: "1001", Name: "测试", Level: 80}
	first, err := store.Save("provider", choice, "v1", panel)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		provider string
		choice   Selection
		id       string
	}{{"other", choice, "1001"}, {"provider", Selection{AccountRef: "other", RoleRef: "role"}, "1001"}, {"provider", choice, "1002"}} {
		items, err := store.Read(test.provider, test.choice, test.id)
		if err != nil || len(items) != 0 {
			t.Fatal("cross-account history exposed")
		}
	}
	if err = store.Remove("other", choice, "1001", first.Ref); err == nil {
		t.Fatal("other provider removed snapshot")
	}
	var wg sync.WaitGroup
	var added atomic.Int32
	for range 24 {
		wg.Go(func() {
			if _, err := store.Save("provider", choice, "v1", panel); err == nil {
				added.Add(1)
			}
		})
	}
	wg.Wait()
	if added.Load() != 19 {
		t.Fatal("retention admission raced", added.Load())
	}
	restarted := &PanelHistoryStore{Directory: store.Directory}
	items, err := restarted.Read("provider", choice, "1001")
	if err != nil || len(items) != 20 {
		t.Fatal("history lost on restart", err)
	}
	if err = restarted.Remove("provider", choice, "1001", first.Ref); err != nil {
		t.Fatal(err)
	}
	if _, err = restarted.Save("provider", choice, "v2", panel); err != nil {
		t.Fatal(err)
	}
}
func TestHistoryComparisonKeepsUnitsAndMissingValues(t *testing.T) {
	before := PanelSnapshot{Panel: CharacterPanel{Name: "测试", Stats: []PanelStat{{ID: "1", Name: "攻击", Value: "1,000"}, {ID: "2", Name: "暴击", Value: "50%"}, {ID: "3", Name: "速度", Value: "100"}}}}
	after := PanelSnapshot{Panel: CharacterPanel{Name: "测试", Stats: []PanelStat{{ID: "1", Name: "攻击", Value: "1,200"}, {ID: "2", Name: "暴击", Value: "60%"}, {ID: "4", Name: "充能", Value: "120%"}}}}
	view := comparePanels(Game{ID: "starrail"}, before, after)
	rows := view.Sections[0].Rows
	if !strings.Contains(rows[0].Value, "+200.00") || !strings.Contains(rows[1].Value, "+10.00 个百分点") || strings.Contains(rows[2].Value, "（") || strings.Contains(rows[3].Value, "（") {
		t.Fatal(rows)
	}
	after.Panel.Stats[1].Value = "60"
	rows = comparePanels(Game{}, before, after).Sections[0].Rows
	if strings.Contains(rows[1].Value, "（") {
		t.Fatal("different units subtracted")
	}
}
