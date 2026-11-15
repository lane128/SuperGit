package lobedesk

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

func refChoicesForTests() []refChoice {
	return []refChoice{
		{label: "all"},
		{ref: historyRef{Kind: historyRefLocal, Name: "feat"}, label: "feat"},
		{ref: historyRef{Kind: historyRefLocal, Name: "fix"}, label: "fix"},
		{ref: historyRef{Kind: historyRefTag, Name: "v1"}, label: "v1"},
	}
}

func TestRefCursorWrapsAndFollowsTheQuery(t *testing.T) {
	matches := refChoicesForTests()
	for _, test := range []struct {
		name   string
		cursor int
		delta  int
		want   int
	}{
		{"down from no highlight takes the top", -1, 1, 0},
		{"up from no highlight takes the bottom", -1, -1, 3},
		{"down steps", 1, 1, 2},
		{"down wraps", 3, 1, 0},
		{"up wraps", 0, -1, 3},
	} {
		if got := stepRefCursor(matches, test.cursor, test.delta); got != test.want {
			t.Errorf("%s: stepRefCursor(%d, %d) = %d; want %d", test.name, test.cursor, test.delta, got, test.want)
		}
	}
	if got := stepRefCursor(nil, -1, 1); got != -1 {
		t.Errorf("an empty list left the cursor at %d", got)
	}

	// A rebuilt list keeps the ref in use highlighted while the query shows
	// it, and otherwise falls back to the first row.
	if got := refCursorTarget(matches, historyRef{Kind: historyRefLocal, Name: "fix"}); got != 2 {
		t.Errorf("the applied ref landed at %d; want 2", got)
	}
	if got := refCursorTarget(matches, historyRef{Kind: historyRefTag, Name: "gone"}); got != 0 {
		t.Errorf("a hidden applied ref landed at %d; want the first row", got)
	}
	if got := refCursorTarget(nil, historyRef{}); got != -1 {
		t.Errorf("no matches landed at %d; want -1", got)
	}
}

func TestRefChoiceAtFallsBackToTheFirstMatch(t *testing.T) {
	matches := refChoicesForTests()
	if choice, ok := refChoiceAt(matches, 3); !ok || choice.label != "v1" {
		t.Errorf("cursor on v1 gave %q (%v)", choice.label, ok)
	}
	if choice, ok := refChoiceAt(matches, 9); !ok || choice.label != "all" {
		t.Errorf("a stale cursor gave %q (%v); want the first row", choice.label, ok)
	}
	if _, ok := refChoiceAt(nil, -1); ok {
		t.Error("an empty list offered a choice")
	}
}

// The ref selector keeps the keyboard in its search box, so its own shortcuts
// have to move the highlight; Enter takes the row they mark.
func TestHistoryRefPickerArrowsMoveTheHighlight(t *testing.T) {
	a := &app{
		locale: localeChinese,
		branches: []gitrepo.Branch{
			{Name: "feat", SHA: "aaaa"},
			{Name: "fix", SHA: "aaaa"},
		},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(600).Height(500).Children(func() {
			a.historyRefMenu(c)
		})
	}, 600, 500)
	anchor, ok := tt.Find(a.t("allRefs"))
	if !ok {
		t.Fatal("the ref anchor is not laid out")
	}
	tt.ClickAt(anchor.X+anchor.W/2, anchor.Y+anchor.H/2)
	tt.Frame()
	if !a.historyRefOpen {
		t.Fatal("clicking the anchor did not open the ref picker")
	}
	if got := a.historyRefCursor; got != 0 {
		t.Fatalf("the picker opened with row %d highlighted; want the ref in use (all) at 0", got)
	}
	if a.historyRefReveal {
		t.Error("the first frame kept asking to reveal the highlight after drawing it")
	}
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if got := a.historyRefCursor; got != 1 {
		t.Errorf("Down left the highlight at %d; want feat at 1", got)
	}
	if a.historyRefReveal {
		t.Error("the reveal request outlived the frame that scrolled to the highlight")
	}
	tt.Key(0, ui.KeyUp)
	tt.Frame()
	if got := a.historyRefCursor; got != 0 {
		t.Errorf("Up left the highlight at %d; want all at 0", got)
	}
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.historyRefOpen {
		t.Error("Enter did not close the picker")
	}
	if want := (historyRef{Kind: historyRefLocal, Name: "feat"}); a.historyRef != want {
		t.Errorf("Enter applied %+v; want the highlighted feat", a.historyRef)
	}
}

// The ref picker faces the same shrink: refs are rewritten by a background
// refresh, and the highlight it fixes has to be scrolled into view, or Enter
// would take a ref the reader cannot see.
func TestRefPickerRevealsTheFixedHighlightAfterAShrink(t *testing.T) {
	branches := make([]gitrepo.Branch, 0, 40)
	for i := 0; i < 40; i++ {
		branches = append(branches, gitrepo.Branch{Name: fmt.Sprintf("br%02d", i), SHA: "aaaa"})
	}
	a := &app{locale: localeChinese, branches: branches}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(600).Height(500).Children(func() {
			a.historyRefMenu(c)
		})
	}, 600, 500)
	anchor, ok := tt.Find(a.t("allRefs"))
	if !ok {
		t.Fatal("the ref anchor is not laid out")
	}
	tt.ClickAt(anchor.X+anchor.W/2, anchor.Y+anchor.H/2)
	tt.Frame()
	for i := 0; i < 30; i++ {
		tt.Key(0, ui.KeyDown)
		tt.Frame()
	}
	if got := a.historyRefCursor; got != 30 {
		t.Fatalf("Down thirty times left the highlight at %d; want 30", got)
	}
	a.branches = branches[:25]
	tt.Frame()
	if got := a.historyRefCursor; got != 0 {
		t.Fatalf("after the list shrank, the highlight sits at %d; want the all-refs row at 0", got)
	}
	row, ok := tt.AccessByRole(ui.RoleOption, a.t("allRefs"))
	if !ok || row.Bounds.W <= 0 || row.Bounds.H <= 0 || row.Bounds.Y < 0 {
		t.Fatalf("the all-refs row is off screen after the list shrank: %+v (%v)", row.Bounds, ok)
	}
}
