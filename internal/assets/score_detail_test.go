package assets

import (
	"context"
	"encoding/json"
	"os"
	"testing"
)

// TestScoreDetailFollowsZZZPlugin checks the panel detail against
// ZZZ-Plugin's propertyStats: rolls include the initial one, values are the
// base roll times the rolls, and the list is ordered by rolls then weight.
func TestScoreDetailFollowsZZZPlugin(t *testing.T) {
	raw, err := os.ReadFile("testdata/score-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture scoreFixture
	if json.Unmarshal(raw, &fixture) != nil {
		t.Fatal("invalid score fixture")
	}
	vector := fixture.Vectors[0]
	engine := calcEngine(t)
	for _, record := range engine.Metadata().Characters {
		if record.Key != vector.Key {
			continue
		}
		out, err := engine.Score(context.Background(), record, scoreInput(t, fixture.Contexts[vector.Context], vector.Main))
		if err != nil {
			t.Fatal(err)
		}
		var detail struct {
			Weights map[string]float64 `json:"weights"`
			Stats   []struct {
				ID     int     `json:"id"`
				Name   string  `json:"name"`
				Weight float64 `json:"weight"`
				Value  string  `json:"value"`
				Count  int     `json:"count"`
			} `json:"stats"`
			Pieces []struct {
				Props []struct {
					ID     int     `json:"id"`
					Count  int     `json:"count"`
					Weight float64 `json:"weight"`
				} `json:"props"`
			} `json:"pieces"`
		}
		if err := json.Unmarshal(out, &detail); err != nil || len(detail.Stats) == 0 || len(detail.Pieces) == 0 {
			t.Fatalf("detail = %s", out)
		}
		rolls := map[int]int{}
		for _, piece := range detail.Pieces {
			for _, prop := range piece.Props {
				rolls[prop.ID] += prop.Count + 1
			}
		}
		for index, stat := range detail.Stats {
			if stat.Count != rolls[stat.ID] || stat.Name == "" {
				t.Errorf("stat %+v, want %d rolls", stat, rolls[stat.ID])
			}
			if index > 0 {
				previous := detail.Stats[index-1]
				if previous.Count < stat.Count || previous.Count == stat.Count && previous.Weight < stat.Weight {
					t.Errorf("stats out of order at %d", index)
				}
			}
			if stat.ID == 20103 && stat.Value != "21.6%" {
				t.Errorf("crit rate = %+v", stat)
			}
		}
		return
	}
	t.Fatalf("%s left the catalog", vector.Key)
}
