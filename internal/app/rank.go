package app

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// Group panel ranks follow miao-plugin's ProfileRank: every panel a member
// views in a group enters that group's ranking under its UID and character,
// ranked by the expected damage of the character's default detail or by the
// equipment score.

// RankEntry is one UID's character in a group's ranking: the panel viewed last
// in the group with its equipment score and grade and the damage of the
// default detail.
type RankEntry struct {
	ActorID     string          `json:"actor_id"`
	Nickname    string          `json:"nickname"`
	UID         string          `json:"uid"`
	CharacterID string          `json:"character_id"`
	Name        string          `json:"name"`
	Score       float64         `json:"score"`
	Grade       string          `json:"grade,omitempty"`
	Damage      *RankDamage     `json:"damage,omitempty"`
	Panel       *CharacterPanel `json:"panel,omitempty"`
	UpdatedAtMS int64           `json:"updated_at_ms"`
	// Avatar is the member's chat avatar for rank images; it is not stored.
	Avatar string `json:"-"`
}

// RankDamage is the default detail's title and expected damage; Text keeps
// the result of a detail upstream shows as text.
type RankDamage struct {
	Title string  `json:"title"`
	Value float64 `json:"value"`
	Text  string  `json:"text,omitempty"`
}

// rankLength is miao's default rankNumber, the rows a character's ranking
// shows.
const rankLength = 15

// rankValue is what an entry ranks by in a mode. miao's score detail no
// longer carries crit or weighted-roll counts, so upstream's 双爆 and 词条
// rankings stay empty; they do here too.
func rankValue(entry RankEntry, mode string) (float64, bool) {
	switch mode {
	case "dmg":
		if entry.Damage == nil {
			return 0, false
		}
		return entry.Damage.Value, true
	case "mark":
		return entry.Score, entry.Score > 0
	}
	return 0, false
}

// rankEntries picks a ranking: a character's top rows by the mode's value, or
// with no character each character's best entry, by UID, rarity and ID as
// miao lists them.
func (a *App) rankEntries(all []RankEntry, id, mode string) []RankEntry {
	best := map[string]RankEntry{}
	for _, entry := range all {
		if entry.Panel == nil || id != "" && entry.CharacterID != id {
			continue
		}
		value, ok := rankValue(entry, mode)
		if !ok {
			continue
		}
		key := entry.CharacterID
		if id != "" {
			key = entry.UID
		}
		if current, seen := best[key]; seen {
			if old, _ := rankValue(current, mode); old >= value {
				continue
			}
		}
		best[key] = entry
	}
	out := []RankEntry{}
	for _, entry := range best {
		out = append(out, entry)
	}
	if id != "" {
		slices.SortFunc(out, func(x, y RankEntry) int {
			a, _ := rankValue(x, mode)
			b, _ := rankValue(y, mode)
			return cmp.Or(cmp.Compare(b, a), strings.Compare(x.UID, y.UID))
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

// recordRank puts a panel viewed in a group into that group's ranking, with
// its equipment score and the damage of the default detail when either could
// be calculated.
func (a *App) recordRank(event *rayleabot.EventContext, uid string, panel CharacterPanel, damage *BuildResult) {
	scope := groupScope(event)
	if event.Event.EventType != "message.group" || !scope.valid() || event.Event.Actor.ID == "" || uid == "" || panel.ID == "" {
		return
	}
	name := event.Event.Actor.Nickname
	if name == "" {
		name = event.Event.Actor.ID
	}
	entry := RankEntry{ActorID: event.Event.Actor.ID, Nickname: name, UID: uid, CharacterID: panel.ID, Name: panel.Name, UpdatedAtMS: time.Now().UnixMilli()}
	a.rankScores(&entry, panel, damage)
	if entry.Score == 0 && entry.Damage == nil {
		return
	}
	// A failed record must not cost the user the panel reply.
	_ = a.Groups.Submit(scope, entry)
}

// rankScores fills an entry's panel, score and damage from a scored panel and
// its damage.
func (a *App) rankScores(entry *RankEntry, panel CharacterPanel, damage *BuildResult) {
	stored := panel
	entry.Panel = &stored
	entry.Score, entry.Grade, entry.Damage = 0, "", nil
	if panel.TotalScore != nil && panel.ScoredEquipment > 0 {
		entry.Score = *panel.TotalScore
		if panel.ScoreDetail != nil {
			entry.Grade = panel.ScoreDetail.Grade
		}
	}
	if damage == nil {
		return
	}
	for _, result := range damage.Baseline.Results {
		if !result.Default {
			continue
		}
		if result.Expected != nil {
			entry.Damage = &RankDamage{Title: result.Title, Value: *result.Expected}
		} else if value, err := strconv.ParseFloat(strings.ReplaceAll(result.Text, "%", ""), 64); err == nil {
			// miao ranks a text result by its number, without the percent sign.
			entry.Damage = &RankDamage{Title: result.Title, Value: value, Text: result.Text}
		}
	}
}

// Submit keeps one entry per UID and character, replacing the older one.
func (s *GroupStore) Submit(scope GroupScope, entry RankEntry) error {
	return s.Update(scope, func(data *GroupData) error {
		if entry.UID == "" || entry.CharacterID == "" || entry.Panel == nil || !finiteRange(entry.Score, 0, 100000) || entry.Damage != nil && math.IsInf(entry.Damage.Value, 0) {
			return gameError("input_invalid", "排名数据不完整。")
		}
		if data.RankSinceMS == 0 {
			data.RankSinceMS = entry.UpdatedAtMS
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
	mode := "mark"
	entries := a.rankEntries(data.Rank, character.ID, mode)
	if len(entries) == 0 {
		return event.SendText("暂无排名：请通过【" + a.Game.Prefix + "面板】查看角色面板以更新排名信息...")
	}
	view := rankView(a.Game, character, mode, entries)
	if a.rankImage != nil {
		for index := range entries {
			// The host shows OneBot members' avatars from QQ's avatar service.
			if scope.Protocol == "onebot11" {
				entries[index].Avatar = "https://q1.qlogo.cn/g?b=qq&nk=" + entries[index].ActorID + "&s=100"
			}
		}
		if drawn, ok := a.rankImage(a.imageContext(ctx), RankImage{Word: event.Event.Command(), Mode: mode, Character: character, Entries: entries, SinceMS: data.RankSinceMS}); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
}

// rankView is the ranking in text, for replies without images.
func rankView(game Game, character Entry, mode string, entries []RankEntry) View {
	title := "最强排行"
	if mode == "mark" {
		title = "最高分排行"
	}
	if character.ID != "" {
		title = character.Name + map[string]string{"mark": "评分"}[mode] + "排行"
	}
	v := View{Title: game.Name + " · " + title, Rows: []Row{}, Note: "本群成员在群内查看过的面板。"}
	for index, entry := range entries {
		value := fmt.Sprintf("评分 %.1f", entry.Score)
		if mode == "dmg" && entry.Damage != nil {
			value = entry.Damage.Title + " " + entry.Damage.Text
			if entry.Damage.Text == "" {
				value = entry.Damage.Title + " " + strconv.FormatFloat(entry.Damage.Value, 'f', 1, 64)
			}
		}
		label := strconv.Itoa(index+1) + ". " + entry.Nickname
		if character.ID == "" {
			label = entry.Name + " · " + entry.Nickname
		}
		v.Rows = append(v.Rows, Row{Label: label, Value: value + " · UID " + entry.UID})
	}
	return v
}
