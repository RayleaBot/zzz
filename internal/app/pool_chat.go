package app

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// ZZZ-Plugin's poolHistory answers four rules, in this order:
// <名称>复刻记录 (banner-history) gives the rarity summaries or one item's
// banners, 卡池 (banner-current) the banners open now, 卡池记录 (banner-all)
// every version in a forwarded message, and <版本>卡池 (banner-version) one
// version or half. Upstream keeps the forwarded message a week; here it is
// built from the day's data each time, so it never lags the other replies.

// poolSummaries are dispatchHandler's words for the rarity summaries.
var poolSummaries = []struct {
	words      *regexp.Regexp
	rank, kind string
}{
	{regexp.MustCompile(`(?i)^(五星|S级?)?(角色|代理人)$`), "S", "角色"},
	{regexp.MustCompile(`(?i)^(四星|A级?)?(角色|代理人)$`), "A", "角色"},
	{regexp.MustCompile(`(?i)^(五星|S级?)(武器|音擎)$`), "S", "武器"},
	{regexp.MustCompile(`(?i)^(四星|A级?)(武器|音擎)$`), "A", "武器"},
}

// poolTimes drops the clock from a banner's timer, as upstream shows it.
var poolTimes = regexp.MustCompile(` [0-9]{2}:[0-9]{2}:[0-9]{2}`)

// poolHalves reads the half a <版本>卡池 names.
var poolHalves = regexp.MustCompile(`(上半|下半)卡池$`)

// poolCommand answers the four rules; args are the name or version the
// trigger reads, and upstream answers no word after them.
func (a *App) poolCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	if len(args) != map[string]int{"banner-history": 1, "banner-version": 1}[command] || a.banners == nil {
		return event.Result(map[string]any{"handled": false})
	}
	records, _, err := a.banners(ctx)
	if err != nil {
		_, _ = event.Actions().LoggerWrite(ctx, rayleabot.LoggerWriteRequest{
			Level: "warn", Message: "卡池历史记录数据获取失败", Fields: map[string]any{"error": err.Error()},
		})
		return event.SendText("卡池历史记录数据获取失败，请稍后重试。")
	}
	send := func(segments []rayleabot.Segment) error {
		return event.Send(event.Event.Target.Type, event.Event.Target.ID, segments...)
	}
	now := time.Now()
	switch command {
	case "banner-history":
		name := strings.TrimSpace(args[0])
		for _, summary := range poolSummaries {
			if summary.words.MatchString(name) {
				return event.SendText(poolSummary(records, summary.rank, summary.kind, now))
			}
		}
		return event.SendText(poolHistory(records, a.poolName(name, a.aliasMap(event))))
	case "banner-current":
		return send(currentPools(records, now))
	case "banner-all":
		title := "绝区零全版本卡池记录"
		parts := [][]rayleabot.Segment{{rayleabot.Text(title)}}
		versions := []string{}
		for _, record := range records {
			version := strings.TrimSuffix(strings.TrimSuffix(record.Version, "上半"), "下半")
			if !slices.Contains(versions, version) {
				versions = append(versions, version)
			}
		}
		for _, version := range slices.Backward(versions) {
			parts = append(parts, versionPools(records, version, ""))
		}
		return a.sendForward(ctx, event, parts)
	}
	// banner-version
	half := ""
	if match := poolHalves.FindStringSubmatch(event.Event.Command()); match != nil {
		half = match[1]
	}
	return send(versionPools(records, args[0], half))
}

// poolName is ZZZ-Plugin's aliasToName: a character's name or alias gives the
// name GachaClock writes, and other words stay as written. Upstream returns
// the key of its alias list, which for five agents is the full name that
// GachaClock does not use (浮波柚叶 for 柚叶), so 柚叶复刻记录 found nothing;
// the short name of PartnerId2Data is used instead.
func (a *App) poolName(word string, aliases map[string]string) string {
	entry, _, found := a.aliasOwner(word, aliases)
	if !found || entry.Kind != "character" {
		return word
	}
	for _, record := range a.Game.Calc.Metadata().Characters {
		if partner, _ := record.Data["partner"].(map[string]any); record.ID == entry.ID && asText(partner["name"]) != "" {
			return asText(partner["name"])
		}
	}
	return entry.Name
}

// poolSummary is handleSummary: how long each S or A agent or W-Engine has
// gone without a banner, longest first, leaving out those on or after the
// current one.
func poolSummary(records []PoolRecord, rank, kind string, now time.Time) string {
	names := []string{}
	latest := map[string]PoolRecord{}
	for _, record := range records {
		if record.Type != kind || record.Start.IsZero() || record.End.IsZero() {
			continue
		}
		targets := record.A
		if rank == "S" {
			targets = nil
			if record.S != "" {
				targets = []string{record.S}
			}
		}
		for _, name := range targets {
			previous, seen := latest[name]
			if !seen {
				names = append(names, name)
			}
			if !seen || record.End.After(previous.End) {
				latest[name] = record
			}
		}
	}
	type waiting struct {
		name string
		days int
	}
	list := []waiting{}
	for _, name := range names {
		if end := latest[name].End; now.After(end) {
			list = append(list, waiting{name, int(now.Sub(end) / (24 * time.Hour))})
		}
	}
	slices.SortStableFunc(list, func(a, b waiting) int { return b.days - a.days })
	lines := []string{}
	for _, item := range list {
		lines = append(lines, fmt.Sprintf("%s: %3d天未复刻", item.name, item.days))
	}
	shown := "代理人"
	if kind == "武器" {
		shown = "音擎"
	}
	return "【 " + rank + "级" + shown + "复刻统计 】\n" + strings.Join(lines, "\n")
}

// poolHistory is handleHistoryQuery: every banner an agent or W-Engine was
// on, oldest first.
func poolHistory(records []PoolRecord, name string) string {
	found := []PoolRecord{}
	for _, record := range records {
		if record.S == name || slices.Contains(record.A, name) {
			found = append(found, record)
		}
	}
	if len(found) == 0 {
		return "未找到【" + name + "】卡池记录，请确保角色名称/别称存在"
	}
	kind, rarity := "代理人", "A级"
	if found[0].Type == "武器" {
		kind = "音擎"
	}
	if found[0].S == name {
		rarity = "S级"
	}
	lines := []string{}
	for index, record := range found {
		lines = append(lines, strconv.Itoa(index+1)+". "+record.Version+" ("+poolTimes.ReplaceAllString(record.Timer, "")+")")
	}
	return "【 " + name + "(" + rarity + kind + ") 卡池记录 】\n" + strings.Join(lines, "\n")
}

// currentPools is queryCurrentPool: the version and time of the banners open
// now with the days left, then the agent and W-Engine banners with their
// pictures.
func currentPools(records []PoolRecord, now time.Time) []rayleabot.Segment {
	active := []PoolRecord{}
	for _, record := range records {
		if !record.Start.IsZero() && !record.End.IsZero() && !now.Before(record.Start) && !now.After(record.End) {
			active = append(active, record)
		}
	}
	if len(active) == 0 {
		return []rayleabot.Segment{rayleabot.Text("当前没有正在进行的活动卡池。")}
	}
	sample := active[0]
	left := int(math.Ceil(float64(sample.End.Sub(now)) / float64(24*time.Hour)))
	segments := []rayleabot.Segment{rayleabot.Text("=== 📅 绝区零本期卡池 ===\n"), rayleabot.Text("版本：v" + sample.Version + "\n时间：" + sample.Timer + "\n时间：剩余约" + strconv.Itoa(left) + "天\n")}
	shown := 0
	for _, group := range []struct{ kind, title string }{{"角色", "\n【 角色调频 】\n"}, {"武器", "\n【 音擎调频 】\n"}} {
		pools := slices.DeleteFunc(slices.Clone(active), func(record PoolRecord) bool { return record.Type != group.kind })
		if len(pools) == 0 {
			continue
		}
		shown += len(pools)
		segments = append(segments, rayleabot.Text(group.title))
		for _, record := range pools {
			segments = append(segments, rayleabot.Text("◈ S-"+record.S+" | A-"+strings.Join(record.A, "，")+"\n"), rayleabot.Image(record.Img))
		}
	}
	if shown == 0 {
		return []rayleabot.Segment{rayleabot.Text("暂无卡池数据信息")}
	}
	return segments
}

// versionPools is generatePoolMsg: a version's banners by half, each half's
// time, then its agent and W-Engine banners with their pictures. Upstream
// takes every version the number begins, so 2.1 would also list 2.10; here
// the number must be followed by something other than a digit.
func versionPools(records []PoolRecord, version, half string) []rayleabot.Segment {
	pools := []PoolRecord{}
	stages := []string{}
	for _, record := range records {
		rest, found := strings.CutPrefix(record.Version, version)
		if !found || rest != "" && rest[0] >= '0' && rest[0] <= '9' || half != "" && !strings.Contains(record.Version, half) {
			continue
		}
		pools = append(pools, record)
		if !slices.Contains(stages, record.Version) {
			stages = append(stages, record.Version)
		}
	}
	if len(pools) == 0 {
		return []rayleabot.Segment{rayleabot.Text("未查询到绝区零" + version + half + "版本的卡池数据")}
	}
	slices.SortFunc(stages, func(a, b string) int {
		switch {
		case strings.Contains(a, "上半") && strings.Contains(b, "下半"):
			return -1
		case strings.Contains(a, "下半") && strings.Contains(b, "上半"):
			return 1
		}
		return strings.Compare(a, b)
	})
	title := version
	if len(stages) == 1 {
		title = stages[0]
	}
	segments := []rayleabot.Segment{rayleabot.Text("【 绝区零 v" + title + " 卡池 】\n")}
	for _, stage := range stages {
		stagePools := slices.DeleteFunc(slices.Clone(pools), func(record PoolRecord) bool { return record.Version != stage })
		if len(stages) > 1 {
			segments = append(segments, rayleabot.Text("【 "+stage+" 】\n"))
		}
		segments = append(segments, rayleabot.Text("⏱️ "+poolTimes.ReplaceAllString(stagePools[0].Timer, "")+"\n"))
		for _, group := range []struct{ kind, label string }{{"角色", "角色"}, {"武器", "音擎"}} {
			for _, record := range stagePools {
				if record.Type == group.kind {
					segments = append(segments, rayleabot.Text("◈ "+group.label+"：S-"+record.S+" | A-"+strings.Join(record.A, "，")+"\n"), rayleabot.Image(record.Img))
				}
			}
		}
	}
	return segments
}
