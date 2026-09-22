package app

import (
	"context"
	"fmt"
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"slices"
	"strings"
	"time"
)

type RoleCard struct {
	Name     string `json:"name"`
	Count    int    `json:"count"`
	Unlocked bool   `json:"unlocked"`
}
type RoleCardOffer struct {
	ActorID  string     `json:"actor_id"`
	Nickname string     `json:"nickname"`
	UID      string     `json:"uid"`
	Region   string     `json:"region"`
	Cards    []RoleCard `json:"cards"`
	SavedMS  int64      `json:"saved_ms"`
}
type RoleCardMatch struct {
	ActorID  string   `json:"actor_id"`
	Nickname string   `json:"nickname"`
	UID      string   `json:"uid"`
	Give     []string `json:"give"`
	Receive  []string `json:"receive"`
	Score    int      `json:"score"`
	SavedMS  int64    `json:"saved_ms"`
}

func parseRoleCards(data map[string]any) ([]RoleCard, error) {
	state := asObject(data["tarot_card_state"])
	raw, ok := state["list"].([]any)
	if !ok || len(raw) > 200 {
		return nil, gameError("cards_unavailable", "官方尚未返回完整月谕圣牌收藏。")
	}
	cards := []RoleCard{}
	seen := map[string]bool{}
	for _, v := range raw {
		m := asObject(v)
		name := plainGameText(asText(m["name"]))
		unlocked, ok := m["is_unlock"].(bool)
		n, known := challengeNumber(m["unlock_num"])
		if name == "" || len([]rune(name)) > 64 || !ok || !known || !finiteRange(n, 0, 10000) || n != float64(int(n)) || seen[name] {
			return nil, gameError("cards_unavailable", "圣牌数据缺少数量、解锁状态或名称。")
		}
		seen[name] = true
		cards = append(cards, RoleCard{name, int(n), unlocked})
	}
	slices.SortFunc(cards, func(a, b RoleCard) int { return strings.Compare(a.Name, b.Name) })
	return cards, nil
}
func roleCardView(role Role, cards []RoleCard) View {
	view := View{Title: "月谕圣牌收藏", Subtitle: role.Nickname + " · " + role.UID, Rows: []Row{}, Note: "交换匹配需在群内主动提交；查询本身不加入群交换列表。"}
	for _, c := range cards {
		text := fmt.Sprintf("持有%d张", c.Count)
		if !c.Unlocked || c.Count <= 0 {
			text += " · 缺少"
		} else if c.Count > 1 {
			text += fmt.Sprintf(" · 多余%d张", c.Count-1)
		}
		view.Rows = append(view.Rows, Row{c.Name, text})
	}
	return view
}
func roleCardMatches(self RoleCardOffer, others []RoleCardOffer, now int64) []RoleCardMatch {
	extra, need := func(cards []RoleCard) (map[string]bool, map[string]bool) {
		extra, need := map[string]bool{}, map[string]bool{}
		for _, c := range cards {
			if c.Unlocked && c.Count > 1 {
				extra[c.Name] = true
			}
			if !c.Unlocked || c.Count <= 0 {
				need[c.Name] = true
			}
		}
		return extra, need
	}(self.Cards)
	out := []RoleCardMatch{}
	for _, other := range others {
		if other.ActorID == self.ActorID || other.UID == self.UID || other.Region != self.Region || other.SavedMS <= now-int64(30*24*time.Hour/time.Millisecond) {
			continue
		}
		candidate := RoleCardMatch{ActorID: other.ActorID, Nickname: other.Nickname, UID: other.UID, SavedMS: other.SavedMS, Give: []string{}, Receive: []string{}}
		for _, c := range other.Cards {
			if c.Unlocked && c.Count > 1 && need[c.Name] {
				candidate.Receive = append(candidate.Receive, c.Name)
			}
			if (!c.Unlocked || c.Count <= 0) && extra[c.Name] {
				candidate.Give = append(candidate.Give, c.Name)
			}
		}
		if len(candidate.Give) > 0 && len(candidate.Receive) > 0 {
			candidate.Score = len(candidate.Give) + len(candidate.Receive)
			out = append(out, candidate)
		}
	}
	slices.SortFunc(out, func(a, b RoleCardMatch) int {
		if a.Score != b.Score {
			return b.Score - a.Score
		}
		return strings.Compare(a.ActorID, b.ActorID)
	})
	return out[:min(8, len(out))]
}
func (a *App) roleCardsQuery(ctx context.Context, client AccountsClient, choice Selection) (QueryResult, []RoleCard, error) {
	result, err := client.Execute(ctx, choice, "genshin.theater", map[string]any{"need_detail": true})
	if err != nil {
		return result, nil, err
	}
	cards, err := parseRoleCards(result.Data)
	return result, cards, err
}
func (a *App) cardsAction(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	if a.Game.ID != "genshin" {
		return nil, gameError("operation_denied", "月谕圣牌仅适用于原神。")
	}
	if action == "cards.query" {
		result, cards, err := a.roleCardsQuery(ctx, a.accountClient(event), Selection{asText(input["account_ref"]), asText(input["role_ref"])})
		if err != nil {
			return nil, err
		}
		return map[string]any{"cards": cards, "view": roleCardView(result.Role, cards)}, nil
	}
	var q struct {
		Scope   GroupScope `json:"scope"`
		ActorID string     `json:"actor_id"`
		Confirm bool       `json:"confirm"`
	}
	if decodeObject(input, &q) != nil || !q.Scope.valid() {
		return nil, gameError("input_invalid", "请提供完整群身份。")
	}
	if action == "cards.group.remove" {
		if !q.Confirm || q.ActorID == "" {
			return nil, gameError("input_invalid", "请确认移除此成员的交换记录。")
		}
		err := a.Groups.Update(q.Scope, func(g *GroupData) error {
			g.Cards = slices.DeleteFunc(g.Cards, func(v RoleCardOffer) bool { return v.ActorID == q.ActorID })
			return nil
		})
		return map[string]any{"removed": err == nil}, err
	}
	data, err := a.Groups.Read(q.Scope)
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	items := slices.DeleteFunc(slices.Clone(data.Cards), func(v RoleCardOffer) bool { return v.SavedMS <= now-int64(30*24*time.Hour/time.Millisecond) })
	return map[string]any{"items": items}, nil
}
func (a *App) cardsCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	if a.Game.ID != "genshin" {
		return event.Result(map[string]any{"handled": false})
	}
	group := event.Event.Target.Type == "group"
	if command == "role-cards-exchange" && !group {
		return event.SendText("圣牌交换请在目标群操作。")
	}
	if len(args) > 1 {
		return event.SendText("可提供本人UID选择角色。")
	}
	uid := ""
	if len(args) == 1 {
		uid = args[0]
	}
	client := a.accountClient(event)
	listed, err := client.List(ctx, 0)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	choice, _, err := Choose(listed, a.Game.ID, uid)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	result, cards, err := a.roleCardsQuery(ctx, client, choice)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	now := time.Now().UnixMilli()
	self := RoleCardOffer{ActorID: event.Event.Actor.ID, Nickname: plainGameText(event.Event.Actor.Nickname), UID: result.Role.UID, Region: result.Role.Region, Cards: cards, SavedMS: now}
	scope := groupScope(event)
	// Viewing or exchanging in a group enters the caller into that group's
	// exchange matching for 30 days, as upstream does.
	if group {
		err = a.Groups.Update(scope, func(g *GroupData) error {
			g.Cards = slices.DeleteFunc(g.Cards, func(v RoleCardOffer) bool {
				return v.ActorID == self.ActorID || v.SavedMS <= now-int64(30*24*time.Hour/time.Millisecond)
			})
			if len(g.Cards) >= 2000 {
				return gameError("cards_limit", "本群交换记录达到2000人上限。")
			}
			g.Cards = append(g.Cards, self)
			return nil
		})
		if err != nil {
			return event.SendText(friendlyError(err))
		}
	}
	if command == "role-cards" {
		view := roleCardView(result.Role, cards)
		if group {
			view.Note += "\n已记入本群月谕圣牌交换匹配，有效 30 天，再次查看时刷新。"
		}
		return a.sendView(ctx, event, view)
	}
	data, err := a.Groups.Read(scope)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	matches := roleCardMatches(self, data.Cards, now)
	view := View{Title: "月谕圣牌交换匹配", Subtitle: result.Role.UID, Note: "仅列双方互补且同区服的记录；数据最多保留30天。请自行与对方确认，本插件不执行赠送。"}
	for _, m := range matches {
		view.Sections = append(view.Sections, Section{Title: m.Nickname + " · " + m.ActorID + " · " + m.UID, Rows: []Row{{"你可给对方", strings.Join(m.Give, "、")}, {"对方可给你", strings.Join(m.Receive, "、")}, {"对方更新时间", calendarMS(m.SavedMS)}}})
	}
	if len(matches) == 0 {
		view.Note = "当前没有双方互补且同区服的有效记录，可请群友主动提交收藏。"
	}
	return a.sendView(ctx, event, view)
}
