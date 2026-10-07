package app

import (
	"reflect"
	"testing"

	"github.com/RayleaBot/zzz/internal/pluginmeta"
)

func TestCommandSetFollowsHostMatchingOrder(t *testing.T) {
	manifest, err := pluginmeta.Read([]byte(`{"id":"raylea.test","version":"1.0.0","commands":[
		{"id":"note","name":"体力","trigger":{"type":"exact","names":["体力","树脂"]}},
		{"id":"refresh","name":"更新面板","trigger":{"type":"exact","names":["更新面板"]}},
		{"id":"panel","name":"面板","trigger":{"type":"pattern","pattern":"^(?P<character>.+?)面板$"}},
		{"id":"late","name":"晚声明","trigger":{"type":"exact","names":["艾莲面板"]}}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	set, err := newCommandSet(manifest)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		word, id string
		args     []string
	}{
		{"树脂", "note", []string{"100"}},
		{"更新面板", "refresh", []string{"100"}},
		// A pattern declared first wins over a later exact name, as in the host.
		{"艾莲面板", "panel", []string{"艾莲", "100"}},
	} {
		id, args, ok := set.resolve(tc.word, []string{"100"})
		if !ok || id != tc.id || !reflect.DeepEqual(args, tc.args) {
			t.Fatalf("%s resolved to %q %v %v", tc.word, id, args, ok)
		}
	}
	if _, _, ok := set.resolve("原神体力", nil); ok {
		t.Fatal("undeclared word resolved")
	}
}
