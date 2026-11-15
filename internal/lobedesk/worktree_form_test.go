package lobedesk

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"supergit/internal/gitrepo"

	"github.com/egoist/mygo/ui"
)

// worktreeApp builds an app over a real repository with one commit, so a
// creation runs git rather than a stub. The returned channel receives the
// continuation performDoneAs would post to the UI thread; the tests run it
// themselves, which both waits for git and keeps the restored form on the
// test's goroutine.
func worktreeApp(t *testing.T) (*app, <-chan func(), string) {
	t.Helper()
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %q: %v\n%s", args, err, out)
		}
	}
	run("init", "-b", "main")
	run("config", "user.name", "Test Developer")
	run("config", "user.email", "test@example.invalid")
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "--", "base.txt")
	run("commit", "-m", "initial")
	repo, err := gitrepo.Open(context.Background(), dir)
	if err != nil {
		t.Fatalf("open the fixture repository: %v", err)
	}
	done := make(chan func(), 1)
	a := &app{locale: localeChinese, ctx: context.Background(), repo: repo, page: "history"}
	a.runOnMain = func(fn func()) { done <- fn }
	return a, done, repo.Root
}

// runContinuation waits for the operation to finish and runs the completion
// performDoneAs queued, as the event loop would.
func runContinuation(t *testing.T, done <-chan func()) {
	t.Helper()
	select {
	case fn := <-done:
		fn()
	case <-time.After(10 * time.Second):
		t.Fatal("the worktree creation never finished")
	}
}

// Enter in an incomplete form does nothing: the branch is the one field the
// reader has to type, and the empty string is not a branch name to ask git to
// create.
func TestEnterNeedsACompleteWorktreeForm(t *testing.T) {
	a, done, _ := worktreeApp(t)
	a.worktreeForm = worktreeFormState{
		open: true, path: filepath.Join(t.TempDir(), "wt"), branch: "   ", start: "main",
	}
	tt := dialogTester(a, a.worktreeFormDialog)
	tt.Key(0, ui.KeyEnter)
	if !a.worktreeForm.open {
		t.Fatal("Enter in a form without a branch closed it")
	}
	if a.busy {
		t.Fatal("an incomplete form started a creation")
	}
	select {
	case <-done:
		t.Fatal("an incomplete form started a creation")
	default:
	}
}

// Enter in a complete form creates the worktree for real: git is asked for a
// new branch checked out in the destination directory.
func TestEnterOnACompleteWorktreeFormCreatesTheWorktree(t *testing.T) {
	a, done, root := worktreeApp(t)
	dest := filepath.Join(t.TempDir(), "feature")
	a.worktreeForm = worktreeFormState{
		open: true, path: dest, branch: "feature", start: "main",
	}
	tt := dialogTester(a, a.worktreeFormDialog)
	tt.Key(0, ui.KeyEnter)
	if a.worktreeForm.open {
		t.Fatal("Enter in a complete form left it open")
	}
	if !a.busy || a.busyKey != string(busyCreateWorktree) {
		t.Fatalf("after Enter busy=%v key=%q; want a running creation", a.busy, a.busyKey)
	}
	runContinuation(t, done)
	if a.busy {
		t.Error("the finished creation left the app busy")
	}
	if a.worktreeForm.open {
		t.Error("a successful creation reopened the form")
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("git did not create %q: %v", dest, err)
	}
	branch, err := exec.Command("git", "-C", root, "branch", "--list", "--format=%(refname:short)", "feature").Output()
	if err != nil {
		t.Fatalf("list branches: %v", err)
	}
	if strings.TrimSpace(string(branch)) != "feature" {
		t.Errorf("the new branch reads %q; want feature", strings.TrimSpace(string(branch)))
	}
}

// git's rejection brings the form back with everything typed: a taken
// destination is the common failure, and retyping the path, branch and start
// to fix one of them was the reason the form is restored at all.
func TestAFailedWorktreeCreationReopensTheFormWithItsValues(t *testing.T) {
	a, done, _ := worktreeApp(t)
	dest := filepath.Join(t.TempDir(), "taken")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	a.worktreeForm = worktreeFormState{
		open: true, path: dest, branch: "topic", start: "main",
	}
	tt := dialogTester(a, a.worktreeFormDialog)
	tt.Key(0, ui.KeyEnter)
	runContinuation(t, done)
	if !a.worktreeForm.open {
		t.Fatal("git's rejection closed the form and lost the input")
	}
	if got := a.worktreeForm; got.path != dest || got.branch != "topic" || got.start != "main" {
		t.Errorf("the form came back as %+v; want the typed values", got)
	}
	if a.busy {
		t.Error("the failed creation left the app busy")
	}
}
