package assets

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/RayleaBot/zzz/internal/reference"
)

// score-vectors.json holds three shared panels and the expected per-slot scores
// for every agent. The scores were checked against the earlier native port of
// ZZZ-Plugin Score when scoring moved to the script engine; "$dmg" stands for
// the slot 5 main stat named by each vector.
type scoreFixture struct {
	Contexts []map[string]any `json:"contexts"`
	Vectors  []struct {
		Key     string             `json:"key"`
		Context int                `json:"context"`
		Main    string             `json:"main"`
		Total   float64            `json:"total"`
		Pieces  map[string]float64 `json:"pieces"`
	} `json:"vectors"`
}

func scoreInput(t *testing.T, template map[string]any, main string) map[string]any {
	t.Helper()
	raw, _ := json.Marshal(template)
	var input map[string]any
	if json.Unmarshal(raw, &input) != nil {
		t.Fatal("invalid score context")
	}
	kept := []any{}
	for _, item := range input["equipment"].([]any) {
		disc := item.(map[string]any)
		if disc["main"] == "$dmg" {
			if main == "" {
				continue
			}
			disc["main"] = main
		}
		kept = append(kept, disc)
	}
	input["equipment"] = kept
	return input
}

func TestScoreVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/score-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture scoreFixture
	if json.Unmarshal(raw, &fixture) != nil || len(fixture.Vectors) == 0 {
		t.Fatal("invalid score fixture")
	}
	engine := calcEngine(t)
	records := map[string]reference.Character{}
	for _, c := range engine.Metadata().Characters {
		records[c.Key] = c
	}
	var wg sync.WaitGroup
	failures := make(chan string, len(fixture.Vectors))
	limit := make(chan struct{}, 8)
	for _, v := range fixture.Vectors {
		record, ok := records[v.Key]
		if !ok {
			t.Fatalf("agent %s left the catalog", v.Key)
		}
		input := scoreInput(t, fixture.Contexts[v.Context], v.Main)
		wg.Add(1)
		limit <- struct{}{}
		go func() {
			defer func() { <-limit; wg.Done() }()
			name := v.Key + "/" + strconv.Itoa(v.Context)
			result, err := engine.Score(context.Background(), record, input)
			if err != nil {
				failures <- name + ": " + err.Error()
				return
			}
			var output struct {
				Total  float64 `json:"total"`
				Pieces []struct {
					Slot  int     `json:"slot"`
					Score float64 `json:"score"`
				} `json:"pieces"`
			}
			equal := func(a, b float64) bool { return math.Abs(a-b) <= 1e-8*math.Max(1, math.Abs(b)) }
			if json.Unmarshal(result, &output) != nil || len(output.Pieces) != len(v.Pieces) || !equal(output.Total, v.Total) {
				failures <- name + ": total or piece count differs"
				return
			}
			for _, piece := range output.Pieces {
				if expected, ok := v.Pieces[strconv.Itoa(piece.Slot)]; !ok || !equal(piece.Score, expected) {
					failures <- name + ": slot " + strconv.Itoa(piece.Slot) + " differs"
					return
				}
			}
		}()
	}
	wg.Wait()
	close(failures)
	for failure := range failures {
		t.Error(failure)
	}
}
