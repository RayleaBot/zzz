package app

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type Entry struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Aliases     []string       `json:"aliases,omitempty"`
	Kind        string         `json:"kind"`
	Rarity      int            `json:"rarity"`
	Element     string         `json:"element,omitempty"`
	Weapon      string         `json:"weapon,omitempty"`
	Description string         `json:"description,omitempty"`
	Stats       map[string]any `json:"stats,omitempty"`
}
type Catalog struct {
	Version string  `json:"version"`
	Source  string  `json:"source"`
	Entries []Entry `json:"entries"`
	byID    map[string]int
}

func ParseCatalog(raw []byte) (Catalog, error) {
	var catalog Catalog
	if json.Unmarshal(raw, &catalog) != nil || catalog.Version == "" {
		return Catalog{}, fmt.Errorf("invalid game catalog")
	}
	catalog.byID = map[string]int{}
	for index, entry := range catalog.Entries {
		if _, exists := catalog.byID[entry.ID]; exists || entry.ID == "" || entry.Name == "" {
			return Catalog{}, fmt.Errorf("duplicate or invalid catalog entry")
		}
		catalog.byID[entry.ID] = index
	}
	return catalog, nil
}
func (c Catalog) Get(id string) (Entry, bool) {
	index, ok := c.byID[id]
	if !ok {
		return Entry{}, false
	}
	return c.Entries[index], true
}
func (c Catalog) Search(query, kind string, limit int, aliases map[string]string) []Entry {
	query = strings.ToLower(strings.TrimSpace(query))
	limit = min(max(limit, 1), 100)
	if id := aliases[query]; id != "" {
		if entry, ok := c.Get(id); ok && (kind == "" || kind == entry.Kind) {
			return []Entry{entry}
		}
	}
	var exact, partial []Entry
	for _, entry := range c.Entries {
		if kind != "" && kind != entry.Kind {
			continue
		}
		names := append([]string{entry.ID, entry.Name}, entry.Aliases...)
		isExact, isPartial := false, false
		for _, name := range names {
			name = strings.ToLower(name)
			if name == query {
				isExact = true
			}
			if strings.Contains(name, query) {
				isPartial = true
			}
		}
		if isExact {
			exact = append(exact, entry)
		} else if isPartial {
			partial = append(partial, entry)
		}
	}
	slices.SortFunc(exact, func(a, b Entry) int { return strings.Compare(a.ID, b.ID) })
	slices.SortFunc(partial, func(a, b Entry) int { return strings.Compare(a.ID, b.ID) })
	result := append(exact, partial...)
	if len(result) > limit {
		result = result[:limit]
	}
	if result == nil {
		result = []Entry{}
	}
	return result
}

// Resolve picks the entry a user means by a typed name: a custom alias, else
// the only entry whose ID, name or built-in alias equals the query, else the
// only entry containing it. An exact match wins over longer names that merely
// contain the query, so 丹恒 is not blocked by 丹恒•饮月.
func (c Catalog) Resolve(query, kind string, aliases map[string]string) (Entry, bool) {
	matches := c.Search(query, kind, 100, aliases)
	wanted := strings.ToLower(strings.TrimSpace(query))
	exact := []Entry{}
	for _, entry := range matches {
		for _, name := range append([]string{entry.ID, entry.Name}, entry.Aliases...) {
			if strings.ToLower(name) == wanted {
				exact = append(exact, entry)
				break
			}
		}
	}
	if len(exact) == 1 {
		return exact[0], true
	}
	if len(exact) == 0 && len(matches) == 1 {
		return matches[0], true
	}
	return Entry{}, false
}
