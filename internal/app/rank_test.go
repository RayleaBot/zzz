package app

import (
	"sync"
	"testing"
)

func TestRankKeepsOneEntryPerUIDAndCharacter(t *testing.T) {
	store := &GroupStore{Directory: t.TempDir()}
	scope := GroupScope{Protocol: "onebot11", Adapter: "adapter", BotID: "bot", GroupID: "group"}
	base := RankEntry{ActorID: "user1", UID: "100000001", CharacterID: "10000046", Score: 100, Panel: &CharacterPanel{ID: "10000046"}, UpdatedAtMS: 5}
	var wg sync.WaitGroup
	for range 6 {
		wg.Go(func() {
			if err := store.Submit(scope, base); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	// The same member's second UID ranks on its own, as upstream ranks UIDs.
	base.UID = "100000002"
	if err := store.Submit(scope, base); err != nil {
		t.Fatal(err)
	}
	data, _ := store.Read(scope)
	if len(data.Rank) != 2 || data.RankSinceMS != 5 {
		t.Fatalf("rank = %d entries since %d", len(data.Rank), data.RankSinceMS)
	}
	base.Score = -1
	if err := store.Submit(scope, base); err == nil {
		t.Fatal("invalid score accepted")
	}
}

func TestRankScoresTakesTheDefaultDetail(t *testing.T) {
	a := &App{}
	total := 180.5
	expected := 12345.0
	damage := &BuildResult{Baseline: BuildScenario{Results: []BuildSkillResult{{Title: "E伤害", Expected: new(float64)}, {Title: "Q伤害", Expected: &expected, Default: true}, {Title: "治疗", Text: "12.5%"}}}}
	entry := RankEntry{}
	a.rankScores(&entry, CharacterPanel{ID: "1", TotalScore: &total, ScoredEquipment: 5, ScoreDetail: &ScoreDetail{Grade: "SS"}}, damage)
	if entry.Score != total || entry.Grade != "SS" || entry.Damage == nil || entry.Damage.Title != "Q伤害" || entry.Damage.Value != expected || entry.Panel == nil {
		t.Fatalf("entry = %+v", entry)
	}
	damage.Baseline.Results[1].Default, damage.Baseline.Results[2].Default = false, true
	a.rankScores(&entry, CharacterPanel{ID: "1"}, damage)
	if entry.Score != 0 || entry.Damage == nil || entry.Damage.Value != 12.5 || entry.Damage.Text != "12.5%" {
		t.Fatalf("text entry = %+v", entry)
	}
}
