// Package banners reads ZZZ's banner history from GachaClock online, as
// ZZZ-Plugin's poolHistory does.
package banners

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/RayleaBot/plugin-zzz/internal/app"
)

// dataURL is ZZZ-Plugin's DATA_URL.
const dataURL = "https://raw.githubusercontent.com/iaoongin/GachaClock/main/spider/data/zzz/history.json"

var cache = struct {
	sync.Mutex
	pools   []app.PoolInfo
	fetched time.Time
}{}

type entry struct {
	Type      string   `json:"type"`
	S         string   `json:"s"`
	A         []string `json:"a"`
	Version   string   `json:"version"`
	Timer     string   `json:"timer"`
	StartTime string   `json:"startTime"`
	EndTime   string   `json:"endTime"`
}

var versionHalf = regexp.MustCompile(`^([0-9.]+)(.*)$`)

func stamp(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "/", "-")
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02", time.RFC3339} {
		if t, err := time.Parse(layout, value); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
	}
	return ""
}

// Source reads GachaClock's history, kept for six hours; a failed read keeps
// the last answer.
func Source(ctx context.Context) ([]app.PoolInfo, string, error) {
	cache.Lock()
	defer cache.Unlock()
	if cache.pools != nil && time.Since(cache.fetched) < 6*time.Hour {
		return cache.pools, "GachaClock", nil
	}
	pools, err := fetch(ctx)
	if err != nil {
		if cache.pools != nil {
			return cache.pools, "GachaClock", nil
		}
		return nil, "", err
	}
	cache.pools, cache.fetched = pools, time.Now()
	return pools, "GachaClock", nil
}

func fetch(ctx context.Context) ([]app.PoolInfo, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, dataURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil || response.StatusCode != http.StatusOK {
		return nil, io.ErrUnexpectedEOF
	}
	var entries []entry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, err
	}
	return pools(entries), nil
}

// pools reads GachaClock entries the way the bundled snapshot was imported:
// a banner without a start begins at eleven the day after the previous one
// ends, and is marked as estimated.
func pools(entries []entry) []app.PoolInfo {
	type dated struct {
		entry    entry
		from, to string
		version  string
		half     string
	}
	list := []dated{}
	for _, item := range entries {
		from, to := stamp(item.StartTime), stamp(item.EndTime)
		if item.Timer != "" {
			parts := strings.Split(item.Timer, "~")
			from, to = stamp(parts[0]), stamp(parts[len(parts)-1])
		}
		match := versionHalf.FindStringSubmatch(item.Version)
		if to == "" || match == nil {
			continue
		}
		list = append(list, dated{item, from, to, match[1], match[2]})
	}
	slices.SortStableFunc(list, func(a, b dated) int { return strings.Compare(a.to, b.to) })
	pools := []app.PoolInfo{}
	for index, item := range list {
		pool := app.PoolInfo{Version: item.version, Half: item.half, From: item.from, To: item.to, Kind: "character", Characters5: []string{}, Characters4: []string{}, Weapons5: []string{}, Weapons4: []string{}}
		if pool.From == "" {
			if strings.HasPrefix(item.entry.Timer, "公测开启后") {
				pool.From = "2024-07-04 10:00:00"
			} else if index > 0 {
				// The latest banner that ended before this one; a character
				// and a W-Engine banner end together.
				latest := ""
				for _, other := range list[:index] {
					if other.to < item.to {
						latest = max(latest, other.to)
					}
				}
				previous, _ := time.Parse("2006-01-02 15:04:05", latest)
				pool.From = previous.AddDate(0, 0, 1).Format("2006-01-02") + " 11:00:00"
				pool.EstimatedStart = true
			}
		}
		five := []string{}
		if item.entry.S != "" {
			five = []string{item.entry.S}
		}
		if item.entry.Type == "角色" {
			pool.Characters5, pool.Characters4 = five, append([]string{}, item.entry.A...)
		} else {
			pool.Kind, pool.Weapons5, pool.Weapons4 = "weapon", five, append([]string{}, item.entry.A...)
		}
		pools = append(pools, pool)
	}
	return pools
}
