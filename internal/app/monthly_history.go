package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/zzz/internal/localdata"
)

type MonthlySnapshot struct {
	Month   string          `json:"month"`
	SavedMS int64           `json:"saved_ms"`
	Data    json.RawMessage `json:"data"`
}
type MonthlyArchive struct {
	Revision uint64            `json:"revision"`
	Items    []MonthlySnapshot `json:"items"`
}
type MonthlyStore struct {
	mu        sync.Mutex
	Directory string
}

func (s *MonthlyStore) file(provider string, choice Selection) string {
	raw, _ := json.Marshal([]string{provider, choice.AccountRef, choice.RoleRef})
	sum := sha256.Sum256(raw)
	return filepath.Join(s.Directory, hex.EncodeToString(sum[:])+".json")
}
func (s *MonthlyStore) Read(provider string, choice Selection) (MonthlyArchive, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := MonthlyArchive{Items: []MonthlySnapshot{}}
	err := localdata.Read(s.file(provider, choice), &out)
	return out, err
}
func (s *MonthlyStore) Update(provider string, choice Selection, revision uint64, fn func(*MonthlyArchive) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := MonthlyArchive{Items: []MonthlySnapshot{}}
	file := s.file(provider, choice)
	if err := localdata.Read(file, &out); err != nil {
		return err
	}
	if out.Revision != revision {
		return gameError("history_changed", "月报档案已变化，请刷新后重试。")
	}
	if err := fn(&out); err != nil {
		return err
	}
	out.Revision++
	return localdata.Write(file, out)
}
func monthlyKey(data map[string]any) (string, error) {
	raw := asText(data["data_month"])
	month, err := strconv.Atoi(raw)
	if err == nil && month >= 202001 && month <= 210012 && month%100 >= 1 && month%100 <= 12 {
		return fmt.Sprintf("%04d-%02d", month/100, month%100), nil
	}
	return "", gameError("monthly_invalid", "官方月报未提供可辨识的月份，未保存。")
}
func decodeMonthly(raw json.RawMessage) (map[string]any, error) {
	var data map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	err := decoder.Decode(&data)
	return data, err
}
func monthlyAmounts(data map[string]any) map[string]float64 {
	out := map[string]float64{}
	month := asObject(data["month_data"])
	for _, raw := range asList(month["list"]) {
		item := asObject(raw)
		label := map[string]string{"PolychromesData": "菲林", "MatserTapeData": "母带", "BooponsData": "邦布券"}[asText(item["data_type"])]
		if v, ok := challengeNumber(item["count"]); ok && label != "" {
			out[label] = v
		}
	}
	return out
}
func monthlySummaries(archive MonthlyArchive) map[string]any {
	items := []map[string]any{}
	totals := map[string]float64{}
	coverage := map[string]int{}
	slices.SortFunc(archive.Items, func(a, b MonthlySnapshot) int { return strings.Compare(b.Month, a.Month) })
	for _, v := range archive.Items {
		data, err := decodeMonthly(v.Data)
		if err != nil {
			continue
		}
		amounts := monthlyAmounts(data)
		for k, n := range amounts {
			totals[k] += n
			coverage[k]++
		}
		items = append(items, map[string]any{"month": v.Month, "saved_ms": v.SavedMS, "amounts": amounts})
	}
	return map[string]any{"items": items, "totals": totals, "coverage": coverage, "revision": archive.Revision}
}
func (a *App) monthlyAction(ctx context.Context, client AccountsClient, action string, input map[string]any) (map[string]any, error) {
	choice := Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}
	var r struct {
		Roles []Role `json:"roles"`
	}
	if err := client.call(ctx, "roles", map[string]any{"account_ref": choice.AccountRef, "game": a.Game.ID}, &r); err != nil {
		return nil, err
	}
	i := slices.IndexFunc(r.Roles, func(role Role) bool { return role.Ref == choice.RoleRef && role.Game == a.Game.ID })
	if i < 0 {
		return nil, gameError("role_missing", "所选角色授权不可用。")
	}
	role := r.Roles[i]
	archive, err := a.Monthly.Read(client.Provider, choice)
	if err != nil {
		return nil, err
	}
	if action == "monthly.list" {
		return monthlySummaries(archive), nil
	}
	if action == "monthly.fetch" {
		args := map[string]any{}
		if month, exists := input["month"]; exists && asText(month) != "" {
			args["month"] = month
		}
		result, err := client.Execute(ctx, choice, a.Game.ID+".monthly", args)
		if err != nil {
			return nil, err
		}
		key, err := a.Monthly.Save(client.Provider, choice, archive.Revision, result.Data, time.Now())
		if err != nil {
			return nil, err
		}
		available := []string{}
		for _, value := range asList(result.Data["optional_month"]) {
			month, err := strconv.Atoi(asText(value))
			if err == nil && (month >= 202001 && month <= 210012 && month%100 >= 1 && month%100 <= 12) {
				available = append(available, strconv.Itoa(month))
			}
		}
		return map[string]any{"month": key, "available_months": available, "saved": true}, nil
	}
	month := asText(input["month"])
	i = slices.IndexFunc(archive.Items, func(v MonthlySnapshot) bool { return v.Month == month })
	if i < 0 {
		return nil, gameError("history_missing", "没有保存此月份。")
	}
	if action == "monthly.remove" {
		var q struct {
			Revision uint64 `json:"revision"`
		}
		if decodeObject(input, &q) != nil {
			return nil, gameError("input_invalid", "月报档案版本无效。")
		}
		err = a.Monthly.Update(client.Provider, choice, q.Revision, func(v *MonthlyArchive) error {
			v.Items = slices.DeleteFunc(v.Items, func(item MonthlySnapshot) bool { return item.Month == month })
			return nil
		})
		return map[string]any{"removed": err == nil}, err
	}
	if action != "monthly.get" {
		return nil, gameError("operation_denied", "月报操作不存在。")
	}
	data, err := decodeMonthly(archive.Items[i].Data)
	if err != nil {
		return nil, err
	}
	result := QueryResult{Operation: a.Game.ID + ".monthly", Role: role, Data: data, FetchedAtMS: archive.Items[i].SavedMS}
	operation, _ := a.operation(result.Operation)
	view := BusinessView(a.Game, operation, result, a.Catalog)
	view.Note = "已保存月报 · " + month + " · 仅代表保存时的数据"
	return map[string]any{"view": view}, nil
}

func (a *App) monthlyCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	uid := ""
	if len(args) > 1 {
		return event.SendText("请提供至多一个 UID。")
	}
	if len(args) == 1 {
		uid = args[0]
	}
	client := a.accountClient(event)
	listed, err := client.List(ctx, 0)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	choice, role, err := Choose(listed, a.Game.ID, uid)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	input := map[string]any{"account_ref": choice.AccountRef, "role_ref": choice.RoleRef}
	// Upstream saves the months the official report still offers before it
	// counts.
	if err := a.refreshMonthly(ctx, client, choice); err != nil {
		return event.SendText(friendlyError(err))
	}
	result, err := a.monthlyAction(ctx, client, "monthly.list", input)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	view := View{Title: a.Game.Name + "已保存月报累计", Subtitle: role.UID, Rows: []Row{}, Note: "统计前会保存官方仍提供的月份；只累计本地保存的月份，更早的缺失月份和未提供项目不补算。"}
	totals := result["totals"].(map[string]float64)
	coverage := result["coverage"].(map[string]int)
	keys := []string{}
	for key := range totals {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		view.Rows = append(view.Rows, Row{Label: key, Value: fmt.Sprintf("%.0f · 覆盖 %d 个月", totals[key], coverage[key])})
	}
	if a.monthlyStats != nil {
		archive, err := a.Monthly.Read(client.Provider, choice)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		months := []SavedMonth{}
		for _, item := range archive.Items {
			if data, err := decodeMonthly(item.Data); err == nil {
				months = append(months, SavedMonth{Month: item.Month, Data: data})
			}
		}
		slices.SortFunc(months, func(a, b SavedMonth) int { return strings.Compare(a.Month, b.Month) })
		if drawn, ok := a.monthlyStats(a.imageContext(ctx), MonthlyStats{Role: role, Word: event.Event.Command(), Months: months}); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
}

// refreshMonthly saves this month's report and each month the official report
// still offers that was not saved after it ended; a month that fails to load
// is left as saved.
func (a *App) refreshMonthly(ctx context.Context, client AccountsClient, choice Selection) error {
	operation := a.Game.ID + ".monthly"
	current, err := client.Execute(ctx, choice, operation, map[string]any{})
	if err != nil {
		return err
	}
	now := a.now()
	if err := a.Monthly.Keep(client.Provider, choice, current.Data, now); err != nil {
		return err
	}
	archive, err := a.Monthly.Read(client.Provider, choice)
	if err != nil {
		return err
	}
	for _, value := range asList(current.Data["optional_month"]) {
		month := asText(value)
		if month == asText(current.Data["data_month"]) {
			continue
		}
		key, err := monthlyKey(map[string]any{"data_month": month})
		if err != nil || monthlyFinal(archive, key) {
			continue
		}
		result, err := client.Execute(ctx, choice, operation, map[string]any{"month": month})
		if err != nil {
			continue
		}
		_ = a.Monthly.Keep(client.Provider, choice, result.Data, now)
	}
	return nil
}

// monthlyFinal reports whether the archive holds the month as saved after it
// ended, which no later read changes.
func monthlyFinal(archive MonthlyArchive, key string) bool {
	start, err := time.ParseInLocation("2006-01", key, time.FixedZone("UTC+8", 28800))
	if err != nil {
		return false
	}
	end := start.AddDate(0, 1, 0).UnixMilli()
	return slices.ContainsFunc(archive.Items, func(item MonthlySnapshot) bool { return item.Month == key && item.SavedMS >= end })
}

// Keep saves a month over whatever archive is current, for reads the user did
// not start from the archive page.
func (s *MonthlyStore) Keep(provider string, choice Selection, data map[string]any, now time.Time) error {
	archive, err := s.Read(provider, choice)
	if err != nil {
		return err
	}
	_, err = s.Save(provider, choice, archive.Revision, data, now)
	return err
}

func (s *MonthlyStore) Save(provider string, choice Selection, revision uint64, data map[string]any, now time.Time) (string, error) {
	if asObject(data["month_data"]) == nil {
		return "", gameError("monthly_invalid", "官方月报没有月度内容，未保存。")
	}
	key, err := monthlyKey(data)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(data)
	if err != nil || len(raw) > 64*1024 {
		return "", gameError("history_limit", "单月月报超出保存上限。")
	}
	snapshot := MonthlySnapshot{key, now.UnixMilli(), raw}
	err = s.Update(provider, choice, revision, func(v *MonthlyArchive) error {
		i := slices.IndexFunc(v.Items, func(item MonthlySnapshot) bool { return item.Month == key })
		if i >= 0 {
			v.Items[i] = snapshot
		} else {
			if len(v.Items) >= 120 {
				return gameError("history_limit", "已保存 120 个月，请先移除旧记录。")
			}
			v.Items = append(v.Items, snapshot)
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	return key, nil
}

func (s *ReminderStore) tickMonthly(task *Reminder, now int64, query func(Reminder) (QueryResult, error), send func(Reminder, string) error, game Game) error {
	task.NextCheckMS = nextChallengeCheck(now, task.Hour, task.Minute, 0)
	task.LastCheckedMS = now
	result, err := query(*task)
	if err != nil {
		task.LastCode = PublicError(err).Code
		switch task.LastCode {
		case "plugin.account_delegation_denied", "plugin.account_caller_denied", "plugin.account_not_found", "plugin.account_role_denied", "plugin.upstream_auth_invalid":
			task.Enabled = false
		case "plugin.upstream_device_required", "plugin.upstream_challenge_required":
			task.NextCheckMS = max(task.NextCheckMS, now+int64(24*time.Hour/time.Millisecond))
		}
		return s.save(*task)
	}
	task.LastCode = "collected"
	if task.Notify {
		task.LastAttemptMS = now
	}
	if err = s.save(*task); err != nil {
		return err
	}
	if task.Notify {
		if err = send(*task, game.Name+"默认月份月报已保存 · "+task.Role.UID+" · "+asText(result.Data["data_month"])); err != nil {
			task.LastCode = "collected.notification_failed"
			return s.save(*task)
		}
	}
	return nil
}

var monthlyWords = regexp.MustCompile(`^(?:([0-9]{4})年)?(?:([0-9]{1,2}|上)月)?$`)

// monthlyWord reads ZZZ-Plugin's month words as its getDateString does: a
// year before 2023 or none is this year, and no month, 0月 or a month still
// to come reads the default month, 0. valid is false for a month above 12.
// 上月 is the month before this one; upstream means the same but reads 上 as
// a number, which is never one, and so always falls back to the default.
func monthlyWord(word string, now time.Time) (month int, valid bool) {
	match := monthlyWords.FindStringSubmatch(word)
	if match == nil || match[2] == "" {
		return 0, match != nil
	}
	now = now.In(time.FixedZone("UTC+8", 8*3600))
	year, _ := strconv.Atoi(match[1])
	if year < 2023 {
		year = now.Year()
	}
	number, _ := strconv.Atoi(match[2])
	if match[2] == "上" {
		number = int(now.Month()) - 1
		if number == 0 {
			number, year = 12, year-1
		}
	}
	if number == 0 {
		return 0, true
	}
	if number > 12 {
		return 0, false
	}
	if time.Date(year, time.Month(number), 1, 0, 0, 0, 0, now.Location()).After(now) {
		return 0, true
	}
	return year*100 + number, true
}
