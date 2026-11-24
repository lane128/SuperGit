package lobedesk

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

// A node has to stay a node in every combination of the node preset and the
// line width. The dot scales with the preset, but a thick line with the small
// preset draws a dot barely wider than the line, and the commit then reads as
// a bare line. The floor keeps one DIP of margin beyond the line.
func TestGraphNodesStayVisibleInEveryLineWidth(t *testing.T) {
	nodes := []float32{graphNodeSmall, graphNodeStandard, graphNodeLarge}
	lines := []float32{graphLineThin, graphLineStandard, graphLineThick}
	for _, node := range nodes {
		for _, line := range lines {
			dot := graphNodeDot(node, line)
			cmp := func(what string, got, want float32) {
				if got < want {
					t.Errorf("node %v line %v: %s = %v, want at least %v", node, line, what, got, want)
				}
			}
			cmp("dot", dot, 4*node)
			cmp("dot", dot, line+1)
			merge := graphNodeMerge(node, line)
			cmp("merge square", merge, 5*node)
			// The square keeps two DIPs past the lane, and stays a step wider
			// than the dot so a merge commit does not read as a plain node.
			cmp("merge square", merge, line+2)
			cmp("merge square", merge, dot+1)
		}
	}
	// The combination that motivated the floor: the dot has to grow past
	// what the preset alone would draw.
	dot := graphNodeDot(graphNodeSmall, graphLineThick)
	if dot <= 4*graphNodeSmall {
		t.Errorf("thick line with the small node kept dot %v; it should grow past %v", dot, 4*graphNodeSmall)
	}
	// A thick line with the small node used to leave the merge square the
	// same 8 points as a dot; it has to keep its own weight and outline.
	if merge := graphNodeMerge(graphNodeSmall, graphLineThick); merge < dot+1 || merge < graphLineThick+2 {
		t.Errorf("small node with the thick line gives a %v half-square beside a %v dot", merge, dot)
	}
	// The selected ring has to keep its gap from the dot: a ring that grew
	// with the dot but not past it would close into one fat disc.
	for _, node := range nodes {
		for _, line := range lines {
			inner, ring := graphNodeRing(node, line)
			dot := graphNodeDot(node, line)
			if inner-1 < dot+1.5 {
				t.Errorf("node %v line %v: ring inner %v leaves no gap from dot %v", node, line, inner, dot)
			}
			if ring < inner+2 {
				t.Errorf("node %v line %v: halo %v is not beyond ring %v", node, line, ring, inner)
			}
		}
	}
}

func TestCommitGraphLaysOutLinearHistory(t *testing.T) {
	commits := []gitrepo.Commit{
		{SHA: "a", Parents: []string{"b"}},
		{SHA: "b", Parents: []string{"c"}},
		{SHA: "c"},
	}
	graph := newCommitGraph("", commits, nil)
	if graph.Lanes != 1 {
		t.Fatalf("lanes = %d; want 1", graph.Lanes)
	}
	for i, row := range graph.Rows {
		if row.Lane != 0 || row.Boundary {
			t.Errorf("row %d = %+v; want lane 0 without boundary", i, row)
		}
	}
	if len(graphRowSpans(graph, 0)) != 1 || graphRowSpans(graph, 0)[0].End != 1 {
		t.Fatalf("row 0 spans = %+v; want one edge to row 1", graphRowSpans(graph, 0))
	}
	if len(graphRowSpans(graph, 2)) != 1 {
		t.Fatalf("row 2 spans = %+v; want the incoming edge", graphRowSpans(graph, 2))
	}
}

func TestCommitGraphBranchesAndMerges(t *testing.T) {
	commits := []gitrepo.Commit{
		{SHA: "a", Parents: []string{"b", "c"}},
		{SHA: "b", Parents: []string{"d"}},
		{SHA: "c", Parents: []string{"d"}},
		{SHA: "d"},
	}
	graph := newCommitGraph("", commits, nil)
	if graph.Lanes != 2 {
		t.Fatalf("lanes = %d; want 2", graph.Lanes)
	}
	// The merge's first parent continues the node's lane; the second parent
	// takes a lane of its own.
	var mergeEdge *commitGraphEdge
	for i := range graphRowSpans(graph, 0) {
		if graphRowSpans(graph, 0)[i].Merge {
			mergeEdge = &graphRowSpans(graph, 0)[i]
		}
	}
	if mergeEdge == nil {
		t.Fatal("merge edge missing")
	}
	if mergeEdge.Start != 0 || mergeEdge.End != 2 || mergeEdge.ViaLane != 1 {
		t.Errorf("merge edge = %+v; want rows 0..2 in lane 1", *mergeEdge)
	}
	// Row 1 carries the merge edge, the edge from the merge into b, and the
	// edge from b toward d.
	if got := len(graphRowSpans(graph, 1)); got != 3 {
		t.Errorf("row 1 spans = %d; want 3", got)
	}
	if graph.Rows[3].Lane != 0 {
		t.Errorf("row 3 lane = %d; want 0", graph.Rows[3].Lane)
	}
}

func TestCommitGraphMarksMissingParents(t *testing.T) {
	graph := newCommitGraph("", []gitrepo.Commit{{SHA: "a", Parents: []string{"gone"}}}, nil)
	if !graph.Rows[0].Boundary {
		t.Error("a parent outside the loaded list should mark the boundary")
	}
	if len(graphRowSpans(graph, 0)) != 0 {
		t.Errorf("spans = %+v; want none for a missing parent", graphRowSpans(graph, 0))
	}
}

func TestLaneColorCycles(t *testing.T) {
	if laneColor(nil, 0) != lanePalette[0] || laneColor(nil, len(lanePalette)) != lanePalette[0] {
		t.Error("lane colors should cycle through the palette")
	}
	if laneColor(nil, 1) == laneColor(nil, 0) {
		t.Error("adjacent lanes should not share a color")
	}
}

func TestRefLaneColorFollowsName(t *testing.T) {
	tk := theme.Dark()
	palette := lanePaletteOf(tk)
	want := laneColor(palette, laneColorIndex("main", len(palette)))
	got := refLaneColor(palette, "main")
	if got != want {
		t.Fatalf("ref color = %v; want its palette slot %v", got, want)
	}
	if got != refLaneColor(palette, "main") {
		t.Fatal("the same name should keep its color")
	}
	inPalette := false
	for _, color := range palette {
		if color == got {
			inPalette = true
			break
		}
	}
	if !inPalette {
		t.Errorf("ref color %v is not one of the graph's lane colors", got)
	}
}

func TestLaneInkStaysPutOnDarkSurfaces(t *testing.T) {
	dark := theme.Dark()
	if got := laneInk(dark, dark.Blue); got != dark.Blue {
		t.Errorf("dark surface lane ink = %v; want the lane color unchanged", got)
	}
	light := theme.Light()
	if got := laneInk(light, light.Gold); got == light.Gold {
		t.Error("light surface lane ink should darken the gold accent")
	}
	if surfaceLuminance(light.BgContainer) <= surfaceLuminance(dark.BgContainer) {
		t.Error("the light theme's surface should be brighter than the dark one's")
	}
}

func TestLaneInkMeetsLineAndTileContrast(t *testing.T) {
	light := theme.Light()
	ground := light.Blue.Alpha(0.14).Over(light.BgContainer)
	tile := light.FillTertiary.Over(light.BgContainer)
	for _, color := range lanePaletteOf(light) {
		ink := laneInk(light, color)
		if ratio := contrastRatio(ink, ground); ratio < 3.0 {
			t.Errorf("lane %v ink on the selected row = %.2f; want at least 3:1", color, ratio)
		}
		if ratio := contrastRatio(ink, tile); ratio < 3.0 {
			t.Errorf("lane %v ink on the icon tile = %.2f; want at least 3:1", color, ratio)
		}
	}
}

func TestDeepInkMeetsContrastOnLight(t *testing.T) {
	light := theme.Light()
	surface := light.Blue.Alpha(0.14).Over(light.BgContainer)
	ink := deepInk(light.Gold, surface, 4.5)
	if ratio := contrastRatio(ink, surface); ratio < 4.5 {
		t.Errorf("gold ink contrast = %.2f; want at least 4.5:1", ratio)
	}
	if ink == light.Gold {
		t.Error("light surfaces should darken the gold accent before it labels a badge")
	}
	dark := theme.Dark()
	if got := deepInk(dark.Gold, dark.BgContainer, 4.5); got != dark.Gold {
		t.Errorf("dark surface ink = %v; want the color unchanged", got)
	}
}

// A capsule paints its label on its own tint, which is darker than the row,
// so the ink has to clear AA against the tint rather than against the ground.
func TestRefCapsuleInkClearsItsTint(t *testing.T) {
	light := theme.Light()
	ground := light.Blue.Alpha(0.14).Over(light.BgContainer)
	for _, family := range []string{"gold", "success", "geekblue", "blue"} {
		base, ok := refColor(light, family)
		if !ok {
			t.Fatalf("%s: no family color", family)
		}
		style := refCapsuleColors(base, ground)
		if ratio := contrastRatio(style.Ink, style.Surface); ratio < 4.5 {
			t.Errorf("%s label on its capsule = %.2f; want at least 4.5:1", family, ratio)
		}
		if style.Ink == base && contrastRatio(base, style.Surface) < 4.5 {
			t.Errorf("%s kept a base tone that does not pass on its tint", family)
		}
	}
	dark := theme.Dark()
	style := refCapsuleColors(dark.Gold, dark.BgContainer)
	if ratio := contrastRatio(style.Ink, style.Surface); ratio < 4.5 {
		t.Errorf("dark gold label on its capsule = %.2f; want at least 4.5:1", ratio)
	}
	// Gold is bright enough on its own tint, so the dark surface keeps it
	// unchanged; only the saturated accents need lifting.
	if style.Ink != dark.Gold {
		t.Errorf("dark capsule ink = %v; want the gold lane color unchanged", style.Ink)
	}
}

// A neutral capsule has no hue to be recognized by, so its fill has to read as
// a pill on its own: at least 1.15:1 against the row, across the plain, hover
// and selected grounds of every scheme. The label sits on that fill, so it owes
// the usual 4.5:1 there rather than against the row.
func TestNeutralCapsuleReadsOnEveryRow(t *testing.T) {
	for _, th := range appThemes {
		for _, dark := range []bool{false, true} {
			tk := th.config.Tokens(dark)
			rows := map[string]ui.Color{
				"plain":    graphNodeGround(tk, false, false),
				"hover":    graphNodeGround(tk, false, true),
				"selected": graphNodeGround(tk, true, false),
			}
			for name, ground := range rows {
				style := neutralCapsule(tk, ground)
				if ratio := contrastRatio(style.Surface, ground); ratio < 1.15 {
					t.Errorf("%s dark=%v %s: capsule fill on its row = %.2f; want at least 1.15:1", th.key, dark, name, ratio)
				}
				if ratio := contrastRatio(style.Border, ground); ratio < capsuleBorderMin {
					t.Errorf("%s dark=%v %s: capsule edge on its row = %.2f; want at least %.2f:1", th.key, dark, name, ratio, capsuleBorderMin)
				}
				if ratio := contrastRatio(style.Ink, style.Surface); ratio < 4.5 {
					t.Errorf("%s dark=%v %s: label on the capsule = %.2f; want at least 4.5:1", th.key, dark, name, ratio)
				}
			}
		}
	}
}

// Every capsule sits on one of three row grounds: plain, hover and selected.
// The selected row is the mixed surface -- the accent tint over the container
// -- and it is the one a label has to survive in all four schemes and both
// modes. This walks every lane color on every ground so a future palette
// change cannot leave one chip readable only on plain rows.
func TestCapsulesReadOnEveryMixedGround(t *testing.T) {
	for _, th := range appThemes {
		for _, dark := range []bool{false, true} {
			tk := th.config.Tokens(dark)
			a := &app{palette: th.key}
			grounds := map[string]ui.Color{
				"plain":    graphNodeGround(tk, false, false),
				"hover":    graphNodeGround(tk, false, true),
				"selected": graphNodeGround(tk, true, false),
			}
			for name, ground := range grounds {
				for _, color := range a.lanePalette(tk) {
					style := refCapsuleColors(color, ground)
					if ratio := contrastRatio(style.Ink, style.Surface); ratio < 4.5 {
						t.Errorf("%s dark=%v %s: lane %v label on its capsule = %.2f; want at least 4.5:1",
							th.key, dark, name, color, ratio)
					}
					// The edge has to separate by luminance, not only by hue,
					// even where the lane color sits close to the row's tint.
					if ratio := contrastRatio(style.Border, ground); ratio < capsuleBorderMin {
						t.Errorf("%s dark=%v %s: lane %v capsule edge on its row = %.2f; want at least %.2f:1",
							th.key, dark, name, color, ratio, capsuleBorderMin)
					}
				}
				neutral := neutralCapsule(tk, ground)
				if ratio := contrastRatio(neutral.Ink, neutral.Surface); ratio < 4.5 {
					t.Errorf("%s dark=%v %s: neutral label on its capsule = %.2f; want at least 4.5:1",
						th.key, dark, name, ratio)
				}
				if ratio := contrastRatio(neutral.Surface, ground); ratio < 1.15 {
					t.Errorf("%s dark=%v %s: neutral capsule fill on its row = %.2f; want at least 1.15:1",
						th.key, dark, name, ratio)
				}
			}
		}
	}
}

func TestFilledTagInkKeepsHueAndContrast(t *testing.T) {
	dark := theme.Dark()
	if _, ok := filledTagInk(dark, "success"); ok {
		t.Error("dark surfaces keep the tag's own color")
	}
	light := theme.Light()
	ground := light.Blue.Alpha(0.14).Over(light.BgContainer)
	for _, family := range []string{"success", "error", "warning", "info"} {
		ink, ok := filledTagInk(light, family)
		if !ok {
			t.Errorf("%s: light surfaces should override the tag color", family)
			continue
		}
		// The rendered tag adds its own 10% tint over the row, which darkens
		// the ground; the 5.2 target on the row keeps the real pair above AA.
		tinted := familyFill(light, family).Over(ground)
		if ratio := contrastRatio(ink, tinted); ratio < 4.5 {
			t.Errorf("%s contrast on its own fill = %.2f; want at least 4.5:1", family, ratio)
		}
	}
}

// familyFill is the tint tagColorsFor uses for a filled status tag.
func familyFill(tk *theme.Tokens, family string) ui.Color {
	name := strings.ToUpper(family[:1]) + family[1:]
	fill, _ := tk.Color(name + "FillTertiary")
	return fill
}

func TestDeepInkPreservesFamilyHue(t *testing.T) {
	light := theme.Light()
	surface := light.Blue.Alpha(0.14).Over(light.BgContainer)
	ink := deepInk(light.Success, surface, 5.2)
	if ratio := contrastRatio(ink, surface); ratio < 5.2 {
		t.Errorf("contrast = %.2f; want at least 5.2:1", ratio)
	}
	// Pulling toward body text would leave red and green nearly equal; a hue
	// that stays green keeps G above R and B.
	if !(ink.G > ink.R && ink.G > ink.B) {
		t.Errorf("ink = %v; want a green-dominant tone", ink)
	}
}

func TestGraphInkKeepsLaneLinesOnLight(t *testing.T) {
	light := theme.Light()
	ground := light.Blue.Alpha(0.14).Over(light.BgContainer)
	for _, color := range lanePaletteOf(light) {
		ink := graphInk(light, color)
		if ratio := contrastRatio(ink, ground); ratio < 3.0 {
			t.Errorf("lane %v contrast = %.2f; want at least 3:1", color, ratio)
		}
	}
	dark := theme.Dark()
	color := lanePaletteOf(dark)[0]
	if got := graphInk(dark, color); got != color {
		t.Errorf("dark lanes should keep their accent, got %v want %v", got, color)
	}
}

func TestLaneSeedPrefersBranchesOverTags(t *testing.T) {
	tests := []struct {
		refs, want string
	}{
		{"HEAD -> main, tag: v1", "main"},
		{"origin/main, tag: v1", "origin/main"},
		{"tag: v1", "v1"},
		{"HEAD", "abc"},
		{"", "abc"},
	}
	for _, tt := range tests {
		if got := laneSeed(gitrepo.Commit{SHA: "abc", Refs: tt.refs}); got != tt.want {
			t.Errorf("laneSeed(%q) = %q; want %q", tt.refs, got, tt.want)
		}
	}
}

func TestCommitGraphKeepsBranchColorWhenFiltered(t *testing.T) {
	full := newCommitGraph("", []gitrepo.Commit{
		{SHA: "a", Refs: "HEAD -> master", Parents: []string{"b"}},
		{SHA: "x", Refs: "feature", Parents: []string{"y"}},
		{SHA: "b"},
		{SHA: "y"},
	}, nil)
	filtered := newCommitGraph("", []gitrepo.Commit{
		{SHA: "a", Refs: "HEAD -> master", Parents: []string{"b"}},
		{SHA: "b"},
	}, nil)
	if full.Rows[0].Color != laneColorIndex("master", len(lanePalette)) {
		t.Errorf("full master color = %d; want %d", full.Rows[0].Color, laneColorIndex("master", len(lanePalette)))
	}
	if filtered.Rows[0].Color != full.Rows[0].Color {
		t.Errorf("filtered master color = %d; full = %d; want the same", filtered.Rows[0].Color, full.Rows[0].Color)
	}
	if full.Rows[2].Color != full.Rows[0].Color {
		t.Errorf("master line color changes between rows: %d then %d", full.Rows[0].Color, full.Rows[2].Color)
	}
	if full.Rows[1].Color != laneColorIndex("feature", len(lanePalette)) {
		t.Errorf("feature color = %d; want %d", full.Rows[1].Color, laneColorIndex("feature", len(lanePalette)))
	}
}

func TestCommitGraphAvoidsNeighboringLaneColorClash(t *testing.T) {
	// Find two branch names whose palette slots collide, then check the
	// second lane steps past the color the first one shows.
	byColor := map[int]string{}
	var first, second string
	for i := 0; i < 100 && second == ""; i++ {
		name := fmt.Sprintf("branch-%d", i)
		if prev, ok := byColor[laneColorIndex(name, len(lanePalette))]; ok {
			first, second = prev, name
		} else {
			byColor[laneColorIndex(name, len(lanePalette))] = name
		}
	}
	if second == "" {
		t.Fatal("no colliding branch names found")
	}
	graph := newCommitGraph("", []gitrepo.Commit{
		{SHA: "a", Refs: first, Parents: []string{"p"}},
		{SHA: "b", Refs: second},
		{SHA: "p"},
	}, nil)
	if graph.Rows[0].Color != laneColorIndex(first, len(lanePalette)) {
		t.Errorf("first lane color = %d; want its own slot %d", graph.Rows[0].Color, laneColorIndex(first, len(lanePalette)))
	}
	if graph.Rows[1].Color == graph.Rows[0].Color {
		t.Errorf("neighboring lanes share color %d", graph.Rows[0].Color)
	}
}

func TestRefBadgeClassification(t *testing.T) {
	a := &app{remotes: []gitrepo.Remote{{Name: "origin"}}}
	tests := []struct {
		ref, label, color string
	}{
		{"tag: v1.0", "v1.0", "gold"},
		{"origin/main", "origin/main", "geekblue"},
		{"feature/login", "feature/login", "success"},
		{"HEAD -> main", "HEAD -> main", "success"},
		{"HEAD", "HEAD", "geekblue"},
	}
	for _, tt := range tests {
		badge := a.refBadge(tt.ref)
		if badge.Label != tt.label || badge.Color != tt.color || badge.Icon == nil {
			t.Errorf("refBadge(%q) = %+v; want %q %q with an icon", tt.ref, badge, tt.label, tt.color)
		}
	}
}

func TestBoundaryInkClearsRowGrounds(t *testing.T) {
	light := theme.Light()
	rows := map[string]ui.Color{
		"plain":    light.BgContainer,
		"selected": light.Blue.Alpha(0.14).Over(light.BgContainer),
		"hovered":  light.FillTertiary.Over(light.BgContainer),
	}
	for name, bg := range rows {
		if ratio := contrastRatio(boundaryInk(light, bg), bg); ratio < 3.0 {
			t.Errorf("%s: dashed tail = %.2f; want at least 3:1", name, ratio)
		}
	}
	dark := theme.Dark()
	if got := boundaryInk(dark, dark.BgContainer); got != dark.TextQuaternary {
		t.Errorf("dark tail = %v; want the quaternary token unchanged", got)
	}
}

// A ref's capsule wears the color of its own line in the graph, so the same
// branch reads as one color in the graph, the history row and the branch list.
// Light surfaces tint the capsule from the lane color and deepen the label
// against that tint; dark flavors keep the exact lane color.
func TestRefBadgesWearTheirLaneColor(t *testing.T) {
	light := catppuccinLatte.tokens(*theme.Light())
	dark := catppuccinMocha.tokens(*theme.Dark())
	ground := light.Blue.Alpha(0.14).Over(light.BgContainer)
	lp, dp := catppuccinLatte.laneColors(), catppuccinMocha.laneColors()
	for _, name := range []string{"main", "feature/login", "origin/main", "v1.2.0"} {
		style := refCapsuleColors(refBadgeColor(lp, nil, 0, historyRefLocal, name), ground)
		if got := contrastRatio(style.Ink, style.Surface); got < 4.5 {
			t.Errorf("light %q label on its capsule %.2f; want at least 4.5", name, got)
		}
		if lane := refLaneColor(lp, name); style.Surface == lane {
			t.Errorf("light %q capsule surface is the raw lane color, not a tint", name)
		}
		darkStyle := refCapsuleColors(refBadgeColor(dp, nil, 0, historyRefLocal, name), dark.BgContainer)
		lane := refLaneColor(dp, name)
		if got := contrastRatio(darkStyle.Ink, darkStyle.Surface); got < 4.5 {
			t.Errorf("dark %q label on its capsule %.2f; want at least 4.5", name, got)
		}
		if contrastRatio(lane, darkStyle.Surface) >= 4.5 {
			if darkStyle.Ink != lane {
				t.Errorf("dark %q ink = %v; want the lane color %v that already passes", name, darkStyle.Ink, lane)
			}
		} else if darkStyle.Ink == lane {
			t.Errorf("dark %q kept a lane color that does not pass on its tint", name)
		}
	}
}

// When two ref names hash to the same color slot, the graph steps the second
// lane to a free color. The label has to follow that step, or it points at the
// first branch's line instead of its own.
func TestGraphRefColorFollowsSteppedLane(t *testing.T) {
	byColor := map[int]string{}
	var first, second string
	for i := 0; i < 100 && second == ""; i++ {
		name := fmt.Sprintf("branch-%d", i)
		if prev, ok := byColor[laneColorIndex(name, len(lanePalette))]; ok {
			first, second = prev, name
		} else {
			byColor[laneColorIndex(name, len(lanePalette))] = name
		}
	}
	if second == "" {
		t.Fatal("no colliding branch names found")
	}
	graph := newCommitGraph("", []gitrepo.Commit{
		{SHA: "a", Refs: first, Parents: []string{"p"}},
		{SHA: "b", Refs: second},
		{SHA: "p"},
	}, nil)
	if got, ok := graph.RefColor(first); !ok || got != laneColor(nil, laneColorIndex(first, len(lanePalette))) {
		t.Errorf("first ref color = %v (ok %v); want its hash slot", got, ok)
	}
	stepped, ok := graph.RefColor(second)
	if !ok {
		t.Fatal("second ref has no recorded lane color")
	}
	if want := graph.LaneColor(graph.Rows[1].Color); stepped != want {
		t.Errorf("second ref color = %v; want the stepped lane color %v", stepped, want)
	}
	if hash := laneColor(nil, laneColorIndex(second, len(lanePalette))); stepped == hash {
		t.Error("second ref kept its hash color even though its lane stepped")
	}
	// The badge color follows the stepped color on light themes.
	lp := catppuccinLatte.laneColors()
	if got := refBadgeColor(lp, graph, 1, historyRefLocal, second); got != stepped {
		t.Errorf("second badge color = %v; want the stepped %v", got, stepped)
	}
	if refBadgeColor(lp, graph, 0, historyRefLocal, first) == refBadgeColor(lp, graph, 1, historyRefLocal, second) {
		t.Error("colliding names still read as one color")
	}
}

// A second branch on a commit whose lane another ref started never allocates a
// lane of its own. Its hash slot would then name a color no line on screen
// shows, so its label takes the color of the line its row carries. A tag is a
// point rather than a line: it keeps the name color the tag list and the ref
// selector use, so the same tag reads the same on every page.
func TestBadgeColorFollowsTheRefKind(t *testing.T) {
	light := catppuccinLatte.tokens(*theme.Light())
	palette := lanePaletteOf(&light)
	graph := newCommitGraph("", []gitrepo.Commit{
		{SHA: "a", Refs: "HEAD -> main", Parents: []string{"b"}},
		{SHA: "b"},
	}, palette)
	tag := ""
	for i := 0; i < 100 && tag == ""; i++ {
		name := fmt.Sprintf("tag-%d", i)
		if laneColorIndex(name, len(lanePalette)) != graph.Rows[1].Color {
			tag = name
		}
	}
	if tag == "" {
		t.Fatal("no tag name outside the row's color slot found")
	}
	if _, ok := graph.RefColor(tag); ok {
		t.Fatal("the tag recorded a lane of its own")
	}
	rowColor := graph.LaneColor(graph.Rows[1].Color)
	if got := refBadgeColor(palette, graph, 1, historyRefLocal, tag); got != rowColor {
		t.Errorf("branch color = %v; want its row's line color %v", got, rowColor)
	}
	if refBadgeColor(palette, graph, 1, historyRefLocal, tag) == refLaneColor(palette, tag) {
		t.Error("branch kept its hash color even though the row draws another color")
	}
	// A tag ignores the row's line and keeps the color its own list shows.
	if got, want := refBadgeColor(palette, graph, 1, historyRefTag, tag), refLaneColor(palette, tag); got != want {
		t.Errorf("tag color = %v; want the tag's name color %v", got, want)
	}
	if refBadgeColor(palette, graph, 1, historyRefTag, tag) == rowColor {
		t.Error("tag wears its row's line color instead of its name color")
	}
	// The branch that started the lane keeps its own color on every row.
	if got, want := refBadgeColor(palette, graph, 0, historyRefLocal, "main"), refLaneColor(palette, "main"); got != want {
		t.Errorf("main's color = %v; want its lane color %v", got, want)
	}
	if _, ok := graph.RowLaneColor(2); ok {
		t.Error("a row past the list reported a lane color")
	}
	// With the graph hidden there is no line to follow; every label keeps the
	// name color the branch and tag lists use.
	if got, want := refBadgeColor(palette, nil, 0, historyRefLocal, tag), refLaneColor(palette, tag); got != want {
		t.Errorf("color without a graph = %v; want the name color %v", got, want)
	}
}

func TestGraphNodeGroundFollowsRow(t *testing.T) {
	light := theme.Light()
	selected := light.Blue.Alpha(0.14).Over(light.BgContainer)
	if got := graphNodeGround(light, true, false); got != selected {
		t.Errorf("selected ground = %v; want the selected row tint %v", got, selected)
	}
	// A selected row stays selected while the pointer is over it.
	if got := graphNodeGround(light, true, true); got != selected {
		t.Errorf("selected+hover ground = %v; want the selected row tint %v", got, selected)
	}
	hover := light.FillTertiary.Over(light.BgContainer)
	if got := graphNodeGround(light, false, true); got != hover {
		t.Errorf("hover ground = %v; want %v", got, hover)
	}
	if got := graphNodeGround(light, false, false); got != light.BgContainer {
		t.Errorf("plain ground = %v; want the container %v", got, light.BgContainer)
	}
}

func TestGraphWidthContainsEveryLane(t *testing.T) {
	for lanes := 1; lanes <= 40; lanes++ {
		width := graphWidth(lanes)
		// The selected node's halo is 18 wide and sits on the last lane.
		right := laneX(0, lanes-1) + 9
		if right > width {
			t.Errorf("%d lanes: node edge %.0f outside the %.0f wide column", lanes, right, width)
		}
		if width < 14*float32(lanes) {
			t.Errorf("%d lanes: width %.0f does not cover the lanes", lanes, width)
		}
	}
	if got := graphWidth(0); got <= 0 {
		t.Errorf("empty graph width = %.0f; want a spacer", got)
	}
	// A small graph keeps the width its single lane needs.
	graph := newCommitGraph("", []gitrepo.Commit{{SHA: "a"}}, nil)
	if want := float32(16 + graphLaneGap); graph.Width != want {
		t.Errorf("single lane width = %.0f; want %.0f", graph.Width, want)
	}
}

// A narrow list packs the lanes instead of letting the graph take half the
// row; a graph that already fits keeps its standard gap.
func TestANarrowGraphPacksItsLanes(t *testing.T) {
	g := &commitGraph{Lanes: 8, Width: graphWidth(8), Gap: graphLaneGap}
	const budget = 144 // Eight 16-DIP lanes plus the graph's horizontal padding.
	g.FitWidth(budget)
	if g.Width > budget {
		t.Errorf("the graph takes %v of a %d-DIP budget", g.Width, budget)
	}
	if g.Gap >= graphLaneGap {
		t.Errorf("the graph kept its wide %v gap", g.Gap)
	}
	if want := 16 + 8*g.Gap; g.Width != want {
		t.Errorf("graph width %v does not match its gap %v (want %v)", g.Width, g.Gap, want)
	}

	wide := &commitGraph{Lanes: 2, Width: graphWidth(2), Gap: graphLaneGap}
	wide.FitWidth(400)
	if wide.Gap != graphLaneGap || wide.Width != graphWidth(2) {
		t.Errorf("a graph with room changed: gap %v width %v", wide.Gap, wide.Width)
	}
}

func TestBlameGutterInkClearsRows(t *testing.T) {
	light := theme.Light()
	rows := map[string]ui.Color{
		"dialog surface": light.BgElevated,
		"hovered row":    light.FillTertiary.Over(light.BgElevated),
	}
	for name, bg := range rows {
		if ratio := contrastRatio(gutterInk(light, bg), bg); ratio < 4.5 {
			t.Errorf("%s: blame date and line number = %.2f; want at least 4.5:1", name, ratio)
		}
	}
	if got := gutterInk(theme.Dark(), theme.Dark().BgElevated); got != theme.Dark().TextTertiary {
		t.Errorf("dark blame gutter = %v; want the tertiary token unchanged", got)
	}
}

func TestGutterInkClearsDiffRows(t *testing.T) {
	light := theme.Light()
	a := &app{}
	rows := map[string]ui.Color{
		"plain":   light.BgContainer,
		"added":   a.diffColors(light, "add").background,
		"removed": a.diffColors(light, "del").background,
	}
	for name, bg := range rows {
		if ratio := contrastRatio(gutterInk(light, bg), bg); ratio < 4.5 {
			t.Errorf("%s: line number = %.2f; want at least 4.5:1", name, ratio)
		}
	}
	dark := theme.Dark()
	if got := gutterInk(dark, dark.BgContainer); got != dark.TextTertiary {
		t.Errorf("dark gutter = %v; want the tertiary token unchanged", got)
	}
}

// inkSaturation is the HSV saturation of a color, 0 (grey) to 1 (fully
// saturated). A label that passes contrast by turning grey has stopped
// pointing at its lane.
func inkSaturation(c ui.Color) float64 {
	hi := float64(max(c.R, c.G, c.B))
	lo := float64(min(c.R, c.G, c.B))
	if hi == 0 {
		return 0
	}
	return (hi - lo) / hi
}

// Deepening a lane color for small text must keep the hue on every light
// scheme: contrast alone is not enough when the badge then reads as grey and
// the branch loses the line it names. The floor comes from the schemes' own
// darkest accents (Lobe's cyan), which still keep about 0.39.
func TestRefLabelInkKeepsHueOnEveryLightScheme(t *testing.T) {
	const minSaturation = 0.35
	checked := 0
	for _, th := range appThemes {
		light := th.config.Tokens(false)
		if light.Dark {
			t.Fatalf("%s: light tokens report dark", th.key)
		}
		ground := selectionBg(light, light.BgContainer)
		for _, color := range th.lanesLight {
			style := refCapsuleColors(color, ground)
			if ratio := contrastRatio(style.Ink, style.Surface); ratio < 4.5 {
				t.Errorf("%s: lane %v label on its capsule = %.2f; want at least 4.5:1", th.key, color, ratio)
			}
			if sat := inkSaturation(style.Ink); sat < minSaturation {
				t.Errorf("%s: lane %v ink saturation = %.2f; want at least %.2f", th.key, color, sat, minSaturation)
			}
			checked++
		}
		for _, family := range []string{"gold", "success", "geekblue", "blue", "purple", "volcano"} {
			base, ok := refColor(light, family)
			if !ok {
				t.Fatalf("%s: no family color for %s", th.key, family)
			}
			style := refCapsuleColors(base, ground)
			if ratio := contrastRatio(style.Ink, style.Surface); ratio < 4.5 {
				t.Errorf("%s: %s label on its capsule = %.2f; want at least 4.5:1", th.key, family, ratio)
			}
			if sat := inkSaturation(style.Ink); sat < 0.3 {
				t.Errorf("%s: %s badge saturation = %.2f; want at least 0.30", th.key, family, sat)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no light scheme was checked")
	}
}

// The dark flavors' pastel lanes mostly pass on their own tint; the saturated
// reds and blues do not, so they lift toward white until AA. The lift has to
// stay inside the hue: a lane that turns white stops naming its line. The floor
// comes from the schemes' own pastel pink, which is 0.21 saturated.
func TestRefCapsuleInkKeepsHueOnDarkSchemes(t *testing.T) {
	const minSaturation = 0.2
	checked, lifted := 0, 0
	for _, th := range appThemes {
		dark := th.config.Tokens(true)
		if !dark.Dark {
			t.Fatalf("%s: dark tokens report light", th.key)
		}
		for _, color := range th.lanesDark {
			style := refCapsuleColors(color, dark.BgContainer)
			if ratio := contrastRatio(style.Ink, style.Surface); ratio < 4.5 {
				t.Errorf("%s: lane %v label on its capsule = %.2f; want at least 4.5:1", th.key, color, ratio)
			}
			if sat := inkSaturation(style.Ink); sat < minSaturation {
				t.Errorf("%s: lane %v ink saturation = %.2f; want at least %.2f", th.key, color, sat, minSaturation)
			}
			if style.Ink != color {
				lifted++
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no dark scheme was checked")
	}
	if lifted == 0 {
		t.Error("no dark lane was lifted; the contrast case is no longer exercised")
	}
}

// Every ref capsule must point at a color the reader can see on its row: the
// row's own node, or a line crossing it. A badge naming a color with no line
// behind it sends the reader looking for a lane that is not there. A tag is the
// exception -- it is a point, not a line -- and keeps its name color.
func TestEveryRefPointsAtAColorOnItsRow(t *testing.T) {
	a := &app{remotes: []gitrepo.Remote{{Name: "origin"}}}
	commits := []gitrepo.Commit{
		{SHA: "a", Refs: "HEAD -> main, origin/main, tag: v1.0", Parents: []string{"b", "c"}},
		{SHA: "b", Refs: "feature/x", Parents: []string{"d"}},
		{SHA: "c", Refs: "feature/y", Parents: []string{"d"}},
		{SHA: "d", Refs: "origin/dev, tag: v2.0"},
	}
	graph := newCommitGraph("", commits, lanePalette)
	checked := 0
	for row := range graph.Rows {
		drawn := map[ui.Color]bool{}
		if color, ok := graph.RowLaneColor(row); ok {
			drawn[color] = true
		}
		for span := range graph.spans(row) {
			drawn[graph.LaneColor(span.Color)] = true
		}
		for _, ref := range commitRefs(commits[row].Refs) {
			kind, name, ok := a.parseCommitRef(ref)
			if !ok {
				continue
			}
			color := refBadgeColor(lanePalette, graph, row, kind, name)
			if kind == historyRefTag {
				if want := refLaneColor(lanePalette, name); color != want {
					t.Errorf("row %d tag %q wears %v; want its name color %v", row, name, color, want)
				}
				checked++
				continue
			}
			if !drawn[color] {
				t.Errorf("row %d ref %q points at %v, which no line on the row draws", row, name, color)
			}
			if want, ok := graph.RefColor(name); ok && color != want {
				t.Errorf("row %d ref %q wears %v; the graph drew its line with %v", row, name, color, want)
			}
			checked++
		}
	}
	if checked < 7 {
		t.Fatalf("checked only %d refs; the fixture should exercise branches, remotes and tags", checked)
	}
}

// The graph's two display presets round-trip through their names, and an
// unknown name (or an out-of-range width) falls back to the standard style.
func TestGraphStylePresets(t *testing.T) {
	for _, key := range []string{"thin", "standard", "thick"} {
		if got := graphLineKey(graphLineOf(key)); got != key {
			t.Errorf("graphLineKey(graphLineOf(%q)) = %q", key, got)
		}
	}
	for _, key := range []string{"small", "standard", "large"} {
		if got := graphNodeKey(graphNodeOf(key)); got != key {
			t.Errorf("graphNodeKey(graphNodeOf(%q)) = %q", key, got)
		}
	}
	if graphLineOf("neon") != graphLineStandard || graphNodeOf("huge") != graphNodeStandard {
		t.Error("unknown presets should fall back to standard")
	}
	var zero app
	line, node := zero.graphStyle()
	if line != graphLineStandard || node != graphNodeStandard {
		t.Fatalf("zero app drew line=%v node=%v; want standard", line, node)
	}
	styled := app{historyLine: graphLineThick, historyNode: graphNodeLarge}
	line, node = styled.graphStyle()
	if line != graphLineThick || node != graphNodeLarge {
		t.Fatalf("styled app drew line=%v node=%v; want thick/large", line, node)
	}
}
