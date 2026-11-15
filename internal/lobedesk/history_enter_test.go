package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// Arrow keys move the list selection while the clicked row keeps the focus.
// Enter on that row used to activate it and snap the selection back to the
// row the pointer last chose, undoing the arrows. The history page has no
// Enter action, so the focused row must leave the selection alone.
func TestEnterOnAHistoryRowKeepsTheArrowSelection(t *testing.T) {
	commits := []gitrepo.Commit{
		{SHA: "aaaa", Subject: "first", Parents: []string{"root"}},
		{SHA: "bbbb", Subject: "second", Parents: []string{"aaaa"}},
	}
	a := &app{locale: localeChinese, page: "history", commits: commits}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(700).Height(200).Children(func() {
			for i, commit := range commits {
				a.commitRow(c, commit, i, 0, len(commits), &commitGraph{})
			}
		})
	}, 700, 200)
	first, ok := tt.AccessByRole(ui.RoleOption, "first")
	if !ok {
		t.Fatal("the first commit row is not an option")
	}
	tt.ClickAt(first.Bounds.X+first.Bounds.W/2, first.Bounds.Y+first.Bounds.H/2)
	if a.selectedSHA != "aaaa" {
		t.Fatalf("clicking the first row selected %q", a.selectedSHA)
	}
	a.moveSelection(1)
	if a.selectedSHA != "bbbb" {
		t.Fatalf("Down selected %q; want bbbb", a.selectedSHA)
	}
	if !tt.Focused("first") {
		t.Fatal("the clicked row lost the focus; the test cannot reproduce the mismatch")
	}
	tt.Key(0, ui.KeyEnter)
	if a.selectedSHA != "bbbb" {
		t.Errorf("Enter on the focused row selected %q; want the arrow selection bbbb", a.selectedSHA)
	}
}
