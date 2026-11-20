package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"supergit/internal/gitrepo"
)

// The narrowest window a reader can drag to still has to show the working
// tree's commit editor: the summary field and the Commit button must sit
// inside the window instead of being pushed under its right edge.
func TestTheWorkbenchFitsTheNarrowestWindow(t *testing.T) {
	a := &app{
		page: "history", locale: localeEnglish, themeMode: "light", palette: themePaletteSuperGit,
		worktreeSelected: true,
		repo:             &gitrepo.Repository{Root: t.TempDir()},
		status: gitrepo.Status{Branch: "main", Entries: []gitrepo.Entry{
			{Path: "same.go", Index: 'M', Worktree: 'M'},
		}},
		selectedPath: "same.go",
		commitText:   "keep this title",
		diff:         gitrepo.Diff{Text: "+index version", Lines: []gitrepo.DiffLine{{Kind: "add", Text: "+index version"}}},
	}
	view := ui.NewTester(a.root, minWindowWidth, minWindowHeight)
	view.SetTitleBar(ui.TitleBar{Height: 38, Left: 72})
	for range 5 {
		view.Frame()
	}
	summary, ok := view.AccessByRole(ui.RoleTextField, a.t("commitMessage"))
	if !ok {
		t.Fatal("the narrowest window hides the commit summary field")
	}
	file, ok := view.AccessByRole(ui.RoleOption, "same.go")
	if !ok || summary.Bounds.X < file.Bounds.X+file.Bounds.W {
		t.Fatalf("commit editor overlaps the Files column: editor=%+v file=%+v", summary.Bounds, file.Bounds)
	}
	if summary.Offscreen || summary.Bounds.W < 120 {
		t.Fatalf("the commit summary field is squeezed to %+v", summary.Bounds)
	}
	commit, ok := view.AccessByRole(ui.RoleButton, a.t("commit"))
	if !ok {
		t.Fatal("the narrowest window hides the Commit button")
	}
	for _, node := range []ui.AccessInfo{summary, commit} {
		if node.Bounds.X < 0 || node.Bounds.X+node.Bounds.W > float32(minWindowWidth)+0.5 {
			t.Fatalf("%q leaves the %d-wide window: %+v", node.Name, minWindowWidth, node.Bounds)
		}
	}
	if a.commitText != "keep this title" {
		t.Fatalf("the layout changed the draft to %q", a.commitText)
	}
}
