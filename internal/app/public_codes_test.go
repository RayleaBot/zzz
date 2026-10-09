package app

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPublicCodesWithoutLiveNavigationUsesOfficialPosts(t *testing.T) {
	c := PublicContentClient{HTTP: httpDoer(func(r *http.Request) (*http.Response, error) {
		data := ""
		switch r.URL.Path {
		case "/apihub/api/home/new":
			data = `{"navigator":[{"name":"签到福利","app_path":"https://act.mihoyo.com/?act_id=SIGN123"}]}`
		case "/painter/api/user_instant/list":
			if r.URL.Host != "bbs-api.mihoyo.com" || r.URL.Query().Get("uid") != "152039148" {
				t.Errorf("unexpected posts source: %s", r.URL)
			}
			data = `{"list":[{"post":{"content":"https://webstatic.mihoyo.com/bbs/event/live/index.html?act_id=ABC123&game_biz=nap"}}]}`
		case "/event/miyolive/index":
			if r.Header.Get("x-rpc-act_id") != "ABC123" {
				t.Errorf("unexpected activity: %s", r.Header.Get("x-rpc-act_id"))
			}
			data = `{"live":{"code_ver":"v1","title":"测试版本前瞻"}}`
		case "/event/miyolive/refreshCode":
			data = `{"code_list":[{"code":"TEST1234","title":"菲林*300"}]}`
		default:
			t.Errorf("unexpected request: %s", r.URL)
			return nil, io.ErrUnexpectedEOF
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"retcode":0,"data":` + data + `}`))}, nil
	})}
	a := pluginApp(t)
	a.Content = c
	host := newSDKHost(t, a, func(hostCall) (map[string]any, string) {
		t.Error("public codes must not require an account service")
		return nil, "plugin.service_unavailable"
	})
	end, _ := host.groupMessage("%兑换码", "兑换码")
	text := terminalText(end)
	if !strings.Contains(text, "TEST1234") || !strings.Contains(text, "菲林*300") || !strings.Contains(text, "测试版本前瞻") {
		t.Fatalf("unexpected code reply: %+v", end)
	}
}

func TestPublicCodesLiveAnonymousSource(t *testing.T) {
	if os.Getenv("RAYLEA_PUBLIC_SMOKE") != "1" {
		t.Skip("explicit anonymous network smoke")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Second)
	defer cancel()
	result, err := (PublicContentClient{}).codes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	items := result["items"].([]map[string]any)
	t.Logf("title=%s codes=%d source=%s", result["title"], len(items), result["source"])
}
