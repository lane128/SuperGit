package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

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
	r, ok := tt.Find("a.go")
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
