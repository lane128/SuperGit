package lobedesk

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

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
