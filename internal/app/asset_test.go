package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func TestPublicCodesFixedSourcesEstimatedExpiryAndNoCredentials(t *testing.T) {
	calls := 0
	c := PublicContentClient{HTTP: httpDoer(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Cookie") != "" || r.URL.Scheme != "https" {
			t.Fatal("unexpected credentials/transport")
		}
		data := ""
		switch r.URL.Path {
		case "/apihub/api/home/new":
			data = `{"navigator":[{"url":"https://act.mihoyo.com/?act_id=ABC123"}]}`
		case "/event/miyolive/index":
			if r.Header.Get("x-rpc-act_id") != "ABC123" {
				t.Fatal("wrong act")
			}
			data = `{"live":{"code_ver":"v1","title":"前瞻"}}`
		case "/event/miyolive/refreshCode":
			if r.URL.Query().Get("time") == "" {
				t.Fatal("missing time")
			}
			data = `{"code_list":[{"code":"TEST1234","title":"奖励","to_get_time":1720000000},{"code":"TEST1234"},{"code":"<script>"}]}`
		default:
			t.Fatal(r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"retcode":0,"data":` + data + `}`))}, nil
	})}
	result, err := c.codes(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	rows := result["items"].([]map[string]any)
	if calls != 3 || len(rows) != 1 || rows[0]["expiry_estimated"] != true {
		t.Fatal(result, calls)
	}
	if _, err = c.get(t.Context(), "http://bbs-api.miyoushe.com/", nil); err == nil {
		t.Fatal("http accepted")
	}
	if _, err = c.get(t.Context(), "https://other.example/", nil); err == nil {
		t.Fatal("unknown host accepted")
	}
}

type billingCaller struct {
	role  Role
	deny  bool
	calls int
}

func (c *billingCaller) CallService(_ context.Context, r rayleabot.ServiceCallRequest, out any) error {
	if c.deny {
		return gameError("role_missing", "denied")
	}
	if r.Method == "roles" {
		raw, _ := json.Marshal(map[string]any{"roles": []Role{c.role}})
		return json.Unmarshal(raw, out)
	}
	if r.Method == "list" {
		*out.(*Accounts) = Accounts{Items: []Account{{Ref: "account", Roles: []Role{c.role}}}}
		return nil
	}
	if r.Method != "execute" {
		return gameError("operation_denied", "unsupported")
	}
	c.calls++
	result := out.(*QueryResult)
	result.Role = c.role
	result.Data = map[string]any{"items": []any{map[string]any{"id": "1844674407370955101", "datetime": "2026-09-01", "add_num": json.Number("9007199254740993"), "action": "获得"}}, "has_more": true, "next_end_id": "1844674407370955101", "next_page": "2"}
	if c.calls > 1 {
		result.Data = map[string]any{"items": []any{}, "has_more": false, "next_page": "3"}
	}
	return nil
}
func TestContentCancellationIgnoresLateResults(t *testing.T) {
	s := ContentJobs{}
	defer s.Close()
	done := make(chan struct{})
	j, err := s.Start(func(ctx context.Context) (map[string]any, error) { <-done; return map[string]any{"late": true}, nil })
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Poll(j.Ref, true); err != nil {
		t.Fatal(err)
	}
	close(done)
	time.Sleep(time.Millisecond)
	got, _ := s.Poll(j.Ref, false)
	if got.State != "canceled" || got.Result != nil {
		t.Fatal("late result revived canceled job")
	}
}
