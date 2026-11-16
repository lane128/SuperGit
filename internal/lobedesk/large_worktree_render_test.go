package lobedesk

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestLargeWorktreeFilesBuildNearbyRowsAndRevealStagedVersion(t *testing.T) {
	a := &app{page: "history", worktreeSelected: true, locale: localeEnglish}
	for i := range 1000 {
		a.status.Entries = append(a.status.Entries, gitrepo.Entry{Path: fmt.Sprintf("files/%04d.txt", i), Index: 'M', Worktree: 'M'})
	}
	tt := pageTester(a, func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.ButtonBase(c).Label("before files").Height(24)
			ui.Row(c).Height(400).FillWidth().Children(func() { a.historyFilesPane(c) })
			ui.ButtonBase(c).Label("after files").Height(24)
		})
	})
	for range 3 {
		tt.Frame()
	}
	count := 0
	for _, node := range tt.AccessAll() {
		if node.Role == ui.RoleOption {
			count++
		}
	}
	if count == 0 || count > 100 {
		t.Fatalf("built %d worktree rows for a small viewport; want 1..100", count)
	}
	first, ok := tt.AccessByRole(ui.RoleOption, "files/0000.txt")
	if !ok {
		t.Fatal("first modified row missing")
	}
	tt.ClickAt(first.Bounds.X+first.Bounds.W/2, first.Bounds.Y+first.Bounds.H/2)
	tt.Key(0, ui.KeyEnd)
	for range 5 {
		tt.Frame()
	}
	found := false
	for _, node := range tt.AccessAll() {
		if node.Role == ui.RoleOption && node.Name == "files/0999.txt" && node.Description == a.t("fileGroupStaged") && node.Focused {
			found = true
			if node.PosInSet != 1000 || node.SetSize != 1000 || node.Bounds.Y < a.commitFilesViewport.Y || node.Bounds.Y+node.Bounds.H > a.commitFilesViewport.Y+a.commitFilesViewport.H+1 {
				t.Fatalf("staged endpoint lost position or viewport visibility: %+v", node)
			}
		}
	}
	if !found || !a.selectedStaged || a.selectedPath != "files/0999.txt" {
		t.Fatalf("End did not focus/select staged endpoint: found=%v path=%s staged=%v", found, a.selectedPath, a.selectedStaged)
	}
	if a.commitFilesScroll.MaxY < 1900*fileRowHeight() {
		t.Fatal("worktree virtualization lost complete grouped extent")
	}
	tt.Key(0, ui.KeyTab)
	// The overflow scrollbar has its own Tab stop outside the row focus group.
	if scroll, ok := tt.AccessByRole(ui.RoleButton, "Scroll"); ok && scroll.Focused {
		tt.Key(0, ui.KeyTab)
	}
	after, ok := tt.AccessByRole(ui.RoleButton, "after files")
	if !ok || !after.Focused {
		for _, node := range tt.AccessAll() {
			if node.Focused {
				t.Logf("focused after Tab: %+v", node)
			}
		}
		t.Fatalf("Tab did not leave grouped list in one step; after=%+v", after)
	}
	a.commitFilesScroll.Y = 0
	for range 4 {
		tt.Frame()
	}
	tt.Key(ui.Shift, ui.KeyTab)
	if scroll, ok := tt.AccessByRole(ui.RoleButton, "Scroll"); ok && scroll.Focused {
		tt.Key(ui.Shift, ui.KeyTab)
	}
	for range 5 {
		tt.Frame()
	}
	found = false
	for _, node := range tt.AccessAll() {
		if node.Role == ui.RoleOption && node.Name == "files/0999.txt" && node.Description == a.t("fileGroupStaged") && node.Focused {
			found = true
			if node.Bounds.Y < a.commitFilesViewport.Y || node.Bounds.Y+node.Bounds.H > a.commitFilesViewport.Y+a.commitFilesViewport.H+1 {
				t.Fatalf("remembered staged row was not revealed: %+v", node)
			}
		}
	}
	if !found {
		t.Fatal("Shift+Tab did not return to remembered staged version after scrolling it offscreen")
	}
}
