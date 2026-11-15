package lobedesk

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// The commit's branch picker keeps the keyboard in its search box, so Up/Down
// have to move a highlight of its own; the row they mark is what Enter takes.
func TestStepCheckoutCursorSkipsTheCurrentBranch(t *testing.T) {
	matches := []gitrepo.Branch{{Name: "main", Current: true}, {Name: "feat"}, {Name: "fix"}}
	for _, test := range []struct {
		name   string
		cursor int
		delta  int
		want   int
	}{
		{"down from no highlight takes the first choice", -1, 1, 1},
		{"up from no highlight takes the last choice", -1, -1, 2},
		{"down skips the current branch", 0, 1, 1},
		{"down steps to the next row", 1, 1, 2},
		{"down wraps to the top", 2, 1, 1},
		{"up wraps to the bottom", 1, -1, 2},
	} {
		if got := stepCheckoutCursor(matches, test.cursor, test.delta); got != test.want {
			t.Errorf("%s: stepCheckoutCursor(%d, %d) = %d; want %d", test.name, test.cursor, test.delta, got, test.want)
		}
	}
	if got := stepCheckoutCursor([]gitrepo.Branch{{Name: "main", Current: true}}, -1, 1); got != -1 {
		t.Errorf("a list of only the current branch left the cursor at %d", got)
	}
	if got := stepCheckoutCursor(nil, -1, 1); got != -1 {
		t.Errorf("an empty list left the cursor at %d", got)
	}
}

func TestCheckoutBranchAtTakesTheHighlightedRow(t *testing.T) {
	matches := []gitrepo.Branch{{Name: "main", Current: true}, {Name: "feat"}, {Name: "fix"}}
	if name, ok := checkoutBranchAt(matches, 2); !ok || name != "fix" {
		t.Errorf("cursor on fix gave %q (%v); want fix", name, ok)
	}
	// Before the arrows are touched -- or while the highlight sits on the
	// current branch -- Enter still means the first choice.
	if name, ok := checkoutBranchAt(matches, 0); !ok || name != "feat" {
		t.Errorf("cursor on the current branch gave %q (%v); want the first choice feat", name, ok)
	}
	if name, ok := checkoutBranchAt(matches, -1); !ok || name != "feat" {
		t.Errorf("no highlight gave %q (%v); want feat", name, ok)
	}
	if _, ok := checkoutBranchAt([]gitrepo.Branch{{Name: "main", Current: true}}, -1); ok {
		t.Error("a list with only the current branch offered a choice")
	}
}

// The Keys go to the focused search field, so the dialog's own shortcuts have
// to see them first; otherwise Down would scroll the list or do nothing.
func TestCheckoutDialogArrowsMoveTheHighlight(t *testing.T) {
	a := &app{
		locale: localeChinese,
		branches: []gitrepo.Branch{
			{Name: "main", SHA: "aaaa", Current: true},
			{Name: "feat", SHA: "aaaa"},
			{Name: "fix", SHA: "aaaa"},
		},
	}
	a.openCheckoutBranchDialog(gitrepo.Commit{SHA: "aaaa", Subject: "pick one"})
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(520).Height(600).Children(func() {
			a.checkoutBranchDialog(c)
		})
	}, 520, 600)
	if got := a.checkoutBranch.cursor; got != 1 {
		t.Fatalf("opening the picker highlighted row %d; want the first choice (feat) at 1", got)
	}
	if a.checkoutBranch.reveal {
		t.Error("the first frame kept asking to reveal the highlight after drawing it")
	}
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if got := a.checkoutBranch.cursor; got != 2 {
		t.Errorf("Down left the highlight at %d; want fix at 2", got)
	}
	if a.checkoutBranch.reveal {
		t.Error("the reveal request outlived the frame that scrolled to the highlight")
	}
	tt.Key(0, ui.KeyUp)
	tt.Frame()
	if got := a.checkoutBranch.cursor; got != 1 {
		t.Errorf("Up left the highlight at %d; want feat at 1", got)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.checkoutBranch.open {
		t.Error("Enter did not take the highlighted branch and close the picker")
	}
}

// Typing narrows the list; the highlight moves to the first branch the new
// query leaves, so Enter takes a row that is actually on screen.
func TestCheckoutDialogQueryReplacesTheHighlight(t *testing.T) {
	a := &app{
		locale: localeChinese,
		branches: []gitrepo.Branch{
			{Name: "main", SHA: "aaaa", Current: true},
			{Name: "feat", SHA: "aaaa"},
			{Name: "fix", SHA: "aaaa"},
		},
	}
	a.openCheckoutBranchDialog(gitrepo.Commit{SHA: "aaaa", Subject: "pick one"})
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(520).Height(600).Children(func() {
			a.checkoutBranchDialog(c)
		})
	}, 520, 600)
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	tt.Type("fix")
	tt.Frame()
	if got := a.checkoutBranch.query; got != "fix" {
		t.Fatalf("typing left the query at %q; want fix", got)
	}
	if got := a.checkoutBranch.cursor; got != 0 {
		t.Errorf("the highlight sits at %d; want the only remaining branch at 0", got)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.checkoutBranch.open {
		t.Error("Enter on the narrowed list did not close the picker")
	}
}

// A row click or Tab moves the focus off the search box, which would leave the
// arrows and typing dead; the dialog asks for the box back on every frame so a
// click cannot quietly disarm the keyboard.
func TestCheckoutDialogKeepsTheKeyboardInTheSearch(t *testing.T) {
	a := &app{
		locale: localeChinese,
		branches: []gitrepo.Branch{
			{Name: "main", SHA: "aaaa", Current: true},
			{Name: "feat", SHA: "aaaa"},
			{Name: "fix", SHA: "aaaa"},
		},
	}
	a.openCheckoutBranchDialog(gitrepo.Commit{SHA: "aaaa", Subject: "pick one"})
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(520).Height(600).Children(func() {
			a.checkoutBranchDialog(c)
		})
	}, 520, 600)
	row, ok := tt.Find("main")
	if !ok {
		t.Fatal("the current branch's row is not laid out")
	}
	// Clicking the current branch is not a choice, but it does move the focus.
	tt.ClickAt(row.X+row.W/2, row.Y+row.H/2)
	tt.Frame()
	if !a.checkoutBranch.open {
		t.Fatal("clicking the current branch closed the picker")
	}
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if got := a.checkoutBranch.cursor; got != 2 {
		t.Errorf("after a row click, Down left the highlight at %d; want fix at 2", got)
	}
	tt.Type("fi")
	tt.Frame()
	if got := a.checkoutBranch.query; got != "fi" {
		t.Errorf("after a row click, typing left the query at %q; want fi", got)
	}
}

// A filter that leaves only the branch already checked out has nothing to
// offer. The list says so, instead of leaving the arrows and Enter dead with
// no explanation.
func TestCheckoutDialogExplainsWhenOnlyTheCurrentBranchRemains(t *testing.T) {
	a := &app{
		locale: localeChinese,
		branches: []gitrepo.Branch{
			{Name: "main", SHA: "aaaa", Current: true},
			{Name: "feat", SHA: "aaaa"},
		},
	}
	a.openCheckoutBranchDialog(gitrepo.Commit{SHA: "aaaa", Subject: "pick one"})
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(520).Height(600).Children(func() {
			a.checkoutBranchDialog(c)
		})
	}, 520, 600)
	if _, ok := tt.Find(a.t("checkoutNoOtherBranch")); ok {
		t.Fatal("the hint appeared while another branch could be checked out")
	}
	tt.Type("main")
	tt.Frame()
	if _, ok := tt.Find(a.t("checkoutNoOtherBranch")); !ok {
		t.Error("filtering to the current branch left the picker with no explanation")
	}
	if got := a.checkoutBranch.cursor; got != -1 {
		t.Errorf("the highlight sits at %d; want none when only the current branch matches", got)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if !a.checkoutBranch.open {
		t.Error("Enter on the only current branch closed the picker")
	}
}

// A reload can shorten the branch list while the picker is open -- an
// auto-fetch finishing, or branches being deleted elsewhere. The highlight is
// moved to a row that still exists, and the list has to scroll to it: Enter
// acts on the highlighted row, so leaving it off screen would check out a
// branch the reader cannot see.
func TestCheckoutDialogRevealsTheFixedHighlightAfterAShrink(t *testing.T) {
	branches := make([]gitrepo.Branch, 0, 40)
	for i := 0; i < 40; i++ {
		branches = append(branches, gitrepo.Branch{Name: fmt.Sprintf("br%02d", i), SHA: "aaaa"})
	}
	a := &app{locale: localeChinese, branches: branches}
	a.openCheckoutBranchDialog(gitrepo.Commit{SHA: "aaaa", Subject: "pick one"})
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(520).Height(600).Children(func() {
			a.checkoutBranchDialog(c)
		})
	}, 520, 600)
	for i := 0; i < 30; i++ {
		tt.Key(0, ui.KeyDown)
		tt.Frame()
	}
	if got := a.checkoutBranch.cursor; got != 30 {
		t.Fatalf("Down thirty times left the highlight at %d; want 30", got)
	}
	a.branches = branches[:25]
	tt.Frame()
	if got := a.checkoutBranch.cursor; got != 0 {
		t.Fatalf("after the list shrank, the highlight sits at %d; want the first branch at 0", got)
	}
	row, ok := tt.Find("br00")
	if !ok || row.W <= 0 || row.H <= 0 {
		t.Fatalf("the first branch is off screen after the list shrank: %+v (%v)", row, ok)
	}
}
