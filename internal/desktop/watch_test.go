package desktop

import (
	"errors"
	"supergit/internal/gitrepo"
	"testing"
	"time"
)

func watchApp() *app {
	a := &app{repo: &gitrepo.Repository{Root: "/repo"}, path: "file.txt", diff: selectionDiff(), selectedLines: map[int]bool{2: true}, focused: true}
	a.guard.Switch()
	a.watch.stamp = "base"
	return a
}
func TestWatchCoalescesAndSkipsBusyWork(t *testing.T) {
	now := time.Now()
	a := watchApp()
	req, ok := a.beginProbe(now)
	if !ok {
		t.Fatal("idle probe refused")
	}
	if _, ok := a.beginProbe(now.Add(time.Minute)); ok {
		t.Fatal("overlapping probe started")
	}
	if a.acceptProbe(req, "base", a.diff, nil, now) {
		t.Fatal("unchanged repo refreshed")
	}
	if _, ok := a.beginProbe(now); ok {
		t.Fatal("rate limit ignored")
	}
	a.loading = true
	if _, ok := a.beginProbe(now.Add(time.Minute)); ok {
		t.Fatal("snapshot overlaps probe")
	}
	a.loading = false
	a.guard.StartWrite()
	if _, ok := a.beginProbe(now.Add(time.Minute)); ok {
		t.Fatal("write overlaps probe")
	}
	a.guard.EndWrite()
	a.diffLoading = true
	if _, ok := a.beginProbe(now.Add(time.Minute)); ok {
		t.Fatal("Diff load overlaps probe")
	}
}
func TestWatchDetectsDirtyContentAndPreservesEditingState(t *testing.T) {
	now := time.Now()
	a := watchApp()
	a.message = "commit draft"
	a.conflictText = "edited result"
	target := &actionTarget{Root: "/repo"}
	a.confirmation = target
	req, _ := a.beginProbe(now)
	if !a.acceptProbe(req, "base", gitrepo.Diff{Text: "edited again"}, nil, now) {
		t.Fatal("already-dirty file content change missed")
	}
	if a.message != "commit draft" || a.conflictText != "edited result" || a.confirmation != target {
		t.Fatal("probe overwrote user input or target")
	}
	if !a.selectedLines[2] {
		t.Fatal("probe mutated selection before Diff reload")
	}
}
func TestWatchRejectsOldRepositoryAndOldFileDiff(t *testing.T) {
	now := time.Now()
	a := watchApp()
	req, _ := a.beginProbe(now)
	a.path = "other.txt"
	if a.acceptProbe(req, "base", gitrepo.Diff{Text: "old file"}, nil, now) {
		t.Fatal("old file Diff triggered refresh")
	}
	req, _ = a.beginProbe(now.Add(time.Minute))
	a.guard.Switch()
	a.watch.reading = true
	if a.acceptProbe(req, "changed", gitrepo.Diff{}, nil, now) {
		t.Fatal("old repo result triggered refresh")
	}
	if !a.watch.reading {
		t.Fatal("old repo result cleared current in-flight read")
	}
}
func TestWatchBacksOffErrorsAndStopsAfterClose(t *testing.T) {
	now := time.Now()
	a := watchApp()
	req, _ := a.beginProbe(now)
	a.acceptProbe(req, "", gitrepo.Diff{}, errors.New("read failed"), now)
	if len(a.logs) != 1 {
		t.Fatal("probe error was not visible")
	}
	if _, ok := a.beginProbe(now.Add(4 * time.Second)); ok {
		t.Fatal("failed probe did not back off")
	}
	req, _ = a.beginProbe(now.Add(time.Minute))
	a.acceptProbe(req, "", gitrepo.Diff{}, errors.New("read failed"), now.Add(time.Minute))
	if len(a.logs) != 1 {
		t.Fatal("repeated error spammed log")
	}
	a.closed = true
	if _, ok := a.beginProbe(now.Add(2 * time.Minute)); ok {
		t.Fatal("closed app starts probe")
	}
}
func TestWatchDiffFailureCannotHideKnownRepositoryChanges(t *testing.T) {
	now := time.Now()
	a := watchApp()
	req, _ := a.beginProbe(now)
	if !a.acceptProbe(req, "changed", gitrepo.Diff{}, errors.New("selected Diff too large"), now) {
		t.Fatal("valid changed stamp was discarded because the selected Diff failed")
	}
}
func TestWatchDiffFailureDoesNotBackOffRepositoryDetection(t *testing.T) {
	now := time.Now()
	a := watchApp()
	req, _ := a.beginProbe(now)
	if a.acceptProbe(req, "base", gitrepo.Diff{}, errors.New("selected Diff too large"), now) {
		t.Fatal("unchanged metadata refreshed on Diff error")
	}
	if _, ok := a.beginProbe(now.Add(4 * time.Second)); !ok {
		t.Fatal("Diff error backed off metadata checks")
	}
}
func TestAutomaticUnchangedDiffKeepsSelection(t *testing.T) {
	a := watchApp()
	ticket := a.guard.Begin("diff")
	a.applyDiffResult(ticket, a.diff, false)
	if !a.selectedLines[2] {
		t.Fatal("unchanged Diff lost line selection")
	}
	a.applyDiffResult(ticket, gitrepo.Diff{Text: "external change"}, false)
	if len(a.selectedLines) != 0 {
		t.Fatal("changed Diff retained stale lines")
	}
}
