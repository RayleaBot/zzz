package reference

import (
	"context"
	"encoding/json"
	"testing"
	"testing/fstest"
	"time"
)

// A synthetic game: the prelude defines a shared counter, the character script
// mutates it, and the runner reports it together with the input.
func syntheticProfile() Profile {
	files := fstest.MapFS{
		"catalog.json":       {Data: []byte(`{"version":"synthetic","characters":[{"key":"t_1","id":"1","name":"测试","script":"characters/1-测试.js"},{"key":"t_2","id":"2","name":"循环","script":"characters/2-循环.js"}],"weapons":[{"game":"t","id":"9","name":"武器"}]}`)},
		"prelude.js":         {Data: []byte(`var shared={count:0};`)},
		"characters/1-测试.js": {Data: []byte(`shared.count+=1;`)},
		"characters/2-循环.js": {Data: []byte(`while(true){}`)},
		"runner.js":          {Data: []byte(`function runBuild(character,weapons,input){return {name:character.name,count:shared.count,weapons:weapons.length,value:input.value,lodash:typeof _.cloneDeep}}`)},
	}
	return Profile{
		Files:       files,
		Prelude:     []Script{Lodash(), {Files: files, Path: "prelude.js"}},
		Runner:      Script{Files: files, Path: "runner.js"},
		PassWeapons: true,
	}
}

func TestRunIsolatesEachCalculation(t *testing.T) {
	engine, err := New(syntheticProfile())
	if err != nil {
		t.Fatal(err)
	}
	record := engine.Metadata().Characters[0]
	for range 3 {
		raw, err := engine.Run(context.Background(), record, map[string]any{"value": 7})
		if err != nil {
			t.Fatal(err)
		}
		var out struct {
			Name    string
			Count   int
			Weapons int
			Value   int
			Lodash  string
		}
		if err = json.Unmarshal(raw, &out); err != nil {
			t.Fatal(err)
		}
		if out.Name != "测试" || out.Count != 1 || out.Weapons != 1 || out.Value != 7 || out.Lodash != "function" {
			t.Fatalf("output = %+v, state leaked or prelude skipped", out)
		}
	}
}

func TestRunIsBoundedAndRejectsForeignScripts(t *testing.T) {
	engine, err := New(syntheticProfile())
	if err != nil {
		t.Fatal(err)
	}
	loop := engine.Metadata().Characters[1]
	start := time.Now()
	if _, err = engine.Run(context.Background(), loop, map[string]any{}); err == nil {
		t.Fatal("an endless script completed")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("calculation time was not bounded")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = engine.Run(canceled, engine.Metadata().Characters[0], map[string]any{}); err == nil {
		t.Fatal("canceled calculation ran")
	}
	for _, script := range []string{"runner.js", "../prelude.js", "characters/../runner.js", ""} {
		if _, err = engine.Run(context.Background(), Character{Script: script}, map[string]any{}); err == nil {
			t.Fatalf("script %q outside characters/ ran", script)
		}
	}
	profile := syntheticProfile()
	profile.Files.(fstest.MapFS)["catalog.json"] = &fstest.MapFile{Data: []byte(`{"characters":[{"key":"x","script":"runner.js"}]}`)}
	if _, err = New(profile); err == nil {
		t.Fatal("a catalog naming a non-character script was accepted")
	}
}

// Damage and Score load the character's damage and scoring rules together,
// and run for a character that only has a scoring rule.
func TestDamageAndScoreLoadBothRules(t *testing.T) {
	profile := syntheticProfile()
	files := profile.Files.(fstest.MapFS)
	files["catalog.json"] = &fstest.MapFile{Data: []byte(`{"characters":[{"key":"t_1","id":"1","script":"characters/1-测试.js","score_script":"scores/1-测试.js"},{"key":"t_3","id":"3","score_script":"scores/1-测试.js"}]}`)}
	files["scores/1-测试.js"] = &fstest.MapFile{Data: []byte(`shared.scored=true;`)}
	files["runner.js"] = &fstest.MapFile{Data: []byte(`function runDamage(){return {count:shared.count,scored:!!shared.scored}}const runScore=runDamage;`)}
	engine, err := New(profile)
	if err != nil {
		t.Fatal(err)
	}
	for index, want := range []string{`{"count":1,"scored":true}`, `{"count":0,"scored":true}`} {
		for name, run := range map[string]entry{"Damage": engine.Damage, "Score": engine.Score} {
			raw, err := run(context.Background(), engine.Metadata().Characters[index], map[string]any{})
			if err != nil || string(raw) != want {
				t.Errorf("%s of character %d ran %s, %v; want %s", name, index, raw, err, want)
			}
		}
	}
	if _, err = engine.Damage(context.Background(), Character{Script: "runner.js"}, map[string]any{}); err == nil {
		t.Error("a script outside characters/ ran")
	}
}

type entry func(context.Context, Character, map[string]any) (json.RawMessage, error)

// A rule that fails to compile or run is left out with the rules after it, as
// ZZZ-Plugin skips the rest of an agent's files after one fails to import:
// the runner then finds no such rule.
func TestRulesThatFailToLoadAreLeftOut(t *testing.T) {
	profile := syntheticProfile()
	files := profile.Files.(fstest.MapFS)
	files["catalog.json"] = &fstest.MapFile{Data: []byte(`{"characters":[` +
		`{"key":"t_4","id":"4","script":"characters/4-抛出.js","score_script":"scores/4-正常.js"},` +
		`{"key":"t_5","id":"5","script":"characters/5-语法.js","score_script":"scores/4-正常.js"},` +
		`{"key":"t_6","id":"6","script":"characters/6-正常.js","score_script":"scores/6-抛出.js"}]}`)}
	files["characters/4-抛出.js"] = &fstest.MapFile{Data: []byte(`var characterRule=(()=>{throw Error('broken')})();`)}
	files["characters/5-语法.js"] = &fstest.MapFile{Data: []byte(`var characterRule=;`)}
	files["characters/6-正常.js"] = &fstest.MapFile{Data: []byte(`var characterRule=true;`)}
	files["scores/4-正常.js"] = &fstest.MapFile{Data: []byte(`var scoreRule=true;`)}
	files["scores/6-抛出.js"] = &fstest.MapFile{Data: []byte(`var scoreRule=(()=>{throw Error('broken')})();`)}
	files["runner.js"] = &fstest.MapFile{Data: []byte(`function runDamage(){return {damage:typeof characterRule!=='undefined',score:typeof scoreRule!=='undefined'}}const runScore=runDamage;`)}
	engine, err := New(profile)
	if err != nil {
		t.Fatal(err)
	}
	for index, want := range []string{`{"damage":false,"score":false}`, `{"damage":false,"score":false}`, `{"damage":true,"score":false}`} {
		for name, run := range map[string]entry{"Damage": engine.Damage, "Score": engine.Score} {
			raw, err := run(context.Background(), engine.Metadata().Characters[index], map[string]any{})
			if err != nil || string(raw) != want {
				t.Errorf("%s of character %d ran %s, %v; want %s", name, index, raw, err, want)
			}
		}
	}
}
