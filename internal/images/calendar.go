package images

import (
	"regexp"
	"strings"

	"github.com/RayleaBot/zzz/internal/app"
)

var calendarPicture = regexp.MustCompile(`<img.*?src="(.*?)".*?>`)

// Calendar answers 日历 the way ZZZ-Plugin does: it finds the official
// announcement titled with 日历 and subtitled 活动日历 and shows the first
// picture in it. Without one, upstream says it found none; the announcement
// list in text answers instead.
func Calendar(context app.ImageContext, calendar app.CalendarImage) (app.Image, bool) {
	for _, raw := range calendar.Announcements.Contents {
		ann, _ := raw.(map[string]any)
		if !strings.Contains(app.Text(ann["title"]), "日历") || !strings.Contains(app.Text(ann["subtitle"]), "活动日历") {
			continue
		}
		match := calendarPicture.FindStringSubmatch(app.Text(ann["content"]))
		if match == nil {
			return app.Image{}, false
		}
		resources := &app.ImageResources{Context: context}
		picture := resources.URL("mihoyo", match[1])
		if picture == "" {
			return app.Image{}, false
		}
		return app.Image{Template: "calendar", Data: map[string]any{"picture": picture}, Resources: resources.List}, true
	}
	return app.Image{}, false
}
