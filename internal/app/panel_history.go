package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/RayleaBot/zzz/internal/localdata"
)

type PanelSnapshot struct {
	Ref       string         `json:"ref"`
	SavedAtMS int64          `json:"saved_at_ms"`
	Version   string         `json:"version"`
	Panel     CharacterPanel `json:"panel"`
}
type PanelSnapshotSummary struct {
	Ref       string `json:"ref"`
	SavedAtMS int64  `json:"saved_at_ms"`
	Version   string `json:"version"`
	Name      string `json:"name"`
	Level     int    `json:"level"`
	Weapon    string `json:"weapon"`
}
type PanelHistoryStore struct {
	mu        sync.Mutex
	Directory string
}

func (s *PanelHistoryStore) file(provider string, choice Selection, id string) string {
	raw, _ := json.Marshal([]string{provider, choice.AccountRef, choice.RoleRef, id})
	sum := sha256.Sum256(raw)
	return filepath.Join(s.Directory, hex.EncodeToString(sum[:])+".json")
}
func (s *PanelHistoryStore) Read(provider string, choice Selection, id string) ([]PanelSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := []PanelSnapshot{}
	err := localdata.Read(s.file(provider, choice, id), &items)
	return items, err
}
func (s *PanelHistoryStore) Save(provider string, choice Selection, version string, panel CharacterPanel) (PanelSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := PanelSnapshot{Ref: rand.Text(), SavedAtMS: time.Now().UnixMilli(), Version: version, Panel: panel}
	file := s.file(provider, choice, panel.ID)
	items := []PanelSnapshot{}
	if err := localdata.Read(file, &items); err != nil {
		return item, err
	}
	if len(items) >= 20 {
		return item, gameError("history_limit", "此角色已保存 20 份历史，请先移除不再需要的记录。")
	}
	raw, err := json.Marshal(item)
	if err != nil || len(raw) > 256*1024 {
		return item, gameError("history_limit", "此面板超出单份历史大小限制。")
	}
	return item, localdata.Write(file, append(items, item))
}
func (s *PanelHistoryStore) Remove(provider string, choice Selection, id, ref string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	file := s.file(provider, choice, id)
	items := []PanelSnapshot{}
	if err := localdata.Read(file, &items); err != nil {
		return err
	}
	i := slices.IndexFunc(items, func(p PanelSnapshot) bool { return p.Ref == ref })
	if i < 0 {
		return gameError("history_missing", "面板历史不存在。")
	}
	return localdata.Write(file, slices.Delete(items, i, i+1))
}
func (a *App) panelHistory(ctx context.Context, client AccountsClient, action string, input map[string]any) (map[string]any, error) {
	choice := Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}
	id := asText(input["character_id"])
	parsed, err := strconv.Atoi(id)
	if err != nil || parsed < 1 || parsed > 1000000000 {
		return nil, gameError("input_invalid", "请选择有效角色 ID。")
	}
	var result struct {
		Roles []Role `json:"roles"`
	}
	if err = client.call(ctx, "roles", map[string]any{"account_ref": choice.AccountRef, "game": a.Game.ID}, &result); err != nil {
		return nil, err
	}
	index := slices.IndexFunc(result.Roles, func(r Role) bool { return r.Ref == choice.RoleRef && r.Game == a.Game.ID })
	if index < 0 {
		return nil, gameError("role_missing", "角色不在当前授权范围内。")
	}
	role := result.Roles[index]
	if action == "panel.history.save" {
		panel, err := a.queryCharacterPanel(ctx, client, choice, id)
		if err != nil {
			return nil, err
		}
		snapshot, err := a.PanelHistory.Save(client.Provider, choice, a.Catalog.Version, panel)
		return map[string]any{"snapshot": snapshot}, err
	}
	if action == "panel.history.remove" {
		err := a.PanelHistory.Remove(client.Provider, choice, id, asText(input["ref"]))
		return map[string]any{"removed": err == nil}, err
	}
	items, err := a.PanelHistory.Read(client.Provider, choice, id)
	if err != nil {
		return nil, err
	}
	if action == "panel.history.list" {
		list := []PanelSnapshotSummary{}
		for i := len(items) - 1; i >= 0; i-- {
			p := items[i]
			weapon := "未提供"
			if p.Panel.WeaponKnown {
				weapon = "未装备"
			}
			if p.Panel.Weapon != nil {
				weapon = p.Panel.Weapon.Name
			}
			list = append(list, PanelSnapshotSummary{Ref: p.Ref, SavedAtMS: p.SavedAtMS, Version: p.Version, Name: p.Panel.Name, Level: p.Panel.Level, Weapon: weapon})
		}
		return map[string]any{"items": list}, nil
	}
	find := func(ref string) (PanelSnapshot, error) {
		for _, p := range items {
			if p.Ref == ref {
				return p, nil
			}
		}
		return PanelSnapshot{}, gameError("history_missing", "面板历史不存在。")
	}
	if action == "panel.history.compare" {
		before, err := find(asText(input["before_ref"]))
		if err != nil {
			return nil, err
		}
		after, err := find(asText(input["after_ref"]))
		if err != nil {
			return nil, err
		}
		return map[string]any{"view": comparePanels(before, after)}, nil
	}
	snapshot, err := find(asText(input["ref"]))
	if err != nil {
		return nil, err
	}
	switch action {
	case "panel.history.get":
		view := PanelView(a.Game, []CharacterPanel{snapshot.Panel}, role.UID)
		view.Note = "保存于 " + time.UnixMilli(snapshot.SavedAtMS).UTC().Format(time.RFC3339) + "；资料版本 " + snapshot.Version + "。这是历史快照。"
		return map[string]any{"snapshot": snapshot, "view": view}, nil
	case "panel.history.export":
		return map[string]any{"export": map[string]any{"format": "raylea-panel-history", "version": 1, "game": a.Game.ID, "uid": role.UID, "region": role.Region, "saved_at_ms": snapshot.SavedAtMS, "catalog_version": snapshot.Version, "panel": snapshot.Panel}}, nil
	}
	return nil, gameError("operation_denied", "历史操作不存在。")
}
func comparePanels(before, after PanelSnapshot) View {
	v := View{Title: after.Panel.Name + " · 历史面板对比", Subtitle: time.UnixMilli(before.SavedAtMS).UTC().Format(time.RFC3339) + " → " + time.UnixMilli(after.SavedAtMS).UTC().Format(time.RFC3339), Rows: []Row{{Label: "等级", Value: fmt.Sprintf("%d → %d", before.Panel.Level, after.Panel.Level)}}, Note: "对比保存时的面板；缺失属性不补零，百分属性差值为百分点，不表示实际伤害变化。"}
	if before.Panel.RankKnown && after.Panel.RankKnown {
		v.Rows = append(v.Rows, Row{Label: "解锁层数", Value: fmt.Sprintf("%d → %d", before.Panel.Rank, after.Panel.Rank)})
	}
	a, b := map[string]PanelStat{}, map[string]PanelStat{}
	keys := []string{}
	for _, s := range before.Panel.Stats {
		a[s.ID] = s
		keys = append(keys, s.ID)
	}
	for _, s := range after.Panel.Stats {
		b[s.ID] = s
		if _, ok := a[s.ID]; !ok {
			keys = append(keys, s.ID)
		}
	}
	rows := []Row{}
	for _, id := range keys {
		x, y := a[id], b[id]
		label := y.Name
		if label == "" {
			label = x.Name
		}
		left, right := x.Value, y.Value
		if left == "" {
			left = "未提供"
		}
		if right == "" {
			right = "未提供"
		}
		value := left + " → " + right
		xn, xok := decimalStat(x.Value)
		yn, yok := decimalStat(y.Value)
		xp, yp := strings.HasSuffix(x.Value, "%"), strings.HasSuffix(y.Value, "%")
		if xok && yok && xp == yp {
			unit := ""
			if xp {
				unit = " 个百分点"
			}
			value += fmt.Sprintf("（%+.2f%s）", yn-xn, unit)
		}
		rows = append(rows, Row{Label: label, Value: value})
	}
	v.Sections = append(v.Sections, Section{Title: "属性变化", Rows: rows})
	for _, named := range []struct {
		title string
		item  PanelSnapshot
	}{{"之前配装", before}, {"之后配装", after}} {
		p := named.item.Panel
		rows := []Row{}
		if p.Weapon != nil {
			rows = append(rows, Row{Label: "音擎", Value: fmt.Sprintf("%s · 等级 %d · 星级 %d", p.Weapon.Name, p.Weapon.Level, p.Weapon.Refinement)})
		}
		for _, e := range p.Equipment {
			rows = append(rows, Row{Label: fmt.Sprintf("位置 %d", e.Slot), Value: fmt.Sprintf("%s · %s · 等级 %d", e.Name, e.SetName, e.Level)})
		}
		for _, skill := range p.Skills {
			if skill.Level > 0 {
				rows = append(rows, Row{Label: skill.Name, Value: strconv.Itoa(skill.Level)})
			}
		}
		v.Sections = append(v.Sections, Section{Title: named.title + " · " + named.item.Version, Rows: rows})
	}
	return v
}
