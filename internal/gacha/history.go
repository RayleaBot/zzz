package gacha

import (
	"slices"
	"strconv"
	"strings"
	"time"
)

type HistoryFilter struct {
	Pool     string `json:"pool"`
	Rank     string `json:"rank"`
	Query    string `json:"query"`
	From     string `json:"from"`
	To       string `json:"to"`
	Offset   int    `json:"offset"`
	Limit    int    `json:"limit"`
	Revision string `json:"revision"`
}
type HistoryRecord struct {
	Record
	Interval *Rare `json:"interval,omitempty"`
}
type MonthCount struct {
	Month   string `json:"month"`
	Total   int    `json:"total"`
	Top     int    `json:"top"`
	Unknown int    `json:"unknown"`
}
type ItemCount struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Rank  string `json:"rank"`
	Count int    `json:"count"`
}
type Analytics struct {
	Ranks             map[string]int `json:"ranks"`
	TopRate           *float64       `json:"top_rate"`
	CompleteIntervals int            `json:"complete_intervals"`
	AverageInterval   *float64       `json:"average_interval"`
	Months            []MonthCount   `json:"months"`
	Items             []ItemCount    `json:"items"`
	ItemKinds         int            `json:"item_kinds"`
	From              string         `json:"from"`
	To                string         `json:"to"`
}
type History struct {
	Records    []HistoryRecord `json:"records"`
	Total      int             `json:"total"`
	NextOffset *int            `json:"next_offset"`
	Revision   string          `json:"revision"`
	Analytics  Analytics       `json:"analytics"`
}

func (f HistoryFilter) Validate() error {
	if f.Offset < 0 || f.Offset > 200000 || f.Limit < 0 || f.Limit > 100 || len(f.Query) > 128 || !slices.Contains([]string{"", "unknown", "2", "3", "4", "5"}, f.Rank) {
		return ErrInvalid
	}
	if f.Pool != "" && !slices.Contains(pools, f.Pool) {
		return ErrInvalid
	}
	for _, date := range []string{f.From, f.To} {
		if date != "" {
			if _, err := time.Parse("2006-01-02", date); err != nil {
				return ErrInvalid
			}
		}
	}
	if f.From != "" && f.To != "" && f.From > f.To {
		return ErrInvalid
	}
	if f.Offset > 0 && f.Revision == "" {
		return ErrInvalid
	}
	return nil
}
func Browse(archive Archive, f HistoryFilter) (History, error) {
	out := History{Records: []HistoryRecord{}, Revision: strconv.FormatInt(archive.Revision, 10), Analytics: Analytics{Ranks: map[string]int{}, Months: []MonthCount{}, Items: []ItemCount{}}}
	if err := f.Validate(); err != nil {
		return out, err
	}
	if err := Validate(archive); err != nil {
		return out, err
	}
	if f.Revision != "" && f.Revision != out.Revision {
		return out, ErrConflict
	}
	if f.Limit == 0 {
		f.Limit = 50
	}
	intervals := map[string]Rare{}
	for _, pool := range Summarize(archive) {
		for _, rare := range pool.Rare {
			intervals[pool.Pool+":"+rare.ID] = rare
		}
	}
	highest := "4"
	months := map[string]*MonthCount{}
	items := map[string]*ItemCount{}
	filtered := []HistoryRecord{}
	totalInterval, top := 0, 0
	for _, record := range archive.Records {
		if f.Pool != "" && f.Pool != record.GachaType {
			continue
		}
		rank := record.Rank
		if rank == "" {
			rank = "unknown"
		}
		if f.Rank != "" && f.Rank != rank {
			continue
		}
		if f.Query != "" && !strings.Contains(strings.ToLower(record.Name), strings.ToLower(f.Query)) && !strings.Contains(record.ItemID, f.Query) {
			continue
		}
		date := record.Time[:10]
		if f.From != "" && date < f.From || f.To != "" && date > f.To {
			continue
		}
		item := HistoryRecord{Record: record}
		if rare, ok := intervals[record.GachaType+":"+record.ID]; ok {
			item.Interval = &rare
			if !rare.LowerBound && !rare.Uncertain {
				totalInterval += rare.Pulls
				out.Analytics.CompleteIntervals++
			}
		}
		filtered = append(filtered, item)
		out.Analytics.Ranks[rank]++
		if out.Analytics.From == "" {
			out.Analytics.From = record.Time
		}
		out.Analytics.To = record.Time
		month := date[:7]
		if months[month] == nil {
			months[month] = &MonthCount{Month: month}
		}
		months[month].Total++
		if rank == highest {
			top++
			months[month].Top++
		}
		if rank == "unknown" {
			months[month].Unknown++
		}
		key := record.ItemID + ":" + rank
		if items[key] == nil {
			items[key] = &ItemCount{ID: record.ItemID, Name: record.Name, Rank: rank}
		}
		items[key].Count++
	}
	out.Total = len(filtered)
	if out.Total > 0 && out.Analytics.Ranks["unknown"] == 0 {
		value := float64(top) * 100 / float64(out.Total)
		out.Analytics.TopRate = &value
	}
	if out.Analytics.CompleteIntervals > 0 {
		value := float64(totalInterval) / float64(out.Analytics.CompleteIntervals)
		out.Analytics.AverageInterval = &value
	}
	for _, month := range months {
		out.Analytics.Months = append(out.Analytics.Months, *month)
	}
	slices.SortFunc(out.Analytics.Months, func(a, b MonthCount) int { return strings.Compare(b.Month, a.Month) })
	for _, item := range items {
		out.Analytics.Items = append(out.Analytics.Items, *item)
	}
	slices.SortFunc(out.Analytics.Items, func(a, b ItemCount) int {
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		if a.ID != b.ID {
			return strings.Compare(a.ID, b.ID)
		}
		return strings.Compare(a.Rank, b.Rank)
	})
	out.Analytics.ItemKinds = len(out.Analytics.Items)
	if len(out.Analytics.Items) > 100 {
		out.Analytics.Items = out.Analytics.Items[:100]
	}
	for i := f.Offset; i < min(out.Total, f.Offset+f.Limit); i++ {
		out.Records = append(out.Records, filtered[out.Total-1-i])
	}
	if f.Offset+len(out.Records) < out.Total {
		next := f.Offset + len(out.Records)
		out.NextOffset = &next
	}
	return out, nil
}
