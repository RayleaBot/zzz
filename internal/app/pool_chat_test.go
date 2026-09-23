package app

import (
	"reflect"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// poolRecords are banners as the GachaClock source hands them over.
func poolRecords() []PoolRecord {
	day := func(month, date, hour int) time.Time {
		return time.Date(2024, time.Month(month), date, hour, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	}
	return []PoolRecord{
		{Type: "角色", Version: "1.0上半", S: "艾莲", A: []string{"安东", "苍角"}, Img: "https://example.com/1.png", Timer: "2024/07/04 10:00:00 ~ 2024/07/24 11:59:59", Start: day(7, 4, 10), End: day(7, 24, 12)},
		{Type: "武器", Version: "1.0上半", S: "深海访客", A: []string{"含羞恶面"}, Img: "https://example.com/2.png", Timer: "2024/07/04 10:00:00 ~ 2024/07/24 11:59:59", Start: day(7, 4, 10), End: day(7, 24, 12)},
		{Type: "角色", Version: "1.0下半", S: "朱鸢", A: []string{"妮可", "安东"}, Img: "https://example.com/3.png", Timer: "2024/07/24 12:00:00 ~ 2024/08/13 14:59:59", Start: day(7, 24, 12), End: day(8, 13, 15)},
		{Type: "角色", Version: "1.10", S: "青衣", A: []string{"苍角"}, Img: "https://example.com/4.png", Timer: "2024/08/14 11:00:00 ~ 2024/09/04 11:59:59", Start: day(8, 14, 11), End: day(9, 4, 12)},
	}
}

func segmentsText(segments []rayleabot.Segment) (text string, images []string) {
	for _, segment := range segments {
		if segment.Type == "image" {
			images = append(images, segment.Data["url"].(string))
		} else {
			text += segment.Data["text"].(string)
		}
	}
	return text, images
}

func TestPoolRepliesFollowZZZPlugin(t *testing.T) {
	records := poolRecords()
	now := time.Date(2024, 8, 20, 12, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	// 苍角 was on a banner last in the one still open, so it is left out.
	if got, want := poolSummary(records, "A", "角色", now), "【 A级代理人复刻统计 】\n安东:   6天未复刻\n妮可:   6天未复刻"; got != want {
		t.Errorf("summary = %q, want %q", got, want)
	}
	if got, want := poolSummary(records, "S", "武器", now), "【 S级音擎复刻统计 】\n深海访客:  27天未复刻"; got != want {
		t.Errorf("summary = %q, want %q", got, want)
	}
	if got, want := poolHistory(records, "安东"), "【 安东(A级代理人) 卡池记录 】\n1. 1.0上半 (2024/07/04 ~ 2024/07/24)\n2. 1.0下半 (2024/07/24 ~ 2024/08/13)"; got != want {
		t.Errorf("history = %q, want %q", got, want)
	}
	if got := poolHistory(records, "深海访客"); got != "【 深海访客(S级音擎) 卡池记录 】\n1. 1.0上半 (2024/07/04 ~ 2024/07/24)" {
		t.Errorf("history = %q", got)
	}
	if got := poolHistory(records, "雅"); got != "未找到【雅】卡池记录，请确保角色名称/别称存在" {
		t.Errorf("missing = %q", got)
	}

	text, images := segmentsText(currentPools(records, now))
	if want := "=== 📅 绝区零本期卡池 ===\n版本：v1.10\n时间：2024/08/14 11:00:00 ~ 2024/09/04 11:59:59\n时间：剩余约15天\n\n【 角色调频 】\n◈ S-青衣 | A-苍角\n"; text != want || !reflect.DeepEqual(images, []string{"https://example.com/4.png"}) {
		t.Errorf("current = %q %v", text, images)
	}
	if text, _ := segmentsText(currentPools(records, now.AddDate(1, 0, 0))); text != "当前没有正在进行的活动卡池。" {
		t.Errorf("none open = %q", text)
	}

	text, images = segmentsText(versionPools(records, "1.0", ""))
	if want := "【 绝区零 v1.0 卡池 】\n【 1.0上半 】\n⏱️ 2024/07/04 ~ 2024/07/24\n◈ 角色：S-艾莲 | A-安东，苍角\n◈ 音擎：S-深海访客 | A-含羞恶面\n【 1.0下半 】\n⏱️ 2024/07/24 ~ 2024/08/13\n◈ 角色：S-朱鸢 | A-妮可，安东\n"; text != want || len(images) != 3 {
		t.Errorf("version = %q %v", text, images)
	}
	if text, _ := segmentsText(versionPools(records, "1.0", "下半")); text != "【 绝区零 v1.0下半 卡池 】\n⏱️ 2024/07/24 ~ 2024/08/13\n◈ 角色：S-朱鸢 | A-妮可，安东\n" {
		t.Errorf("half = %q", text)
	}
	// Upstream would take 1.10 for 1.1 as well.
	if text, _ := segmentsText(versionPools(records, "1.1", "")); text != "未查询到绝区零1.1版本的卡池数据" {
		t.Errorf("1.1 = %q", text)
	}
}

func TestPoolNameGivesGachaClockNames(t *testing.T) {
	a := pluginApp(t)
	// The alias list keys 浮波柚叶 by its full name; GachaClock writes 柚叶.
	for word, want := range map[string]string{"柚叶": "柚叶", "伏波": "柚叶", "安比·德玛拉": "安比", "深海访客": "深海访客", "不存在": "不存在"} {
		if got := a.poolName(word, nil); got != want {
			t.Errorf("%s names %q, want %q", word, got, want)
		}
	}
}

func TestPoolInfosListGachaClock(t *testing.T) {
	pools := poolInfos(poolRecords())
	if len(pools) != 4 || pools[0].Version != "1.0" || pools[0].Half != "上半" || pools[0].From != "2024-07-04 10:00:00" || pools[1].Kind != "weapon" || pools[1].Weapons5[0] != "深海访客" || pools[3].Half != "" {
		t.Errorf("pools = %+v", pools)
	}
}
