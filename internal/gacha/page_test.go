package gacha

import (
	"errors"
	"testing"
)

func pageRecord(uid, pool, id string) map[string]any {
	return map[string]any{"uid": uid, "gacha_type": pool, "gacha_id": "9001", "id": id, "item_id": "1001", "time": "2026-09-01 08:00:00", "rank_type": "4", "count": "1"}
}

func TestParsePageConvertsPoolsAndTimezones(t *testing.T) {
	for _, tc := range []struct {
		game, region, pool, actual, want string
		data                             map[string]any
		zone                             int
	}{
		// ZZZ reports the zone relative to UTC+8 and names pools 1/2/3/5/102/103.
		{"zzz", "prod_gf_cn", "12001", "102", "102", map[string]any{"region_time_zone": 0.0}, 8},
		{"zzz", "prod_gf_us", "1001", "1001", "1", map[string]any{"region_time_zone": -13.0}, -5},
	} {
		data := tc.data
		data["list"] = []any{pageRecord("100000001", tc.actual, "1844674407370955101")}
		page, err := ParsePage("100000001", tc.region, tc.pool, "", data)
		if err != nil || len(page.Records) != 1 || page.Timezone != tc.zone || page.More || page.NextID != "1844674407370955101" {
			t.Fatalf("%s %s: %+v %v", tc.game, tc.region, page, err)
		}
		if page.Records[0].GachaType != tc.want {
			t.Fatalf("%s pool %s became %s", tc.game, tc.actual, page.Records[0].GachaType)
		}
	}
}

func TestParsePageRejectsRecordsOutsideTheRequest(t *testing.T) {
	mutations := map[string]func(map[string]any){
		"other uid":       func(r map[string]any) { r["uid"] = "other" },
		"repeated cursor": func(r map[string]any) { r["id"] = "100" },
		"other pool":      func(r map[string]any) { r["gacha_type"] = "2001" },
		"bad time":        func(r map[string]any) { r["time"] = "yesterday" },
		"bad item":        func(r map[string]any) { r["item_id"] = "x" },
	}
	for name, mutate := range mutations {
		record := pageRecord("100000001", "1001", "99")
		mutate(record)
		_, err := ParsePage("100000001", "prod_gf_cn", "1001", "100", map[string]any{"list": []any{record}})
		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("%s accepted: %v", name, err)
		}
	}
	list := []any{}
	for range 21 {
		list = append(list, pageRecord("100000001", "1001", "99"))
	}
	if _, err := ParsePage("100000001", "prod_gf_cn", "1001", "", map[string]any{"list": list}); !errors.Is(err, ErrInvalid) {
		t.Fatal("oversized page accepted")
	}
	if _, err := ParsePage("100000001", "prod_gf_cn", "1001", "", map[string]any{"list": []any{}, "region": "prod_gf_us"}); !errors.Is(err, ErrInvalid) {
		t.Fatal("page of another region accepted")
	}
}
