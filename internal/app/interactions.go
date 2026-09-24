package app

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/localdata"
	"slices"
	"strings"
	"sync"
	"time"
)

type InteractionProfile struct {
	Ref                 string  `json:"ref"`
	Owner               Subject `json:"owner"`
	LastImage           string  `json:"last_image,omitempty"`
	LastImageMS         int64   `json:"last_image_ms,omitempty"`
	LastImageTargetType string  `json:"last_image_target_type,omitempty"`
	LastImageTargetID   string  `json:"last_image_target_id,omitempty"`
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
	return InteractionProfile{Ref: interactionRef(owner), Owner: owner}, nil
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
		items = append(items, InteractionProfile{Ref: interactionRef(owner), Owner: owner})
		i = len(items) - 1
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

// originalImage answers 原图 with the portrait of the panel or damage image
// the requester last viewed in this chat within ten minutes, as ZZZ-Plugin
// sends the portrait of a panel message.
func (a *App) originalImage(event *rayleabot.EventContext) error {
	owner := chatOwner(event)
	if !validInteractionOwner(owner) {
		return event.SendText("聊天身份不完整。")
	}
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
	if ref, ok := strings.CutPrefix(p.LastImage, "panel:"); ok {
		id, name, _ := strings.Cut(ref, "/")
		if raw, err := a.PanelImages.Read(id, name); err == nil {
			return event.Send(event.Event.Target.Type, event.Event.Target.ID, rayleabot.Image("base64://"+base64.StdEncoding.EncodeToString(raw)))
		}
	}
	return event.SendText("未找到原图")
}
