package lobedesk

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// The line-number handle is the part of a diff row that takes the keyboard
// focus. The menu key has to reach the line's menu from there, or staging and
// discarding a single line are mouse-only; the tag and branch rows already
// work this way.
func TestDiffLineMenuOpensFromTheLineHandle(t *testing.T) {
	long := strings.Repeat("wide ", 20)
	lines := []gitrepo.DiffLine{
		{Kind: "hunk", Text: "@@ -1,2 +1,2 @@", Hunk: 0},
		{Kind: "context", Text: " kept", Old: 1, New: 1},
		{Kind: "add", Text: "+changed " + long, New: 2},
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
			Hunks:         []gitrepo.Hunk{{Header: "@@ -1,2 +1,2 @@"}},
			Lines:         lines,
		},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(700).Height(500).Children(func() {
			a.diffBody(c)
		})
	}, 700, 500)
	handle, ok := tt.AccessByRole(ui.RoleButton, a.lineGutterLabel(lines[2]))
	if !ok {
		t.Fatalf("the added line has no focusable handle %q", a.lineGutterLabel(lines[2]))
	}
	tt.ClickAt(handle.Bounds.X+handle.Bounds.W/2, handle.Bounds.Y+handle.Bounds.H/2)
	tt.Frame()
	if !a.lineSel[2] {
		t.Fatal("clicking the handle did not select the added line")
	}
	tt.Key(0, ui.KeyContextMenu)
	// The menu item's role, not just its text: a hovered row draws a button
	// with the same label, and Find would accept that instead of the menu.
	if _, ok := tt.AccessByRole(ui.RoleMenuItem, a.t("stageLine")); !ok {
		t.Errorf("the menu key on the line handle shows no %q menu item", a.t("stageLine"))
	}
}

// A right click on the code beside the handle keeps opening the menu: the code
// is a selectable paragraph, so it cannot be the menu's focusable target
// itself, but it is still part of the row the pointer acts on.
func TestDiffLineMenuStillOpensOnRightClickInTheCode(t *testing.T) {
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
			Lines: []gitrepo.DiffLine{
				{Kind: "hunk", Text: "@@ -1 +1 @@", Hunk: 0},
				{Kind: "add", Text: "+changed", New: 1},
			},
		},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(700).Height(500).Children(func() {
			a.diffBody(c)
		})
	}, 700, 500)
	row, ok := tt.Find("+changed")
	if !ok {
		t.Fatal("the added line's code is not laid out")
	}
	// The blank row beside the code, not the line-number handle: the handle is
	// a button and already opens the menu, so a press there would not prove the
	// rest of the row does too. The code text itself opens the platform's text
	// menu instead, which is the open question in TODO's "差异代码区的右键归属".
	tt.RightClickAt(row.X+row.W+40, row.Y+row.H/2)
	if _, ok := tt.AccessByRole(ui.RoleMenuItem, a.t("stageLine")); !ok {
		t.Errorf("right-clicking the row beside the code shows no %q menu item", a.t("stageLine"))
	}
	if !a.lineSel[1] {
		t.Error("right-clicking beside the code did not select the line it acted on")
	}
}
