// Package reference runs the pinned upstream calculation scripts of one game in
// an embedded JavaScript interpreter. The scripts are upstream code bundled by
// the game plugin, so their results match the reference plugin they came from.
package reference

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sync"
	"time"

	"github.com/dop251/goja"
)

//go:embed vendor/lodash.js
var vendor embed.FS

// Script is one JavaScript file run in the calculation interpreter.
type Script struct {
	Files fs.FS
	Path  string
}

// Lodash is the utility library every upstream calculation script expects.
func Lodash() Script { return Script{Files: vendor, Path: "vendor/lodash.js"} }

// Profile describes how to run one game's scripts. Files holds catalog.json and
// the character scripts it names; Prelude runs in order before the character
// script, then Runner defines runBuild and runScore.
type Profile struct {
	Files   fs.FS
	Prelude []Script
	Runner  Script
	// PassWeapons hands the game's weapon catalog to runBuild. The ZZZ runtime
	// carries its own weapon maps and skips it.
	PassWeapons bool
}

// Character names its damage rule under characters/ and its equipment scoring
// rule under scores/. Either may be empty: some characters are only scored,
// and characters without a scoring rule use the game's default weights.
type Character struct {
	Key         string         `json:"key"`
	Game        string         `json:"game"`
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Element     string         `json:"element"`
	WeaponType  string         `json:"weapon_type"`
	Script      string         `json:"script"`
	ScoreScript string         `json:"score_script,omitempty"`
	Data        map[string]any `json:"data"`
}
type Weapon struct {
	Game string         `json:"game"`
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}
type Catalog struct {
	Version    string      `json:"version"`
	Characters []Character `json:"characters"`
	Weapons    []Weapon    `json:"weapons"`
}

// Engine is safe for concurrent use. Compiled programs are shared; every run
// gets a fresh interpreter because upstream scripts mutate their buff and title
// objects while calculating.
type Engine struct {
	files      fs.FS
	catalog    Catalog
	weapons    string
	prelude    []*goja.Program
	runner     *goja.Program
	characters sync.Map // character or score script path -> compiled
}

func New(profile Profile) (*Engine, error) {
	if profile.Files == nil || profile.Runner.Files == nil {
		return nil, errors.New("calculation profile is incomplete")
	}
	raw, err := fs.ReadFile(profile.Files, "catalog.json")
	if err != nil {
		return nil, fmt.Errorf("read calculation catalog: %w", err)
	}
	engine := &Engine{files: profile.Files}
	if err = json.Unmarshal(raw, &engine.catalog); err != nil {
		return nil, fmt.Errorf("parse calculation catalog: %w", err)
	}
	for _, character := range engine.catalog.Characters {
		for _, check := range []struct{ script, dir string }{{character.Script, "characters"}, {character.ScoreScript, "scores"}} {
			if check.script != "" && !validScript(check.script, check.dir) {
				return nil, fmt.Errorf("calculation script path %q is invalid", check.script)
			}
		}
	}
	weapons := []Weapon{}
	if profile.PassWeapons {
		weapons = engine.catalog.Weapons
	}
	encoded, err := json.Marshal(weapons)
	if err != nil {
		return nil, err
	}
	engine.weapons = string(encoded)
	// The shared scripts are compiled once, so a broken bundle fails at start.
	for _, script := range profile.Prelude {
		p, err := compile(script)
		if err != nil {
			return nil, err
		}
		engine.prelude = append(engine.prelude, p)
	}
	if engine.runner, err = compile(profile.Runner); err != nil {
		return nil, err
	}
	return engine, nil
}

func validScript(name, dir string) bool {
	return fs.ValidPath(name) && path.Dir(name) == dir && path.Ext(name) == ".js"
}

func compile(script Script) (*goja.Program, error) {
	source, err := fs.ReadFile(script.Files, script.Path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", script.Path, err)
	}
	p, err := goja.Compile(script.Path, string(source), false)
	if err != nil {
		return nil, fmt.Errorf("compile %s: %w", script.Path, err)
	}
	return p, nil
}

// Metadata is shared and must be treated as read-only.
func (e *Engine) Metadata() Catalog { return e.catalog }

type compiled struct {
	program *goja.Program
	err     error
}

func (e *Engine) character(path string) (*goja.Program, error) {
	if value, ok := e.characters.Load(path); ok {
		c := value.(compiled)
		return c.program, c.err
	}
	p, err := compile(Script{Files: e.files, Path: path})
	value, _ := e.characters.LoadOrStore(path, compiled{p, err})
	c := value.(compiled)
	return c.program, c.err
}

// Run calculates damage with the character's rule. It accepts numerical
// business input only and returns the runner's JSON.
func (e *Engine) Run(ctx context.Context, character Character, input map[string]any) (json.RawMessage, error) {
	if !validScript(character.Script, "characters") {
		return nil, errors.New("no calculation rule for this character")
	}
	return e.call(ctx, "runBuild", character.Script, character, input)
}

// Change calculates the properties of a changed panel from its character,
// weapon, equipment and traces, as 面板换装 does upstream.
func (e *Engine) Change(ctx context.Context, character Character, input map[string]any) (json.RawMessage, error) {
	script := ""
	if validScript(character.Script, "characters") {
		// The panel takes the rule's static bonuses.
		script = character.Script
	}
	return e.call(ctx, "runChange", script, character, input)
}

// Score rates the equipment with the character's scoring rule, or with the
// game's default weights when the character has none.
func (e *Engine) Score(ctx context.Context, character Character, input map[string]any) (json.RawMessage, error) {
	if character.ScoreScript != "" && !validScript(character.ScoreScript, "scores") {
		return nil, errors.New("invalid scoring rule")
	}
	return e.call(ctx, "runScore", character.ScoreScript, character, input)
}

// Showcase turns a showcase service's answer into the official format with
// the runner's runShowcase, for games whose upstream does so in its scripts.
func (e *Engine) Showcase(ctx context.Context, input map[string]any) (json.RawMessage, error) {
	return e.call(ctx, "runShowcase", "", Character{}, input)
}

func (e *Engine) call(ctx context.Context, entry, script string, character Character, input map[string]any) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(input)
	if err != nil || len(encoded) > 512*1024 {
		return nil, errors.New("calculation input invalid")
	}
	data, err := json.Marshal(character)
	if err != nil {
		return nil, err
	}
	list := append(make([]*goja.Program, 0, len(e.prelude)+2), e.prelude...)
	if script != "" {
		rule, err := e.character(script)
		if err != nil {
			return nil, err
		}
		list = append(list, rule)
	}
	list = append(list, e.runner)
	ctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	vm := goja.New()
	vm.SetMaxCallStackSize(512)
	stop := context.AfterFunc(ctx, func() { vm.Interrupt("calculation canceled") })
	defer stop()
	for _, p := range list {
		if _, err = vm.RunProgram(p); err != nil {
			return nil, errors.New("calculation initialization failed")
		}
	}
	_ = vm.Set("inputJSON", string(encoded))
	_ = vm.Set("characterJSON", string(data))
	_ = vm.Set("weaponsJSON", e.weapons)
	result, err := vm.RunString("JSON.stringify(" + entry + "(JSON.parse(characterJSON),JSON.parse(weaponsJSON),JSON.parse(inputJSON)))")
	if err != nil {
		return nil, fmt.Errorf("calculation failed: %w", err)
	}
	raw := []byte(result.String())
	if len(raw) > 1024*1024 || !json.Valid(raw) {
		return nil, errors.New("calculation output invalid")
	}
	return raw, nil
}
