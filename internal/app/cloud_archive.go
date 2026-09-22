package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/RayleaBot/plugin-zzz/internal/localdata"
	"math"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

var cloudTreePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,32}$`)
var cloudTalentPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,15}$`)

type CloudArchive struct {
	Revision uint64                     `json:"revision"`
	UID      string                     `json:"uid"`
	Region   string                     `json:"region"`
	SavedMS  int64                      `json:"saved_ms"`
	Avatars  map[string]json.RawMessage `json:"avatars"`
}
type CloudArchiveStore struct {
	mu        sync.Mutex
	Directory string
}

func (s *CloudArchiveStore) file(provider string, choice Selection) string {
	raw, _ := json.Marshal([]string{provider, choice.AccountRef, choice.RoleRef})
	key := sha256.Sum256(raw)
	return filepath.Join(s.Directory, hex.EncodeToString(key[:])+".json")
}
func (s *CloudArchiveStore) Read(provider string, choice Selection) (CloudArchive, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := CloudArchive{Avatars: map[string]json.RawMessage{}}
	err := localdata.Read(s.file(provider, choice), &out)
	return out, err
}
func (s *CloudArchiveStore) Update(provider string, choice Selection, revision uint64, role Role, avatars map[string]json.RawMessage, remove bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	file := s.file(provider, choice)
	out := CloudArchive{Avatars: map[string]json.RawMessage{}}
	if err := localdata.Read(file, &out); err != nil {
		return err
	}
	if out.Avatars == nil {
		out.Avatars = map[string]json.RawMessage{}
	}
	if out.Revision != revision {
		return gameError("history_changed", "交换档案已变化，请刷新后重试。")
	}
	if out.UID != "" && (out.UID != role.UID || out.Region != role.Region) {
		return gameError("role_missing", "交换档案与当前授权角色不一致。")
	}
	if remove {
		out.Avatars = map[string]json.RawMessage{}
	} else {
		for id, raw := range avatars {
			out.Avatars[id] = raw
		}
	}
	if len(out.Avatars) > 250 {
		return gameError("history_limit", "交换档案最多保存 250 个角色。")
	}
	out.Revision++
	out.UID = role.UID
	out.Region = role.Region
	out.SavedMS = time.Now().UnixMilli()
	raw, err := json.Marshal(out)
	if err != nil || len(raw) > 4*1024*1024 {
		return gameError("history_limit", "交换档案超过 4 MiB 上限。")
	}
	return localdata.Write(file, out)
}
func cleanCloudGear(value any) (map[string]any, error) {
	obj := asObject(value)
	if obj == nil {
		return nil, gameError("cloud_invalid", "装备字段应为对象。")
	}
	out := map[string]any{}
	for _, key := range []string{"id", "mainId", "level", "star"} {
		if raw, exists := obj[key]; exists {
			v, ok := cloudInteger(raw, 0, 1000000000)
			if !ok {
				return nil, gameError("cloud_invalid", "装备编号或数值无效。")
			}
			out[key] = v
		}
	}
	if v, ok := obj["name"].(string); ok && len([]rune(v)) <= 128 {
		out["name"] = plainGameText(v)
	}
	if raw, exists := obj["attrIds"]; exists {
		attrs, ok := raw.([]any)
		if !ok || len(attrs) > 32 {
			return nil, gameError("cloud_invalid", "装备副词条数量无效。")
		}
		rows := []any{}
		for _, v := range attrs {
			if o := asObject(v); o != nil {
				next := map[string]any{}
				for _, key := range []string{"id", "count", "step"} {
					n, ok := cloudInteger(o[key], 0, 1000000000)
					if !ok {
						return nil, gameError("cloud_invalid", "装备副词条格式无效。")
					}
					next[key] = n
				}
				rows = append(rows, next)
			} else {
				text := asText(v)
				if text == "" || len(text) > 64 || strings.Trim(text, "0123456789,") != "" {
					return nil, gameError("cloud_invalid", "装备副词条编号无效。")
				}
				rows = append(rows, text)
			}
		}
		out["attrIds"] = rows
	}
	return out, nil
}
func cleanCloudAvatar(game string, value any) (string, json.RawMessage, error) {
	obj := asObject(value)
	id, ok := cloudInteger(obj["id"], 1, 1000000000)
	if !ok || game == "genshin" && id < 10000000 || game == "starrail" && id >= 10000000 {
		return "", nil, gameError("cloud_invalid", "角色编号与所选游戏不匹配。")
	}
	out := map[string]any{"id": id, "_source": "share"}
	for _, key := range []string{"level", "promote", "cons", "fetter", "costume", "_time", "_update", "_talent"} {
		if raw, exists := obj[key]; exists {
			if strings.HasPrefix(key, "_") {
				if key == "_talent" {
					if flag, ok := raw.(bool); ok {
						out[key] = flag
						continue
					}
				}
				if text, ok := raw.(string); ok && len(text) <= 64 {
					valid := false
					for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
						if _, err := time.Parse(layout, text); err == nil {
							valid = true
							break
						}
					}
					if valid {
						out[key] = text
						continue
					}
				}
			}
			n, valid := challengeNumber(raw)
			if !valid || math.Trunc(n) != n {
				return "", nil, gameError("cloud_invalid", "角色等级或时间字段无效。")
			}
			out[key] = n
		}
	}
	for _, key := range []string{"name", "elem"} {
		if v, ok := obj[key].(string); ok {
			if len([]rune(v)) > 128 {
				return "", nil, gameError("cloud_invalid", "角色名称或元素过长。")
			}
			out[key] = plainGameText(v)
		}
	}
	if weapon, exists := obj["weapon"]; exists && weapon == nil {
		out["weapon"] = nil
	}
	if weapon, exists := obj["weapon"]; exists && weapon != nil {
		w := asObject(weapon)
		if w == nil {
			return "", nil, gameError("cloud_invalid", "武器数据无效。")
		}
		copy := map[string]any{}
		for _, key := range []string{"id", "level", "promote", "affix"} {
			if v, exists := w[key]; exists {
				n, valid := cloudInteger(v, 0, 1000000000)
				if !valid {
					return "", nil, gameError("cloud_invalid", "武器字段无效。")
				}
				copy[key] = n
			}
		}
		if name, ok := w["name"].(string); ok && len([]rune(name)) <= 128 {
			copy["name"] = plainGameText(name)
		}
		out["weapon"] = copy
	}
	if raw, exists := obj["talent"]; exists {
		talents := asObject(raw)
		if talents == nil || len(talents) > 32 {
			return "", nil, gameError("cloud_invalid", "技能字段无效。")
		}
		copy := map[string]any{}
		for key, v := range talents {
			if !cloudTalentPattern.MatchString(key) || !slices.Contains(strings.Fields("a a2 a3 e e1 e2 e3 q q2 q3 t t2 z xe xe2 me me2 mt mt1 mt2"), key) {
				continue
			}
			if data := asObject(v); data != nil {
				v = data["level"]
			}
			n, ok := cloudInteger(v, 0, 100)
			if !ok {
				return "", nil, gameError("cloud_invalid", "技能等级无效。")
			}
			copy[key] = n
		}
		out["talent"] = copy
	}
	if raw, exists := obj["trees"]; exists {
		trees, ok := raw.([]any)
		if !ok || len(trees) > 128 {
			return "", nil, gameError("cloud_invalid", "行迹数据无效。")
		}
		copy := []string{}
		for _, v := range trees {
			text := asText(v)
			if !cloudTreePattern.MatchString(text) {
				return "", nil, gameError("cloud_invalid", "行迹编号无效。")
			}
			copy = append(copy, text)
		}
		out["trees"] = copy
	}
	for _, field := range []string{"artis", "mysArtis"} {
		if raw, exists := obj[field]; exists {
			slots := asObject(raw)
			if slots == nil || len(slots) > 6 {
				return "", nil, gameError("cloud_invalid", "装备部位数据无效。")
			}
			copy := map[string]any{}
			for key, v := range slots {
				slot, e := strconv.Atoi(strings.TrimPrefix(key, "arti"))
				limit := 6
				if game == "genshin" {
					limit = 5
				}
				if e != nil || slot < 1 || slot > limit {
					return "", nil, gameError("cloud_invalid", "装备部位编号无效。")
				}
				gear, err := cleanCloudGear(v)
				if err != nil {
					return "", nil, err
				}
				canonical := strconv.Itoa(slot)
				if _, exists := copy[canonical]; exists {
					return "", nil, gameError("cloud_invalid", "装备部位重复。")
				}
				copy[canonical] = gear
			}
			out[field] = copy
		}
	}
	raw, err := json.Marshal(out)
	if err != nil || len(raw) > 32*1024 {
		return "", nil, gameError("history_limit", "单角色交换数据超过 32 KiB。")
	}
	return strconv.Itoa(id), raw, nil
}
func cleanCloudPlayer(game, uid string, value any) (map[string]json.RawMessage, error) {
	data := asObject(value)
	if asText(data["uid"]) != uid || !uidPattern.MatchString(uid) {
		return nil, gameError("cloud_invalid", "面板文件 UID 与已授权角色不一致。")
	}
	avatars := map[string]json.RawMessage{}
	add := func(v any) error {
		id, raw, err := cleanCloudAvatar(game, v)
		if err != nil {
			return err
		}
		if _, exists := avatars[id]; exists {
			return gameError("cloud_invalid", "面板角色编号重复。")
		}
		avatars[id] = raw
		return nil
	}
	switch rows := data["avatars"].(type) {
	case map[string]any:
		if len(rows) > 250 {
			return nil, gameError("history_limit", "面板文件超过 250 个角色。")
		}
		for _, v := range rows {
			if err := add(v); err != nil {
				return nil, err
			}
		}
	case []any:
		if len(rows) > 250 {
			return nil, gameError("history_limit", "面板文件超过 250 个角色。")
		}
		for _, v := range rows {
			if err := add(v); err != nil {
				return nil, err
			}
		}
	default:
		return nil, gameError("cloud_invalid", "面板文件缺少角色数据。")
	}
	if len(avatars) == 0 {
		return nil, gameError("cloud_invalid", "面板文件没有角色。")
	}
	return avatars, nil
}
func cloudArchiveSummary(archive CloudArchive) map[string]any {
	ids := []string{}
	for id := range archive.Avatars {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	items := []map[string]any{}
	for _, id := range ids {
		var v map[string]any
		_ = json.Unmarshal(archive.Avatars[id], &v)
		items = append(items, map[string]any{"id": id, "name": cloudText(v["name"]), "level": v["level"], "cons": v["cons"]})
	}
	return map[string]any{"uid": archive.UID, "region": archive.Region, "revision": archive.Revision, "saved_ms": archive.SavedMS, "items": items}
}
