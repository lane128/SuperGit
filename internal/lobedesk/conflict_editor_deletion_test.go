package lobedesk

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestConflictEditorDeletionRequiresExplicitFileChoice(t *testing.T) {
	a, _, root := worktreeApp(t)
	callbacks := make(chan func(), 128)
	a.runOnMain = func(fn func()) { callbacks <- fn }
	splitCommitGit(t, root, "checkout", "-b", "incoming")
	splitCommitGit(t, root, "rm", "--", "base.txt")
	splitCommitGit(t, root, "commit", "-m", "Delete incoming file")
	splitCommitGit(t, root, "checkout", "main")
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("current modification\n"), 0644); err != nil {
		t.Fatal(err)
	}
	splitCommitGit(t, root, "commit", "-am", "Modify current file")
	if out, err := exec.Command("git", "-C", root, "merge", "--no-edit", "incoming").CombinedOutput(); err == nil {
		t.Fatalf("expected modify/delete conflict: %s", out)
	}
	var err error
	a.status, err = a.repo.Status(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.operation, err = a.repo.Operation(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.selectedPath, a.worktreeSelected, a.locale = "base.txt", true, localeEnglish
	a.diff, err = a.repo.Diff(a.ctx, "base.txt", false)
	if err != nil {
		t.Fatal(err)
	}
	head, index := splitCommitGit(t, root, "rev-parse", "HEAD"), splitCommitGit(t, root, "ls-files", "--stage")
	a.openConflictEditor()
	runContinuation(t, callbacks)
	s := a.currentConflictEditor()
	if !s.Loaded {
		t.Fatalf("deletion comparison failed: %s", s.Error)
	}
	comparison := pageTester(a, a.diffBody)
	conflictEditorClick(t, comparison, a.t("conflictTheirs"))
	if _, ok := comparison.Find("File absent in this version"); !ok {
		t.Fatal("deletion source is indistinguishable from an empty file")
	}
	s.Text = ""
	a.saveConflictEditor(s)
	runContinuation(t, callbacks)
	if s.Error == "" || splitCommitGit(t, root, "ls-files", "--stage") != index {
		t.Fatal("empty editor text silently chose deletion")
	}
	for a.loading || a.diffLoading || a.commitFilesLoading || a.commitDiffLoading {
		runContinuation(t, callbacks)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Children(func() {
			a.diffBody(c)
			target := ui.ButtonBase(c).Label("conflict file menu").Height(30).FillWidth()
			a.diffHeaderMenu(c, target)
		})
	}, 900, 650)
	conflictEditorClick(t, tt, "Show patch")
	menu, ok := tt.AccessByRole(ui.RoleButton, "conflict file menu")
	if !ok {
		t.Fatal("missing file menu")
	}
	tt.RightClickAt(menu.Bounds.X+20, menu.Bounds.Y+15)
	action, ok := tt.AccessByRole(ui.RoleMenuItem, a.conflictSideText("useTheirs"))
	if !ok {
		t.Fatal("missing explicit deletion side")
	}
	tt.ClickAt(action.Bounds.X+action.Bounds.W/2, action.Bounds.Y+action.Bounds.H/2)
	for a.busy {
		runContinuation(t, callbacks)
	}
	if _, err := os.Stat(filepath.Join(root, "base.txt")); !os.IsNotExist(err) {
		t.Fatalf("explicit deletion did not remove file: %v", err)
	}
	if got := splitCommitGit(t, root, "ls-files", "--stage", "--", "base.txt"); got != "" {
		t.Fatalf("deletion Index=%q", got)
	}
	if splitCommitGit(t, root, "rev-parse", "HEAD") != head {
		t.Fatal("file choice completed merge")
	}
}
