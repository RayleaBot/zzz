// Entertainment draw models derived from the pinned GPL-3.0 Miao-Yunzai and
// Apache-2.0 StarRail-plugin. See distributed LICENSES and source attribution.
package app

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/localdata"
	"math/rand/v2"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SimulationBanner struct {
	ID string `json:"id"`
	PoolInfo
}
type SimulationDeck struct {
	Version        string             `json:"version"`
	FiveCharacters []string           `json:"five_characters"`
	FiveWeapons    []string           `json:"five_weapons"`
	FourCharacters []string           `json:"four_characters"`
	FourWeapons    []string           `json:"four_weapons"`
	ThreeWeapons   []string           `json:"three_weapons"`
	Banners        []SimulationBanner `json:"banners"`
}

type SimulationSelection struct {
	Kind       string `json:"kind"`
	BannerID   string `json:"banner_id"`
	Featured   string `json:"featured"`
	FateTarget string `json:"fate_target"`
	FateLimit  int    `json:"fate_limit"`
}
type SimulationPity struct {
	Five   int  `json:"five"`
	Four   int  `json:"four"`
	UpFive bool `json:"up_five"`
	UpFour bool `json:"up_four"`
	Fate   int  `json:"fate"`
}
type SimulationDraw struct {
	Model  string `json:"model"`
	Name   string `json:"name"`
	Rarity int    `json:"rarity"`
	// Kind is the pool drawn from, Item the kind of thing drawn: "character"
	// or "weapon".
	Kind       string `json:"kind"`
	Item       string `json:"item"`
	Featured   bool   `json:"featured"`
	Guaranteed bool   `json:"guaranteed"`
	Fate       bool   `json:"fate"`
	Interval   int    `json:"interval"`
	Index      int64  `json:"index"`
	TimeMS     int64  `json:"time_ms"`
}
type SimulationBatch struct {
	RequestRef string              `json:"request_ref"`
	Selection  SimulationSelection `json:"selection"`
	Draws      []SimulationDraw    `json:"draws"`
}
type SimulationState struct {
	Version    int                            `json:"version"`
	Active     string                         `json:"active"`
	Selections map[string]SimulationSelection `json:"selections"`
	Pools      map[string]*SimulationPity     `json:"pools"`
	Day        string                         `json:"day"`
	Used       int                            `json:"used"`
	Week       string                         `json:"week"`
	WeeklyTop  int                            `json:"weekly_top"`
	Total      int64                          `json:"total"`
	Owned      map[string]int64               `json:"owned"`
	History    []SimulationDraw               `json:"history"`
	Requests   []SimulationBatch              `json:"requests"`
}
type SimulationStore struct {
	mu        sync.Mutex
	Directory string
	Game      string
	// Deck is nil when the game has no banner simulator.
	Deck   *SimulationDeck
	Random func(int) int
}

func newSimulationState(deck SimulationDeck) SimulationState {
	s := SimulationState{Version: 1, Active: "character", Selections: map[string]SimulationSelection{}, Pools: map[string]*SimulationPity{}, Owned: map[string]int64{}, History: []SimulationDraw{}, Requests: []SimulationBatch{}}
	for _, kind := range []string{"character", "weapon", "standard"} {
		selection := SimulationSelection{Kind: kind}
		if kind != "standard" && len(deck.Banners) > 0 {
			b := deck.Banners[0]
			selection.BannerID = b.ID
			if kind == "character" {
				selection.Featured = b.Characters5[0]
			} else {
				selection.Featured = b.Weapons5[0]
				selection.FateTarget = b.Weapons5[0]
				selection.FateLimit = 2
			}
		}
		s.Selections[kind] = selection
		s.Pools[kind] = &SimulationPity{}
	}
	return s
}
func (s *SimulationStore) filename(scope string) string {
	sum := sha256.Sum256([]byte(scope))
	return filepath.Join(s.Directory, hex.EncodeToString(sum[:])+".json")
}
func (s *SimulationStore) read(scope string, now time.Time) (SimulationState, error) {
	if s.Deck == nil {
		return SimulationState{}, gameError("operation_denied", "此游戏未提供参考模拟抽卡。")
	}
	state := newSimulationState(*s.Deck)
	if err := localdata.Read(s.filename(scope), &state); err != nil {
		return state, err
	}
	if state.Version != 1 || state.Owned == nil || state.Selections == nil || state.Pools == nil {
		return state, gameError("simulation_invalid", "模拟档案格式不兼容。")
	}
	for _, kind := range []string{"character", "weapon", "standard"} {
		if state.Pools[kind] == nil {
			return state, gameError("simulation_invalid", "模拟档案缺少卡池状态。")
		}
	}
	dayTime := simulationDay(now)
	day := dayTime.Format("2006-01-02")
	if state.Day != day {
		state.Day = day
		state.Used = 0
	}
	year, week := dayTime.ISOWeek()
	weekKey := fmt.Sprintf("%d-%d", year, week)
	if state.Week != weekKey {
		state.Week = weekKey
		state.WeeklyTop = 0
	}
	return state, nil
}

// simulationDay shifts a time so its date is the China-server business day,
// which starts at 04:00 UTC+8.
func simulationDay(at time.Time) time.Time {
	return at.In(time.FixedZone("UTC+8", 8*60*60)).Add(-4 * time.Hour)
}
func (s *SimulationStore) limit() (int, error) {
	config := struct {
		DailyLimit int `json:"daily_limit"`
	}{100}
	if err := localdata.Read(filepath.Join(s.Directory, "config.json"), &config); err != nil {
		return 0, err
	}
	if config.DailyLimit < 10 || config.DailyLimit > 1000 {
		return 0, gameError("simulation_invalid", "每日模拟额度配置无效。")
	}
	return config.DailyLimit, nil
}
func simulationProbability(game, kind string, five, weekly int) int {
	base := 60
	if kind == "weapon" {
		base = 70
		if game == "starrail" {
			base = 80
		}
	}
	if game == "genshin" && weekly == 1 {
		if kind == "weapon" {
			base *= 3
		} else {
			base *= 2
		}
	}
	if kind != "weapon" {
		if five >= 90 {
			return 10000
		}
		if five >= 74 {
			return min(10000, 590+(five-74)*530)
		}
		if five >= 60 {
			return min(10000, 60+(five-50)*40)
		}
		return base
	}
	if five >= 80 {
		return 10000
	}
	if five >= 62 {
		return min(10000, base+(five-61)*700)
	}
	start := 45
	if game == "starrail" {
		start = 50
	}
	if five >= start {
		return min(10000, base+(five-start)*60)
	}
	if five >= 10 && five <= 20 {
		return min(10000, base+(five-10)*30)
	}
	return base
}
func simulationFourProbability(game, kind string, four int) int {
	base := 510
	if game == "starrail" && kind == "weapon" {
		base = 660
	}
	if four >= 9 {
		return 10000
	}
	if four >= 5 {
		return min(10000, base+(four-4)*(four-4)*500)
	}
	return base
}
func simulationBanner(deck SimulationDeck, selection SimulationSelection) (SimulationBanner, error) {
	if !slices.Contains([]string{"character", "weapon", "standard"}, selection.Kind) {
		return SimulationBanner{}, gameError("input_invalid", "模拟卡池种类无效。")
	}
	if selection.Kind == "standard" {
		return SimulationBanner{}, nil
	}
	for _, b := range deck.Banners {
		if b.ID == selection.BannerID {
			return b, nil
		}
	}
	return SimulationBanner{}, gameError("input_invalid", "请选择已收录的参考卡池。")
}
func (s *SimulationStore) selectPool(state *SimulationState, selection SimulationSelection) error {
	banner, err := simulationBanner(*s.Deck, selection)
	if err != nil {
		return err
	}
	if selection.Kind == "character" && len(banner.Characters4) == 0 || selection.Kind == "weapon" && len(banner.Weapons4) == 0 {
		return gameError("simulation_data_missing", "此参考期缺少四星池资料，请选择其他期次。")
	}
	if selection.Kind == "character" && !slices.Contains(banner.Characters5, selection.Featured) || selection.Kind == "weapon" && s.Game == "starrail" && !slices.Contains(banner.Weapons5, selection.Featured) {
		return gameError("input_invalid", "所选目标不在参考卡池内。")
	}
	if selection.Kind == "weapon" && s.Game == "genshin" {
		if selection.FateTarget != "" && (!slices.Contains(banner.Weapons5, selection.FateTarget) || selection.FateLimit < 1 || selection.FateLimit > 2) {
			return gameError("input_invalid", "请选择有效定轨目标及 1/2 点模型。")
		}
		old := state.Selections["weapon"]
		if old.BannerID != selection.BannerID || old.FateTarget != selection.FateTarget || old.FateLimit != selection.FateLimit {
			state.Pools["weapon"].Fate = 0
		}
	} else {
		selection.FateTarget = ""
		selection.FateLimit = 0
	}
	state.Active = selection.Kind
	state.Selections[selection.Kind] = selection
	return nil
}
func (s *SimulationStore) drawOne(state *SimulationState, selection SimulationSelection, now time.Time) SimulationDraw {
	deck := *s.Deck
	banner, _ := simulationBanner(deck, selection)
	pity := state.Pools[selection.Kind]
	random := s.Random
	if random == nil {
		random = rand.IntN
	}
	pick := func(items []string) string {
		return items[random(len(items))]
	}
	chance := func(n int) bool { return random(10000)+1 <= n }
	out := SimulationDraw{Model: s.Deck.Version, Kind: selection.Kind, Item: "weapon", TimeMS: now.UnixMilli(), Index: state.Total + 1}
	state.Total++
	state.Used++
	if chance(simulationProbability(s.Game, selection.Kind, pity.Five, state.WeeklyTop)) {
		out.Rarity = 5
		out.Interval = pity.Five + 1
		pity.Five = 0
		pity.Four++
		state.WeeklyTop++
		up := 5000
		if selection.Kind == "weapon" && s.Game == "starrail" {
			up = 7500
		}
		if pity.UpFive {
			up = 10000
		}
		if selection.Kind == "standard" {
			up = 0
		}
		if selection.Kind == "weapon" && s.Game == "genshin" && selection.FateTarget != "" && pity.Fate >= selection.FateLimit {
			out.Name = selection.FateTarget
			out.Featured = true
			out.Fate = true
			pity.UpFive = false
		} else if chance(up) {
			out.Featured = true
			out.Guaranteed = pity.UpFive
			pity.UpFive = false
			if selection.Kind == "weapon" && s.Game == "genshin" {
				out.Name = pick(banner.Weapons5)
			} else {
				out.Name = selection.Featured
				if selection.Kind == "character" {
					out.Item = "character"
				}
			}
		} else {
			if selection.Kind != "standard" {
				pity.UpFive = true
			}
			if selection.Kind == "weapon" || selection.Kind == "standard" && random(2) == 1 {
				out.Name = pick(deck.FiveWeapons)
			} else {
				out.Name = pick(deck.FiveCharacters)
				out.Item = "character"
			}
		}
		if selection.Kind == "weapon" && s.Game == "genshin" && selection.FateTarget != "" {
			if out.Name == selection.FateTarget {
				pity.Fate = 0
			} else {
				pity.Fate++
			}
		}
	} else {
		pity.Five++
		if chance(simulationFourProbability(s.Game, selection.Kind, pity.Four)) {
			out.Rarity = 4
			out.Interval = pity.Four + 1
			pity.Four = 0
			up := 5000
			if selection.Kind == "weapon" && s.Game == "genshin" {
				up = 7500
			}
			if pity.UpFour {
				up = 10000
			}
			if selection.Kind == "standard" {
				up = 0
			}
			if chance(up) {
				out.Featured = true
				out.Guaranteed = pity.UpFour
				pity.UpFour = false
				if selection.Kind == "character" {
					out.Name = pick(banner.Characters4)
					out.Item = "character"
				} else {
					out.Name = pick(banner.Weapons4)
				}
			} else {
				pity.UpFour = selection.Kind != "standard"
				if (s.Game == "genshin" || selection.Kind == "character") && random(2) == 0 {
					out.Name = pick(deck.FourCharacters)
					out.Item = "character"
				} else {
					out.Name = pick(deck.FourWeapons)
				}
			}
		} else {
			pity.Four++
			out.Rarity = 3
			out.Name = pick(deck.ThreeWeapons)
		}
	}
	state.Owned[out.Name]++
	state.History = append(state.History, out)
	if len(state.History) > 200 {
		state.History = state.History[len(state.History)-200:]
	}
	return out
}
func (s *SimulationStore) Action(scope, action string, input map[string]any, now time.Time) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.read(scope, now)
	if err != nil {
		return nil, err
	}
	limit, err := s.limit()
	if err != nil {
		return nil, err
	}
	var batch *SimulationBatch
	switch action {
	case "simulation.status":
	case "simulation.configure":
		var config struct {
			DailyLimit int `json:"daily_limit"`
		}
		if decodeObject(input, &config) != nil || config.DailyLimit < 10 || config.DailyLimit > 1000 {
			return nil, gameError("input_invalid", "每日模拟额度为 10–1000 抽。")
		}
		if err = localdata.Write(filepath.Join(s.Directory, "config.json"), config); err != nil {
			return nil, err
		}
		limit = config.DailyLimit
	case "simulation.select":
		var selection SimulationSelection
		if decodeObject(input, &selection) != nil {
			return nil, gameError("input_invalid", "模拟卡池选择无效。")
		}
		if err = s.selectPool(&state, selection); err != nil {
			return nil, err
		}
		if err = localdata.Write(s.filename(scope), state); err != nil {
			return nil, err
		}
	case "simulation.reset":
		if input["confirm"] != true {
			return nil, gameError("input_invalid", "重置模拟需要明确确认。")
		}
		state.Pools = map[string]*SimulationPity{"character": {}, "weapon": {}, "standard": {}}
		state.Owned = map[string]int64{}
		state.History = []SimulationDraw{}
		state.Total = 0
		if err = localdata.Write(s.filename(scope), state); err != nil {
			return nil, err
		}
	case "simulation.draw":
		var q struct {
			Count int    `json:"count"`
			Ref   string `json:"request_ref"`
		}
		if decodeObject(input, &q) != nil || (q.Count != 1 && q.Count != 10) || q.Ref == "" || len(q.Ref) > 256 {
			return nil, gameError("input_invalid", "每次模拟支持单抽或十连，并需要请求标识。")
		}
		for _, old := range state.Requests {
			if old.RequestRef == q.Ref {
				if len(old.Draws) != q.Count {
					return nil, gameError("simulation_conflict", "重复请求的抽数不一致。")
				}
				copy := old
				batch = &copy
				break
			}
		}
		if batch == nil {
			if state.Used+q.Count > limit {
				return nil, gameError("simulation_quota", "今日模拟额度不足，国服时间 04:00 重置。")
			}
			selection := state.Selections[state.Active]
			if err = s.selectPool(&state, selection); err != nil {
				return nil, err
			}
			value := SimulationBatch{RequestRef: q.Ref, Selection: selection, Draws: []SimulationDraw{}}
			for range q.Count {
				value.Draws = append(value.Draws, s.drawOne(&state, selection, now))
			}
			state.Requests = append(state.Requests, value)
			if len(state.Requests) > 20 {
				state.Requests = state.Requests[len(state.Requests)-20:]
			}
			if err = localdata.Write(s.filename(scope), state); err != nil {
				return nil, err
			}
			batch = &value
		}
	default:
		return nil, gameError("operation_denied", "模拟操作不存在。")
	}
	return map[string]any{"source": "simulation", "model": s.Deck.Version, "deck": s.Deck, "active": state.Active, "selections": state.Selections, "pools": state.Pools, "total": state.Total, "daily_used": state.Used, "daily_limit": limit, "daily_remaining": max(0, limit-state.Used), "weekly_top": state.WeeklyTop, "history": state.History, "owned": state.Owned, "batch": batch}, nil
}
func simulationScope(event *rayleabot.EventContext) (string, error) {
	if event.Event.EventType == "management.action" {
		return "management", nil
	}
	if event.Event.Actor.ID == "" || event.Bot.ID == "" || event.Event.Target.ID == "" {
		return "", gameError("source_invalid", "模拟档案需要完整聊天身份。")
	}
	raw, _ := json.Marshal([]string{event.Event.SourceProtocol, event.Event.SourceAdapter, event.Bot.ID, event.Event.Target.Type, event.Event.Target.ID, event.Event.Actor.ID})
	return string(raw), nil
}
func (a *App) simulationAction(event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	scope, err := simulationScope(event)
	if err != nil {
		return nil, err
	}
	if action == "simulation.configure" && event.Event.EventType != "management.action" {
		return nil, gameError("source_invalid", "每日额度仅在管理页设置。")
	}
	return a.Simulation.Action(scope, action, input, time.Now())
}
func simulationView(game Game, result map[string]any) View {
	v := View{Title: game.Name + "模拟抽卡", Rows: []Row{}, Note: "固定参考娱乐模型，不代表官方概率或当前卡池；模拟数据独立于真实抽卡档案，不消耗真实货币。"}
	if batch, _ := result["batch"].(*SimulationBatch); batch != nil {
		for _, d := range batch.Draws {
			tag := ""
			if d.Featured {
				tag = " · 目标池"
			}
			if d.Fate {
				tag += " · 定轨"
			}
			v.Rows = append(v.Rows, Row{Label: d.Name, Value: fmt.Sprintf("%d 星%s", d.Rarity, tag)})
		}
	} else {
		pools := result["pools"].(map[string]*SimulationPity)
		for _, kind := range []string{"character", "weapon", "standard"} {
			p := pools[kind]
			v.Rows = append(v.Rows, Row{Label: map[string]string{"character": "角色", "weapon": "武器/光锥", "standard": "常驻"}[kind], Value: fmt.Sprintf("连续 %d 抽未出五星 · 连续 %d 抽未出四星 · 下次五星目标池保证 %t", p.Five, p.Four, p.UpFive)})
		}
	}
	v.Rows = append(v.Rows, Row{Label: "今日剩余模拟次数", Value: asText(result["daily_remaining"])})
	return v
}
func (a *App) simulationCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	status, err := a.simulationAction(event, "simulation.status", nil)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if command == "simulation-reset" {
		if len(args) != 1 || args[0] != "确认" {
			return event.SendText("发送“" + a.Game.Prefix + "重置模拟 确认”清空模拟历史和保底；当天已用次数不会恢复。")
		}
		_, err = a.simulationAction(event, "simulation.reset", map[string]any{"confirm": true})
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.SendText("模拟历史与保底已重置，当天已用次数保留。")
	}
	if command == "simulation-fate" {
		selection := status["selections"].(map[string]SimulationSelection)["weapon"]
		banner, _ := simulationBanner(*a.Game.Data.Simulation, selection)
		target := strings.Join(args, " ")
		switch target {
		case "":
			target = simulationNextFate(banner.Weapons5, selection.FateTarget)
		case "取消":
			target = ""
		default:
			if entry, ok := a.Catalog.Resolve(target, "weapon", a.aliasMap(event)); ok {
				target = entry.Name
			}
		}
		selection.FateTarget = target
		selection.FateLimit = 2
		_, err = a.simulationAction(event, "simulation.select", asMap(selection))
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		if target == "" {
			return event.SendText("定轨已取消")
		}
		lines := []string{"定轨成功"}
		for _, weapon := range banner.Weapons5 {
			mark := "[  ] "
			if weapon == target {
				mark = "[√] "
			}
			lines = append(lines, mark+weapon)
		}
		return event.SendText(strings.Join(lines, "\n"))
	}
	if command == "simulation" {
		// As upstream, the word picks the pool: the character pool unless it
		// names 武器/光锥 or 常驻, and its second featured character on 十连2.
		word := event.Event.Command()
		kind := "character"
		if strings.Contains(word, "武器") || strings.Contains(word, "光锥") {
			kind = "weapon"
		} else if strings.Contains(word, "常驻") {
			kind = "standard"
		}
		selection := status["selections"].(map[string]SimulationSelection)[kind]
		if kind == "character" {
			banner, _ := simulationBanner(*a.Game.Data.Simulation, selection)
			pick := 0
			if strings.Contains(word, "2") {
				pick = 1
			}
			if pick < len(banner.Characters5) {
				selection.Featured = banner.Characters5[pick]
			}
		}
		if _, err = a.simulationAction(event, "simulation.select", asMap(selection)); err != nil {
			return event.SendText(friendlyError(err))
		}
		drawn, err := a.simulationAction(event, "simulation.draw", map[string]any{"count": 10, "request_ref": event.Event.EventID})
		if PublicError(err).Code == "plugin.game_simulation_quota" {
			return event.SendText(simulationLimitText(a.Game.ID, event.Event.Actor.Nickname, kind, status, time.Now()))
		}
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		status = drawn
		view := simulationView(a.Game, status)
		if batch := status["batch"].(*SimulationBatch); a.simulationImage != nil {
			pity := *status["pools"].(map[string]*SimulationPity)[batch.Selection.Kind]
			if drawn, ok := a.simulationImage(a.imageContext(ctx), SimulationImage{Word: word, Selection: batch.Selection, Draws: batch.Draws, Pity: pity}); ok {
				view.Image = &drawn
			}
		}
		return a.sendView(ctx, event, view)
	}
	return a.sendView(ctx, event, simulationView(a.Game, status))
}

// simulationNextFate is Yunzai's 定轨 step: each use moves to the pool's
// next five-star weapon, and past the last one cancels.
func simulationNextFate(weapons []string, current string) string {
	if next := slices.Index(weapons, current) + 1; next < len(weapons) {
		return weapons[next]
	}
	return ""
}

// simulationLimitText is the upstream reply once the day's draws are spent.
// Yunzai lists the day's five-stars with their pulls, or counts them past
// three, and the week's when more than one; StarRail-plugin counts the pool's
// draws and five-stars today.
func simulationLimitText(game, name, kind string, status map[string]any, now time.Time) string {
	today := simulationDay(now).Format("2006-01-02")
	count, stars := 0, []string{}
	for _, draw := range status["history"].([]SimulationDraw) {
		if simulationDay(time.UnixMilli(draw.TimeMS)).Format("2006-01-02") != today {
			continue
		}
		pool := draw.Kind == kind
		if game == "genshin" {
			// Yunzai counts the weapon pool apart from the other two.
			pool = (draw.Kind == "weapon") == (kind == "weapon")
		}
		if pool {
			count++
		}
		if draw.Rarity == 5 && (pool || game == "genshin") {
			stars = append(stars, draw.Name+"("+strconv.Itoa(draw.Interval)+")")
		}
	}
	if game != "genshin" {
		five := "没有五星"
		if len(stars) > 0 {
			five = "其中" + strconv.Itoa(len(stars)) + "个五星"
		}
		return "今日抽已抽" + strconv.Itoa(count) + "抽，" + five + "，明日再来吧"
	}
	// lodash.truncate keeps eight characters, "..." included.
	if runes := []rune(name); len(runes) > 8 {
		name = string(runes[:5]) + "..."
	}
	if len(stars) == 0 {
		return strings.Trim(name+"\n今日已抽，累计"+strconv.Itoa(count)+"抽无五星", "\n")
	}
	text := "今日五星：" + strings.Join(stars, "\n")
	if len(stars) >= 4 {
		text = "今日五星：" + strconv.Itoa(len(stars)) + "个"
	}
	if week, _ := status["weekly_top"].(int); week >= 2 {
		text += "\n本周：" + strconv.Itoa(week) + "个五星"
	}
	return strings.Trim(name+"\n"+text, "\n")
}
func asMap(value any) map[string]any {
	out := map[string]any{}
	_ = decodeObject(value, &out)
	return out
}
