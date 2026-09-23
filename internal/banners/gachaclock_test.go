package banners

import (
	"encoding/json"
	"testing"
	"time"
)

func TestProcessFollowsZZZPlugin(t *testing.T) {
	var entries []entry
	raw := `[{"version":"1.1上半","type":"角色","s":"青衣","a":["可琳","比利"],"img":"https://example.com/qingyi.png","timer":"1.1版本更新后 ~ 2024/09/04 11:59:59"},
{"version":"1.0下半","type":"角色","s":"朱鸢","a":["妮可","本"],"timer":"2024/07/24 12:00:00 ~ 2024/08/13 14:59:59"}]`
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		t.Fatal(err)
	}
	records, err := process(entries)
	if err != nil {
		t.Fatal(err)
	}
	// Upstream adds the launch banners GachaClock lacks and orders by end.
	if len(records) != 4 || records[0].S != "艾莲" || records[1].S != "深海访客" || records[2].S != "朱鸢" || records[3].S != "青衣" {
		t.Fatalf("records = %+v", records)
	}
	if records[0].Timer != "2024/07/04 10:00:00 ~ 2024/07/24 11:59:59" || !records[0].Start.Equal(time.Date(2024, 7, 4, 10, 0, 0, 0, zone)) || records[0].Estimated {
		t.Errorf("launch = %+v", records[0])
	}
	// 版本更新后 starts at eleven the day after the previous banner ends.
	if qingyi := records[3]; qingyi.Timer != "2024/08/14 11:00:00 ~ 2024/09/04 11:59:59" || !qingyi.Estimated || !qingyi.End.Equal(time.Date(2024, 9, 4, 11, 59, 59, 0, zone)) || qingyi.Img != "https://example.com/qingyi.png" {
		t.Errorf("estimated = %+v", qingyi)
	}
	// Without an earlier end the start cannot be inferred.
	if _, err := process([]entry{{Version: "1.0上半", Timer: "版本更新后 ~ 2024/07/01 11:59:59"}}); err == nil {
		t.Error("a banner without an earlier end was given a start")
	}
}
