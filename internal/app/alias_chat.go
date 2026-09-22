package app

import (
	"context"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// ZZZ-Plugin's alias commands: 添加X别名Y, 删除别名Y and X别名.
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

func validAlias(alias string) bool {
	return alias != "" && len(alias) <= 64 && !strings.ContainsAny(alias, "\r\n\t")
}

func (a *App) aliasCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	reply := func(text string, err error) error {
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.SendText(text)
	}
	switch command {
	case "aliases":
		entry, ok := a.Catalog.Resolve(strings.Join(args, ""), "character", a.aliasMap(event))
		if !ok {
			return event.Result(map[string]any{"handled": false})
		}
		list := aliasesOf(entry, settings(event).CustomAliases)
		if len(list) == 0 {
			return event.SendText("角色 " + entry.Name + " 暂无别名，可发送“" + a.Game.Prefix + "添加" + entry.Name + "别名xxx”添加")
		}
		return event.SendText("角色 " + entry.Name + " 共 " + strconv.Itoa(len(list)) + " 个别名：\n" + strings.Join(list, "、"))
	case "alias-set":
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
	case "alias-remove":
		alias := strings.Join(args, "")
		return reply(a.changeAliases(ctx, event, func(aliases map[string]string) (string, bool) {
			entry, custom, ok := a.aliasOwner(alias, aliases)
			switch {
			case !ok:
				return "未找到 " + alias + " 的对应角色", false
			case custom == "" && strings.EqualFold(entry.Name, alias):
				return "别名 " + alias + " 为角色本名，无法删除", false
			case custom == "":
				return "别名 " + alias + " 为内置别名，无法删除", false
			}
			delete(aliases, custom)
			return "角色 " + alias + " 别名删除成功", true
		}))
	}
	return event.Result(map[string]any{"handled": false})
}
