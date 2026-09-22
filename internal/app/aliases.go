package app

import (
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"slices"
	"strings"
)

type AliasConflict struct {
	Name    string   `json:"name"`
	Target  string   `json:"target"`
	Builtin []string `json:"builtin"`
}

// aliasConflicts lists custom aliases that shadow a built-in name or alias.
// Custom aliases always win; the list only tells administrators what they
// override.
func (c Catalog) aliasConflicts(aliases map[string]string) []AliasConflict {
	out := []AliasConflict{}
	for name, id := range aliases {
		ids := []string{}
		for _, entry := range c.Entries {
			for _, value := range append([]string{entry.Name, entry.ID}, entry.Aliases...) {
				if strings.EqualFold(value, name) && entry.ID != id && !slices.Contains(ids, entry.ID) {
					ids = append(ids, entry.ID)
				}
			}
		}
		if len(ids) > 0 {
			slices.Sort(ids)
			out = append(out, AliasConflict{name, id, ids})
		}
	}
	slices.SortFunc(out, func(a, b AliasConflict) int { return strings.Compare(a.Name, b.Name) })
	return out
}
func (c Catalog) validateAliases(aliases map[string]string, limit int) (map[string]string, []AliasConflict, error) {
	if len(aliases) > limit {
		return nil, nil, gameError("input_invalid", "别名数量超过上限。")
	}
	normalized := map[string]string{}
	for name, id := range aliases {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" || len(key) > 64 || strings.ContainsAny(key, "\r\n\t") {
			return nil, nil, gameError("input_invalid", "别名应为1–64字节，不含换行或制表符。")
		}
		if _, ok := c.Get(id); !ok {
			return nil, nil, gameError("entry_missing", "别名目标不存在。")
		}
		if _, ok := normalized[key]; ok {
			return nil, nil, gameError("input_invalid", "规范化后的别名重复。")
		}
		normalized[key] = id
	}
	return normalized, c.aliasConflicts(normalized), nil
}
func (a *App) aliasMap(event *rayleabot.EventContext) map[string]string {
	s := settings(event)
	aliases := map[string]string{}
	for name, id := range s.CustomAliases {
		key := strings.ToLower(strings.TrimSpace(name))
		if key != "" && len(key) <= 64 {
			if _, ok := a.Catalog.Get(id); ok {
				aliases[key] = id
			}
		}
	}
	return aliases
}
func (a *App) aliasAction(event *rayleabot.EventContext, input map[string]any) (map[string]any, error) {
	var q struct {
		Aliases map[string]string `json:"aliases"`
	}
	if decodeObject(input, &q) != nil {
		return nil, gameError("input_invalid", "别名输入无效。")
	}
	normalized, conflicts, err := a.Catalog.validateAliases(q.Aliases, 256)
	if err != nil {
		return nil, err
	}
	return map[string]any{"aliases": normalized, "conflicts": conflicts}, nil
}
