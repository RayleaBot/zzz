package app

import (
	"context"
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/pluginmeta"
	"slices"
	"strings"
)

// help lists the commands the requester may use, and the same list grouped
// for a plugin help image.
func (a *App) help(ctx context.Context, event *rayleabot.EventContext, query string) (map[string]any, HelpImage, error) {
	m := a.Manifest
	disabled := false
	if event.Event.Target.Type == "group" {
		config, err := a.Groups.Config(groupScope(event))
		if err != nil {
			return nil, HelpImage{}, err
		}
		disabled = config.Enabled != nil && !*config.Enabled
	}
	commands := []pluginmeta.Command{}
	for _, c := range m.Commands {
		if event.Event.Target.Type == "group" && slices.Contains([]string{"group-settings", "challenge-clear"}, c.ID) && !groupAdministrator(event) {
			continue
		}
		if disabled && !slices.Contains([]string{"help", "version", "group-settings", "unsubscribe", "reminder-stop", "gacha-stop", "poke"}, c.ID) {
			continue
		}
		allowed := event.Event.EventType == "management.action" || c.Permission == "everyone" || c.Permission == "" || slices.Contains(event.SuperAdmins, event.Event.Actor.ID)
		if !allowed && slices.Contains([]string{"admin", "group_admin"}, c.Permission) {
			allowed = groupAdministrator(event)
		}
		if !allowed || query != "" && !strings.Contains(c.Name+c.Description+c.Usage, query) {
			continue
		}
		commands = append(commands, c)
	}
	view := View{Title: a.Game.Name + "帮助", Subtitle: "插件 " + m.Version + " · " + m.License, Rows: []Row{}, Note: "管理页提供完整参数与数据管理。"}
	if disabled {
		view.Note = "本群游戏功能已暂停，当前只列管理与停止操作。"
	}
	// One section per declared group, in manifest order; commands outside every
	// group come last.
	visible := map[string]pluginmeta.Command{}
	for _, c := range commands {
		visible[c.ID] = c
	}
	placed := map[string]bool{}
	image := HelpImage{Title: view.Title, Subtitle: view.Subtitle, Note: view.Note}
	add := func(id, title string, list []pluginmeta.Command) {
		section, group := Section{Title: title, Rows: []Row{}}, HelpGroup{ID: id, Title: title}
		for _, c := range list {
			section.Rows = append(section.Rows, Row{a.usage(c.Usage), c.Description})
			group.Commands = append(group.Commands, HelpCommand{ID: c.ID, Name: c.Name, Usage: a.usage(c.Usage), Description: c.Description})
		}
		if len(section.Rows) > 0 {
			view.Sections = append(view.Sections, section)
			image.Groups = append(image.Groups, group)
		}
	}
	for _, group := range m.Groups {
		list := []pluginmeta.Command{}
		for _, id := range group.Commands {
			if c, ok := visible[id]; ok && !placed[id] {
				list = append(list, c)
				placed[id] = true
			}
		}
		add(group.ID, group.Title, list)
	}
	rest := []pluginmeta.Command{}
	for _, c := range commands {
		if !placed[c.ID] {
			rest = append(rest, c)
		}
	}
	add("other", "其他", rest)
	return map[string]any{"plugin": m, "commands": commands, "catalog_version": a.Catalog.Version, "resource_version": resourceVersion(a.Game), "group_enabled": !disabled, "view": view}, image, nil
}
func (a *App) helpCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	if command == "version" {
		return a.sendView(ctx, event, View{Title: a.Game.Name + "版本", Rows: []Row{{"插件", a.Manifest.Version}, {"最低宿主", a.Manifest.MinCoreVersion}, {"图鉴", a.Catalog.Version}, {"材料与卡池", resourceVersion(a.Game)}, {"许可", a.Manifest.License}}, Note: "插件通过宿主的安装/更新入口替换正式安装包。公开资料查询获取最新官方结果，本地固定资料随插件版本更新。"})
	}
	out, image, err := a.help(ctx, event, strings.Join(args, " "))
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	view := out["view"].(View)
	if a.helpImage != nil {
		if drawn, ok := a.helpImage(a.imageContext(ctx), image); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
}

// HelpImage is the help menu the requester may use, grouped as the manifest
// declares, for plugins that draw it the way their upstream help page does.
type HelpImage struct {
	Title, Subtitle, Note string
	Groups                []HelpGroup
}

// HelpGroup is one declared command group; ID is the manifest's group id, or
// "other" for the commands outside every group.
type HelpGroup struct {
	ID, Title string
	Commands  []HelpCommand
}

// HelpCommand is one command with its name, and its usage written with the
// reply prefix.
type HelpCommand struct {
	ID, Name, Usage, Description string
}

// HelpImageBuilder draws the help menu with the plugin's template, or returns
// false to keep the generic summary card.
type HelpImageBuilder func(ImageContext, HelpImage) (Image, bool)
