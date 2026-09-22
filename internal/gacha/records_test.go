package gacha

import (
	"errors"
	"reflect"
	"testing"
)

func fixture() Archive {
	return Archive{UID: "10000001", Region: "prod_gf_cn", Timezone: 8, Language: "zh-cn", Records: []Record{{ID: "1844674407370955101", GachaType: "2", UIGFType: "2", ItemID: "1041", Name: "测试角色", Rank: "4", Time: "2026-09-01 08:00:00"}, {ID: "1844674407370955102", GachaType: "2", UIGFType: "2", ItemID: "1011", Rank: "3", Time: "2026-09-01 08:00:00"}}}
}
func TestMergePreservesLargeIDsTimeAndPity(t *testing.T) {
	first := fixture()
	merged, added, err := Merge(Archive{}, first)
	if err != nil || added != 2 {
		t.Fatal(err)
	}
	again, added, err := Merge(merged, first)
	if err != nil || added != 0 || !reflect.DeepEqual(again.Records, first.Records) {
		t.Fatal("duplicate import changed records")
	}
	summary := Summarize(again)
	if len(summary) != 1 || summary[0].CurrentPity != 1 || summary[0].PityLowerBound || !summary[0].Rare[0].LowerBound {
		t.Fatal("incorrect history handling")
	}
	first.Records[0].ItemID = "1021"
	if _, _, err := Merge(merged, first); !errors.Is(err, ErrConflict) {
		t.Fatal("conflicting record overwritten")
	}
}
func TestTransferCommitsOnlyAtFinishAndExportIsASnapshot(t *testing.T) {
	store := &Store{Directory: t.TempDir(), Game: "zzz"}
	transfers := &Transfers{}
	data := fixture()
	metadata := data
	metadata.Records = nil
	start, err := transfers.Start(metadata, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transfers.Append(start.Ref, 0, data.Records); err != nil {
		t.Fatal(err)
	}
	if _, err := transfers.Append(start.Ref, 0, data.Records); err != nil {
		t.Fatal("retry not idempotent")
	}
	if list, _ := store.List(); len(list) != 0 {
		t.Fatal("unfinished import changed storage")
	}
	result, err := transfers.Finish(start.Ref, store)
	if err != nil || result.Added != 2 {
		t.Fatal(err)
	}
	if again, err := transfers.Finish(start.Ref, store); err != nil || again != result {
		t.Fatal("finish retry was not idempotent")
	}
	saved, err := store.Read(result.UID, result.Region)
	if err != nil {
		t.Fatal(err)
	}
	export, err := transfers.Start(saved, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Remove(saved.UID, saved.Region); err != nil {
		t.Fatal(err)
	}
	items, more, err := transfers.Read(export.Ref, 0, 500)
	if err != nil || more || !reflect.DeepEqual(items, data.Records) {
		t.Fatal("export changed with live archive")
	}
}
func TestCrossRegionAndTimezoneAreNotMerged(t *testing.T) {
	first := fixture()
	other := fixture()
	other.Region = "prod_gf_us"
	if _, _, err := Merge(first, other); !errors.Is(err, ErrConflict) {
		t.Fatal("regions merged")
	}
	other = fixture()
	other.Timezone = -5
	if _, _, err := Merge(first, other); !errors.Is(err, ErrConflict) {
		t.Fatal("timezone silently converted")
	}
}
