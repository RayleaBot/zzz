package app

import (
	"context"
	"fmt"
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"strconv"
)

func (a *App) accountTaskCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	kind := "community"
	if command == "cloud-game-task" || command == "cloud-game-stop" {
		kind = "cloudgame"
	}
	stop := command == "community-stop" || command == "cloud-game-stop"
	if stop {
		items, err := a.Reminders.List()
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		owner := Subject{event.Event.SourceProtocol, event.Event.SourceAdapter, event.Bot.ID, event.Event.Actor.ID}
		count := 0
		for _, task := range items {
			if task.Kind == kind && task.Owner == owner {
				if _, err = a.removeDelegatedTask(ctx, event, task.Ref, kind); err != nil {
					return event.SendText(friendlyError(err))
				}
				count++
			}
		}
		return event.SendText(fmt.Sprintf("已停止你的 %d 个账号任务。", count))
	}
	usage := a.Game.Prefix + "社区任务 一次/每天 签到/全部 [账号序号]。全部包含浏览、点赞和分享，不自动取消点赞；每日默认北京时间 08:00，有效 30 天。"
	want := 2
	if kind == "cloudgame" {
		usage = a.Game.Prefix + "云游戏任务 一次/每天 [账号序号]，明确查询钱包并领取可能发放的时长。"
		want = 1
	}
	if len(args) < want || len(args) > want+1 || (args[0] != "一次" && args[0] != "每天") || (kind == "community" && args[1] != "签到" && args[1] != "全部") {
		return event.SendText("格式：" + usage)
	}
	index := 1
	if len(args) == want+1 {
		n, err := strconv.Atoi(args[want])
		if err != nil || n < 1 {
			return event.SendText("账号序号须为正整数。")
		}
		index = n
	}
	client := a.accountClient(event)
	listed, err := client.List(ctx, 0)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if index > len(listed.Items) {
		return event.SendText("没有对应账号，请先扫码登录。")
	}
	all := kind == "community" && args[1] == "全部"
	_, err = a.signinTask(ctx, event, kind+".task.create", map[string]any{"account_ref": listed.Items[index-1].Ref, "hour": 8, "days": 30, "once": args[0] == "一次", "read": all, "like": all, "share": all, "confirm": true, "notify": args[0] == "一次"})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText("任务已创建，每分钟最多执行一步；默认不重放结果不确定的写请求。可在游戏管理页查看进度，或发送“" + a.Game.Prefix + map[string]string{"community": "关闭社区任务", "cloudgame": "关闭云游戏任务"}[kind] + "”停止。")
}
