package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/RayleaBot/plugin-zzz/internal/localdata"
	"math/big"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

type BillingRow struct {
	ID     string `json:"id"`
	Time   string `json:"time"`
	Action string `json:"action"`
	Change string `json:"change"`
	Deduct string `json:"deduct,omitempty"`
	Item   string `json:"item,omitempty"`
}
type BillingArchive struct {
	Revision  uint64       `json:"revision"`
	UID       string       `json:"uid"`
	Region    string       `json:"region"`
	Category  string       `json:"category"`
	Direction string       `json:"direction"`
	SavedMS   int64        `json:"saved_ms"`
	Rows      []BillingRow `json:"rows"`
}
type BillingStore struct {
	mu        sync.Mutex
	Directory string
}

func (s *BillingStore) file(provider string, choice Selection, category, direction string) string {
	raw, _ := json.Marshal([]string{provider, choice.AccountRef, choice.RoleRef, category, direction})
	sum := sha256.Sum256(raw)
	return filepath.Join(s.Directory, hex.EncodeToString(sum[:])+".json")
}
func (s *BillingStore) Read(provider string, choice Selection, category, direction string) (BillingArchive, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := BillingArchive{Rows: []BillingRow{}}
	err := localdata.Read(s.file(provider, choice, category, direction), &out)
	return out, err
}
func (s *BillingStore) Write(provider string, choice Selection, expected uint64, value BillingArchive) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	file := s.file(provider, choice, value.Category, value.Direction)
	old := BillingArchive{}
	if err := localdata.Read(file, &old); err != nil {
		return err
	}
	if old.Revision != expected {
		return gameError("history_changed", "资产档案已变化，请刷新后重新收集。")
	}
	value.Revision = expected + 1
	value.SavedMS = time.Now().UnixMilli()
	if len(value.Rows) > 30000 {
		return gameError("history_limit", "资产档案超过 30000 条上限。")
	}
	raw, err := json.Marshal(value)
	if err != nil || len(raw) > 8*1024*1024 {
		return gameError("history_limit", "资产档案超过 8 MiB，请缩小范围。")
	}
	return localdata.Write(file, value)
}

type BillingJob struct {
	lastUsed time.Time
	Ref      string `json:"ref"`
	State    string `json:"state"`
	Sequence int    `json:"sequence"`
	Pages    int    `json:"pages"`
	Count    int    `json:"count"`
	Selection
	Provider  string `json:"-"`
	Role      Role   `json:"-"`
	Category  string `json:"category"`
	Direction string `json:"direction"`
	revision  uint64
	cursor    string
	page      int
	rows      []BillingRow
	last      *BillingJob
	timer     *time.Timer
}
type BillingJobs struct {
	mu     sync.Mutex
	items  map[string]*BillingJob
	closed bool
}

func (s *BillingJobs) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	for _, j := range s.items {
		j.timer.Stop()
	}
	s.items = nil
}
func billingRows(raw []any) ([]BillingRow, error) {
	out := []BillingRow{}
	for _, value := range raw {
		v := asObject(value)
		row := BillingRow{ID: asText(v["id"]), Time: firstText(v, "datetime", "time"), Action: plainGameText(firstText(v, "action", "reason")), Change: firstText(v, "change_num", "add_num"), Deduct: asText(v["sub_num"]), Item: plainGameText(asText(v["item_name"]))}
		if len(row.ID) > 64 || len(row.Time) > 64 || len([]rune(row.Action)) > 256 || len(row.Change) > 128 || len(row.Deduct) > 128 || len([]rune(row.Item)) > 128 {
			return nil, gameError("billing_invalid", "资产记录字段过长。")
		}
		out = append(out, row)
	}
	return out, nil
}
func mergeBilling(old, next []BillingRow) ([]BillingRow, error) {
	out := []BillingRow{}
	byID := map[string]int{}
	for _, r := range old {
		if r.ID != "" {
			byID[r.ID] = len(out)
			out = append(out, r)
		}
	}
	seen := map[string]bool{}
	for _, r := range next {
		if r.ID != "" {
			if seen[r.ID] {
				return nil, gameError("billing_invalid", "官方分页重复记录编号，未提交档案。")
			}
			seen[r.ID] = true
			if i, ok := byID[r.ID]; ok {
				if out[i] != r {
					return nil, gameError("billing_conflict", "相同官方编号的内容发生变化，原档案保留。")
				}
				continue
			}
		}
		out = append(out, r)
	}
	slices.SortStableFunc(out, func(a, b BillingRow) int {
		if a.Time != b.Time {
			return strings.Compare(b.Time, a.Time)
		}
		if len(a.ID) != len(b.ID) {
			if len(a.ID) > len(b.ID) {
				return -1
			}
			return 1
		}
		return strings.Compare(b.ID, a.ID)
	})
	return out, nil
}
func (a *App) billingManage(ctx context.Context, client AccountsClient, action string, input map[string]any) (map[string]any, error) {
	if a.Game.ID == "zzz" {
		return nil, gameError("operation_denied", "当前参考没有此游戏的客服资产记录。")
	}
	if strings.HasPrefix(action, "billing.sync.") && action != "billing.sync.start" {
		a.BillingJobs.mu.Lock()
		defer a.BillingJobs.mu.Unlock()
		ref := asText(input["ref"])
		j := a.BillingJobs.items[ref]
		if j == nil {
			return nil, gameError("billing_missing", "收集任务已失效，请重新开始。")
		}
		if action == "billing.sync.cancel" {
			j.timer.Stop()
			delete(a.BillingJobs.items, ref)
			return map[string]any{"canceled": true}, nil
		}
		j.lastUsed = time.Now()
		j.timer.Reset(15 * time.Minute)
		client.Provider = j.Provider
		role, err := authorizeCloudRole(ctx, client, j.Selection)
		if err != nil {
			return nil, err
		}
		if role.UID != j.Role.UID || role.Region != j.Role.Region {
			return nil, gameError("role_missing", "任务角色授权已变化。")
		}
		var q struct {
			Sequence int `json:"sequence"`
		}
		if decodeObject(input, &q) != nil {
			return nil, gameError("input_invalid", "收集序号无效。")
		}
		if q.Sequence == j.Sequence-1 && j.last != nil {
			return map[string]any{"job": *j.last}, nil
		}
		if action != "billing.sync.step" || j.State != "running" || q.Sequence != j.Sequence {
			return nil, gameError("billing_sequence", "收集进度已变化。")
		}
		params := map[string]any{"category": j.Category, "direction": j.Direction, "end_id": j.cursor, "page": j.page}
		result, err := client.Execute(ctx, j.Selection, a.Game.ID+".billing", params)
		if err != nil {
			return nil, err
		}
		raw, ok := result.Data["items"].([]any)
		if !ok {
			return nil, gameError("billing_invalid", "官方资产列表无效。")
		}
		rows, err := billingRows(raw)
		if err != nil {
			return nil, err
		}
		candidate := append(slices.Clone(j.rows), rows...)
		if len(candidate) > 30000 {
			return nil, gameError("history_limit", "本次资产记录超过 30000 条。")
		}
		more, ok := result.Data["has_more"].(bool)
		if !ok {
			return nil, gameError("billing_invalid", "官方分页标记缺失。")
		}
		nextCursor := asText(result.Data["next_end_id"])
		nextPage := number(result.Data["next_page"])
		if more && (a.Game.ID == "genshin" && (nextCursor == "" || nextCursor == j.cursor) || a.Game.ID == "starrail" && nextPage <= j.page) {
			return nil, gameError("billing_invalid", "官方分页未向前推进。")
		}
		if !more {
			archive, err := a.Billing.Read(j.Provider, j.Selection, j.Category, j.Direction)
			if err != nil {
				return nil, err
			}
			combined, err := mergeBilling(archive.Rows, candidate)
			if err != nil {
				return nil, err
			}
			archive = BillingArchive{UID: role.UID, Region: role.Region, Category: j.Category, Direction: j.Direction, Rows: combined}
			if err = a.Billing.Write(j.Provider, j.Selection, j.revision, archive); err != nil {
				return nil, err
			}
			j.State = "completed"
			j.Count = len(combined)
		} else {
			j.rows = candidate
			j.Count = len(candidate)
		}
		j.Pages++
		j.Sequence++
		j.cursor = nextCursor
		j.page = nextPage
		copy := *j
		copy.rows = nil
		copy.last = nil
		copy.timer = nil
		j.last = &copy
		return map[string]any{"job": copy}, nil
	}
	choice := Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}
	category := asText(input["category"])
	direction := asText(input["direction"])
	if direction == "" {
		direction = "all"
	}
	if !validBillingCategory(a.Game.ID, category) || !slices.Contains([]string{"all", "produce", "consume"}, direction) {
		return nil, gameError("input_invalid", "请选择资产类别及方向。")
	}
	role, err := authorizeCloudRole(ctx, client, choice)
	if err != nil {
		return nil, err
	}
	archive, err := a.Billing.Read(client.Provider, choice, category, direction)
	if err != nil {
		return nil, err
	}
	if archive.UID != "" && (archive.UID != role.UID || archive.Region != role.Region) {
		return nil, gameError("role_missing", "资产档案与授权角色不一致。")
	}
	if action == "billing.sync.start" {
		a.BillingJobs.mu.Lock()
		defer a.BillingJobs.mu.Unlock()
		if a.BillingJobs.closed || len(a.BillingJobs.items) >= 8 {
			return nil, gameError("billing_limit", "收集任务已满，请完成或取消已有任务。")
		}
		if a.BillingJobs.items == nil {
			a.BillingJobs.items = map[string]*BillingJob{}
		}
		j := &BillingJob{Ref: rand.Text(), State: "running", Selection: choice, Provider: client.Provider, Role: role, Category: category, Direction: direction, revision: archive.Revision, cursor: "0", page: 1}
		j.lastUsed = time.Now()
		j.timer = time.AfterFunc(15*time.Minute, func() { a.BillingJobs.mu.Lock(); defer a.BillingJobs.mu.Unlock(); delete(a.BillingJobs.items, j.Ref) })
		a.BillingJobs.items[j.Ref] = j
		return map[string]any{"job": *j}, nil
	}
	if action == "billing.archive.remove" {
		var q struct {
			Revision uint64 `json:"revision"`
			Confirm  bool   `json:"confirm"`
		}
		if decodeObject(input, &q) != nil || !q.Confirm {
			return nil, gameError("input_invalid", "请确认清空本类别资产档案。")
		}
		archive.Rows = []BillingRow{}
		archive.UID = role.UID
		archive.Region = role.Region
		archive.Category = category
		archive.Direction = direction
		err = a.Billing.Write(client.Provider, choice, q.Revision, archive)
		return map[string]any{"removed": err == nil}, err
	}
	if action != "billing.archive.get" && action != "billing.archive.list" {
		return nil, gameError("operation_denied", "资产档案操作不存在。")
	}
	var q struct {
		Offset int    `json:"offset"`
		Month  string `json:"month"`
		Query  string `json:"query"`
	}
	if decodeObject(input, &q) != nil || q.Offset < 0 || q.Offset > 30000 || len(q.Query) > 128 || len(q.Month) > 7 {
		return nil, gameError("input_invalid", "资产档案筛选无效。")
	}
	rows := []BillingRow{}
	totals := map[string]*big.Rat{}
	coverage := map[string]int{}
	unknown := 0
	for _, r := range archive.Rows {
		if q.Month != "" && !strings.HasPrefix(r.Time, q.Month) || q.Query != "" && !strings.Contains(r.Action+r.Item, q.Query) {
			continue
		}
		rows = append(rows, r)
		if v, ok := new(big.Rat).SetString(r.Change); ok {
			sum := totals[r.Action]
			if sum == nil {
				sum = new(big.Rat)
				totals[r.Action] = sum
			}
			sum.Add(sum, v)
			coverage[r.Action]++
		} else {
			unknown++
		}
	}
	summary := map[string]string{}
	for action, total := range totals {
		summary[action] = total.FloatString(2)
	}
	start := min(q.Offset, len(rows))
	end := min(start+50, len(rows))
	var next *int
	if end < len(rows) {
		next = &end
	}
	return map[string]any{"rows": rows[start:end], "total": len(rows), "next_offset": next, "totals": summary, "coverage": coverage, "unknown_amounts": unknown, "revision": archive.Revision, "saved_ms": archive.SavedMS, "uid": role.UID, "note": "合计为所选记录的官方变动数值，不是支付金额；无官方编号的记录只保留本轮快照。"}, nil
}
