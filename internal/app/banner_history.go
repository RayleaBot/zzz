package app

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/RayleaBot/plugin-zzz/internal/gacha"
)

type BannerAppearance struct {
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	Rarity       int    `json:"rarity"`
	Count        int    `json:"count"`
	LastEnd      string `json:"last_end"`
	DaysSinceEnd *int   `json:"days_since_end"`
}

func resourceVersion(game Game) string { return game.Data.Resources.Version }

// PoolRecord is one banner of GachaClock's history after ZZZ-Plugin's
// processData: Timer reads "start ~ end", its start filled in where GachaClock
// writes 公测开启后 or 版本更新后, and Start and End are those times in UTC+8,
// zero when the text is not a time. Estimated marks a start inferred from the
// previous banner's end.
type PoolRecord struct {
	Img       string
	Title     string
	Type      string
	Version   string
	Timer     string
	S         string
	A         []string
	Start     time.Time
	End       time.Time
	Estimated bool
}

// BannerSource reads a game's banner history online, as ZZZ-Plugin reads
// GachaClock, with the source's name.
type BannerSource func(ctx context.Context) ([]PoolRecord, string, error)

// bannerGame is the game with its banners from the plugin's online source
// when it answers, else the bundled snapshot; the shared data is not changed.
func (a *App) bannerGame() Game {
	game := a.Game
	if a.banners == nil || game.Data == nil {
		return game
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	records, version, err := a.banners(ctx)
	pools := poolInfos(records)
	if err != nil || len(pools) == 0 {
		return game
	}
	data := *game.Data
	data.Resources.Pools, data.Resources.Version = pools, version
	game.Data = &data
	return game
}

var versionHalf = regexp.MustCompile(`^([0-9.]+)(.*)$`)

// poolInfos lists GachaClock's banners the way the bundled snapshot lists
// them: the version apart from its half, and the times as UTC+8 text.
func poolInfos(records []PoolRecord) []PoolInfo {
	pools := []PoolInfo{}
	for _, record := range records {
		match := versionHalf.FindStringSubmatch(record.Version)
		if record.End.IsZero() || match == nil {
			continue
		}
		pool := PoolInfo{Version: match[1], Half: match[2], To: record.End.Format(time.DateTime), Kind: "character", EstimatedStart: record.Estimated, Characters5: []string{}, Characters4: []string{}, Weapons5: []string{}, Weapons4: []string{}}
		if !record.Start.IsZero() {
			pool.From = record.Start.Format(time.DateTime)
		}
		five := []string{}
		if record.S != "" {
			five = []string{record.S}
		}
		if record.Type == "角色" {
			pool.Characters5, pool.Characters4 = five, append([]string{}, record.A...)
		} else {
			pool.Kind, pool.Weapons5, pool.Weapons4 = "weapon", five, append([]string{}, record.A...)
		}
		pools = append(pools, pool)
	}
	return pools
}
func allBannerNames(p PoolInfo, kind string) []string {
	out := []string{}
	if kind != "weapon" {
		out = append(out, p.Characters5...)
		out = append(out, p.Characters4...)
	}
	if kind != "character" {
		out = append(out, p.Weapons5...)
		out = append(out, p.Weapons4...)
	}
	return out
}
func (a *App) bannerQuery(input map[string]any) (map[string]any, error) {
	var q struct {
		Version string `json:"version"`
		Query   string `json:"query"`
		Kind    string `json:"kind"`
		Date    string `json:"date"`
		Offset  int    `json:"offset"`
	}
	if decodeObject(input, &q) != nil || len(q.Query) > 128 || len(q.Version) > 32 || q.Offset < 0 || !slices.Contains([]string{"", "all", "character", "weapon"}, q.Kind) {
		return nil, gameError("input_invalid", "卡池筛选条件无效。")
	}
	date := time.Now().In(time.FixedZone("UTC+8", 28800))
	var err error
	if q.Date != "" {
		date, err = time.ParseInLocation("2006-01-02", q.Date, time.FixedZone("UTC+8", 28800))
		if err != nil {
			return nil, gameError("input_invalid", "日期格式应为 YYYY-MM-DD。")
		}
	}
	matches := []PoolInfo{}
	summary := map[string]*BannerAppearance{}
	query := strings.TrimSpace(q.Query)
	if entry, ok := a.Catalog.Resolve(query, "", nil); query != "" && ok {
		query = entry.Name
	}
	game := a.bannerGame()
	for _, p := range game.Data.Resources.Pools {
		if q.Version != "" && p.Version != q.Version {
			continue
		}
		names := allBannerNames(p, q.Kind)
		if len(names) == 0 || query != "" && !slices.ContainsFunc(names, func(n string) bool { return strings.Contains(n, query) }) {
			continue
		}
		if q.Date != "" && (p.From == "" || q.Date < p.From[:10] || q.Date > p.To[:10]) {
			continue
		}
		matches = append(matches, p)
		for _, group := range []struct {
			values []string
			kind   string
			rank   int
		}{{p.Characters5, "character", 5}, {p.Characters4, "character", 4}, {p.Weapons5, "weapon", 5}, {p.Weapons4, "weapon", 4}} {
			if q.Kind != "" && q.Kind != "all" && q.Kind != group.kind {
				continue
			}
			for _, name := range group.values {
				if query != "" && !strings.Contains(name, query) {
					continue
				}
				key := group.kind + ":" + name
				s := summary[key]
				if s == nil {
					s = &BannerAppearance{Name: name, Kind: group.kind, Rarity: group.rank}
					summary[key] = s
				}
				s.Count++
				if p.To > s.LastEnd {
					s.LastEnd = p.To
				}
			}
		}
	}
	appearances := []BannerAppearance{}
	for _, s := range summary {
		if end, err := time.ParseInLocation("2006-01-02 15:04:05", s.LastEnd, time.FixedZone("UTC+8", 28800)); err == nil && date.After(end) {
			days := int(date.Sub(end).Hours() / 24)
			s.DaysSinceEnd = &days
		}
		appearances = append(appearances, *s)
	}
	slices.SortFunc(appearances, func(a, b BannerAppearance) int {
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		return strings.Compare(a.Name, b.Name)
	})
	slices.SortFunc(matches, func(a, b PoolInfo) int { return strings.Compare(b.To, a.To) })
	total := len(matches)
	offset := min(q.Offset, total)
	end := min(offset+50, total)
	var next *int
	if end < total {
		next = &end
	}
	return map[string]any{"pools": matches[offset:end], "appearances": appearances, "total": total, "next_offset": next, "version": resourceVersion(game), "coverage": "fixed_reference", "date": date.Format("2006-01-02")}, nil
}

type VersionDrawCount struct {
	Unknown   int `json:"unknown"`
	end       string
	Version   string `json:"version"`
	Half      string `json:"half"`
	Total     int    `json:"total"`
	Top       int    `json:"top"`
	Estimated int    `json:"estimated"`
}

func versionDraws(game Game, archive gacha.Archive) map[string]any {
	return classifyVersionDraws(game, archive)
}
func classifyVersionDraws(g Game, archive gacha.Archive) map[string]any {
	periods := g.Data.Resources.Pools
	groups := map[string]*VersionDrawCount{}
	unclassified := 0
	top := "4"
	zone := time.FixedZone("archive", archive.Timezone*3600)
	cn := time.FixedZone("UTC+8", 28800)
	for _, record := range archive.Records {
		when, err := time.ParseInLocation("2006-01-02 15:04:05", record.Time, zone)
		if err != nil {
			unclassified++
			continue
		}
		stamp := when.In(cn).Format("2006-01-02 15:04:05")
		found := map[string]PoolInfo{}
		for _, p := range periods {
			if p.From != "" && stamp >= p.From && stamp <= p.To {
				key := p.Version + "/" + p.Half
				if old, ok := found[key]; !ok || old.EstimatedStart {
					found[key] = p
				}
			}
		}
		if len(found) != 1 {
			unclassified++
			continue
		}
		for key, p := range found {
			g := groups[key]
			if g == nil {
				g = &VersionDrawCount{Version: p.Version, Half: p.Half}
				groups[key] = g
			}
			g.Total++
			if !slices.Contains([]string{"2", "3", "4", "5"}, record.Rank) {
				g.Unknown++
			}
			if p.To > g.end {
				g.end = p.To
			}
			if record.Rank == top {
				g.Top++
			}
			if p.EstimatedStart {
				g.Estimated++
			}
		}
	}
	rows := []VersionDrawCount{}
	for _, g := range groups {
		rows = append(rows, *g)
	}
	slices.SortFunc(rows, func(a, b VersionDrawCount) int {
		if a.end != b.end {
			return strings.Compare(b.end, a.end)
		}
		return strings.Compare(b.Version+"/"+b.Half, a.Version+"/"+a.Half)
	})
	return map[string]any{"items": rows, "total": len(archive.Records), "unclassified": unclassified, "version": resourceVersion(g), "timezone": fmt.Sprintf("UTC%+d", archive.Timezone)}
}
