package app

import (
	"context"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// ark-plugin reads artifact and relic screenshots sent in chat with its OCR:
// 面板换装 takes the pieces read from the attached screenshots, and
// ark重塑识别 (重投 in Star Rail) finds the reforged piece among the kept
// panels and shows the panel wearing its new substats.

// scannedGear is one piece the OCR read; Slot is 0 when the answer does not
// say which slot it is.
type scannedGear struct {
	Slot int
	Gear map[string]any
}

// messageImages lists the images of the message a command replies to, then
// the command's own, as ark-plugin's collectImageUrls.
func messageImages(ctx context.Context, event *rayleabot.EventContext) []string {
	urls := []string{}
	add := func(kind string, data map[string]any) {
		if url := asText(data["url"]); kind == "image" && url != "" && !slices.Contains(urls, url) {
			urls = append(urls, url)
		}
	}
	for _, segment := range event.Event.Message.Segments {
		if segment.Type != "reply" || asText(segment.Data["id"]) == "" {
			continue
		}
		if quoted, err := event.Actions().MessageGet(ctx, asText(segment.Data["id"])); err == nil {
			list, _ := quoted["message"].([]any)
			for _, item := range list {
				add(asText(asObject(item)["type"]), asObject(asObject(item)["data"]))
			}
		}
	}
	for _, segment := range event.Event.Message.Segments {
		add(segment.Type, segment.Data)
	}
	return urls
}

// scanGear sends a screenshot to ark's OCR; forge asks for the pieces before
// and after a reforge.
func (a *App) scanGear(ctx context.Context, image string, forge bool) ([]scannedGear, error) {
	if err := checkOCRImage(image); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	decoded, err := a.Cloud.request(ctx, "ocr/profilechange/"+cloudGame(a.Game.ID), map[string]any{"version": "0.1.0", "image": image, "forge": forge})
	if err != nil {
		return nil, err
	}
	result := asObject(decoded)
	if result == nil {
		return nil, gameError("cloud_invalid", "云服务对象格式暂不兼容。")
	}
	if err := cloudRetcode(result); err != nil {
		return nil, err
	}
	list, ok := result["data"].([]any)
	if !ok && asObject(result["data"]) != nil {
		list = []any{result["data"]}
	}
	gears := []scannedGear{}
	for _, item := range list {
		gear, err := cleanOCRGear(item)
		if err != nil {
			return nil, err
		}
		slot, _ := strconv.Atoi(strings.TrimPrefix(asText(asObject(item)["type"]), "arti"))
		gears = append(gears, scannedGear{Slot: slot, Gear: gear})
	}
	return gears, nil
}

// scannedPieces reads every attached screenshot for 面板换装, by slot.
func (a *App) scannedPieces(ctx context.Context, images []string) (map[int]PanelEquipment, error) {
	pieces := map[int]PanelEquipment{}
	for _, image := range images[:min(len(images), 6)] {
		gears, err := a.scanGear(ctx, image, false)
		if err != nil {
			return nil, err
		}
		for _, gear := range gears {
			if piece := readGear(a.Game, gear); piece != nil {
				pieces[piece.Slot] = *piece
			}
		}
	}
	if len(pieces) == 0 {
		return nil, gameError("cloud_invalid", "未能从截图识别出"+a.gearWord()+"。")
	}
	return pieces, nil
}

// readGear explains a piece the OCR read, in the slot its answer names or,
// without one, the slot its name or ID belongs to.
func readGear(game Game, gear scannedGear) *PanelEquipment {
	for slot := 1; slot <= 6; slot++ {
		if gear.Slot == 0 || gear.Slot == slot {
			if _, piece := decodeCloudGear(game, slot, gear.Gear); piece != nil {
				return piece
			}
		}
	}
	return nil
}

func (a *App) gearWord() string {
	if a.Game.ID == "starrail" {
		return "遗器"
	}
	return "圣遗物"
}

// sameGear tells whether a kept piece is the one read from a screenshot:
// the same substats with the same values, as far as the game shows them.
// Like ark-plugin, Star Rail's speed may be a few rolls apart.
func sameGear(game string, kept, read PanelEquipment) bool {
	if len(kept.Sub) != len(read.Sub) {
		return false
	}
	values := map[string]float64{}
	for _, stat := range read.Sub {
		value, _ := buildStatNumber(stat.Value)
		values[mapCloudStat(stat.Key)] = value
	}
	for _, stat := range kept.Sub {
		value, ok := buildStatNumber(stat.Value)
		want, found := values[mapCloudStat(stat.Key)]
		if !ok || !found {
			return false
		}
		tolerance := 1.01
		if strings.HasSuffix(strings.TrimSpace(stat.Value), "%") {
			tolerance = 0.11
		}
		if game == "starrail" && stat.Key == "speed" {
			tolerance = 2
		}
		if math.Abs(value-want) > tolerance {
			return false
		}
	}
	return true
}

func mainKey(piece PanelEquipment) string {
	if len(piece.Main) == 0 {
		return ""
	}
	return mapCloudStat(piece.Main[0].Key)
}

// reforgeCommand is ark-plugin's ark重塑识别: it reads the screenshot of a
// reforge, finds the piece before it among the UID's kept panels, and
// answers with that character's panel wearing the piece after it.
func (a *App) reforgeCommand(ctx context.Context, event *rayleabot.EventContext) error {
	if a.Game.Calc == nil {
		return event.Result(map[string]any{"handled": false})
	}
	owner, err := a.panelOwner(ctx, event, "")
	if err != nil || owner.UID == "" {
		return event.SendText("请先绑定UID")
	}
	word := map[bool]string{true: "重投", false: "重塑"}[a.Game.ID == "starrail"]
	images := messageImages(ctx, event)
	if len(images) == 0 {
		return event.SendText("请发送" + a.gearWord() + word + "图片")
	}
	gears, err := a.scanGear(ctx, images[0], true)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if len(gears) != 2 {
		return event.SendText("未能从截图识别出" + a.gearWord() + "的前后两件。")
	}
	before, after := gears[0], gears[1]
	saved, err := a.Profiles.Read(owner.UID)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if len(saved.Panels) == 0 {
		return event.SendText("未找到UID:" + owner.UID + "的" + a.Game.Name + "本地数据，请确保面板中包含此" + a.gearWord())
	}
	original := readGear(a.Game, before)
	if original == nil {
		return event.SendText("未能解释截图中的原" + a.gearWord() + "。")
	}
	// A match with the same main stat first, as ark-plugin prefers.
	ids := []string{}
	for id := range saved.Panels {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	wearer, slot, best := "", 0, false
	for _, id := range ids {
		for _, piece := range saved.Panels[id].panel().Equipment {
			if piece.Slot != original.Slot || !sameGear(a.Game.ID, piece, *original) {
				continue
			}
			if wearer == "" || !best && mainKey(piece) == mainKey(*original) {
				wearer, slot, best = id, piece.Slot, mainKey(piece) == mainKey(*original)
			}
		}
	}
	if wearer == "" {
		return event.SendText("未在本地数据中找到匹配的圣遗物/遗器")
	}
	// The reforged piece keeps what the answer leaves out of the original.
	for _, key := range []string{"name", "id", "star"} {
		if after.Gear[key] == nil {
			after.Gear[key] = before.Gear[key]
		}
	}
	after.Slot = slot
	reforged := readGear(a.Game, after)
	if reforged == nil {
		return event.SendText("未能解释截图中" + word + "后的" + a.gearWord() + "。")
	}
	panel, err := a.changedPanel(ctx, owner.UID, PanelChange{CharacterID: wearer, Pieces: map[int]changeSource{}, Scanned: map[int]PanelEquipment{slot: *reforged}})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	_ = event.SendText("找到匹配" + a.gearWord() + "，正在生成" + word + "面板...")
	return a.sendView(ctx, event, a.fullPanelView(ctx, event, panel, owner.UID, false, "ark"+word+"识别"))
}
