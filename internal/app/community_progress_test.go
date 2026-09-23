package app

import (
	"strings"
	"testing"
)

func TestCommunityTaskLineShowsProgress(t *testing.T) {
	task := Reminder{Kind: "community", Hour: 8, Enabled: true, Community: CommunityPlan{State: "running", Cursor: 3, Steps: make([]CommunityStep, 9), Remaining: "40"}}
	line := communityTaskLine(task, "1234****89")
	for _, part := range []string{"社区任务", "每天 08:00", "第 3/9 步", "还可获得 40 米游币"} {
		if !strings.Contains(line, part) {
			t.Errorf("%s lacks %s", line, part)
		}
	}
	task = Reminder{Kind: "cloudgame", Once: true, Hour: 8, LastCode: "plugin.upstream_cloud_unavailable"}
	if line := communityTaskLine(task, "x"); !strings.Contains(line, "一次") || !strings.Contains(line, "上次未完成") || !strings.Contains(line, "已停止") {
		t.Error(line)
	}
}
