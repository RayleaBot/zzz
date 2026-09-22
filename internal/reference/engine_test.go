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
