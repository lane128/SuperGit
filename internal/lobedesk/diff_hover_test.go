package lobedesk

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

func TestDiffHoverActionLeftTracksTheVisibleBand(t *testing.T) {
	for _, test := range []struct {
		name     string
		patch    float32
		viewport float32
		scrollX  float32
		want     float32
	}{
		{"a patch that fits ends at the row", 600, 700, 0, 566},
		{"a wide patch ends at the pane's right edge", 2000, 700, 0, 666},
		{"scrolling right moves the action with the pane", 2000, 700, 500, 1166},
		{"an unmeasured pane falls back to the patch", 2000, 0, 0, 1966},
		{"a tiny row never goes negative", 10, 0, 0, 0},
		{"a scrolled tiny row stays at the start", 10, 700, 300, 0},
	} {
		a := &app{diffViewportRect: ui.Rect{W: test.viewport}}
		a.diffScroll.X = test.scrollX
		if got := a.diffHoverActionLeft(test.patch); got != test.want {
			t.Errorf("%s: diffHoverActionLeft(%v) = %v; want %v", test.name, test.patch, got, test.want)
		}
	}
}

// A long line makes the patch wider than the pane. The hover action has to
// stay inside the pane: at the end of the patch row it would sit past the
// horizontal scroll and no click could reach it.
func TestDiffHoverActionStaysInsideThePane(t *testing.T) {
	const paneWidth = 1000.0
	long := strings.Repeat("wide ", 60)
	hunkText := "@@ -1,3 +1,4 @@ " + long
	a := &app{
		locale:       localeChinese,
		repo:         &gitrepo.Repository{Root: t.TempDir()},
		selectedPath: "long.go",
		status: gitrepo.Status{Entries: []gitrepo.Entry{
			{Path: "long.go", Index: '.', Worktree: 'M'},
		}},
		diff: gitrepo.Diff{
			Text:          "a patch",
			CanStageHunks: true,
			Hunks:         []gitrepo.Hunk{{Header: hunkText}},
			Lines: []gitrepo.DiffLine{
				{Kind: "hunk", Text: hunkText, Hunk: 0},
				{Kind: "add", Text: long, New: 1},
			},
		},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(paneWidth).Height(700).Children(func() {
			a.diffBody(c)
		})
	}, paneWidth, 700)
	row, ok := tt.Find(long)
	if !ok {
		t.Fatal("the long row is not laid out")
	}
	tt.Move(row.X+paneWidth/2, row.Y+row.H/2)
	tt.Frame()
	button, ok := tt.Find(a.t("stageLine"))
	if !ok {
		t.Fatal("hovering the long row did not draw its stage button")
	}
	if button.X < 0 || button.X+button.W > paneWidth {
		t.Errorf("the stage button sits at x = %v (width %v), outside the %v pane", button.X, button.W, paneWidth)
	}
	if button.X+button.W < paneWidth-60 {
		t.Errorf("the stage button sits at x = %v, far from the pane's right edge", button.X)
	}

	// A hunk header can carry a long function signature too; its action uses
	// the same band.
	hunkRow, ok := tt.Find(hunkText)
	if !ok {
		t.Fatal("the hunk row is not laid out")
	}
	tt.Move(hunkRow.X+50, hunkRow.Y+hunkRow.H/2)
	tt.Frame()
	hunkButton, ok := tt.Find(a.t("stageHunk"))
	if !ok {
		t.Fatal("hovering the long hunk header did not draw its stage button")
	}
	if hunkButton.X < 0 || hunkButton.X+hunkButton.W > paneWidth {
		t.Errorf("the hunk's stage button sits at x = %v (width %v), outside the %v pane", hunkButton.X, hunkButton.W, paneWidth)
	}
}

// A tall patch draws a scroll bar over the pane's right edge, and the hunk row
// centers its children unless it is told to start. Both presses have to clear
// the bar: its hit area takes the right edge of anything under it, so the part
// of the button that overlaps it cannot be clicked.
func TestDiffHoverActionsClearTheScrollBar(t *testing.T) {
	const (
		paneWidth  = 700.0
		paneHeight = 700.0
		barWidth   = 12.0
	)
	// The patch is wider than the pane, so the action is pinned to the pane's
	// right edge, and tall enough for the scroll bar to be drawn over it.
	long := strings.Repeat("wide ", 60)
	hunkText := "@@ -1,80 +1,80 @@ " + long
	lines := []gitrepo.DiffLine{{Kind: "hunk", Text: hunkText, Hunk: 0}}
	for i := 1; i <= 80; i++ {
		lines = append(lines, gitrepo.DiffLine{Kind: "add", Text: fmt.Sprintf("line %d %s", i, long), New: i})
	}
	a := &app{
		locale:       localeChinese,
		repo:         &gitrepo.Repository{Root: t.TempDir()},
		selectedPath: "long.go",
		status: gitrepo.Status{Entries: []gitrepo.Entry{
			{Path: "long.go", Index: '.', Worktree: 'M'},
		}},
		diff: gitrepo.Diff{
			Text:          "a patch",
			CanStageHunks: true,
			Hunks:         []gitrepo.Hunk{{Header: hunkText}},
			Lines:         lines,
		},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(paneWidth).Height(paneHeight).Children(func() {
			a.diffBody(c)
		})
	}, paneWidth, paneHeight)

	check := func(rowText, action string) {
		t.Helper()
		row, ok := tt.Find(rowText)
		if !ok {
			t.Fatalf("%q is not laid out", rowText)
		}
		tt.Move(row.X+20, row.Y+row.H/2)
		tt.Frame()
		button, ok := tt.Find(action)
		if !ok {
			t.Fatalf("hovering %q did not draw %q", rowText, action)
		}
		if button.X+button.W > paneWidth-barWidth {
			t.Errorf("the %q action ends at x = %v, under the %v-wide scroll bar", action, button.X+button.W, barWidth)
		}
	}
	check(hunkText, a.t("stageHunk"))
	check(fmt.Sprintf("line 7 %s", long), a.t("stageLine"))
}

// A wheel changes the horizontal scroll after the rows have placed their hover
// actions from the old value. The pane has to ask for one more frame so the
// action catches up at once; asking again on the same value would redraw
// forever.
func TestDiffHoverScrollCatchUpAsksForOneFrame(t *testing.T) {
	a := &app{}
	if a.syncDiffHoverScroll() {
		t.Error("the first frame asked for a catch-up with nothing to catch up to")
	}
	a.diffScroll.X = 120
	if !a.syncDiffHoverScroll() {
		t.Error("a moved horizontal scroll did not ask for a catch-up frame")
	}
	if a.syncDiffHoverScroll() {
		t.Error("the same scroll asked for another frame, so the pane would redraw forever")
	}
	a.diffScroll.X = 40
	if !a.syncDiffHoverScroll() {
		t.Error("scrolling back did not ask for a catch-up frame")
	}
}

// A dirty submodule's diff is only the "Subproject commit X-dirty" pseudo
// patch. git apply accepts it and changes nothing, so the pane must not offer
// a hunks-and-lines staging action that would report a false success; the file
// row is the way to stage a submodule.
func TestDirtySubmoduleOffersNoHunkAction(t *testing.T) {
	a := &app{
		locale:       localeChinese,
		repo:         &gitrepo.Repository{Root: t.TempDir()},
		selectedPath: "sub",
		status: gitrepo.Status{Entries: []gitrepo.Entry{
			{Path: "sub", Index: ' ', Worktree: 'M', Submodule: "S.M."},
		}},
		diff: gitrepo.Diff{
			Text:          "a patch",
			CanStageHunks: true,
			Hunks:         []gitrepo.Hunk{{Header: "@@ -1 +1 @@"}},
			Lines: []gitrepo.DiffLine{
				{Kind: "hunk", Text: "@@ -1 +1 @@", Hunk: 0},
				{Kind: "del", Text: "-Subproject commit 1111111", Old: 1},
				{Kind: "add", Text: "+Subproject commit 1111111-dirty", New: 1},
			},
		},
	}
	if a.hunkActionsAllowed() {
		t.Error("a dirty submodule offered a hunk action")
	}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(800).Height(500).Children(func() {
			a.diffBody(c)
		})
	}, 800, 500)
	row, ok := tt.Find("@@ -1 +1 @@")
	if !ok {
		t.Fatal("the hunk row is not laid out")
	}
	tt.Move(row.X+20, row.Y+row.H/2)
	tt.Frame()
	if _, ok := tt.Find(a.t("stageHunk")); ok {
		t.Error("hovering a dirty submodule's hunk drew a stage button")
	}
}

// Geometry alone does not prove the action is reachable: after a horizontal
// scroll the rows move under the pointer, and a tall patch draws a scroll bar
// over the pane's right edge whose hit area takes what sits under it. A press
// has to reach the button, which MyGo shows by giving it the focus; cancelling
// the press keeps the handler from staging anything.
func TestDiffHoverActionTakesThePressAfterScrollingRight(t *testing.T) {
	const (
		paneWidth  = 700.0
		paneHeight = 700.0
		barWidth   = 12.0
	)
	long := strings.Repeat("wide ", 60)
	hunkText := "@@ -1,80 +1,80 @@ " + long
	lines := []gitrepo.DiffLine{{Kind: "hunk", Text: hunkText, Hunk: 0}}
	for i := 1; i <= 80; i++ {
		lines = append(lines, gitrepo.DiffLine{Kind: "add", Text: fmt.Sprintf("line %d %s", i, long), New: i})
	}
	a := &app{
		locale:       localeChinese,
		repo:         &gitrepo.Repository{Root: t.TempDir()},
		selectedPath: "long.go",
		status: gitrepo.Status{Entries: []gitrepo.Entry{
			{Path: "long.go", Index: '.', Worktree: 'M'},
		}},
		diff: gitrepo.Diff{
			Text:          "a patch",
			CanStageHunks: true,
			Hunks:         []gitrepo.Hunk{{Header: hunkText}},
			Lines:         lines,
		},
	}
	// Start the pane partway through the patch, as a wheel or drag would.
	a.diffScroll.X = 420
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(paneWidth).Height(paneHeight).Children(func() {
			a.diffBody(c)
		})
	}, paneWidth, paneHeight)
	tt.Frame()
	if a.diffScroll.X == 0 {
		t.Fatal("the pane did not keep the horizontal scroll")
	}
	hunkRow, ok := tt.Find(hunkText)
	if !ok {
		t.Fatal("the hunk row is not laid out")
	}
	tt.Move(paneWidth/2, hunkRow.Y+hunkRow.H/2)
	tt.Frame()
	button, ok := tt.Find(a.t("stageHunk"))
	if !ok {
		t.Fatal("hovering the scrolled hunk row did not draw its stage button")
	}
	if button.X+button.W > paneWidth-barWidth {
		t.Fatalf("the stage button ends at x = %v, under the scroll bar", button.X+button.W)
	}
	// Press near the button's right edge: that is the part the scroll bar's
	// hit area would take if the action did not keep its reserve.
	tt.Press(button.X+button.W-2, button.Y+button.H/2)
	if !tt.Focused(a.t("stageHunk")) {
		t.Error("a press on the hover action did not reach it at this scroll")
	}
	// Focused also reports an ancestor of the focused node, and the hunk row
	// is a button too: pressing anywhere on the row would satisfy the check
	// above. The row must not have the focus for this press to prove the hit
	// box followed the drawn button.
	if tt.Focused(hunkText) {
		t.Error("the press landed on the hunk row, not on the hover action")
	}
	tt.CancelPress()
}
