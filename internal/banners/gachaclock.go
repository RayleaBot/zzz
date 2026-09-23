// Package banners reads ZZZ's banner history from GachaClock online, as
// ZZZ-Plugin's poolHistory does.
package banners

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/RayleaBot/plugin-zzz/internal/app"
)

// dataURL is ZZZ-Plugin's DATA_URL.
const dataURL = "https://raw.githubusercontent.com/iaoongin/GachaClock/main/spider/data/zzz/history.json"

// As upstream, a read is kept for a day and a failed read keeps the last one.
var cache = struct {
	sync.Mutex
	records []app.PoolRecord
	fetched time.Time
}{}

// entry is a banner as GachaClock writes it.
type entry struct {
	Img     string   `json:"img"`
	Title   string   `json:"title"`
	Type    string   `json:"type"`
	Version string   `json:"version"`
	Timer   string   `json:"timer"`
	S       string   `json:"s"`
	A       []string `json:"a"`
}

// launchBanners are the 1.0上半 banners ZZZ-Plugin adds while GachaClock
// lacks them.
var launchBanners = []entry{
	{Img: "https://patchwiki.biligame.com/images/zzz/thumb/7/7f/8pesvtvchbs3t2jhqjhckd9k08pe7ui.png/900px-%E7%8B%AC%E5%AE%B6%E9%A2%91%E6%AE%B5001%E6%9C%9F.png", Title: "「慵懒逐浪」001期独家频段", Type: "角色", Version: "1.0上半", Timer: "公测开启后 ~ 2024/07/24 11:59:59", S: "艾莲", A: []string{"安东", "苍角"}},
	{Img: "https://patchwiki.biligame.com/images/zzz/thumb/3/32/gs2uajlo6v2h6pljzij84wdiwhu9fkj.png/900px-%E9%9F%B3%E6%93%8E%E9%A2%91%E6%AE%B5001%E6%9C%9F.png", Title: "「喧哗奏鸣」001期音擎频段", Type: "武器", Version: "1.0上半", Timer: "公测开启后 ~ 2024/07/24 11:59:59", S: "深海访客", A: []string{"含羞恶面", "旋钻机-赤轴"}},
}

// Source reads GachaClock's history the way ZZZ-Plugin's fetchData does.
func Source(ctx context.Context) ([]app.PoolRecord, string, error) {
	cache.Lock()
	defer cache.Unlock()
	if cache.records != nil && time.Since(cache.fetched) < 24*time.Hour {
		return cache.records, "GachaClock", nil
	}
	entries, err := fetch(ctx)
	var records []app.PoolRecord
	if err == nil {
		records, err = process(entries)
	}
	if err != nil {
		if cache.records != nil {
			return cache.records, "GachaClock", nil
		}
		return nil, "", err
	}
	cache.records, cache.fetched = records, time.Now()
	return records, "GachaClock", nil
}

func fetch(ctx context.Context) ([]entry, error) {
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, dataURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Cache-Control", "no-cache")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败：%d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	var entries []entry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// zone is the time zone ZZZ-Plugin's dates are read in, that of the mainland
// servers.
var zone = time.FixedZone("UTC+8", 8*3600)

// parse reads a date as JavaScript's Date does GachaClock's; zero when it is
// not one.
func parse(value string) time.Time {
	value = strings.ReplaceAll(strings.TrimSpace(value), "-", "/")
	for _, layout := range []string{"2006/01/02 15:04:05", "2006/01/02 15:04", "2006/01/02"} {
		if t, err := time.ParseInLocation(layout, value, zone); err == nil {
			return t
		}
	}
	return time.Time{}
}

// process is ZZZ-Plugin's processData: banners are ordered by their end, and
// a timer that starts with 公测开启后 or 版本更新后 gets its start, the latter
// at eleven the day after the previous banner ends.
func process(entries []entry) ([]app.PoolRecord, error) {
	if !slices.ContainsFunc(entries, func(e entry) bool { return e.Version == "1.0上半" }) {
		entries = append(entries, launchBanners...)
	}
	type dated struct {
		entry entry
		end   time.Time
	}
	list := []dated{}
	for _, item := range entries {
		parts := strings.Split(item.Timer, "~")
		end := time.Time{}
		if len(parts) >= 2 {
			end = parse(parts[1])
		}
		list = append(list, dated{item, end})
	}
	slices.SortStableFunc(list, func(a, b dated) int { return cmp.Compare(a.end.UnixMilli(), b.end.UnixMilli()) })
	records := []app.PoolRecord{}
	for index, item := range list {
		record := app.PoolRecord{Img: item.entry.Img, Title: item.entry.Title, Type: item.entry.Type, Version: item.entry.Version, Timer: item.entry.Timer, S: item.entry.S, A: item.entry.A}
		parts := strings.Split(item.entry.Timer, "~")
		start, end := strings.TrimSpace(parts[0]), ""
		if len(parts) >= 2 {
			end = strings.TrimSpace(parts[1])
		}
		switch {
		case strings.HasPrefix(item.entry.Timer, "公测开启后"):
			start = "2024/07/04 10:00:00"
			record.Timer = start + " ~ " + end
		case strings.Contains(item.entry.Timer, "版本更新后"):
			var previous time.Time
			for _, other := range slices.Backward(list[:index]) {
				if !other.end.IsZero() && other.end.Before(item.end) {
					previous = other.end
					break
				}
			}
			if previous.IsZero() {
				return nil, fmt.Errorf("无法根据“版本更新后”计算卡池起始时间，数据异常：%s %s", item.entry.Version, item.entry.Title)
			}
			start = previous.AddDate(0, 0, 1).Format("2006/01/02") + " 11:00:00"
			record.Timer, record.Estimated = start+" ~ "+end, true
		}
		record.Start, record.End = parse(start), parse(end)
		records = append(records, record)
	}
	return records, nil
}
