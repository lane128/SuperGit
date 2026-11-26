package lobedesk

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

// Removing the titlebar integration or letting controls overlap breaks reachability.
func TestWorkspaceQuickActionsTitlebar(t *testing.T) {
	for _, width := range []float32{900, 1320} {
		for _, language := range []locale{localeChinese, localeEnglish} {
			a := createBranchApp(t)
			a.locale = language
			a.status.Entries = []gitrepo.Entry{{Path: "edit.txt", Index: ' ', Worktree: 'M'}}
			tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); ui.Column(c).Fill().Children(func() { a.titleBar(c) }) }, int(width), 640)
			assertCommandControlBounds(t, tt, a, width)
			branch, _ := tt.AccessByRole(ui.RoleButton, a.t("branches"))
			stash, ok := tt.AccessByRole(ui.RoleButton, a.t("workspaceStash"))
			if !ok || stash.Disabled {
				t.Fatal("quick Stash missing or disabled")
			}
			clean, ok := tt.AccessByRole(ui.RoleButton, a.t("workspaceCleanAction"))
			if !ok || clean.Disabled {
				t.Fatal("quick clean missing or disabled")
			}
			sync, _ := tt.AccessByRole(ui.RoleMenuButton, a.t("syncMenu"))
			if stash.Bounds.X < branch.Bounds.X+branch.Bounds.W || clean.Bounds.X < stash.Bounds.X+stash.Bounds.W || clean.Bounds.X+clean.Bounds.W > sync.Bounds.X {
				t.Fatal("titlebar quick controls overlap branch or sync")
			}
			clickRepositoryTabsControl(t, tt, ui.RoleMenuButton, a.t("saveStashAction"))
			for _, label := range []string{a.t("saveStash"), a.t("saveStashUntracked")} {
				if _, ok := tt.AccessByRole(ui.RoleMenuItem, label); !ok {
					t.Fatalf("lost named Stash flow %q", label)
				}
			}
		}
	}
}

func TestWorkspaceCleanEmptyDirectoryOnly(t *testing.T) {
	a, callbacks, _ := workspaceSelectionFixture(t)
	// Save the fixture changes so only a Git-invisible empty directory remains.
	if err := a.repo.SaveWorkspaceStash(context.Background()); err != nil {
		t.Fatal(err)
	}
	empty := filepath.Join(a.repo.Root, "empty")
	if err := os.Mkdir(empty, 0755); err != nil {
		t.Fatal(err)
	}
	var err error
	a.status, err = a.repo.Status(context.Background())
	if err != nil || len(a.status.Entries) != 0 {
		t.Fatalf("fixture not clean: %+v %v", a.status, err)
	}
	tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.titleQuickActions(c) }, 900, 640)
	clean, ok := tt.AccessByRole(ui.RoleButton, a.t("workspaceCleanAction"))
	if !ok || clean.Disabled {
		t.Fatal("empty-directory cleanup unreachable")
	}
	stash, ok := tt.AccessByRole(ui.RoleButton, a.t("workspaceStash"))
	if !ok || !stash.Disabled {
		t.Fatal("Stash should remain disabled without Git changes")
	}
	a.openWorkspaceClean()
	select {
	case fn := <-callbacks:
		fn()
	case <-time.After(10 * time.Second):
		t.Fatal("empty-directory preview did not open")
	}
	if a.workspaceClean == nil || a.workspaceClean.err != "" || a.workspaceClean.preview.CanExecute {
		t.Fatalf("default preview: %+v", a.workspaceClean)
	}
	a.prepareWorkspaceClean(true)
	select {
	case fn := <-callbacks:
		fn()
	case <-time.After(10 * time.Second):
		t.Fatal("delete preview timed out")
	}
	if a.workspaceClean.err != "" || !a.workspaceClean.preview.CanExecute {
		t.Fatalf("delete preview: %+v", a.workspaceClean)
	}
	a.confirmWorkspaceClean()
	waitSelectionOperation(t, a, callbacks)
	if _, err := os.Stat(empty); !os.IsNotExist(err) {
		t.Fatalf("empty directory survived: %v", err)
	}
}

func TestWorkspaceCleanOwnsModalAndRejectsLateResults(t *testing.T) {
	a := createBranchApp(t)
	state := &workspaceCleanState{open: true, repo: a.repo, root: a.repo.Root, loading: true}
	a.workspaceClean = state
	if !a.hasPendingDialog(toastMain) || a.hasPendingDialog(toastSettings) {
		t.Fatal("clean dialog does not own main modal layer")
	}
	a.closeWorkspaceClean()
	a.finishWorkspaceCleanPreparation(state, gitrepo.WorkspaceCleanPreview{Head: "old", CanExecute: true}, nil)
	if a.workspaceClean != nil {
		t.Fatal("closed dialog resurrected by late preparation")
	}
	a.workspaceClean = state
	state.open = true
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.finishWorkspaceCleanPreparation(state, gitrepo.WorkspaceCleanPreview{CanExecute: true}, nil)
	if a.workspaceCleanOpen() {
		t.Fatal("old repository result attached to new checkout")
	}
}

func TestWorkspaceCleanDialogDefaultsAndCancel(t *testing.T) {
	a := createBranchApp(t)
	a.workspaceClean = &workspaceCleanState{open: true, repo: a.repo, root: a.repo.Root, preview: gitrepo.WorkspaceCleanPreview{Head: "1234567", NewPaths: []string{"new.txt"}}}
	tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.workspaceCleanDialog(c) }, 900, 640)
	tt.Frame()
	confirm, ok := tt.AccessByRole(ui.RoleButton, a.t("workspaceCleanConfirm"))
	if !ok || !confirm.Disabled {
		t.Fatal("untracked-only default must not execute")
	}
	if !tt.Focused(a.t("cancel")) {
		t.Fatal("destructive dialog did not initially focus Cancel")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.workspaceCleanOpen() {
		t.Fatal("Escape did not close clean dialog")
	}
}

func TestWorkspaceCleanPreparationCancelledOnClose(t *testing.T) {
	a := createBranchApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	a.workspaceClean = &workspaceCleanState{open: true, repo: a.repo, root: a.repo.Root, cancel: cancel}
	a.closeWorkspaceClean()
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("preparation kept running after close")
	}
}

func TestWorkspaceCleanIgnoresPreviousOptionResult(t *testing.T) {
	a := createBranchApp(t)
	old := &workspaceCleanState{open: true, repo: a.repo, root: a.repo.Root}
	current := &workspaceCleanState{open: true, repo: a.repo, root: a.repo.Root, deleteNew: true, loading: true}
	a.workspaceClean = current
	a.finishWorkspaceCleanPreparation(old, gitrepo.WorkspaceCleanPreview{Head: "stale", CanExecute: true}, nil)
	if !current.loading || current.preview.CanExecute {
		t.Fatal("previous option authorized current preview")
	}
	a.finishWorkspaceCleanPreparation(current, gitrepo.WorkspaceCleanPreview{Head: "current", DeleteNew: true, CanExecute: true}, nil)
	if current.loading || current.preview.Head != "current" {
		t.Fatal("current result lost")
	}
}

func TestWorkspaceCleanReopenResetsDeletionAndExecutesDefault(t *testing.T) {
	a, callbacks, _ := workspaceSelectionFixture(t)
	// An addition staged before cleaning must remain on disk after default cleanup.
	newPath := filepath.Join(a.repo.Root, "staged-new.txt")
	if err := os.WriteFile(newPath, []byte("keep this addition\n"), 0600); err != nil {
		t.Fatal(err)
	}
	selectionGit(t, a.repo.Root, "add", "--", "staged-new.txt")
	var err error
	a.status, err = a.repo.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	a.prepareWorkspaceClean(true)
	select {
	case fn := <-callbacks:
		fn()
	case <-time.After(10 * time.Second):
		t.Fatal("prepare timed out")
	}
	if a.workspaceClean == nil || a.workspaceClean.err != "" {
		t.Fatalf("preview: %+v", a.workspaceClean)
	}
	a.closeWorkspaceClean()
	a.openWorkspaceClean()
	select {
	case fn := <-callbacks:
		fn()
	case <-time.After(10 * time.Second):
		t.Fatal("reopen timed out")
	}
	if a.workspaceClean == nil || a.workspaceClean.deleteNew || a.workspaceClean.preview.DeleteNew {
		t.Fatal("deletion persisted across openings")
	}
	a.confirmWorkspaceClean()
	if !a.busy || a.workspaceClean != nil {
		t.Fatal("confirmed action did not start exactly once")
	}
	waitSelectionOperation(t, a, callbacks)
	data, err := os.ReadFile(newPath)
	if err != nil || string(data) != "keep this addition\n" {
		t.Fatal("default UI cleanup deleted staged addition")
	}
	if got := selectionGit(t, a.repo.Root, "diff", "--cached"); got != "" {
		t.Fatal("default UI cleanup retained staged changes")
	}
}

func TestWorkspaceQuickStashSavesWholeWorkspaceWithoutPrompt(t *testing.T) {
	a, callbacks, _ := workspaceSelectionFixture(t)
	a.selectedPath = "b.go"
	a.quickWorkspaceStash()
	if !a.busy || a.prompt.open {
		t.Fatal("quick Stash did not start directly")
	}
	waitSelectionOperation(t, a, callbacks)
	status, err := a.repo.Status(context.Background())
	if err != nil || len(status.Entries) != 0 {
		t.Fatalf("quick Stash did not save all staged/unstaged/untracked changes: %+v %v", status, err)
	}
	stashes, err := a.repo.Stashes(context.Background())
	if err != nil || len(stashes) != 1 || !strings.Contains(stashes[0].Message, "SuperGit workspace") {
		t.Fatalf("missing automatic Stash: %+v %v", stashes, err)
	}
}

func TestWorkspaceCleanBlocksKeyboardBehindDialog(t *testing.T) {
	a := createBranchApp(t)
	a.page = "history"
	a.worktreeSelected = true
	a.selectedPath = "example.txt"
	a.workspaceClean = &workspaceCleanState{open: true, repo: a.repo, root: a.repo.Root}
	if a.canCommitShortcut() || a.canOpenSelectedFile() || a.detailNavigationAllowed() {
		t.Fatal("keyboard action escaped clean dialog")
	}
	a.closeWorkspaceClean()
	if !a.canCommitShortcut() {
		t.Fatal("closed dialog still blocks composer")
	}
}

func TestWorkspaceCleanCancellationKeepsSafetyGuidance(t *testing.T) {
	a, callbacks, _ := workspaceSelectionFixture(t)
	preview, err := a.repo.PrepareWorkspaceClean(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	a.workspaceClean = &workspaceCleanState{open: true, repo: a.repo, root: a.repo.Root, preview: preview}
	a.confirmWorkspaceClean()
	a.cancelOperation()
	waitSelectionOperation(t, a, callbacks)
	found := false
	for _, n := range a.notices[toastMain] {
		if n.title == a.t("workspaceCleanFailed") {
			found = true
		}
	}
	if !found {
		t.Fatal("cancellation hid cleanup failure and partial-change guidance")
	}
}

func TestWorkspaceQuickStashCancellationKeepsSafetyGuidance(t *testing.T) {
	a, callbacks, _ := workspaceSelectionFixture(t)
	a.quickWorkspaceStash()
	a.cancelOperation()
	waitSelectionOperation(t, a, callbacks)
	for _, n := range a.notices[toastMain] {
		if n.title == a.t("saveStashFailed") {
			return
		}
	}
	t.Fatal("cancellation hid Stash failure guidance")
}

func TestWorkspaceCleanDisclosesIndexOnlyLoss(t *testing.T) {
	a := createBranchApp(t)
	a.locale = localeEnglish
	a.workspaceClean = &workspaceCleanState{open: true, repo: a.repo, root: a.repo.Root, preview: gitrepo.WorkspaceCleanPreview{Head: "1234567", NewPaths: []string{"missing-new.txt"}, IndexOnlyPaths: []string{"missing-new.txt"}, CanExecute: true}}
	tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.workspaceCleanDialog(c) }, 900, 640)
	if !tt.HasText(a.t("workspaceCleanIndexOnly")) || tt.HasText(a.t("workspaceCleanKeepNew")+" (1)") {
		t.Fatal("index-only data loss incorrectly presented as preserving a file")
	}
}

func TestWorkspaceQuickActionErrorsLocalized(t *testing.T) {
	a := &app{locale: localeEnglish}
	err := &gitrepo.WorkspaceActionError{Phase: "restore", English: "Could not restore files; cleanup may be partially complete.", Partial: true, Cause: errors.New("git restore: permission denied")}
	got := a.operationErrorDescription(err)
	if !strings.Contains(got, "partially complete") || !strings.Contains(got, "permission denied") {
		t.Fatalf("lost localized guidance or Git detail: %q", got)
	}
}
