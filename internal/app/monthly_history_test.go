package app

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

type monthCaller struct {
	role        Role
	data        map[string]any
	denied      bool
	beforeReply func()
}

func (c *monthCaller) CallService(_ context.Context, req rayleabot.ServiceCallRequest, out any) error {
	if c.denied {
		return gameError("role_missing", "revoked")
	}
	if req.Method == "roles" {
		return decodeObject(map[string]any{"roles": []Role{c.role}}, out)
	}
	if req.Method == "execute" && req.Params["operation"] == c.role.Game+".monthly" {
		if c.beforeReply != nil {
			c.beforeReply()
		}
		return decodeObject(QueryResult{Role: c.role, Data: c.data}, out)
	}
	return gameError("operation_denied", "unexpected")
}
func TestMonthlyHistoryCoverageAndConcurrentChanges(t *testing.T) {
	role := Role{Ref: "role", Game: "zzz", UID: "10000001", Region: "prod_gf_cn"}
	caller := &monthCaller{role: role, data: map[string]any{"data_month": 202608, "month_data": map[string]any{"list": []any{map[string]any{"data_type": "PolychromesData", "count": 1000}, map[string]any{"data_type": "MatserTapeData", "count": 0}}}, "optional_month": []any{202608, 202609}}}
	client := AccountsClient{Caller: caller, Provider: "p", Game: "zzz"}
	choice := Selection{"account", "role"}
	input := map[string]any{"account_ref": "account", "role_ref": "role"}
	a := App{Game: Game{ID: "zzz"}, Monthly: &MonthlyStore{Directory: filepath.Join(t.TempDir(), "monthly")}}
	defer a.Close()
	call := func(action string) map[string]any {
		t.Helper()
		out, err := a.monthlyAction(t.Context(), client, action, input)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	call("monthly.fetch")
	caller.data = map[string]any{"data_month": 202609, "month_data": map[string]any{"list": []any{map[string]any{"data_type": "PolychromesData", "count": 2000}}}}
	call("monthly.fetch")
	a.Monthly = &MonthlyStore{Directory: a.Monthly.Directory}
	list := call("monthly.list")
	if list["totals"].(map[string]float64)["菲林"] != 3000 || list["coverage"].(map[string]int)["母带"] != 1 {
		t.Fatal(list)
	}
	caller.beforeReply = func() {
		if err := a.Monthly.Update(client.Provider, choice, list["revision"].(uint64), func(v *MonthlyArchive) error { v.Items = nil; return nil }); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := a.monthlyAction(t.Context(), client, "monthly.fetch", input); err == nil {
		t.Fatal("inflight fetch resurrected deleted archive")
	}
	caller.beforeReply = nil
	archive, _ := a.Monthly.Read(client.Provider, choice)
	if len(archive.Items) != 0 {
		t.Fatal(archive)
	}
	caller.denied = true
	if _, err := a.monthlyAction(t.Context(), client, "monthly.list", input); err == nil {
		t.Fatal("revoked account read history")
	}
	data, err := decodeMonthly(json.RawMessage(`{"id":1844674407370955101}`))
	if err != nil || asText(data["id"]) != "1844674407370955101" {
		t.Fatal(data, err)
	}
}
func TestDailyMonthlyCollectionPersistsAdmissionAndKeepsNotifyOff(t *testing.T) {
	now := time.Date(2026, 9, 21, 16, 0, 0, 0, time.UTC).UnixMilli()
	s := reminderStore(t.TempDir())
	task := Reminder{Ref: "month", Kind: "monthly", Hour: 23, Minute: 50, Enabled: true, ExpiresAtMS: now + int64(3*24*time.Hour/time.Millisecond)}
	if err := seedReminders(s, []Reminder{task}); err != nil {
		t.Fatal(err)
	}
	queries, sends := 0, 0
	query := func(Reminder) (QueryResult, error) {
		queries++
		return QueryResult{Data: map[string]any{"data_month": 202609}}, nil
	}
	send := func(Reminder, string) error { sends++; return nil }
	if err := s.Tick(task.Ref, now, query, send, Game{ID: "zzz"}); err != nil {
		t.Fatal(err)
	}
	s = &ReminderStore{taskFiles[Reminder]{Directory: s.Directory}}
	if err := s.Tick(task.Ref, now, query, send, Game{ID: "zzz"}); err != nil {
		t.Fatal(err)
	}
	items, _ := s.List()
	if queries != 1 || sends != 0 || items[0].LastCode != "collected" || items[0].NextCheckMS <= now {
		t.Fatal(queries, sends, items)
	}
}

// monthsCaller answers each monthly read with the month asked for.
type monthsCaller struct {
	role   Role
	months map[string]map[string]any
	asked  []string
}

func (c *monthsCaller) CallService(_ context.Context, req rayleabot.ServiceCallRequest, out any) error {
	month := asText(asObject(req.Params["input"])["month"])
	c.asked = append(c.asked, month)
	return decodeObject(QueryResult{Role: c.role, Data: c.months[month]}, out)
}

func TestRefreshMonthlyKeepsOfferedMonthsNotYetFinal(t *testing.T) {
	role := Role{Ref: "role", Game: "zzz", UID: "10000001"}
	month := func(key int) map[string]any {
		return map[string]any{"data_month": key, "month_data": map[string]any{"list": []any{}}, "optional_month": []any{202607, 202608, 202609}}
	}
	caller := &monthsCaller{role: role, months: map[string]map[string]any{"": month(202609), "202607": month(202607), "202608": month(202608)}}
	client := AccountsClient{Caller: caller, Provider: "p", Game: "zzz"}
	choice := Selection{"account", "role"}
	// The refresh runs in September, the month the report reads by default.
	a := App{Game: Game{ID: "zzz"}, Monthly: &MonthlyStore{Directory: filepath.Join(t.TempDir(), "monthly")}, clock: &fakeClock{at: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)}}
	// July was saved after it ended, August while it ran.
	august := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	if err := a.Monthly.Keep("p", choice, month(202607), august); err != nil {
		t.Fatal(err)
	}
	if err := a.Monthly.Keep("p", choice, month(202608), august); err != nil {
		t.Fatal(err)
	}
	if err := a.refreshMonthly(t.Context(), client, choice); err != nil {
		t.Fatal(err)
	}
	if len(caller.asked) != 2 || caller.asked[0] != "" || caller.asked[1] != "202608" {
		t.Fatalf("asked %q", caller.asked)
	}
	archive, _ := a.Monthly.Read("p", choice)
	if len(archive.Items) != 3 || !monthlyFinal(archive, "2026-08") || !monthlyFinal(archive, "2026-07") || monthlyFinal(archive, "2026-09") {
		t.Fatalf("archive = %+v", archive.Items)
	}
}

func TestMonthlyWordsFollowZZZPlugin(t *testing.T) {
	china := time.FixedZone("UTC+8", 8*3600)
	january := time.Date(2026, 1, 15, 12, 0, 0, 0, china)
	for word, want := range map[string]int{"上月": 202512, "3月": 0, "2025年3月": 202503, "2022年3月": 0, "2025年": 0, "2025年上月": 202412, "1月": 202601, "0月": 0, "2025年0月": 0} {
		if month, valid := monthlyWord(word, january); !valid || month != want {
			t.Errorf("%s = %d %v, want %d", word, month, valid, want)
		}
	}
	// A month still to come, or with no year in the past, reads the default.
	if month, _ := monthlyWord("2022年12月", time.Date(2026, 9, 1, 0, 0, 0, 0, china)); month != 0 {
		t.Errorf("an old year is this year: %d", month)
	}
	if month, _ := monthlyWord("8月", time.Date(2026, 9, 1, 0, 0, 0, 0, china)); month != 202608 {
		t.Errorf("8月 = %d", month)
	}
	if _, valid := monthlyWord("13月", january); valid {
		t.Error("13月 accepted")
	}
}
