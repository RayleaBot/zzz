package app

import (
	"context"
	"github.com/RayleaBot/plugin-zzz/internal/localdata"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type GuideSource struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Collections []string `json:"collections"`
}

var guideSources = map[string][]GuideSource{
	"genshin":  {{"1", "西风驿站", []string{"2319292", "2319293", "2319295", "2319296", "2319299", "2319294", "2319298", "642956"}}, {"2", "原神观测枢", []string{"813033"}}, {"3", "派蒙喵喵屋", []string{"341284"}}, {"4", "OH是姜姜呀", []string{"341523"}}, {"5", "曉K", []string{"1582613"}}, {"6", "坤易", []string{"22148"}}, {"7", "婧枫赛赛", []string{"1812949"}}},
	"starrail": {{"1", "初始镜像", []string{"1996095"}}, {"2", "小橙子阿", []string{"1998643"}}, {"3", "星穹中心", []string{"2029394", "2009142", "2038092"}}},
	"zzz":      {{"1", "新艾利都快讯", []string{"2712859"}}, {"2", "清茶沐沐Kiyotya", []string{"2727116"}}, {"3", "小橙子阿", []string{"2721968"}}, {"4", "猫冬", []string{"2724610"}}, {"5", "月光中心", []string{"2722266"}}, {"6", "苦雪的清心花凉糕Suki", []string{"2723586"}}, {"7", "HoYo青枫", []string{"2716049"}}},
}

type GuideConfig struct {
	Revision uint64 `json:"revision"`
	Default  string `json:"default_source"`
}
type GuideSettings struct {
	mu   sync.Mutex
	Path string
}

func (s *GuideSettings) Manage(game, action string, input map[string]any) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := GuideConfig{Default: "1"}
	if err := localdata.Read(s.Path, &v); err != nil {
		return nil, err
	}
	if action == "guides.configure" {
		var q GuideConfig
		if decodeObject(input, &q) != nil || !slices.ContainsFunc(guideSources[game], func(s GuideSource) bool { return s.ID == q.Default }) {
			return nil, gameError("input_invalid", "攻略来源无效。")
		}
		if q.Revision != v.Revision {
			return nil, gameError("settings_changed", "攻略设置已变化，请刷新。")
		}
		v.Default = q.Default
		v.Revision++
		if err := localdata.Write(s.Path, v); err != nil {
			return nil, err
		}
	}
	return map[string]any{"sources": guideSources[game], "settings": v}, nil
}
func (c PublicContentClient) guides(ctx context.Context, game string, q ContentQuery) (map[string]any, error) {
	sources := guideSources[game]
	index := slices.IndexFunc(sources, func(v GuideSource) bool { return v.ID == q.Source })
	if index < 0 || q.CollectionOffset < 0 || q.CollectionOffset >= len(sources[index].Collections) || len([]rune(q.Query)) > 100 || q.Offset < 0 {
		return nil, gameError("input_invalid", "攻略来源或筛选无效。")
	}
	source := sources[index]
	gid, _ := publicGame(game)
	params := url.Values{"gids": {gid}, "order_type": {"2"}, "collection_id": {source.Collections[q.CollectionOffset]}}
	data, err := c.get(ctx, "https://bbs-api.mihoyo.com/post/wapi/getPostFullInCollection?"+params.Encode(), nil)
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
		p, err := normalizePublicPost(game, v, true)
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
func (a *App) mapQuery(input map[string]any) (map[string]any, error) {
	if a.Game.ID != "genshin" {
		return nil, gameError("operation_denied", "参考地图服务仅用于原神。")
	}
	var q struct {
		Query string `json:"query"`
		MapID int    `json:"map_id"`
	}
	if decodeObject(input, &q) != nil || len([]rune(strings.TrimSpace(q.Query))) < 1 || len([]rune(q.Query)) > 80 || !slices.Contains([]int{0, 7, 9}, q.MapID) {
		return nil, gameError("input_invalid", "请输入地图资源名并选择地图。")
	}
	params := url.Values{"resource_name": {q.Query}, "is_cluster": {"false"}}
	if q.MapID > 0 {
		params.Set("map_id", strconv.Itoa(q.MapID))
	}
	return map[string]any{"url": "https://map.minigg.cn/map/get_map?" + params.Encode(), "source": "逍遥参考 · minigg 地图服务", "query": q.Query}, nil
}
