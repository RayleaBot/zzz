package app

import (
	"strings"
	"testing"
)

// The help word alone answers the menu. With more after it, which the host
// hands over as arguments, the message is not the command, as upstream's
// rule ends at the word: the help another bot answers in a group, whose text
// starts with a prefix and the word, gets no reply.
func TestHelpAnswersTheWordAlone(t *testing.T) {
	a := pluginApp(t)
	host := newSDKHost(t, a, func(hostCall) (map[string]any, string) {
		return nil, "plugin.service_unavailable"
	})
	end, _ := host.groupMessage("%帮助", "帮助")
	if !strings.Contains(terminalText(end), a.Game.Name+"帮助") {
		t.Fatalf("answered %+v", end)
	}
	end, _ = host.groupMessage("绝区零帮助\r>下图为指令集，点击查看原图", "帮助", ">下图为指令集，点击查看原图")
	if asObject(end["data"])["handled"] != false || len(host.sent) != 0 {
		t.Fatalf("answered the other bot's help with %+v and %+v", end, host.sent)
	}
}
