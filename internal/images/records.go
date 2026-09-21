package images

import (
	"fmt"
	"time"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

// recordResources collects a record image's resources: ZZZ-Plugin art and the
// official images the response links to, cached on demand.
type recordResources struct{ gamekit.ImageResources }

func newRecordResources(context gamekit.ImageContext, tables ...[][2]string) *recordResources {
	resources := &recordResources{gamekit.ImageResources{Context: context}}
	for _, table := range tables {
		for _, item := range table {
			resources.Artwork(item[0], "zzz-plugin", item[1])
		}
	}
	return resources
}

func (r *recordResources) official(url any) string { return r.URL("mihoyo", url) }

// team is a record's three character slots and its Bangboo the way ZZZ-Plugin's
// record pages list them; a missing character leaves a blank slot.
func (r *recordResources) team(item map[string]any) map[string]any {
	avatars, _ := item["avatar_list"].([]any)
	slots := []any{}
	for index := range 3 {
		if index >= len(avatars) {
			slots = append(slots, nil)
			continue
		}
		avatar, _ := avatars[index].(map[string]any)
		slots = append(slots, map[string]any{"rank": gamekit.Int(avatar["rank"]), "rarity": rarity(avatar["rarity"]), "icon": r.official(avatar["role_square_url"])})
	}
	var buddy any
	if raw, _ := item["buddy"].(map[string]any); raw != nil && gamekit.Int(raw["id"]) != 0 {
		buddy = map[string]any{"rarity": rarity(raw["rarity"]), "icon": r.official(raw["bangboo_rectangle_url"])}
	}
	return map[string]any{"slots": slots, "buddy": buddy}
}

// rarity is a character's or Bangboo's rank letter; ZZZ-Plugin reads a missing
// one as A.
func rarity(value any) string {
	if text := gamekit.Text(value); text != "" {
		return text
	}
	return "A"
}

// playerCard is what ZZZ-Plugin's player info header shows.
func playerCard(role gamekit.Role) map[string]any {
	region := regionNames[role.Region]
	if region == "" {
		region = role.Region
	}
	return map[string]any{"nickname": role.Nickname, "level": role.Level, "region": region, "uid": role.UID}
}

// rankBackground picks ZZZ-Plugin's rank badge for a top percentage given in
// hundredths.
func rankBackground(percent int) int {
	value := float64(percent) / 100
	switch {
	case value < 1:
		return 1
	case value < 5:
		return 2
	case value < 10:
		return 3
	case value < 50:
		return 4
	}
	return 5
}

// rankText prints a top percentage given in hundredths.
func rankText(percent int) string { return fmt.Sprintf("%.2f%%", float64(percent)/100) }

// recordTime formats the official {year, month, day, hour, minute, second}
// time; "" when missing.
func recordTime(value any, layout string) string {
	fields, _ := value.(map[string]any)
	year := gamekit.Int(fields["year"])
	if year == 0 {
		return ""
	}
	return time.Date(year, time.Month(gamekit.Int(fields["month"])), gamekit.Int(fields["day"]),
		gamekit.Int(fields["hour"]), gamekit.Int(fields["minute"]), gamekit.Int(fields["second"]), 0, time.UTC).Format(layout)
}

// rankCommand is the command ZZZ-Plugin's closing note names for joining the
// group ranking; here it is this plugin's challenge submission for the mode.
func rankCommand(context gamekit.ImageContext, mode string) string {
	return context.Game.Prefix + "提交挑战 " + mode
}
