package lobedesk

import (
	"strings"
	"supergit/internal/gitrepo"
	"testing"
	"time"
)

// A refresh marks known rows pending while their detail read is queued. A
// child checkout that moves in that window must still reconcile the parent
// gitlink diff, which was read before the new observation landed.
func TestPendingChildHeadChangeReconcilesParent(t *testing.T) {
	a := &app{submodules: []gitrepo.Submodule{{Path: "child", CheckoutKnown: true, Initialized: true, Current: "B", Index: "A", Recorded: "A", Pending: true}}}
	if !a.applySubmoduleUpdates([]gitrepo.Submodule{{Path: "child", Initialized: true, Current: "C"}}) {
		t.Fatal("pending known child moved B to C without parent reconciliation")
	}
	if got := a.submodules[0].Current; got != "C" {
		t.Fatalf("reconciled row kept stale checkout %q", got)
	}
}

func TestPendingChildDetailReadReconcilesParentDiff(t *testing.T) {
	a, cbs, _ := worktreeApp(t)
	a.selectedPath = "base.txt"
	a.loadSeq = 3
	a.submodules = []gitrepo.Submodule{{
		Path: "module", Initialized: true, CheckoutKnown: true,
		Current: "B", Index: "A", Recorded: "A", Pending: true,
	}}
	a.applySubmoduleDetail(3, a.repo, gitrepo.Submodule{Path: "module", Initialized: true, Current: "C"})
	if a.diffSeq != 1 {
		t.Fatalf("moved pending child did not re-read working-tree details: %d", a.diffSeq)
	}
	for a.diffLoading {
		runContinuation(t, cbs)
	}
}

func TestChildErrorRecoveryReconcilesMovedKnownHead(t *testing.T) {
	a := &app{submodules: []gitrepo.Submodule{{Path: "child", Initialized: true, Current: "B", Index: "A", Recorded: "A", Error: "temporarily unavailable"}}}
	if !a.applySubmoduleUpdates([]gitrepo.Submodule{{Path: "child", Initialized: true, Current: "C"}}) {
		t.Fatal("recovered child moved B to C without parent reconciliation")
	}
	a.submodules = []gitrepo.Submodule{{Path: "child", Error: "first read failed"}}
	if a.applySubmoduleUpdates([]gitrepo.Submodule{{Path: "child", Initialized: true, Current: "C"}}) {
		t.Fatal("first successful observation unnecessarily reopened parent")
	}
}

func TestRefReloadReconcilesPrunedSelectionAndDetails(t *testing.T) {
	a, cbs, root := worktreeApp(t)
	a.openRepository(root, false)
	for a.loading || a.commitFilesLoading || a.commitDiffLoading {
		runContinuation(t, cbs)
	}
	a.worktreeSelected = false
	a.selectedSHA = "pruned-commit"
	a.commits = append(a.commits, gitrepo.Commit{SHA: "pruned-commit"})
	a.commitPath = "stale-detail"
	a.reloadHistoryFromTip("")
	for a.historyReloading {
		runContinuation(t, cbs)
	}
	if a.selectedSHA != a.status.Head {
		t.Fatalf("completed first page retained pruned selection %q", a.selectedSHA)
	}
	if !a.commitFilesLoading {
		t.Fatal("replacement selection did not reload commit details")
	}
	for a.commitFilesLoading || a.commitDiffLoading {
		runContinuation(t, cbs)
	}
	if len(a.commitFiles) != 1 || a.commitFiles[0].Path != "base.txt" {
		t.Fatalf("wrong replacement details: %+v", a.commitFiles)
	}
}

func TestRefReloadKeepsSelectionUntilLastPage(t *testing.T) {
	a, cbs, want := automaticHistoryFixture(t, 420)
	a.worktreeSelected = false
	a.selectedSHA = want[len(want)-1]
	selected := a.selectedSHA
	a.reloadHistoryFromTip("")
	historyCompletion(t, cbs)()
	if !a.historyMore || a.selectedSHA != selected {
		t.Fatal("partial first page discarded selection on later page")
	}
	for a.historyMore || a.historyLoadingMore {
		historyCompletion(t, cbs)()
	}
	if a.selectedSHA != selected {
		t.Fatal("reachable selection was lost")
	}
}
func TestKnownUninitializedChildRecoveryReconciles(t *testing.T) {
	a := &app{submodules: []gitrepo.Submodule{{Path: "child", CheckoutKnown: true, Error: "read failed"}}}
	if !a.applySubmoduleUpdates([]gitrepo.Submodule{{Path: "child", Initialized: true, Current: "C"}}) {
		t.Fatal("known uninitialized state was treated as unknown")
	}
}
func TestChildErrorRecoveryWithIdenticalParentPorcelain(t *testing.T) {
	a, cbs, child := submoduleChangesFixture(t)
	submoduleFixtureGit(t, child, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-m", "B")
	a.openRepository(a.repo.Root, false)
	for a.loading || pendingChildDetails(a) || a.localWatch.initializing {
		runContinuation(t, cbs)
	}
	before, e := a.repo.StatusText(a.ctx)
	if e != nil {
		t.Fatal(e)
	}
	epoch := a.loadSeq
	a.applySubmoduleUpdates([]gitrepo.Submodule{{Path: "module", Error: "temporary failure"}})
	submoduleFixtureGit(t, child, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-m", "C")
	after, e := a.repo.StatusText(a.ctx)
	if e != nil || before != after {
		t.Fatalf("fixture parent porcelain changed: %v", e)
	}
	a.pollLocalChanges(a.localWatch.next.Add(time.Hour))
	for a.localWatch.reading || a.loading || pendingChildDetails(a) {
		runContinuation(t, cbs)
	}
	if a.loadSeq == epoch {
		t.Fatal("recovered child HEAD did not refresh parent snapshot")
	}
	current := strings.TrimSpace(submoduleFixtureGit(t, child, "rev-parse", "HEAD"))
	if a.submodules[0].Current != current {
		t.Fatal("child HEAD remains stale")
	}
}
