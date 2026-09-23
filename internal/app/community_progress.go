package app

import (
	"cmp"
	"context"
	"fmt"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// communityProgress answers 社区任务进度: the sender's community and cloud
// game tasks, with the account, when each runs, how far the current round
// went and what the official page still offers.
func (a *App) communityProgress(ctx context.Context, event *rayleabot.EventContext) error {
	items, err := a.Reminders.List()
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	labels := map[string]string{}
	if listed, err := a.accountClient(event).List(ctx, 0); err == nil {
		for _, account := range listed.Items {
			labels[account.Ref] = account.AccountLabel
		}
	}
	owner := Subject{event.Event.SourceProtocol, event.Event.SourceAdapter, event.Bot.ID, event.Event.Actor.ID}
	lines := []string{}
	for _, task := range items {
		if task.Owner != owner || task.Kind != "community" && task.Kind != "cloudgame" {
			continue
		}
		lines = append(lines, communityTaskLine(task, cmp.Or(labels[task.AccountRef], "已不可用")))
	}
	if len(lines) == 0 {
		return event.SendText("没有社区任务。发送“" + a.Game.Prefix + "社区任务”查看用法。")
	}
	return event.SendText(strings.Join(lines, "\n"))
}

// communityTaskLine describes one account task for 社区任务进度.
func communityTaskLine(task Reminder, account string) string {
	zone := time.FixedZone("UTC+8", 28800)
	when := "每天"
	if task.Once {
		when = "一次"
	}
	plan := task.Community
	state := map[string]string{"": "等待执行", "running": fmt.Sprintf("进行中，第 %d/%d 步", plan.Cursor, len(plan.Steps)), "completed": "本轮已完成", "failed": "本轮未完成"}[plan.State]
	if task.Kind == "cloudgame" {
		state = map[bool]string{true: "等待执行", false: "上次已执行"}[task.LastCode == ""]
		if task.LastCode != "" && !strings.HasPrefix(task.LastCode, "cloud_") {
			state = "上次未完成"
		}
	}
	line := fmt.Sprintf("%s · 账号 %s · %s %02d:%02d · %s", map[string]string{"community": "社区任务", "cloudgame": "云游戏任务"}[task.Kind], account, when, task.Hour, task.Minute, state)
	if plan.Remaining != "" {
		line += " · 今日还可获得 " + plan.Remaining + " 米游币"
	}
	switch {
	case !task.Enabled:
		line += " · 已停止"
	case task.NextCheckMS > 0:
		line += " · 下次 " + time.UnixMilli(task.NextCheckMS).In(zone).Format("01-02 15:04")
	}
	if !task.Once && task.ExpiresAtMS > 0 {
		line += " · " + time.UnixMilli(task.ExpiresAtMS).In(zone).Format("01-02") + " 到期"
	}
	return line
}
