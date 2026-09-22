package app

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// DailyMaterialImage is 今日素材: a UID's kept characters, for the talent
// books and weapon materials of miao's material group Week (1 for Monday and
// Thursday, 2 and 3 for the days after).
type DailyMaterialImage struct {
	UID    string
	Week   int
	Panels []SavedPanel
}

type DailyMaterialImageBuilder func(ImageContext, DailyMaterialImage) (Image, bool)

var dailyWeekday = regexp.MustCompile(`周([1-7]|一|二|三|四|五|六|日)`)

// dailyMaterialDay is the weekday miao's TodayMaterial reads from the command
// word, Monday 0 to Sunday 6: the day turns at four in the morning, 明天 and
// 明日 are the next day and 周X names one.
func dailyMaterialDay(word string, now time.Time) int {
	now = now.In(time.FixedZone("UTC+8", 8*3600))
	if now.Hour() < 4 {
		now = now.AddDate(0, 0, -1)
	}
	if strings.Contains(word, "明天") || strings.Contains(word, "明日") {
		now = now.AddDate(0, 0, 1)
	}
	day := (int(now.Weekday()) + 6) % 7
	if match := dailyWeekday.FindStringSubmatch(word); match != nil {
		if n, err := strconv.Atoi(match[1]); err == nil {
			day = n - 1
		} else {
			day = strings.Index("一二三四五六日", match[1]) / len("一")
		}
	}
	return day
}

// dailyMaterial is miao's 今日素材. As miao refreshes a player's characters
// from the account when they are over two hours old, the user's own UID is
// read again from the account first.
func (a *App) dailyMaterial(ctx context.Context, event *rayleabot.EventContext) error {
	day := dailyMaterialDay(event.Event.Command(), time.Now())
	if day == 6 {
		return event.SendText("今天周日，全部素材都可以刷哦~")
	}
	owner, err := a.panelOwner(ctx, event, "")
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	saved, err := a.Profiles.Read(owner.UID)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if owner.Owned && (saved.Service != "米游社" || time.Since(time.UnixMilli(saved.RefreshedAtMS)) > 2*time.Hour) {
		if panels, err := a.accountPanels(ctx, a.accountClient(event), owner.Choice); err == nil && len(panels) > 0 {
			if kept, err := a.Profiles.Keep(owner.UID, panels, "米游社", &ShowcaseProfile{Nickname: owner.Role.Nickname, Level: owner.Role.Level}); err == nil {
				saved = kept
			}
		}
	}
	if len(saved.Panels) == 0 {
		return event.SendText("查询失败，暂未获得" + a.Game.Prefix + owner.UID + "角色数据，请绑定CK或 " + a.Game.Prefix + "更新面板")
	}
	panels := saved.Sorted(a.Catalog, nil)
	week := day%3 + 1
	view := View{Title: a.Game.Name + "今日素材", Subtitle: "UID " + owner.UID + " · " + []string{"周一/周四", "周二/周五", "周三/周六"}[week-1], Note: "图片按城市列出当天可升天赋的角色与可突破的武器。"}
	if a.dailyMaterialImage != nil {
		if drawn, ok := a.dailyMaterialImage(a.imageContext(ctx), DailyMaterialImage{UID: owner.UID, Week: week, Panels: panels}); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
}
