package banners

import (
	"encoding/json"
	"testing"
)

func TestPoolsReadGachaClockLikeTheBundledImport(t *testing.T) {
	var entries []entry
	raw := `[{"version":"1.0上半","type":"角色","s":"艾莲","a":["安比","妮可"],"timer":"公测开启后~2024/07/24 11:59"},
{"version":"1.0下半","type":"音擎","s":"深海访客","a":[],"timer":"~2024/08/14 15:59"},
{"version":"2.0上半","type":"角色","s":"仪玄","a":["橘福福"],"startTime":"2025-06-06 06:00","endTime":"2025-06-24 11:59"}]`
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		t.Fatal(err)
	}
	list := pools(entries)
	if len(list) != 3 || list[0].From != "2024-07-04 10:00:00" || list[0].Characters5[0] != "艾莲" || list[0].Half != "上半" {
		t.Fatalf("first = %+v", list[0])
	}
	if list[1].Kind != "weapon" || !list[1].EstimatedStart || list[1].From != "2024-07-25 11:00:00" || list[1].Weapons5[0] != "深海访客" {
		t.Fatalf("estimated = %+v", list[1])
	}
	if list[2].From != "2025-06-06 06:00:00" || list[2].To != "2025-06-24 11:59:00" {
		t.Fatalf("times = %+v", list[2])
	}
}
