package app

import (
	"fmt"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/artwork"
)

// artworkCommand starts downloads or reports their progress. A download runs
// in the background, so the reply only confirms the start.
func (a *App) artworkCommand(event *rayleabot.EventContext, command string, args []string) error {
	if len(a.Artwork.Sources) == 0 {
		return event.SendText(a.Game.Name + "插件没有可下载的素材。")
	}
	if command == "artwork-status" {
		return event.SendText(artworkStatusText(a.Artwork.Statuses()))
	}
	started := []string{}
	for _, source := range a.Artwork.Sources {
		if len(source.Mirrors) > 0 || !artworkSelected(source, args) {
			continue
		}
		if _, err := a.Artwork.Start(source.ID); err != nil {
			return event.SendText(friendlyError(err))
		}
		started = append(started, source.Name)
	}
	if len(started) == 0 {
		names := []string{}
		for _, source := range a.Artwork.Sources {
			names = append(names, source.Name)
		}
		return event.SendText("没有匹配的素材。可下载：" + strings.Join(names, "、") + "。")
	}
	return event.SendText("已开始下载：" + strings.Join(started, "、") + "。下载在后台进行，发送“" + a.Game.Prefix + "素材状态”查看进度。")
}

func artworkSelected(source artwork.Source, args []string) bool {
	if len(args) == 0 {
		return true
	}
	for _, arg := range args {
		if source.ID == arg || strings.Contains(source.Name, arg) {
			return true
		}
	}
	return false
}

func artworkStatusText(statuses []artwork.Status) string {
	lines := []string{"素材状态"}
	for _, status := range statuses {
		line := status.Name + "："
		switch status.State {
		case "downloading":
			line += fmt.Sprintf("下载中，已接收 %.1f MB", float64(status.Received)/1e6)
		case "on_demand":
			line += fmt.Sprintf("出图时按需下载，已缓存 %d 个文件（%.1f MB）", status.Files, float64(status.Bytes)/1e6)
		case "bundled":
			line += fmt.Sprintf("随插件安装包附带 %d 个文件（%.1f MB）", status.Files, float64(status.Bytes)/1e6)
			if status.Commit != "" {
				line += "，版本 " + status.Commit[:min(7, len(status.Commit))]
			}
		case "ready":
			line += fmt.Sprintf("已下载 %d 个文件（%.1f MB）", status.Files, float64(status.Bytes)/1e6)
			if status.Commit != "" {
				line += "，版本 " + status.Commit[:min(7, len(status.Commit))]
			}
			if status.DownloadedMS > 0 {
				line += "，" + time.UnixMilli(status.DownloadedMS).In(time.FixedZone("UTC+8", 8*3600)).Format("2006-01-02 15:04")
			}
		default:
			line += "未下载"
		}
		if status.Error != "" {
			line += "\n  上次下载失败：" + status.Error
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (a *App) artworkAction(action string, input map[string]any) (map[string]any, error) {
	id := asText(input["id"])
	switch action {
	case "artwork.status":
	case "artwork.download":
		if _, err := a.Artwork.Start(id); err != nil {
			return nil, gameError("input_invalid", "素材来源不存在。")
		}
	case "artwork.delete":
		if err := a.Artwork.Delete(id); err != nil {
			return nil, gameError("operation_denied", "素材正在下载或无法删除："+err.Error())
		}
	default:
		return nil, gameError("operation_denied", "操作不存在。")
	}
	return map[string]any{"sources": a.Artwork.Statuses()}, nil
}
