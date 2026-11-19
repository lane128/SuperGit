package lobedesk

import (
	"hash/fnv"
	"iter"
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

// lanePalette gives each graph lane a stable color when no theme is at hand,
// as in the tests. The values are mid-tone so the same line stays legible on
// both the light and dark surfaces, and they follow the theme lanes' hue
// order: blue, orange, green, purple, red, yellow, cyan, magenta.
var lanePalette = []ui.Color{
	ui.Hex("#4c8df6"),
	ui.Hex("#e8833a"),
	ui.Hex("#2fae7e"),
	ui.Hex("#c061cb"),
	ui.Hex("#e25c6b"),
	ui.Hex("#d4a72c"),
	ui.Hex("#34a9c9"),
	ui.Hex("#d560a7"),
}

// lanePaletteOf takes the graph's colors from the active theme, so the lines
// follow the selected color scheme. The slots are chosen for separation: the
// roles stay the same across schemes, and adjacent slots never land on hues
// that read alike. Gold and Geekblue were dropped for that reason — Lobe's
// Gold sits close to its Orange, and Catppuccin's Lavender close to its Blue —
// so wide graphs now use Yellow and Magenta instead.
func lanePaletteOf(tk *theme.Tokens) []ui.Color {
	return []ui.Color{tk.Blue, tk.Orange, tk.Green, tk.Purple, tk.Red, tk.Yellow, tk.Cyan, tk.Magenta}
}

func laneColor(palette []ui.Color, lane int) ui.Color {
	if len(palette) == 0 {
		palette = lanePalette
	}
	if lane < 0 {
		lane = -lane
	}
	return palette[lane%len(palette)]
}

// laneColorIndex derives a branch's palette slot from its name, so the same
// branch keeps its color when the history is filtered to another ref set.
// Without this the color followed the lane number, which changes whenever a
// filter re-lays out the rows.
func laneColorIndex(seed string, slots int) int {
	if slots <= 0 {
		slots = len(lanePalette)
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(seed))
	return int(hash.Sum32() % uint32(slots))
}

// refLaneColor gives a ref the color its line has in the graph. The branch,
// remote and tag lists use it for their row icons, so a name in a list can be
// matched to its lane by sight, the way the ref selector's dots already can.
func refLaneColor(palette []ui.Color, seed string) ui.Color {
	return laneColor(palette, laneColorIndex(seed, len(palette)))
}

// laneInk keeps a lane color readable as a small icon or a 2px line on the
// light theme. The ground is the selected row's tint, the darkest surface a
// line crosses and darker than the neutral tile an icon sits on, so one ink
// serves both and a branch keeps exactly one color across the graph, the
// branch page and the ref selector. Dark flavors need no change.
func laneInk(tk *theme.Tokens, color ui.Color) ui.Color {
	ground := selectionBg(tk, tk.BgContainer)
	return deepInk(color, ground, laneInkMin)
}

// gutterInk keeps a diff's line numbers readable on the row they sit on. The
// numbers are tertiary text drawn over plain, added, removed and selected
// rows; an added line under a selection can pull the tertiary token down to
// 3:1, so each row darkens its own gutter. On a plain surface the result is
// close to the token, so the gutter stays recessive. Dark flavors are left
// alone, as with the other ink helpers.
func gutterInk(tk *theme.Tokens, bg ui.Color) ui.Color {
	return deepInk(tk.TextTertiary, bg, 4.5)
}

// graphInk is the same ink as laneInk for the lines and nodes of the graph.
// It stays a named call site so the drawing code reads as "the ink a lane
// line uses", even though the two now resolve to one formula.
func graphInk(tk *theme.Tokens, color ui.Color) ui.Color {
	return laneInk(tk, color)
}

// contrastRatio is the WCAG contrast ratio between two opaque colors, from 1
// (identical) upwards.
func contrastRatio(a, b ui.Color) float64 {
	la, lb := surfaceLuminance(a), surfaceLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// surfaceLuminance is the sRGB relative luminance of a color, 0 (black) to 1
// (white); alpha is ignored because surfaces are opaque.
func surfaceLuminance(c ui.Color) float64 {
	linear := func(v uint8) float64 {
		x := float64(v) / 255
		if x <= 0.04045 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(c.R) + 0.7152*linear(c.G) + 0.0722*linear(c.B)
}

// boundaryInk keeps the dashed tail at the end of the loaded history visible.
// The tail is a quaternary hint, but it marks that the line continues past
// what is loaded, so it has to clear the non-text 3:1 minimum on the row it
// sits on; dark flavors keep the token as with the other ink helpers.
func boundaryInk(tk *theme.Tokens, bg ui.Color) ui.Color {
	return deepInk(tk.TextQuaternary, bg, 3.0)
}

// graphNodeGround is the surface behind a commit row's graph node. The
// selected ring's halo and the hollow centre of a merge node paint it, so it
// has to be the row's own background: the page colour would leave a pale disc
// inside a tinted row.
func graphNodeGround(tk *theme.Tokens, selected, hovered bool) ui.Color {
	switch {
	case selected:
		return selectionBg(tk, tk.BgContainer)
	case hovered:
		return tk.FillTertiary.Over(tk.BgContainer)
	}
	return tk.BgContainer
}

// laneSeed names the branch a commit starts from. A local branch wins over a
// remote branch, which wins over a tag, and unattached commits fall back to
// their SHA.
func laneSeed(commit gitrepo.Commit) string {
	var tag string
	for _, ref := range commitRefs(commit.Refs) {
		switch {
		case ref == "HEAD":
			// Detached HEAD: no branch name to seed a lane with, so the
			// commit falls back to its SHA unless a tag names it below.
		case strings.HasPrefix(ref, "tag: "):
			if tag == "" {
				tag = ref[len("tag: "):]
			}
		default:
			return ref
		}
	}
	if tag != "" {
		return tag
	}
	return commit.SHA
}

// commitGraphEdge connects a commit to one of its parents. ViaLane is the lane
// the edge reserves while it travels: it keeps commits of interleaved branches
// from claiming the same column, and it names the color the line wears. The
// line itself is drawn as one curve from dot to dot, not along that column.
type commitGraphEdge struct {
	Start, End             int
	SourceLane, ParentLane int
	ViaLane                int
	Color                  int
	Merge                  bool
}

type commitGraphRow struct {
	Lane     int
	Color    int
	Boundary bool
	Fork     bool // More than one distinct child in the loaded DAG.
}

// commitGraph stores each edge once. An interval index finds edges crossing
// a visible row without expanding long edges into copies for every row.
type commitGraph struct {
	Worktree   *worktreeGraphLink
	Rows       []commitGraphRow
	edges      []commitGraphEdge // Ordered by source row, then original parent order.
	edgeMaxEnd []int             // Subtree maximum End in the implicit balanced interval tree.
	Lanes      int
	Width      float32
	// Gap is the distance between lane centres. It is graphLaneGap until a
	// narrow list makes FitWidth compress the graph to leave the subject room.
	Gap float32
	// Palette holds the lane colors, in the order the color indices use; an
	// empty palette falls back to lanePalette.
	Palette []ui.Color
	// RefLanes records the color index a ref name was drawn with. A name whose
	// hash slot is already taken steps to the next free color, so this can
	// differ from what laneColorIndex alone would pick.
	RefLanes map[string]int
}

// LaneColor returns the color of a lane or edge color index.
func (g *commitGraph) LaneColor(index int) ui.Color {
	return laneColor(g.Palette, index)
}

// RefColor returns the color the graph actually drew a ref's line with.
func (g *commitGraph) RefColor(name string) (ui.Color, bool) {
	index, ok := g.RefLanes[name]
	if !ok {
		return ui.Color{}, false
	}
	return g.LaneColor(index), true
}

// RowLaneColor returns the color of the line a row carries, so a label whose
// name never allocated a lane can still wear the color of the line it sits on.
func (g *commitGraph) RowLaneColor(row int) (ui.Color, bool) {
	if row < 0 || row >= len(g.Rows) {
		return ui.Color{}, false
	}
	return g.LaneColor(g.Rows[row].Color), true
}

func newCommitGraph(commits []gitrepo.Commit, palette []ui.Color) *commitGraph {
	graph := &commitGraph{Palette: palette, RefLanes: map[string]int{}}
	if len(palette) == 0 {
		graph.Palette = lanePalette
	}
	slots := len(graph.Palette)
	if len(commits) == 0 {
		return graph
	}
	positions := make(map[string]int, len(commits))
	for i, commit := range commits {
		positions[commit.SHA] = i
	}
	children := make([]int, len(commits))
	lastChild := make(map[string]int)
	for child, commit := range commits {
		for _, parent := range commit.Parents {
			if row, ok := positions[parent]; ok && row > child && lastChild[parent] != child+1 {
				children[row]++
				lastChild[parent] = child + 1
			}
		}
	}
	graph.Rows = make([]commitGraphRow, len(commits))
	var lanes []string
	var colors []int
	find := func(sha string) int {
		for i, value := range lanes {
			if value == sha {
				return i
			}
		}
		return -1
	}
	allocate := func(sha, seed string) int {
		lane := -1
		for i, value := range lanes {
			if value == "" {
				lane = i
				break
			}
		}
		if lane < 0 {
			lanes = append(lanes, "")
			colors = append(colors, 0)
			lane = len(lanes) - 1
		}
		// Start from the branch's own palette slot, then step past slots a
		// live lane already shows so two neighboring lines do not share a
		// color. A filtered list usually has fewer live lanes, so a branch
		// that moved keeps its own slot.
		color := laneColorIndex(seed, slots)
		for tries := 0; tries < slots; tries++ {
			used := false
			for i, value := range lanes {
				if i != lane && value != "" && colors[i] == color {
					used = true
					break
				}
			}
			if !used {
				break
			}
			color = (color + 1) % slots
		}
		lanes[lane] = sha
		colors[lane] = color
		graph.RefLanes[seed] = color
		return lane
	}
	var edges []commitGraphEdge
	for i, commit := range commits {
		lane := find(commit.SHA)
		if lane < 0 {
			lane = allocate(commit.SHA, laneSeed(commit))
		}
		row := commitGraphRow{Lane: lane, Color: colors[lane], Fork: children[i] > 1}
		for j, value := range lanes {
			if value == commit.SHA {
				lanes[j] = ""
			}
		}
		for parentIndex, parent := range commit.Parents {
			end, ok := positions[parent]
			if !ok || end <= i {
				// The parent is filtered out or not loaded yet; the row draws
				// a dashed tail instead of a line to nowhere.
				row.Boundary = true
				continue
			}
			via := find(parent)
			merge := parentIndex > 0
			switch {
			case !merge:
				via = lane
				lanes[via] = parent
			case via < 0:
				via = allocate(parent, laneSeed(commits[end]))
			}
			edges = append(edges, commitGraphEdge{
				Start: i, End: end, SourceLane: lane, ViaLane: via, Color: colors[via], Merge: merge,
			})
		}
		graph.Rows[i] = row
	}
	for i := range edges {
		edges[i].ParentLane = graph.Rows[edges[i].End].Lane
	}
	graph.edges = edges
	graph.edgeMaxEnd = make([]int, len(edges))
	graph.indexEdges(0, len(edges))
	maxLane := 0
	for _, edge := range edges {
		maxLane = max(maxLane, edge.ViaLane, edge.SourceLane, edge.ParentLane)
	}
	for _, row := range graph.Rows {
		maxLane = max(maxLane, row.Lane)
	}
	graph.Lanes = maxLane + 1
	graph.Width = graphWidth(graph.Lanes)
	graph.Gap = graphLaneGap
	return graph
}

// indexEdges builds a balanced interval tree in-place over source-ordered edges.
func (g *commitGraph) indexEdges(lo, hi int) int {
	if lo >= hi {
		return -1
	}
	mid := lo + (hi-lo)/2
	end := max(g.edges[mid].End, g.indexEdges(lo, mid), g.indexEdges(mid+1, hi))
	g.edgeMaxEnd[mid] = end
	return end
}

// spans yields exactly the inclusive intervals crossing row, preserving painter
// order. It allocates no result slice, even when thousands of lanes cross a row.
func (g *commitGraph) spans(row int) iter.Seq[commitGraphEdge] {
	return func(yield func(commitGraphEdge) bool) {
		if row < 0 || row >= len(g.Rows) {
			return
		}
		g.visitSpans(row, 0, len(g.edges), yield)
	}
}

func (g *commitGraph) visitSpans(row, lo, hi int, yield func(commitGraphEdge) bool) bool {
	if lo >= hi || g.edges[lo].Start > row {
		return true
	}
	mid := lo + (hi-lo)/2
	if g.edgeMaxEnd[mid] < row {
		return true
	}
	if !g.visitSpans(row, lo, mid, yield) {
		return false
	}
	edge := g.edges[mid]
	if edge.Start <= row && edge.End >= row && !yield(edge) {
		return false
	}
	return g.visitSpans(row, mid+1, hi, yield)
}

// The largest selected halo reaches 12 DIP from the lane center and the
// thickest adjacent stroke reaches 1.5 DIP toward it. Leave room for both
// and their antialiasing. Extra lanes scroll within the same viewport.
const graphLaneGapMin float32 = 15

func (g *commitGraph) contentWidth() float32 {
	return g.worktreeGutter() + 16 + float32(g.Lanes)*g.LaneGap()
}

// FitWidth caps the viewport while retaining readable lane geometry.
func (g *commitGraph) FitWidth(maxWidth float32) {
	if maxWidth <= 0 || g.Lanes <= 0 {
		return
	}
	budget := maxWidth
	gutter := g.worktreeGutter()
	maxWidth -= gutter
	natural := graphWidth(g.Lanes)
	if natural <= maxWidth {
		g.Width = natural + gutter
		g.Gap = graphLaneGap
		return
	}
	gap := (maxWidth - 16) / float32(g.Lanes)
	gap = max(graphLaneGapMin, min(graphLaneGap, gap))
	g.Gap = gap
	g.Width = min(budget, g.contentWidth())
}

// LaneGap is the distance between lane centres this graph is drawn with.
func (g *commitGraph) LaneGap() float32 {
	if g == nil || g.Gap <= 0 {
		return graphLaneGap
	}
	return g.Gap
}

// The graph's line thickness and node size are display preferences, not
// layout constants: a dense history reads better with thinner lines, and a
// large screen makes a larger node easier to hit. The lane spacing itself is
// fixed, so changing these never moves a lane's x position.
const (
	graphLineThin     float32 = 1.5
	graphLineStandard float32 = 2
	graphLineThick    float32 = 3

	graphNodeSmall    float32 = 0.8
	graphNodeStandard float32 = 1
	graphNodeLarge    float32 = 1.3

	// graphDotRadius is the radius of a plain commit dot at the standard node
	// preset, and graphMergeHalf the radius of a merge/HEAD ring. Both scale
	// with the preset and are floored against a thick line (see graphNodeDot).
	graphDotRadius = 4
	graphMergeHalf = 5
	// laneInkMin is the contrast a lane color must reach on the light theme's
	// selected-row tint before the app draws it as a 2px line or a list icon.
	laneInkMin = 3.2
)

func graphLineOf(key string) float32 {
	switch key {
	case "thin":
		return graphLineThin
	case "thick":
		return graphLineThick
	}
	return graphLineStandard
}

func graphLineKey(width float32) string {
	switch width {
	case graphLineThin:
		return "thin"
	case graphLineThick:
		return "thick"
	}
	return "standard"
}

func graphNodeOf(key string) float32 {
	switch key {
	case "small":
		return graphNodeSmall
	case "large":
		return graphNodeLarge
	}
	return graphNodeStandard
}

func graphNodeKey(scale float32) string {
	switch scale {
	case graphNodeSmall:
		return "small"
	case graphNodeLarge:
		return "large"
	}
	return "standard"
}

// graphStyle returns the line width and node scale to draw with. A zero value
// means the settings have not been applied yet, so the standard style stands
// in; that keeps a fresh app and a test-built one identical.
func (a *app) graphStyle() (line, node float32) {
	line, node = a.historyLine, a.historyNode
	if line <= 0 {
		line = graphLineStandard
	}
	if node <= 0 {
		node = graphNodeStandard
	}
	return line, node
}

// graphNodeDot is the radius of a solid commit dot. The node preset alone
// decides it in every combination but one: a thick line with the small node
// leaves a dot 0.2 DIP wider than the line it sits on, so the commit reads as
// a bare line with no node. The dot keeps one DIP of margin beyond the line
// instead, which is the smallest bump that still reads as a point.
func graphNodeDot(node, line float32) float32 {
	return max(graphDotRadius*node, line+1)
}

// graphNodeMerge is the radius of a merge/HEAD ring or the half-diagonal
// of a fork diamond. It stays wider than a plain dot at every line preset.
func graphNodeMerge(node, line float32) float32 {
	return max(graphMergeHalf*node, graphNodeDot(node, line)+1)
}

// graphNodeRing is the selected node's 2 DIP ring and the ground halo behind
// it. The ring keeps a visible gap from the node itself: when the thick line's
// floor grows the dot, the ring grows with it, so the two never merge into one
// fat disc. The halo stays two DIPs beyond the ring.
func graphNodeRing(node, line float32) (inner, ring float32) {
	// Stroke is inside its bounds. Leave a 1.5-DIP surface gap between
	// its 2-DIP ink and the largest identity mark, including merge rings.
	inner = max(7*node, graphNodeMerge(node, line)+3.5)
	ring = max(9*node, inner+2)
	return inner, ring
}

// graphWidth is the graph column's width for a lane count. It follows the
// lanes instead of stopping at a fixed maximum: the selected commit's ring
// reaches nine pixels past its lane's centre, so a cap would let the widest
// lanes draw outside the column and over the subject text. A repository with
// dozens of live branches gives the graph a wide column, which is the same
// trade Sublime Merge makes; the subject keeps its tooltip.
func graphWidth(lanes int) float32 {
	if lanes <= 0 {
		return 16
	}
	return float32(16 + lanes*graphLaneGap)
}

// graphLaneGap is the horizontal distance between two lane columns. A bend
// needs room to read as a curve: with the gap far narrower than the row, the
// S turns back into the diagonal it replaced. 20 is the tightest that keeps
// the widest node (18.2 at the 1.3 preset) from touching its neighbour while
// still leaving the bend visible; the reader asked for the lanes to sit close,
// and the history row's own budget compresses this further on a narrow list.
const graphLaneGap = 20

func laneX(x, lane int) float32 {
	return laneXGap(x, lane, graphLaneGap)
}

// laneXGap places a lane for a graph whose gap FitWidth may have compressed.
func laneXGap(x, lane int, gap float32) float32 {
	return float32(x+14) + float32(lane)*gap
}

// graphSwoopSamples is how many points a full bend is cut into. A row only
// draws part of a bend, so it takes the share that falls inside it.
const graphSwoopSamples = 16

// graphSwoop samples the S an edge draws between two dots: it leaves the
// source dot straight down, swings across its lane change, and arrives straight
// down at the parent's dot. Both ends keep a vertical tangent, so edges that
// meet at a node join without a corner.
func graphSwoop(x0, y0, x1, y1 float32) [][2]float32 {
	return graphSwoopRange(x0, y0, x1, y1, y0, y1)
}

// graphSwoopRange returns the part of a bend between top and bottom, so the
// row above and the row below each draw their half and the halves meet with
// the same slope on either side of the row boundary.
func graphSwoopRange(x0, y0, x1, y1, top, bottom float32) [][2]float32 {
	if y1 <= y0 || bottom < y0 || top > y1 {
		return nil
	}
	top = max(top, y0)
	bottom = min(bottom, y1)
	if bottom < top {
		return nil
	}
	d := y1 - y0
	t0 := (top - y0) / d
	t1 := (bottom - y0) / d
	samples := int((t1-t0)*graphSwoopSamples + 0.5)
	if samples < 1 {
		samples = 1
	}
	points := make([][2]float32, 0, samples+1)
	for i := 0; i <= samples; i++ {
		t := t0 + (t1-t0)*float32(i)/float32(samples)
		// Ease across, linear down: the horizontal speed is zero at both ends
		// and fastest in the middle. The fifth-order curve holds the lane
		// longer at the dots and turns harder across the middle, so the same
		// lane gap reads as a rounder bend than a plain smoothstep would.
		x := x0 + (x1-x0)*(t*t*t*(t*(t*6-15)+10))
		y := y0 + (y1-y0)*t
		points = append(points, [2]float32{x, y})
	}
	return points
}

// graphEdgePoints draws the part of an edge that falls inside one row. The
// line leaves the source dot, bends into its via lane over one whole row,
// runs straight down that lane, and bends into the parent dot the same way.
// The bend gets a full row of height and a wide lane gap, so it reads as the
// arc the reference graph draws; a bend squeezed into half a row is the
// diagonal that reads as a straight line. An edge that spans a single row
// joins the two dots with one bend.
func graphEdgePoints(sourceX, sourceY, viaX, parentX, parentY, rowHeight, top, bottom float32) [][2]float32 {
	if parentY-sourceY <= 0 || rowHeight <= 0 {
		return nil
	}
	if parentY-sourceY <= rowHeight*1.001 {
		return graphSwoopRange(sourceX, sourceY, parentX, parentY, top, bottom)
	}
	points := graphSwoopRange(sourceX, sourceY, viaX, sourceY+rowHeight, top, bottom)
	if lo, hi := max(sourceY+rowHeight, top), min(parentY-rowHeight, bottom); lo < hi {
		points = append(points, [2]float32{viaX, lo}, [2]float32{viaX, hi})
	}
	return append(points, graphSwoopRange(viaX, parentY-rowHeight, parentX, parentY, top, bottom)...)
}

// refBadge describes how a commit ref is drawn in the history list and the
// details pane: local branches, remote branches and tags each get their own
// color and icon, the way Sublime Merge separates them.
type refBadge struct {
	Label string
	Color string
	Icon  *ui.SVG
}

// refBadgeColor is the raw line color a ref badge points at. A capsule tints
// its surface from this color, so it needs the undarkened tone; darkening first
// would turn the surface muddy.
//
// A branch or a remote branch is a line, so it wears the color the graph draws
// its line with; a branch that never allocated a lane of its own takes its
// row's line color, or its hash slot would name a color no line shows. A tag is
// a point: it never grows a lane, so it keeps the name color the tag list and
// the ref selector show for it.
func refBadgeColor(palette []ui.Color, graph *commitGraph, row int, kind, name string) ui.Color {
	if kind == historyRefTag {
		return refLaneColor(palette, name)
	}
	if graph != nil {
		if drawn, ok := graph.RefColor(name); ok {
			return drawn
		}
		if drawn, ok := graph.RowLaneColor(row); ok {
			return drawn
		}
	}
	return refLaneColor(palette, name)
}

// refColor resolves a badge family to the tone Lobe paints it in: named
// colors use their darker Active tone, status colors their base accent, as
// tagColorsFor does.
func refColor(tk *theme.Tokens, family string) (ui.Color, bool) {
	name := strings.ToUpper(family[:1]) + family[1:]
	color, ok := tk.Color(name + "Active")
	switch family {
	case "success", "error", "warning", "info", "processing":
		color, ok = tk.Color(name)
	}
	return color, ok
}

// deepInk darkens a color toward black until it clears min against surface.
// Mixing toward black rather than toward body text keeps the hue: a tag label
// stays recognizably green, red or gold while gaining contrast, where mixing
// toward a neutral text color drains it to grey on the light flavors. Dark
// surfaces are returned unchanged.
func deepInk(color, surface ui.Color, min float64) ui.Color {
	if surfaceLuminance(surface) < 0.5 {
		return color
	}
	black := ui.RGB(0, 0, 0)
	for step := float32(0.05); step <= 0.95; step += 0.05 {
		mixed := color.Mix(black, step)
		if contrastRatio(mixed, surface) >= min {
			return mixed
		}
	}
	return black
}

func (a *app) refBadge(ref string) refBadge {
	switch {
	case ref == "HEAD":
		return refBadge{Label: ref, Color: "geekblue", Icon: lucide.GitCommitHorizontal()}
	case len(ref) > 5 && ref[:5] == "tag: ":
		return refBadge{Label: ref[5:], Color: "gold", Icon: lucide.Tag()}
	case len(ref) > 7 && ref[:7] == "HEAD -> ":
		return refBadge{Label: ref, Color: "success", Icon: lucide.GitBranch()}
	case a.isRemoteRef(ref):
		return refBadge{Label: ref, Color: "geekblue", Icon: lucide.Cloud()}
	default:
		return refBadge{Label: ref, Color: "success", Icon: lucide.GitBranch()}
	}
}

// parseCommitRef splits a ref as the history graph prints it into the kind
// and name the ref selector uses. A bare HEAD has no single ref to open, and
// a symbolic "HEAD -> origin/main" is reported as its target.
func (a *app) parseCommitRef(ref string) (kind, name string, ok bool) {
	ref = strings.TrimSpace(ref)
	switch {
	case ref == "" || ref == "HEAD":
		return "", "", false
	case strings.HasPrefix(ref, "tag: "):
		return historyRefTag, strings.TrimPrefix(ref, "tag: "), true
	case strings.HasPrefix(ref, "HEAD -> "):
		ref = strings.TrimPrefix(ref, "HEAD -> ")
	case strings.Contains(ref, " -> "):
		return "", "", false
	}
	if a.isRemoteRef(ref) {
		return historyRefRemote, ref, true
	}
	return historyRefLocal, ref, true
}

// isRemoteRef reports whether the part before the first slash names a
// configured remote, so a local branch such as feature/login is not mistaken
// for a remote branch.
func (a *app) isRemoteRef(ref string) bool {
	for i, r := range ref {
		if r != '/' {
			continue
		}
		prefix := ref[:i]
		for _, remote := range a.remotes {
			if remote.Name == prefix {
				return true
			}
		}
		return false
	}
	return false
}
