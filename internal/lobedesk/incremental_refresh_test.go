package lobedesk

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"supergit/internal/gitrepo"
	"testing"
	"time"
)

func TestBlockedChildAllowsParentPublication(t *testing.T) {
	a, cbs, child := submoduleChangesFixture(t)
	fault := setupSubmoduleStatusFault(t, child, a.repo.Root)
	writeSubmoduleFixtureFile(t, filepath.Dir(fault), filepath.Base(fault), "child-timeout\n")
	a.openRepository(a.repo.Root, false)
	deadline := time.After(6 * time.Second)
	for a.loading {
		select {
		case cb := <-cbs:
			cb()
		case <-deadline:
			t.Fatal("parent publication waited for blocked child")
		}
	}
	if a.status.Head == "" || len(a.submodules) != 1 {
		t.Fatal("incomplete parent snapshot")
	}
}

func TestRefOnlyWatcherPreservesParentGeneration(t *testing.T) {
	a, cbs, _ := worktreeApp(t)
	a.openRepository(a.repo.Root, false)
	for a.loading {
		runContinuation(t, cbs)
	}
	epoch := a.loadSeq
	submoduleFixtureGit(t, a.repo.Root, "branch", "external-ref")
	for range 2 {
		a.pollLocalChanges(a.localWatch.next.Add(time.Hour))
		for a.loading || a.localWatch.reading {
			runContinuation(t, cbs)
		}
	}
	if a.loadSeq != epoch {
		t.Fatal("ref change reopened the checkout")
	}
	for _, b := range a.branches {
		if b.Name == "external-ref" {
			return
		}
	}
	t.Fatal("ref reconciliation lost external branch")
}

func interceptFixtureGit(t *testing.T, body string) {
	t.Helper()
	log := gitCallLog(t)
	shim := filepath.Join(filepath.Dir(log), "git")
	data, err := os.ReadFile(shim)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(shim, []byte(strings.Replace(string(data), `exec "$SUPERGIT_REAL_GIT"`, body+"\n"+`exec "$SUPERGIT_REAL_GIT"`, 1)), 0755); err != nil {
		t.Fatal(err)
	}
}
func TestInitialPublicationRetriesChangingCheckout(t *testing.T) {
	for _, kind := range []string{"head", "index", "operation"} {
		t.Run(kind, func(t *testing.T) {
			a, cbs, root := worktreeApp(t)
			t.Setenv("RACE_KIND", kind)
			t.Setenv("RACE_ROOT", root)
			t.Setenv("RACE_ONCE", filepath.Join(t.TempDir(), "changed"))
			writeSubmoduleFixtureFile(t, root, "race.txt", "stage me")
			interceptFixtureGit(t, `case " $* " in *" status "*) if mkdir "$RACE_ONCE" 2>/dev/null; then
 "$SUPERGIT_REAL_GIT" "$@" || exit 1
 case "$RACE_KIND" in head) "$SUPERGIT_REAL_GIT" -C "$RACE_ROOT" commit --quiet --allow-empty -m raced;; index) "$SUPERGIT_REAL_GIT" -C "$RACE_ROOT" add race.txt;; operation) "$SUPERGIT_REAL_GIT" -C "$RACE_ROOT" rev-parse HEAD > "$RACE_ROOT/.git/MERGE_HEAD";; esac;exit $?;fi;;esac`)
			a.openRepository(root, false)
			for a.loading {
				runContinuation(t, cbs)
			}
			if a.loadErr != "" {
				t.Fatal(a.loadErr)
			}
			switch kind {
			case "head":
				if a.headMessage != "raced" {
					t.Fatal(a.headMessage)
				}
			case "index":
				if len(a.status.Entries) != 1 || a.status.Entries[0].Index != 'A' {
					t.Fatal(a.status)
				}
			case "operation":
				if a.operation.Kind != "merge" {
					t.Fatal(a.operation)
				}
			}
		})
	}
}
func TestCancelledSecondRemoteReconcilesFirstRemote(t *testing.T) {
	a, cbs, root := worktreeApp(t)
	t.Setenv("GIT_ALLOW_PROTOCOL", "file")
	bare := filepath.Join(t.TempDir(), "remote.git")
	pullFixtureGit(t, root, "clone", "--bare", root, bare)
	pullFixtureGit(t, root, "remote", "add", "first", bare)
	pullFixtureGit(t, root, "remote", "add", "second", bare)
	a.openRepository(root, false)
	for a.loading || a.localWatch.initializing {
		runContinuation(t, cbs)
	}
	epoch := a.loadSeq
	marker := filepath.Join(t.TempDir(), "second")
	t.Setenv("FETCH_MARKER", marker)
	interceptFixtureGit(t, `case "$*" in *" fetch "*"-- second") touch "$FETCH_MARKER";exec sleep 60;;esac`)
	a.fetchAll()
	deadline := time.After(10 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		select {
		case f := <-cbs:
			f()
		case <-time.After(10 * time.Millisecond):
		case <-deadline:
			t.Fatal("second remote never started")
		}
	}
	a.cancelOperation()
	for a.busy || a.fetching || a.loading || a.historyReloading {
		runContinuation(t, cbs)
	}
	if a.loadSeq != epoch || len(a.remoteBranches) != 1 || a.remoteBranches[0].Name != "first/main" {
		t.Fatalf("partial fetch reconciliation: epoch=%d refs=%v", a.loadSeq, a.remoteBranches)
	}
	if n := a.notices[toastMain]; len(n) != 1 || n[0].title != a.t("operationCancelled") {
		t.Fatalf("wrong cancellation outcome: %v", n)
	}
}

func TestInitialWatcherRetainsEditAfterStatus(t *testing.T) {
	a, cbs, root := worktreeApp(t)
	t.Setenv("EDIT_ONCE", filepath.Join(t.TempDir(), "changed"))
	t.Setenv("RACE_ROOT", root)
	interceptFixtureGit(t, `case " $* " in *" status "*) if mkdir "$EDIT_ONCE" 2>/dev/null; then "$SUPERGIT_REAL_GIT" "$@" || exit 1; touch "$RACE_ROOT/late.txt";exit $?;fi;;esac`)
	a.openRepository(root, false)
	for a.loading || a.localWatch.initializing {
		runContinuation(t, cbs)
	}
	a.pollLocalChanges(time.Now().Add(time.Hour))
	for a.localWatch.reading || a.loading {
		runContinuation(t, cbs)
	}
	if len(a.status.Entries) != 1 || a.status.Entries[0].Path != "late.txt" {
		t.Fatalf("initial baseline hid edit: %+v", a.status)
	}
}

// Working-tree details can be read before the watcher baseline finishes
// observing already-dirty files. Those reads must be repeated after the
// baseline lands, or the save between them becomes invisible to polling.
func TestLateWatcherBaselineReconcilesWorkingTreeDetails(t *testing.T) {
	a, cbs, _ := worktreeApp(t)
	a.selectedPath = "base.txt"
	baseline := make(chan string, 1)
	a.installWatchBaseline(a.loadSeq, a.repo, baseline)
	a.loadDiff()
	if !a.localWatch.detailsPending {
		t.Fatal("diff read before the baseline was not remembered")
	}
	if a.diffSeq != 1 {
		t.Fatalf("diff read did not start: %d", a.diffSeq)
	}
	baseline <- "baseline"
	for a.localWatch.initializing {
		runContinuation(t, cbs)
	}
	if a.diffSeq != 2 {
		t.Fatalf("late baseline did not re-read working-tree details: %d", a.diffSeq)
	}
	for a.diffLoading {
		runContinuation(t, cbs)
	}
}

func TestExternalCheckoutMutationDuringFetch(t *testing.T) {
	for _, kind := range []string{"head", "index", "operation"} {
		t.Run(kind, func(t *testing.T) {
			a, cbs, root := worktreeApp(t)
			bare := filepath.Join(t.TempDir(), "remote.git")
			pullFixtureGit(t, root, "clone", "--bare", root, bare)
			pullFixtureGit(t, root, "remote", "add", "origin", bare)
			writeSubmoduleFixtureFile(t, root, "race.txt", "contents")
			a.openRepository(root, false)
			for a.loading || a.localWatch.initializing {
				runContinuation(t, cbs)
			}
			epoch := a.loadSeq
			t.Setenv("RACE_ROOT", root)
			t.Setenv("RACE_KIND", kind)
			interceptFixtureGit(t, `case " $* " in *" fetch "*) case "$RACE_KIND" in head) "$SUPERGIT_REAL_GIT" -C "$RACE_ROOT" commit --quiet --allow-empty -m raced;; index) "$SUPERGIT_REAL_GIT" -C "$RACE_ROOT" add race.txt;; operation) "$SUPERGIT_REAL_GIT" -C "$RACE_ROOT" rev-parse HEAD > "$RACE_ROOT/.git/MERGE_HEAD";; esac;;esac`)
			a.fetchAll()
			for a.fetching || a.busy || a.loading {
				runContinuation(t, cbs)
			}
			if epoch == a.loadSeq || a.loadErr != "" {
				t.Fatalf("unreconciled local change: %d %s", a.loadSeq, a.loadErr)
			}
			switch kind {
			case "head":
				if a.headMessage != "raced" {
					t.Fatal(a.headMessage)
				}
			case "index":
				if len(a.status.Entries) != 1 || a.status.Entries[0].Index != 'A' {
					t.Fatal(a.status)
				}
			case "operation":
				if a.operation.Kind != "merge" {
					t.Fatal(a.operation)
				}
			}
		})
	}
}
func TestFailedChildKeepsLastKnownCount(t *testing.T) {
	a, cbs, child := submoduleChangesFixture(t)
	writeSubmoduleFixtureFile(t, child, "untracked", "dirty")
	a.openRepository(a.repo.Root, false)
	for a.loading || pendingChildDetails(a) {
		runContinuation(t, cbs)
	}
	fault := setupSubmoduleStatusFault(t, child, a.repo.Root)
	writeSubmoduleFixtureFile(t, filepath.Dir(fault), filepath.Base(fault), "child-error\n")
	a.openRepository(a.repo.Root, false)
	epoch := a.loadSeq
	a.openRepository(a.repo.Root, false)
	if a.loadSeq != epoch {
		t.Fatal("overlapping manual refresh restarted traversal")
	}
	for a.loading || pendingChildDetails(a) {
		runContinuation(t, cbs)
	}
	m := a.submodules[0]
	if m.Error == "" || m.Pending || m.ChangedFiles != 1 || !m.Dirty {
		t.Fatalf("error erased dirty state: %+v", m)
	}
}

func TestSupersededBaselineAndChildCallbackCannotPublish(t *testing.T) {
	a, cbs, root := worktreeApp(t)
	pg := a.repo
	a.loadSeq = 1
	stamp := make(chan string, 1)
	a.installWatchBaseline(1, pg, stamp)
	a.publishSubmoduleInventory(a.ctx, 1, pg, []gitrepo.Submodule{{Path: "missing", Pending: true}}, true, false, nil)
	var done func()
	select {
	case done = <-cbs:
	case <-time.After(5 * time.Second):
		t.Fatal("missing child callback")
	}
	next, err := gitrepo.Open(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	a.repo = next
	a.loadSeq++
	a.localWatch.stamp = "new"
	a.localWatch.initializing = false
	a.submodules = []gitrepo.Submodule{{Path: "missing", Dirty: true, ChangedFiles: 9}}
	stamp <- "old"
	done()
	runContinuation(t, cbs)
	if a.localWatch.stamp != "new" || a.submodules[0].ChangedFiles != 9 {
		t.Fatal("superseded observation published into replacement checkout")
	}
}
