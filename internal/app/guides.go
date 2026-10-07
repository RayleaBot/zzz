package app

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/RayleaBot/zzz/internal/localdata"
)

type GuideSource struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Collections []string `json:"collections"`
}

var guideSources = []GuideSource{{"1", "新艾利都快讯", []string{"2712859"}}, {"2", "清茶沐沐Kiyotya", []string{"2727116"}}, {"3", "小橙子阿", []string{"2721968"}}, {"4", "猫冬", []string{"2724610"}}, {"5", "月光中心", []string{"2722266"}}, {"6", "苦雪的清心花凉糕Suki", []string{"2723586"}}, {"7", "HoYo青枫", []string{"2716049"}}}

// GuideConfig is the default source, "0" for ZZZ-Plugin's all, and how many
// sources all shows.
type GuideConfig struct {
	Revision     uint64 `json:"revision"`
	Default      string `json:"default_source"`
	ForwardCount int    `json:"forward_count,omitempty"`
}

// forwardCount is how many sources 攻略all shows, upstream's
// max_forward_guides.
func (c GuideConfig) forwardCount() int {
	if c.ForwardCount < 1 || c.ForwardCount > len(guideSources) {
		return 4
	}
	return c.ForwardCount
}

type GuideSettings struct {
	mu   sync.Mutex
	Path string
}

func (s *GuideSettings) Manage(action string, input map[string]any) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := GuideConfig{Default: "1"}
	if err := localdata.Read(s.Path, &v); err != nil {
		return nil, err
	}
	if action == "guides.configure" {
		var q struct {
			Revision     uint64 `json:"revision"`
			Default      string `json:"default_source"`
			ForwardCount *int   `json:"forward_count"`
		}
		if decodeObject(input, &q) != nil || q.Default != "0" && !slices.ContainsFunc(guideSources, func(s GuideSource) bool { return s.ID == q.Default }) || q.ForwardCount != nil && (*q.ForwardCount < 1 || *q.ForwardCount > len(guideSources)) {
			return nil, gameError("input_invalid", "攻略来源无效。")
		}
		if q.Revision != v.Revision {
			return nil, gameError("settings_changed", "攻略设置已变化，请刷新。")
		}
		v.Default = q.Default
		if q.ForwardCount != nil {
			v.ForwardCount = *q.ForwardCount
		}
		v.Revision++
		if err := localdata.Write(s.Path, v); err != nil {
			return nil, err
		}
	}
	return map[string]any{"sources": guideSources, "settings": v}, nil
}
func (c PublicContentClient) guides(ctx context.Context, q ContentQuery) (map[string]any, error) {
	sources := guideSources
	index := slices.IndexFunc(sources, func(v GuideSource) bool { return v.ID == q.Source })
	if index < 0 || q.CollectionOffset < 0 || q.CollectionOffset >= len(sources[index].Collections) || len([]rune(q.Query)) > 100 || q.Offset < 0 {
		return nil, gameError("input_invalid", "攻略来源或筛选无效。")
	}
	source := sources[index]
	data, err := c.collection(ctx, source.Collections[q.CollectionOffset])
	if err != nil {
		return nil, err
	}
	raw, ok := data["posts"].([]any)
	if !ok || len(raw) > 2000 {
		return nil, gameError("public_invalid", "攻略合集格式暂不兼容。")
	}
	matches := []any{}
	for _, v := range raw {
		p := asObject(asObject(v)["post"])
		if p == nil {
			p = asObject(v)
		}
		if q.Query == "" || strings.Contains(publicText(asText(p["subject"])), q.Query) {
			matches = append(matches, v)
		}
	}
	start, end := min(q.Offset, len(matches)), min(q.Offset+10, len(matches))
	items := []PublicPost{}
	for _, v := range matches[start:end] {
		p, err := normalizePublicPost(v, true)
		if err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	next := -1
	if end < len(matches) {
		next = end
	}
	return map[string]any{"items": items, "total": len(matches), "source": source, "collection_offset": q.CollectionOffset, "next_offset": next, "fetched_at_ms": time.Now().UnixMilli()}, nil
}
