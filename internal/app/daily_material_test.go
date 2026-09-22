package app

import (
	"testing"
	"time"
)

func TestDailyMaterialDayFollowsMiao(t *testing.T) {
	zone := time.FixedZone("UTC+8", 8*3600)
	monday := time.Date(2026, 9, 21, 10, 0, 0, 0, zone)
	cases := []struct {
		word string
		now  time.Time
		day  int
	}{
		{"今日素材", monday, 0},
		// The day turns at four in the morning.
		{"今日素材", time.Date(2026, 9, 21, 3, 59, 0, 0, zone), 6},
		{"明天素材", monday, 1},
		{"周三素材", monday, 2},
		{"周6材料", monday, 5},
		{"周日素材", monday, 6},
	}
	for _, c := range cases {
		if day := dailyMaterialDay(c.word, c.now); day != c.day {
			t.Errorf("%s at %s = %d, want %d", c.word, c.now.Format("15:04"), day, c.day)
		}
	}
}
