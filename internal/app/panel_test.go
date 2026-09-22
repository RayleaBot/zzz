package app

import (
	"context"
	"strings"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func TestPanelsUseMetadataAndKeepOfficialValues(t *testing.T) {
	for _, tc := range []struct {
		game, data string
		want       []string
	}{
		{"genshin", `{"property_map":{"2001":{"name":"攻击力"},"20":{"name":"暴击率"}},"list":[{"base":{"id":1001,"name":"测试角色","level":90,"actived_constellation_num":2},"base_properties":[{"property_type":2001,"final":"2,345","base":"900","add":"1,445"}],"weapon":{"id":2001,"name":"测试武器","level":90,"affix_level":3},"relics":[{"id":1,"pos":1,"name":"测试之花","level":20,"rarity":5,"main_property":{"property_type":2,"value":"4780"},"sub_property_list":[{"property_type":20,"value":"10.5%"}]}],"skills":[{"name":"普攻","level":10,"is_unlock":true,"desc":"<b>实际说明</b>"}],"constellations":[{"name":"命座一","is_actived":true,"effect":"效果"}]}]}`, []string{"攻击力：2,345", "测试武器", "暴击率：10.5%", "普攻：等级 10", "命座一：已解锁"}},
		{"starrail", `{"property_info":{"5":{"name":"暴击率"}},"avatar_list":[{"id":1001,"name":"测试角色","level":80,"rank":1,"properties":[{"property_type":5,"final":"76.4%"}],"equip":{"name":"测试光锥","level":80,"rank":2},"ornaments":[{"id":1,"pos":5,"name":"测试球","level":15,"rarity":5,"main_property":{"property_type":12,"value":"38.8%"},"properties":[]}],"skills":[{"point_id":"12345","level":6,"is_activated":true,"skill_stages":[{"name":"战技","desc":"说明"}]}]}]}`, []string{"暴击率：76.4%", "测试光锥", "5号位 · 测试球", "战技：等级 6"}},
		{"zzz", `{"avatar_list":[{"id":1001,"full_name_mi18n":"测试角色","level":60,"rank":1,"element_type":203,"properties":[{"property_id":20103,"property_name":"暴击率","final":"80%","base":"5%"}],"equip":[{"equipment_type":4,"name":"测试盘","level":15,"rarity":"S","main_properties":[{"property_id":20103,"property_name":"暴击率","base":"24%"}],"properties":[{"property_id":12102,"property_name":"攻击力","base":"9%"}]}],"weapon":{"name":"测试音擎","level":60,"star":3,"main_properties":[]},"equip_plan_info":{"equip_rating_score":123,"equip_rating":"S"}}]}`, []string{"暴击率：80%", "测试音擎", "4号位 · 测试盘", "官方配装评分：123 · S"}},
	} {
		t.Run(tc.game, func(t *testing.T) {
			panels := NormalizePanels(tc.game, QueryResult{Data: decoded(t, tc.data)}, Catalog{})
			if len(panels) != 1 || len(panels[0].Stats) == 0 {
				t.Fatal("panel fields lost")
			}
			text := PanelView(Game{ID: tc.game}, panels, "100000001").Text()
			for _, want := range tc.want {
				if !strings.Contains(text, want) {
					t.Fatalf("missing %q", want)
				}
			}
			if len(panels[0].Skills) > 0 && strings.Contains(panels[0].Skills[0].Description, "<b>") {
				t.Fatal("raw markup retained")
			}
		})
	}
}
func TestMissingPropertiesAreNotInvented(t *testing.T) {
	panels := NormalizePanels("genshin", QueryResult{Data: decoded(t, `{"list":[{"base":{"id":1001,"name":"空面板"},"base_properties":[{"property_type":2001}]}]}`)}, Catalog{})
	if len(panels[0].Stats) != 0 || len(panels[0].Equipment) != 0 || panels[0].Weapon != nil {
		t.Fatal("missing panel data became zero-valued stats or equipment")
	}
	if queryOperationName("starrail.character", nil) != "starrail.characters" {
		t.Fatal("empty starrail selector fetched all detailed panels")
	}
}
func scoringPanel(slot4 string) CharacterPanel {
	panel := CharacterPanel{ID: "10000025", Element: "hydro", Rank: 6, RankKnown: true, EquipmentKnown: true, Weapon: &PanelEquipment{Name: "祭礼剑", Refinement: 5}}
	for id, value := range map[string]string{"2000": "18000", "2001": "2100", "2002": "800", "20": "70.0%", "22": "150.0%", "23": "180.0%", "28": "40"} {
		panel.Stats = append(panel.Stats, PanelStat{ID: id, Value: value})
	}
	for slot, main := range map[int][2]string{1: {"hpPlus", "4780"}, 2: {"atkPlus", "311"}, 3: {"atk", "46.6%"}, 4: {slot4, "46.6%"}, 5: {"cpct", "31.1%"}} {
		panel.Equipment = append(panel.Equipment, PanelEquipment{Slot: slot, SetName: "绝缘之旗印", Complete: true, Main: []PanelStat{{Key: main[0], Value: main[1]}}, Sub: []PanelStat{{Key: "cpct", Value: "10.5%"}, {Key: "cdmg", Value: "21.0%"}, {Key: "atk", Value: "5.8%"}, {Key: "recharge", Value: "6.5%"}}})
	}
	return panel
}

func TestMiaoScoringRunsThePinnedRules(t *testing.T) {
	a := pluginApp(t, "genshin")
	correct, err := a.scorePanel(context.Background(), scoringPanel("hydro"))
	if err != nil || correct.ScoredEquipment != 5 || correct.TotalScore == nil {
		t.Fatalf("complete panel was not scored: %v", err)
	}
	// 行秋's upstream artis.js switches to the vaporize weights above 120 mastery.
	if correct.ScoreRule != "行秋-通用" {
		t.Fatalf("default rule not selected: %q", correct.ScoreRule)
	}
	vaporize := scoringPanel("hydro")
	for i := range vaporize.Stats {
		if vaporize.Stats[i].ID == "28" {
			vaporize.Stats[i].Value = "200"
		}
	}
	if scored, err := a.scorePanel(context.Background(), vaporize); err != nil || scored.ScoreRule != "行秋-蒸发" {
		t.Fatalf("character rule not applied: %q %v", scored.ScoreRule, err)
	}
	wrong, err := a.scorePanel(context.Background(), scoringPanel("pyro"))
	if err != nil || *wrong.TotalScore >= *correct.TotalScore {
		t.Fatal("off-element goblet did not take the reference main-stat penalty")
	}
	partial := scoringPanel("hydro")
	partial.Equipment[0].Sub[0].Value = "unknown"
	scored, err := a.scorePanel(context.Background(), partial)
	if err != nil || scored.ScoredEquipment != 4 || scored.Equipment[0].Score != nil || !strings.Contains(scored.ScoreNote, "部分装备") {
		t.Fatal("unreadable piece was scored or hid the others")
	}
	unknown := scoringPanel("hydro")
	unknown.RankKnown = false
	if _, err := a.scorePanel(context.Background(), unknown); err == nil {
		t.Fatal("rules that read the constellation ran without it")
	}
}

func TestFullPanelViewKeepsTheScoreWhenDamageIsUnavailable(t *testing.T) {
	a := pluginApp(t, "genshin")
	// The synthetic panel has no skills, so the damage rule cannot run.
	view := a.fullPanelView(context.Background(), &rayleabot.EventContext{}, scoringPanel("hydro"), "100000001", true, "")
	text := view.Text()
	if !strings.Contains(text, "装备评分合计") || !strings.Contains(text, "伤害：") {
		t.Fatalf("panel reply lost the score or hid why damage is missing:\n%s", text)
	}
}
