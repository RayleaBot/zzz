package app

import (
	"cmp"
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/localdata"
)

// ZZZ-Plugin's remind app: 开启挑战提醒 checks 式舆防卫战 by the floors rated
// S and 危局强袭战 by its stars at a daily or weekly time, and sends the user
// one message when either falls short. Here that is one challenge reminder
// of the user's role, marked Pair, holding a delegation for each of the two
// operations under its task. A user's own thresholds and time override the
// global ones in the plugin settings; changing either moves the task.

// challengeRemindOff is upstream's reply while the global switch is off.
const challengeRemindOff = "当前未启用防卫战/危局挑战提醒功能"

// challengeGlobals are the global switch, time and thresholds, as
// remind.yaml keeps them; values out of range fall back to upstream's
// defaults.
type challengeGlobals struct {
	Enabled bool
	Time    string
	Abyss   int
	Deadly  int
}

func (s Settings) challengeGlobals() challengeGlobals {
	value := challengeGlobals{Enabled: s.ChallengeRemind, Time: s.ChallengeRemindTime, Abyss: s.ChallengeAbyssLevel, Deadly: s.ChallengeDeadlyStars}
	if _, _, _, ok := parseChallengeTime(value.Time); !ok {
		value.Time = "每日20时"
	}
	if value.Abyss < 0 || value.Abyss > 6 {
		value.Abyss = 5
	}
	if value.Deadly < 0 || value.Deadly > 9 {
		value.Deadly = 6
	}
	return value
}

// ChallengePreference is a user's own thresholds and time; unset ones
// follow the global settings.
type ChallengePreference struct {
	Owner       Subject `json:"owner"`
	AbyssLevel  *int    `json:"abyss_level,omitempty"`
	DeadlyStars *int    `json:"deadly_stars,omitempty"`
	RemindTime  string  `json:"remind_time,omitempty"`
}

// targets are the thresholds and time in effect for the user.
func (p ChallengePreference) targets(global challengeGlobals) (abyss, deadly int, remindTime string) {
	abyss, deadly, remindTime = global.Abyss, global.Deadly, global.Time
	if p.AbyssLevel != nil {
		abyss = *p.AbyssLevel
	}
	if p.DeadlyStars != nil {
		deadly = *p.DeadlyStars
	}
	if p.RemindTime != "" {
		remindTime = p.RemindTime
	}
	return abyss, deadly, remindTime
}

// ChallengePreferences keeps every user's preference in one file.
type ChallengePreferences struct {
	mu   sync.Mutex
	Path string
}

func (s *ChallengePreferences) read() ([]ChallengePreference, error) {
	items := []ChallengePreference{}
	err := localdata.Read(s.Path, &items)
	return items, err
}

// Get returns the user's preference and whether the user set any.
func (s *ChallengePreferences) Get(owner Subject) (ChallengePreference, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if index := slices.IndexFunc(items, func(p ChallengePreference) bool { return p.Owner == owner }); err == nil && index >= 0 {
		return items[index], true, nil
	}
	return ChallengePreference{Owner: owner}, false, err
}

// List returns every preference by user.
func (s *ChallengePreferences) List() (map[Subject]ChallengePreference, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	out := map[Subject]ChallengePreference{}
	for _, item := range items {
		out[item.Owner] = item
	}
	return out, err
}

// Edit changes the user's preference; edit reports whether to write it.
func (s *ChallengePreferences) Edit(owner Subject, edit func(*ChallengePreference) bool) (ChallengePreference, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return ChallengePreference{}, err
	}
	index := slices.IndexFunc(items, func(p ChallengePreference) bool { return p.Owner == owner })
	value := ChallengePreference{Owner: owner}
	if index >= 0 {
		value = items[index]
	}
	if !edit(&value) {
		return value, nil
	}
	if index >= 0 {
		items[index] = value
	} else {
		if len(items) >= 4096 {
			return value, gameError("reminder_limit", "提醒设置数量已满。")
		}
		items = append(items, value)
	}
	return value, localdata.Write(s.Path, items)
}

func challengePreferences(directory string) *ChallengePreferences {
	return &ChallengePreferences{Path: filepath.Join(directory, "challenge-preferences.json")}
}

// abyssSLayers counts the 式舆防卫战 floors rated S as ZZZ-Plugin does: the
// fifth counts with S or S+, and floors skipped by starting higher count when
// the first floor played is rated S. fifth is the fifth floor's rating.
func abyssSLayers(abyss map[string]any) (count int, fifth string) {
	fifth = asText(fieldAt(abyss, "brief.rating"))
	if strings.HasPrefix(fifth, "S") {
		count = 1
	}
	layer := func(name string) (map[string]any, bool) {
		detail, ok := abyss[name+"_layer_detail"].(map[string]any)
		return detail, ok
	}
	first, hasFirst := layer("first")
	second, hasSecond := layer("second")
	third, hasThird := layer("third")
	fourth, hasFourth := layer("fourth")
	if !hasThird && hasFourth && asText(fourth["rating"]) == "S" {
		return count + 4, fifth
	}
	if !hasSecond && hasThird && asText(third["rating"]) == "S" {
		count += 2
	} else if !hasFirst && hasSecond && asText(second["rating"]) == "S" {
		count++
	}
	for _, detail := range []map[string]any{first, second, third, fourth} {
		if asText(detail["rating"]) == "S" {
			count++
		}
	}
	return count, fifth
}

// challengePairLines are ZZZ-Plugin's lines for one mode of the pair:
// "challenge" against the S-rated floors (6 also asks for S+ on the fifth)
// or "deadly" against the stars. all lists the state whatever it is, as
// 查询挑战状态 does; otherwise nothing is listed when the target is met.
func challengePairLines(kind string, threshold int, data map[string]any, all bool, now time.Time) []string {
	lines := []string{}
	var begin, end map[string]any
	switch kind {
	case "challenge":
		abyss := asObject(data["hadal_info_v2"])
		if abyss == nil {
			lines = append(lines, "式舆防卫战S评级: 0/5")
			break
		}
		begin, end = asObject(abyss["hadal_begin_time"]), asObject(abyss["hadal_end_time"])
		count, fifth := abyssSLayers(abyss)
		shown := fifth
		if shown == "" {
			shown = "无"
		}
		switch {
		case threshold == 0:
			if all {
				lines = append(lines, fmt.Sprintf("式舆防卫战S评级: %d/5", count), "第五层评价: "+shown)
			}
		case threshold < 6:
			status := ""
			if count >= threshold {
				status = " ✓"
			}
			if all || count < threshold {
				lines = append(lines, fmt.Sprintf("式舆防卫战S评级: %d/5%s", count, status))
				if fifth != "" {
					lines = append(lines, "第五层评价: "+fifth)
				}
			}
		default:
			status := ""
			if fifth == "S+" {
				status = " ✓"
			}
			if all || count < 5 || fifth != "S+" {
				lines = append(lines, fmt.Sprintf("式舆防卫战S评级: %d/5%s", count, status), "第五层评价: "+shown+status)
			}
		}
	case "deadly":
		begin, end = asObject(data["start_time"]), asObject(data["end_time"])
		if data["has_data"] != true {
			lines = append(lines, "危局强袭战星数: 0/9")
			break
		}
		stars := Int(data["total_star"])
		status := ""
		if threshold > 0 && stars >= threshold {
			status = " ✓"
		}
		if all || threshold > 0 && stars < threshold {
			lines = append(lines, fmt.Sprintf("危局强袭战星数: %d/9%s", stars, status))
		}
	}
	if len(lines) > 0 && begin != nil && end != nil {
		date := func(value map[string]any) string {
			return asText(value["year"]) + "/" + asText(value["month"]) + "/" + asText(value["day"])
		}
		china := time.FixedZone("UTC+8", 8*3600)
		until := time.Date(Int(end["year"]), time.Month(Int(end["month"])), Int(end["day"]), Int(end["hour"]), Int(end["minute"]), Int(end["second"]), 0, china)
		hours := max(0, int(until.Sub(now)/time.Hour))
		lines = append(lines, "统计周期："+date(begin)+" - "+date(end), fmt.Sprintf("刷新剩余: %d天%d小时", hours/24, hours%24))
	}
	return lines
}

// challengePairDescription is upstream's account of what the pair checks.
func challengePairDescription(abyss, deadly int) string {
	abyssText := "式舆防卫战不提醒"
	if abyss > 0 {
		abyssText = "防卫战S评级<" + strconv.Itoa(min(5, abyss)) + "层"
		if abyss == 6 {
			abyssText += "或第五层<S+评价"
		}
		abyssText += "进行提醒"
	}
	deadlyText := "危局强袭战不提醒"
	if deadly > 0 {
		deadlyText = "危局<" + strconv.Itoa(deadly) + "星进行提醒"
	}
	return abyssText + "，" + deadlyText
}

// challengePairTasks are the user's 开启挑战提醒 tasks.
func (a *App) challengePairTasks(owner Subject) ([]Reminder, error) {
	items, err := a.Reminders.List()
	return slices.DeleteFunc(items, func(task Reminder) bool { return !task.Pair || task.Kind != "challenge" || task.Owner != owner }), err
}

// syncChallengePairs gives the pair tasks of owner, or of every user when
// owner is nil, the thresholds and time in effect under global.
func (a *App) syncChallengePairs(global challengeGlobals, owner *Subject) error {
	items, err := a.Reminders.List()
	if err != nil {
		return err
	}
	preferences, err := a.ChallengePrefs.List()
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	for _, task := range items {
		if !task.Pair || task.Kind != "challenge" || owner != nil && task.Owner != *owner {
			continue
		}
		preference := preferences[task.Owner]
		abyss, deadly, remindTime := preference.targets(global)
		hour, minute, weekday, _ := parseChallengeTime(remindTime)
		if task.Threshold == float64(abyss) && task.DeadlyThreshold == float64(deadly) && task.Hour == hour && task.Minute == minute && task.Weekday == weekday {
			continue
		}
		err = a.Reminders.edit(task.Ref, func(items *[]Reminder, i int) error {
			if i < 0 {
				return nil
			}
			current := &(*items)[i]
			current.Threshold, current.DeadlyThreshold = float64(abyss), float64(deadly)
			if current.Hour != hour || current.Minute != minute || current.Weekday != weekday {
				current.Hour, current.Minute, current.Weekday = hour, minute, weekday
				current.NextCheckMS = nextChallengeCheck(now, hour, minute, weekday)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// challengePairModes are the two modes a pair checks, in upstream's order:
// the operation's kind, the mode's name and whether it takes the 危局
// threshold.
var challengePairModes = []struct {
	kind, name string
	deadly     bool
}{{"challenge", "式舆防卫战", false}, {"deadly", "危局强袭战", true}}

// queryChallengePair reads both modes of a pair task, each under its own
// delegation; a mode whose threshold is 0 is not read. The result holds each
// mode's official data by kind, and under "errors" the reply for a mode that
// failed; the error is returned only when every mode read failed.
func (a *App) queryChallengePair(ctx context.Context, client AccountsClient, task Reminder) (QueryResult, error) {
	result := QueryResult{Data: map[string]any{}}
	failures := map[string]any{}
	read := 0
	var last error
	for _, mode := range challengePairModes {
		threshold, delegation := task.Threshold, task.DelegationRef
		if mode.deadly {
			threshold, delegation = task.DeadlyThreshold, task.DeadlyDelegationRef
		}
		if threshold <= 0 {
			continue
		}
		read++
		var answer QueryResult
		err := client.call(ctx, "execute", map[string]any{"account_ref": task.AccountRef, "role_ref": task.RoleRef, "operation": a.Game.ID + "." + mode.kind, "input": map[string]any{}, "delegation_ref": delegation}, &answer)
		if err != nil {
			failures[mode.kind], last = friendlyError(err), err
			continue
		}
		result.Role, result.Data[mode.kind] = answer.Role, answer.Data
	}
	if read > 0 && len(failures) == read {
		return result, last
	}
	if len(failures) > 0 {
		result.Data["errors"] = failures
	}
	return result, nil
}

// challengePairReport is upstream's checkUser: the lines of both modes, a
// mode that could not be read named as failed. Unless all is set, a mode
// whose threshold is 0 is left out.
func challengePairReport(abyss, deadly int, data map[string]any, all bool, now time.Time) []string {
	lines := []string{}
	failures := asObject(data["errors"])
	for _, mode := range challengePairModes {
		threshold := abyss
		if mode.deadly {
			threshold = deadly
		}
		if threshold <= 0 && !all {
			continue
		}
		if failure := asText(failures[mode.kind]); failure != "" {
			lines = append(lines, mode.name+"查询失败: "+failure)
			continue
		}
		lines = append(lines, challengePairLines(mode.kind, threshold, asObject(data[mode.kind]), all, now)...)
	}
	return lines
}

// tickChallengePair runs a 开启挑战提醒 task: both modes are checked
// together, and one message lists what falls short, as upstream's.
func (s *ReminderStore) tickChallengePair(task *Reminder, now int64, query func(Reminder) (QueryResult, error), send func(Reminder, string) error) error {
	task.LastCheckedMS = now
	task.NextCheckMS = nextChallengeCheck(now, task.Hour, task.Minute, task.Weekday)
	if task.Threshold <= 0 && task.DeadlyThreshold <= 0 {
		// As ZZZ-Plugin, thresholds of 0 check nothing.
		task.LastCode = "threshold_off"
		return s.save(*task)
	}
	result, err := query(*task)
	if err != nil {
		task.failCheck(now, err)
		return s.save(*task)
	}
	lines := challengePairReport(int(task.Threshold), int(task.DeadlyThreshold), result.Data, false, time.UnixMilli(now))
	if len(lines) == 0 {
		task.LastCode = "target_met"
		return s.save(*task)
	}
	task.LastAttemptMS = now
	task.LastCode = "notification_attempted"
	if err = s.save(*task); err != nil {
		return err
	}
	if err = send(*task, "【式舆/危局挑战提醒】\n"+strings.Join(lines, "\n")); err != nil {
		task.LastCode = "notification_failed"
	} else {
		task.LastCode = "notified"
	}
	return s.save(*task)
}

// challengeModeWord tells a 阈值 command's mode word: false for 式舆防卫战,
// true for 危局强袭战.
func challengeModeWord(word string) bool {
	return strings.Contains(word, "危局") || strings.Contains(word, "强袭")
}

// challengePairCommand answers ZZZ-Plugin's remind commands: 开启挑战提醒,
// 查询挑战状态, the user's own thresholds and time, and for super
// administrators the global switch, thresholds and time.
func (a *App) challengePairCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	config := settings(event)
	global := config.challengeGlobals()
	owner := chatOwner(event)
	switch command {
	case "challenge-global-switch":
		enable := len(args) > 0 && (args[0] == "开启" || args[0] == "启用")
		if global.Enabled == enable {
			if enable {
				return event.SendText("全局防卫战/危局挑战提醒功能已启用，请勿重复操作")
			}
			return event.SendText("全局防卫战/危局挑战提醒功能已禁用，请勿重复操作")
		}
		if _, err := event.Actions().ConfigWrite(ctx, map[string]any{"challenge_remind_enabled": enable}); err != nil {
			return event.SendText(friendlyError(err))
		}
		if enable {
			return event.SendText("全局防卫战/危局挑战提醒功能已启用")
		}
		return event.SendText("全局防卫战/危局挑战提醒功能已禁用")
	case "challenge-global-time-status":
		return event.SendText("当前全局提醒时间: " + global.Time)
	case "challenge-global-time":
		remindTime, ok, reply := challengeRemindTime(args)
		if !ok {
			return event.Result(map[string]any{"handled": false})
		}
		if reply != "" {
			return event.SendText(reply)
		}
		if _, err := event.Actions().ConfigWrite(ctx, map[string]any{"challenge_remind_time": remindTime}); err != nil {
			return event.SendText(friendlyError(err))
		}
		global.Time = remindTime
		if err := a.syncChallengePairs(global, nil); err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.SendText("全局提醒时间已更新为: " + remindTime)
	case "challenge-global-threshold", "challenge-threshold":
		isGlobal := command == "challenge-global-threshold"
		value, ok := numberArg(args, 2)
		if !ok {
			return event.Result(map[string]any{"handled": false})
		}
		if !isGlobal && !global.Enabled {
			return event.SendText(challengeRemindOff)
		}
		deadly := challengeModeWord(args[0])
		if !deadly && (value < 0 || value > 6) {
			return event.SendText("防卫战阈值必须在0到6之间，设置为0时不提醒式舆防卫战")
		}
		if deadly && (value < 0 || value > 9) {
			return event.SendText("危局阈值必须在0到9之间，设置为0时不提醒危局强袭战")
		}
		var err error
		switch {
		case isGlobal && deadly:
			_, err = event.Actions().ConfigWrite(ctx, map[string]any{"challenge_deadly_stars": value})
			global.Deadly = value
		case isGlobal:
			_, err = event.Actions().ConfigWrite(ctx, map[string]any{"challenge_abyss_level": value})
			global.Abyss = value
		default:
			_, err = a.ChallengePrefs.Edit(owner, func(p *ChallengePreference) bool {
				if deadly {
					p.DeadlyStars = &value
				} else {
					p.AbyssLevel = &value
				}
				return true
			})
		}
		if err == nil {
			scope := &owner
			if isGlobal {
				scope = nil
			}
			err = a.syncChallengePairs(global, scope)
		}
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		prefix := ""
		if isGlobal {
			prefix = "全局默认"
		}
		switch {
		case value == 0 && deadly:
			return event.SendText(prefix + "危局强袭战挑战提醒已关闭")
		case value == 0:
			return event.SendText(prefix + "式舆防卫战挑战提醒已关闭")
		case deadly:
			return event.SendText(prefix + "危局强袭战挑战提醒阈值已设为: <" + strconv.Itoa(value) + "星时提醒")
		}
		text := prefix + "式舆防卫战挑战提醒阈值已设为：S层数<" + strconv.Itoa(min(5, value)) + "层"
		if value == 6 {
			text += "或第五层<S+评价"
		}
		return event.SendText(text + "时提醒")
	case "challenge-time":
		remindTime, ok, reply := challengeRemindTime(args)
		if !ok {
			return event.Result(map[string]any{"handled": false})
		}
		if !global.Enabled {
			return event.SendText(challengeRemindOff)
		}
		if reply != "" {
			return event.SendText(reply)
		}
		_, err := a.ChallengePrefs.Edit(owner, func(p *ChallengePreference) bool {
			p.RemindTime = remindTime
			return true
		})
		if err == nil {
			err = a.syncChallengePairs(global, &owner)
		}
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.SendText("您的个人提醒时间已设置为: " + remindTime)
	case "challenge-time-reset":
		if !global.Enabled {
			return event.SendText(challengeRemindOff)
		}
		had := false
		_, err := a.ChallengePrefs.Edit(owner, func(p *ChallengePreference) bool {
			had = p.RemindTime != ""
			p.RemindTime = ""
			return had
		})
		if err == nil && had {
			err = a.syncChallengePairs(global, &owner)
		}
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		if !had {
			return event.SendText("个人提醒时间尚未设置")
		}
		return event.SendText("个人提醒时间已重置为全局默认时间")
	case "challenge-time-status":
		preference, set, err := a.ChallengePrefs.Get(owner)
		tasks, listErr := a.challengePairTasks(owner)
		if err != nil || listErr != nil {
			return event.SendText(friendlyError(cmp.Or(err, listErr)))
		}
		lines := []string{}
		if preference.RemindTime != "" {
			lines = append(lines, "当前个人提醒时间: "+preference.RemindTime)
		} else {
			lines = append(lines, "个人提醒时间未设置，默认使用全局时间: "+global.Time)
		}
		if set || len(tasks) > 0 {
			abyss, deadly, remindTime := preference.targets(global)
			if slices.ContainsFunc(tasks, func(task Reminder) bool { return task.Enabled }) {
				lines = append(lines, "当前提醒功能已开启\n将在"+remindTime+"对"+challengePairDescription(abyss, deadly))
			} else {
				lines = append(lines, "当前提醒功能已关闭")
			}
		}
		return event.SendText(strings.Join(lines, "\n"))
	case "challenge-enable":
		return a.challengePairEnable(ctx, event, global, owner, args)
	case "challenge-check":
		return a.challengePairCheck(ctx, event, global, owner, args)
	}
	return event.Result(map[string]any{"handled": false})
}

// challengeRemindTimes are upstream's 提醒时间 forms, 每日20时30分 and
// 每周六20时.
var challengeRemindTimes = regexp.MustCompile(`^(?:每日|每周.)[0-9]+时(?:[0-9]+分)?$`)

// challengeRemindTime reads the time of a 提醒时间 command: ok is false when
// the command holds no time in upstream's forms, which upstream's rule does
// not match, and reply is upstream's when the time is not a valid one.
// Upstream wants minutes in whole tens because it checks every ten minutes
// on the minute; the reminders here are scheduled every five minutes and each
// is checked at the first run after its own time, so any minute works.
func challengeRemindTime(args []string) (remindTime string, ok bool, reply string) {
	if len(args) != 1 || !challengeRemindTimes.MatchString(args[0]) {
		return "", false, ""
	}
	if _, _, _, valid := parseChallengeTime(args[0]); !valid {
		return "", true, "时间格式错误"
	}
	return args[0], true, ""
}

// challengePairEnable is 开启挑战提醒: the pair for the user's role, unless
// the user already has one running.
func (a *App) challengePairEnable(ctx context.Context, event *rayleabot.EventContext, global challengeGlobals, owner Subject, args []string) error {
	if !global.Enabled {
		return event.SendText(challengeRemindOff)
	}
	if len(args) > 1 {
		return event.SendText("格式：" + a.Game.Prefix + "开启挑战提醒 [UID]")
	}
	tasks, err := a.challengePairTasks(owner)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if slices.ContainsFunc(tasks, func(task Reminder) bool { return task.Enabled }) {
		return event.SendText("提醒已开启，请勿重复操作")
	}
	uid := ""
	if len(args) == 1 {
		uid = args[0]
	}
	listed, err := a.accountClient(event).List(ctx, 0)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	choice, _, err := Choose(listed, a.Game.ID, uid)
	if err != nil {
		if PublicError(err).Code == "plugin.game_role_missing" {
			return event.SendText("未绑定UID，请先绑定")
		}
		return event.SendText(friendlyError(err))
	}
	// A pair that expired or paused is replaced.
	for _, task := range tasks {
		if _, err = a.removeDelegatedTask(ctx, event, task.Ref, "challenge"); err != nil {
			return event.SendText(friendlyError(err))
		}
	}
	preference, _, err := a.ChallengePrefs.Get(owner)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	abyss, deadly, remindTime := preference.targets(global)
	if _, err = a.createChallengePair(ctx, a.accountClient(event), eventScheduler(event), choice, abyss, deadly, remindTime); err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText("提醒功能已开启，将在" + remindTime + "对" + challengePairDescription(abyss, deadly) + "\n有效 30 天，到期后请重新开启。")
}

// createChallengePair creates the 开启挑战提醒 task of a role for 30 days.
func (a *App) createChallengePair(ctx context.Context, client AccountsClient, schedule taskScheduler, choice Selection, abyss, deadly int, remindTime string) (Reminder, error) {
	kind, _ := challengeKind("challenge")
	hour, minute, weekday, _ := parseChallengeTime(remindTime)
	threshold := float64(abyss)
	return a.createChallengeReminder(ctx, client, schedule, kind, challengeRequest{Selection: choice, Kind: kind.ID, Metric: "s_layers", Threshold: &threshold, DeadlyThreshold: float64(deadly), Hour: hour, Minute: minute, Weekday: weekday, Days: 30, Confirm: true, Pair: true})
}

// challengePairCheck is 查询挑战状态: both modes read now and listed
// whatever their state.
func (a *App) challengePairCheck(ctx context.Context, event *rayleabot.EventContext, global challengeGlobals, owner Subject, args []string) error {
	if len(args) > 1 {
		return event.SendText("格式：" + a.Game.Prefix + "查询挑战状态 [UID]")
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
		if PublicError(err).Code == "plugin.game_role_missing" {
			return event.SendText("未绑定UID，请先绑定")
		}
		return event.SendText(friendlyError(err))
	}
	preference, _, err := a.ChallengePrefs.Get(owner)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	abyss, deadly, _ := preference.targets(global)
	notice(ctx, event, "正在查询，请稍候...")
	data, failures := map[string]any{}, map[string]any{}
	for _, mode := range challengePairModes {
		result, queryErr := client.Execute(ctx, choice, a.Game.ID+"."+mode.kind, nil)
		if queryErr != nil {
			failures[mode.kind] = friendlyError(queryErr)
			continue
		}
		data[mode.kind] = result.Data
	}
	data["errors"] = failures
	lines := challengePairReport(abyss, deadly, data, true, time.Now())
	if len(lines) == 0 {
		return event.SendText("查询失败，请稍后再试")
	}
	return event.SendText(strings.Join(lines, "\n"))
}
