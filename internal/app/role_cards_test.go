package app

import (
	"testing"
	"time"
)

func TestRoleCardMatchingRequiresMutualNeedSameRegionAndFreshOptIn(t *testing.T) {
	now := time.Now().UnixMilli()
	self := RoleCardOffer{ActorID: "self", UID: "1", Region: "cn_gf01", Cards: []RoleCard{{"A", 2, true}, {"B", 0, false}}}
	other := RoleCardOffer{ActorID: "other", UID: "2", Region: "cn_gf01", Cards: []RoleCard{{"A", 0, false}, {"B", 3, true}}, SavedMS: now}
	rows := []RoleCardOffer{other}
	foreign := other
	foreign.Region = "os_asia"
	rows = append(rows, foreign)
	stale := other
	stale.SavedMS = now - int64(31*24*time.Hour/time.Millisecond)
	rows = append(rows, stale)
	oneway := other
	oneway.ActorID = "oneway"
	oneway.Cards = []RoleCard{{"A", 1, true}, {"B", 3, true}}
	rows = append(rows, oneway)
	matches := roleCardMatches(self, rows, now)
	if len(matches) != 1 || len(matches[0].Give) != 1 || matches[0].Give[0] != "A" || matches[0].Receive[0] != "B" {
		t.Fatal(matches)
	}
	if _, err := parseRoleCards(map[string]any{"tarot_card_state": map[string]any{"list": []any{map[string]any{"name": "A", "is_unlock": true}}}}); err == nil {
		t.Fatal("missing count inferred")
	}
}
