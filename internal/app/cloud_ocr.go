package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

type CloudOCR struct {
	Original      *PanelEquipment `json:"original,omitempty"`
	Replacement   *PanelEquipment `json:"replacement,omitempty"`
	Slot          int             `json:"slot"`
	NeedsIdentity bool            `json:"needs_identity"`
	replacement   map[string]any
}

func cloudOCRRequest(game string, input CloudInput) (string, map[string]any, error) {
	maxSlot := 5
	if game == "starrail" {
		maxSlot = 6
	}
	if input.Slot < 1 || input.Slot > maxSlot {
		return "", nil, gameError("input_invalid", "请选择装备部位并提供公开 HTTPS 图片地址，不含账号认证或片段。")
	}
	if err := checkOCRImage(input.ImageURL); err != nil {
		return "", nil, err
	}
	return "ocr/profilechange/" + cloudGame(game), map[string]any{"version": "0.1.0", "image": input.ImageURL, "forge": input.Forge}, nil
}

// checkOCRImage accepts a public HTTPS image address without credentials.
func checkOCRImage(image string) error {
	parsed, err := url.Parse(image)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" || len(image) > 4096 {
		return gameError("input_invalid", "请提供公开 HTTPS 图片地址，不含账号认证或片段。")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return gameError("input_invalid", "请使用可公开访问的图片地址。")
	}
	if ip, err := netip.ParseAddr(host); err == nil && (ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsMulticast()) {
		return gameError("input_invalid", "图片地址不能是本地或私有 IP。")
	}
	for key := range parsed.Query() {
		key = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
		if slices.Contains([]string{"ck", "cookie", "cookietoken", "stoken", "ltoken", "authkey", "loginticket", "ticket"}, key) {
			return gameError("input_invalid", "图片链接不能包含 CK 或登录凭据字段。")
		}
	}
	return nil
}
func cleanOCRGear(value any) (map[string]any, error) {
	data := asObject(value)
	if nested := asObject(data["data"]); nested != nil {
		data = nested
	}
	out := map[string]any{}
	for _, key := range []string{"id", "level", "star", "mainId"} {
		if v, exists := data[key]; exists {
			n, ok := cloudInteger(v, 0, 1000000000)
			if !ok {
				return nil, gameError("cloud_invalid", "OCR 装备数值格式暂不兼容。")
			}
			out[key] = n
		}
	}
	if name := cloudText(data["name"]); name != "" {
		out["name"] = name
	}
	attrs, ok := data["attrIds"].([]any)
	if !ok || len(attrs) > 32 || out["mainId"] == nil || out["level"] == nil {
		return nil, gameError("cloud_invalid", "OCR 未提供完整的装备主副词条。")
	}
	clean := []any{}
	for _, raw := range attrs {
		if obj := asObject(raw); obj != nil {
			next := map[string]any{}
			for _, key := range []string{"id", "count", "step"} {
				n, ok := cloudInteger(obj[key], 0, 100000)
				if !ok {
					return nil, gameError("cloud_invalid", "OCR 词条档位格式暂不兼容。")
				}
				next[key] = n
			}
			clean = append(clean, next)
		} else {
			v := asText(raw)
			if len(v) > 64 || strings.Trim(v, "0123456789,") != "" || v == "" {
				return nil, gameError("cloud_invalid", "OCR 词条编号格式暂不兼容。")
			}
			clean = append(clean, v)
		}
	}
	out["attrIds"] = clean
	return out, nil
}
func cloudOCRResult(game Game, input CloudInput, result map[string]any) (CloudResult, error) {
	list, ok := result["data"].([]any)
	expected := 1
	if input.Forge {
		expected = 2
	}
	if !ok || len(list) != expected {
		return CloudResult{}, gameError("cloud_invalid", "OCR 未返回所选模式的装备数量。")
	}
	after, err := cleanOCRGear(list[len(list)-1])
	if err != nil {
		return CloudResult{}, err
	}
	section, replacement := decodeCloudGear(game, input.Slot, after)
	section.Title = "识别的新装备 · " + section.Title
	ocr := &CloudOCR{Replacement: replacement, Slot: input.Slot, NeedsIdentity: replacement == nil, replacement: after}
	view := View{Title: game.Name + "装备识别", Rows: []Row{}, Note: "来源：ark OCR；请核对截图识别内容。可明确选择本人角色及部位进行试算，结果不写回游戏或官方历史。"}
	if input.Forge {
		before, err := cleanOCRGear(list[0])
		if err != nil {
			return CloudResult{}, err
		}
		originalSection, original := decodeCloudGear(game, input.Slot, before)
		originalSection.Title = "截图原装备 · " + originalSection.Title
		ocr.Original = original
		view.Sections = append(view.Sections, originalSection)
	}
	view.Sections = append(view.Sections, section)
	if replacement == nil {
		view.Note += " 识别数据未能独立解释套装或品质；比较时可明确选择沿用当前部位身份，仍不完整时停止计算。"
	}
	return CloudResult{View: &view, OCR: ocr}, nil
}
func (a *App) compareCloudOCR(ctx context.Context, client AccountsClient, input map[string]any) (map[string]any, error) {
	ref := asText(input["ref"])
	a.Cloud.mu.Lock()
	job := a.Cloud.jobs[ref]
	if job == nil || job.game != a.Game.ID || job.State != "completed" || job.OCR == nil {
		a.Cloud.mu.Unlock()
		return nil, gameError("cloud_missing", "OCR 结果已失效，请重新识别。")
	}
	slot := job.OCR.Slot
	raw, _ := json.Marshal(job.OCR.replacement)
	a.Cloud.mu.Unlock()
	var gear map[string]any
	_ = json.Unmarshal(raw, &gear)
	choice := Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}
	id := asText(input["character_id"])
	if _, err := strconv.Atoi(id); err != nil {
		if entry, ok := a.Catalog.Resolve(id, "character", nil); ok {
			id = entry.ID
		}
	}
	panel, err := a.queryCharacterPanel(ctx, client, choice, id)
	if err != nil {
		return nil, err
	}
	if input["inherit_identity"] == true {
		for _, old := range panel.Equipment {
			if old.Slot != slot {
				continue
			}
			if asText(gear["name"]) == "" {
				gear["name"] = old.Name
			}
			if gear["id"] == nil {
				gear["id"] = old.ID
			}
			if gear["star"] == nil {
				gear["star"] = old.Rarity
			}
		}
	}
	_, after := decodeCloudGear(a.Game, slot, gear)
	if after == nil {
		return nil, gameError("cloud_invalid", "装备名称、品质或词条不完整，不能进行可靠的换装试算。")
	}
	record, err := findBuildCharacter(a.Game.Calc, a.Game.ID, panel)
	if err != nil {
		return nil, err
	}
	profile, err := buildProfile(a.Game.Calc, a.Game.ID, panel, record)
	if err != nil {
		return nil, err
	}
	replacement, err := buildGearSet(a.Game.ID, CharacterPanel{EquipmentKnown: true, Equipment: []PanelEquipment{*after}})
	if err != nil {
		return nil, err
	}
	candidate := slices.Clone(profile.Equipment)
	i := slices.IndexFunc(candidate, func(v BuildGear) bool { return v.Slot == slot })
	if i < 0 {
		candidate = append(candidate, replacement[0])
	} else {
		candidate[i] = replacement[0]
	}
	profile.CandidateEquipment = &candidate
	result, err := referenceBuild(ctx, a.Game.Calc, record, profile)
	if err != nil {
		return nil, err
	}
	applyBuildIdentity(&result, panel, record)
	view := cloudBuildComparisonView(a.Game, result)
	view.Note += " 将 OCR 新装备放入所选角色部位的模拟；截图旧装备不自动视为此角色当前装备。"
	return map[string]any{"build": result, "view": view, "replacement": after}, nil
}

func cloudBuildComparisonView(game Game, result BuildResult) View {
	view := View{Title: result.Character + " · OCR 换装试算", Subtitle: game.Name + " · " + result.Version, Rows: []Row{}, Note: "保持固定参考战斗情境；当前装备与识别装备分别计算，缺失伤害不补零。"}
	if result.Candidate == nil {
		return view
	}
	candidates := map[string]BuildSkillResult{}
	for _, v := range result.Candidate.Results {
		candidates[v.ID] = v
	}
	for _, before := range result.Baseline.Results {
		after, ok := candidates[before.ID]
		if !ok {
			continue
		}
		text := before.Text + " → " + after.Text
		if before.Expected != nil && after.Expected != nil {
			text = fmt.Sprintf("期望 %.1f → %.1f", *before.Expected, *after.Expected)
			if *before.Expected != 0 {
				text += fmt.Sprintf(" · %+.2f%%", (*after.Expected / *before.Expected - 1)*100)
			}
		}
		view.Rows = append(view.Rows, Row{Label: before.Title, Value: text})
	}
	return view
}
