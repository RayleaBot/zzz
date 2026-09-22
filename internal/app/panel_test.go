package app

import (
	"strings"
	"testing"
)

func TestPanelsUseMetadataAndKeepOfficialValues(t *testing.T) {
	for _, tc := range []struct {
		game, data string
		want       []string
	}{
		{"zzz", `{"avatar_list":[{"id":1001,"full_name_mi18n":"测试角色","level":60,"rank":1,"element_type":203,"properties":[{"property_id":20103,"property_name":"暴击率","final":"80%","base":"5%"}],"equip":[{"equipment_type":4,"name":"测试盘","level":15,"rarity":"S","main_properties":[{"property_id":20103,"property_name":"暴击率","base":"24%"}],"properties":[{"property_id":12102,"property_name":"攻击力","base":"9%"}]}],"weapon":{"name":"测试音擎","level":60,"star":3,"main_properties":[]},"equip_plan_info":{"equip_rating_score":123,"equip_rating":"S"}}]}`, []string{"暴击率：80%", "测试音擎", "4号位 · 测试盘", "官方配装评分：123 · S"}},
	} {
		t.Run(tc.game, func(t *testing.T) {
			panels := NormalizePanels(QueryResult{Data: decoded(t, tc.data)}, Catalog{})
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
	panels := NormalizePanels(QueryResult{Data: decoded(t, `{"avatar_list":[{"id":1001,"name_mi18n":"空面板","properties":[{"property_id":11101}]}]}`)}, Catalog{})
	if len(panels[0].Stats) != 0 || len(panels[0].Equipment) != 0 || panels[0].Weapon != nil {
		t.Fatal("missing panel data became zero-valued stats or equipment")
	}
}
