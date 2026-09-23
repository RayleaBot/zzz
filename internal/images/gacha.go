package images

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"sort"
	"strconv"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/app"
	"github.com/RayleaBot/plugin-zzz/internal/gacha"
)

const timeLayout = "2006-01-02 15:04:05"

// chinaTime is the time zone upstream compares gacha times in.
var chinaTime = time.FixedZone("UTC+8", 8*3600)

// gachaPools are ZZZ-Plugin's channels in its display order, with the pool
// codes they are stored under and the pull counts it colours green and red.
var gachaPools = []struct {
	name, code  string
	good, bad   int
	upLevels    []float64
	usesAverage bool
}{
	{"独家频段", "2", 60, 80, []float64{74, 87, 99, 105, 120}, false},
	{"独家重映", "102", 60, 80, []float64{74, 87, 99, 105, 120}, false},
	{"音擎频段", "3", 50, 70, []float64{62, 75, 88, 99, 111}, false},
	{"音擎回响", "103", 50, 70, []float64{62, 75, 88, 99, 111}, false},
	{"常驻频段", "1", 60, 80, []float64{53, 60, 68, 73, 75}, true},
	{"邦布频段", "5", 50, 70, []float64{51, 55, 61, 68, 70}, false},
}

// standardItems never count as the rate-up upstream.
var standardItems = []string{"「11号」", "猫又", "莱卡恩", "丽娜", "格莉丝", "珂蕾妲", "拘缚者", "燃狱齿轮", "嵌合编译器", "钢铁肉垫", "硫磺石", "啜泣摇篮"}

// offBannerLimited are limited items that can be pulled off-banner; upstream
// counts them as not rate-up after 2026-07-29.
var offBannerLimited = []string{"1221", "1071", "1241", "14122", "14107", "14124"}

var (
	luckTags   = []string{"非到极致", "运气不好", "平稳保底", "小欧一把", "欧狗在此"}
	luckEmojis = [][]int{{4, 8, 13}, {1, 10, 5}, {16, 15, 2}, {12, 3, 9}, {6, 14, 7}}
)

// luckLevel is upstream's getLevelFromList: the first threshold the average
// stays under decides the level, four being the luckiest.
func luckLevel(average float64, thresholds []float64) int {
	if average == 0 {
		return 2
	}
	for index, threshold := range thresholds {
		if average <= threshold {
			return 4 - index
		}
	}
	return 0
}

// Gacha draws the gacha analysis the way ZZZ-Plugin's gachalog does: the
// player card, then each channel with pulls since the last S rank, averages,
// the luck tag and every S-rank item with its pull count.
func Gacha(context app.ImageContext, image app.GachaImage) (app.Image, bool) {
	if len(image.Archive.Records) == 0 {
		return app.Image{}, false
	}
	resources := []rayleabot.RenderImageResource{}
	for _, item := range append(append([][2]string{}, commonArtwork...), gachaArtwork...) {
		if resource, ok := context.ArtworkResource(item[0], "zzz-plugin", item[1]); ok {
			resources = append(resources, resource)
		}
	}
	resources = append(resources, fontResources(context)...)
	maps := readMaps(context)
	icons := map[string]string{}
	icon := func(record gacha.Record) string {
		if id, seen := icons[record.ItemID]; seen {
			return id
		}
		id := "item-" + record.ItemID
		var resource rayleabot.RenderImageResource
		ok := false
		switch record.ItemType {
		case "音擎":
			if code := maps.weapons[record.ItemID].CodeName; code != "" {
				resource, ok = context.FetchArtworkResource(id, "zzzerouid", "weapon/"+code+"_High.png")
			}
		case "邦布":
			resource, ok = context.FetchArtworkResource(id, "zzzerouid", "square_bangbo/bangboo_rectangle_avatar_"+record.ItemID+".png")
		default:
			resource, ok = context.FetchArtworkResource(id, "mys-zzz", "role_square_avatar/role_square_avatar_"+record.ItemID+".png")
		}
		if ok {
			resources = append(resources, resource)
		} else {
			id = ""
		}
		icons[record.ItemID] = id
		return id
	}
	cutoff := time.Date(2026, 7, 29, 0, 0, 0, 0, chinaTime)
	channels := []any{}
	for index, pool := range gachaPools {
		records := []gacha.Record{}
		for _, record := range image.Archive.Records {
			if record.GachaType == pool.code {
				records = append(records, record)
			}
		}
		// Upstream reads each channel newest first.
		sort.Slice(records, func(i, j int) bool {
			if len(records[i].ID) != len(records[j].ID) {
				return len(records[i].ID) > len(records[j].ID)
			}
			return records[i].ID > records[j].ID
		})
		type item struct {
			id    string
			up    bool
			count string
			color string
		}
		list := []item{}
		lastFive, previous := -1, 0
		colour := func(pulls int) string {
			switch {
			case pulls <= pool.good:
				return "rgb(63, 255, 0)"
			case pulls >= pool.bad:
				return "rgb(255, 20, 20)"
			}
			return "white"
		}
		for position, record := range records {
			if record.Rank == "4" {
				up := !slices.Contains(standardItems, record.Name)
				if at, err := time.ParseInLocation(timeLayout, record.Time, chinaTime); err == nil && at.After(cutoff) && slices.Contains(offBannerLimited, record.ItemID) {
					up = false
				}
				if lastFive < 0 {
					lastFive = position
				}
				if len(list) > 0 {
					list[len(list)-1].count, list[len(list)-1].color = strconv.Itoa(position-previous), colour(position-previous)
				}
				list = append(list, item{id: icon(record), up: up, count: "-", color: "white"})
				previous = position
			}
			if position == len(records)-1 && len(list) > 0 {
				list[len(list)-1].count, list[len(list)-1].color = strconv.Itoa(position-previous+1), colour(position-previous+1)
			}
		}
		total, fives, ups := len(records), len(list), 0
		for _, entry := range list {
			if entry.up {
				ups++
			}
		}
		timeRange, avgFive, avgUp, noWai := "还没有抽卡", "-", "-", "-"
		var five, rateUp float64
		if total > 0 {
			timeRange = records[0].Time + " ～ " + records[total-1].Time
			if lastFive >= 0 {
				// Upstream works on the averages rounded to one decimal.
				if fives > 0 {
					avgFive = fmt.Sprintf("%.1f", float64(total-lastFive)/float64(fives))
					five, _ = strconv.ParseFloat(avgFive, 64)
				}
				if ups > 0 {
					avgUp = fmt.Sprintf("%.1f", float64(total-lastFive)/float64(ups))
					rateUp, _ = strconv.ParseFloat(avgUp, 64)
				}
				if five != 0 && rateUp != 0 {
					noWai = fmt.Sprintf("%.0f%%", (2-rateUp/five)*100)
				}
			}
		}
		last := strconv.Itoa(lastFive)
		if lastFive < 0 {
			last = "-"
			if total > 0 {
				last = strconv.Itoa(total)
			}
		}
		level := 2
		if pool.usesAverage && avgFive != "-" {
			level = luckLevel(five, pool.upLevels)
		} else if !pool.usesAverage && avgUp != "-" {
			level = luckLevel(rateUp, pool.upLevels)
		}
		emojis := luckEmojis[level]
		cards := []any{}
		for _, entry := range list {
			cards = append(cards, map[string]any{"icon": entry.id, "up": entry.up, "count": entry.count, "color": entry.color})
		}
		channels = append(channels, map[string]any{
			"name": pool.name, "band": index/2 + 1, "last": last, "time_range": timeRange,
			"avg_five": avgFive, "avg_up": avgUp, "total": total, "no_wai": noWai,
			"show_no_wai": pool.code != "1" && pool.code != "5",
			"tag":         luckTags[level], "emoji": emojis[rand.IntN(len(emojis))], "cards": cards,
		})
	}
	region := regionNames[image.Role.Region]
	if region == "" {
		region = image.Role.Region
	}
	return app.Image{Template: "gacha", Data: map[string]any{
		"player":   map[string]any{"nickname": image.Role.Nickname, "level": image.Role.Level, "region": region, "uid": image.UID},
		"channels": channels,
	}, Resources: resources}, true
}
