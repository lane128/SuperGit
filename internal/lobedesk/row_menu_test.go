package lobedesk

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

func TestHistoryCommitMenuFitsShortWindow(t *testing.T) {
	const width, height = 980, 640
	commit := gitrepo.Commit{SHA: "ab123456", Refs: "HEAD -> main", Author: "Reviewer", Day: "2026-11-22"}
	a := &app{locale: localeEnglish, repo: &gitrepo.Repository{Root: t.TempDir()}, selectedSHA: commit.SHA, commitMessage: "Menu fixture"}
	a.status = gitrepo.Status{Branch: "main", Head: commit.SHA}
	a.branches = []gitrepo.Branch{{Name: "main", SHA: commit.SHA, Current: true}, {Name: "topic", SHA: commit.SHA}}
	a.comparison.base = a.comparisonRevision("fedcba98", "base")
	view := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		row := ui.ButtonBase(c).Width(400).Height(40).Label("Fixture commit")
		a.commitMenu(c, row, commit)
	}, width, height)
	view.RightClickAt(200, 20)
	view.Advance(time.Second)
	panel, ok := view.AccessByRole(ui.RoleMenu, "")
	if !ok {
		t.Fatal("commit menu did not open")
	}
	var count int
	for _, node := range view.AccessAll() {
		if node.Role != ui.RoleMenuItem {
			continue
		}
		count++
		b := node.Bounds
		if node.Offscreen || b.Y < panel.Bounds.Y || b.Y+b.H > panel.Bounds.Y+panel.Bounds.H+0.01 {
			t.Errorf("menu item %q clipped on first open: row=%+v panel=%+v", node.Name, b, panel.Bounds)
		}
	}
	if count != 20 {
		t.Fatalf("expected all 20 commands including Compare and Reset, got %d", count)
	}
}

// Opening a row's menu selects the row, so the menu and the page describe the
// same item. The path reads the menu's open state from ui.Local, which no other
// test exercises. MyGo draws this menu as a panel, so the items are found on
// screen rather than through the system's menu API.
func TestTagRowMenuSelectsTheRow(t *testing.T) {
	a := &app{locale: localeChinese, page: "branches"}
	tag := gitrepo.Tag{Name: "v1.4.0", TargetSHA: "0123456789abcdef", TargetType: "commit"}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(600).Height(400).Children(func() {
			a.tagRow(c, tag, 1, 1)
		})
	}, 600, 400)
	r, ok := tt.Find("v1.4.0")
	if !ok {
		t.Fatal("the tag row is not laid out")
	}
	item := a.t("copyTagName")
	tt.RightClickAt(r.X+r.W/2, r.Y+r.H/2)
	if a.branchSel != "tag:v1.4.0" {
		t.Errorf("opening the tag menu selected %q; want tag:v1.4.0", a.branchSel)
	}
	if _, ok := tt.Find(item); !ok {
		t.Fatalf("right-clicking the tag row shows no %q item", item)
	}
	tt.Key(0, ui.KeyEscape)
	// The menu key works while the row has the keyboard: a row that is a
	// button must not lose that path.
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	tt.Key(0, ui.KeyContextMenu)
	if _, ok := tt.Find(item); !ok {
		t.Errorf("the menu key shows no %q item", item)
	}
}

func TestFileRowMenuSelectsTheFile(t *testing.T) {
	a := &app{locale: localeChinese, page: "history", worktreeSelected: true}
	entry := gitrepo.Entry{Path: "dir/a.go", Worktree: 'M'}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(600).Height(400).Children(func() {
			a.fileRow(c, entry, false)
		})
	}, 600, 400)
	r, ok := tt.Find(entry.Path)
	if !ok {
		t.Fatal("the file row is not laid out")
	}
	item := a.t("copyPath")
	tt.RightClickAt(r.X+r.W/2, r.Y+r.H/2)
	if a.selectedPath != "dir/a.go" || a.selectedStaged {
		t.Errorf("opening the file menu selected %q (staged %v); want dir/a.go unstaged", a.selectedPath, a.selectedStaged)
	}
	if _, ok := tt.Find(item); !ok {
		t.Fatalf("right-clicking the file row shows no %q item", item)
	}
	tt.Key(0, ui.KeyEscape)
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	tt.Key(0, ui.KeyContextMenu)
	if _, ok := tt.Find(item); !ok {
		t.Errorf("the menu key shows no %q item", item)
	}
}
