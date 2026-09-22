package app

import (
	"context"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// The upstreams' alias commands: miao's 喵喵别名设置, 删除 and 列表 for its
// custom aliases; StarRail-plugin's 设置X别名 (the aliases follow in the next
// message), 删除别名 and X别名; ZZZ-Plugin's 添加X别名Y, 删除别名 and X别名.
// Custom aliases are the plugin's custom_aliases setting, the one the
// management page edits. They apply to every group, so only super
// administrators change them; 群设置 别名 keeps a group's own.

// customAliases serializes chat changes to custom_aliases. Events of several
// groups run at once, each with the configuration of its start, so the value
// last written, or announced by config.changed, is the base of a change.
type customAliases struct {
	mu     sync.Mutex
	latest map[string]string
}

func (c *customAliases) observe(aliases map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.latest = maps.Clone(aliases)
	if c.latest == nil {
		c.latest = map[string]string{}
	}
}

// changeAliases applies edit to the newest custom aliases and writes them
// when edit says so, returning edit's reply.
func (a *App) changeAliases(ctx context.Context, event *rayleabot.EventContext, edit func(map[string]string) (string, bool)) (string, error) {
	a.aliases.mu.Lock()
	defer a.aliases.mu.Unlock()
	current := a.aliases.latest
	if current == nil {
		current = settings(event).CustomAliases
	}
	next := maps.Clone(current)
	if next == nil {
		next = map[string]string{}
	}
	reply, write := edit(next)
	if !write {
		return reply, nil
	}
	if len(next) > 256 {
		return "自定义别名最多 256 个。", nil
	}
	if _, err := event.Actions().ConfigWrite(ctx, map[string]any{"custom_aliases": next}); err != nil {
		return "", err
	}
	a.aliases.latest = next
	return reply, nil
}

// aliasOwner is the entry a word names exactly: by name, a built-in alias or
// a custom one; custom tells which.
func (a *App) aliasOwner(word string, aliases map[string]string) (entry Entry, custom string, found bool) {
	key := strings.ToLower(strings.TrimSpace(word))
	for alias, id := range aliases {
		if strings.ToLower(alias) == key {
			entry, found = a.Catalog.Get(id)
			return entry, alias, found
		}
	}
	for _, entry := range a.Catalog.Entries {
		if strings.ToLower(entry.Name) == key || slices.ContainsFunc(entry.Aliases, func(alias string) bool { return strings.ToLower(alias) == key }) {
			return entry, "", true
		}
	}
	return Entry{}, "", false
}

// aliasesOf are a character's built-in aliases, then its custom ones.
func aliasesOf(entry Entry, aliases map[string]string) []string {
	out := slices.Clone(entry.Aliases)
	custom := []string{}
	for alias, id := range aliases {
		if id == entry.ID {
			custom = append(custom, alias)
		}
	}
	slices.Sort(custom)
	return append(out, custom...)
}

var miaoAliasInvalid = regexp.MustCompile(`[,，:：\s]`)

func validAlias(alias string) bool {
	return alias != "" && len(alias) <= 64 && !strings.ContainsAny(alias, "\r\n\t")
}

func (a *App) aliasCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	style := a.Game.ID
	if strings.HasPrefix(event.Event.Command(), "喵喵别名") {
		style = "miao"
	}
	game := map[string]string{"genshin": "原神", "starrail": "星铁", "zzz": "绝区零"}[a.Game.ID]
	aliases := settings(event).CustomAliases
	reply := func(text string, err error) error {
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.SendText(text)
	}
	switch {
	case command == "alias-list":
		names := map[string][]string{}
		order := []string{}
		for alias, id := range aliases {
			entry, ok := a.Catalog.Get(id)
			if !ok {
				continue
			}
			if names[entry.Name] == nil {
				order = append(order, entry.Name)
			}
			names[entry.Name] = append(names[entry.Name], alias)
		}
		if len(order) == 0 {
			return event.SendText("暂无" + game + "自定义别名")
		}
		slices.Sort(order)
		lines := []string{"【" + game + "自定义别名】"}
		for _, name := range order {
			slices.Sort(names[name])
			lines = append(lines, name+"："+strings.Join(names[name], "，"))
		}
		return event.SendText(strings.Join(lines, "\n"))
	case command == "aliases":
		entry, ok := a.Catalog.Resolve(strings.Join(args, ""), "character", a.aliasMap(event))
		if !ok && style == "zzz" {
			return event.Result(map[string]any{"handled": false})
		}
		if !ok {
			return event.SendText("未识别到角色")
		}
		list := aliasesOf(entry, aliases)
		if style == "zzz" {
			if len(list) == 0 {
				return event.SendText("角色 " + entry.Name + " 暂无别名，可发送“" + a.Game.Prefix + "添加" + entry.Name + "别名xxx”添加")
			}
			return event.SendText("角色 " + entry.Name + " 共 " + strconv.Itoa(len(list)) + " 个别名：\n" + strings.Join(list, "、"))
		}
		parts := [][]rayleabot.Segment{}
		for index, alias := range list {
			parts = append(parts, []rayleabot.Segment{rayleabot.Text(strconv.Itoa(index+1) + "." + alias + "\n")})
		}
		if len(parts) == 0 {
			return event.SendText(entry.Name + "别名，0个")
		}
		return a.sendForward(ctx, event, parts)
	case command == "alias-add":
		entry, _, ok := a.aliasOwner(strings.Join(args, ""), aliases)
		if !ok || entry.Kind != "character" {
			return event.SendText("未识别到角色")
		}
		_, err := event.Ask(ctx, "请发送"+entry.Name+"别名，多个用空格隔开", rayleabot.SessionWaitOptions{Scope: "user", Timeout: 20 * time.Second}, func(ctx context.Context, next *rayleabot.EventContext) error {
			text := strings.TrimSpace(next.Event.Message.PlainText)
			for _, segment := range next.Event.Message.Segments {
				if segment.Type == "at" || segment.Type == "image" {
					text = ""
				}
			}
			if text == "" {
				return next.SendText("设置错误：请发送正确内容")
			}
			answer, err := a.changeAliases(ctx, next, func(aliases map[string]string) (string, bool) {
				added := []string{}
				for _, alias := range strings.Split(text, " ") {
					if _, _, taken := a.aliasOwner(alias, aliases); !validAlias(alias) || taken {
						continue
					}
					aliases[alias] = entry.ID
					added = append(added, alias)
				}
				if len(added) == 0 {
					return "设置失败：别名错误或已存在", false
				}
				return "设置别名成功：" + strings.Join(added, "、"), true
			})
			if err != nil {
				return next.SendText(friendlyError(err))
			}
			return next.SendText(answer)
		})
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return nil
	case command == "alias-set" && style == "miao":
		if len(args) != 2 {
			return event.SendText("命令格式：" + a.Game.Prefix + "喵喵别名设置 角色名 别名")
		}
		name, alias := args[0], args[1]
		if miaoAliasInvalid.MatchString(alias) || !validAlias(alias) {
			return event.SendText("别名不能为空，且不能包含逗号、冒号或空格")
		}
		if strings.Trim(alias, "0123456789") == "" {
			return event.SendText("别名不能为纯数字，避免与角色ID冲突")
		}
		return reply(a.changeAliases(ctx, event, func(aliases map[string]string) (string, bool) {
			entry, _, ok := a.aliasOwner(name, aliases)
			if !ok || entry.Kind != "character" {
				return "未找到该角色", false
			}
			if owner, _, taken := a.aliasOwner(alias, aliases); taken {
				if owner.ID == entry.ID {
					return "「" + entry.Name + "」已拥有别名「" + alias + "」，无需重复添加", false
				}
				return "别名「" + alias + "」已被角色「" + owner.Name + "」使用，请更换别名", false
			}
			aliases[alias] = entry.ID
			return entry.Name + "：" + alias + " 添加成功。", true
		}))
	case command == "alias-set":
		if len(args) != 2 {
			return event.Result(map[string]any{"handled": false})
		}
		name, alias := args[0], args[1]
		return reply(a.changeAliases(ctx, event, func(aliases map[string]string) (string, bool) {
			entry, _, ok := a.aliasOwner(name, aliases)
			if !ok || entry.Kind != "character" {
				return "未找到 " + name + " 的对应角色", false
			}
			if _, _, taken := a.aliasOwner(alias, aliases); taken || !validAlias(alias) {
				return "别名 " + alias + " 已存在", false
			}
			aliases[alias] = entry.ID
			return "角色 " + name + " 别名 " + alias + " 成功", true
		}))
	case command == "alias-remove":
		alias := strings.Join(args, "")
		if style == "miao" && len(args) != 1 {
			return event.SendText("命令格式：" + a.Game.Prefix + "喵喵别名删除 别名")
		}
		return reply(a.changeAliases(ctx, event, func(aliases map[string]string) (string, bool) {
			entry, custom, ok := a.aliasOwner(alias, aliases)
			if custom != "" {
				delete(aliases, custom)
			}
			switch {
			case style == "miao" && custom == "":
				return "不存在该别名，或该别名为预设，不支持删除", false
			case style == "miao":
				return "别名「" + alias + "」删除成功", true
			case style == "zzz" && !ok:
				return "未找到 " + alias + " 的对应角色", false
			case style == "zzz" && custom == "" && strings.EqualFold(entry.Name, alias):
				return "别名 " + alias + " 为角色本名，无法删除", false
			case style == "zzz" && custom == "":
				return "别名 " + alias + " 为内置别名，无法删除", false
			case style == "zzz":
				return "角色 " + alias + " 别名删除成功", true
			case !ok:
				return "未识别到角色", false
			case custom == "":
				return "默认别名设置，不能删除！", false
			}
			return "删除" + entry.Name + "别名成功：" + alias, true
		}))
	}
	return event.Result(map[string]any{"handled": false})
}
