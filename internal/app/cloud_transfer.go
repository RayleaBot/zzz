package app

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"slices"
	"sync"
	"time"
)

type cloudTransfer struct {
	Selection
	Provider string
	Role     Role
	Revision uint64
	Expected int
	Export   bool
	Items    []json.RawMessage
	IDs      []string
	Bytes    int
	Timer    *time.Timer
}
type CloudTransfers struct {
	mu     sync.Mutex
	items  map[string]*cloudTransfer
	closed bool
}

func (s *CloudTransfers) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	for _, v := range s.items {
		v.Timer.Stop()
	}
	s.items = nil
}
func (s *CloudTransfers) start(t cloudTransfer) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || len(s.items) >= 8 {
		return "", gameError("transfer_limit", "交换传输已满，请完成或取消已有传输。")
	}
	if s.items == nil {
		s.items = map[string]*cloudTransfer{}
	}
	ref := rand.Text()
	t.Timer = time.AfterFunc(15*time.Minute, func() { s.mu.Lock(); defer s.mu.Unlock(); delete(s.items, ref) })
	s.items[ref] = &t
	return ref, nil
}
func (s *CloudTransfers) remove(ref string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v := s.items[ref]; v != nil {
		v.Timer.Stop()
		delete(s.items, ref)
	}
}
func authorizeCloudRole(ctx context.Context, client AccountsClient, choice Selection) (Role, error) {
	var response struct {
		Roles []Role `json:"roles"`
	}
	if err := client.call(ctx, "roles", map[string]any{"account_ref": choice.AccountRef, "game": client.Game}, &response); err != nil {
		return Role{}, err
	}
	for _, role := range response.Roles {
		if role.Ref == choice.RoleRef && role.Game == client.Game {
			return role, nil
		}
	}
	return Role{}, gameError("role_missing", "所选角色不在当前授权范围内。")
}
func (a *App) cloudTransferAction(ctx context.Context, client AccountsClient, action string, input map[string]any) (map[string]any, error) {
	if action == "cloud.archive.import.start" || action == "cloud.archive.export.start" {
		choice := Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}
		role, err := authorizeCloudRole(ctx, client, choice)
		if err != nil {
			return nil, err
		}
		archive, err := a.CloudArchive.Read(client.Provider, choice)
		if err != nil {
			return nil, err
		}
		t := cloudTransfer{Selection: choice, Provider: client.Provider, Role: role, Revision: archive.Revision, Export: action == "cloud.archive.export.start"}
		if !t.Export {
			var q struct {
				Count int    `json:"count"`
				UID   string `json:"uid"`
			}
			if decodeObject(input, &q) != nil || q.Count < 1 || q.Count > 250 || q.UID != role.UID {
				return nil, gameError("input_invalid", "导入文件 UID 或角色数量无效。")
			}
			t.Expected = q.Count
		}
		if t.Export {
			if archive.UID != role.UID || archive.Region != role.Region {
				return nil, gameError("history_missing", "没有此角色的交换档案。")
			}
			for id := range archive.Avatars {
				t.IDs = append(t.IDs, id)
			}
			slices.Sort(t.IDs)
			for _, id := range t.IDs {
				t.Items = append(t.Items, archive.Avatars[id])
			}
		}
		ref, err := a.CloudTransfers.start(t)
		return map[string]any{"ref": ref, "uid": role.UID, "region": role.Region, "total": len(t.Items), "revision": archive.Revision}, err
	}
	ref := asText(input["ref"])
	if action == "cloud.archive.import.cancel" || action == "cloud.archive.export.close" {
		a.CloudTransfers.remove(ref)
		return map[string]any{"closed": true}, nil
	}
	a.CloudTransfers.mu.Lock()
	defer a.CloudTransfers.mu.Unlock()
	t := a.CloudTransfers.items[ref]
	if t == nil {
		return nil, gameError("transfer_missing", "交换传输已过期，请重新开始。")
	}
	client.Provider = t.Provider
	role, err := authorizeCloudRole(ctx, client, t.Selection)
	if err != nil {
		return nil, err
	}
	if role.UID != t.Role.UID || role.Region != t.Role.Region {
		return nil, gameError("role_missing", "传输角色授权已变化。")
	}
	var q struct {
		Offset int `json:"offset"`
	}
	if decodeObject(input, &q) != nil || q.Offset < 0 {
		return nil, gameError("input_invalid", "交换传输序号无效。")
	}
	switch action {
	case "cloud.archive.import.append":
		if t.Export || q.Offset > len(t.Items) || q.Offset >= t.Expected {
			return nil, gameError("input_invalid", "交换传输顺序无效。")
		}
		id, raw, err := cleanCloudAvatar(a.Game.ID, input["avatar"])
		if err != nil {
			return nil, err
		}
		if q.Offset < len(t.Items) {
			if string(t.Items[q.Offset]) != string(raw) {
				return nil, gameError("transfer_conflict", "重复上传的角色内容不一致。")
			}
			return map[string]any{"next_offset": len(t.Items)}, nil
		}
		if slices.Contains(t.IDs, id) {
			return nil, gameError("transfer_conflict", "同次传输不能重复角色编号。")
		}
		if t.Bytes+len(raw) > 4*1024*1024 {
			return nil, gameError("history_limit", "交换文件超过 4 MiB。")
		}
		t.IDs = append(t.IDs, id)
		t.Items = append(t.Items, raw)
		t.Bytes += len(raw)
		return map[string]any{"next_offset": len(t.Items)}, nil
	case "cloud.archive.import.finish":
		if t.Export || len(t.Items) != t.Expected || t.Expected == 0 {
			return nil, gameError("input_invalid", "没有可保存的传输内容。")
		}
		avatars := map[string]json.RawMessage{}
		for i, id := range t.IDs {
			avatars[id] = t.Items[i]
		}
		if err = a.CloudArchive.Update(t.Provider, t.Selection, t.Revision, role, avatars, false); err != nil {
			return nil, err
		}
		t.Timer.Stop()
		delete(a.CloudTransfers.items, ref)
		return map[string]any{"imported": len(avatars)}, nil
	case "cloud.archive.export.read":
		if !t.Export || q.Offset >= len(t.Items) {
			return nil, gameError("input_invalid", "导出序号无效。")
		}
		var avatar map[string]any
		_ = json.Unmarshal(t.Items[q.Offset], &avatar)
		var next *int
		if q.Offset+1 < len(t.Items) {
			n := q.Offset + 1
			next = &n
		}
		return map[string]any{"avatar": avatar, "next_offset": next}, nil
	}
	return nil, gameError("operation_denied", "交换传输操作不存在。")
}
