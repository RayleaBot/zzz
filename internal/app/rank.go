package app

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/reference"
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

var (
	rankListWord   = regexp.MustCompile(`排名|排行|列表`)
	rankMarkWord   = regexp.MustCompile(`分|圣遗物|遗器|评分|ACE`)
	rankStrip      = regexp.MustCompile(`#|星铁|最强|最高分|第一|词条|双爆|双暴|极限|最高|最多|最牛|圣遗物|遗器|评分|群内|群|排名|排行|面板|面版|详情|榜`)
	rankResetStrip = regexp.MustCompile(`#|星铁|重置|重设|排名|排行|群|群内|面板|详情|面版`)
)

// rankMode is the ranking a command word asks for, as miao's groupRank reads
// it: equipment score ("mark"), crit or weighted-roll counts, or damage.
func rankMode(word string) string {
	switch {
	case strings.Contains(word, "双爆") || strings.Contains(word, "双暴"):
		return "crit"
	case strings.Contains(word, "词条"):
		return "valid"
	case rankMarkWord.MatchString(word):
		return "mark"
	}
	return "dmg"
}

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

// rankCommand answers miao's group rank commands: <角色>排名 lists a
// character's ranking by damage (by score with 圣遗物, 遗器 or 评分), a word
// without a character lists every character's best, 最强<角色> and
// 最高分<角色> show the top panel, and administrators reset, refresh, open or
// close the ranking.
func (a *App) rankCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	scope := groupScope(event)
	if event.Event.EventType != "message.group" || event.Event.Target.Type != "group" || !scope.valid() || event.Event.Actor.ID == "" {
		// Upstream leaves these words to other plugins outside groups.
		return event.Result(map[string]any{"handled": false})
	}
	word := event.Event.Command()
	switch command {
	case "rank-reset":
		return a.rankReset(event, scope, word)
	case "rank-refresh":
		return a.rankRefresh(ctx, event, scope)
	case "rank-switch":
		return a.rankSwitch(event, scope, word)
	}
	list, mode := rankListWord.MatchString(word), rankMode(word)
	if a.Game.ID == "zzz" {
		// ZZZ-Plugin has no panel ranking; this plugin's 排行 lists
		// equipment scores.
		mode = "mark"
	}
	name := strings.TrimSpace(rankStrip.ReplaceAllString(word, ""))
	if name == "" {
		name = strings.Join(args, " ")
	}
	var character Entry
	if name != "" {
		entry, ok := a.Catalog.Resolve(name, "character", a.aliasMap(event))
		if !ok {
			return event.Result(map[string]any{"handled": false})
		}
		character = entry
	} else if !list {
		return event.Result(map[string]any{"handled": false})
	}
	data, err := a.Groups.Read(scope)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if data.RankOff {
		return event.SendText("本群已关闭群排名，群管理员或Bot主人可通过【" + a.Game.Prefix + "启用排名】启用...")
	}
	entries := a.rankEntries(data.Rank, character.ID, mode)
	if len(entries) == 0 {
		if mode == "dmg" && character.ID != "" && !a.hasDamageRule(character.ID) {
			return event.SendText("暂无排名：" + character.Name + "暂不支持伤害计算，无法进行排名..")
		}
		return event.SendText("暂无排名：请通过【" + a.Game.Prefix + "面板】查看角色面板以更新排名信息...")
	}
	if !list {
		top := entries[0]
		return a.sendView(ctx, event, a.fullPanelView(ctx, event, *top.Panel, top.UID, false, ""))
	}
	view := rankView(a.Game, character, mode, entries)
	if a.rankImage != nil {
		for index := range entries {
			// The host shows OneBot members' avatars from QQ's avatar service.
			if scope.Protocol == "onebot11" {
				entries[index].Avatar = "https://q1.qlogo.cn/g?b=qq&nk=" + entries[index].ActorID + "&s=100"
			}
		}
		if drawn, ok := a.rankImage(a.imageContext(ctx), RankImage{Word: word, Mode: mode, Character: character, Entries: entries, SinceMS: data.RankSinceMS}); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
}

// hasDamageRule reports whether the character has a pinned damage rule.
func (a *App) hasDamageRule(id string) bool {
	if a.Game.Calc == nil {
		return false
	}
	return slices.ContainsFunc(a.Game.Calc.Metadata().Characters, func(record reference.Character) bool { return record.ID == id && record.Script != "" })
}

// rankReset is miao's 重置排名: the bot's super administrators clear the
// group's ranking, or one character's.
func (a *App) rankReset(event *rayleabot.EventContext, scope GroupScope, word string) error {
	if !slices.Contains(event.SuperAdmins, event.Event.Actor.ID) {
		return event.SendText("只有管理员可重置排名")
	}
	name := strings.TrimSpace(rankResetStrip.ReplaceAllString(word, ""))
	id, label := "", "全部角色"
	if name != "" {
		character, ok := a.Catalog.Resolve(name, "character", a.aliasMap(event))
		if !ok {
			return event.SendText("重置排名失败，角色：" + name + "不存在")
		}
		id, label = character.ID, character.Name
	}
	err := a.Groups.Update(scope, func(data *GroupData) error {
		data.Rank = slices.DeleteFunc(data.Rank, func(entry RankEntry) bool { return id == "" || entry.CharacterID == id })
		if id == "" {
			data.RankSinceMS = time.Now().UnixMilli()
		}
		return nil
	})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText("本群" + label + "排名已重置...")
}

// rankSwitch is miao's 开启排名 and 关闭排名 for group administrators.
func (a *App) rankSwitch(event *rayleabot.EventContext, scope GroupScope, word string) error {
	closing := strings.Contains(word, "关闭") || strings.Contains(word, "禁用")
	if !groupAdministrator(event) {
		action := "启用"
		if closing {
			action = "禁用"
		}
		return event.SendText("只有主人及群管理员可" + action + "排名...")
	}
	if err := a.Groups.Update(scope, func(data *GroupData) error { data.RankOff = closing; return nil }); err != nil {
		return event.SendText(friendlyError(err))
	}
	if closing {
		return event.SendText("当前群排名功能已禁用...")
	}
	return event.SendText("当前群排名功能已启用...\n如数据有问题可通过【" + a.Game.Prefix + "刷新排名】命令来刷新当前群内排名")
}

// rankRefresh is miao's 刷新排名: every recorded panel is scored and
// calculated again with the pinned rules, four at a time.
func (a *App) rankRefresh(ctx context.Context, event *rayleabot.EventContext, scope GroupScope) error {
	if !groupAdministrator(event) {
		return event.SendText("只有主人及群管理员可刷新排名...")
	}
	data, err := a.Groups.Read(scope)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	_, _ = event.Actions().MessageSend(ctx, rayleabot.MessageSendRequest{SourceProtocol: scope.Protocol, SourceAdapter: scope.Adapter, TargetType: "group", TargetID: scope.GroupID,
		Message: rayleabot.MessageOut{Segments: []rayleabot.Segment{rayleabot.Text("面板数据刷新中，等待时间可能较长，请耐心等待...")}}})
	refreshed := make([]RankEntry, len(data.Rank))
	var group sync.WaitGroup
	slots := make(chan struct{}, 4)
	for index, entry := range data.Rank {
		refreshed[index] = entry
		if entry.Panel == nil {
			continue
		}
		group.Add(1)
		slots <- struct{}{}
		go func() {
			defer func() { <-slots; group.Done() }()
			// As miao reloads each UID's data, a panel kept since is ranked.
			panel := *entry.Panel
			if saved, err := a.Profiles.Read(entry.UID); err == nil {
				if kept, ok := saved.Panels[entry.CharacterID]; ok {
					panel = kept.panel()
				}
			}
			if scored, err := a.scorePanel(ctx, panel); err == nil {
				panel = scored
			}
			var damage *BuildResult
			if result, err := a.panelDamage(ctx, panel); err == nil {
				damage = &result
			}
			a.rankScores(&refreshed[index], panel, damage)
		}()
	}
	group.Wait()
	uids := map[string]bool{}
	err = a.Groups.Update(scope, func(current *GroupData) error {
		for _, entry := range refreshed {
			index := slices.IndexFunc(current.Rank, func(r RankEntry) bool { return r.UID == entry.UID && r.CharacterID == entry.CharacterID })
			if index >= 0 && entry.Panel != nil {
				current.Rank[index] = entry
				uids[entry.UID] = true
			}
		}
		return nil
	})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText("本群排名已刷新，共刷新" + strconv.Itoa(len(uids)) + "个UID数据...")
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

// RankSetName is miao's set label: a single set by its name and piece count
// when that fits in seven characters, otherwise up to two sets by their
// abbreviations. Sets upstream gives no abbreviation keep their name.
func RankSetName(catalog Catalog, panel CharacterPanel) string {
	counts, order := map[string]int{}, []string{}
	for _, piece := range panel.Equipment {
		if piece.SetName == "" {
			continue
		}
		if counts[piece.SetName] == 0 {
			order = append(order, piece.SetName)
		}
		counts[piece.SetName]++
	}
	short, full := []string{}, []string{}
	for _, name := range order {
		if counts[name] < 2 {
			continue
		}
		count := "2"
		if counts[name] >= 4 {
			count = "4"
		}
		abbr := catalog.SetAbbrs[name]
		if abbr == "" {
			abbr = name
		}
		short, full = append(short, abbr+count), append(full, name+count)
	}
	if len(full) == 0 {
		return ""
	}
	if len(short) > 1 || utf8.RuneCountInString(full[0]) > 7 {
		return strings.Join(short[:min(2, len(short))], "+")
	}
	return full[0]
}

// RankDamageTitle shortens a detail title as miao's rank list does: past ten
// characters without spaces and dots, then without a trailing 伤害.
func RankDamageTitle(title string) string {
	if utf8.RuneCountInString(title) > 10 {
		title = strings.NewReplacer(" ", "", "·", "").Replace(title)
	}
	if utf8.RuneCountInString(title) > 10 {
		title = strings.TrimSuffix(title, "伤害")
	}
	return title
}
