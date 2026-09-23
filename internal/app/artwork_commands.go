package app

import (
	"context"
	"crypto/rand"
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

// artworkTask prefixes the scheduled task that finishes 下载全部资源.
const artworkTask = "game.artwork."

// artworkJob is a running 下载全部资源: every file by its group and index, how
// far it got, the counts by group and the chat to answer in.
type artworkJob struct {
	groups          []ArtworkGroup
	items           [][2]int
	next            int
	success, failed []int
	owner           Subject
	target          rayleabot.Target
	expires         time.Time
}

// artworkJobs holds the one 下载全部资源 that may run at a time, as
// upstream's; job is nil while a trigger works on it.
type artworkJobs struct {
	mu      sync.Mutex
	running bool
	job     *artworkJob
}

// step fetches files, several at a time, until all are done or stop
// passes, and reports whether all are done.
func (j *artworkJob) step(ctx context.Context, store *artwork.Store, stop time.Time) bool {
	for j.next < len(j.items) && time.Now().Before(stop) && ctx.Err() == nil {
		batch := j.items[j.next:min(j.next+16, len(j.items))]
		results := make([]bool, len(batch))
		var group sync.WaitGroup
		slots := make(chan struct{}, 8)
		for index, item := range batch {
			group.Add(1)
			slots <- struct{}{}
			go func() {
				defer func() { <-slots; group.Done() }()
				_, results[index] = store.Fetch(ctx, j.groups[item[0]].Source, j.groups[item[0]].Files[item[1]])
			}()
		}
		group.Wait()
		for index, item := range batch {
			if results[index] {
				j.success[item[0]]++
			} else {
				j.failed[item[0]]++
			}
		}
		j.next += len(batch)
	}
	return j.next >= len(j.items)
}

// summary is upstream's closing reply.
func (j *artworkJob) summary() string {
	lines := []string{"资源下载完成（成功包含已下载资源）"}
	for index, group := range j.groups {
		lines = append(lines, fmt.Sprintf("%s：总数%d，成功%d，失败%d", group.Label, len(group.Files), j.success[index], j.failed[index]))
	}
	return strings.Join(append(lines, "注：下载失败可能缘于该资源尚处于内测中"), "\n")
}

// artworkAll is 下载全部资源: the repositories are updated as 素材更新 does,
// and every file the templates may read from the on-demand sources is
// fetched, as ZZZ-Plugin fetches every agent, W-Engine, drive disc and
// Bangboo picture. What does not finish within the event continues on a
// scheduled task, which answers in the same chat.
func (a *App) artworkAll(ctx context.Context, event *rayleabot.EventContext) error {
	a.artworkJobs.mu.Lock()
	if a.artworkJobs.running {
		a.artworkJobs.mu.Unlock()
		return event.SendText("下载任务正在进行中，请稍后再试")
	}
	a.artworkJobs.running = true
	a.artworkJobs.mu.Unlock()
	finish := func() {
		a.artworkJobs.mu.Lock()
		a.artworkJobs.running, a.artworkJobs.job = false, nil
		a.artworkJobs.mu.Unlock()
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
	job := &artworkJob{owner: syncOwner(event), target: event.Event.Target, expires: time.Now().Add(2 * time.Hour)}
	if a.downloads != nil {
		for _, group := range a.downloads(a.imageContext(ctx)) {
			if len(group.Files) == 0 {
				continue
			}
			for file := range group.Files {
				job.items = append(job.items, [2]int{len(job.groups), file})
			}
			job.groups = append(job.groups, group)
		}
	}
	job.success, job.failed = make([]int, len(job.groups)), make([]int, len(job.groups))
	if job.step(ctx, a.Artwork, time.Now().Add(40*time.Second)) {
		finish()
		return event.SendText(job.summary())
	}
	a.artworkJobs.mu.Lock()
	a.artworkJobs.job = job
	a.artworkJobs.mu.Unlock()
	if _, err := event.Actions().SchedulerCreate(ctx, rayleabot.SchedulerCreateRequest{TaskID: artworkTask + rand.Text(), Cron: "* * * * *", LogLabel: a.Game.Name + "下载全部资源", Payload: map[string]any{"kind": "artwork_all"}}); err != nil {
		finish()
		return event.SendText("资源较多，本次未能全部下载，请稍后再试。")
	}
	return event.Result(map[string]any{"handled": true})
}

// runArtworkJob continues 下载全部资源 on its scheduled task and answers in
// the chat it was asked in.
func (a *App) runArtworkJob(ctx context.Context, event *rayleabot.EventContext) error {
	if event.Event.SourceProtocol != "scheduler" || event.Event.SourceAdapter != "scheduler.internal" {
		return event.Fail("plugin.game_source_invalid", "任务来源无效。")
	}
	ref := asText(event.Event.Payload["task_id"])
	a.artworkJobs.mu.Lock()
	job, running := a.artworkJobs.job, a.artworkJobs.running
	a.artworkJobs.job = nil
	a.artworkJobs.mu.Unlock()
	if job == nil {
		// A trigger still working keeps the task; one left from before a
		// restart is removed.
		if !running {
			_, _ = event.Actions().SchedulerDelete(ctx, ref)
		}
		return event.Result(map[string]any{"handled": true})
	}
	if !time.Now().After(job.expires) && !job.step(ctx, a.Artwork, time.Now().Add(40*time.Second)) {
		a.artworkJobs.mu.Lock()
		a.artworkJobs.job = job
		a.artworkJobs.mu.Unlock()
		return event.Result(map[string]any{"handled": true})
	}
	_, _ = event.Actions().SchedulerDelete(ctx, ref)
	a.artworkJobs.mu.Lock()
	a.artworkJobs.running = false
	a.artworkJobs.mu.Unlock()
	_, _ = event.Actions().MessageSend(ctx, rayleabot.MessageSendRequest{SourceProtocol: job.owner.SourceProtocol, SourceAdapter: job.owner.SourceAdapter, TargetType: job.target.Type, TargetID: job.target.ID, Message: rayleabot.MessageOut{Segments: []rayleabot.Segment{rayleabot.Text(job.summary())}}})
	return event.Result(map[string]any{"handled": true})
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
