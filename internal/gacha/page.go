package gacha

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var recordDigits = regexp.MustCompile(`^[0-9]{1,19}$`)

// ZZZ record pools as the official log names them, mapped to archive pools.
var zzzPools = map[string]string{"1001": "1", "2001": "2", "3001": "3", "5001": "5", "12001": "102", "13001": "103"}

// ParsePage checks one page of official gacha records against the role and
// pool that were requested and converts it to archive records. The accounts
// plugin forwards the page as the official API returned it, without
// credential fields; every record check and the business timezone live here.
func ParsePage(game, uid, region, pool, endID string, data map[string]any) (RemotePage, error) {
	items, ok := data["list"].([]any)
	if !ok || len(items) > 20 {
		return RemotePage{}, fmt.Errorf("%w: gacha list", ErrInvalid)
	}
	if got := text(data["region"]); got != "" && got != region {
		return RemotePage{}, fmt.Errorf("%w: gacha region", ErrInvalid)
	}
	timezone, err := pageTimezone(game, region, data)
	if err != nil {
		return RemotePage{}, err
	}
	if endID == "" {
		endID = "0"
	}
	expected := pool
	if game == "zzz" {
		expected = zzzPools[pool]
	}
	page := RemotePage{Records: make([]Record, 0, len(items)), Timezone: timezone, Language: "zh-cn"}
	last := endID
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if text(item["uid"]) != uid {
			return RemotePage{}, fmt.Errorf("%w: gacha uid", ErrInvalid)
		}
		id := text(item["id"])
		if !recordDigits.MatchString(id) || id == "0" || last != "0" && !decimalLess(id, last) {
			return RemotePage{}, fmt.Errorf("%w: gacha cursor", ErrInvalid)
		}
		record := Record{ID: id, GachaType: text(item["gacha_type"]), GachaID: text(item["gacha_id"]), ItemID: text(item["item_id"]), Name: text(item["name"]), ItemType: text(item["item_type"]), Rank: text(item["rank_type"]), Count: text(item["count"]), Time: text(item["time"])}
		if game == "zzz" && zzzPools[record.GachaType] != "" {
			record.GachaType = zzzPools[record.GachaType]
		}
		// Genshin's second character event pool (400) shares pity with 301.
		if record.GachaType != expected && !(game == "genshin" && (pool == "301" || pool == "400") && (record.GachaType == "301" || record.GachaType == "400")) {
			return RemotePage{}, fmt.Errorf("%w: gacha pool", ErrInvalid)
		}
		if game == "genshin" {
			record.UIGFType = record.GachaType
			if record.UIGFType == "400" {
				record.UIGFType = "301"
			}
		}
		if !recordDigits.MatchString(record.ItemID) || game == "starrail" && !recordDigits.MatchString(record.GachaID) {
			return RemotePage{}, fmt.Errorf("%w: gacha item", ErrInvalid)
		}
		if _, err := time.Parse("2006-01-02 15:04:05", record.Time); err != nil {
			return RemotePage{}, fmt.Errorf("%w: gacha time", ErrInvalid)
		}
		page.Records = append(page.Records, record)
		last = id
	}
	page.NextID = last
	page.More = len(page.Records) == 20
	return page, nil
}

// pageTimezone is the UTC offset of the record times. The official field is
// used when present; ZZZ reports it relative to UTC+8. Without it, overseas
// regions fall back to their server offset and the Chinese servers to UTC+8.
func pageTimezone(game, region string, data map[string]any) (int, error) {
	if zone, exists := data["region_time_zone"]; exists {
		offset, err := strconv.Atoi(text(zone))
		if game == "zzz" {
			offset += 8
		}
		if err != nil || offset < -12 || offset > 14 {
			return 0, fmt.Errorf("%w: gacha timezone", ErrInvalid)
		}
		return offset, nil
	}
	switch {
	case strings.HasSuffix(region, "usa") || region == "prod_gf_us":
		return -5, nil
	case strings.HasSuffix(region, "euro") || strings.HasSuffix(region, "_eur") || region == "prod_gf_eu":
		return 1, nil
	}
	return 8, nil
}

func decimalLess(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}

func text(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case json.Number:
		return v.String()
	case int:
		return strconv.Itoa(v)
	case nil:
		return ""
	}
	return fmt.Sprint(value)
}
