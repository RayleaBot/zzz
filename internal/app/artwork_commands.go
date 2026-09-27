package app

import (
	"context"
	"fmt"
	"strings"
	"sync"
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

// ArtworkGroup is one kind of picture 下载全部资源 fetches from a source the
// templates read on demand: its name in the reply and every file of it the
// templates may ask for.
type ArtworkGroup struct {
	Label  string
	Source string
	Files  []string
}

// ArtworkGroupsBuilder lists the groups by the known IDs; the image builders
// know the file names.
type ArtworkGroupsBuilder func(ImageContext) []ArtworkGroup

// artworkAll is 下载全部资源: the repositories are updated as 素材更新 does,
// and every file the templates may read from the on-demand sources is
// fetched, as ZZZ-Plugin fetches every agent, W-Engine, drive disc and
// Bangboo picture. The command's event moves to the background and answers
// the counts once every file is fetched; one download runs at a time, as
// upstream's, and another may start before the counts are answered.
func (a *App) artworkAll(ctx context.Context, event *rayleabot.EventContext) error {
	if !a.flows.begin("artwork") {
		return event.SendText("下载任务正在进行中，请稍后再试")
	}
	answer := a.downloadAll(ctx, event)
	a.flows.end("artwork")
	return answer()
}

// downloadAll updates the repositories and fetches every on-demand file in
// the background, and returns the answer.
func (a *App) downloadAll(ctx context.Context, event *rayleabot.EventContext) func() error {
	if _, err := event.Detach(ctx, nil); err != nil {
		return reply(event, detachFailure(err).Message)
	}
	started := []string{}
	for _, source := range a.Artwork.Sources {
		if len(source.Mirrors) == 0 {
			if _, err := a.Artwork.Start(source.ID); err == nil {
				started = append(started, source.Name)
			}
		}
	}
	text := "开始下载全部资源：代理人、音擎、驱动盘、邦布图片等，请耐心等待……"
	if len(started) > 0 {
		text += "\n同时在后台更新：" + strings.Join(started, "、") + "，发送“" + a.Game.Prefix + "素材状态”查看进度。"
	}
	notice(ctx, event, text)
	groups := []ArtworkGroup{}
	if a.downloads != nil {
		for _, group := range a.downloads(a.imageContext(ctx)) {
			if len(group.Files) > 0 {
				groups = append(groups, group)
			}
		}
	}
	success, failed := a.fetchArtwork(ctx, groups)
	if err := ctx.Err(); err != nil {
		return func() error { return err }
	}
	return reply(event, artworkSummary(groups, success, failed))
}

// fetchArtwork fetches every file of groups, eight at a time, and counts by
// group the files fetched or already cached and those that failed.
func (a *App) fetchArtwork(ctx context.Context, groups []ArtworkGroup) (success, failed []int) {
	success, failed = make([]int, len(groups)), make([]int, len(groups))
	var mu sync.Mutex
	var fetching sync.WaitGroup
	slots := make(chan struct{}, 8)
	for index, group := range groups {
		for _, file := range group.Files {
			if ctx.Err() != nil {
				break
			}
			slots <- struct{}{}
			fetching.Go(func() {
				defer func() { <-slots }()
				_, ok := a.Artwork.Fetch(ctx, group.Source, file)
				mu.Lock()
				defer mu.Unlock()
				if ok {
					success[index]++
				} else {
					failed[index]++
				}
			})
		}
	}
	fetching.Wait()
	return success, failed
}

// artworkSummary is upstream's closing reply: each kind's total, fetched and
// failed files.
func artworkSummary(groups []ArtworkGroup, success, failed []int) string {
	lines := []string{"资源下载完成（成功包含已下载资源）"}
	for index, group := range groups {
		lines = append(lines, fmt.Sprintf("%s：总数%d，成功%d，失败%d", group.Label, len(group.Files), success[index], failed[index]))
	}
	return strings.Join(append(lines, "注：下载失败可能缘于该资源尚处于内测中"), "\n")
}

// artworkDeleteAll is 删除全部资源: the caches of the sources fetched on
// demand are removed, as ZZZ-Plugin removes only its own download folders;
// downloaded repositories and the files the package ships remain.
func (a *App) artworkDeleteAll(ctx context.Context, event *rayleabot.EventContext) error {
	notice(ctx, event, "【注意】正在删除所有资源图片，后续使用需要重新下载！")
	busy := []string{}
	for _, source := range a.Artwork.Sources {
		if len(source.Mirrors) == 0 {
			continue
		}
		if err := a.Artwork.Delete(source.ID); err != nil {
			busy = append(busy, source.Name)
		}
	}
	if len(busy) > 0 {
		return event.SendText("资源图片已删除！\n以下素材正在下载或无法删除：" + strings.Join(busy, "、"))
	}
	return event.SendText("资源图片已删除！")
}
