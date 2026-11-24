package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestDetailTabTrackPreservesKeyboardFocus(t *testing.T) {
	a := &app{page: "history", locale: localeEnglish, detailReveal: true}
	for _, path := range []string{"first-long-configuration-file.go", "second-long-configuration-file.go"} {
		a.detailTabs().Open(detailFileKey{Path: path})
	}
	view := pageTester(a, func(c *ui.Context) {
		ui.Column(c).Width(230).Height(200).Children(func() { a.historyDiffPane(c) })
	})
	view.Frame()
	tab, ok := view.AccessByRole(ui.RoleTab, "second-long-configuration-file.go")
	if !ok {
		t.Fatal("missing tab")
	}
	view.ClickAt(tab.Bounds.X+tab.Bounds.W/2, tab.Bounds.Y+tab.Bounds.H/2)
	view.Press(110, 36.5)
	view.Move(2, 36.5)
	view.Release(2, 36.5)
	view.Frame()
	tab, _ = view.AccessByRole(ui.RoleTab, "second-long-configuration-file.go")
	if !tab.Focused {
		t.Fatal("scroll track cleared the tab focus")
	}
	view.Key(0, ui.KeyHome)
	if !a.detailTabs().Active.Summary {
		t.Fatal("Home stopped navigating after dragging the track")
	}
}

func TestDetailTabStripHasOneTabStop(t *testing.T) {
	a := &app{page: "history"}
	a.detailTabs().Open(detailFileKey{Path: "a.go"})
	a.detailTabs().Open(detailFileKey{Path: "b.go"})
	tt := pageTester(a, func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.ButtonBase(c).Label("before tabs").Height(24)
			ui.Column(c).Height(200).Children(func() { a.historyDiffPane(c) })
			ui.ButtonBase(c).Label("after tabs").Height(24)
		})
	})
	before, ok := tt.AccessByRole(ui.RoleButton, "before tabs")
	if !ok {
		t.Fatal("missing focus sentinel")
	}
	tt.ClickAt(before.Bounds.X+before.Bounds.W/2, before.Bounds.Y+before.Bounds.H/2)
	tt.Key(0, ui.KeyTab)
	found := false
	for _, node := range tt.AccessAll() {
		if node.Role == ui.RoleTab && node.Focused {
			found = true
		}
	}
	if !found {
		t.Fatal("Tab did not enter tab group")
	}
	tt.Key(0, ui.KeyTab)
	after, ok := tt.AccessByRole(ui.RoleButton, "after tabs")
	if !ok || !after.Focused {
		t.Fatal("Tab did not leave group in one step")
	}
}
