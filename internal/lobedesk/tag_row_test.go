package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// Enter opens the selected tag, and a tag row is a button so it keeps Tab,
// Shift+F10 and the accessibility role. The row's shortcut has to take Enter
// before the button does: otherwise Enter would only reselect the row, and
// clicking a tag first would make Enter stop working.
func TestTagRowEnterOpensTheTag(t *testing.T) {
	a := &app{
		locale: localeChinese,
		page:   "branches",
		// jumpToCommit only needs a repository to look at; the tag's commit is
		// not in the empty history, so nothing runs git.
		repo: &gitrepo.Repository{Root: t.TempDir()},
		tags: []gitrepo.Tag{{
			Name: "v1.4.0", TargetSHA: "0123456789abcdef", TargetType: "commit",
		}},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(600).Height(400).Children(func() {
			a.tagRow(c, a.tags[0], 1, 1)
		})
	}, 600, 400)
	r, ok := tt.Find("v1.4.0")
	if !ok {
		t.Fatal("the tag row is not laid out")
	}
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	if a.branchSel != "tag:v1.4.0" {
		t.Errorf("clicking the tag row selected %q", a.branchSel)
	}
	tt.Key(0, ui.KeyEnter)
	if a.page != "history" {
		t.Errorf("Enter after clicking the tag left the page on %q; want history", a.page)
	}
}
