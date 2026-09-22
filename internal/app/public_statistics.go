package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"slices"
	"time"
)

var statisticsURLs = map[string]string{"ownership": "https://api.lelaer.com/ys/getRoleAvg.php?star=all&lang=zh-Hans", "abyss": "https://api.yshelper.com/ys/getAbyssRank.php?star=all&role=all&lang=zh-Hans", "stygian": "https://api.lelaer.com/ys/getAbyssRank2.php?star=all&role=all&lang=zh-Hans"}

type PublicTeam struct {
	Names   []string `json:"names"`
	IDs     []string `json:"ids"`
	Up      *float64 `json:"up"`
	Middle  *float64 `json:"middle"`
	Down    *float64 `json:"down"`
	UseRate *float64 `json:"use_rate"`
}

func (c PublicContentClient) statsRaw(ctx context.Context, mode string) (map[string]any, error) {
	address := statisticsURLs[mode]
	if address == "" {
		return nil, gameError("input_invalid", "统计来源无效。")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, gameError("statistics_unavailable", "公开统计服务暂时不可用。")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, gameError("statistics_unavailable", "公开统计服务暂时不可用。")
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 2*1024*1024+1))
	if err != nil || len(raw) > 2*1024*1024 {
		return nil, gameError("statistics_invalid", "公开统计数据过大或无法读取。")
	}
	var data map[string]any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if d.Decode(&data) != nil || d.Decode(new(any)) != io.EOF || asText(data["code"]) != "200" || data["result"] == nil {
		return nil, gameError("statistics_invalid", "公开统计响应暂不兼容。")
	}
	return data, nil
}
func statNumber(v any) *float64 {
	n, ok := challengeNumber(v)
	if !ok || !finiteRange(n, -1e15, 1e15) {
		return nil
	}
	return &n
}
func (c Catalog) statIdentity(name string) string {
	for _, e := range c.Entries {
		if e.Kind != "character" {
			continue
		}
		if name == e.Name || slices.Contains(e.Aliases, name) {
			return e.ID
		}
	}
	return ""
}
func (a *App) publicStatistics(ctx context.Context, q ContentQuery) (map[string]any, error) {
	if a.Game.ID != "genshin" || !slices.Contains([]string{"ownership", "abyss", "stygian"}, q.Source) {
		return nil, gameError("operation_denied", "此参考统计仅适用于原神。")
	}
	data, err := a.Content.statsRaw(ctx, q.Source)
	if err != nil {
		return nil, err
	}
	rows := []map[string]any{}
	teams := []PublicTeam{}
	meta := map[string]any{}
	for _, k := range []string{"version", "last_update", "title", "star36_rate", "star36_once_rate", "restart_times_avg", "nandu"} {
		if v, ok := data[k]; ok {
			meta[k] = v
		}
	}
	if q.Source == "ownership" {
		holdData, err := a.Content.statsRaw(ctx, "abyss")
		holdings := map[string]*float64{}
		holdingAvailable := err == nil
		if err == nil {
			for _, v := range asList(holdData["has_list"]) {
				m := asObject(v)
				holdings[asText(m["name"])] = statNumber(m["own_rate"])
			}
		}
		for _, v := range asList(data["result"]) {
			m := asObject(v)
			name := asText(m["role"])
			row := map[string]any{"name": name, "id": a.Catalog.statIdentity(name), "holding_rate": holdings[name], "sample": statNumber(m["role_sum"]), "level": statNumber(m["avg_level"]), "average_constellation": statNumber(m["avg_class"]), "constellations": []*float64{statNumber(m["c0"]), statNumber(m["c1"]), statNumber(m["c2"]), statNumber(m["c3"]), statNumber(m["c4"]), statNumber(m["c5"]), statNumber(m["c6"])}}
			rows = append(rows, row)
		}
		meta["holding_source_available"] = holdingAvailable
		meta["holding_source"] = "Yshelper"
		meta["constellation_source"] = "Lelaer"
	} else {
		results := asList(data["result"])
		if len(results) < 1 {
			return nil, gameError("statistics_invalid", "公开统计缺少分组。")
		}
		for _, tier := range asList(results[0]) {
			t := asObject(tier)
			for _, v := range asList(t["list"]) {
				m := asObject(v)
				name := asText(m["name"])
				rows = append(rows, map[string]any{"name": name, "id": a.Catalog.statIdentity(name), "tier": asText(t["rank_name"]), "use_rate": statNumber(m["use_rate"]), "holding_rate": statNumber(m["own_rate"]), "used": statNumber(m["use"]), "owned": statNumber(m["own"]), "time": statNumber(m["time"]), "constellations": []*float64{statNumber(m["c0_rate"]), statNumber(m["c1_rate"]), statNumber(m["c2_rate"]), statNumber(m["c3_rate"]), statNumber(m["c4_rate"]), statNumber(m["c5_rate"]), statNumber(m["c6_rate"])}})
			}
		}
		names := map[string]string{}
		for _, v := range asList(data["has_list"]) {
			m := asObject(v)
			names[asText(m["avatar"])] = asText(m["name"])
		}
		for _, group := range results {
			for _, v := range asList(group) {
				m := asObject(v)
				chars, ok := m["role"].([]any)
				if !ok {
					continue
				}
				team := PublicTeam{Names: []string{}, IDs: []string{}, Up: statNumber(m["up_use_num"]), Middle: statNumber(m["mid_use_num"]), Down: statNumber(m["down_use_num"]), UseRate: statNumber(m["use_rate"])}
				for _, raw := range chars {
					char := asObject(raw)
					name := asText(char["name"])
					if name == "" {
						name = names[asText(char["avatar"])]
					}
					if name == "" {
						name = "资料未匹配"
					}
					team.Names = append(team.Names, name)
					team.IDs = append(team.IDs, a.Catalog.statIdentity(name))
				}
				if len(team.Names) > 0 && len(team.Names) <= 4 {
					teams = append(teams, team)
				}
			}
		}
	}
	if len(rows) > 1000 || len(teams) > 1000 {
		return nil, gameError("statistics_invalid", "统计结果数量超过上限。")
	}
	return map[string]any{"rows": rows, "teams": teams, "meta": meta, "source": q.Source, "fetched_at_ms": time.Now().UnixMilli(), "note": "来源为样本统计，不代表全体玩家；命座分布是持有者中的比例，持有率来自独立样本，缺失值保持未知。"}, nil
}
func (c PublicContentClient) estimate(ctx context.Context, game string) (map[string]any, error) {
	keyword := map[string]string{"genshin": "原石统计汇总", "starrail": "星琼统计汇总", "zzz": "菲林统计汇总"}[game]
	data, err := c.get(ctx, "https://bbs-api.miyoushe.com/painter/api/user_instant/search/list?uid=137101761&size=20&offset=0&sort_type=2&keyword="+url.QueryEscape(keyword), nil)
	if err != nil {
		return nil, err
	}
	rows := []PublicPost{}
	for _, v := range asList(data["list"]) {
		m := asObject(v)
		if post := asObject(m["post"]); post != nil {
			m = post
		}
		p, err := normalizePublicPost(game, m, false)
		if err != nil {
			return nil, err
		}
		rows = append(rows, p)
	}
	return map[string]any{"items": rows, "more": false, "source": "米游社作者137101761", "fetched_at_ms": time.Now().UnixMilli(), "note": "内容为作者的预估或盘点，请核对原文版本与发布时间。"}, nil
}
