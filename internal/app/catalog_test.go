package app

import "testing"

func TestResolvePrefersTheExactName(t *testing.T) {
	catalog, err := ParseCatalog([]byte(`{"version":"test","entries":[
		{"id":"1002","name":"丹恒","kind":"character"},
		{"id":"1213","name":"丹恒•饮月","aliases":["饮月"],"kind":"character"},
		{"id":"8006","name":"星·同谐","kind":"character"},
		{"id":"8010","name":"星·同谐","kind":"character"}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	for query, want := range map[string]string{"丹恒": "1002", "饮月": "1213", "丹恒•": "1213", "8010": "8010", "龙尊": "1213"} {
		aliases := map[string]string{"龙尊": "1213"}
		entry, ok := catalog.Resolve(query, "character", aliases)
		if !ok || entry.ID != want {
			t.Fatalf("%s resolved to %q %v, want %s", query, entry.ID, ok, want)
		}
	}
	// Two entries share the name exactly; only the ID can pick one.
	if _, ok := catalog.Resolve("星·同谐", "character", nil); ok {
		t.Fatal("an ambiguous name resolved")
	}
}
