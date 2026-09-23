package app

import (
	"strings"
	"testing"
)

func TestChallengeReminderChatForms(t *testing.T) {
	for text, want := range map[string][3]int{"21:30": {21, 30, 0}, "每日20时": {20, 0, 0}, "每日8时30分": {8, 30, 0}, "每周一20时": {20, 0, 1}, "每周日9:05": {9, 5, 7}, "每周天9时": {9, 0, 7}} {
		hour, minute, weekday, ok := parseChallengeTime(text)
		if !ok || [3]int{hour, minute, weekday} != want {
			t.Errorf("%s: %d %d %d %v", text, hour, minute, weekday, ok)
		}
	}
	for _, text := range []string{"25:00", "每周八20时", "20", "每日20时70分"} {
		if _, _, _, ok := parseChallengeTime(text); ok {
			t.Errorf("%s accepted", text)
		}
	}
	kind := challengeKinds()[0]
	for _, name := range []string{kind.Metrics[0].Key, kind.Metrics[0].Label} {
		if metric, ok := challengeMetric(kind, name); !ok || metric.Key != kind.Metrics[0].Key {
			t.Errorf("metric %s", name)
		}
	}
	if challengeSchedule(20, 0, 0) != "每日 20:00" || challengeSchedule(9, 5, 7) != "每周日 09:05" {
		t.Error("schedule")
	}
	line := challengeReminderLine(Reminder{ChallengeKind: kind.ID, Metric: kind.Metrics[0].Key, Threshold: 3, Hour: 20, Weekday: 1, Enabled: true, LastCode: "notified", Role: Role{UID: "100000001"}})
	if !strings.Contains(line, kind.Label) || !strings.Contains(line, "每周一 20:00") || !strings.Contains(line, "已私聊通知") {
		t.Error(line)
	}
}
