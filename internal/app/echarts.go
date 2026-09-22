package app

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ark-plugin's graph/stats draws its chart with ECharts in the browser.
// Plugin templates do not run scripts, so EChartsLine builds the same chart
// as SVG shapes for the templates to place.

// EChartsLine lays out an ECharts line chart with value axes in a width x
// height box with the grid's (top, right, bottom, left) margins: x runs over
// 0–100, y from 0 to yMax with the ticks ECharts picks for splitNumber, and
// the points are joined as ECharts smooths a line. It returns the line and
// area paths, each point with the place of its label above the symbol, and
// the y ticks labelled by label.
func EChartsLine(xs, values []float64, width, height float64, grid [4]float64, yMax float64, splitNumber int, smooth float64, label func(float64) string) map[string]any {
	left, top := grid[3], grid[0]
	right, bottom := width-grid[1], height-grid[2]
	x := func(value float64) float64 { return left + value/100*(right-left) }
	y := func(value float64) float64 {
		if yMax <= 0 {
			return bottom
		}
		return bottom - value/yMax*(bottom-top)
	}
	ticks := []any{}
	for _, tick := range echartsTicks(yMax, splitNumber) {
		ticks = append(ticks, map[string]any{"y": svgNumber(y(tick)), "label": label(tick)})
	}
	points := make([][2]float64, len(values))
	shapes := []any{}
	for index, value := range values {
		points[index] = [2]float64{x(xs[index]), y(value)}
		shapes = append(shapes, map[string]any{"x": svgNumber(points[index][0]), "y": svgNumber(points[index][1]), "value": strconv.FormatFloat(value, 'f', -1, 64)})
	}
	line := echartsSmooth(points, smooth)
	area := ""
	if len(points) > 0 {
		area = fmt.Sprintf("%s L %s %s L %s %s Z", line, svgNumber(points[len(points)-1][0]), svgNumber(bottom), svgNumber(points[0][0]), svgNumber(bottom))
	}
	return map[string]any{"width": svgNumber(width), "height": svgNumber(height), "left": svgNumber(left), "right": svgNumber(right), "top": svgNumber(top),
		"bottom": svgNumber(bottom), "line": line, "area": area, "points": shapes, "ticks": ticks}
}

// echartsNice is ECharts' numberUtil.nice with rounding.
func echartsNice(value float64) float64 {
	exponent := math.Floor(math.Log10(value))
	exp10 := math.Pow(10, exponent)
	f := value / exp10
	nf := 10.0
	switch {
	case f < 1.5:
		nf = 1
	case f < 2.5:
		nf = 2
	case f < 4:
		nf = 3
	case f < 7:
		nf = 5
	}
	return nf * exp10
}

// echartsTicks are the ticks of an ECharts interval scale fixed to 0–max:
// multiples of the nice interval, then max itself when it is not one.
func echartsTicks(maximum float64, splitNumber int) []float64 {
	if maximum <= 0 || splitNumber < 1 {
		return []float64{0}
	}
	interval := echartsNice(maximum / float64(splitNumber))
	ticks := []float64{}
	for tick := 0.0; tick <= maximum+interval*1e-9; tick = math.Round((tick+interval)*1e9) / 1e9 {
		ticks = append(ticks, tick)
	}
	if ticks[len(ticks)-1] < maximum {
		ticks = append(ticks, maximum)
	}
	return ticks
}

// echartsSmooth is the path ECharts' poly shape draws through points with a
// smooth factor and no monotone direction: each control pair follows the
// neighbours' direction, split by the lengths of the two segments and kept
// within the neighbouring points.
func echartsSmooth(points [][2]float64, smooth float64) string {
	if len(points) == 0 {
		return ""
	}
	path := []string{"M " + svgNumber(points[0][0]) + " " + svgNumber(points[0][1])}
	cp0 := points[0]
	prev := points[0]
	for index := 1; index < len(points); index++ {
		point := points[index]
		if index == len(points)-1 {
			path = append(path, fmt.Sprintf("C %s %s %s %s %s %s", svgNumber(cp0[0]), svgNumber(cp0[1]), svgNumber(point[0]), svgNumber(point[1]), svgNumber(point[0]), svgNumber(point[1])))
			break
		}
		next := points[index+1]
		lenPrev := math.Hypot(point[0]-prev[0], point[1]-prev[1])
		lenNext := math.Hypot(next[0]-point[0], next[1]-point[1])
		ratio := lenNext / (lenNext + lenPrev)
		vx, vy := next[0]-prev[0], next[1]-prev[1]
		nextCp := [2]float64{point[0] + vx*smooth*ratio, point[1] + vy*smooth*ratio}
		nextCp[0] = math.Max(math.Min(nextCp[0], math.Max(next[0], point[0])), math.Min(next[0], point[0]))
		nextCp[1] = math.Max(math.Min(nextCp[1], math.Max(next[1], point[1])), math.Min(next[1], point[1]))
		vx, vy = nextCp[0]-point[0], nextCp[1]-point[1]
		cp1 := [2]float64{point[0] - vx*lenPrev/lenNext, point[1] - vy*lenPrev/lenNext}
		cp1[0] = math.Max(math.Min(cp1[0], math.Max(prev[0], point[0])), math.Min(prev[0], point[0]))
		cp1[1] = math.Max(math.Min(cp1[1], math.Max(prev[1], point[1])), math.Min(prev[1], point[1]))
		vx, vy = point[0]-cp1[0], point[1]-cp1[1]
		nextCp = [2]float64{point[0] + vx*lenNext/lenPrev, point[1] + vy*lenNext/lenPrev}
		path = append(path, fmt.Sprintf("C %s %s %s %s %s %s", svgNumber(cp0[0]), svgNumber(cp0[1]), svgNumber(cp1[0]), svgNumber(cp1[1]), svgNumber(point[0]), svgNumber(point[1])))
		cp0, prev = nextCp, point
	}
	return strings.Join(path, " ")
}
