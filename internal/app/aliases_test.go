package app

import (
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func TestCustomAliasesOverrideBuiltinNames(t *testing.T) {
	catalog, err := ParseCatalog([]byte(`{"version":"test","entries":[{"id":"1","name":"角色甲","kind":"character","aliases":["小甲"]},{"id":"2","name":"角色乙","kind":"character"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	a := App{Catalog: catalog}
	// A policy saved by an earlier version no longer changes the outcome.
	event := &rayleabot.EventContext{Config: map[string]any{"custom_aliases": map[string]string{"小甲": "2"}, "alias_policy": "prefer_builtin"}}
	if got, ok := catalog.Resolve("小甲", "", a.aliasMap(event)); !ok || got.ID != "2" {
		t.Fatal(got)
	}
	normalized, conflicts, err := catalog.validateAliases(map[string]string{"小甲": "2"}, 256)
	if err != nil || normalized["小甲"] != "2" || len(conflicts) != 1 || conflicts[0].Builtin[0] != "1" {
		t.Fatal("an alias shadowing a built-in name was rejected or not reported")
	}
	if _, _, err = catalog.validateAliases(map[string]string{"alias": "1", " ALIAS ": "2"}, 256); err == nil {
		t.Fatal("normalized duplicates accepted")
	}
	applyGroupConfig(event, GroupConfig{Aliases: map[string]string{"小甲": "1"}})
	if got, ok := catalog.Resolve("小甲", "", a.aliasMap(event)); !ok || got.ID != "1" {
		t.Fatal("group alias did not override the global one")
	}
}
