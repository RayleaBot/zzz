package gacha

import (
	"context"
	"crypto/rand"
	"errors"
	"sync"
	"time"
)

var ErrSync = errors.New("gacha sync expired or invalid")

type RemotePage struct {
	Records  []Record `json:"list"`
	NextID   string   `json:"next_end_id"`
	More     bool     `json:"has_more"`
	Timezone int      `json:"timezone"`
	Language string   `json:"lang"`
}
type FetchPage func(context.Context, string, string, int) (RemotePage, error)
type SyncInfo struct {
	Ref      string        `json:"ref"`
	State    string        `json:"state"`
	Sequence int           `json:"sequence"`
	Pool     string        `json:"pool"`
	Pages    int           `json:"pages"`
	Fetched  int           `json:"fetched"`
	Result   *ImportResult `json:"result,omitempty"`
}
type SyncChoice struct{ AccountRef, RoleRef string }
type syncJob struct {
	mu      sync.Mutex
	view    SyncInfo
	choice  SyncChoice
	archive Archive
	version uint64
	full    bool
	known   map[string]Record
	pools   []string
	pool    int
	page    int
	endID   string
	expires time.Time
}
type Syncs struct {
	mu   sync.Mutex
	jobs map[string]*syncJob
}

func (s *Syncs) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range s.jobs {
		j.mu.Lock()
		j.view.State = "canceled"
		j.archive.Records = nil
		j.known = nil
		j.mu.Unlock()
	}
	s.jobs = nil
}

var syncPools = map[string][]string{"genshin": {"100", "200", "301", "302", "400", "500"}, "starrail": {"1", "2", "11", "12", "21", "22"}, "zzz": {"1001", "2001", "3001", "5001", "12001", "13001"}}

func (s *Syncs) Start(store *Store, choice SyncChoice, uid, region string, full bool) (SyncInfo, error) {
	if choice.AccountRef == "" || choice.RoleRef == "" {
		return SyncInfo{}, ErrSync
	}
	archive, version, err := store.Snapshot(uid, region)
	if err != nil {
		return SyncInfo{}, err
	}
	incoming := Archive{UID: uid, Region: region, Timezone: 8, Language: "zh-cn", Records: []Record{}}
	if Validate(store.Game, incoming) != nil || len(syncPools[store.Game]) == 0 {
		return SyncInfo{}, ErrInvalid
	}
	known := map[string]Record{}
	for _, r := range archive.Records {
		known[Pool(store.Game, r.GachaType)+":"+r.ID] = r
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.jobs == nil {
		s.jobs = map[string]*syncJob{}
	}
	for ref, job := range s.jobs {
		job.mu.Lock()
		expired := time.Now().After(job.expires)
		job.mu.Unlock()
		if expired {
			delete(s.jobs, ref)
		}
	}
	if len(s.jobs) >= 8 {
		return SyncInfo{}, ErrSync
	}
	ref := rand.Text()
	view := SyncInfo{Ref: ref, State: "running", Pool: syncPools[store.Game][0]}
	s.jobs[ref] = &syncJob{view: view, choice: choice, archive: incoming, version: version, full: full, known: known, pools: syncPools[store.Game], page: 1, endID: "0", expires: time.Now().Add(15 * time.Minute)}
	return view, nil
}
func (s *Syncs) job(ref string) (*syncJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[ref]
	if job == nil {
		return nil, ErrSync
	}
	return job, nil
}
func (s *Syncs) Choice(ref string) (SyncChoice, error) {
	job, err := s.job(ref)
	if err != nil {
		return SyncChoice{}, err
	}
	job.mu.Lock()
	defer job.mu.Unlock()
	if time.Now().After(job.expires) {
		return SyncChoice{}, ErrSync
	}
	return job.choice, nil
}

func (s *Syncs) Info(ref string) (SyncInfo, error) {
	job, err := s.job(ref)
	if err != nil {
		return SyncInfo{}, err
	}
	job.mu.Lock()
	defer job.mu.Unlock()
	if time.Now().After(job.expires) || job.view.State == "canceled" {
		return SyncInfo{}, ErrSync
	}
	return job.view, nil
}

// Forget releases a task-owned buffer; interactive syncs keep their replay API.
func (s *Syncs) Forget(ref string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job := s.jobs[ref]; job != nil {
		job.mu.Lock()
		job.view.State = "canceled"
		job.archive.Records = nil
		job.known = nil
		job.mu.Unlock()
		delete(s.jobs, ref)
	}
}
func (s *Syncs) Cancel(ref string) error {
	job, err := s.job(ref)
	if err != nil {
		return err
	}
	job.mu.Lock()
	defer job.mu.Unlock()
	if job.view.State != "completed" {
		job.view.State = "canceled"
		job.archive.Records = nil
		job.known = nil
	}
	return nil
}
func (s *Syncs) Step(ctx context.Context, store *Store, ref string, sequence int, fetch FetchPage) (SyncInfo, error) {
	job, err := s.job(ref)
	if err != nil {
		return SyncInfo{}, err
	}
	job.mu.Lock()
	defer job.mu.Unlock()
	if time.Now().After(job.expires) || job.view.State == "canceled" {
		return SyncInfo{}, ErrSync
	}
	if sequence == job.view.Sequence-1 {
		return job.view, nil
	}
	if sequence != job.view.Sequence || job.view.State != "running" {
		return SyncInfo{}, ErrSync
	}
	if err := ctx.Err(); err != nil {
		return SyncInfo{}, err
	}
	// Fetch owns only this event's caller; never retain it after this step.
	page, err := fetch(ctx, job.pools[job.pool], job.endID, job.page)
	if err != nil {
		return SyncInfo{}, err
	}
	if page.Timezone < -12 || page.Timezone > 14 || page.Language != "zh-cn" || len(page.Records) > 20 || page.More && (len(page.Records) == 0 || page.NextID == job.endID || page.NextID == "0") {
		return SyncInfo{}, ErrInvalid
	}
	if len(job.archive.Records) > 0 && len(page.Records) > 0 && job.archive.Timezone != page.Timezone {
		return SyncInfo{}, ErrConflict
	}
	batch := job.archive
	if len(page.Records) > 0 {
		batch.Timezone = page.Timezone
	}
	batch.Records = page.Records
	if err := Validate(store.Game, batch); err != nil {
		return SyncInfo{}, err
	}
	if len(job.archive.Records)+len(page.Records) > 200000 || job.view.Pages >= 10000 {
		return SyncInfo{}, ErrInvalid
	}
	stop := !page.More
	for _, r := range page.Records {
		if old, exists := job.known[Pool(store.Game, r.GachaType)+":"+r.ID]; exists {
			if old.ItemID != r.ItemID || old.Time != r.Time || old.GachaType != r.GachaType || old.GachaID != r.GachaID {
				return SyncInfo{}, ErrConflict
			}
			if !job.full {
				stop = true
			}
		}
	}
	// Build candidate state first so a canceled request or failed final write
	// can retry the exact same page without duplicating its buffered records.
	if err := ctx.Err(); err != nil {
		return SyncInfo{}, err
	}
	if stop && job.pool+1 == len(job.pools) {
		candidate := job.archive
		if len(page.Records) > 0 {
			candidate.Timezone = page.Timezone
		}
		if len(job.archive.Records) == 0 && len(page.Records) == 0 {
			previous, _, err := store.Snapshot(job.archive.UID, job.archive.Region)
			if err != nil {
				return SyncInfo{}, err
			}
			if previous.UID != "" {
				candidate.Timezone = previous.Timezone
			}
		}
		candidate.Records = append(append([]Record{}, job.archive.Records...), page.Records...)
		merged, added, err := store.ImportIfUnchanged(candidate, job.version)
		if err != nil {
			return SyncInfo{}, err
		}
		job.view.State = "completed"
		job.view.Result = &ImportResult{UID: merged.UID, Region: merged.Region, Added: added, Total: len(merged.Records), Revision: merged.Revision}
		job.archive.Records = nil
		job.known = nil
	} else {
		if len(page.Records) > 0 {
			job.archive.Timezone = page.Timezone
		}
		job.archive.Records = append(job.archive.Records, page.Records...)
		if stop {
			job.pool++
			job.page = 1
			job.endID = "0"
		} else {
			job.page++
			job.endID = page.NextID
		}
		job.view.Pool = job.pools[job.pool]
	}
	job.view.Sequence++
	job.view.Pages++
	job.view.Fetched += len(page.Records)
	job.expires = time.Now().Add(15 * time.Minute)
	return job.view, nil
}
