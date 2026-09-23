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
