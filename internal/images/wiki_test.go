package images

import (
	"encoding/json"
	"testing"

	"github.com/RayleaBot/plugin-zzz/internal/app"
)

func TestWikiFollowsZZZPlugin(t *testing.T) {
	for word, want := range map[string][6]int{"艾莲技能": {12, 12, 12, 12, 12, 6}, "艾莲技能10.11 9": {10, 11, 9, 12, 12, 6}, "艾莲技能A.B.C.D.E.F": {1, 2, 3, 4, 5, 6}} {
		if got, ok := app.SkillLevels(word); !ok || got != want {
			t.Errorf("SkillLevels(%s) = %v %v", word, got, ok)
		}
	}
	if _, ok := app.SkillLevels("艾莲技能13"); ok {
		t.Error("levels past 12 are refused, as upstream")
	}
	got := nanokaRichText("点按 <IconMap:Icon_Normal> 发动：\n造成<color=#98EFF0>冰属性伤害</color><b>")
	if got != `<div class="line">点按 <span class="skill-icon Normal"></span> 发动：</div><div class="line">造成<span style="color:#98EFF0"><strong>冰属性伤害</strong></span>&lt;b&gt;</div>` {
		t.Errorf("rich text = %s", got)
	}
	// Upstream takes the parameter object's first value in its own order.
	value, ok := nanokaFirstValue(json.RawMessage(`{"9":{"main":100,"format":"%"},"1":{"main":1}}`))
	if !ok || value.Main != 100 || value.Format != "%" {
		t.Errorf("first value = %+v", value)
	}
}
