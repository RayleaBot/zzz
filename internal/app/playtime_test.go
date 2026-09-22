package app

import "testing"

func TestPlaytimeSplitsMidnightAndMergesOverlappingWindows(t *testing.T) {
	days := estimatePlaytime([]BillingRow{{Time: "2026-09-20 00:03:00", Action: "recovery", Change: "1"}, {Time: "2026-09-20 00:05:00", Action: "recovery", Change: "1"}, {Time: "2026-09-20 00:05:00", Action: "recovery", Change: "1"}, {Time: "2026-09-20 12:00:00", Action: "recovery", Change: "10"}, {Time: "2026-09-20 13:00:00", Action: "other", Change: "1"}}, "recovery")
	if len(days) != 2 || days[0].Minutes != 5 || days[1].Minutes != 3 || days[1].Periods[0] != [2]int{1437, 1440} {
		t.Fatal(days)
	}
}
