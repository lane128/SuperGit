package lobedesk

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

func submoduleFixtureGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %q: %v\n%s", args, err, out)
	}
	return string(out)
}

func writeSubmoduleFixtureFile(t *testing.T, dir, path, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, path), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func submoduleChangesFixture(t *testing.T) (*app, <-chan func(), string) {
	t.Helper()
	app, callbacks, parent := worktreeApp(t)
	_, _, source := worktreeApp(t)
	writeSubmoduleFixtureFile(t, source, "second.txt", "base\n")
	submoduleFixtureGit(t, source, "add", "second.txt")
	submoduleFixtureGit(t, source, "commit", "-m", "Add second file")
	submoduleFixtureGit(t, parent, "-c", "protocol.file.allow=always", "submodule", "add", source, "module")
	submoduleFixtureGit(t, parent, "commit", "-m", "Add submodule")
	app.locale = localeEnglish
	app.sidebarOpen = map[string]bool{"submodules": true}
	return app, callbacks, filepath.Join(parent, "module")
}

func TestParentWorkspaceShowsGitlinkChanges(t *testing.T) {
	a, _, child := submoduleChangesFixture(t)
	writeSubmoduleFixtureFile(t, child, "base.txt", "child edit\n")
	writeSubmoduleFixtureFile(t, child, "new.txt", "new child file\n")
	writeSubmoduleFixtureFile(t, a.repo.Root, "base.txt", "parent edit\n")
	load := func() {
		t.Helper()
		var err error
		a.status, err = a.repo.Status(t.Context())
		if err != nil {
			t.Fatal(err)
		}
	}
	load()
	if entries := a.unstagedEntries(); len(entries) != 1 || entries[0].Path != "base.txt" {
		t.Fatalf("dirty child appeared as a parent file change: %+v", entries)
	}
	for _, kind := range []string{"stage", "stageTracked", "discard"} {
		targets, reason := a.batchFileTargets(kind)
		if reason != "" || len(targets) != 1 || targets[0].Entry.Path != "base.txt" {
			t.Fatalf("%s cannot target just the changed parent file: %+v, %q", kind, targets, reason)
		}
	}
	submoduleFixtureGit(t, child, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-am", "Move submodule HEAD")
	load()
	if entries := a.unstagedEntries(); len(entries) != 2 || entries[1].Path != "module" {
		t.Fatalf("moved child HEAD is missing from parent files: %+v", entries)
	}
	if targets, reason := a.batchFileTargets("stage"); reason != "" || len(targets) != 2 {
		t.Fatalf("moved gitlink is not stageable: %+v, %q", targets, reason)
	}
	if targets, reason := a.batchFileTargets("discard"); reason == "" || len(targets) != 0 {
		t.Fatalf("discard included a protected gitlink: %+v, %q", targets, reason)
	}
	submoduleFixtureGit(t, a.repo.Root, "add", "module")
	load()
	a.selectedPath, a.selectedStaged = "module", false
	a.reconcileSelection()
	if entries := a.stagedEntries(); len(entries) != 1 || entries[0].Path != "module" || !a.selectedStaged {
		t.Fatalf("the staged gitlink has an incorrect selection or row: %+v, staged=%t", entries, a.selectedStaged)
	}
	if entries := a.unstagedEntries(); len(entries) != 1 || entries[0].Path != "base.txt" {
		t.Fatalf("dirty child duplicated the staged pointer: %+v", entries)
	}
}

func TestSidebarSubmoduleBadgeCountsDistinctFiles(t *testing.T) {
	a, _, child := submoduleChangesFixture(t)
	submoduleFixtureGit(t, child, "rm", "base.txt")
	writeSubmoduleFixtureFile(t, child, "base.txt", "recreated\n")
	writeSubmoduleFixtureFile(t, child, "second.txt", "modified\n")
	var err error
	a.submodules, err = a.repo.Submodules(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	tt := sidebarTester(a)
	row, ok := tt.AccessByRole(ui.RoleButton, "module")
	if !ok || !strings.Contains(row.Description, "2 files changed") {
		t.Fatalf("submodule row did not describe the two changed paths: %q", row.Description)
	}
	badge, ok := tt.Find("2")
	if !ok || badge.X <= row.Bounds.X+row.Bounds.W/2 || badge.X+badge.W > row.Bounds.X+row.Bounds.W {
		t.Fatalf("count missing from the right side of the row: row=%+v count=%+v", row.Bounds, badge)
	}
}

func TestSubmoduleCountLeavesRoomForThePathTail(t *testing.T) {
	a := &app{locale: localeEnglish}
	const tail = "Previews"
	const ref = "feature/long-branch-name-with-many-segments"
	var tailWidth float32
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		tailWidth, _ = c.MeasureText(0, ui.Span{Text: tail, Size: scaledFont(11)})
		ui.Column(c).Width(144).Children(func() {
			a.sidebarRefRowWithState(c, lucide.Boxes(), "libraries/"+tail, "", ref, 2, false, false, func() {})
		})
	}, 144, 40)
	leaf, leafOK := tt.Find(tail)
	branch, branchOK := tt.Find(ref)
	badge, badgeOK := tt.Find("2")
	if !leafOK || !branchOK || !badgeOK || leaf.W < tailWidth-1 || branch.W < 24 || leaf.X+leaf.W > branch.X || branch.X+branch.W > badge.X {
		t.Fatalf("path, ref and badge do not fit: leaf=%+v (needs %.1f), ref=%+v, badge=%+v", leaf, tailWidth, branch, badge)
	}
}

func TestCountedSubmoduleRowFitsLargeText(t *testing.T) {
	original := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(original) })
	setUIFontScale(fontScaleLarge)
	a := &app{locale: localeEnglish}
	for _, label := range []string{"libraries/AnExtremelyLongSubmoduleName", "module"} {
		const ref = "feature/a-long-branch"
		tt := ui.NewTester(func(c *ui.Context) {
			theme.Apply(c)
			ui.Column(c).Width(144).Children(func() {
				a.sidebarRefRowWithState(c, lucide.Boxes(), label, "", ref, 12345, false, false, func() {})
			})
		}, 144, 40)
		row, rowOK := tt.AccessByRole(ui.RoleButton, label)
		badge, badgeOK := tt.Find("12345")
		branch, branchOK := tt.Find(ref)
		if !rowOK || !badgeOK || !branchOK || badge.W < 20 || badge.X+badge.W > row.Bounds.X+row.Bounds.W {
			t.Fatalf("%s: badge clipped at Large scale: row=%+v badge=%+v", label, row.Bounds, badge)
		}
		if branch.X-row.Bounds.X < 46 {
			t.Fatalf("%s: branch leaves no legible name at Large scale: row=%+v branch=%+v", label, row.Bounds, branch)
		}
	}
}

func TestLocalWatchRefreshesSubmoduleCountWithUnchangedParentStatus(t *testing.T) {
	a, callbacks, child := submoduleChangesFixture(t)
	writeSubmoduleFixtureFile(t, child, "base.txt", "child edit\n")
	writeSubmoduleFixtureFile(t, child, "new.txt", "new child file\n")
	var err error
	a.status, err = a.repo.Status(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	a.openRepository(a.repo.Root, false)
	for a.loading {
		runContinuation(t, callbacks)
	}
	if a.localWatch.stamp == "" || len(a.submodules) != 1 {
		t.Fatal("fixture did not load the repository and submodule")
	}
	before, err := a.repo.StatusText(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	writeSubmoduleFixtureFile(t, child, "second.txt", "another tracked edit\n")
	after, err := a.repo.StatusText(t.Context())
	if err != nil || before != after {
		t.Fatalf("fixture must keep the parent porcelain identical: %v\n%q\n%q", err, before, after)
	}
	epoch := a.loadSeq
	a.lineSel = map[int]bool{4: true}
	a.pollLocalChanges(time.Now().Add(time.Minute))
	for a.localWatch.reading || a.loading {
		runContinuation(t, callbacks)
	}
	if a.loadSeq != epoch || !a.lineSel[4] {
		t.Fatal("changing only the submodule count refreshed parent history or cleared line selection")
	}
	row, _ := sidebarTester(a).AccessByRole(ui.RoleButton, "module")
	if !strings.Contains(row.Description, "3 files changed") {
		t.Fatalf("refreshed submodule row has stale count: %q", row.Description)
	}
	writeSubmoduleFixtureFile(t, child, "another-new.txt", "one more untracked file\n")
	a.pollLocalChanges(a.localWatch.next.Add(time.Second))
	for a.localWatch.reading || a.loading {
		runContinuation(t, callbacks)
	}
	if n := a.submodules[0].ChangedFiles; n != 4 || a.loadSeq != epoch || !a.lineSel[4] {
		t.Fatalf("new untracked child file reloaded parent or lost count: %d files, epoch %d to %d", n, epoch, a.loadSeq)
	}
	if err := os.Remove(filepath.Join(child, "another-new.txt")); err != nil {
		t.Fatal(err)
	}
	submoduleFixtureGit(t, child, "restore", "second.txt")
	a.revalidateActivatedRepository()
	for a.loading {
		runContinuation(t, callbacks)
	}
	if n := a.submodules[0].ChangedFiles; n != 2 {
		t.Fatalf("tab reactivation showed %d changed files; want 2", n)
	}
	submoduleFixtureGit(t, child, "restore", "base.txt")
	if err := os.Remove(filepath.Join(child, "new.txt")); err != nil {
		t.Fatal(err)
	}
	a.pollLocalChanges(a.localWatch.next.Add(time.Second))
	for a.localWatch.reading || a.loading {
		runContinuation(t, callbacks)
	}
	row, _ = sidebarTester(a).AccessByRole(ui.RoleButton, "module")
	if a.submodules[0].ChangedFiles != 0 || strings.Contains(row.Description, "changed file") {
		t.Fatalf("clean child retained a badge: %q", row.Description)
	}
}

func TestSubmoduleWatchClearsBadgeWhenParentSnapshotWasClean(t *testing.T) {
	a, callbacks, child := submoduleChangesFixture(t)
	stamp, err := readLocalWatchStamp(a.ctx, a.repo, nil)
	if err != nil {
		t.Fatal(err)
	}
	writeSubmoduleFixtureFile(t, child, "base.txt", "edit made after parent read\n")
	a.submodules, err = a.repo.Submodules(t.Context())
	if err != nil || len(a.submodules) != 1 || a.submodules[0].ChangedFiles != 1 {
		t.Fatalf("fixture did not observe the child edit after parent status: %+v, %v", a.submodules, err)
	}
	a.localWatch.stamp = stamp
	submoduleFixtureGit(t, child, "restore", "base.txt")
	current, err := readLocalWatchStamp(a.ctx, a.repo, nil)
	if err != nil || current != stamp {
		t.Fatalf("fixture changed the parent stamp: %v", err)
	}
	a.pollLocalChanges(time.Now())
	for a.localWatch.reading || a.loading {
		runContinuation(t, callbacks)
	}
	if a.submodules[0].ChangedFiles != 0 {
		t.Fatal("badge survived after child returned to the parent's observed state")
	}
}

func TestSubmoduleWatchChecksEntriesHiddenByNestedIgnore(t *testing.T) {
	a, callbacks, middle := submoduleChangesFixture(t)
	_, _, leaf := worktreeApp(t)
	submoduleFixtureGit(t, middle, "-c", "protocol.file.allow=always", "submodule", "add", leaf, "leaf")
	submoduleFixtureGit(t, middle, "config", "-f", ".gitmodules", "submodule.leaf.ignore", "all")
	submoduleFixtureGit(t, middle, "add", ".gitmodules")
	submoduleFixtureGit(t, middle, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-m", "Add leaf")
	submoduleFixtureGit(t, a.repo.Root, "commit", "-am", "Update middle gitlink")
	a.openRepository(a.repo.Root, false)
	for a.loading {
		runContinuation(t, callbacks)
	}
	if len(a.submodules) != 1 || a.submodules[0].ChangedFiles != 0 {
		t.Fatalf("expected clean middle checkout: %+v", a.submodules)
	}
	leafCheckout := filepath.Join(middle, "leaf")
	submoduleFixtureGit(t, leafCheckout, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-m", "Move leaf")
	status, err := a.repo.Status(t.Context())
	if err != nil || len(status.Entries) != 0 {
		t.Fatalf("parent should not have child flags for this fixture: %+v, %v", status.Entries, err)
	}
	a.pollLocalChanges(time.Now().Add(time.Minute))
	for a.localWatch.reading || a.loading {
		runContinuation(t, callbacks)
	}
	if n := a.submodules[0].ChangedFiles; n != 1 {
		t.Fatalf("nested gitlink change did not update middle count: got %d, want 1", n)
	}
}

func TestSubmoduleWatchSynchronizesRowsForOneCheckout(t *testing.T) {
	a, callbacks, child := submoduleChangesFixture(t)
	writeSubmoduleFixtureFile(t, child, "base.txt", "child edit\n")
	var err error
	a.status, err = a.repo.Status(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.openRepository(a.repo.Root, false)
	for a.loading {
		runContinuation(t, callbacks)
	}
	module := a.submodules[0]
	// Simulate independently read aliases that straddled a filesystem edit.
	module.Name, module.ChangedFiles = "alias", 2
	a.submodules = append(a.submodules, module)
	a.pollLocalChanges(a.localWatch.next.Add(time.Second))
	for a.localWatch.reading || a.loading {
		runContinuation(t, callbacks)
	}
	if n := a.submodules[1].ChangedFiles; n != 1 || a.submodules[1].Name != "alias" {
		t.Fatalf("shared checkout did not update both rows independently: %+v", a.submodules)
	}
}

func TestBatchDiscardRejectsNewlyMovedSubmodule(t *testing.T) {
	a, callbacks, child := submoduleChangesFixture(t)
	writeSubmoduleFixtureFile(t, child, "base.txt", "child edit\n")
	writeSubmoduleFixtureFile(t, a.repo.Root, "base.txt", "parent edit\n")
	var err error
	a.status, err = a.repo.Status(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.confirmDiscardAll()
	runContinuation(t, callbacks)
	if !a.confirm.open || a.confirm.action == nil {
		t.Fatal("dirty-only submodule prevented confirming discard of parent edit")
	}
	submoduleFixtureGit(t, child, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-am", "Move child HEAD")
	confirm := a.confirm.action
	a.confirm = confirmState{}
	confirm()
	runContinuation(t, callbacks)
	contents, err := os.ReadFile(filepath.Join(a.repo.Root, "base.txt"))
	if err != nil || string(contents) != "parent edit\n" {
		t.Fatalf("stale confirmation discarded parent edit: %q, %v", contents, err)
	}
}

func setupSubmoduleStatusFault(t *testing.T, child, parent string) string {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	fault := filepath.Join(bin, "fault")
	script := `#!/bin/sh
repo=""
previous=""
for arg do
    if [ "$previous" = '-C' ]; then repo="$arg"; fi
    if [ "$arg" = 'status' ]; then
        mode=$(cat "$SG_TEST_STATUS_FAULT" 2>/dev/null)
        case "$mode:$repo" in
            "child-error:$SG_TEST_CHILD"|"parent-error:$SG_TEST_PARENT") exit 1 ;;
            "child-timeout:$SG_TEST_CHILD") exec sleep 10 ;;
        esac
        break
    fi
    previous="$arg"
done
exec "$SG_TEST_GIT" "$@"
`
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SG_TEST_GIT", git)
	t.Setenv("SG_TEST_CHILD", child)
	t.Setenv("SG_TEST_PARENT", parent)
	t.Setenv("SG_TEST_STATUS_FAULT", fault)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return fault
}

func TestSubmoduleWatchRecoversFromChildStatusError(t *testing.T) {
	a, callbacks, child := submoduleChangesFixture(t)
	writeSubmoduleFixtureFile(t, child, "base.txt", "child edit\n")
	fault := setupSubmoduleStatusFault(t, child, a.repo.Root)
	writeSubmoduleFixtureFile(t, filepath.Dir(fault), filepath.Base(fault), "child-error\n")
	a.openRepository(a.repo.Root, false)
	for a.loading {
		runContinuation(t, callbacks)
	}
	if a.loadErr != "" || len(a.submodules) != 1 || a.submodules[0].Error == "" || a.submodules[0].ChangedFiles != 0 {
		t.Fatalf("child error escaped its own row: %q, %+v", a.loadErr, a.submodules)
	}
	writeSubmoduleFixtureFile(t, filepath.Dir(fault), filepath.Base(fault), "\n")
	a.pollLocalChanges(a.localWatch.next.Add(time.Second))
	for a.localWatch.reading || a.loading {
		runContinuation(t, callbacks)
	}
	if a.submodules[0].Error != "" || a.submodules[0].ChangedFiles != 1 {
		t.Fatalf("child did not recover: %+v", a.submodules)
	}
	writeSubmoduleFixtureFile(t, filepath.Dir(fault), filepath.Base(fault), "parent-error\n")
	a.openRepository(a.repo.Root, false)
	for a.loading {
		runContinuation(t, callbacks)
	}
	if a.loadErr == "" || a.canCommit() {
		t.Fatalf("failed parent status must block Commit: %q", a.loadErr)
	}
}

func TestSlowSubmoduleProbeKeepsOtherCompletedResults(t *testing.T) {
	a, _, child := submoduleChangesFixture(t)
	source := strings.TrimSpace(submoduleFixtureGit(t, a.repo.Root, "config", "-f", ".gitmodules", "--get", "submodule.module.url"))
	submoduleFixtureGit(t, a.repo.Root, "-c", "protocol.file.allow=always", "submodule", "add", source, "second-module")
	submoduleFixtureGit(t, a.repo.Root, "commit", "-m", "Add second module")
	modules, err := a.repo.Submodules(a.ctx)
	if err != nil || len(modules) != 2 {
		t.Fatalf("could not read two initialized modules: %v %+v", err, modules)
	}
	writeSubmoduleFixtureFile(t, filepath.Join(a.repo.Root, "second-module"), "base.txt", "edit in responsive child\n")
	fault := setupSubmoduleStatusFault(t, child, a.repo.Root)
	writeSubmoduleFixtureFile(t, filepath.Dir(fault), filepath.Base(fault), "child-timeout\n")
	updates := readSubmoduleUpdates(t.Context(), a.repo, modules)
	if len(updates) != 1 || updates[0].Path != "second-module" || updates[0].ChangedFiles != 1 {
		t.Fatalf("timed-out child lost a completed result or replaced its last known state: %+v", updates)
	}
}

func TestWorkspaceKeepsGitlinkDeletionAndTypeChange(t *testing.T) {
	a := &app{status: gitrepo.Status{Entries: []gitrepo.Entry{
		{Path: "deleted", Index: '.', Worktree: 'D', Submodule: "S..."},
		{Path: "replaced", Index: '.', Worktree: 'T', Submodule: "S..."},
	}}}
	if entries := a.unstagedEntries(); len(entries) != 2 {
		t.Fatalf("real gitlink changes disappeared: %+v", entries)
	}
}
