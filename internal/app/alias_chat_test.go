package app

import (
	"slices"
	"testing"
)

func TestAliasOwnerPrefersCustomAliases(t *testing.T) {
	catalog, err := ParseCatalog([]byte(`{"version":"1","entries":[{"id":"1102","name":"希儿","aliases":["Seele"],"kind":"character"},{"id":"1101","name":"布洛妮娅","aliases":["鸭鸭"],"kind":"character"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	a := &App{Catalog: catalog}
	custom := map[string]string{"小鸭": "1101", "SEELE": "1101"}
	if entry, alias, ok := a.aliasOwner("seele", custom); !ok || entry.ID != "1101" || alias != "SEELE" {
		t.Fatal(entry, alias, ok)
	}
	if entry, alias, ok := a.aliasOwner("鸭鸭", custom); !ok || entry.ID != "1101" || alias != "" {
		t.Fatal(entry, alias, ok)
	}
	if _, _, ok := a.aliasOwner("不存在", custom); ok {
		t.Fatal("found a missing alias")
	}
	if list := aliasesOf(catalog.Entries[1], custom); !slices.Equal(list, []string{"鸭鸭", "SEELE", "小鸭"}) {
		t.Fatal(list)
	}
}
