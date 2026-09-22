package app

import (
	"context"
	"encoding/json"
	"regexp"
	"slices"
	"strconv"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

var cloudRankUID = regexp.MustCompile(`[0-9]{9,10}`)

// cloudChatCommand answers ark-plugin's rank words in chat: 角色排名<角色>
// [UID] is the character's rank among ark's uploaded panels, updated first;
// 总排名 [UID] ranks every character kept for the UID; <角色>排名统计 is the
// character's score distribution. Each runs the matching ark query and
// replies with its result.
func (a *App) cloudChatCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	text := strings.Join(args, "")
	q := CloudInput{Consent: true}
	switch command {
	case "cloud-character-rank":
		q.Mode, q.Query, q.Refresh = "rank", "dmg", true
		q.UID = cloudRankUID.FindString(text)
		name := strings.TrimSpace(cloudRankUID.ReplaceAllString(text, ""))
		entry, ok := a.Catalog.Resolve(name, "character", a.aliasMap(event))
		if !ok {
			return event.SendText("命令格式错误，示例：" + a.Game.Prefix + "角色排名雷电将军123456789")
		}
		q.CharacterID = entry.ID
	case "cloud-total-rank":
		q.Mode, q.UID = "self_rank", cloudRankUID.FindString(text)
	case "cloud-rank-stats":
		entry, ok := a.Catalog.Resolve(strings.TrimSpace(text), "character", a.aliasMap(event))
		if !ok {
			return event.SendText("未找到该角色，请使用角色全名或别名。")
		}
		q.Mode, q.CharacterID = "distribution", entry.ID
	}
	if q.Mode != "distribution" {
		owner, err := a.panelOwner(ctx, event, q.UID)
		if err != nil {
			return event.SendText("请先绑定UID")
		}
		q.UID = owner.UID
	}
	if q.Mode == "self_rank" {
		// ark ranks the characters upstream keeps for the UID.
		saved, err := a.Profiles.Read(q.UID)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		for id := range saved.Panels {
			q.CharacterIDs = append(q.CharacterIDs, id)
		}
		slices.Sort(q.CharacterIDs)
		if len(q.CharacterIDs) == 0 {
			return event.SendText(a.panelReply("list_empty", map[string]string{"uid": q.UID}))
		}
	}
	job, err := a.Cloud.Start(a.Game, q)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	job = a.Cloud.Wait(ctx, job)
	if job.View == nil {
		message := "ark 云服务暂未返回结果，请稍后重试。"
		if job.Message != "" {
			message = job.Message
		}
		return event.SendText(message)
	}
	view := *job.View
	if q.Mode == "distribution" && job.Stats != nil && a.rankStatsImage != nil {
		entry, _ := a.Catalog.Get(q.CharacterID)
		if drawn, ok := a.rankStatsImage(a.imageContext(ctx), RankStatsImage{Character: entry.Name, Stats: *job.Stats}); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
}

// RankStatsImage is what 排名统计 draws on, as ark-plugin's graph/stats: the
// character and its damage distribution.
type RankStatsImage struct {
	Character string
	Stats     CloudStats
}

// RankStatsImageBuilder draws 排名统计 with the plugin's template, or
// returns false to keep the scores in text.
type RankStatsImageBuilder func(ImageContext, RankStatsImage) (Image, bool)

// cloudExchangeCommand is ark-plugin's 导出面板数据 and 导入面板数据<UID>:
// export uploads the UID's kept panels that came from the account, as the
// exchange format, for ten minutes; import downloads a UID's upload into the
// user's own exchange archive, the one the management page shows. As
// upstream, only a UID of the user's own account may be exported or
// imported.
func (a *App) cloudExchangeCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	uid := cloudRankUID.FindString(strings.Join(args, ""))
	owner, err := a.panelOwner(ctx, event, uid)
	if err != nil || !owner.Owned {
		return event.SendText("为确保数据安全，目前仅允许绑定CK用户导入/导出自己UID的面板数据，请联系Bot主人导入/导出...")
	}
	if command == "cloud-export" {
		saved, err := a.Profiles.Read(owner.UID)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		avatars := map[string]json.RawMessage{}
		for _, kept := range saved.Panels {
			if kept.Official == nil {
				continue
			}
			if id, raw, err := officialCloudAvatar(ctx, a.Game, kept.panel(), kept.Official); err == nil {
				avatars[id] = raw
			}
		}
		if len(avatars) == 0 {
			return event.SendText("面板数据文件不存在，请先更新面板数据")
		}
		raw, _ := json.Marshal(map[string]any{"uid": owner.UID, "avatars": avatars})
		out, err := a.startCloudJob(ctx, event, CloudInput{Mode: "exchange_upload", UID: owner.UID, Consent: true, Authenticated: true, exchangeData: string(raw)})
		if job, _ := out["job"].(CloudJob); err == nil && job.State == "completed" {
			return event.SendText("导出成功，请在另一个安装此插件的Bot上输入 " + a.Game.Prefix + "导入面板数据" + owner.UID + " ，有效期十分钟~")
		} else if err == nil {
			return event.SendText(cmpOr(job.Message, "ark 云服务暂未返回结果，请稍后重试。"))
		}
		return event.SendText(friendlyError(err))
	}
	out, err := a.startCloudJob(ctx, event, CloudInput{Mode: "exchange_download", UID: owner.UID, Consent: true, Authenticated: true})
	job, _ := out["job"].(CloudJob)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if job.State != "completed" || len(job.exchange) == 0 {
		return event.SendText(cmpOr(job.Message, "ark 云服务暂未返回结果，请稍后重试。"))
	}
	data, err := cloudPlayerObject(string(job.exchange))
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	avatars, err := cleanCloudPlayer(a.Game.ID, owner.UID, data)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	// As upstream writes the download into the UID's player data, the panels
	// are kept where 面板 reads them.
	panels := []CharacterPanel{}
	for _, raw := range avatars {
		panel, err := a.cloudAvatarPanel(ctx, raw)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		panels = append(panels, panel)
	}
	if _, err = a.Profiles.Keep(owner.UID, panels, "share", nil); err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText("导入成功")
}

// cloudAvatarPanel turns an exchange avatar (miao player data, as cleaned by
// cleanCloudAvatar) into a panel, its properties calculated from its parts as
// miao does when it loads saved player data.
func (a *App) cloudAvatarPanel(ctx context.Context, raw json.RawMessage) (CharacterPanel, error) {
	var avatar struct {
		ID      int            `json:"id"`
		Elem    string         `json:"elem"`
		Level   int            `json:"level"`
		Promote *int           `json:"promote"`
		Cons    int            `json:"cons"`
		Talent  map[string]int `json:"talent"`
		Trees   []string       `json:"trees"`
		Weapon  *struct {
			ID      int    `json:"id"`
			Name    string `json:"name"`
			Level   int    `json:"level"`
			Promote *int   `json:"promote"`
			Affix   int    `json:"affix"`
		} `json:"weapon"`
		Artis map[string]map[string]any `json:"artis"`
	}
	if err := json.Unmarshal(raw, &avatar); err != nil || a.Game.Calc == nil {
		return CharacterPanel{}, gameError("cloud_invalid", "面板文件的角色数据无法读取。")
	}
	id := strconv.Itoa(avatar.ID)
	record, err := findReferenceCharacter(a.Game.Calc, a.Game.ID, CharacterPanel{ID: id, Element: avatar.Elem})
	if err != nil {
		return CharacterPanel{}, err
	}
	panel := CharacterPanel{ID: id, Level: avatar.Level, Promote: avatar.Promote, Rank: max(0, min(6, avatar.Cons)), Element: record.Element, Source: "share", RankKnown: true, WeaponKnown: true, EquipmentKnown: true,
		Stats: []PanelStat{}, Equipment: []PanelEquipment{}, Skills: []PanelSkill{}, Ranks: []PanelSkill{}}
	if entry, ok := a.Catalog.Get(id); ok {
		panel.Name = entry.Name
	}
	for slot := 1; slot <= 6; slot++ {
		if gear := avatar.Artis[strconv.Itoa(slot)]; gear != nil {
			if _, piece := decodeCloudGear(a.Game, slot, gear); piece != nil {
				panel.Equipment = append(panel.Equipment, *piece)
			}
		}
	}
	weapon := PanelEquipment{Main: []PanelStat{}, Sub: []PanelStat{}, Complete: true}
	if avatar.Weapon != nil {
		// Genshin Impact player data names the weapon; Star Rail gives its ID.
		for _, item := range a.Game.Calc.Metadata().Weapons {
			if item.Game == cloudGame(a.Game.ID) && (avatar.Weapon.ID != 0 && item.ID == strconv.Itoa(avatar.Weapon.ID) || avatar.Weapon.Name != "" && item.Name == avatar.Weapon.Name) {
				weapon.ID, weapon.Name = item.ID, item.Name
			}
		}
		if entry, ok := a.Catalog.Get(weapon.ID); ok {
			weapon.Rarity = strconv.Itoa(entry.Rarity)
		}
		weapon.Level, weapon.Promote, weapon.Refinement = avatar.Weapon.Level, avatar.Weapon.Promote, max(1, min(5, avatar.Weapon.Affix))
	}
	talents := map[string]int{}
	for key, level := range avatar.Talent {
		talents[key] = level + talentBonus(a.Game.ID, record, key, panel.Rank)
	}
	return a.computedPanel(ctx, record, panel, weapon, talents, avatar.Trees)
}

func cmpOr(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
