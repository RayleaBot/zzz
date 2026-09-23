package app

import (
	"context"
	"strconv"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// calendarCommand answers 日历 with the plugin's calendar page, or the
// announced activities and their times as text.
func (a *App) calendarCommand(ctx context.Context, event *rayleabot.EventContext) error {
	announcements, err := a.Content.announcements(ctx)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	data, err := calendarItems(announcements)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	view := View{Title: a.Game.Name + "公开资料", Note: "发送“" + a.Game.Prefix + "公告”“" + a.Game.Prefix + "活动”查看正文。"}
	items, _ := data["items"].([]PublicActivity)
	for _, v := range items[:min(20, len(items))] {
		text := "时间未确定"
		if v.End != "" {
			text = v.Start + " 至 " + v.End
			if v.Start == "" {
				text = "开始时间未确定；截止 " + v.End
			}
		}
		view.Rows = append(view.Rows, Row{v.Title, text})
	}
	view.Note += " 本次返回 " + strconv.Itoa(len(items)) + " 篇游戏公告。"
	if a.calendarImage != nil {
		if drawn, ok := a.calendarImage(a.imageContext(ctx), CalendarImage{Word: event.Event.Command(), Announcements: announcements}); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
}
