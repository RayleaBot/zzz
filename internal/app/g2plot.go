package app

import (
	"fmt"
	"math"
	"strconv"
)

// Yunzai's ledger pages draw their charts with G2Plot in the browser. Plugin
// templates do not run scripts, so these build the same charts as SVG shapes
// for the templates to place.

// G2Column is a G2Plot column chart of one series in a width x height box
// with G2Plot's padding (top, right, bottom, left): the y axis ticks G2 picks
// with its Wilkinson extended algorithm and their grid lines, one bar per
// value at half its band as G2 draws a single dodged series, each bar's value
// above it, and the category labels under the axis.
func G2Column(labels []string, values []float64, width, height float64, padding [4]float64) map[string]any {
	left, top := padding[3], padding[0]
	plotWidth, plotHeight := width-padding[1]-padding[3], height-padding[0]-padding[2]
	maximum := 0.0
	for _, value := range values {
		maximum = max(maximum, value)
	}
	ticks := g2Ticks(0, maximum, 5)
	scaleMax := max(maximum, ticks[len(ticks)-1])
	y := func(value float64) float64 {
		if scaleMax == 0 {
			return top + plotHeight
		}
		return top + plotHeight - value/scaleMax*plotHeight
	}
	grid := []any{}
	for _, tick := range ticks {
		grid = append(grid, map[string]any{"y": svgNumber(y(tick)), "label": strconv.FormatFloat(tick, 'f', -1, 64)})
	}
	bars := []any{}
	band := plotWidth / float64(max(len(values), 1))
	for index, value := range values {
		center := left + band*(float64(index)+0.5)
		barTop := y(value)
		bars = append(bars, map[string]any{"x": svgNumber(center - band/4), "y": svgNumber(barTop), "width": svgNumber(band / 2),
			"height": svgNumber(top + plotHeight - barTop), "center": svgNumber(center), "value": strconv.FormatFloat(value, 'f', -1, 64),
			"value_y": svgNumber(barTop - 2), "label": labels[index]})
	}
	return map[string]any{"width": svgNumber(width), "height": svgNumber(height), "left": svgNumber(left), "right": svgNumber(left + plotWidth),
		"bottom": svgNumber(top + plotHeight), "label_y": svgNumber(top + plotHeight + 16), "axis_x": svgNumber(left - 8), "grid": grid, "bars": bars}
}

// G2Ring is a G2Plot pie with an inner radius, as SVG sectors around (cx, cy)
// starting at the top and running clockwise; each sector over 2% carries its
// rounded share at mid radius, as the ledger pages' label formatter does. A
// value that is the whole ring is drawn as two halves.
func G2Ring(values []float64, colors []string, cx, cy, outer, inner float64) []any {
	total := 0.0
	for _, value := range values {
		total += value
	}
	point := func(radius, at float64) string {
		return fmt.Sprintf("%.2f %.2f", cx+radius*math.Cos(at), cy+radius*math.Sin(at))
	}
	sectors := []any{}
	angle := -math.Pi / 2
	for index, value := range values {
		if total == 0 || value == 0 {
			continue
		}
		share := value / total
		end := angle + share*2*math.Pi
		large := 0
		if end-angle > math.Pi {
			large = 1
		}
		path := fmt.Sprintf("M %s A %.0f %.0f 0 %d 1 %s L %s A %.0f %.0f 0 %d 0 %s Z", point(outer, angle), outer, outer, large, point(outer, end),
			point(inner, end), inner, inner, large, point(inner, angle))
		if share >= 0.999999 {
			path = fmt.Sprintf("M %s A %.0f %.0f 0 1 1 %s A %.0f %.0f 0 1 1 %s M %s A %.0f %.0f 0 1 0 %s A %.0f %.0f 0 1 0 %s Z",
				point(outer, -math.Pi/2), outer, outer, point(outer, math.Pi/2), outer, outer, point(outer, -math.Pi/2),
				point(inner, -math.Pi/2), inner, inner, point(inner, math.Pi/2), inner, inner, point(inner, -math.Pi/2))
		}
		sector := map[string]any{"path": path, "color": colors[index]}
		if percent := int(math.Round(share * 100)); percent > 2 {
			middle := (angle + end) / 2
			sector["label"], sector["x"], sector["y"] = strconv.Itoa(percent)+"%", svgNumber(cx+(outer+inner)/2*math.Cos(middle)), svgNumber(cy+(outer+inner)/2*math.Sin(middle))
		}
		sectors = append(sectors, sector)
		angle = end
	}
	return sectors
}

func svgNumber(value float64) string {
	return strconv.FormatFloat(math.Round(value*10)/10, 'f', -1, 64)
}

// g2Ticks is @antv/scale's wilkinson-extended tick method with G2's
// defaults: Q = [1, 5, 2, 2.5, 4, 3], weights [0.25, 0.2, 0.5, 0.05], and
// only ticks that cover the data.
func g2Ticks(dMin, dMax float64, m int) []float64 {
	if dMax-dMin < 1e-15 || m == 1 {
		return []float64{dMin}
	}
	q := []float64{1, 5, 2, 2.5, 4, 3}
	w := [4]float64{0.25, 0.2, 0.5, 0.05}
	eps := 100 * 2.220446049250313e-16
	simplicity := func(index int, j, lMin, lMax, lStep float64) float64 {
		v := 0.0
		mod := math.Mod(math.Mod(lMin, lStep)+lStep, lStep)
		if (mod < eps || lStep-mod < eps) && lMin <= 0 && lMax >= 0 {
			v = 1
		}
		return 1 - float64(index)/float64(len(q)-1) - j + v
	}
	simplicityMax := func(index int, j float64) float64 { return 1 - float64(index)/float64(len(q)-1) - j + 1 }
	density := func(k, m, lMin, lMax float64) float64 {
		r := (k - 1) / (lMax - lMin)
		rt := (m - 1) / (math.Max(lMax, dMax) - math.Min(dMin, lMin))
		return 2 - math.Max(r/rt, rt/r)
	}
	densityMax := func(k, m float64) float64 {
		if k >= m {
			return 2 - (k-1)/(m-1)
		}
		return 1
	}
	coverage := func(lMin, lMax float64) float64 {
		span := dMax - dMin
		return 1 - 0.5*(math.Pow(dMax-lMax, 2)+math.Pow(dMin-lMin, 2))/math.Pow(0.1*span, 2)
	}
	coverageMax := func(span float64) float64 {
		data := dMax - dMin
		if span > data {
			half := (span - data) / 2
			return 1 - math.Pow(half, 2)/math.Pow(0.1*data, 2)
		}
		return 1
	}
	best := struct{ score, lMin, lMax, lStep float64 }{score: -2}
	target := float64(m)
	for j := 1.0; ; j++ {
		stop := false
		for index, value := range q {
			sm := simplicityMax(index, j)
			if w[0]*sm+w[1]+w[2]+w[3] < best.score {
				stop = true
				break
			}
			for k := 2.0; ; k++ {
				dm := densityMax(k, target)
				if w[0]*sm+w[1]+w[2]*dm+w[3] < best.score {
					break
				}
				delta := (dMax - dMin) / (k + 1) / j / value
				for z := math.Ceil(math.Log10(delta)); ; z++ {
					step := j * value * math.Pow(10, z)
					cm := coverageMax(step * (k - 1))
					if w[0]*sm+w[1]*cm+w[2]*dm+w[3] < best.score {
						break
					}
					minStart := math.Floor(dMax/step)*j - (k-1)*j
					maxStart := math.Ceil(dMin/step) * j
					for start := minStart; start <= maxStart; start++ {
						lMin := start * (step / j)
						lMax := lMin + step*(k-1)
						score := w[0]*simplicity(index, j, lMin, lMax, step) + w[1]*coverage(lMin, lMax) + w[2]*density(k, target, lMin, lMax) + w[3]
						if score > best.score && lMin <= dMin && lMax >= dMax {
							best.score, best.lMin, best.lMax, best.lStep = score, lMin, lMax, step
						}
					}
				}
			}
		}
		if stop {
			break
		}
	}
	pretty := func(value float64) float64 {
		if math.Abs(value) < 1e-15 {
			return value
		}
		parsed, _ := strconv.ParseFloat(strconv.FormatFloat(value, 'f', 15, 64), 64)
		return parsed
	}
	lMin, lMax, lStep := pretty(best.lMin), pretty(best.lMax), pretty(best.lStep)
	count := int(math.Floor(math.Round((lMax-lMin)/lStep*1e12)/1e12)) + 1
	ticks := []float64{lMin}
	for index := 1; index < count; index++ {
		ticks = append(ticks, pretty(ticks[index-1]+lStep))
	}
	return ticks
}
