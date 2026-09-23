package app

import (
	"sync"
	"testing"
)

func TestRankKeepsOneEntryPerUIDAndCharacter(t *testing.T) {
	store := &GroupStore{Directory: t.TempDir()}
	scope := GroupScope{Protocol: "onebot11", Adapter: "adapter", BotID: "bot", GroupID: "group"}
	base := RankEntry{Nickname: "user1", UID: "100000001", CharacterID: "10000046", Score: 100}
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
	if len(data.Rank) != 2 {
		t.Fatalf("rank = %d entries", len(data.Rank))
	}
	base.Score = -1
	if err := store.Submit(scope, base); err == nil {
		t.Fatal("invalid score accepted")
	}
}
