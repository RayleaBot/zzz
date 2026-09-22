package gacha

import (
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"
)

type Record struct {
	ID        string `json:"id"`
	GachaType string `json:"gacha_type"`
	UIGFType  string `json:"uigf_gacha_type,omitempty"`
	GachaID   string `json:"gacha_id,omitempty"`
	ItemID    string `json:"item_id"`
	Name      string `json:"name,omitempty"`
	ItemType  string `json:"item_type,omitempty"`
	Rank      string `json:"rank_type,omitempty"`
	Count     string `json:"count,omitempty"`
	Time      string `json:"time"`
}
type Archive struct {
	UID      string   `json:"uid"`
	Region   string   `json:"region"`
	Timezone int      `json:"timezone"`
	Language string   `json:"lang"`
	Records  []Record `json:"list"`
	Revision int64    `json:"revision"`
}

var digits = regexp.MustCompile(`^[0-9]{1,19}$`)
var pools = []string{"1", "2", "3", "5", "102", "103"}
var ErrInvalid = errors.New("invalid gacha archive")
var ErrConflict = errors.New("conflicting gacha record")

func Validate(archive Archive) error {
	if !digits.MatchString(archive.UID) || archive.Region == "" || len(archive.Region) > 64 || archive.Timezone < -12 || archive.Timezone > 14 || len(archive.Records) > 200000 {
		return ErrInvalid
	}
	for _, record := range archive.Records {
		if !digits.MatchString(record.ID) || !digits.MatchString(record.ItemID) || !slices.Contains(pools, record.GachaType) || len(record.Name) > 256 || len(record.ItemType) > 64 {
			return ErrInvalid
		}
		if record.Count != "" && record.Count != "1" {
			return ErrInvalid
		}
		if record.Rank != "" && !slices.Contains([]string{"2", "3", "4", "5"}, record.Rank) {
			return ErrInvalid
		}
		if _, err := time.Parse("2006-01-02 15:04:05", record.Time); err != nil {
			return ErrInvalid
		}
		for _, value := range []string{record.Name, record.ItemType} {
			for _, secret := range []string{"stoken=", "ltoken=", "cookie_token=", "authkey="} {
				if strings.Contains(strings.ToLower(value), secret) {
					return ErrInvalid
				}
			}
		}
	}
	return nil
}

func Merge(existing, incoming Archive) (Archive, int, error) {
	if err := Validate(incoming); err != nil {
		return Archive{}, 0, err
	}
	if existing.UID != "" && (existing.UID != incoming.UID || existing.Region != incoming.Region || existing.Timezone != incoming.Timezone) {
		return Archive{}, 0, ErrConflict
	}
	result := incoming
	result.Records = append([]Record{}, existing.Records...)
	byID := map[string]int{}
	for i, item := range result.Records {
		byID[item.GachaType+":"+item.ID] = i
	}
	added := 0
	for _, item := range incoming.Records {
		key := item.GachaType + ":" + item.ID
		if index, ok := byID[key]; ok {
			old := result.Records[index]
			if old.ItemID != item.ItemID || old.Time != item.Time || old.GachaType != item.GachaType || old.GachaID != item.GachaID || old.Rank != "" && item.Rank != "" && old.Rank != item.Rank {
				return Archive{}, 0, ErrConflict
			}
			if old.Name == "" {
				old.Name = item.Name
			}
			if old.ItemType == "" {
				old.ItemType = item.ItemType
			}
			if old.Rank == "" {
				old.Rank = item.Rank
			}
			if old.Count == "" {
				old.Count = item.Count
			}
			result.Records[index] = old
			continue
		}
		byID[key] = len(result.Records)
		result.Records = append(result.Records, item)
		added++
	}
	if len(result.Records) > 200000 {
		return Archive{}, 0, ErrInvalid
	}
	slices.SortFunc(result.Records, func(a, b Record) int {
		if a.Time != b.Time {
			return strings.Compare(a.Time, b.Time)
		}
		if len(a.ID) != len(b.ID) {
			return len(a.ID) - len(b.ID)
		}
		return strings.Compare(a.ID, b.ID)
	})
	result.Revision = time.Now().UnixNano()
	return result, added, nil
}

type Rare struct {
	ID         string `json:"id"`
	ItemID     string `json:"item_id"`
	Name       string `json:"name"`
	Time       string `json:"time"`
	Pulls      int    `json:"pulls"`
	LowerBound bool   `json:"lower_bound"`
	Uncertain  bool   `json:"uncertain"`
}
type PoolSummary struct {
	Pool           string `json:"pool"`
	Total          int    `json:"total"`
	Top            int    `json:"top"`
	CurrentPity    int    `json:"current_pity"`
	PityLowerBound bool   `json:"pity_lower_bound"`
	PityUncertain  bool   `json:"pity_uncertain"`
	UnknownRank    int    `json:"unknown_rank"`
	Rare           []Rare `json:"rare"`
}

func Summarize(archive Archive) []PoolSummary {
	groups := map[string]*PoolSummary{}
	highest := "4"
	for _, item := range archive.Records {
		pool := item.GachaType
		group := groups[pool]
		if group == nil {
			group = &PoolSummary{Pool: pool, PityLowerBound: true, Rare: []Rare{}}
			groups[pool] = group
		}
		group.Total++
		group.CurrentPity++
		if item.Rank == "" {
			group.UnknownRank++
			group.PityUncertain = true
		}
		if item.Rank == highest {
			group.Top++
			group.Rare = append(group.Rare, Rare{ID: item.ID, ItemID: item.ItemID, Name: item.Name, Time: item.Time, Pulls: group.CurrentPity, LowerBound: group.PityLowerBound, Uncertain: group.PityUncertain})
			group.CurrentPity = 0
			group.PityLowerBound = false
			group.PityUncertain = false
		}
	}
	result := make([]PoolSummary, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}
	slices.SortFunc(result, func(a, b PoolSummary) int { return strings.Compare(a.Pool, b.Pool) })
	return result
}
