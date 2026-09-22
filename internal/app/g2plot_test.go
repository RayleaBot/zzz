package app

import (
	"reflect"
	"testing"
)

// The expected ticks are what the g2plot.min.js Yunzai ships returns for the
// same maxima.
func TestG2TicksMatchG2Plot(t *testing.T) {
	cases := map[float64][]float64{
		0.5: {0, 0.1, 0.2, 0.3, 0.4, 0.5}, 7: {0, 2, 4, 6, 8}, 99: {0, 25, 50, 75, 100}, 480: {0, 100, 200, 300, 400, 500},
		1600: {0, 400, 800, 1200, 1600}, 3150: {0, 800, 1600, 2400, 3200}, 7777: {0, 2000, 4000, 6000, 8000},
		12345: {0, 2500, 5000, 7500, 10000, 12500}, 16000: {0, 4000, 8000, 12000, 16000}, 23800: {0, 5000, 10000, 15000, 20000, 25000},
		48000: {0, 10000, 20000, 30000, 40000, 50000}, 99999: {0, 25000, 50000, 75000, 100000}, 150000: {0, 50000, 100000, 150000},
	}
	for maximum, want := range cases {
		if got := g2Ticks(0, maximum, 5); !reflect.DeepEqual(got, want) {
			t.Errorf("g2Ticks(0, %v) = %v, want %v", maximum, got, want)
		}
	}
}
