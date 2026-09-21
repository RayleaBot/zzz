package images

import (
	"regexp"
	"strings"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

var calendarPicture = regexp.MustCompile(`<img.*?src="(.*?)".*?>`)

// Calendar answers 日历 the way ZZZ-Plugin does: it finds the official
// announcement titled with 日历 and subtitled 活动日历 and shows the first
// picture in it. Without one, upstream says it found none; the announcement
// list in text answers instead.
func Calendar(context gamekit.ImageContext, calendar gamekit.CalendarImage) (gamekit.Image, bool) {
	for _, raw := range calendar.Announcements.Contents {
		ann, _ := raw.(map[string]any)
		if !strings.Contains(gamekit.Text(ann["title"]), "日历") || !strings.Contains(gamekit.Text(ann["subtitle"]), "活动日历") {
			continue
		}
		match := calendarPicture.FindStringSubmatch(gamekit.Text(ann["content"]))
		if match == nil {
			return gamekit.Image{}, false
		}
		resources := &gamekit.ImageResources{Context: context}
		picture := resources.URL("mihoyo", match[1])
		if picture == "" {
			return gamekit.Image{}, false
		}
		return gamekit.Image{Template: "calendar", Data: map[string]any{"picture": picture}, Resources: resources.List}, true
	}
	return gamekit.Image{}, false
}
