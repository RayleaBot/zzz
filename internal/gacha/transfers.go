package gacha

import (
	"crypto/rand"
	"errors"
	"slices"
	"sync"
	"time"
)

var ErrTransfer = errors.New("gacha transfer expired or invalid")

type transfer struct {
	archive   Archive
	exporting bool
	expires   time.Time
	completed *ImportResult
}

type ImportResult struct {
	UID      string `json:"uid"`
	Region   string `json:"region"`
	Added    int    `json:"added"`
	Total    int    `json:"total"`
	Revision int64  `json:"revision"`
}
type Transfers struct {
	mu    sync.Mutex
	items map[string]*transfer
}

func (t *Transfers) Reset() { t.mu.Lock(); defer t.mu.Unlock(); t.items = nil }

type TransferInfo struct {
	Ref      string `json:"ref"`
	UID      string `json:"uid"`
	Region   string `json:"region"`
	Timezone int    `json:"timezone"`
	Language string `json:"lang"`
	Total    int    `json:"total"`
}

func (t *Transfers) Start(archive Archive, exporting bool) (TransferInfo, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.items == nil {
		t.items = map[string]*transfer{}
	}
	for ref, item := range t.items {
		if time.Now().After(item.expires) {
			delete(t.items, ref)
		}
	}
	if len(t.items) >= 8 {
		return TransferInfo{}, ErrTransfer
	}
	ref := rand.Text()
	copy := archive
	copy.Records = append([]Record{}, archive.Records...)
	t.items[ref] = &transfer{archive: copy, exporting: exporting, expires: time.Now().Add(10 * time.Minute)}
	return TransferInfo{Ref: ref, UID: archive.UID, Region: archive.Region, Timezone: archive.Timezone, Language: archive.Language, Total: len(archive.Records)}, nil
}
func (t *Transfers) get(ref string, exporting bool) (*transfer, error) {
	item := t.items[ref]
	if item == nil || time.Now().After(item.expires) || item.exporting != exporting {
		delete(t.items, ref)
		return nil, ErrTransfer
	}
	return item, nil
}
func (t *Transfers) Append(ref string, offset int, records []Record) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	item, err := t.get(ref, false)
	if err != nil {
		return 0, err
	}
	if item.completed != nil {
		return 0, ErrTransfer
	}
	if offset < 0 || len(records) > 500 || offset+len(records) > 200000 {
		return 0, ErrTransfer
	}
	if offset < len(item.archive.Records) {
		if offset+len(records) <= len(item.archive.Records) && slices.Equal(item.archive.Records[offset:offset+len(records)], records) {
			return len(item.archive.Records), nil
		}
		return 0, ErrConflict
	}
	if offset != len(item.archive.Records) {
		return 0, ErrTransfer
	}
	item.archive.Records = append(item.archive.Records, records...)
	return len(item.archive.Records), nil
}
func (t *Transfers) Finish(ref string, store *Store) (ImportResult, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	item, err := t.get(ref, false)
	if err != nil {
		return ImportResult{}, err
	}
	if item.completed != nil {
		return *item.completed, nil
	}
	archive, added, err := store.Import(item.archive)
	if err != nil {
		return ImportResult{}, err
	}
	result := ImportResult{UID: archive.UID, Region: archive.Region, Added: added, Total: len(archive.Records), Revision: archive.Revision}
	item.completed = &result
	item.archive.Records = nil
	return result, nil
}
func (t *Transfers) Read(ref string, offset, limit int) ([]Record, bool, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	item, err := t.get(ref, true)
	if err != nil {
		return nil, false, err
	}
	if offset < 0 || offset > len(item.archive.Records) {
		return nil, false, ErrTransfer
	}
	limit = min(max(limit, 1), 500)
	end := min(offset+limit, len(item.archive.Records))
	return append([]Record{}, item.archive.Records[offset:end]...), end < len(item.archive.Records), nil
}
func (t *Transfers) Close(ref string) { t.mu.Lock(); delete(t.items, ref); t.mu.Unlock() }
