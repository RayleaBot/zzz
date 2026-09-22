package app

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/localdata"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"
)

type InteractionProfile struct {
	Ref                 string              `json:"ref"`
	Owner               Subject             `json:"owner"`
	Poke                bool                `json:"poke"`
	Lists               map[string][]string `json:"lists"`
	LastImage           string              `json:"last_image,omitempty"`
	LastImageMS         int64               `json:"last_image_ms,omitempty"`
	LastImageTargetType string              `json:"last_image_target_type,omitempty"`
	LastImageTargetID   string              `json:"last_image_target_id,omitempty"`
	LastEvent           string              `json:"last_event,omitempty"`
	LastPokeMS          int64               `json:"last_poke_ms,omitempty"`
}
type InteractionStore struct {
	mu   sync.Mutex
	Path string
}

func interactionRef(owner Subject) string {
	raw, _ := json.Marshal(owner)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func (s *InteractionStore) read() ([]InteractionProfile, error) {
	items := []InteractionProfile{}
	err := localdata.Read(s.Path, &items)
	return items, err
}
func (s *InteractionStore) List() ([]InteractionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read()
}
func (s *InteractionStore) Get(owner Subject) (InteractionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return InteractionProfile{}, err
	}
	for _, v := range items {
		if v.Owner == owner {
			return v, nil
		}
	}
	return InteractionProfile{Ref: interactionRef(owner), Owner: owner, Lists: map[string][]string{}}, nil
}
func (s *InteractionStore) edit(owner Subject, fn func(*InteractionProfile) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return err
	}
	i := slices.IndexFunc(items, func(v InteractionProfile) bool { return v.Owner == owner })
	if i < 0 {
		if len(items) >= 2000 {
			return gameError("interaction_limit", "互动偏好超过2000个主体，请先清理。")
		}
		items = append(items, InteractionProfile{Ref: interactionRef(owner), Owner: owner, Lists: map[string][]string{}})
		i = len(items) - 1
	}
	if items[i].Lists == nil {
		items[i].Lists = map[string][]string{}
	}
	if err = fn(&items[i]); err != nil {
		return err
	}
	return localdata.Write(s.Path, items)
}
func chatOwner(event *rayleabot.EventContext) Subject {
	return Subject{event.Event.SourceProtocol, event.Event.SourceAdapter, event.Bot.ID, event.Event.Actor.ID}
}
func validInteractionOwner(s Subject) bool {
	return s.SourceProtocol != "" && s.SourceAdapter != "" && s.BotID != "" && s.ActorID != "" && len(s.ActorID) <= 256 && len(s.BotID) <= 256
}
func (a *App) interactionEligible(owner Subject, target rayleabot.Target) (bool, error) {
	if !validInteractionOwner(owner) {
		return false, nil
	}
	v, err := a.Interactions.Get(owner)
	if err != nil {
		return false, err
	}
	if !v.Poke {
		return false, nil
	}
	if target.Type == "group" {
		config, err := a.Groups.Config(GroupScope{owner.SourceProtocol, owner.SourceAdapter, owner.BotID, target.ID})
		if err != nil {
			return false, err
		}
		if config.Enabled != nil && !*config.Enabled {
			return false, nil
		}
	}
	return true, nil
}

func (s *InteractionStore) claimPoke(owner Subject, eventID string, now int64) (bool, error) {
	claimed := false
	err := s.edit(owner, func(v *InteractionProfile) error {
		if !v.Poke || v.LastEvent == eventID || v.LastPokeMS+10000 > now {
			return nil
		}
		v.LastEvent = eventID
		v.LastPokeMS = now
		claimed = true
		return nil
	})
	return claimed, err
}
func (a *App) handlePoke(ctx context.Context, event *rayleabot.EventContext) error {
	owner := chatOwner(event)
	if asText(event.Event.Payload["target_id"]) != event.Bot.ID || owner.ActorID == owner.BotID {
		return event.Result(map[string]any{"handled": false})
	}
	eligible, err := a.interactionEligible(owner, event.Event.Target)
	if err != nil {
		return event.Fail(PublicError(err).Code, PublicError(err).Message)
	}
	if !eligible {
		return event.Result(map[string]any{"handled": false})
	}
	claimed, err := a.Interactions.claimPoke(owner, event.Event.EventID, time.Now().UnixMilli())
	if err != nil {
		return event.Fail(PublicError(err).Code, PublicError(err).Message)
	}
	if !claimed {
		return event.Result(map[string]any{"handled": false})
	}
	profile, _ := a.Interactions.Get(owner)
	ids := []string{}
	for _, list := range profile.Lists {
		for _, id := range list {
			if !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
	}
	if len(ids) == 0 {
		for _, e := range a.Catalog.Entries {
			if e.Kind == "character" {
				ids = append(ids, e.ID)
			}
		}
	}
	if len(ids) == 0 {
		return event.Result(map[string]any{"handled": false})
	}
	entry, ok := a.Catalog.Get(ids[rand.IntN(len(ids))])
	if !ok {
		return event.Result(map[string]any{"handled": false})
	}
	return a.sendCharacterMedia(ctx, event, entry, false)
}
func (a *App) sendMedia(event *rayleabot.EventContext, entry MediaEntry) error {
	a.Media.mu.Lock()
	v, err := a.Media.read(entry.Ref)
	a.Media.mu.Unlock()
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if err = a.rememberImage(event, entry.Ref); err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, rayleabot.Text(entry.Title+"\n来源："+entry.Source), rayleabot.Image("base64://"+base64.StdEncoding.EncodeToString(v.Data)))
}
func (a *App) sendCharacterMedia(ctx context.Context, event *rayleabot.EventContext, entry Entry, photoOnly bool) error {
	a.Media.mu.Lock()
	all, err := a.Media.entries()
	a.Media.mu.Unlock()
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	matches := []MediaEntry{}
	for _, v := range all {
		if v.CatalogID == entry.ID && (v.Category == "photo" || v.Category == "character" || v.Category == "birthday") {
			matches = append(matches, v)
		}
	}
	// Imported images and downloaded photos are drawn from together, as
	// miao draws from character-img and its upload directory.
	photos := a.characterPhotos(entry)
	if total := len(matches) + len(photos); total > 0 {
		if pick := rand.IntN(total); pick < len(matches) {
			return a.sendMedia(event, matches[pick])
		} else {
			return a.sendArtwork(event, photos[pick-len(matches)])
		}
	}
	if photoOnly {
		return event.SendText(strings.TrimSpace("暂无图片。" + a.pictureHint(a.Game.Pictures.Photos) + "也可在管理页的图库导入图片并关联角色。"))
	}
	return a.sendView(ctx, event, EntryView(a.Game, entry))
}

var relationLabels = map[string]string{"老婆": "wife", "媳妇": "wife", "女友": "gf", "女朋友": "gf", "老公": "husband", "男友": "bf", "男朋友": "bf", "女儿": "daughter", "儿子": "son", "伙伴": "partner"}

func (a *App) interactionCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	owner := chatOwner(event)
	if !validInteractionOwner(owner) {
		return event.SendText("聊天身份不完整。")
	}
	switch command {
	case "poke":
		if len(args) != 1 || (args[0] != "开启" && args[0] != "关闭") {
			return event.SendText("使用“" + a.Game.Prefix + "戳一戳 开启/关闭”。多个游戏同时开启时，按原神、星铁、绝区零顺序只响应一次。")
		}
		err := a.Interactions.edit(owner, func(p *InteractionProfile) error { p.Poke = args[0] == "开启"; return nil })
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.SendText(a.Game.Name + "戳一戳互动已" + args[0] + "。")
	case "original-image":
		p, err := a.Interactions.Get(owner)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		if p.LastImage == "" || time.Now().UnixMilli()-p.LastImageMS > 600000 || p.LastImageTargetType != event.Event.Target.Type || p.LastImageTargetID != event.Event.Target.ID {
			return event.SendText("最近十分钟没有由你查询的本地图片。")
		}
		if ref, ok := strings.CutPrefix(p.LastImage, "artwork:"); ok {
			source, file, _ := strings.Cut(ref, "/")
			return a.sendArtwork(event, artworkFile{source, file})
		}
		a.Media.mu.Lock()
		v, err := a.Media.read(p.LastImage)
		a.Media.mu.Unlock()
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.Send(event.Event.Target.Type, event.Event.Target.ID, rayleabot.Image("base64://"+base64.StdEncoding.EncodeToString(v.Data)))
	case "photo":
		entry, ok := a.Catalog.Resolve(strings.Join(args, " "), "character", a.aliasMap(event))
		if len(args) == 0 || !ok {
			return event.SendText("请提供可唯一识别的角色名。")
		}
		return a.sendCharacterMedia(ctx, event, entry, true)
	case "image-library":
		if len(args) < 1 {
			return event.SendText("使用“" + a.Game.Prefix + "图鉴图 分类 [关键词]”，分类见图库管理页。")
		}
		result, err := a.mediaAction("media.list", map[string]any{"category": args[0], "query": strings.Join(args[1:], " ")})
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		items := result["items"].([]MediaEntry)
		if len(items) == 0 {
			return event.SendText("本地图库没有匹配素材。")
		}
		return a.sendMedia(event, items[rand.IntN(len(items))])
	}
	if len(args) < 2 || relationLabels[args[0]] == "" {
		return event.SendText("使用“" + a.Game.Prefix + "互动 伙伴/老婆/老公/女友/男友/女儿/儿子 设置/添加/移除/列表/随机/照片/卡片 [角色名]”。这是游戏角色的娱乐偏好。")
	}
	kind, action := relationLabels[args[0]], args[1]
	profile, err := a.Interactions.Get(owner)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if action == "列表" {
		names := []string{}
		for _, id := range profile.Lists[kind] {
			if e, ok := a.Catalog.Get(id); ok {
				names = append(names, e.Name)
			}
		}
		if len(names) == 0 {
			return event.SendText("尚未设置此角色列表。")
		}
		return event.SendText(strings.Join(names, "、"))
	}
	if action == "设置" || action == "添加" || action == "移除" {
		if len(args) < 3 {
			return event.SendText("请提供角色名，多位以逗号分隔。")
		}
		names := strings.FieldsFunc(strings.Join(args[2:], " "), func(r rune) bool { return strings.ContainsRune(",，、;；", r) })
		if len(names) > 32 {
			return event.SendText("每份列表最多32位角色。")
		}
		ids := []string{}
		for _, name := range names {
			entry, ok := a.Catalog.Resolve(strings.TrimSpace(name), "character", a.aliasMap(event))
			if !ok {
				return event.SendText("角色名未唯一匹配：" + name)
			}
			if !slices.Contains(ids, entry.ID) {
				ids = append(ids, entry.ID)
			}
		}
		err = a.Interactions.edit(owner, func(p *InteractionProfile) error {
			next := ids
			if action == "添加" {
				next = slices.Clone(p.Lists[kind])
				for _, id := range ids {
					if !slices.Contains(next, id) {
						next = append(next, id)
					}
				}
			}
			if action == "移除" {
				next = slices.DeleteFunc(p.Lists[kind], func(id string) bool { return slices.Contains(ids, id) })
			}
			if len(next) > 32 {
				return gameError("input_invalid", "每份列表最多32位角色。")
			}
			p.Lists[kind] = next
			return nil
		})
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.SendText("角色互动列表已保存。")
	}
	if !slices.Contains([]string{"随机", "照片", "卡片"}, action) {
		return event.SendText("互动操作无效。")
	}
	ids := profile.Lists[kind]
	if len(ids) == 0 || action == "随机" {
		ids = []string{}
		for _, e := range a.Catalog.Entries {
			if e.Kind == "character" {
				ids = append(ids, e.ID)
			}
		}
	}
	if len(ids) == 0 {
		return event.SendText("没有可选择的角色资料。")
	}
	entry, ok := a.Catalog.Get(ids[rand.IntN(len(ids))])
	if !ok {
		return event.SendText("角色资料已变化，请重新设置列表。")
	}
	if action == "卡片" {
		return a.sendView(ctx, event, EntryView(a.Game, entry))
	}
	return a.sendCharacterMedia(ctx, event, entry, action == "照片")
}
func (a *App) interactionManage(action string, input map[string]any) (map[string]any, error) {
	s := a.Interactions
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return nil, err
	}
	if action == "interaction.list" {
		var q struct {
			Offset int `json:"offset"`
		}
		if decodeObject(input, &q) != nil || q.Offset < 0 {
			return nil, gameError("input_invalid", "分页无效。")
		}
		start, end := min(q.Offset, len(items)), min(q.Offset+50, len(items))
		var next *int
		if end < len(items) {
			next = &end
		}
		return map[string]any{"items": items[start:end], "next_offset": next}, nil
	}
	if action != "interaction.remove" || input["confirm"] != true {
		return nil, gameError("input_invalid", "请确认移除该主体的互动偏好。")
	}
	ref := asText(input["ref"])
	i := slices.IndexFunc(items, func(p InteractionProfile) bool { return p.Ref == ref })
	if i < 0 {
		return nil, gameError("interaction_missing", "互动偏好不存在。")
	}
	items = slices.Delete(items, i, i+1)
	err = localdata.Write(s.Path, items)
	return map[string]any{"removed": err == nil}, err
}
