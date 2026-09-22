package app

import (
	"slices"
	"strings"
	"testing"
)

func TestEChartsLineFollowsEChartsScale(t *testing.T) {
	// A fixed maximum off the nice interval ends the axis with itself.
	if ticks := echartsTicks(112700, 6); !slices.Equal(ticks, []float64{0, 20000, 40000, 60000, 80000, 100000, 112700}) {
		t.Fatal(ticks)
	}
	if ticks := echartsTicks(60, 6); !slices.Equal(ticks, []float64{0, 10, 20, 30, 40, 50, 60}) {
		t.Fatal(ticks)
	}
	chart := EChartsLine([]float64{0, 50, 100}, []float64{100, 50, 0}, 1120, 520, [4]float64{30, 40, 50, 80}, 100, 6, 0.4, func(v float64) string { return "" })
	points := chart["points"].([]any)
	// The plot runs from x 80 to 1080 and y 30 to 470.
	if first := points[0].(map[string]any); first["x"] != "80" || first["y"] != "30" || points[2].(map[string]any)["y"] != "470" {
		t.Fatal(points)
	}
	// Points on a straight line keep their control points on it.
	if line := chart["line"].(string); !strings.HasPrefix(line, "M 80 30 C 80 30 ") || !strings.HasSuffix(line, "1080 470 1080 470") {
		t.Fatal(line)
	}
}
