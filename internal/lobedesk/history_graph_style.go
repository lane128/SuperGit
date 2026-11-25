package lobedesk

import "math"

// historyGraphEdgePoints defines one edge independently of the row drawing it.
// Turns are local to the nodes; the middle stays on viaX even for adjacent
// nodes. Unknown styles use rounded. The returned polyline is clipped to the
// closed interval [top, bottom], preserving horizontal segments at either
// boundary. Empty/reversed intervals and non-finite coordinates return nil.
func historyGraphEdgePoints(style string, sourceX, sourceY, viaX, parentX, parentY, rowHeight, top, bottom float32) [][2]float32 {
	for _, value := range [...]float32{sourceX, sourceY, viaX, parentX, parentY, rowHeight, top, bottom} {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return nil
		}
	}
	if parentY <= sourceY || rowHeight <= 0 || bottom <= top || bottom < sourceY || top > parentY {
		return nil
	}
	path := historyGraphPath{top: top, bottom: bottom}
	x, y, v, X, Y := float64(sourceX), float64(sourceY), float64(viaX), float64(parentX), float64(parentY)
	scale, span := float64(rowHeight)/44, Y-y
	start, end := historyGraphPoint{x, y}, historyGraphPoint{X, Y}
	if x == v && X == v {
		path.line(start, end)
		return path.points
	}
	switch style {
	case "compact":
		// Cubic controls leave/enter the node vertically, matching R4.
		current := start
		if x != v {
			a := min(18*scale, span*.4, math.Abs(v-x))
			current = historyGraphPoint{v, y + a}
			path.cubic(start, historyGraphPoint{x, y + a/2}, historyGraphPoint{v, y + a/2}, current)
		}
		if X != v {
			a := min(18*scale, span*.4, math.Abs(X-v))
			bend := historyGraphPoint{v, Y - a}
			path.line(current, bend)
			path.cubic(bend, historyGraphPoint{v, Y - a/2}, historyGraphPoint{X, Y - a/2}, end)
		} else {
			path.line(current, end)
		}
	case "chamfer":
		// R2 is a short diagonal polyline with quadratic corner rounding.
		vertices := []historyGraphPoint{start}
		radius := 7 * scale
		// Leave the central tenth of even an adjacent edge vertical. The
		// rounding extends beyond each diagonal's nominal endpoint.
		if x != v {
			h := min(20*scale, span*.4, math.Abs(v-x))
			vertices = append(vertices, historyGraphPoint{v, y + h})
			radius = min(radius, span*.45-h)
		}
		if X != v {
			h := min(20*scale, span*.4, math.Abs(X-v))
			vertices = append(vertices, historyGraphPoint{v, Y - h})
			radius = min(radius, span*.45-h)
		}
		vertices = append(vertices, end)
		current := start
		for i := 1; i < len(vertices)-1; i++ {
			a, b, c := vertices[i-1], vertices[i], vertices[i+1]
			before, after := math.Hypot(b[0]-a[0], b[1]-a[1]), math.Hypot(c[0]-b[0], c[1]-b[1])
			r := min(radius, before/2, after/2)
			p, q := b.toward(a, r/before), b.toward(c, r/after)
			path.line(current, p)
			path.quadratic(p, b, q)
			current = q
		}
		path.line(current, end)
	default:
		// R1 exits sideways and rounds into the lane with a quarter circle.
		current := start
		if x != v {
			r := min(8*scale, math.Abs(v-x), span*.4)
			sign := math.Copysign(1, v-x)
			bend := historyGraphPoint{v - sign*r, y}
			path.line(start, bend)
			current = historyGraphPoint{v, y + r}
			path.curve(bend, current, func(t float64) historyGraphPoint {
				angle := t * math.Pi / 2
				return historyGraphPoint{v - sign*r + sign*r*math.Sin(angle), y + r - r*math.Cos(angle)}
			})
		}
		if X != v {
			r := min(8*scale, math.Abs(X-v), span*.4)
			sign := math.Copysign(1, X-v)
			bend, exit := historyGraphPoint{v, Y - r}, historyGraphPoint{v + sign*r, Y}
			path.line(current, bend)
			path.curve(bend, exit, func(t float64) historyGraphPoint {
				angle := t * math.Pi / 2
				return historyGraphPoint{v + sign*r - sign*r*math.Cos(angle), Y - r + r*math.Sin(angle)}
			})
			path.line(exit, end)
		} else {
			path.line(current, end)
		}
	}
	return path.points
}

type historyGraphPoint [2]float64

func (p historyGraphPoint) toward(q historyGraphPoint, t float64) historyGraphPoint {
	return historyGraphPoint{p[0] + (q[0]-p[0])*t, p[1] + (q[1]-p[1])*t}
}

func (p historyGraphPoint) float32() [2]float32 {
	return [2]float32{float32(p[0]), float32(p[1])}
}

type historyGraphPath struct {
	top, bottom float32
	points      [][2]float32
}

func (p *historyGraphPath) add(point [2]float32) {
	if len(p.points) == 0 || p.points[len(p.points)-1] != point {
		p.points = append(p.points, point)
	}
}

func (p *historyGraphPath) line(from, to historyGraphPoint) {
	a, b := from.float32(), to.float32()
	if b[1] < p.top || a[1] > p.bottom {
		return
	}
	lo, hi := a, b
	// Use the original segment for both intersections. A horizontal segment
	// needs no division and is retained even when it lies on a row boundary.
	if a[1] < p.top {
		lo = [2]float32{a[0] + (b[0]-a[0])*(p.top-a[1])/(b[1]-a[1]), p.top}
	}
	if b[1] > p.bottom {
		hi = [2]float32{a[0] + (b[0]-a[0])*(p.bottom-a[1])/(b[1]-a[1]), p.bottom}
	}
	p.add(lo)
	p.add(hi)
}

// curve samples only visible segments of a fixed grid. Clipping a canonical
// grid (instead of resampling each row) preserves position and slope at seams.
// Finding the first visible segment is logarithmic in the fixed sample count;
// neither work nor storage depends on the total number of rows an edge spans.
func (p *historyGraphPath) curve(start, end historyGraphPoint, at func(float64) historyGraphPoint) {
	if float32(end[1]) < p.top || float32(start[1]) > p.bottom {
		return
	}
	const samples = 16
	sample := func(i int) historyGraphPoint {
		if i == 0 {
			return start
		}
		if i == samples {
			return end
		}
		return at(float64(i) / samples)
	}
	lo, hi := 1, samples
	for lo < hi {
		mid := (lo + hi) / 2
		if float32(sample(mid)[1]) < p.top {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	a := sample(lo - 1)
	for i := lo; i <= samples && float32(a[1]) <= p.bottom; i++ {
		b := sample(i)
		p.line(a, b)
		a = b
	}
}

func (p *historyGraphPath) quadratic(a, b, c historyGraphPoint) {
	p.curve(a, c, func(t float64) historyGraphPoint {
		return a.toward(b, t).toward(b.toward(c, t), t)
	})
}

func (p *historyGraphPath) cubic(a, b, c, d historyGraphPoint) {
	p.curve(a, d, func(t float64) historyGraphPoint {
		ab, bc, cd := a.toward(b, t), b.toward(c, t), c.toward(d, t)
		return ab.toward(bc, t).toward(bc.toward(cd, t), t)
	})
}
