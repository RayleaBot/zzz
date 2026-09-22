package app

import (
	"regexp"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

var cloudQQPattern = regexp.MustCompile(`^[1-9][0-9]{4,11}$`)
var cloudCodePattern = regexp.MustCompile(`^[A-Za-z0-9_:\-]{1,128}$`)

func cloudVerifyRequest(game string, input CloudInput) (string, map[string]any, error) {
	if input.owner == nil || input.owner.SourceProtocol != "onebot11" || input.owner.SourceAdapter == "" || input.owner.BotID == "" || !cloudQQPattern.MatchString(input.owner.ActorID) || !uidPattern.MatchString(input.UID) || !input.Consent {
		return "", nil, gameError("input_invalid", "云验证需要本人 OneBot11 私聊身份、有效 UID 和明确确认。")
	}
	body := map[string]any{"version": "0.1.0", "uid": input.UID, "type": cloudGame(game)}
	if input.Mode == "verify_code" {
		return "verify/code", body, nil
	}
	body["qq"] = input.owner.ActorID
	if input.Mode == "verify_bind" {
		return "verify", body, nil
	}
	if input.Mode == "verify_status" {
		return "verify/user", body, nil
	}
	return "", nil, gameError("operation_denied", "云验证操作不存在。")
}
func cloudVerifyResult(game Game, input CloudInput, result map[string]any) (CloudResult, error) {
	code := asText(result["retcode"])
	if code != "100" {
		messages := map[string]string{"200": "服务尚未确认此 QQ 与 UID 的关联。", "302": "游戏签名与验证码不符，请核对并等待签名生效后重试。", "304": "此 UID 尚未完成号主验证。", "305": "验证已过期，请重新获取验证码。", "306": "云服务暂未读到游戏签名，请稍后重试。", "307": "云服务尚未收录此 UID。", "104": "云服务查询过于频繁，请稍后重试。", "201": "云服务查询过于频繁，请五分钟后重试。"}
		message := messages[code]
		if message == "" {
			message = "云服务未能完成验证，请稍后检查状态。"
		}
		return CloudResult{}, gameError("cloud_verification_failed", message)
	}
	v := View{Title: game.Name + "云 UID 验证", Subtitle: "UID " + input.UID, Rows: []Row{}, Note: "第三方 ark.ivny.cn 的跨机器人关联，不改变本地账号授权。"}
	switch input.Mode {
	case "verify_code":
		value := asText(asObject(result["data"])["verifyCode"])
		if !cloudCodePattern.MatchString(value) {
			return CloudResult{}, gameError("cloud_invalid", "验证码格式暂不兼容，请稍后重新获取。")
		}
		v.Rows = append(v.Rows, Row{Label: "签名验证码", Value: value})
		v.Note = "将验证码设为游戏公开签名，等待签名审核/生效后，在本私聊发送“" + game.Prefix + "云验证 提交 " + input.UID + " 确认”。验证码由服务提供，有效期 24 小时；无需发送 CK。"
	case "verify_bind":
		v.Rows = append(v.Rows, Row{Label: "验证状态", Value: "服务已确认本人 QQ 与 UID 的关联"})
	case "verify_status":
		v.Rows = append(v.Rows, Row{Label: "验证状态", Value: "本人 QQ 与 UID 已关联"})
	default:
		return CloudResult{}, gameError("operation_denied", "云验证操作不存在。")
	}
	return CloudResult{View: &v}, nil
}
func (c *CloudClient) pollVerification(game string, owner Subject, cancel bool) (CloudJob, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for ref, j := range c.jobs {
		if j.game == game && strings.HasPrefix(j.mode, "verify_") && j.owner != nil && *j.owner == owner {
			return c.pollLocked(ref, cancel, &owner)
		}
	}
	return CloudJob{}, gameError("cloud_missing", "没有有效的云验证请求，请先获取验证码或查询关联状态。")
}
func (a *App) cloudVerifyCommand(event *rayleabot.EventContext, args []string) error {
	if (a.Game.ID != "genshin" && a.Game.ID != "starrail") || event.Event.EventType != "message.private" || event.Event.SourceProtocol != "onebot11" {
		return event.SendText("云 UID 验证仅支持原神/星铁的 OneBot11 本人私聊；不会把其他协议的开放 ID 当作 QQ。")
	}
	owner := Subject{SourceProtocol: event.Event.SourceProtocol, SourceAdapter: event.Event.SourceAdapter, BotID: event.Bot.ID, ActorID: event.Event.Actor.ID}
	if len(args) == 1 && (args[0] == "进度" || args[0] == "取消") {
		j, err := a.Cloud.pollVerification(a.Game.ID, owner, args[0] == "取消")
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		if j.State == "running" {
			return event.SendText("云验证仍在处理，最长等待 25 秒，请稍后再次查询进度。")
		}
		if j.State == "canceled" {
			return event.SendText("本地云验证等待已取消。已到达云服务的提交可能仍生效，可重新查询关联状态。")
		}
		if j.State == "failed" {
			return event.SendText(j.Message)
		}
		return event.SendText(j.View.Text())
	}
	mode := ""
	if len(args) > 0 {
		mode = map[string]string{"获取": "verify_code", "提交": "verify_bind", "状态": "verify_status"}[args[0]]
	}
	if len(args) != 3 || args[2] != "确认" || mode == "" {
		return event.SendText("云验证由 ark.ivny.cn 提供。获取会发送游戏 UID；提交/状态会发送你的 QQ 与 UID，提交通过后用于跨机器人关联。使用“" + a.Game.Prefix + "云验证 获取/提交/状态 UID 确认”，然后用“" + a.Game.Prefix + "云验证 进度”查看。验证码需自行写入游戏签名，不发送 CK。")
	}
	_, err := a.Cloud.Start(a.Game, CloudInput{Mode: mode, UID: args[1], Consent: true, owner: &owner})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText("云验证请求已开始，结果保留 5 分钟。请发送“" + a.Game.Prefix + "云验证 进度”查看；不要重复提交。")
}
