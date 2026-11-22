package lobedesk

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestRefreshKeepsPreviewIndependentOfFilesSelection(t *testing.T) {
	for _, staged := range []bool{false, true} {
		group, otherGroup := filesSectionUnstaged, filesSectionStaged
		action := "stageFile"
		if staged {
			group, otherGroup = otherGroup, group
			action = "unstageFile"
		}
		t.Run(group, func(t *testing.T) {
			a, callbacks, view := workspaceSelectionFixture(t)
			a.reload() // Establish Git's canonical checkout path before selecting rows.
			waitSelectionOperation(t, a, callbacks)
			const path = "same.go"
			clickSelectionRow(t, view, path, group, 0)
			clickSelectionRow(t, view, path, otherGroup, 0)
			a.filesCollapsed = map[string]bool{}
			a.foldFilesSection(group)
			tab := detailTabKey{File: detailFileKey{Path: path, Staged: staged}}
			a.activateDetailTab(tab)
			if a.detailTabs().Active != tab || !a.workspaceSelection.Files[fileRef{path, !staged}] {
				t.Fatal("fixture did not open the preview independently of the Files selection")
			}
			want, err := a.repo.Diff(a.ctx, path, staged)
			if err != nil {
				t.Fatal(err)
			}
			for range 2 {
				a.reload()
				waitSelectionOperation(t, a, callbacks)
				for a.diffLoading {
					runContinuation(t, callbacks)
				}
				if a.detailTabs().Active != tab || a.selectedPath != path || a.selectedStaged != staged || a.diff.Text != want.Text {
					t.Fatalf("refresh changed preview: active=%+v target=%q staged=%v diff=%q", a.detailTabs().Active, a.selectedPath, a.selectedStaged, a.diff.Text)
				}
				selection := a.workspaceSelection
				if len(selection.Files) != 1 || !selection.Files[fileRef{path, !staged}] {
					t.Fatalf("refresh changed the independent Files selection: %+v", selection)
				}
				header := pageTester(a, a.historyDiffPane)
				if _, ok := header.AccessByRole(ui.RoleButton, a.t(action)); !ok {
					t.Fatalf("missing %s action after refresh", action)
				}
			}
		})
	}
}

func TestRefreshAfterDeselectingLastFile(t *testing.T) {
	a, callbacks, view := workspaceSelectionFixture(t)
	a.reload()
	waitSelectionOperation(t, a, callbacks)
	clickSelectionRow(t, view, "b.go", filesSectionUnstaged, 0)
	clickSelectionRow(t, view, "b.go", filesSectionUnstaged, ui.Cmd)
	a.reload()
	waitSelectionOperation(t, a, callbacks)
	for a.diffLoading {
		runContinuation(t, callbacks)
	}
	if len(a.workspaceSelection.Files) != 0 || !a.detailTabs().Active.Summary || a.selectedPath != "" || a.diff.Text != "" {
		t.Fatalf("refresh reopened the deselected preview: tab=%+v path=%q diff=%q", a.detailTabs().Active, a.selectedPath, a.diff.Text)
	}

	tab := detailTabKey{File: detailFileKey{Path: "b.go"}}
	a.activateDetailTab(tab)
	a.reload()
	waitSelectionOperation(t, a, callbacks)
	for a.diffLoading {
		runContinuation(t, callbacks)
	}
	want, err := a.repo.Diff(a.ctx, tab.File.Path, false)
	if err != nil {
		t.Fatal(err)
	}
	if a.detailTabs().Active != tab || len(a.workspaceSelection.Files) != 0 || a.selectedPath != tab.File.Path || a.diff.Text != want.Text {
		t.Fatalf("refresh lost the manually reopened tab: tab=%+v path=%q diff=%q", a.detailTabs().Active, a.selectedPath, a.diff.Text)
	}
}

func TestHistoryPaginationPreservesIndependentWorkspacePreview(t *testing.T) {
	a, callbacks, _ := automaticHistoryFixture(t, 420)
	t.Cleanup(a.resetHistoryTraversal)
	a.page = "history"
	a.selectWorkspaceFile(fileRef{"same.go", false}, 0, false)
	a.selectWorkspaceFile(fileRef{"same.go", true}, 0, false)
	a.filesCollapsed = map[string]bool{filesSectionUnstaged: true}
	tab := detailTabKey{File: detailFileKey{Path: "same.go"}}
	a.activateDetailTab(tab)
	for a.diffLoading {
		runContinuation(t, callbacks)
	}
	want := a.diff.Text
	if want == "" || !a.historyMore || a.selectedStaged {
		t.Fatal("fixture requires an unstaged patch and multiple history pages")
	}
	a.loadMoreHistory()
	for a.historyMore {
		historyCompletion(t, callbacks)()
	}
	if a.detailTabs().Active != tab || a.selectedPath != tab.File.Path || a.selectedStaged || a.diff.Text != want {
		t.Fatalf("history pagination changed preview: tab=%+v path=%q staged=%v diff=%q", a.detailTabs().Active, a.selectedPath, a.selectedStaged, a.diff.Text)
	}
}

func TestRefreshPreservesActiveFileTabThroughFoldedGroups(t *testing.T) {
	for _, samePath := range []bool{false, true} {
		t.Run(map[bool]string{false: "different files", true: "same path different stages"}[samePath], func(t *testing.T) {
			a, callbacks := overviewFixture(t)
			active := detailFileKey{Path: "same.go"}
			neighbor := detailFileKey{Path: "same.go", Staged: true}
			if !samePath {
				neighbor.Path = "b.go"
				if err := os.WriteFile(filepath.Join(a.repo.Root, "b.go"), []byte("other index\n"), 0600); err != nil {
					t.Fatal(err)
				}
				command := exec.Command("git", "add", "--", "b.go")
				command.Dir = a.repo.Root
				if output, err := command.CombinedOutput(); err != nil {
					t.Fatalf("fixture add: %v %s", err, output)
				}
				var err error
				a.status, err = a.repo.Status(a.ctx)
				if err != nil {
					t.Fatal(err)
				}
			}
			a.page = "history"
			a.detailTabs().Open(neighbor)
			a.detailTabs().Open(active)
			a.selectedPath, a.selectedStaged = active.Path, active.Staged
			a.filesCollapsed = map[string]bool{filesSectionUnstaged: true}
			a.diffScroll.Y = 123
			// This is the actual successful refresh reconciliation order.
			a.reconcileSelection()
			a.reconcileWorktreeTabs(a.overviewKeys())
			if a.detailTabs().Active.File != active || a.selectedPath != active.Path || a.selectedStaged != active.Staged {
				t.Fatalf("active tab %+v renders/writes selected target %q staged=%v", a.detailTabs().Active, a.selectedPath, a.selectedStaged)
			}
			if a.diffScroll.Y != 123 {
				t.Fatal("target restoration reset reading position")
			}
			expected, err := a.repo.Diff(a.ctx, active.Path, active.Staged)
			if err != nil {
				t.Fatal(err)
			}
			a.loadDiff()
			timer := time.NewTimer(5 * time.Second)
			defer timer.Stop()
			for a.diffLoading {
				select {
				case callback := <-callbacks:
					callback()
				case <-timer.C:
					t.Fatal("active tab diff did not finish")
				}
			}
			if a.diff.Text != expected.Text {
				t.Fatal("file tab rendered another version's patch")
			}
			entry, ok := a.selectedEntry()
			if !ok || entry.Path != active.Path {
				t.Fatal("file header targets another entry")
			}
			tt := pageTester(a, a.historyDiffPane)
			if _, ok := tt.AccessByRole(ui.RoleButton, a.t("stageFile")); !ok {
				t.Fatal("unstaged tab offers the wrong stage action")
			}
		})
	}
}
