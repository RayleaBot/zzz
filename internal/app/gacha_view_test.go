package app

import (
	"strings"
	"testing"

	"github.com/RayleaBot/zzz/internal/gacha"
)

func TestGachaTextListsEachTopPullWithUpAndAverages(t *testing.T) {
	game := Game{ID: "zzz", Name: "绝区零", Data: &GameData{Resources: GameResources{Pools: []PoolInfo{{From: "2026-01-01 00:00:00", To: "2026-01-21 23:59:59", Characters5: []string{"艾莲"}}}}}}
	records := []gacha.Record{}
	add := func(id, name, rank, when string) {
		records = append(records, gacha.Record{ID: id, Name: name, Rank: rank, GachaType: "2", Time: when})
	}
	for index := range 9 {
		add(string(rune('a'+index)), "「月相」-望", "2", "2026-01-02 00:00:00")
	}
	add("j", "安东", "3", "2026-01-02 00:00:00")
	add("k", "猫宫又奈", "4", "2026-01-02 00:00:01")
	add("l", "「月相」-望", "2", "2026-01-03 00:00:00")
	add("m", "艾莲", "4", "2026-01-03 00:00:01")
	view := GachaView(game, gacha.Archive{UID: "100000001", Records: records})
	text := view.Text()
	for _, want := range []string{"猫宫又奈 11+", "艾莲 2 UP", "UP 平均：13.0 抽", "1 个 · 平均 13.0 抽"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in\n%s", want, text)
		}
	}
}
