package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// The details pane follows the list, and the arrows move that selection. A
// screen reader has to hear which row it is on, so both lists expose their
// choice the way the branch list does.
func TestCommitRowsSayWhichOneIsSelected(t *testing.T) {
	a := &app{locale: localeChinese, selectedSHA: "bbbb"}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(700).Height(400).Children(func() {
			a.commitRow(c, gitrepo.Commit{SHA: "aaaa", Subject: "first"}, 0, 0, 2, &commitGraph{})
			a.commitRow(c, gitrepo.Commit{SHA: "bbbb", Subject: "second"}, 1, 0, 2, &commitGraph{})
		})
	}, 700, 400)
	second, ok := tt.AccessByRole(ui.RoleOption, "second")
	if !ok {
		t.Fatal("a commit row is not an option of the list")
	}
	if !second.Selected {
		t.Errorf("the commit whose details are open reads %+v; want it selected", second)
	}
	if first, ok := tt.AccessByRole(ui.RoleOption, "first"); !ok || first.Selected {
		t.Errorf("the other commit reads %+v; want an option that is not selected", first)
	}
}
