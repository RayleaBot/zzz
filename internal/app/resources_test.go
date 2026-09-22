package app

import "testing"

func TestMaterialRotationAndRelatedCatalogEntries(t *testing.T) {
	a := &App{Game: testGame(t, "genshin"), Catalog: Catalog{Entries: []Entry{{Name: "测试角色", Materials: map[string]string{"天赋材料": "「勤劳」的哲学"}}}}}
	result, err := a.resourceQuery("materials.query", map[string]any{"query": "勤劳", "weekday": 2})
	if err != nil || result["total"].(int) != 3 {
		t.Fatal(result, err)
	}
	for _, m := range result["materials"].([]MaterialInfo) {
		if len(m.Related) != 1 {
			t.Fatal("material family lost relation")
		}
	}
	result, _ = a.resourceQuery("materials.query", map[string]any{"query": "勤劳", "weekday": 1})
	if result["total"].(int) != 0 {
		t.Fatal("wrong rotation day")
	}
	result, _ = a.resourceQuery("materials.query", map[string]any{"query": "勤劳", "weekday": 7})
	if result["total"].(int) != 3 {
		t.Fatal("Sunday not available")
	}
}
func TestCalendarVersionDateAndCoverageBoundaries(t *testing.T) {
	a := &App{Game: testGame(t, "genshin")}
	result, err := a.resourceQuery("calendar.query", map[string]any{"date": "2020-09-28", "version": "1.0"})
	if err != nil || result["total"].(int) != 1 || result["pools"].([]PoolInfo)[0].Characters5[0] != "温迪" {
		t.Fatal(result, err)
	}
	result, _ = a.resourceQuery("calendar.query", map[string]any{"date": "2026-07-15"})
	found := false
	for _, b := range result["birthdays"].([]Birthday) {
		found = found || b.Name == "胡桃"
	}
	if !found {
		t.Fatal("birthday not selected")
	}
	if _, err = a.resourceQuery("calendar.query", map[string]any{"date": "2026-02-30"}); err == nil {
		t.Fatal("invalid date accepted")
	}
	result, _ = a.resourceQuery("calendar.query", map[string]any{"version": "unknown"})
	if result["total"].(int) != 0 {
		t.Fatal("calendar extrapolated unknown version")
	}
}
