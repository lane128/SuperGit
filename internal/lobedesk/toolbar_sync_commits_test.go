package lobedesk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestSyncMenuBrowsesBothSidesOfRealDivergence(t *testing.T) {
	a, _, root := worktreeApp(t)
	callbacks := make(chan func(), 64)
	a.runOnMain = func(fn func()) { callbacks <- fn }
	git := func(args ...string) string { return strings.TrimSpace(selectionGit(t, root, args...)) }
	tree, base := git("rev-parse", "HEAD^{tree}"), git("rev-parse", "HEAD")
	upstream := git("commit-tree", tree, "-p", base, "-m", "Only upstream")
	git("remote", "add", "origin", filepath.Join(t.TempDir(), "absent"))
	git("update-ref", "refs/remotes/origin/main", upstream)
	git("branch", "--set-upstream-to=origin/main")
	head := base
	for i := 0; i <= syncCommitPageSize; i++ {
		head = git("commit-tree", tree, "-p", head, "-m", fmt.Sprintf("Only local %d", i))
	}
	git("update-ref", "refs/heads/main", head, base)
	git("config", "status.aheadBehind", "false")
	var err error
	a.status, err = a.repo.Status(a.ctx)
	if err != nil || a.status.Ahead != syncCommitPageSize+1 || a.status.Behind != 1 {
		t.Fatalf("divergence: %+v, %v", a.status, err)
	}
	a.remotes = []gitrepo.Remote{{Name: "origin"}}
	a.locale, a.page = localeEnglish, "branches"
	view := pageTester(a, a.titleBar)
	click := func(role ui.Role, label string) { t.Helper(); clickRepositoryTabsControl(t, view, role, label) }
	wait := func() {
		t.Helper()
		for a.syncCommits != nil && a.syncCommits.cancel != nil {
			runContinuation(t, callbacks)
		}
		view.Frame()
		if a.syncCommits == nil || a.syncCommits.err != "" {
			t.Fatalf("commit list unavailable: %+v", a.syncCommits)
		}
	}
	click(ui.RoleMenuButton, "Sync")
	if !view.HasText(a.t("syncToPush")) || !view.HasText(a.t("syncToPull")) ||
		!view.HasText(fmt.Sprint(a.status.Ahead)) || !view.HasText("1") {
		t.Fatal("Sync menu does not draw both pending rows with their counts")
	}
	click(ui.RoleMenuItem, a.tf("branchAhead", syncCommitPageSize+1))
	wait()
	if len(a.syncCommits.commits) != syncCommitPageSize || a.syncCommits.commits[0].SHA != head || !a.syncCommits.more {
		t.Fatalf("unexpected first page: %+v", a.syncCommits)
	}
	// A live tracking ref can move between pages. Paging still describes
	// the original pair of tips until the popover is dismissed.
	git("update-ref", "refs/remotes/origin/main", head, upstream)
	click(ui.RoleMenuItem, "Next page")
	wait()
	oldest := git("rev-parse", fmt.Sprintf("HEAD~%d", syncCommitPageSize))
	if len(a.syncCommits.commits) != 1 || a.syncCommits.commits[0].SHA != oldest || a.syncCommits.more {
		t.Fatalf("unexpected last page: %+v", a.syncCommits)
	}
	click(ui.RoleMenuItem, "Previous page")
	wait()
	click(ui.RoleMenuItem, "Back to Sync")
	git("update-ref", "refs/remotes/origin/main", upstream, head)
	click(ui.RoleMenuItem, a.tf("branchBehind", 1))
	wait()
	if len(a.syncCommits.commits) != 1 || a.syncCommits.commits[0].SHA != upstream {
		t.Fatalf("incoming side: %+v", a.syncCommits)
	}
	// The previous History view did not even include the upstream ref.
	a.fileFilter = "base.txt"
	a.historyQuery = "unrelated"
	click(ui.RoleMenuItem, shortSHA(upstream)+" · Only upstream")
	for a.historyReloading {
		runContinuation(t, callbacks)
	}
	if a.syncOpen || a.page != "history" || a.selectedSHA != upstream || !a.revealSelection || a.fileFilter != "" || a.historyQuery != "" {
		t.Fatalf("failed to reveal full commit in history: %v %q %q", a.syncOpen, a.page, a.selectedSHA)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "FETCH_HEAD")); !os.IsNotExist(err) {
		t.Fatal("viewing a comparison initiated a fetch")
	}
}

func TestSyncMenuZeroGoneAndStaleRead(t *testing.T) {
	a, callbacks, root := worktreeApp(t)
	a.locale = localeEnglish
	a.status = gitrepo.Status{Branch: "main", Upstream: "origin/main", Head: strings.TrimSpace(selectionGit(t, root, "rev-parse", "HEAD"))}
	view := pageTester(a, a.titleBar)
	clickRepositoryTabsControl(t, view, ui.RoleMenuButton, "Sync")
	for _, label := range []string{"0 ahead", "0 behind"} {
		if item, ok := view.AccessByRole(ui.RoleMenuItem, label); !ok || !item.Disabled {
			t.Errorf("empty side is actionable or missing: %+v", item)
		}
	}
	a.status.Ahead = 1
	view.Frame()
	clickRepositoryTabsControl(t, view, ui.RoleMenuItem, "1 ahead")
	runContinuation(t, callbacks)
	view.Frame()
	if a.syncCommits == nil || !view.HasText(a.t("syncTrackingChanged")) {
		t.Fatalf("tracking error was hidden: %+v", a.syncCommits)
	}
	clickRepositoryTabsControl(t, view, ui.RoleMenuItem, a.t("retry"))
	pending := a.syncCommits
	a.loadSeq++
	runContinuation(t, callbacks)
	if pending.cancel == nil || a.syncCommitsCurrent(pending) {
		t.Fatal("a refresh accepted a previous generation's read")
	}
	a.status.UpstreamGone = true
	view.Frame()
	if a.syncCommits != nil || !view.HasText("origin/main · "+a.t("upstreamGone")) {
		t.Fatal("removed tracking target reported as synchronized")
	}
}
