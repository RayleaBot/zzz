package app

import (
	"context"
	"regexp"
	"slices"
	"strconv"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// UIDs follow the upstream user commands (Yunzai 原神插件 user.js): 绑定uid
// binds any UID without an account and uses it, uid lists the account and
// bound UIDs by number, uid<序号> switches to one and 删除uid<序号> removes a
// bound one. The account plugin keeps the bindings.

// UIDBinding is a game's UIDs bound without an account and the UID in use.
type UIDBinding struct {
	UIDs    []string `json:"uids"`
	Current string   `json:"current,omitempty"`
}

// listedUID is one numbered UID of 我的uid: an account role, or a UID bound
// without an account.
type listedUID struct {
	UID    string
	Role   *Role
	Choice Selection
}

func (c AccountsClient) ChangeUID(ctx context.Context, action, uid string) error {
	return c.call(ctx, "uids."+action, map[string]any{"game": c.Game, "uid": uid}, nil)
}

// uidList numbers the account roles first, then the bound UIDs, as upstream
// lists CK UIDs before bound ones.
func (a *App) uidList(listed Accounts) []listedUID {
	list := []listedUID{}
	seen := map[string]bool{}
	for _, account := range listed.Items {
		for _, role := range account.Roles {
			if role.Game == a.Game.ID && !seen[role.UID] {
				seen[role.UID] = true
				list = append(list, listedUID{UID: role.UID, Role: &role, Choice: Selection{AccountRef: account.Ref, RoleRef: role.Ref}})
			}
		}
	}
	for _, uid := range listed.UIDBindings[a.Game.ID].UIDs {
		if !seen[uid] {
			seen[uid] = true
			list = append(list, listedUID{UID: uid})
		}
	}
	return list
}

// currentUID is the UID in use: the one the user picked, else the default
// account role.
func (a *App) currentUID(listed Accounts) string {
	if current := listed.UIDBindings[a.Game.ID].Current; current != "" {
		return current
	}
	if _, role, err := Choose(listed, a.Game.ID, ""); err == nil {
		return role.UID
	}
	return ""
}

var uidIndex = regexp.MustCompile(`[0-9]{1,2}$`)

// uidCommand answers 绑定uid, uid, uid<序号> and 删除uid<序号>.
func (a *App) uidCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	client := a.accountClient(event)
	listed, err := client.List(ctx, 0)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	list := a.uidList(listed)
	prefix := a.Game.Prefix
	switch command {
	case "select":
		if len(args) != 1 {
			return event.SendText("请发送“" + prefix + "绑定uid”加上游戏 UID，例如“" + prefix + "绑定uid100000001”。")
		}
		uid := args[0]
		if choice, _, err := Choose(listed, a.Game.ID, uid); err == nil {
			// An account's UID also becomes its default role.
			err = client.Select(ctx, choice)
			if err == nil {
				err = client.ChangeUID(ctx, "use", uid)
			}
			if err != nil {
				return event.SendText(friendlyError(err))
			}
		} else if err := client.ChangeUID(ctx, "bind", uid); err != nil {
			return event.SendText(friendlyError(err))
		}
	case "accounts":
		index := uidIndex.FindString(event.Event.Command())
		if index == "" {
			break
		}
		n, _ := strconv.Atoi(index)
		if n < 1 || n > len(list) {
			return event.SendText("uid序号输入错误")
		}
		picked := list[n-1]
		if picked.Role != nil {
			if err := client.Select(ctx, picked.Choice); err != nil {
				return event.SendText(friendlyError(err))
			}
		}
		if err := client.ChangeUID(ctx, "use", picked.UID); err != nil {
			return event.SendText(friendlyError(err))
		}
	case "uid-remove":
		index := ""
		if len(args) > 0 {
			index = args[0]
		}
		if index == "" {
			return event.SendText("删除uid请带上序号\n例如：" + prefix + "删除uid1\n发送【" + prefix + "uid】可查看绑定的uid以及对应的序号")
		}
		n, _ := strconv.Atoi(index)
		if n < 1 || n > len(list) {
			return event.SendText("uid序号输入错误")
		}
		if list[n-1].Role != nil {
			return event.SendText("CK对应UID无法直接删除，请通过【删除ck】命令来删除")
		}
		if err := client.ChangeUID(ctx, "remove", list[n-1].UID); err != nil {
			return event.SendText(friendlyError(err))
		}
	}
	if listed, err = client.List(ctx, 0); err != nil {
		return event.SendText(friendlyError(err))
	}
	view, image := a.uidView(listed)
	if a.uidListImage != nil {
		if drawn, ok := a.uidListImage(a.imageContext(ctx), image); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
}

// UIDListImage is what 我的uid draws on, as the Yunzai 原神插件's
// html/user/uid-list: the numbered UIDs and the player each UID's saved
// panels show.
type UIDListImage struct {
	Entries []UIDListEntry
}

type UIDListEntry struct {
	UID string
	// Account marks a UID of a logged-in account (upstream's CK), not one
	// bound by number; Active the UID in use.
	Account, Active bool
	Nickname        string
	Level           int
	// Character is the first character kept for the UID, whose face and
	// name card miao shows for the player.
	Character string
}

// UIDListImageBuilder draws 我的uid with the plugin's template, or returns
// false to keep the list in text.
type UIDListImageBuilder func(ImageContext, UIDListImage) (Image, bool)

// uidView lists the numbered UIDs, marking the one in use, and gathers the
// same list for the image.
func (a *App) uidView(listed Accounts) (View, UIDListImage) {
	list := a.uidList(listed)
	view := View{Title: a.Game.Name + " UID", Rows: []Row{}, Note: "发送“" + a.Game.Prefix + "uid序号”切换，“" + a.Game.Prefix + "删除uid序号”删除绑定的 UID。"}
	image := UIDListImage{Entries: []UIDListEntry{}}
	if len(list) == 0 {
		view.Note = "暂无绑定的" + a.Game.Name + " UID。发送“" + a.Game.Prefix + "绑定uid”加 UID 绑定，或发送“扫码登录”绑定米游社账号。"
		return view, image
	}
	current := a.currentUID(listed)
	for index, item := range list {
		entry := UIDListEntry{UID: item.UID, Account: item.Role != nil, Active: item.UID == current}
		if saved, err := a.Profiles.Read(item.UID); err == nil {
			entry.Nickname, entry.Level = saved.Nickname, saved.Level
			// miao takes the first saved character in its key order, the
			// smallest ID.
			ids := []int{}
			for id := range saved.Panels {
				if n, err := strconv.Atoi(id); err == nil {
					ids = append(ids, n)
				}
			}
			if len(ids) > 0 {
				entry.Character = strconv.Itoa(slices.Min(ids))
			}
		}
		image.Entries = append(image.Entries, entry)
		label := strconv.Itoa(index + 1)
		if entry.Active {
			label = "☑" + label
		}
		value := item.UID + " [绑定]"
		if item.Role != nil {
			value = item.UID + " [CK] " + item.Role.Nickname
		}
		view.Rows = append(view.Rows, Row{Label: label, Value: value})
	}
	return view, image
}
