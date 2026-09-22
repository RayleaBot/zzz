package app

import (
	"strings"
	"testing"

	"github.com/RayleaBot/plugin-zzz/internal/gacha"
)

func TestGachaTextListsEachTopPullWithUpAndAverages(t *testing.T) {
	game := Game{ID: "genshin", Name: "原神", Data: &GameData{Resources: GameResources{Pools: []PoolInfo{{From: "2026-01-01 00:00:00", To: "2026-01-21 23:59:59", Characters5: []string{"胡桃"}}}}}}
	records := []gacha.Record{}
	add := func(id, name, rank, when string) {
		records = append(records, gacha.Record{ID: id, Name: name, Rank: rank, GachaType: "301", Time: when})
	}
	for index := range 9 {
		add(string(rune('a'+index)), "饰铁之花", "3", "2026-01-02 00:00:00")
	}
	add("j", "香菱", "4", "2026-01-02 00:00:00")
	add("k", "迪卢克", "5", "2026-01-02 00:00:01")
	add("l", "饰铁之花", "3", "2026-01-03 00:00:00")
	add("m", "胡桃", "5", "2026-01-03 00:00:01")
	view := GachaView(game, gacha.Archive{UID: "100000001", Records: records})
	text := view.Text()
	for _, want := range []string{"迪卢克 11+", "胡桃 2 UP", "UP 平均：13.0 抽", "1 个 · 平均 13.0 抽"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in\n%s", want, text)
		}
	}
}
