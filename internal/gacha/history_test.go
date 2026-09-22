package gacha

import (
	"errors"
	"strconv"
	"testing"
)

func historyFixture(t *testing.T) Archive {
	t.Helper()
	a := fixture()
	a.Records = nil
	for i, rank := range []string{"4", "2", "", "4", "3", "4", "2"} {
		time := "2026-08-31 23:00:00"
		if i >= 4 {
			time = "2026-09-01 01:00:00"
		}
		a.Records = append(a.Records, Record{ID: strconv.Itoa(i + 1), ItemID: strconv.Itoa(100 + i%3), Name: "测试", GachaType: "2", UIGFType: "2", Rank: rank, Time: time})
	}
	result, _, err := Merge(Archive{}, a)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func TestHistoryPaginationFiltersAndUncertainIntervals(t *testing.T) {
	a := historyFixture(t)
	result, err := Browse(a, HistoryFilter{Limit: 2})
	if err != nil || result.Total != 7 || len(result.Records) != 2 || *result.NextOffset != 2 || result.Records[0].ID != "7" {
		t.Fatal(result, err)
	}
	if result.Analytics.TopRate != nil || result.Analytics.CompleteIntervals != 1 || *result.Analytics.AverageInterval != 2 || len(result.Analytics.Months) != 2 {
		t.Fatal(result.Analytics)
	}
	page, err := Browse(a, HistoryFilter{Limit: 2, Offset: 2, Revision: result.Revision})
	if err != nil || page.Records[1].ID != "4" || !page.Records[1].Interval.Uncertain {
		t.Fatal(page, err)
	}
	month, err := Browse(a, HistoryFilter{Pool: "2", Rank: "4", From: "2026-09-01", To: "2026-09-01"})
	if err != nil || month.Total != 1 || month.Records[0].Interval.Pulls != 2 || month.Records[0].Interval.LowerBound {
		t.Fatal(month, err)
	}
	a.Revision++
	if _, err := Browse(a, HistoryFilter{Offset: 2, Revision: result.Revision}); !errors.Is(err, ErrConflict) {
		t.Fatal("mixed archive revisions", err)
	}
	for _, f := range []HistoryFilter{{Offset: 2}, {Limit: 101}, {From: "2026-02-30"}, {Pool: "301"}, {From: "2026-09-01", To: "2026-08-01"}} {
		if _, err := Browse(a, f); !errors.Is(err, ErrInvalid) {
			t.Fatal("invalid filter accepted", f, err)
		}
	}
}
func TestUnknownRankDoesNotClaimCertainPity(t *testing.T) {
	a := historyFixture(t)
	a.Records = a.Records[:3]
	s := Summarize(a)[0]
	if !s.PityUncertain || s.PityLowerBound {
		t.Fatal(s)
	}
	a.Records = append(a.Records, historyFixture(t).Records[3])
	s = Summarize(a)[0]
	if s.PityUncertain || !s.Rare[1].Uncertain || !s.Rare[0].LowerBound {
		t.Fatal(s)
	}
}
func TestZZZNewChannelsAndLocalTimeRemainSeparate(t *testing.T) {
	a := Archive{UID: "10000001", Region: "prod_gf_cn", Timezone: 8, Records: []Record{}}
	for i, pool := range []string{"2", "102", "3", "103"} {
		a.Records = append(a.Records, Record{ID: strconv.Itoa(i + 1), ItemID: "101", GachaType: pool, Rank: "4", Time: "2026-09-01 00:01:00"})
	}
	result, err := Browse(a, HistoryFilter{Pool: "102"})
	if err != nil || result.Total != 1 || *result.Analytics.TopRate != 100 || result.Records[0].Interval.Pulls != 1 || result.Analytics.Months[0].Month != "2026-09" {
		t.Fatal(result, err)
	}
}
