package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/localdata"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

type GroupScope struct {
	Protocol string `json:"source_protocol"`
	Adapter  string `json:"source_adapter"`
	BotID    string `json:"bot_id"`
	GroupID  string `json:"group_id"`
}

func (s GroupScope) valid() bool {
	for _, v := range []string{s.Protocol, s.Adapter, s.BotID, s.GroupID} {
		if v == "" || len(v) > 256 {
			return false
		}
	}
	return true
}
func groupScope(event *rayleabot.EventContext) GroupScope {
	return GroupScope{Protocol: event.Event.SourceProtocol, Adapter: event.Event.SourceAdapter, BotID: event.Bot.ID, GroupID: event.Event.Target.ID}
}

type GroupConfig struct {
	Enabled      *bool             `json:"enabled,omitempty"`
	ImageReplies *bool             `json:"image_replies,omitempty"`
	Aliases      map[string]string `json:"aliases"`
}
type GroupData struct {
	Revision uint64      `json:"revision"`
	Scope    GroupScope  `json:"scope"`
	Config   GroupConfig `json:"config"`
	Rank     []RankEntry `json:"rank"`
	// RankSinceMS is when the ranking started.
	RankSinceMS int64 `json:"rank_since_ms,omitempty"`
	// QueryRanks are the UIDs in each query ranking, by ranking ID and UID.
	QueryRanks map[string]map[string]QueryRankMember `json:"query_ranks,omitempty"`
}

// GroupStore keeps each group's data in its own file. The settings every
// group message checks are kept in memory once read and replaced whenever the
// group is written; rankings stay on disk.
type GroupStore struct {
	mu        sync.Mutex
	Directory string
	configs   map[GroupScope]GroupConfig
}

func (s *GroupStore) file(scope GroupScope) string {
	raw, _ := json.Marshal(scope)
	sum := sha256.Sum256(raw)
	return filepath.Join(s.Directory, hex.EncodeToString(sum[:])+".json")
}
func (s *GroupStore) read(scope GroupScope) (GroupData, error) {
	if !scope.valid() {
		return GroupData{}, gameError("source_invalid", "群身份不完整。")
	}
	data := GroupData{Scope: scope, Config: GroupConfig{Aliases: map[string]string{}}, Rank: []RankEntry{}}
	err := localdata.Read(s.file(scope), &data)
	if data.Scope != scope {
		return GroupData{}, gameError("source_invalid", "群记录身份不一致。")
	}
	return data, err
}
func (s *GroupStore) Read(scope GroupScope) (GroupData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read(scope)
}

// Config is a group's settings.
func (s *GroupStore) Config(scope GroupScope) (GroupConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if config, ok := s.configs[scope]; ok {
		return config, nil
	}
	data, err := s.read(scope)
	if err != nil {
		return GroupConfig{}, err
	}
	s.remember(scope, data.Config)
	return data.Config, nil
}

func (s *GroupStore) remember(scope GroupScope, config GroupConfig) {
	if s.configs == nil || len(s.configs) >= 4096 {
		s.configs = map[GroupScope]GroupConfig{}
	}
	s.configs[scope] = config
}

func (s *GroupStore) Update(scope GroupScope, fn func(*GroupData) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.read(scope)
	if err != nil {
		return err
	}
	if err = fn(&data); err != nil {
		return err
	}
	data.Revision++
	if err = localdata.Write(s.file(scope), data); err != nil {
		delete(s.configs, scope)
		return err
	}
	s.remember(scope, data.Config)
	return nil
}
func (s *GroupStore) List(page int) ([]map[string]any, *int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	files, err := os.ReadDir(s.Directory)
	if os.IsNotExist(err) {
		return []map[string]any{}, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	out := []map[string]any{}
	matched := 0
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		matched++
		if matched <= page*50 {
			continue
		}
		if len(out) == 50 {
			next := page + 1
			return out, &next, nil
		}
		var data GroupData
		if err := localdata.Read(filepath.Join(s.Directory, f.Name()), &data); err != nil {
			return nil, nil, err
		}
		out = append(out, map[string]any{"scope": data.Scope, "config": data.Config, "rank_entries": len(data.Rank), "revision": data.Revision})
	}
	return out, nil, nil
}
func applyGroupConfig(event *rayleabot.EventContext, config GroupConfig) {
	copy := map[string]any{}
	for k, v := range event.Config {
		copy[k] = v
	}
	if config.ImageReplies != nil {
		copy["image_replies"] = *config.ImageReplies
	}
	aliases := settings(event).CustomAliases
	merged := map[string]string{}
	for k, v := range aliases {
		merged[k] = v
	}
	for k, v := range config.Aliases {
		merged[k] = v
	}
	copy["custom_aliases"] = merged
	event.Config = copy
}
func groupAdministrator(event *rayleabot.EventContext) bool {
	return event.Event.Actor.Role == "admin" || event.Event.Actor.Role == "owner" || slices.Contains(event.SuperAdmins, event.Event.Actor.ID)
}
func (a *App) groupSettings(event *rayleabot.EventContext) error {
	if !groupAdministrator(event) {
		return event.SendText("只有群管理员、群主或机器人超级管理员可调整本群游戏设置。")
	}
	scope := groupScope(event)
	args := event.Event.Args()
	if len(args) == 0 {
		data, err := a.Groups.Read(scope)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		enabled := "开启"
		if data.Config.Enabled != nil && !*data.Config.Enabled {
			enabled = "关闭"
		}
		mode := "跟随全局"
		if data.Config.ImageReplies != nil {
			mode = "文本"
			if *data.Config.ImageReplies {
				mode = "图片"
			}
		}
		return event.SendText(fmt.Sprintf("本群%s：%s · 回复%s · 群别名 %d 个\n群设置 开启/关闭/图片/文本/默认\n群设置 别名 名称 角色或装备ID\n群设置 删除别名 名称", a.Game.Name, enabled, mode, len(data.Config.Aliases)))
	}
	err := a.Groups.Update(scope, func(data *GroupData) error {
		switch args[0] {
		case "开启", "关闭":
			v := args[0] == "开启"
			data.Config.Enabled = &v
		case "图片", "文本":
			v := args[0] == "图片"
			data.Config.ImageReplies = &v
		case "默认":
			data.Config = GroupConfig{Aliases: map[string]string{}}
		case "别名":
			if len(args) != 3 || len(args[1]) > 64 || strings.TrimSpace(args[1]) == "" {
				return gameError("input_invalid", "别名格式：群设置 别名 名称 角色或装备ID。")
			}
			if _, ok := a.Catalog.Get(args[2]); !ok {
				return gameError("entry_missing", "未找到目标资料 ID。")
			}
			if data.Config.Aliases == nil {
				data.Config.Aliases = map[string]string{}
			}
			if _, exists := data.Config.Aliases[strings.ToLower(args[1])]; !exists && len(data.Config.Aliases) >= 64 {
				return gameError("input_invalid", "每群最多 64 个自定义别名。")
			}
			data.Config.Aliases[strings.ToLower(args[1])] = args[2]
			normalized, _, err := a.Catalog.validateAliases(data.Config.Aliases, 64)
			if err != nil {
				return err
			}
			data.Config.Aliases = normalized
		case "删除别名":
			if len(args) != 2 {
				return gameError("input_invalid", "请提供要删除的群别名。")
			}
			delete(data.Config.Aliases, strings.ToLower(args[1]))
		default:
			return gameError("input_invalid", "支持开启、关闭、图片、文本、默认、别名和删除别名。")
		}
		return nil
	})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText("本群" + a.Game.Name + "设置已保存。")
}
func (a *App) manageGroups(action string, input map[string]any) (map[string]any, error) {
	if action == "groups.list" {
		var q struct {
			Page int `json:"page"`
		}
		if decodeObject(input, &q) != nil || q.Page < 0 || q.Page > 100000 {
			return nil, gameError("input_invalid", "群列表页码无效。")
		}
		items, next, err := a.Groups.List(q.Page)
		return map[string]any{"items": items, "next_page": next}, err
	}
	if action == "groups.clear" || action == "groups.set" {
		var value struct {
			Scope    GroupScope  `json:"scope"`
			Revision uint64      `json:"revision"`
			Config   GroupConfig `json:"config"`
		}
		if decodeObject(input, &value) != nil || !value.Scope.valid() {
			return nil, gameError("input_invalid", "请提供完整群身份。")
		}
		if len(value.Config.Aliases) > 64 {
			return nil, gameError("input_invalid", "每群最多 64 个别名。")
		}
		if action == "groups.set" {
			normalized, _, err := a.Catalog.validateAliases(value.Config.Aliases, 64)
			if err != nil {
				return nil, err
			}
			value.Config.Aliases = normalized
		}
		for alias, id := range value.Config.Aliases {
			if strings.TrimSpace(alias) == "" || len(alias) > 64 {
				return nil, gameError("input_invalid", "别名不应为空或超过 64 字节。")
			}
			if _, exists := a.Catalog.Get(id); !exists {
				return nil, gameError("entry_missing", "别名指向的资料不存在。")
			}
		}
		err := a.Groups.Update(value.Scope, func(d *GroupData) error {
			if d.Revision != value.Revision {
				return gameError("group_changed", "群设置或参评记录已更新，请刷新后重试。")
			}
			if action == "groups.set" {
				aliases := map[string]string{}
				for k, v := range value.Config.Aliases {
					key := strings.ToLower(strings.TrimSpace(k))
					if _, exists := aliases[key]; exists {
						return gameError("input_invalid", "群别名存在重复名称。")
					}
					aliases[key] = v
				}
				value.Config.Aliases = aliases
				d.Config = value.Config
				return nil
			}
			d.Config = GroupConfig{Aliases: map[string]string{}}
			d.Rank = []RankEntry{}
			d.QueryRanks = nil
			return nil
		})
		return map[string]any{"updated": err == nil}, err
	}
	return nil, gameError("operation_denied", "群管理操作不存在。")
}
