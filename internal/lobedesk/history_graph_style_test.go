package lobedesk

import (
	"fmt"
	"math"
	"reflect"
	"testing"
)

func TestHistoryGraphStyleEndpointsAndLaneBounds(t *testing.T) {
	for _, style := range []string{"rounded", "chamfer", "compact"} {
		for _, gap := range []float32{15, 20} {
			for _, row := range []float32{28, 44} {
				for _, rows := range []float32{1, 2, 100} {
					for _, lanes := range [][3]float32{{0, 0, 0}, {0, 1, 1}, {1, 1, 0}, {2, 1, 0}, {0, 1, 0}, {2, 1, 2}} {
						name := fmt.Sprintf("%s/gap%g/row%g/span%g/lanes%v", style, gap, row, rows, lanes)
						t.Run(name, func(t *testing.T) {
							x, via, end := lanes[0]*gap, lanes[1]*gap, lanes[2]*gap
							y, lastY := row/2, row/2+rows*row
							points := historyGraphEdgePoints(style, x, y, via, end, lastY, row, y-row, lastY+row)
							if len(points) < 2 || points[0] != [2]float32{x, y} || points[len(points)-1] != [2]float32{end, lastY} {
								t.Fatalf("edge must reach both nodes: %v", points)
							}
							for i, p := range points {
								if p[0] < min(x, via, end) || p[0] > max(x, via, end) || p[1] < y || p[1] > lastY || (i > 0 && p[1] < points[i-1][1]) {
									t.Fatalf("point %d escapes lane/row bounds or doubles back: %v", i, p)
								}
							}
							middle := historyGraphEdgePoints(style, x, y, via, end, lastY, row, y+(lastY-y)*.49, y+(lastY-y)*.51)
							for _, p := range middle {
								if p[0] != via {
									t.Fatalf("middle must remain on via lane %g: %v", via, middle)
								}
							}
							if x == via && end == via && len(points) != 2 {
								t.Fatalf("same-lane edge needs only two endpoints, got %d", len(points))
							}
						})
					}
				}
			}
		}
	}
}

func TestHistoryGraphStylePreviewLandmarks(t *testing.T) {
	// The preview uses a 20px lane gap and 44px rows. These landmarks
	// distinguish the three approved shapes from the previous whole-row swoop.
	rounded := historyGraphEdgePoints("rounded", 0, 0, 20, 0, 132, 44, 0, 132)
	assertHistoryGraphPoint(t, rounded, [2]float32{12, 0})
	assertHistoryGraphPoint(t, rounded, [2]float32{20, 8})
	assertHistoryGraphPoint(t, rounded, [2]float32{20, 124})
	assertHistoryGraphPoint(t, rounded, [2]float32{12, 132})
	arcMid := float32(8 / math.Sqrt2)
	assertHistoryGraphPoint(t, rounded, [2]float32{12 + arcMid, 8 - arcMid})

	compact := historyGraphEdgePoints("compact", 0, 0, 20, 0, 132, 44, 0, 132)
	assertHistoryGraphPoint(t, compact, [2]float32{10, 9})
	assertHistoryGraphPoint(t, compact, [2]float32{20, 18})
	assertHistoryGraphPoint(t, compact, [2]float32{20, 114})
	assertHistoryGraphPoint(t, compact, [2]float32{10, 123})
	if compact[1][0] >= compact[1][1] {
		t.Fatalf("compact must depart vertically: %v", compact[:2])
	}

	chamfer := historyGraphEdgePoints("chamfer", 0, 0, 20, 0, 132, 44, 0, 132)
	cut := float32(7 / math.Sqrt2)
	assertHistoryGraphPoint(t, chamfer, [2]float32{20 - cut, 20 - cut})
	assertHistoryGraphPoint(t, chamfer, [2]float32{20, 27})
	assertHistoryGraphPoint(t, chamfer, [2]float32{20, 105})
	assertHistoryGraphPoint(t, chamfer, [2]float32{20 - cut, 112 + cut})
	if reflect.DeepEqual(rounded, chamfer) || reflect.DeepEqual(rounded, compact) || reflect.DeepEqual(chamfer, compact) {
		t.Fatal("styles must have distinct geometry")
	}
	for _, unknown := range []string{"", "invalid"} {
		if got := historyGraphEdgePoints(unknown, 0, 0, 20, 0, 132, 44, 0, 132); !reflect.DeepEqual(got, rounded) {
			t.Fatalf("style %q must default to rounded", unknown)
		}
	}
}

func TestHistoryGraphStyleMirrorsAndScales(t *testing.T) {
	for _, style := range []string{"rounded", "chamfer", "compact"} {
		points := historyGraphEdgePoints(style, 0, 0, 20, 40, 132, 44, 0, 132)
		mirror := historyGraphEdgePoints(style, 40, 0, 20, 0, 132, 44, 0, 132)
		scaled := historyGraphEdgePoints(style, 0, 0, 10, 20, 66, 22, 0, 66)
		if len(points) != len(mirror) || len(points) != len(scaled) {
			t.Fatalf("%s: transforms changed sample count", style)
		}
		for i, p := range points {
			if !historyGraphNear(mirror[i], [2]float32{40 - p[0], p[1]}) || !historyGraphNear(scaled[i], [2]float32{p[0] / 2, p[1] / 2}) {
				t.Fatalf("%s: geometry does not mirror/scale at point %d", style, i)
			}
		}
	}
}

func TestHistoryGraphStyleClippingSeams(t *testing.T) {
	for _, style := range []string{"rounded", "chamfer", "compact"} {
		for _, row := range []float32{28, 44} {
			for _, gap := range []float32{15, 20} {
				y, endY := row/2, row*3.5
				full := historyGraphEdgePoints(style, 0, y, gap, 2*gap, endY, row, 0, row*4)
				// Include arbitrary cuts through turns and exact node levels, where
				// rounded paths contain horizontal segments on the clipping boundary.
				for _, seam := range []float32{y, y + 1.23, y + 5, row, row * 2, endY - 5, endY - 1.23, endY} {
					for _, interval := range [][2]float32{{0, seam}, {seam, row * 4}} {
						got := historyGraphEdgePoints(style, 0, y, gap, 2*gap, endY, row, interval[0], interval[1])
						want := clipHistoryGraphTestPolyline(full, interval[0], interval[1])
						if len(got) != len(want) {
							t.Fatalf("%s row%g gap%g clip%v: got %v; want %v", style, row, gap, interval, got, want)
						}
						for i := range want {
							if !historyGraphNear(got[i], want[i]) {
								t.Fatalf("%s clip%v point%d: got%v want%v", style, interval, i, got[i], want[i])
							}
						}
					}
				}
			}
		}
	}
}

func TestHistoryGraphStyleLongEdgesBounded(t *testing.T) {
	for _, style := range []string{"rounded", "chamfer", "compact"} {
		for _, end := range []float32{4400, 4400000, 440000000} {
			middle := historyGraphEdgePoints(style, 0, 0, 20, 0, end, 44, 1000, 1044)
			if !reflect.DeepEqual(middle, [][2]float32{{20, 1000}, {20, 1044}}) {
				t.Fatalf("%s long middle should have two points: %v", style, middle)
			}
			start := historyGraphEdgePoints(style, 0, 0, 20, 0, end, 44, 0, 44)
			if len(start) > 40 {
				t.Fatalf("%s: visible point count grows with edge length: %d", style, len(start))
			}
		}
	}
}

func TestHistoryGraphStyleInvalidIntervals(t *testing.T) {
	for _, style := range []string{"rounded", "chamfer", "compact"} {
		for _, args := range [][5]float32{
			{0, 100, 44, 20, 10}, {0, 100, 44, 20, 20},
			{0, 100, 44, 101, 150}, {0, 100, 44, -50, -1},
			{100, 0, 44, 0, 100}, {0, 0, 44, 0, 100},
			{0, 100, 0, 0, 100}, {0, 100, -1, 0, 100},
			{0, 100, 44, float32(math.NaN()), 100},
			{0, float32(math.Inf(1)), 44, 0, 100},
		} {
			if got := historyGraphEdgePoints(style, 0, args[0], 20, 0, args[1], args[2], args[3], args[4]); len(got) != 0 {
				t.Fatalf("%s invalid arguments %v produced %v", style, args, got)
			}
		}
	}
}

func assertHistoryGraphPoint(t *testing.T, points [][2]float32, want [2]float32) {
	t.Helper()
	for _, p := range points {
		if historyGraphNear(p, want) {
			return
		}
	}
	t.Fatalf("missing expected landmark %v in %v", want, points)
}

func historyGraphNear(a, b [2]float32) bool {
	return math.Abs(float64(a[0]-b[0])) < .0001 && math.Abs(float64(a[1]-b[1])) < .0001
}

// Clip a complete polyline independently, to ensure row clipping cannot
// change its sample grid, seam position, or tangent on either side of a seam.
func clipHistoryGraphTestPolyline(points [][2]float32, top, bottom float32) [][2]float32 {
	var result [][2]float32
	add := func(p [2]float32) {
		if len(result) == 0 || result[len(result)-1] != p {
			result = append(result, p)
		}
	}
	for i := 1; i < len(points); i++ {
		a, b := points[i-1], points[i]
		if b[1] < top || a[1] > bottom {
			continue
		}
		intersect := func(y float32) [2]float32 {
			return [2]float32{a[0] + (b[0]-a[0])*(y-a[1])/(b[1]-a[1]), y}
		}
		lo, hi := a, b
		if a[1] < top {
			lo = intersect(top)
		}
		if b[1] > bottom {
			hi = intersect(bottom)
		}
		add(lo)
		add(hi)
	}
	return result
}
