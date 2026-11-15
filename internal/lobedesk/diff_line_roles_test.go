package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// An added and a deleted line are content that was inserted or removed, not
// two ordinary paragraphs that happen to start with a sign, and their gutter
// handles need different names even when they carry the same numbers.
func TestDiffLinesSayInsertedAndDeleted(t *testing.T) {
	lines := []gitrepo.DiffLine{
		{Kind: "hunk", Text: "@@ -1 +1 @@", Hunk: 0},
		{Kind: "context", Text: " kept", Old: 1, New: 1},
		{Kind: "del", Text: "-old", Old: 1},
		{Kind: "add", Text: "+new", New: 2},
	}
	a := &app{
		locale:       localeChinese,
		repo:         &gitrepo.Repository{Root: t.TempDir()},
		selectedPath: "file.go",
		status: gitrepo.Status{Entries: []gitrepo.Entry{
			{Path: "file.go", Index: '.', Worktree: 'M'},
		}},
		diff: gitrepo.Diff{
			Text:          "a patch",
			CanStageHunks: true,
			Hunks:         []gitrepo.Hunk{{Header: "@@ -1 +1 @@"}},
			Lines:         lines,
		},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(700).Height(500).Children(func() {
			a.diffBody(c)
		})
	}, 700, 500)

	roles := map[ui.Role]bool{}
	for _, info := range tt.AccessAll() {
		roles[info.Role] = true
	}
	if !roles[ui.RoleDeletion] {
		t.Error("the deleted line does not read as a deletion")
	}
	if !roles[ui.RoleInsertion] {
		t.Error("the added line does not read as an insertion")
	}
	for _, label := range []string{"1 " + a.t("diffDeletedLine"), "2 " + a.t("diffAddedLine")} {
		if _, ok := tt.AccessByRole(ui.RoleButton, label); !ok {
			t.Errorf("no line handle named %q", label)
		}
	}
}
