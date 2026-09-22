package showcase_test

import (
	"errors"
	"testing"

	"github.com/RayleaBot/plugin-zzz/internal/app"
	"github.com/RayleaBot/plugin-zzz/internal/assets"
	"github.com/RayleaBot/plugin-zzz/internal/showcase"
)

// The fixture follows an Enka answer's shape with one drive disc; its values
// are made up.
const enkaFixture = `{"ttl":60,"PlayerInfo":{"SocialDetail":{"ProfileDetail":{"Nickname":"绳匠","Level":60}},"ShowcaseDetail":{"AvatarList":[
{"Id":1011,"Level":60,"TalentLevel":2,"PromotionLevel":6,"CoreSkillEnhancement":0,"SkinId":0,"SkillLevelList":[{"Index":0,"Level":10},{"Index":5,"Level":7}],
"Weapon":{"Id":12001,"Level":60,"UpgradeLevel":1,"BreakLevel":5},
"EquippedList":[{"Slot":1,"Equipment":{"Id":31041,"Level":15,"MainPropertyList":[{"PropertyId":11103,"PropertyValue":550,"PropertyLevel":1}],
"RandomPropertyList":[{"PropertyId":12103,"PropertyValue":19,"PropertyLevel":2},{"PropertyId":20103,"PropertyValue":240,"PropertyLevel":1}]}}]},
{"Id":1,"Level":1,"TalentLevel":0,"PromotionLevel":1,"SkillLevelList":[],"EquippedList":[]}]}}}`

func TestParseRunsZZZPluginsEnkaFormat(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	profile, err := showcase.Parse(t.Context(), application.Game, application.Catalog, []byte(enkaFixture))
	if err != nil || profile.Nickname != "绳匠" {
		t.Fatal(profile, err)
	}
	// Enka2Mys skips the agent its data does not know.
	if len(profile.Panels) != 1 {
		t.Fatalf("panels = %d", len(profile.Panels))
	}
	panel := profile.Panels[0]
	if panel.ID != "1011" || panel.Level != 60 || panel.Rank != 2 || panel.Weapon == nil || panel.Weapon.Refinement != 1 || panel.Official == nil {
		t.Fatalf("panel = %+v", panel)
	}
	disc := panel.Equipment[0]
	if disc.Slot != 1 || disc.SetName == "" || disc.Main[0].Value != "2200" || disc.Sub[0].Value != "38" || disc.Sub[1].Value != "2.4%" {
		t.Fatalf("disc = %+v", disc)
	}
	// Mindscape two leaves the skills at their levels; the core skill keeps
	// its own.
	levels := map[int]int{}
	for _, skill := range panel.Skills {
		levels[skill.SkillType] = skill.Level
	}
	if levels[0] != 10 || levels[5] != 7 {
		t.Fatalf("skills = %v", levels)
	}
	if _, err = showcase.Parse(t.Context(), application.Game, application.Catalog, []byte(`{"PlayerInfo":{"ShowcaseDetail":{"AvatarList":[]}}}`)); !errors.Is(err, app.ErrShowcaseEmpty) {
		t.Fatal(err)
	}
}
