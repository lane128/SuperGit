package lobedesk

// Regression coverage for refresh observation and virtualized input ownership.
// Git writes below affect only repositories created by the test fixtures.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestReviewRefreshRetainsOlderSelection(t *testing.T) {
	a, callbacks, want := automaticHistoryFixture(t, 420)
	// Match production Open's canonical root; the fixture helper uses TempDir's alias.
	repo, err := gitrepo.Open(a.ctx, a.repo.Root)
	if err != nil {
		t.Fatal(err)
	}
	a.repo = repo
	a.loadMoreHistory()
	for a.historyMore {
		historyCompletion(t, callbacks)()
	}
	a.worktreeSelected = false
	a.setSelectedCommit(want[300])
	selected := a.selectedSHA
	file := detailFileKey{Path: "same.go"}
	a.detailTabs().Open(file)
	a.commitInfoScroll.Y, a.commitFilesScroll.Y, a.commitDiffScroll.Y = 101, 202, 303
	a.page = "history"
	// Exercise frame reconciliation without layout-clamping synthetic scroll offsets.
	a.historyDetailsHidden = true
	tt := pageTester(a, a.historyPage)
	a.reload()
	for a.loading || a.historyMore || a.historyLoadingMore || a.commitFilesLoading || a.commitDiffLoading {
		historyCompletion(t, callbacks)()
		if !a.loading {
			tt.Frame()
			if commit, ok := a.selectedCommit(); !ok || commit.SHA != selected {
				t.Fatal("selected commit metadata disappeared during replacement traversal")
			}
		}
	}
	if tabs := a.detailTabs(); len(tabs.OpenFiles) != 1 || tabs.Active.File != file {
		t.Fatalf("same-SHA refresh discarded file tab: tabs=%+v context=%+v worktree=%v", tabs, a.currentDetailContext(), a.worktreeSelected)
	}
	if a.commitInfoScroll.Y != 101 || a.commitFilesScroll.Y != 202 || a.commitDiffScroll.Y != 303 {
		t.Fatal("same-SHA refresh discarded reading positions")
	}
	if a.selectedSHA != selected {
		t.Fatalf("unchanged same-root refresh replaced selected commit %s with %s; total %d", selected, a.selectedSHA, len(a.commits))
	}
}

func TestRefreshDropsSelectionOnlyAfterSuccessfulAbsence(t *testing.T) {
	a, callbacks, want := automaticHistoryFixture(t, 420)
	a.loadMoreHistory()
	for a.historyMore {
		historyCompletion(t, callbacks)()
	}
	a.worktreeSelected = false
	a.setSelectedCommit(want[300])
	if out, err := exec.Command("git", "-C", a.repo.Root, "branch", "-D", "automatic-history").CombinedOutput(); err != nil {
		t.Fatalf("delete fixture branch: %v %s", err, out)
	}
	a.reload()
	for a.loading || a.historyMore || a.historyLoadingMore || a.commitFilesLoading || a.commitDiffLoading {
		historyCompletion(t, callbacks)()
	}
	if a.selectedSHA == want[300] || a.selectedSHA != a.status.Head {
		t.Fatalf("completed traversal retained unreachable selection: %s", a.selectedSHA)
	}
	if commit, ok := a.selectedCommit(); !ok || commit.SHA != a.status.Head {
		t.Fatal("fallback metadata missing")
	}
}

func TestReviewLongPatchDragPastOverscan(t *testing.T) {
	a := &app{page: "history", locale: localeEnglish, selectedSHA: "aaaa", commits: []gitrepo.Commit{{SHA: "aaaa", Subject: "large"}}, commitPressIndex: -1}
	lines := make([]gitrepo.DiffLine, 1000)
	for i := range lines {
		lines[i] = gitrepo.DiffLine{Kind: "add", New: i + 1, Text: fmt.Sprintf("+line %d", i)}
	}
	a.commitDiff, a.commitSections = buildCommitDiff([]gitrepo.FileDiff{{Path: "large.go", Diff: gitrepo.Diff{Lines: lines}}})
	tt := pageTester(a, a.historyDiffPane)
	for range 3 {
		tt.Frame()
	}
	first, ok := tt.AccessByRole(ui.RoleButton, a.lineGutterLabel(lines[0]))
	if !ok {
		t.Fatal("first gutter missing")
	}
	x, y := first.Bounds.X+first.Bounds.W/2, first.Bounds.Y+first.Bounds.H/2
	tt.Press(x, y)
	second, ok := tt.AccessByRole(ui.RoleButton, a.lineGutterLabel(lines[1]))
	if !ok {
		t.Fatal("second gutter missing")
	}
	tt.Move(x, second.Bounds.Y+second.Bounds.H/2)
	tt.Move(x, a.commitDiffViewport.Y+a.commitDiffViewport.H-2)
	for range 160 {
		tt.Frame()
	}
	t.Logf("scroll=%v viewport=%v cursor=%v held=%v press=%v", a.commitInfoScroll.Y, a.commitDiffViewport, a.commitCursor, a.commitDragHeld, a.commitPressIndex)
	if a.commitInfoScroll.Y < 4*a.commitDiffViewport.H {
		t.Fatalf("held drag stopped before crossing overscan: y=%v", a.commitInfoScroll.Y)
	}
	tt.Release(x, a.commitDiffViewport.Y+a.commitDiffViewport.H-2)
	for range 3 {
		tt.Frame()
	}
	if a.commitDragHeld || a.commitPressIndex >= 0 {
		t.Fatal("released drag remains armed")
	}
}

func TestReviewVirtualFilesRestoreRememberedTabFocus(t *testing.T) {
	a := &app{page: "history", selectedSHA: "aaaa"}
	for i := range 1000 {
		a.commitFiles = append(a.commitFiles, gitrepo.ChangedFile{Path: fmt.Sprintf("files/%04d.txt", i), Status: "A"})
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
	first, ok := tt.AccessByRole(ui.RoleOption, "files/0000.txt")
	if !ok {
		t.Fatal("missing first")
	}
	tt.ClickAt(first.Bounds.X+first.Bounds.W/2, first.Bounds.Y+first.Bounds.H/2)
	tt.Key(0, ui.KeyTab)
	a.commitFilesScroll.Y = 900 * (fileRowHeight() + fileRowGap)
	for range 4 {
		tt.Frame()
	}
	tt.Key(ui.Shift, ui.KeyTab)
	for range 4 {
		tt.Frame()
	}
	first, ok = tt.AccessByRole(ui.RoleOption, "files/0000.txt")
	if !ok || !first.Focused {
		for _, n := range tt.AccessAll() {
			if n.Focused {
				t.Logf("actual focus: %+v", n)
			}
		}
		t.Fatalf("Shift+Tab lost remembered first file after scroll; firstExists=%v scroll=%v", ok, a.commitFilesScroll.Y)
	}
}

func TestReviewSaveDuringRefreshRemainsObservable(t *testing.T) {
	for _, skip := range []bool{false, true} {
		t.Run(fmt.Sprintf("after-second-status-%t", skip), func(t *testing.T) {
			testSaveDuringRefresh(t, skip)
		})
	}
}

func testSaveDuringRefresh(t *testing.T, skip bool) {
	t.Helper()
	a, callbacks := overviewFixture(t)
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(realGit, "-C", a.repo.Root, "reset", "--hard", "HEAD").CombinedOutput(); err != nil {
		t.Fatalf("fixture reset %v %s", err, out)
	}
	dir := t.TempDir()
	script := `#!/bin/sh
for arg do
 if [ "$arg" = status ] && [ ! -e "$REVIEW_MARKER" ]; then
  if [ "$REVIEW_SKIP" = 1 ] && [ ! -e "$REVIEW_MARKER.skip" ]; then
   /usr/bin/touch "$REVIEW_MARKER.skip"
   exec "$REVIEW_REAL_GIT" "$@"
  fi
  "$REVIEW_REAL_GIT" "$@"
  rc=$?
  /usr/bin/touch "$REVIEW_MARKER"
  /usr/bin/printf 'external save during refresh\n' > "$REVIEW_TARGET"
  exit "$rc"
 fi
done
exec "$REVIEW_REAL_GIT" "$@"
`
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	if skip {
		t.Setenv("REVIEW_SKIP", "1")
	} else {
		t.Setenv("REVIEW_SKIP", "0")
	}
	t.Setenv("REVIEW_REAL_GIT", realGit)
	t.Setenv("REVIEW_TARGET", filepath.Join(a.repo.Root, "same.go"))
	t.Setenv("REVIEW_MARKER", filepath.Join(dir, "ran"))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	a.openRepository(a.repo.Root, false)
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for a.loading {
		select {
		case fn := <-callbacks:
			fn()
		case <-timer.C:
			t.Fatal("reload timeout")
		}
	}
	actual, err := a.repo.Status(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(actual.Entries) != 1 {
		t.Fatal("injected ordinary save did not dirty file")
	}
	epoch := a.loadSeq
	for i := 1; i <= 3; i++ {
		a.pollLocalChanges(time.Now().Add(time.Duration(i) * time.Minute))
		for a.localWatch.reading || a.loading {
			select {
			case fn := <-callbacks:
				fn()
			case <-timer.C:
				t.Fatal("probe timeout")
			}
		}
	}
	if len(a.status.Entries) == 0 {
		t.Fatalf("actual Git dirty but displayed status still clean after three polls; epoch=%d->%d", epoch, a.loadSeq)
	}
}
