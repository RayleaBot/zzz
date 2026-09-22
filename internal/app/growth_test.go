package app

import "testing"

func TestGrowthParsesOfficialCalculatorShapes(t *testing.T) {
	for _, tc := range []struct {
		game           string
		detail         string
		expectedSkills int
	}{{"genshin", `{"avatar":{"name":"测试","level_current":50,"max_level":90},"weapon":{"id":123,"name":"测试武器","level_current":40,"max_level":90},"skill_list":[{"group_id":101,"name":"普攻","level_current":3,"max_level":10},{"group_id":102,"name":"被动","level_current":1,"max_level":1}]}`, 1}, {"starrail", `{"avatar":{"item_name":"测试","cur_level":50,"max_level":80},"equipment":{"item_id":"123","item_name":"测试光锥","cur_level":40,"max_level":80},"skills":[{"point_id":"101","cur_level":3,"max_level":10},{"point_id":"102","cur_level":0,"max_level":1}]}`, 2}} {
		p, err := prepareGrowth(tc.game, "1001", decoded(t, tc.detail))
		if err != nil || p.Current != 50 || p.Target != 50 || p.WeaponCurrent != 40 || len(p.Skills) != tc.expectedSkills {
			t.Fatal(p, err)
		}
	}
	if _, err := prepareGrowth("genshin", "1001", decoded(t, `{"avatar":{"name":"test"}}`)); err == nil {
		t.Fatal("missing current level invented")
	}
}
func TestGrowthMaterialAggregationDoesNotDoubleCountItemBreakdown(t *testing.T) {
	data := decoded(t, `{"overall_material_consume":{"avatar_consume":[{"consume":[{"id":1,"name":"摩拉","num":100}]}],"weapon_consume":[{"consume":[{"id":1,"name":"摩拉","num":200}]}]},"items":[{"avatar_consume":[{"id":1,"name":"摩拉","num":100}]}]}`)
	v := growthView(Game{ID: "genshin"}, GrowthPlan{}, QueryResult{Data: data})
	if len(v.Rows) != 1 || v.Rows[0].Value != "300" {
		t.Fatal(v)
	}
	data = decoded(t, `{"avatar_consume":[{"item_id":"1","item_name":"信用点","num":100}],"skill_consume":[{"item_id":"1","item_name":"信用点","num":20}],"coin_id":"1"}`)
	v = growthView(Game{ID: "starrail"}, GrowthPlan{}, QueryResult{Data: data})
	if len(v.Rows) != 1 || v.Rows[0].Value != "120" {
		t.Fatal(v)
	}
}
