package app

import (
	"slices"
	"testing"
)

func TestAliasOwnerPrefersCustomAliases(t *testing.T) {
	catalog, err := ParseCatalog([]byte(`{"version":"1","entries":[{"id":"1191","name":"艾莲·乔","aliases":["Ellen"],"kind":"character"},{"id":"1041","name":"「11号」","aliases":["11号"],"kind":"character"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	a := &App{Catalog: catalog}
	custom := map[string]string{"十一": "1041", "ELLEN": "1041"}
	if entry, alias, ok := a.aliasOwner("ellen", custom); !ok || entry.ID != "1041" || alias != "ELLEN" {
		t.Fatal(entry, alias, ok)
	}
	if entry, alias, ok := a.aliasOwner("11号", custom); !ok || entry.ID != "1041" || alias != "" {
		t.Fatal(entry, alias, ok)
	}
	if _, _, ok := a.aliasOwner("不存在", custom); ok {
		t.Fatal("found a missing alias")
	}
	if list := aliasesOf(catalog.Entries[1], custom); !slices.Equal(list, []string{"11号", "ELLEN", "十一"}) {
		t.Fatal(list)
	}
}
