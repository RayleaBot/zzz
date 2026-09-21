package assets

import (
	"encoding/json"
	"io/fs"
	"math"
	"testing"

	"github.com/dop251/goja"
)

func supplementVM(t *testing.T) *goja.Runtime {
	t.Helper()
	vm := goja.New()
	for _, script := range CalcProfile().Prelude {
		source, err := fs.ReadFile(script.Files, script.Path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = vm.RunScript(script.Path, string(source)); err != nil {
			t.Fatal(err)
		}
	}
	return vm
}
func TestZZZSupplementValuesAndSkillScope(t *testing.T) {
	vm := supplementVM(t)
	value, err := vm.RunString(`JSON.stringify((()=>{
 const list=id=>calcFnc.weapon[referenceMaps.WeaponId2Data[id].Name].buffs;
 const blood=list('13021')[0].value({calc:{avatar:{weapon:{star:1}},initial_properties:{CRITRate:1.3},get:()=>1.3}});
 const capped=list('13021')[0].value({calc:{avatar:{weapon:{star:5}},initial_properties:{CRITRate:9},get:()=>9}});
 const luna=list('12016')[0];
 return {count:supplementalZZZWeapons.size,blood,capped,luna:luna.value,lunaRange:luna.range,solar:list('14155').map(b=>b.value),none:list('12003').length};
 })())`)
	if err != nil {
		t.Fatal(err)
	}
	var r struct {
		Count         int
		Blood, Capped float64
		Luna          []float64
		LunaRange     []string
		Solar         []any
		None          int
	}
	if json.Unmarshal([]byte(value.String()), &r) != nil {
		t.Fatal("invalid result")
	}
	if r.Count != 33 || math.Abs(r.Blood-.144) > 1e-9 || r.Capped != .4 || r.Luna[0] != .18 || r.Luna[4] != .3 || r.LunaRange[0] != "A" || r.None != 0 {
		t.Fatal(r)
	}
	if r.Solar[0].(float64) != .2 || r.Solar[1].([]any)[0].(float64) != .16 {
		t.Fatal("fixed critical rate mixed with resistance", r.Solar)
	}
}
func TestZZZProfessionSevenSharpensOnlyMatchingSkills(t *testing.T) {
	vm := supplementVM(t)
	value, err := vm.RunString(`JSON.stringify((()=>{
 function result(weaponId,type){
  const avatar={level:60,rank:0,element_type:203,avatar_profession:7,weapon:weaponId?{name:referenceMaps.WeaponId2Data[weaponId].Name,star:1,profession:7}:null,
   base_properties:{ATK:1000,HP:10000,DEF:500},initial_properties:{ATK:1000,HP:10000,DEF:500,CRITRate:.05,CRITDMG:.5,PenRatio:0,Pen:0}};
  const bm=new BuffManager(avatar),calc=new Calculator(bm);if(weaponId)avatarPipeline.weapon_buff(avatar.weapon,bm);
  calc.new({name:'synthetic',type,element:'Electric',multiplier:1});return calc.calc()[0].result.expectDMG;
 }
 const baseline=result(null,'A');return {normal:result('14161','A')/baseline,sharp:result('14161','锐化')/baseline};
 })())`)
	if err != nil {
		t.Fatal(err)
	}
	var r struct{ Normal, Sharp float64 }
	_ = json.Unmarshal([]byte(value.String()), &r)
	crit := (1 + .3*.5) / (1 + .05*.5)
	if math.Abs(r.Normal-crit*1.15) > 1e-8 || math.Abs(r.Sharp-crit*1.25) > 1e-8 {
		t.Fatal(r)
	}
}
