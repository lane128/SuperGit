package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestWorkspaceSummaryKeepsSelectionFeedbackIndependentOfPreview(t *testing.T) {
	a, _, _ := workspaceSelectionFixture(t)
	tt := pageTester(a, func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() {
			ui.Row(c).Children(func() {
				a.detailTabButton(c, theme.TokensOf(c), detailTabKey{Summary: true})
				a.detailTabButton(c, theme.TokensOf(c), detailTabKey{File: detailFileKey{Path: "c.go"}})
			})
			for _, group := range a.workingFileGroups() {
				for _, entry := range group.Entries {
					a.fileRow(c.Key(group.Key+entry.Path), entry, group.Staged).Description(group.Key)
				}
			}
		})
	})
	clickSelectionRow(t, tt, "c.go", filesSectionUnstaged, 0)
	clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
	clickSelectionRow(t, tt, "same.go", filesSectionUnstaged, ui.Cmd)
	anchor := a.workspaceSelection.Anchor
	tab, ok := tt.AccessByRole(ui.RoleTab, a.detailSummaryLabel())
	if !ok {
		t.Fatal("Summary tab missing")
	}
	tt.ClickAt(tab.Bounds.X+tab.Bounds.W/2, tab.Bounds.Y+tab.Bounds.H/2)
	if a.detailTabShown() != detailTabSummary {
		t.Fatal("Summary click did not activate Summary")
	}
	for _, path := range []string{"b.go", "same.go"} {
		if !selectionRow(t, tt, path, filesSectionUnstaged).Selected {
			t.Fatalf("Summary hid retained selection feedback for %s", path)
		}
	}
	if len(a.workspaceSelection.Files) != 2 || a.workspaceSelection.Anchor != anchor {
		t.Fatal("Summary changed selection membership or anchor")
	}
	tab, ok = tt.AccessByRole(ui.RoleTab, "c.go · Unstaged")
	if !ok {
		t.Fatal("existing file tab missing")
	}
	tt.ClickAt(tab.Bounds.X+tab.Bounds.W/2, tab.Bounds.Y+tab.Bounds.H/2)
	if a.selectedPath != "c.go" || a.selectedStaged {
		t.Fatal("file tab did not preview its own version")
	}
	if len(a.workspaceSelection.Files) != 2 || a.workspaceSelection.Anchor != anchor {
		t.Fatal("file preview changed independent selected set")
	}
	if selectionRow(t, tt, "c.go", filesSectionUnstaged).Selected {
		t.Fatal("preview-only file silently joined selected set")
	}
	for _, path := range []string{"b.go", "same.go"} {
		if !selectionRow(t, tt, path, filesSectionUnstaged).Selected {
			t.Fatalf("preview switch hid selection for %s", path)
		}
	}
	// Preserve the existing Summary behavior for a preview-only cursor that
	// was not established as an explicit list selection.
	a.selectFile("c.go", false)
	tab, _ = tt.AccessByRole(ui.RoleTab, a.detailSummaryLabel())
	tt.ClickAt(tab.Bounds.X+tab.Bounds.W/2, tab.Bounds.Y+tab.Bounds.H/2)
	if selectionRow(t, tt, "c.go", filesSectionUnstaged).Selected {
		t.Fatal("legacy preview-only cursor gained a Summary selection")
	}
}
