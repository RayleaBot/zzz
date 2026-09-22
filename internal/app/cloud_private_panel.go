package app

import (
	"context"
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"strings"
)

func privateCloudPanelRequest(game string, input CloudInput) (string, map[string]any, error) {
	if input.owner == nil || input.owner.SourceProtocol != "onebot11" || input.owner.SourceAdapter == "" || input.owner.BotID == "" || !cloudQQPattern.MatchString(input.owner.ActorID) || !uidPattern.MatchString(input.UID) || !input.Consent {
		return "", nil, gameError("input_invalid", "长期云面板读取需要本人 OneBot11 私聊身份、UID 和明确同意。")
	}
	return "panel/data", map[string]any{"version": "0.1.0", "uid": input.UID, "type": cloudGame(game), "qq": input.owner.ActorID}, nil
}
func (c *CloudClient) pollPrivatePanel(game string, owner Subject, cancel bool) (CloudJob, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for ref, j := range c.jobs {
		if j.game == game && j.mode == "private_panel" && j.owner != nil && *j.owner == owner {
			return c.pollLocked(ref, cancel, &owner)
		}
	}
	return CloudJob{}, gameError("cloud_missing", "没有有效的本人云面板任务。")
}
func (a *App) privateCloudPanelCommand(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	if (a.Game.ID != "genshin" && a.Game.ID != "starrail") || event.Event.EventType != "message.private" || event.Event.SourceProtocol != "onebot11" {
		return event.SendText("长期云面板仅支持原神/星铁 OneBot11 本人私聊；需要先完成 ark 的 UID 签名验证。")
	}
	owner := Subject{SourceProtocol: event.Event.SourceProtocol, SourceAdapter: event.Event.SourceAdapter, BotID: event.Bot.ID, ActorID: event.Event.Actor.ID}
	if len(args) == 1 && strings.Contains("|进度|取消|保存|", "|"+args[0]+"|") {
		job, err := a.Cloud.pollPrivatePanel(a.Game.ID, owner, args[0] == "取消")
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		switch job.State {
		case "running":
			return event.SendText("云面板仍在读取，请稍后查询进度。")
		case "canceled":
			return event.SendText("本人云面板任务已取消。")
		case "failed":
			return event.SendText(job.Message)
		}
		if args[0] == "保存" {
			if job.Exchange == nil || len(job.exchange) == 0 {
				return event.SendText("此任务没有可保存的面板。")
			}
			client := a.accountClient(event)
			listed, err := client.List(ctx, 0)
			if err != nil {
				return event.SendText(friendlyError(err))
			}
			choice, role, err := Choose(listed, a.Game.ID, job.Exchange.UID)
			if err != nil {
				return event.SendText(friendlyError(err))
			}
			data, err := cloudPlayerObject(string(job.exchange))
			if err != nil {
				return event.SendText(friendlyError(err))
			}
			avatars, err := cleanCloudPlayer(a.Game.ID, role.UID, data)
			if err != nil {
				return event.SendText(friendlyError(err))
			}
			archive, err := a.CloudArchive.Read(client.Provider, choice)
			if err != nil {
				return event.SendText(friendlyError(err))
			}
			if err = a.CloudArchive.Update(client.Provider, choice, archive.Revision, role, avatars, false); err != nil {
				return event.SendText(friendlyError(err))
			}
			return event.SendText("本人云面板已保存为独立交换档案，可在本游戏管理页查看、导出或再次交换。")
		}
		if job.View == nil {
			return event.SendText("云服务未返回可展示的面板信息。")
		}
		return event.SendText(job.View.Text() + "\n如需本地保存，请发送“" + a.Game.Prefix + "云面板 保存”；只允许保存到本地已绑定的相同 UID。")
	}
	if len(args) != 3 || args[0] != "获取" || args[2] != "确认" {
		return event.SendText("长期云面板由 ark 提供，需要先完成云 UID 验证。发送“" + a.Game.Prefix + "云面板 获取 UID 确认”会向 ark 发送你的 QQ 与 UID，然后用“" + a.Game.Prefix + "云面板 进度/保存/取消”处理结果。")
	}
	_, err := a.Cloud.Start(a.Game, CloudInput{Mode: "private_panel", UID: args[1], Consent: true, owner: &owner})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText("本人云面板请求已开始，结果保留五分钟。请发送“" + a.Game.Prefix + "云面板 进度”查看。")
}
