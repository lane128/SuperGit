package lobedesk

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestOverviewLineActionKeepsItsFileAndStage(t *testing.T) {
	for _, staged := range []bool{false, true} {
		t.Run(map[bool]string{false: "stage", true: "unstage"}[staged], func(t *testing.T) {
			a, callbacks := overviewFixture(t)
			a.page = "history"
			a.loadWorktreeOverview()
			waitOverview(t, a, callbacks)
			var target overviewPatchTarget
			for _, entry := range a.overviewEntries {
				if entry.Key.Staged == staged {
					target = overviewPatchTarget{Root: a.overviewRoot, Generation: a.overviewSeq, File: entry.Key, Diff: entry.Diff}
				}
			}
			// The preloaded file cursor describes a different file and version.
			a.selectedPath, a.selectedStaged = "unrelated.go", !staged
			before, err := os.ReadFile(filepath.Join(a.repo.Root, "same.go"))
			if err != nil {
				t.Fatal(err)
			}
			action := a.overviewLineAction(target, changedLineIndexes(target.Diff))
			if action == nil {
				t.Fatal("visible overview patch cannot prepare a line operation")
			}
			if err := action(a.ctx, a.repo); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("git", "-C", a.repo.Root, "show", ":same.go")
			index, err := cmd.Output()
			if err != nil {
				t.Fatal(err)
			}
			want := "working version\n"
			if staged {
				want = "original\n"
			}
			if string(index) != want {
				t.Fatalf("Index = %q; want %q", index, want)
			}
			after, err := os.ReadFile(filepath.Join(a.repo.Root, "same.go"))
			if err != nil || string(after) != string(before) {
				t.Fatal("overview operation changed working file")
			}
		})
	}
}

func TestOverviewLineHandleOffersItsOwnStageAction(t *testing.T) {
	for _, staged := range []bool{false, true} {
		a, callbacks := overviewFixture(t)
		a.page = "history"
		a.loadWorktreeOverview()
		waitOverview(t, a, callbacks)
		var entry worktreeOverviewEntry
		for _, candidate := range a.overviewEntries {
			if candidate.Key.Staged == staged {
				entry = candidate
			}
		}
		target := overviewPatchTarget{Root: a.overviewRoot, Generation: a.overviewSeq, File: entry.Key, Diff: entry.Diff}
		index := changedLineIndexes(entry.Diff)[0]
		tt := pageTester(a, func(c *ui.Context) { a.overviewPatchLine(c, target, index, 700) })
		handle, ok := tt.AccessByRole(ui.RoleButton, a.lineGutterLabel(entry.Diff.Lines[index]))
		if !ok {
			t.Fatal("overview line has no focusable selection handle")
		}
		tt.ClickAt(handle.Bounds.X+handle.Bounds.W/2, handle.Bounds.Y+handle.Bounds.H/2)
		if !a.lineSel[index] {
			t.Fatal("overview handle did not select its row")
		}
		tt.Key(0, ui.KeyContextMenu)
		label := "stageLine"
		if staged {
			label = "unstageLine"
		}
		if _, ok := tt.AccessByRole(ui.RoleMenuItem, a.t(label)); !ok {
			t.Fatalf("overview line menu omits %q", a.t(label))
		}
	}
}

func TestDetailTabSwitchClearsInvisibleSelection(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.page = "history"
	a.loadWorktreeOverview()
	waitOverview(t, a, callbacks)
	first, second := a.overviewEntries[0], a.overviewEntries[1]
	target := func(entry worktreeOverviewEntry) overviewPatchTarget {
		return overviewPatchTarget{Root: a.overviewRoot, Generation: a.overviewSeq, File: entry.Key, Diff: entry.Diff}
	}
	a.selectOverviewLine(target(first), changedLineIndexes(first.Diff)[1], 0)
	a.selectOverviewLine(target(second), changedLineIndexes(second.Diff)[0], ui.Shift)
	if len(a.lineSel) != 1 {
		t.Fatalf("Shift extended selection across versions: %v", a.lineSel)
	}
	if a.activeWorktreeDiff().Text != second.Diff.Text {
		t.Fatal("selection reads the background file cache")
	}
	a.openDetailFile(first.Key)
	if len(a.lineSel) != 0 || a.overviewLineTarget != nil {
		t.Fatal("file tab retained overview selection")
	}
	a.activateDetailTab(detailTabKey{Summary: true})
	a.selectOverviewLine(target(first), changedLineIndexes(first.Diff)[0], 0)
	a.invalidateWorktreeOverview()
	if len(a.lineSel) != 0 || a.overviewLineTarget != nil {
		t.Fatal("refresh retained obsolete overview selection")
	}
}

func TestBackgroundFileReadKeepsOverviewSelection(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.page = "history"
	a.loadWorktreeOverview()
	waitOverview(t, a, callbacks)
	entry := a.overviewEntries[0]
	a.selectedPath, a.selectedStaged = entry.Key.Path, entry.Key.Staged
	a.loadDiff()
	var completion func()
	select {
	case completion = <-callbacks:
	case <-time.After(10 * time.Second):
		t.Fatal("single-file read did not finish")
	}
	target := overviewPatchTarget{Root: a.overviewRoot, Generation: a.overviewSeq, File: entry.Key, Diff: entry.Diff}
	a.selectOverviewLine(target, changedLineIndexes(entry.Diff)[0], 0)
	completion()
	if len(a.lineSel) != 1 || a.overviewLineTarget == nil {
		t.Fatal("background single-file result erased visible overview selection")
	}
}

func TestBackgroundFileReadRejectsChangedTargets(t *testing.T) {
	for _, change := range []string{"repository", "path", "stage"} {
		t.Run(change, func(t *testing.T) {
			a, callbacks := overviewFixture(t)
			a.selectedPath = "same.go"
			a.loadDiff()
			var completion func()
			select {
			case completion = <-callbacks:
			case <-time.After(10 * time.Second):
				t.Fatal("single-file read did not finish")
			}
			switch change {
			case "repository":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			case "path":
				a.selectedPath = "other.go"
			case "stage":
				a.selectedStaged = true
			}
			a.diff = gitrepo.Diff{Text: "current target"}
			completion()
			if a.diff.Text != "current target" {
				t.Fatal("old callback replaced a different target")
			}
		})
	}
}

func TestOverviewSelectionDoesNotDiscardTheBackgroundFile(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.page = "history"
	a.loadWorktreeOverview()
	waitOverview(t, a, callbacks)
	entry := a.overviewEntries[0]
	a.selectedPath, a.diff = entry.Key.Path, entry.Diff
	target := overviewPatchTarget{Root: a.overviewRoot, Generation: a.overviewSeq, File: entry.Key, Diff: entry.Diff}
	a.selectOverviewLine(target, changedLineIndexes(entry.Diff)[0], 0)
	if a.discardLinesAllowed() {
		t.Fatal("overview selection inherited background file's discard action")
	}
}

func TestOverviewLineActionRejectsStaleTargets(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.page = "history"
	a.loadWorktreeOverview()
	waitOverview(t, a, callbacks)
	entry := a.overviewEntries[0]
	target := overviewPatchTarget{Root: a.overviewRoot, Generation: a.overviewSeq, File: entry.Key, Diff: entry.Diff}
	action := a.overviewLineAction(target, changedLineIndexes(target.Diff))
	if action == nil {
		t.Fatal("cannot prepare current operation")
	}
	// The operation cannot be reused against another repository.
	if err := action(a.ctx, &gitrepo.Repository{Root: t.TempDir()}); err == nil {
		t.Fatal("operation accepted a different repository")
	}
	if err := os.WriteFile(filepath.Join(a.repo.Root, "same.go"), []byte("external edit\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := action(a.ctx, a.repo); err == nil {
		t.Fatal("operation accepted a stale patch")
	}
	a.overviewSeq++
	if a.overviewLineAction(target, changedLineIndexes(target.Diff)) != nil {
		t.Fatal("old overview generation still prepares writes")
	}
}
