package app

import (
	"bytes"
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/gacha"
)

// 导出记录 and 导入记录 exchange gacha logs in chat as UIGF files, written as
// Yunzai's gcLog does for the other two games (ZZZ-Plugin has no such
// command): a group chat needs 强制, the export is sent as a file, and
// 导入记录 takes the file, or an https link to one, that the sender sends
// next within two minutes.

// gachaFileField is this game's member of a UIGF v4 file.
const gachaFileField = "nap"

// gachaFileLimit keeps a file within the host's message frame once encoded;
// imports read up to gachaImportLimit.
const (
	gachaFileLimit   = 5 << 20
	gachaImportLimit = 64 << 20
)

// fileImports are the senders asked for a file by 导入记录.
type fileImports struct {
	mu    sync.Mutex
	until map[string]time.Time
}

func (f *fileImports) wait(key string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now()
	if f.until == nil {
		f.until = map[string]time.Time{}
	}
	for other, until := range f.until {
		if now.After(until) {
			delete(f.until, other)
		}
	}
	f.until[key] = now.Add(2 * time.Minute)
}

func (f *fileImports) waiting(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return time.Now().Before(f.until[key])
}

func (f *fileImports) done(key string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.until, key)
}

func senderKey(event *rayleabot.EventContext) string {
	return strings.Join([]string{event.Event.SourceProtocol, event.Event.SourceAdapter, event.Bot.ID, event.Event.Target.Type, event.Event.Target.ID, event.Event.Actor.ID}, "\x00")
}

func (a *App) gachaFileCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	word := map[string]string{"gacha-export": "导出", "gacha-import": "导入"}[command]
	if event.Event.Target.Type == "group" && !slices.Contains(args, "强制") {
		return event.SendText(fmt.Sprintf("建议私聊%s，若你确认要在此%s，请发送【%s强制%s记录】", word, word, a.Game.Prefix, word))
	}
	if command == "gacha-import" {
		a.fileImports.wait(senderKey(event))
		return event.SendText("请发送Json文件")
	}
	archive, _, err := a.chatArchive(ctx, event, "")
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	content, err := json.Marshal(a.gachaFile(archive))
	if err != nil {
		return err
	}
	if len(content) > gachaFileLimit {
		return event.SendText("导出失败：记录过多，文件超过 5 MB")
	}
	name := archive.UID + ".json"
	notice(ctx, event, fmt.Sprintf("导出成功：%s，共%d条 \n请接收文件", name, len(archive.Records)))
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, rayleabot.Passthrough("file", map[string]any{"file": "base64://" + base64.StdEncoding.EncodeToString(content), "name": name}))
}

// gachaFile writes an archive as UIGF v4; one with the reprise pools UIGF
// does not list is written whole in RayleaBot's archive format instead.
func (a *App) gachaFile(archive gacha.Archive) map[string]any {
	for _, record := range archive.Records {
		if record.GachaType == "102" || record.GachaType == "103" {
			return map[string]any{"info": map[string]any{"format": "raylea-gacha", "version": 1}, "game": a.Game.ID, "archive": map[string]any{"uid": archive.UID, "timezone": archive.Timezone, "lang": archive.Language, "list": archive.Records}}
		}
	}
	now := time.Now()
	return map[string]any{"info": map[string]any{"export_timestamp": now.Unix(), "export_app": "RayleaBot", "export_app_version": a.Manifest.Version, "version": "v4.0"},
		gachaFileField: []any{map[string]any{"uid": archive.UID, "timezone": archive.Timezone, "lang": archive.Language, "list": archive.Records}}}
}

// gachaFileMessage imports the file a sender sends after 导入记录; handled is
// false for other messages, which leave the wait running as Yunzai's context
// does.
func (a *App) gachaFileMessage(ctx context.Context, event *rayleabot.EventContext) (handled bool, err error) {
	key := senderKey(event)
	if !a.fileImports.waiting(key) {
		return false, nil
	}
	if event.Event.Target.Type == "group" {
		if config, err := a.Groups.Config(groupScope(event)); err == nil && config.Enabled != nil && !*config.Enabled {
			return false, nil
		}
	}
	file, name, found := messageFile(ctx, event)
	if !found {
		return false, nil
	}
	a.fileImports.done(key)
	defer func() {
		if event.Event.Target.Type == "group" {
			notice(ctx, event, "已收到文件，请撤回")
		}
	}()
	if file == "" {
		return true, event.SendText("文件链接获取失败")
	}
	raw, err := downloadFile(ctx, file, gachaImportLimit)
	if err != nil {
		return true, event.SendText("下载json文件错误")
	}
	archives, err := a.parseGachaFile(raw)
	if err != nil {
		return true, event.SendText(strings.ReplaceAll(friendlyError(err), "{name}", name))
	}
	if len(archives) == 0 {
		return true, event.SendText("文件中没有" + a.Game.Name + "记录")
	}
	listed, _ := a.accountClient(event).List(ctx, 0)
	lines := []string{name + "，" + a.Game.Name + "记录导入成功"}
	for _, archive := range archives {
		archive.Region = a.archiveRegion(listed, archive.UID)
		if archive.Timezone == noTimezone {
			archive.Timezone = regionTimezone(archive.Region)
		}
		if gacha.Validate(archive) != nil {
			return true, event.SendText("json文件内容错误：记录格式不正确")
		}
		if _, _, err := a.Gacha.Import(archive); err != nil {
			return true, event.SendText("记录与已有档案冲突，原档案已保留")
		}
		if len(archives) > 1 {
			lines = append(lines, "UID："+archive.UID)
		}
		counts := gachaPoolCounts(archive)
		for _, pool := range gachaLinkPools {
			if counts[pool[1]] > 0 {
				lines = append(lines, fmt.Sprintf("%s记录：%d条", pool[2], counts[pool[1]]))
			}
		}
	}
	return true, event.SendText(strings.Join(lines, "\n"))
}

var fileLink = regexp.MustCompile(`https://[^\s]+`)

// messageFile finds the file a message carries: its download address, from
// the segment or asked of the chat platform, and its name; or an https link
// in the text. found is false when the message has neither.
func messageFile(ctx context.Context, event *rayleabot.EventContext) (file, name string, found bool) {
	for _, segment := range event.Event.Message.Segments {
		if segment.Type != "file" {
			continue
		}
		name = asText(segment.Data["name"])
		if name == "" {
			name = asText(segment.Data["file"])
		}
		file = asText(segment.Data["url"])
		if id := asText(segment.Data["file_id"]); !strings.HasPrefix(file, "http") && id != "" {
			var result rayleabot.ActionResult
			var err error
			if event.Event.Target.Type == "group" {
				result, err = event.Actions().FileGroupURLGet(ctx, event.Event.Target.ID, id)
			} else {
				result, err = event.Actions().FilePrivateURLGet(ctx, event.Event.Actor.ID, id)
			}
			if err == nil {
				file = asText(result["url"])
			}
		}
		if !strings.HasPrefix(file, "http") {
			file = ""
		}
		return file, importName(name, file), true
	}
	if link := strings.TrimRight(fileLink.FindString(event.Event.Message.PlainText), ")>】」』\"'，。！？；;"); link != "" {
		return link, importName("", link), true
	}
	return "", "", false
}

// importName is the file name the reply uses, as Yunzai's getImportFileName.
func importName(name, link string) string {
	if name == "" {
		if parsed, err := url.Parse(link); err == nil {
			for _, key := range []string{"filename", "fileName", "name"} {
				if value := parsed.Query().Get(key); value != "" {
					name = value
					break
				}
			}
			if name == "" {
				name = path.Base(parsed.Path)
			}
		}
	}
	if name = path.Base(strings.TrimSpace(name)); name == "" || name == "." || name == "/" {
		name = "记录"
	}
	if path.Ext(name) == "" {
		name += ".json"
	}
	return name
}

// noTimezone marks a legacy file that names no time zone; the region's then
// applies.
const noTimezone = 99

// parseGachaFile reads this game's accounts from a UIGF v4 or a RayleaBot
// archive file. Only the standard record fields are
// read, so links and credentials in a file never reach an archive; names,
// types and ranks a file leaves out come from the catalog, as Yunzai fills
// them from the official list.
func (a *App) parseGachaFile(raw []byte) ([]gacha.Archive, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var root map[string]any
	if decoder.Decode(&root) != nil {
		return nil, gameError("gacha_file_invalid", "{name},json格式错误")
	}
	info := asObject(root["info"])
	var sources []any
	switch {
	case info["format"] == "raylea-gacha":
		if root["game"] != a.Game.ID {
			return nil, nil
		}
		sources = []any{root["archive"]}
	case root["list"] != nil:
		// Legacy files hold the other two games' logs.
		return nil, nil
	case root["hk4e"] != nil || root["hkrpg"] != nil || root["nap"] != nil:
		sources, _ = root[gachaFileField].([]any)
	default:
		return nil, gameError("gacha_file_invalid", "json文件内容错误：非统一祈愿记录标准")
	}
	archives := []gacha.Archive{}
	for _, value := range sources {
		source := asObject(value)
		list, _ := source["list"].([]any)
		if len(list) == 0 {
			continue
		}
		archive := gacha.Archive{UID: fileText(source["uid"]), Timezone: noTimezone, Language: fileText(source["lang"])}
		if timezone, err := strconv.Atoi(fileText(source["timezone"])); err == nil {
			archive.Timezone = timezone
		}
		if archive.Language == "" {
			archive.Language = "zh-cn"
		}
		for _, item := range list {
			record, err := a.fileRecord(asObject(item))
			if err != nil {
				return nil, err
			}
			archive.Records = append(archive.Records, record)
		}
		archives = append(archives, archive)
	}
	return archives, nil
}

func (a *App) fileRecord(raw map[string]any) (gacha.Record, error) {
	record := gacha.Record{ID: fileText(raw["id"]), GachaType: fileText(raw["gacha_type"]), UIGFType: fileText(raw["uigf_gacha_type"]), GachaID: fileText(raw["gacha_id"]), ItemID: fileText(raw["item_id"]),
		Name: fileText(raw["name"]), ItemType: fileText(raw["item_type"]), Rank: fileText(raw["rank_type"]), Count: fileText(raw["count"]), Time: fileText(raw["time"])}
	if record.ItemID == "" && record.Name != "" {
		if entry, ok := a.Catalog.Resolve(record.Name, "", nil); ok && entry.Name == record.Name {
			record.ItemID = entry.ID
		}
	}
	if entry, ok := a.Catalog.Get(record.ItemID); ok {
		record.Name = cmp.Or(record.Name, entry.Name)
		record.ItemType = cmp.Or(record.ItemType, map[string]string{"character": "代理人", "weapon": "音擎", "buddy": "邦布"}[entry.Kind])
		if record.Rank == "" && entry.Rarity > 0 {
			record.Rank = strconv.Itoa(entry.Rarity)
		}
	}
	missing := []string{}
	for _, field := range [][2]string{{"id", record.ID}, {"gacha_type", record.GachaType}, {"item_id", record.ItemID}, {"time", record.Time}} {
		if field[1] == "" {
			missing = append(missing, field[0])
		}
	}
	if len(missing) > 0 {
		return record, gameError("gacha_file_invalid", "json文件内容错误：缺少必要字段 "+strings.Join(missing, ", "))
	}
	return record, nil
}

// fileText reads a string or a JSON number exactly; numbers stay digits so
// long record IDs keep their precision.
func fileText(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	}
	return ""
}

// archiveRegion is the region an imported UID's archive goes under: the
// sender's role with the UID, an archive the UID already has, or the region
// its first digits name.
func (a *App) archiveRegion(listed Accounts, uid string) string {
	if _, role, err := Choose(listed, a.Game.ID, uid); err == nil && role.Region != "" {
		return role.Region
	}
	if summaries, err := a.Gacha.List(); err == nil {
		for _, summary := range summaries {
			if summary.UID == uid {
				return summary.Region
			}
		}
	}
	return UIDRegion(uid)
}
