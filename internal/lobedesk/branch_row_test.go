package lobedesk

import (
	"context"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

func branchRowsApp() *app {
	a := &app{
		locale: localeChinese,
		page:   "branches",
		branches: []gitrepo.Branch{
			{Name: "main", SHA: "aaaa", Current: true},
			{Name: "feat", SHA: "bbbb"},
		},
	}
	a.branchSel = "local:main"
	return a
}

func branchRowTester(a *app) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(600).Height(400).Children(func() {
			for i, branch := range a.branches {
				a.branchRow(c, branch, i+1, len(a.branches))
			}
		})
	}, 600, 400)
}

// A branch row was a plain group: no name, no focus, and no menu key. It now
// takes the keyboard the way a tag row does, and says which branch it is and
// whether it is the one the list has chosen, so a screen reader can offer the
// same list the arrows walk.
func TestBranchRowIsASelectableOption(t *testing.T) {
	a := branchRowsApp()
	tt := branchRowTester(a)

	main, ok := tt.AccessByRole(ui.RoleOption, "main")
	if !ok {
		t.Fatal("the branch row is not an option of the list")
	}
	if !main.Selected || main.Description != a.t("current") {
		t.Errorf("the checked-out branch reads %+v; want the selected option, explained as the current branch", main)
	}
	if feat, ok := tt.AccessByRole(ui.RoleOption, "feat"); !ok {
		t.Fatal("the other branch row is not an option of the list")
	} else if feat.Selected || feat.Description != "" {
		t.Errorf("feat reads %+v; want an option that is not selected and has no current note", feat)
	}

	r, ok := tt.Find("feat")
	if !ok {
		t.Fatal("the feat row is not laid out")
	}
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	if a.branchSel != "local:feat" {
		t.Errorf("clicking the row selected %q; want local:feat", a.branchSel)
	}
	if feat, _ := tt.AccessByRole(ui.RoleOption, "feat"); !feat.Selected {
		t.Errorf("feat reads %+v after the click; want it selected", feat)
	}
	// Only one row is chosen at a time: the checked-out branch keeps its
	// description but stops reading as the selection.
	if main, _ := tt.AccessByRole(ui.RoleOption, "main"); main.Selected {
		t.Errorf("main still reads as selected after choosing feat: %+v", main)
	}
}

// The row is focusable now, so the menu key and Shift+F10 reach its menu.
// Right-click must keep working and keep selecting the row.
func TestBranchRowMenuOpensFromTheKeyboard(t *testing.T) {
	a := branchRowsApp()
	tt := branchRowTester(a)
	r, ok := tt.Find("feat")
	if !ok {
		t.Fatal("the feat row is not laid out")
	}
	copyItem := a.t("copyName")
	tt.RightClickAt(r.X+r.W/2, r.Y+r.H/2)
	if a.branchSel != "local:feat" {
		t.Errorf("opening the menu selected %q; want local:feat", a.branchSel)
	}
	if _, ok := tt.Find(copyItem); !ok {
		t.Fatalf("right-clicking the row shows no %q item", copyItem)
	}
	tt.Key(0, ui.KeyEscape)

	for _, test := range []struct {
		name string
		send func()
	}{
		{"the menu key", func() { tt.Key(0, ui.KeyContextMenu) }},
		{"Shift+F10", func() { tt.Key(ui.Shift, ui.KeyF10) }},
	} {
		if _, ok := tt.Find(copyItem); ok {
			t.Fatalf("%s: the menu is still open before the key", test.name)
		}
		tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
		test.send()
		if _, ok := tt.Find(copyItem); !ok {
			t.Errorf("%s shows no %q item", test.name, copyItem)
		}
		tt.Key(0, ui.KeyEscape)
	}
}

// The row's Enter shortcut covers its whole focus subtree, and the switch
// button sits inside it. With the keyboard on that button, Enter has to press
// the button it is on -- switching the branch the button names, not the row
// the list has selected. Space already does; the two keys cannot disagree.
func TestSwitchButtonKeepsEnterFromTheRowShortcut(t *testing.T) {
	a := branchRowsApp()
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.ctx = context.Background()
	a.status.Branch = "main"
	a.busy, a.busyKey = false, ""
	tt := branchRowTester(a)
	button, ok := tt.AccessByRole(ui.RoleButton, a.t("switch"))
	if !ok {
		t.Fatal("the feat row has no switch button")
	}
	// Press and cancel: the button takes the focus without switching.
	tt.Press(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
	tt.CancelPress()
	if a.busy {
		t.Fatal("the press already switched; the test cannot tell what Enter did")
	}
	if !tt.Focused(a.t("switch")) {
		t.Fatal("the switch button did not take the focus")
	}
	tt.Key(0, ui.KeyEnter)
	if !a.busy || a.busyKey != string(busySwitch) {
		t.Errorf("Enter on the switch button left busy=%v key=%q; want it to switch feat", a.busy, a.busyKey)
	}
}

// Enter on the row itself keeps running the page's Enter action: check out
// the branch the list has selected.
func TestEnterOnTheBranchRowChecksItOut(t *testing.T) {
	a := branchRowsApp()
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.ctx = context.Background()
	a.status.Branch = "main"
	a.branchSel = "local:feat"
	tt := branchRowTester(a)
	row, ok := tt.AccessByRole(ui.RoleOption, "feat")
	if !ok {
		t.Fatal("the feat row is not an option of the list")
	}
	tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	if a.branchSel != "local:feat" {
		t.Fatalf("clicking the row selected %q; want local:feat", a.branchSel)
	}
	tt.Key(0, ui.KeyEnter)
	if !a.busy || a.busyKey != string(busySwitch) {
		t.Errorf("Enter on the row left busy=%v key=%q; want it to check feat out", a.busy, a.busyKey)
	}
}

func remoteBranchRowsApp() *app {
	a := &app{
		locale: localeChinese,
		page:   "branches",
		remotes: []gitrepo.Remote{
			{Name: "origin"},
		},
		branches: []gitrepo.Branch{
			{Name: "feat", SHA: "bbbb", Upstream: "origin/feat"},
			{Name: "topic", SHA: "cccc", Upstream: "origin/topic"},
		},
	}
	a.branchSel = "remote:origin/feat"
	return a
}

func remoteBranchRowTester(a *app) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(600).Height(400).Children(func() {
			a.remoteBranchRow(c, gitrepo.Branch{Name: "origin/feat", SHA: "bbbb"}, 1, 2)
			a.remoteBranchRow(c, gitrepo.Branch{Name: "origin/topic", SHA: "cccc"}, 2, 2)
		})
	}, 600, 400)
}

// The remote rows were the last section of the branch list that a screen
// reader saw as loose text and the keyboard could not reach. They are options
// now, like the local and tag rows: named, selected, focusable, and able to
// open their menu from the keyboard.
func TestRemoteBranchRowIsASelectableOption(t *testing.T) {
	a := remoteBranchRowsApp()
	tt := remoteBranchRowTester(a)
	feat, ok := tt.AccessByRole(ui.RoleOption, "feat")
	if !ok {
		t.Fatal("a remote branch row is not an option of the list")
	}
	if !feat.Selected {
		t.Errorf("the chosen remote branch reads %+v; want it selected", feat)
	}
	if topic, ok := tt.AccessByRole(ui.RoleOption, "topic"); !ok || topic.Selected {
		t.Errorf("the other remote branch reads %+v; want an option that is not selected", topic)
	}
	r, ok := tt.Find("topic")
	if !ok {
		t.Fatal("the topic row is not laid out")
	}
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	if a.branchSel != "remote:origin/topic" {
		t.Errorf("clicking the row selected %q; want remote:origin/topic", a.branchSel)
	}
	tt.Key(0, ui.KeyContextMenu)
	if _, ok := tt.Find(a.t("copyName")); !ok {
		t.Errorf("the menu key on a remote row shows no %q item", a.t("copyName"))
	}
}
