package app

import (
	"context"
	"fmt"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// 绑定设备, 解绑设备, 绑定设备帮助 and 设置默认设备 follow ZZZ-Plugin, in
// group and private chats alike. The account plugin keeps the device
// information: after 绑定设备 or 设置默认设备 it takes the sender's next
// message that holds it, so it never passes through this plugin.
func (a *App) deviceCommand(ctx context.Context, event *rayleabot.EventContext, command string) error {
	if command == "device-help" {
		help := strings.NewReplacer("{prefix}", a.Game.Prefix, "{url}", settings(event).DeviceURL).Replace(deviceHelp)
		return a.sendForward(ctx, event, [][]rayleabot.Segment{{rayleabot.Text(help)}})
	}
	if command == "device-default" {
		if err := a.accountClient(event).call(ctx, "device.default", map[string]any{}, nil); err != nil {
			if PublicError(err).Code == "plugin.account_subject_denied" {
				return event.SendText("仅限主人设置")
			}
			return event.SendText(friendlyError(err))
		}
		return event.SendText("请发送默认设备信息，或者发送“取消”取消设置默认设备信息")
	}
	var answer struct {
		UID string `json:"uid"`
	}
	method := map[string]string{"device-bind": "device.bind", "device-unbind": "device.unbind"}[command]
	if err := a.accountClient(event).call(ctx, method, map[string]any{}, &answer); err != nil {
		switch PublicError(err).Code {
		case "plugin.account_region_unsupported":
			if command == "device-unbind" {
				// Upstream leaves 解绑设备 of an overseas UID unanswered.
				return event.Result(map[string]any{"handled": false})
			}
			return event.SendText("国际服不需要绑定设备")
		case "plugin.account_not_found":
			return event.SendText(a.deviceUIDReply(ctx, event, "尚未绑定cookie，请先绑定cookie，或者#扫码登录"))
		}
		return event.SendText(friendlyError(err))
	}
	if command == "device-unbind" {
		return event.SendText("解绑设备成功")
	}
	if answer.UID == "" {
		// The account holds no Zenless role: upstream finds no cookie
		// for the UID in use.
		return event.SendText(a.deviceUIDReply(ctx, event, "未绑定UID"))
	}
	return event.SendText(fmt.Sprintf("为UID %s绑定设备，请发送设备信息(建议私聊发送)，或者发送“取消”取消绑定", answer.UID))
}

// deviceUIDReply answers 绑定设备 or 解绑设备 without an account for the UID
// in use as upstream does: it first asks for a UID, which a user without
// one is told, and otherwise replies reply.
func (a *App) deviceUIDReply(ctx context.Context, event *rayleabot.EventContext, reply string) string {
	if listed, err := a.accountClient(event).List(ctx, 0); err == nil && a.currentUID(listed) == "" {
		return a.uidEmptyReply()
	}
	return reply
}

// defaultDeviceURL is ZZZ-Plugin's default config.url.
const defaultDeviceURL = "https://ghproxy.mihomo.me/https://raw.githubusercontent.com/forchannot/get_device_info/main/app/build/outputs/apk/debug/app-debug.apk"

// deviceHelp is ZZZ-Plugin's 绑定设备帮助, sent forwarded as upstream does;
// {url} is the device_download_url setting, upstream's config.url.
const deviceHelp = `[绑定设备]
方法一：
1. 使用抓包软件抓取米游社APP的请求
2. 在请求头内找到【x-rpc-device_id】和【x-rpc-device_fp】
3. 自行构造如下格式的信息：
    {"device_id": "x-rpc-device_id的内容", "device_fp": "x-rpc-device_fp的内容"}
4. 给机器人发送"{prefix}绑定设备"指令
5. 机器人会提示发送设备信息
6. 粘贴自行构造的信息发送
7. 提示绑定成功
--------------------------------
方法二（仅适用于安卓设备）：
1. 使用常用米游社手机下载下面链接的APK文件，并安装
{url}
2. 打开后点击按钮复制
3. 给机器人发送"{prefix}绑定设备"指令
4. 机器人会提示发送设备信息
5. 粘贴设备信息发送
6. 提示绑定成功
--------------------------------
[解绑设备]
发送 {prefix}解绑设备 即可`
