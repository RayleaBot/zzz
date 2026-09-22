package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var cloudVersionPattern = regexp.MustCompile(`^[0-9]{1,2}[._][0-9]{1,2}$`)

func cloudIDs(values []string, maxCount int, uid bool) bool {
	if len(values) < 1 || len(values) > maxCount {
		return false
	}
	seen := map[string]bool{}
	for _, v := range values {
		if seen[v] {
			return false
		}
		seen[v] = true
		if uid {
			if !uidPattern.MatchString(v) {
				return false
			}
		} else {
			n, e := strconv.Atoi(v)
			if e != nil || n < 1 || n > 1000000000 {
				return false
			}
		}
	}
	return true
}
func extendedCloudRequest(game string, input CloudInput) (string, map[string]any, bool, error) {
	body := map[string]any{"version": "0.1.0"}
	invalid := func() (string, map[string]any, bool, error) {
		return "", nil, true, gameError("input_invalid", "云查询需要有效的 UID、角色列表或版本，且列表不能重复。")
	}
	switch input.Mode {
	case "panel_refresh":
		if !uidPattern.MatchString(input.UID) {
			return invalid()
		}
		body["uid"] = input.UID
		body["type"] = cloudGame(game)
		return "panel/refresh", body, true, nil
	case "self_rank":
		if !uidPattern.MatchString(input.UID) || !cloudIDs(input.CharacterIDs, 250, false) {
			return invalid()
		}
		body["uid"] = input.UID
		body["ids"] = input.CharacterIDs
		body["type"] = cloudGame(game)
		return "rank/self", body, true, nil
	case "group_rank":
		if !cloudIDs(input.UIDs, 50, true) || !cloudIDs([]string{input.CharacterID}, 1, false) || !slices.Contains([]string{"", "dmg", "mark"}, input.Query) {
			return invalid()
		}
		body["uids"] = input.UIDs
		body["id"], _ = strconv.Atoi(input.CharacterID)
		body["update"] = 2
		body["query"] = input.Query
		if input.Query == "" {
			body["query"] = "dmg"
		}
		body["data"] = nil
		return "rank/group", body, true, nil
	case "stygian":
		if game != "genshin" || !cloudIDs(input.UIDs, 50, true) {
			return invalid()
		}
		body["uid"] = input.UIDs
		return "rank/stygian", body, true, nil
	case "akasha_stygian":
		if game != "genshin" || input.Authenticated || !cloudIDs(input.UIDs, 50, true) || !cloudVersionPattern.MatchString(input.Version) {
			return invalid()
		}
		return "akasha/stygian", body, true, nil
	}
	return "", nil, false, nil
}
func rankArrayResult(game Game, input CloudInput, list []any) (CloudResult, error) {
	if len(list) < 1 || len(list) > 2 {
		return CloudResult{}, gameError("cloud_invalid", "云排名分组数量暂不兼容。")
	}
	v := View{Title: game.Name + "云排名", Subtitle: "UID " + input.UID, Rows: []Row{}, Note: "来源：ark.ivny.cn。伤害与装备的名次、百分位和分数分别展示。"}
	if len(input.localPanel) > 0 {
		v.Subtitle = "导入面板 · 角色 " + input.CharacterID
		v.Note = "来源：ark.ivny.cn；根据明确上传的本地面板分别计算伤害与装备名次，不代表本人官方实时成绩。"
	}
	for i, raw := range list {
		item := asObject(raw)
		if code := asText(item["retcode"]); code != "100" && code != "0" {
			return CloudResult{}, gameError("cloud_rejected", "云服务没有提供完整排名结果。")
		}
		title := "伤害排名"
		if input.Query == "mark" || i == 1 {
			title = "装备评分排名"
		}
		v.Sections = append(v.Sections, Section{Title: title, Rows: cloudNumericRows(item, "rank", "服务排名", "percent", "百分位（%）", "score", "评分", "sum", "收录总量")})
	}
	return CloudResult{View: &v}, nil
}
func extendedCloudResult(game Game, input CloudInput, result map[string]any) (CloudResult, bool, error) {
	if input.Mode == "panel_refresh" {
		view := View{Title: game.Name + "云面板更新请求已接受", Subtitle: "UID " + input.UID, Rows: []Row{}, Note: "来源：ark；可稍后重新查询云面板。"}
		return CloudResult{View: &view}, true, nil
	}
	if !slices.Contains([]string{"self_rank", "group_rank", "stygian"}, input.Mode) {
		return CloudResult{}, false, nil
	}
	list := asList(result["rank"])
	if input.Mode == "stygian" {
		list = asList(result["data"])
	}
	ids := input.UIDs
	if input.Mode == "self_rank" {
		ids = input.CharacterIDs
	}
	if len(list) != len(ids) {
		return CloudResult{}, true, gameError("cloud_invalid", "云结果与请求列表无法一一对应。")
	}
	title := map[string]string{"self_rank": "多角色全服排名", "group_rank": "多人角色云排名", "stygian": "幽境云排名"}[input.Mode]
	v := View{Title: game.Name + title, Rows: []Row{}, Note: "来源：ark.ivny.cn；只展示请求列表的服务收录数据，不合并不同赛季或算法，不使用米游社 CK。"}
	for i, raw := range list {
		item := asObject(raw)
		label := "UID " + ids[i]
		if input.Mode == "self_rank" {
			label = "角色 " + ids[i]
		}
		rows := cloudNumericRows(item, "rank", "服务排名", "percent", "百分位（%）", "score", "评分", "sum", "收录总量")
		if input.Mode == "stygian" {
			rows = append(rows, cloudNumericRows(item, "hard", "难度（服务记录）", "time", "用时（秒）")...)
		}
		section := Section{Title: label, Rows: rows}
		if len(rows) == 0 {
			section.Text = "服务未提供此项数据。"
		}
		v.Sections = append(v.Sections, section)
	}
	return CloudResult{View: &v}, true, nil
}
func (c *CloudClient) akashaStygian(ctx context.Context, game Game, input CloudInput) (CloudResult, error) {
	if _, _, _, err := extendedCloudRequest(game.ID, input); err != nil {
		return CloudResult{}, err
	}
	filter := ""
	for _, uid := range input.UIDs {
		filter += "[uid]" + uid
	}
	q := url.Values{"sort": {"stygianScore"}, "order": {"-1"}, "size": {"50"}, "page": {"1"}, "uids": {filter}, "p": {""}, "fromId": {""}, "li": {""}, "uid": {""}, "version": {strings.ReplaceAll(input.Version, ".", "_")}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://akasha.cv/api/leaderboards/stygian?"+q.Encode(), nil)
	if err != nil {
		return CloudResult{}, err
	}
	req.Header.Set("User-Agent", "RayleaBot-game-plugin/0.1.0")
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 25 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	response, err := client.Do(req)
	if err != nil {
		return CloudResult{}, gameError("cloud_unavailable", "Akasha 服务暂不可用。")
	}
	defer response.Body.Close()
	if response.StatusCode == 429 {
		return CloudResult{}, gameError("cloud_rate_limited", "Akasha 请求过于频繁，请稍后再试。")
	}
	if response.StatusCode != 200 {
		return CloudResult{}, gameError("cloud_rejected", "Akasha 未提供所选版本的结果。")
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024+1))
	if err != nil || len(raw) > 2*1024*1024 {
		return CloudResult{}, gameError("cloud_invalid", "Akasha 响应过大或无法读取。")
	}
	var result map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&result) != nil || decoder.Decode(new(any)) != io.EOF {
		return CloudResult{}, gameError("cloud_invalid", "Akasha 数据格式暂不兼容。")
	}
	list, ok := result["data"].([]any)
	if !ok || len(list) > 50 {
		return CloudResult{}, gameError("cloud_invalid", "Akasha 排名列表格式暂不兼容。")
	}
	view := View{Title: "原神幽境 · Akasha", Subtitle: "版本 " + input.Version, Rows: []Row{}, Note: "来源：akasha.cv；仅所选版本和 UID 的收录结果，与 ark 排名分开展示。未发送 ark 令牌、米游社 CK 或 QQ。"}
	seen := map[string]bool{}
	for _, row := range list {
		v := asObject(row)
		uid := asText(v["uid"])
		if !slices.Contains(input.UIDs, uid) || seen[uid] {
			continue
		}
		seen[uid] = true
		nickname := cloudText(fieldAt(v, "playerInfo.nickname"))
		view.Sections = append(view.Sections, Section{Title: nickname + " · UID " + uid, Rows: cloudNumericRows(v, "index", "服务排名", "stygianIndex", "难度（服务记录）", "stygianSeconds", "用时（秒）", "stygianScore", "服务评分")})
	}
	if len(view.Sections) == 0 {
		view.Note += " 当前没有匹配的收录记录。"
	}
	return CloudResult{View: &view}, nil
}
