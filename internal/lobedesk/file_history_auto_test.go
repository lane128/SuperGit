package lobedesk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func automaticFileHistoryFixture(t *testing.T, count int) (*app, chan func(), []string) {
	t.Helper()
	a, callbacks := overviewFixture(t)
	var input strings.Builder
	for i := 0; i < count; i++ {
		message := fmt.Sprintf("File history %04d", i)
		text := fmt.Sprintf("revision %d\n", i)
		fmt.Fprintf(&input, "commit refs/heads/file-history\ncommitter Fixture <fixture@example.invalid> %d +0000\ndata %d\n%s\nM 100644 inline same.go\ndata %d\n%s\n", 1700000000+i, len(message), message, len(text), text)
	}
	input.WriteString("done\n")
	cmd := exec.Command("git", "-C", a.repo.Root, "fast-import", "--quiet")
	cmd.Stdin = strings.NewReader(input.String())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("import: %v %s", err, out)
	}
	if out, err := exec.Command("git", "-C", a.repo.Root, "symbolic-ref", "HEAD", "refs/heads/file-history").CombinedOutput(); err != nil {
		t.Fatalf("fixture HEAD: %v %s", err, out)
	}
	out, err := exec.Command("git", "-C", a.repo.Root, "log", "--follow", "--format=%H", "HEAD", "--", "same.go").Output()
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Fields(string(out))
	first, err := a.repo.FileHistoryPage(a.ctx, "same.go", nil, 0, 200)
	if err != nil {
		t.Fatal(err)
	}
	a.fileFilter = "same.go"
	a.fileCommits, a.fileHistoryTips, a.fileHistoryMore = first.Commits, first.Tips, first.HasMore
	a.worktreeSelected = false
	a.selectedSHA = want[17]
	a.commitFilesLoaded = true
	t.Cleanup(a.resetFileHistory)
	return a, callbacks, want
}

func TestFailedRefreshKeepsInterruptedHistoryRecoverable(t *testing.T) {
	for _, scope := range []string{"file", "main"} {
		t.Run(scope, func(t *testing.T) {
			var a *app
			var callbacks chan func()
			var want []string
			if scope == "file" {
				a, callbacks, want = automaticFileHistoryFixture(t, 420)
				a.loadFileHistory()
			} else {
				a, callbacks, want = automaticHistoryFixture(t, 420)
				a.loadMoreHistory()
			}
			old := historyCompletion(t, callbacks)
			metadata := filepath.Join(a.repo.Root, ".git")
			hidden := filepath.Join(a.repo.Root, ".git-unavailable")
			if err := os.Rename(metadata, hidden); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Rename(hidden, metadata) })
			a.reload()
			old()
			for a.loading {
				historyCompletion(t, callbacks)()
			}
			if a.loadErr == "" {
				t.Fatal("fixture refresh unexpectedly succeeded")
			}
			if scope == "file" {
				if !a.fileHistoryMore || a.fileHistoryLoading || !a.fileHistoryPaused && a.fileHistoryErr == "" {
					t.Fatal("failed refresh hides interrupted file traversal")
				}
			} else if !a.historyMore || a.historyLoadingMore || !a.historyPaused && a.historyPageErr == "" {
				t.Fatal("failed refresh falsely shows an active main-history read")
			}
			if err := os.Rename(hidden, metadata); err != nil {
				t.Fatal(err)
			}
			view := a.historyReadStatus
			if scope == "file" {
				view = a.fileHistoryReadStatus
			}
			tt := pageTester(a, view)
			retry, ok := tt.AccessByRole(ui.RoleButton, a.t("retry"))
			if !ok || retry.Disabled {
				t.Fatal("interrupted traversal has no enabled local retry")
			}
			tt.ClickAt(retry.Bounds.X+retry.Bounds.W/2, retry.Bounds.Y+retry.Bounds.H/2)
			for a.fileHistoryLoading || a.historyLoadingMore {
				historyCompletion(t, callbacks)()
			}
			if len(a.historySource()) != len(want) {
				t.Fatal("local retry did not complete pinned traversal")
			}
		})
	}
}

func TestFileHistoryRefreshCancelsAndReloadsToEnd(t *testing.T) {
	a, callbacks, want := automaticFileHistoryFixture(t, 420)
	a.loadFileHistory()
	ctx := a.fileHistoryCtx
	old := historyCompletion(t, callbacks)
	a.openRepository(a.repo.Root, false)
	if ctx.Err() == nil {
		t.Fatal("repository refresh left old file-history worker alive")
	}
	old()
	for a.loading || a.fileHistoryLoading {
		historyCompletion(t, callbacks)()
	}
	if len(a.fileCommits) != len(want) || a.fileHistoryMore {
		t.Fatalf("refresh stopped at %d of %d", len(a.fileCommits), len(want))
	}
}

// Detect capped traversal and selection resets when later pages append.
func TestFileHistoryAutomaticallyCompletesAndKeepsSelection(t *testing.T) {
	a, callbacks, want := automaticFileHistoryFixture(t, 2405)
	selected := a.selectedSHA
	a.loadFileHistory()
	for a.fileHistoryLoading {
		historyCompletion(t, callbacks)()
	}
	if len(a.fileCommits) != len(want) || a.fileHistoryMore {
		t.Fatalf("stopped at %d of %d", len(a.fileCommits), len(want))
	}
	for i, c := range a.fileCommits {
		if c.SHA != want[i] {
			t.Fatalf("wrong immutable identity at %d", i)
		}
	}
	if a.selectedSHA != selected {
		t.Fatal("appending pages replaced selected commit")
	}
}

func TestFileHistoryPageRejectsAbandonedContext(t *testing.T) {
	for _, change := range []string{"root", "generation", "path", "reset"} {
		t.Run(change, func(t *testing.T) {
			a, callbacks, _ := automaticFileHistoryFixture(t, 420)
			a.loadFileHistory()
			callback := historyCompletion(t, callbacks)
			before := len(a.fileCommits)
			switch change {
			case "root":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			case "generation":
				a.loadSeq++
			case "path":
				a.fileFilter = "different.go"
			case "reset":
				a.resetFileHistory()
			}
			callback()
			if len(a.fileCommits) != before {
				t.Fatal("abandoned page changed rows")
			}
		})
	}
}

func TestPausedFirstFileHistoryReadDoesNotClaimNoHistory(t *testing.T) {
	a := &app{fileFilter: "same.go", fileHistoryPaused: true}
	if a.historyListState(0) == historyListNoFileHistory {
		t.Fatal("cancelled first page claims the file has no history")
	}
}

func TestFileHistoryPauseRejectsQueuedPageAndRetryResumes(t *testing.T) {
	a, callbacks, want := automaticFileHistoryFixture(t, 420)
	a.loadFileHistory()
	ctx := a.fileHistoryCtx
	callback := historyCompletion(t, callbacks)
	before := len(a.fileCommits)
	a.pauseFileHistory()
	if ctx.Err() == nil {
		t.Fatal("pause did not cancel worker context")
	}
	callback()
	if len(a.fileCommits) != before || a.fileHistoryLoading || !a.fileHistoryPaused {
		t.Fatal("paused request changed partial results")
	}
	if a.historyListState(before) != historyListRows {
		t.Fatal("paused file history hides loaded rows")
	}
	tt := pageTester(a, a.fileHistoryReadStatus)
	retry, ok := tt.AccessByRole(ui.RoleButton, a.t("retry"))
	if !ok {
		t.Fatal("no resume action")
	}
	tt.ClickAt(retry.Bounds.X+retry.Bounds.W/2, retry.Bounds.Y+retry.Bounds.H/2)
	for a.fileHistoryLoading {
		historyCompletion(t, callbacks)()
	}
	if len(a.fileCommits) != len(want) || a.fileHistoryPaused {
		t.Fatal("retry did not finish pinned history")
	}
}

func TestFileHistoryFailureKeepsRowsAndRetriesPinnedPage(t *testing.T) {
	a, callbacks, want := automaticFileHistoryFixture(t, 420)
	original := a.repo
	a.repo = &gitrepo.Repository{Root: filepath.Join(t.TempDir(), "missing")}
	before := len(a.fileCommits)
	a.loadFileHistory()
	historyCompletion(t, callbacks)()
	if a.fileHistoryErr == "" || !a.fileHistoryMore || len(a.fileCommits) != before {
		t.Fatal("failed page lost rows or claimed completion")
	}
	if a.historyListState(before) != historyListRows {
		t.Fatal("partial failure hides browsable history")
	}
	a.repo = original
	tt := pageTester(a, a.fileHistoryReadStatus)
	retry, ok := tt.AccessByRole(ui.RoleButton, a.t("retry"))
	if !ok {
		t.Fatal("no retry action")
	}
	tt.ClickAt(retry.Bounds.X+retry.Bounds.W/2, retry.Bounds.Y+retry.Bounds.H/2)
	for a.fileHistoryLoading {
		historyCompletion(t, callbacks)()
	}
	if len(a.fileCommits) != len(want) || a.fileHistoryErr != "" {
		t.Fatal("retry lost pinned traversal")
	}
}
