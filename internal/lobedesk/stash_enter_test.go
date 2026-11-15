package lobedesk

import (
	"context"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// The stash rows are buttons, so a focused row takes Enter and activates
// itself. After the arrows moved the selection, Enter used to reload the
// last-clicked stash and pull the highlight back to it; Enter has no stash
// action, so the row must leave the selection where the arrows put it.
func TestEnterOnAStashRowKeepsTheArrowSelection(t *testing.T) {
	stashes := []gitrepo.Stash{
		{SHA: "aaaa", Ref: "stash@{0}", Message: "first"},
		{SHA: "bbbb", Ref: "stash@{1}", Message: "second"},
	}
	a := &app{locale: localeChinese, page: "stash", stashes: stashes}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.ctx = context.Background()
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(600).Height(300).Children(func() {
			for i, stash := range stashes {
				a.stashRow(c, stash, i+1, len(stashes))
			}
		})
	}, 600, 300)
	first, ok := tt.AccessByRole(ui.RoleOption, "first")
	if !ok {
		t.Fatal("the first stash row is not an option")
	}
	tt.ClickAt(first.Bounds.X+first.Bounds.W/2, first.Bounds.Y+first.Bounds.H/2)
	if a.selectedStash != "aaaa" {
		t.Fatalf("clicking the first row selected %q", a.selectedStash)
	}
	a.moveStashSelection(1)
	if a.selectedStash != "bbbb" {
		t.Fatalf("the arrow selected %q; want bbbb", a.selectedStash)
	}
	if !tt.Focused("first") {
		t.Fatal("the clicked row lost the focus; the test cannot reproduce the mismatch")
	}
	tt.Key(0, ui.KeyEnter)
	if a.selectedStash != "bbbb" {
		t.Errorf("Enter on the focused row selected %q; want the arrow selection bbbb", a.selectedStash)
	}
}
