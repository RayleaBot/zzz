package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/zzz/internal/localdata"
)

type historyRecord struct {
	Kind     string  `json:"kind"`
	Ref      string  `json:"ref"`
	Owner    Subject `json:"owner"`
	Label    string  `json:"label"`
	Count    int     `json:"count"`
	Revision string  `json:"revision"`
}

type historyQuery struct {
	action, kind, ref, revision string
	bot                         rayleabot.BotAccount
	rows                        []historyRecord
	data                        any
	found, changed              bool
	oldRef, newRef              string
}

func historyRevision(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// Every collection is read, checked and written under its existing store lock.
// Ref is an opaque lookup key, never a path supplied by the page.
func historyItems[T any](q *historyQuery, kind string, items []T, describe func(T) historyRecord, assign func(*T, Subject)) ([]T, error) {
	for i, item := range items {
		row := describe(item)
		row.Kind, row.Revision = kind, historyRevision(item)
		if q.action == "history.list" {
			if row.Count > 0 {
				q.rows = append(q.rows, row)
			}
			continue
		}
		if row.Ref != q.ref {
			continue
		}
		q.found = true
		if q.action == "history.get" {
			// Delegations are for the scheduler, not for historical inspection.
			var data map[string]any
			if err := decodeObject(item, &data); err != nil {
				return nil, err
			}
			delete(data, "delegation_ref")
			delete(data, "deadly_delegation_ref")
			q.data = data
			return items, nil
		}
		if q.revision != row.Revision {
			return nil, gameError("history_changed", "记录已变化，请刷新后重试。")
		}
		q.oldRef = row.Ref
		if q.action == "history.remove" {
			q.changed = true
			return slices.Delete(items, i, i+1), nil
		}
		if row.Owner.SourceProtocol != "onebot11" || row.Owner.BotID != "" || q.bot.SourceProtocol != "onebot11" || assign == nil {
			return nil, gameError("account_mismatch", "仅可为缺少账号的 OneBot11 记录补齐归属。")
		}
		owner := row.Owner.identity()
		owner.BotID = q.bot.BotID
		assign(&items[i], owner)
		q.newRef = describe(items[i]).Ref
		for j, other := range items {
			if j != i && describe(other).Ref == q.newRef {
				return nil, gameError("history_changed", "目标账号已有此记录，请先处理冲突。")
			}
		}
		q.changed = true
		return items, nil
	}
	return items, nil
}

func (a *App) manageHistory(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	q := &historyQuery{action: action, kind: asText(input["kind"]), ref: asText(input["ref"]), revision: asText(input["revision"]), rows: []historyRecord{}}
	if action == "history.assign" {
		bot, err := managementBot(event)
		if err != nil {
			return nil, err
		}
		q.bot = bot
	}
	if action != "history.list" && action != "history.get" && action != "history.remove" && action != "history.assign" {
		return nil, gameError("operation_denied", "历史数据操作不存在。")
	}
	if (action == "history.remove" || action == "history.assign") && input["confirm"] != true {
		return nil, gameError("input_invalid", "请确认处理此记录。")
	}
	visits := []struct {
		kind  string
		visit func() error
	}{
		{"groups", func() error { return a.Groups.history(q) }},
		{"reminders", func() error {
			return a.Reminders.edit("", func(items *[]Reminder, _ int) (err error) {
				*items, err = historyItems(q, "reminders", *items, func(t Reminder) historyRecord {
					return historyRecord{Ref: t.Ref, Owner: t.Owner, Label: t.Kind + " · " + t.Role.UID + " · " + t.Owner.ActorID, Count: 1}
				}, func(t *Reminder, owner Subject) { t.Owner = owner })
				return err
			})
		}},
		{"daily", func() error {
			return a.DailySyncs.edit("", func(items *[]DailySync, _ int) (err error) {
				*items, err = historyItems(q, "daily", *items, func(t DailySync) historyRecord {
					return historyRecord{Ref: t.Ref, Owner: t.Owner, Label: t.Role.UID + " · " + t.Owner.ActorID, Count: 1}
				}, func(t *DailySync, owner Subject) { t.Owner = owner })
				return err
			})
		}},
		{"subscriptions", func() error { return a.subscriptionHistory(ctx, event, q) }},
		{"interactions", func() error { return a.Interactions.history(q) }},
		{"preferences", func() error { return a.ChallengePrefs.history(q) }},
		{"panels", func() error { return a.PanelHistory.history(q) }},
		{"monthly", func() error { return a.Monthly.history(q) }},
		{"background", func() error { return a.BackgroundSyncs.history(q) }},
	}
	for _, v := range visits {
		if action != "history.list" && q.kind != v.kind {
			continue
		}
		if err := v.visit(); err != nil {
			return nil, err
		}
	}
	if action == "history.list" {
		total := 0
		for _, row := range q.rows {
			total += row.Count
		}
		return map[string]any{"items": q.rows, "total": total}, nil
	}
	if !q.found {
		return nil, gameError("history_missing", "历史记录不存在。")
	}
	if action == "history.get" {
		return map[string]any{"record": q.data}, nil
	}
	if action == "history.remove" && (q.kind == "reminders" || q.kind == "daily") {
		if q.kind == "daily" {
			a.BackgroundSyncs.cancel(func(item BackgroundSync) bool { return item.Task == q.ref }, "sync_canceled")
		}
		if _, err := event.Actions().SchedulerDelete(ctx, q.ref); err != nil {
			return nil, err
		}
	}
	return map[string]any{"updated": true}, nil
}

func (s *GroupStore) history(q *historyQuery) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	paths, err := addressFiles(s.Directory)
	if err != nil {
		return err
	}
	for _, path := range paths {
		var group GroupData
		if err := localdata.Read(path, &group); err != nil {
			return err
		}
		items, err := historyItems(q, "groups", []GroupData{group}, func(g GroupData) historyRecord {
			return historyRecord{Ref: filepath.Base(path), Owner: Subject{SourceProtocol: g.Scope.Protocol, BotID: g.Scope.BotID, SourceAdapter: g.Scope.Adapter}, Label: "群 " + g.Scope.GroupID, Count: 1}
		}, func(g *GroupData, owner Subject) {
			g.Scope.Protocol, g.Scope.BotID, g.Scope.Adapter = owner.SourceProtocol, owner.BotID, ""
			g.Revision++
		})
		if err != nil {
			return err
		}
		if !q.changed {
			continue
		}
		if len(items) > 0 {
			target := s.file(items[0].Scope)
			if _, err := os.Stat(target); err == nil {
				return gameError("history_changed", "目标账号已有此记录，请先处理冲突。")
			} else if !os.IsNotExist(err) {
				return err
			}
			if err := localdata.Write(target, items[0]); err != nil {
				return err
			}
			delete(s.configs, items[0].Scope.identity())
		}
		delete(s.configs, group.Scope.identity())
		return os.Remove(path)
	}
	return nil
}

func (a *App) subscriptionHistory(ctx context.Context, event *rayleabot.EventContext, q *historyQuery) error {
	s := a.Subscriptions
	s.pushMu.Lock()
	defer s.pushMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return err
	}
	items, err = historyItems(q, "subscriptions", items, func(t ContentSubscription) historyRecord {
		return historyRecord{Ref: t.Ref, Owner: t.Owner, Label: "群 " + t.GroupID, Count: 1}
	}, func(t *ContentSubscription, owner Subject) {
		t.Owner = owner
		t.Ref = contentRef(a.Game.ID, owner, t.GroupID)
	})
	if err != nil || !q.changed {
		return err
	}
	if q.action == "history.assign" {
		if _, err := event.Actions().SchedulerCreate(ctx, a.contentJob(q.newRef)); err != nil {
			return err
		}
	}
	if err := localdata.Write(s.Path, items); err != nil {
		return err
	}
	_, err = event.Actions().SchedulerDelete(ctx, q.oldRef)
	return err
}

func (s *InteractionStore) history(q *historyQuery) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return err
	}
	items, err = historyItems(q, "interactions", items, func(t InteractionProfile) historyRecord {
		return historyRecord{Ref: t.Ref, Owner: t.Owner, Label: t.Owner.ActorID, Count: 1}
	}, func(t *InteractionProfile, owner Subject) { t.Owner = owner; t.Ref = interactionRef(owner) })
	if err != nil || !q.changed {
		return err
	}
	return localdata.Write(s.Path, items)
}

func (s *ChallengePreferences) history(q *historyQuery) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return err
	}
	items, err = historyItems(q, "preferences", items, func(t ChallengePreference) historyRecord {
		return historyRecord{Ref: historyRevision(t.Owner), Owner: t.Owner, Label: t.Owner.ActorID, Count: 1}
	}, func(t *ChallengePreference, owner Subject) { t.Owner = owner })
	if err != nil || !q.changed {
		return err
	}
	return localdata.Write(s.Path, items)
}

func (s *PanelHistoryStore) history(q *historyQuery) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	paths, err := addressFiles(s.Directory)
	if err != nil {
		return err
	}
	for _, path := range paths {
		items := []PanelSnapshot{}
		if err := localdata.Read(path, &items); err != nil {
			return err
		}
		items, err = historyItems(q, "panels", items, func(t PanelSnapshot) historyRecord {
			return historyRecord{Ref: filepath.Base(path) + ":" + t.Ref, Owner: t.Owner, Label: t.Panel.Name, Count: 1}
		}, func(t *PanelSnapshot, owner Subject) { t.Owner = owner })
		if err != nil {
			return err
		}
		if q.changed {
			return localdata.Write(path, items)
		}
	}
	return nil
}

func (s *MonthlyStore) history(q *historyQuery) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	paths, err := addressFiles(s.Directory)
	if err != nil {
		return err
	}
	for _, path := range paths {
		var item MonthlyArchive
		if err := localdata.Read(path, &item); err != nil {
			return err
		}
		items, err := historyItems(q, "monthly", []MonthlyArchive{item}, func(t MonthlyArchive) historyRecord {
			return historyRecord{Ref: filepath.Base(path), Owner: t.Owner, Label: "月报档案", Count: len(t.Items)}
		}, func(t *MonthlyArchive, owner Subject) { t.Owner = owner; t.Revision++ })
		if err != nil {
			return err
		}
		if q.changed {
			if len(items) == 0 {
				return os.Remove(path)
			}
			return localdata.Write(path, items[0])
		}
	}
	return nil
}

func (s *backgroundSyncs) history(q *historyQuery) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := historyItems(q, "background", s.items, func(t BackgroundSync) historyRecord {
		return historyRecord{Ref: t.Ref, Owner: t.Owner, Label: t.Role.UID, Count: 1}
	}, nil)
	if err == nil {
		s.items = items
	}
	return err
}
