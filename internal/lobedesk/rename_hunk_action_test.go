package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// A staged rename with an unstaged edit has an entry whose path moved. Its
// hunks are drawn from the new name, and the backend refuses to apply them
// because the index holds the rename; only the file row can stage it. The pane
// must not draw a "stage this hunk" action that can only fail.
func TestRenamedFileOffersNoHunkAction(t *testing.T) {
	a := &app{
		locale:       localeChinese,
		repo:         &gitrepo.Repository{Root: t.TempDir()},
		selectedPath: "b.txt",
		status: gitrepo.Status{Entries: []gitrepo.Entry{
			{Path: "b.txt", OriginalPath: "a.txt", Index: 'R', Worktree: 'M'},
		}},
		diff: gitrepo.Diff{
			Text:          "a patch",
			CanStageHunks: true,
			Hunks:         []gitrepo.Hunk{{Header: "@@ -1 +1 @@"}},
			Lines: []gitrepo.DiffLine{
				{Kind: "hunk", Text: "@@ -1 +1 @@", Hunk: 0},
				{Kind: "del", Text: "-one", Old: 1},
				{Kind: "add", Text: "+two", New: 1},
			},
		},
	}
	if a.hunkActionsAllowed() {
		t.Error("a staged rename offered a hunk action the backend refuses")
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
		t.Error("hovering a renamed file's hunk drew a stage button")
	}
}
