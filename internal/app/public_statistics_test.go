package app

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPublicStatisticsPreserveZeroAndSeparateDenominators(t *testing.T) {
	a := App{Game: Game{ID: "genshin"}, Content: PublicContentClient{HTTP: cloudDoer(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Cookie") != "" {
			t.Fatal("public stats carried credentials")
		}
		data := map[string]any{"code": 200, "result": []any{map[string]any{"role": "测试角色", "c0": 0, "c6": 100, "role_sum": 10}}}
		if r.URL.Host == "api.yshelper.com" {
			data = map[string]any{"code": 200, "result": []any{}, "has_list": []any{map[string]any{"name": "测试角色", "own_rate": 0}}}
		}
		raw, _ := json.Marshal(data)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	})}}
	out, err := a.publicStatistics(t.Context(), ContentQuery{Source: "ownership"})
	if err != nil {
		t.Fatal(err)
	}
	row := out["rows"].([]map[string]any)[0]
	if *row["holding_rate"].(*float64) != 0 || row["constellations"].([]*float64)[1] != nil || *row["constellations"].([]*float64)[0] != 0 {
		t.Fatal("missing values changed to zero")
	}
}
func TestStatisticsTeamsNeverReuseCharacterAcrossHalves(t *testing.T) {
	n := 10.0
	teams := []PublicTeam{{Names: []string{"A", "B"}, IDs: []string{"1", "2"}, Up: &n, Down: &n}, {Names: []string{"C", "D"}, IDs: []string{"3", "4"}, Up: &n, Down: &n}, {Names: []string{"A", "C"}, IDs: []string{"1", "3"}, Up: &n, Down: &n}}
	pairs := pairStatisticsTeams(teams, map[string]float64{"1": 100, "2": 90, "3": 80, "4": 70}, map[string]bool{})
	if len(pairs) == 0 {
		t.Fatal("no pair")
	}
	for _, p := range pairs {
		for _, a := range p.Up.Team.IDs {
			for _, b := range p.Down.Team.IDs {
				if a == b {
					t.Fatal("character reused")
				}
			}
		}
	}
}
