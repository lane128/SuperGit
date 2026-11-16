package lobedesk

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func automaticHistoryFixture(t *testing.T, count int) (*app, chan func(), []string) {
	t.Helper()
	a, callbacks := overviewFixture(t)
	var input strings.Builder
	for i := range count {
		message := fmt.Sprintf("Automatic commit %04d", i)
		fmt.Fprintf(&input, "commit refs/heads/automatic-history\ncommitter Fixture <fixture@example.invalid> %d +0000\ndata %d\n%s\n\n", 1700000000+i, len(message), message)
	}
	input.WriteString("done\n")
	cmd := exec.Command("git", "-C", a.repo.Root, "fast-import", "--quiet")
	cmd.Stdin = strings.NewReader(input.String())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture import: %v %s", err, out)
	}
	out, err := exec.Command("git", "-C", a.repo.Root, "log", "--topo-order", "--all", "HEAD", "--format=%H").Output()
	if err != nil {
		t.Fatal(err)
	}
	first, err := a.repo.HistoryPage(a.ctx, nil, 0, 200)
	if err != nil {
		t.Fatal(err)
	}
	a.commits, a.historyTips, a.historyMore = first.Commits, first.Tips, first.HasMore
	return a, callbacks, strings.Fields(string(out))
}

func historyCompletion(t *testing.T, callbacks chan func()) func() {
	t.Helper()
	select {
	case callback := <-callbacks:
		return callback
	case <-time.After(3 * time.Second):
		t.Fatal("automatic history did not deliver its page completion")
		return nil
	}
}

func TestHistoryLoadsToTraversalEndWithoutMoreButton(t *testing.T) {
	a, callbacks, want := automaticHistoryFixture(t, 2401)
	started := time.Now()
	a.loadMoreHistory()
	for a.historyMore {
		historyCompletion(t, callbacks)()
	}
	if len(a.commits) != len(want) || a.historyLoadingMore {
		t.Fatalf("history stopped at %d of %d", len(a.commits), len(want))
	}
	for i, commit := range a.commits {
		if commit.SHA != want[i] {
			t.Fatalf("history order/identity differs at %d", i)
		}
	}
	t.Logf("loaded %d commits in %s", len(a.commits), time.Since(started))
}

func TestAutomaticHistoryStartsWhenRefReloadFinishes(t *testing.T) {
	a, callbacks, want := automaticHistoryFixture(t, 420)
	a.reloadHistoryFromTip("")
	for a.historyReloading || a.historyMore {
		historyCompletion(t, callbacks)()
	}
	if len(a.commits) != len(want) || a.historyLoadingMore {
		t.Fatal("first-page ref reload did not automatically continue to the end")
	}
}

func TestAutomaticHistoryReadRejectsOldContext(t *testing.T) {
	for _, change := range []string{"root", "generation", "ref", "file filter"} {
		t.Run(change, func(t *testing.T) {
			a, callbacks, _ := automaticHistoryFixture(t, 420)
			a.loadMoreHistory()
			callback := historyCompletion(t, callbacks)
			before := len(a.commits)
			switch change {
			case "root":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			case "generation":
				a.loadSeq++
			case "ref":
				a.historyReload++
			case "file filter":
				a.fileFilter = "other.txt"
			}
			callback()
			if len(a.commits) != before {
				t.Fatal("old context appended history rows")
			}
		})
	}
}

func TestAutomaticHistoryFailureIsNotReportedAsComplete(t *testing.T) {
	a, callbacks, want := automaticHistoryFixture(t, 420)
	original := a.repo
	a.repo = &gitrepo.Repository{Root: filepath.Join(t.TempDir(), "missing")}
	before := len(a.commits)
	a.loadMoreHistory()
	historyCompletion(t, callbacks)()
	if !a.historyMore || a.historyLoadingMore || len(a.commits) != before {
		t.Fatal("failed page lost partial rows or claimed traversal completion")
	}
	if a.historyPageErr == "" {
		t.Fatal("partial read failure has no persistent error state")
	}
	a.repo = original
	tt := pageTester(a, a.historyReadStatus)
	button, ok := tt.AccessByRole(ui.RoleButton, a.t("retry"))
	if !ok {
		t.Fatal("failed partial read has no retry action")
	}
	tt.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
	for a.historyMore {
		historyCompletion(t, callbacks)()
	}
	if len(a.commits) != len(want) || a.historyPageErr != "" {
		t.Fatal("retry did not recover the pinned traversal")
	}
}

func TestAutomaticHistoryCancelRejectsQueuedPageAndCanResume(t *testing.T) {
	a, callbacks, want := automaticHistoryFixture(t, 420)
	a.loadMoreHistory()
	ctx := a.historyReadCtx
	callback := historyCompletion(t, callbacks)
	before := len(a.commits)
	a.pauseHistoryTraversal()
	if ctx.Err() == nil || !a.historyPaused || a.historyLoadingMore || !a.historyMore {
		t.Fatal("cancel did not stop the worker while preserving a partial traversal")
	}
	callback()
	if len(a.commits) != before {
		t.Fatal("queued page returned after cancellation")
	}
	tt := pageTester(a, a.historyReadStatus)
	button, ok := tt.AccessByRole(ui.RoleButton, a.t("retry"))
	if !ok {
		t.Fatal("paused traversal has no resume action")
	}
	tt.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
	for a.historyMore {
		historyCompletion(t, callbacks)()
	}
	if len(a.commits) != len(want) || a.historyPaused {
		t.Fatal("resume lost pinned rows or stopped before the end")
	}
}

func TestAutomaticHistoryJumpResumesAfterFileFilter(t *testing.T) {
	a, _, want := automaticHistoryFixture(t, 420)
	a.fileFilter, a.historyPaused = "other.txt", true
	a.jumpToCommit(want[len(want)-1])
	if a.fileFilter != "" || a.historyPaused || !a.historyLoadingMore {
		t.Fatal("jump beyond a file filter did not resume the full traversal")
	}
}

func TestHistoryListBuildsOnlyViewportRows(t *testing.T) {
	a, _, _ := automaticHistoryFixture(t, 2401)
	all, err := a.repo.ReadHistory(a.ctx, 3000)
	if err != nil {
		t.Fatal(err)
	}
	a.commits, a.historyMore = all.Commits, false
	a.locale, a.page, a.showGraph = localeEnglish, "history", true
	tt := pageTester(a, a.historyPage)
	tt.Frame()
	visible := 0
	for _, commit := range a.commits {
		if _, ok := tt.AccessByRole(ui.RoleOption, commit.Subject); ok {
			visible++
		}
	}
	if visible == 0 || visible > 100 {
		t.Fatalf("built %d commit controls for one viewport from %d rows", visible, len(a.commits))
	}
	t.Logf("built %d controls for %d history rows", visible, len(a.commits))
	last := a.commits[len(a.commits)-1]
	a.worktreeSelected = false
	a.selectedSHA, a.revealSelection = last.SHA, true
	tt.Frame()
	tt.Frame()
	node, ok := tt.AccessByRole(ui.RoleOption, last.Subject)
	if !ok || node.Bounds.Y < a.historyViewport.Y || node.Bounds.Y+node.Bounds.H > a.historyViewport.Y+a.historyViewport.H {
		t.Fatal("virtualized history did not reveal a distant keyboard target inside its viewport")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, a.t("loadMoreHistory")); ok {
		t.Fatal("complete traversal exposes a manual Load More button")
	}
}

func TestSidebarShowsEveryExpandedReference(t *testing.T) {
	a := &app{locale: localeEnglish, page: "history", repo: &gitrepo.Repository{Root: t.TempDir()}, sidebarOpen: map[string]bool{}}
	for _, group := range []string{"branches", "remotes", "tags", "stashes", "worktrees", "submodules"} {
		a.sidebarOpen[group] = true
	}
	var names []string
	for i := range 24 {
		name := fmt.Sprintf("%02d", i)
		a.branches = append(a.branches, gitrepo.Branch{Name: "branch-" + name})
		a.remoteBranches = append(a.remoteBranches, gitrepo.Branch{Name: "remote-" + name})
		a.tags = append(a.tags, gitrepo.Tag{Name: "tag-" + name})
		a.stashes = append(a.stashes, gitrepo.Stash{Message: "stash-" + name, SHA: name})
		a.worktrees = append(a.worktrees, gitrepo.Worktree{Path: "/fixture/worktree-" + name})
		a.submodules = append(a.submodules, gitrepo.Submodule{Name: "module-" + name, Path: "module-" + name})
		for _, prefix := range []string{"branch-", "remote-", "tag-", "stash-", "worktree-", "module-"} {
			names = append(names, prefix+name)
		}
	}
	tt := sidebarTester(a)
	for _, name := range names {
		if _, ok := tt.AccessByRole(ui.RoleButton, name); !ok {
			t.Fatalf("expanded sidebar omitted %s", name)
		}
	}
}
