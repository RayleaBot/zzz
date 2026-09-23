package app

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// Group panel ranks follow miao-plugin's ProfileRank: every panel a member
// views in a group enters that group's ranking under its UID and character,
// ranked by the equipment score.

// RankEntry is one UID's character in a group's ranking: the panel viewed last
// in the group with its equipment score and grade.
type RankEntry struct {
	Nickname    string  `json:"nickname"`
	UID         string  `json:"uid"`
	CharacterID string  `json:"character_id"`
	Name        string  `json:"name"`
	Score       float64 `json:"score"`
}

// rankLength is miao's default rankNumber, the rows a character's ranking
// shows.
const rankLength = 15

// rankEntries picks a ranking of the entries with an equipment score: a
// character's top rows by the score, or with no character each character's
// best entry, by UID, rarity and ID as miao lists them.
func (a *App) rankEntries(all []RankEntry, id string) []RankEntry {
	best := map[string]RankEntry{}
	for _, entry := range all {
		if entry.Score <= 0 || id != "" && entry.CharacterID != id {
			continue
		}
		key := entry.CharacterID
		if id != "" {
			key = entry.UID
		}
		if current, seen := best[key]; seen && current.Score >= entry.Score {
			continue
		}
		best[key] = entry
	}
	out := []RankEntry{}
	for _, entry := range best {
		out = append(out, entry)
	}
	if id != "" {
		slices.SortFunc(out, func(x, y RankEntry) int {
			return cmp.Or(cmp.Compare(y.Score, x.Score), strings.Compare(x.UID, y.UID))
		})
		return out[:min(rankLength, len(out))]
	}
	rarity := func(entry RankEntry) int {
		character, _ := a.Catalog.Get(entry.CharacterID)
		return character.Rarity
	}
	slices.SortFunc(out, func(x, y RankEntry) int {
		return cmp.Or(strings.Compare(x.UID, y.UID), cmp.Compare(rarity(y), rarity(x)), strings.Compare(x.CharacterID, y.CharacterID))
	})
	return out
}

// recordRank puts a panel viewed in a group into that group's ranking when
// its equipment score could be calculated.
func (a *App) recordRank(event *rayleabot.EventContext, uid string, panel CharacterPanel) {
	scope := groupScope(event)
	if event.Event.EventType != "message.group" || !scope.valid() || event.Event.Actor.ID == "" || uid == "" || panel.ID == "" || panel.TotalScore == nil || panel.ScoredEquipment == 0 || *panel.TotalScore == 0 {
		return
	}
	name := event.Event.Actor.Nickname
	if name == "" {
		name = event.Event.Actor.ID
	}
	// A failed record must not cost the user the panel reply.
	_ = a.Groups.Submit(scope, RankEntry{Nickname: name, UID: uid, CharacterID: panel.ID, Name: panel.Name, Score: *panel.TotalScore})
}

// Submit keeps one entry per UID and character, replacing the older one.
func (s *GroupStore) Submit(scope GroupScope, entry RankEntry) error {
	return s.Update(scope, func(data *GroupData) error {
		if entry.UID == "" || entry.CharacterID == "" || !finiteRange(entry.Score, 0, 100000) {
			return gameError("input_invalid", "排名数据不完整。")
		}
		index := slices.IndexFunc(data.Rank, func(r RankEntry) bool { return r.UID == entry.UID && r.CharacterID == entry.CharacterID })
		if index >= 0 {
			data.Rank[index] = entry
			return nil
		}
		if len(data.Rank) >= 500 {
			return gameError("rank_limit", "本群排名记录已达上限，请重置排名后再记录。")
		}
		data.Rank = append(data.Rank, entry)
		return nil
	})
}

// rankCommand answers 排行 [角色]: the group's panels by equipment score, for
// one character or every character's best. ZZZ-Plugin has no panel ranking;
// this is the plugin's own, on the group ranking miao's ProfileRank keeps.
func (a *App) rankCommand(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	scope := groupScope(event)
	if event.Event.EventType != "message.group" || event.Event.Target.Type != "group" || !scope.valid() || event.Event.Actor.ID == "" {
		// Upstream leaves these words to other plugins outside groups.
		return event.Result(map[string]any{"handled": false})
	}
	var character Entry
	if name := strings.Join(args, " "); name != "" {
		entry, ok := a.Catalog.Resolve(name, "character", a.aliasMap(event))
		if !ok {
			return event.Result(map[string]any{"handled": false})
		}
		character = entry
	}
	data, err := a.Groups.Read(scope)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	entries := a.rankEntries(data.Rank, character.ID)
	if len(entries) == 0 {
		return event.SendText("暂无排名：请通过【" + a.Game.Prefix + "面板】查看角色面板以更新排名信息...")
	}
	return a.sendView(ctx, event, rankView(a.Game, character, entries))
}

// rankView is the 排行 reply in text.
func rankView(game Game, character Entry, entries []RankEntry) View {
	title := "最高分排行"
	if character.ID != "" {
		title = character.Name + "评分排行"
	}
	v := View{Title: game.Name + " · " + title, Rows: []Row{}, Note: "本群成员在群内查看过的面板。"}
	for index, entry := range entries {
		value := fmt.Sprintf("评分 %.1f", entry.Score)
		label := strconv.Itoa(index+1) + ". " + entry.Nickname
		if character.ID == "" {
			label = entry.Name + " · " + entry.Nickname
		}
		v.Rows = append(v.Rows, Row{Label: label, Value: value + " · UID " + entry.UID})
	}
	return v
}
