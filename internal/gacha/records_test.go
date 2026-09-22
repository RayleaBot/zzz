package gacha

import (
	"errors"
	"reflect"
	"testing"
)

func fixture() Archive {
	return Archive{UID: "100000001", Region: "cn_gf01", Timezone: 8, Language: "zh-cn", Records: []Record{{ID: "1844674407370955101", GachaType: "301", UIGFType: "301", ItemID: "10000042", Name: "测试角色", Rank: "5", Time: "2026-09-01 08:00:00"}, {ID: "1844674407370955102", GachaType: "400", UIGFType: "301", ItemID: "10000043", Rank: "4", Time: "2026-09-01 08:00:00"}}}
}
func TestMergePreservesLargeIDsTimeAndSharedPoolPity(t *testing.T) {
	first := fixture()
	merged, added, err := Merge("genshin", Archive{}, first)
	if err != nil || added != 2 {
		t.Fatal(err)
	}
	again, added, err := Merge("genshin", merged, first)
	if err != nil || added != 0 || !reflect.DeepEqual(again.Records, first.Records) {
		t.Fatal("duplicate import changed records")
	}
	summary := Summarize("genshin", again)
	if len(summary) != 1 || summary[0].CurrentPity != 1 || summary[0].PityLowerBound || !summary[0].Rare[0].LowerBound {
		t.Fatal("incorrect history or shared pool handling")
	}
	first.Records[0].ItemID = "20001"
	if _, _, err := Merge("genshin", merged, first); !errors.Is(err, ErrConflict) {
		t.Fatal("conflicting record overwritten")
	}
}
func TestTransferCommitsOnlyAtFinishAndExportIsASnapshot(t *testing.T) {
	store := &Store{Directory: t.TempDir(), Game: "genshin"}
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
	other.Region = "os_usa"
	if _, _, err := Merge("genshin", first, other); !errors.Is(err, ErrConflict) {
		t.Fatal("regions merged")
	}
	other = fixture()
	other.Timezone = -5
	if _, _, err := Merge("genshin", first, other); !errors.Is(err, ErrConflict) {
		t.Fatal("timezone silently converted")
	}
}
