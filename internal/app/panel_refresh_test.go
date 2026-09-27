package app

import (
	"strconv"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func characterIDs(n int) []string {
	ids := []string{}
	for i := range n {
		ids = append(ids, strconv.Itoa(1011+10*i))
	}
	return ids
}

func TestRoleIntervalFollowsZZZPlugin(t *testing.T) {
	if got := settings(&rayleabot.EventContext{}).roleInterval(); got != 3*time.Second {
		t.Fatalf("default = %v", got)
	}
	for value, want := range map[int]time.Duration{0: 100 * time.Millisecond, 20: 100 * time.Millisecond, 1500: 1500 * time.Millisecond} {
		if got := (Settings{PanelRoleInterval: value}).roleInterval(); got != want {
			t.Errorf("%d ms = %v", value, got)
		}
	}
}

// The management page asks an agent's details one agent a request.
func TestManagementCharacterQueryAsksOneAgent(t *testing.T) {
	a := pluginApp(t)
	for _, input := range []map[string]any{{}, {"id_list": []any{"1011", "1021"}}} {
		_, err := a.Manage(t.Context(), &rayleabot.EventContext{}, "query", map[string]any{"operation": "zzz.character", "account_ref": "account", "role_ref": "role", "input": input})
		if PublicError(err).Code != "plugin.game_input_invalid" {
			t.Fatalf("%v: %v", input, err)
		}
	}
}
