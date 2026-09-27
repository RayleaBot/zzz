package gacha

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func syncRecord() Record {
	return Record{ID: "100", ItemID: "1001", GachaType: "1", UIGFType: "1", Time: "2026-09-01 08:00:00", Rank: "5"}
}
func testSync(t *testing.T) (*Store, *Syncs, SyncInfo) {
	t.Helper()
	store := &Store{Directory: t.TempDir(), Game: "zzz"}
	jobs := &Syncs{}
	info, err := jobs.Start(store, SyncChoice{AccountRef: "account", RoleRef: "role"}, "100000001", "prod_gf_cn", false)
	if err != nil {
		t.Fatal(err)
	}
	return store, jobs, info
}
func pageOnce(_ context.Context, pool, end string, page int) (RemotePage, error) {
	result := RemotePage{Records: []Record{}, Timezone: 8, Language: "zh-cn", NextID: end}
	if pool == "1001" {
		result.Records = []Record{syncRecord()}
		result.NextID = "100"
	}
	return result, nil
}
func finishSync(t *testing.T, store *Store, jobs *Syncs, info SyncInfo, fetch FetchPage) SyncInfo {
	t.Helper()
	for info.State == "running" {
		var err error
		info, err = jobs.Step(t.Context(), store, info.Ref, info.Sequence, fetch)
		if err != nil {
			t.Fatal(err)
		}
	}
	return info
}
func TestSyncStagesPagesAndCommitsOnce(t *testing.T) {
	store, jobs, info := testSync(t)
	first, err := jobs.Step(t.Context(), store, info.Ref, 0, pageOnce)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Read("100000001", "prod_gf_cn"); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("partial sync wrote archive")
	}
	replayed, err := jobs.Step(t.Context(), store, info.Ref, 0, func(context.Context, string, string, int) (RemotePage, error) {
		t.Fatal("retried step fetched again")
		return RemotePage{}, nil
	})
	if err != nil || replayed.Sequence != first.Sequence || replayed.Fetched != 1 {
		t.Fatal("step replay changed progress")
	}
	done := finishSync(t, store, jobs, first, pageOnce)
	if done.Result.Added != 1 || done.Result.Total != 1 {
		t.Fatal("sync result incorrect")
	}
	if _, err := jobs.Step(t.Context(), store, info.Ref, done.Sequence-1, pageOnce); err != nil {
		t.Fatal("finish replay failed")
	}
}
func TestSyncFailureCancelAndDeletionPreserveArchives(t *testing.T) {
	for _, mode := range []string{"failure", "cancel", "delete", "import"} {
		t.Run(mode, func(t *testing.T) {
			store, jobs, info := testSync(t)
			first, err := jobs.Step(t.Context(), store, info.Ref, 0, pageOnce)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "failure":
				_, err = jobs.Step(t.Context(), store, info.Ref, first.Sequence, func(context.Context, string, string, int) (RemotePage, error) {
					return RemotePage{}, context.DeadlineExceeded
				})
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatal("transient error lost")
				}
				done := finishSync(t, store, jobs, first, pageOnce)
				if done.Result.Added != 1 {
					t.Fatal("resume failed")
				}
				return
			case "cancel":
				_ = jobs.Cancel(info.Ref)
			case "delete":
				_ = store.Remove("100000001", "prod_gf_cn")
			case "import":
				_, _, err = store.Import(Archive{UID: "100000001", Region: "prod_gf_cn", Timezone: 8, Records: []Record{syncRecord()}})
				if err != nil {
					t.Fatal(err)
				}
			}
			for first.State == "running" {
				first, err = jobs.Step(t.Context(), store, info.Ref, first.Sequence, pageOnce)
				if err != nil {
					break
				}
			}
			if mode == "cancel" && !errors.Is(err, ErrSync) || mode != "cancel" && !errors.Is(err, ErrConflict) {
				t.Fatal("stale sync committed")
			}
			archive, readErr := store.Read("100000001", "prod_gf_cn")
			if mode == "import" {
				if readErr != nil || len(archive.Records) != 1 {
					t.Fatal("imported archive changed")
				}
			} else if !errors.Is(readErr, os.ErrNotExist) {
				t.Fatal("canceled/deleted archive resurrected")
			}
		})
	}
}
func TestSyncKeepsOfficialUTCAndDistinctZZZReturnPools(t *testing.T) {
	store := &Store{Directory: t.TempDir(), Game: "zzz"}
	jobs := &Syncs{}
	info, err := jobs.Start(store, SyncChoice{AccountRef: "account", RoleRef: "role"}, "100000001", "prod_gf_cn", false)
	if err != nil {
		t.Fatal(err)
	}
	done := finishSync(t, store, jobs, info, func(_ context.Context, pool, end string, page int) (RemotePage, error) {
		result := RemotePage{Records: []Record{}, Timezone: 0, Language: "zh-cn", NextID: end}
		if pool == "2001" || pool == "12001" {
			record := syncRecord()
			record.UIGFType = ""
			record.Rank = "4"
			record.GachaType = "2"
			if pool == "12001" {
				record.GachaType = "102"
				record.ID = "101"
			}
			result.Records = []Record{record}
			result.NextID = record.ID
		}
		return result, nil
	})
	archive, err := store.Read("100000001", "prod_gf_cn")
	if err != nil || archive.Timezone != 0 || done.Result.Total != 2 || len(Summarize(archive)) != 2 {
		t.Fatal("UTC or independent pools lost")
	}
}

// While a sync waits for a page, starting another sync, reading its progress
// and canceling it answer at once; the page that arrives after the cancel is
// not merged.
func TestSyncsDoNotWaitForAPageBeingRead(t *testing.T) {
	store, jobs, info := testSync(t)
	reading, release := make(chan struct{}), make(chan struct{})
	stepped := make(chan error, 1)
	go func() {
		_, err := jobs.Step(context.Background(), store, info.Ref, 0, func(ctx context.Context, pool, end string, page int) (RemotePage, error) {
			close(reading)
			<-release
			return pageOnce(ctx, pool, end, page)
		})
		stepped <- err
	}()
	<-reading
	answered := make(chan error, 1)
	go func() {
		_, err := jobs.Start(store, SyncChoice{AccountRef: "account", RoleRef: "other"}, "100000002", "prod_gf_cn", false)
		if err == nil {
			_, err = jobs.Info(info.Ref)
		}
		if err == nil {
			err = jobs.Cancel(info.Ref)
		}
		answered <- err
	}()
	select {
	case err := <-answered:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		close(release)
		t.Fatal("Start, Info or Cancel waited for the page being read")
	}
	close(release)
	if err := <-stepped; !errors.Is(err, ErrSync) {
		t.Fatalf("the step canceled while reading ended with %v", err)
	}
	if _, err := store.Read("100000001", "prod_gf_cn"); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a canceled sync wrote the archive")
	}
}
