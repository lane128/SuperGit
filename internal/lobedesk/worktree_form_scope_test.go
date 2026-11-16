package lobedesk

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func scopedWorktreeFormApp(t *testing.T) *app {
	t.Helper()
	return &app{repo: &gitrepo.Repository{Root: t.TempDir()}, status: gitrepo.Status{Branch: "main"}}
}

func TestWorktreeParentResultOnlyUpdatesOriginalForm(t *testing.T) {
	for _, change := range []string{"root", "new-form", "closed"} {
		t.Run(change, func(t *testing.T) {
			a := scopedWorktreeFormApp(t)
			a.openWorktreeForm()
			root, seq := a.repo.Root, a.worktreeFormSeq
			switch change {
			case "root":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			case "new-form":
				a.worktreeForm.open = false
				a.openWorktreeForm()
			case "closed":
				a.worktreeForm = worktreeFormState{}
			}
			before := a.worktreeForm
			a.applyWorktreeParent(root, seq, t.TempDir())
			if a.worktreeForm != before {
				t.Fatal("old directory result changed another form")
			}
		})
	}
	a := scopedWorktreeFormApp(t)
	a.openWorktreeForm()
	a.worktreeForm.path = filepath.Join(t.TempDir(), "typed name")
	parent := t.TempDir()
	a.applyWorktreeParent(a.repo.Root, a.worktreeFormSeq, parent)
	if a.worktreeForm.path != filepath.Join(parent, "typed name") {
		t.Fatal("current directory result lost typed name")
	}
}

func TestWorktreeFormOnlyDrawsInMainWindow(t *testing.T) {
	a := scopedWorktreeFormApp(t)
	a.openWorktreeForm()
	settings := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.worktreeFormDialogFor(c, toastSettings) }, 640, 480)
	if settings.HasText(a.t("newWorktreeTitle")) {
		t.Fatal("worktree form moved to Settings")
	}
	main := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.worktreeFormDialog(c) }, 640, 480)
	if !main.HasText(a.t("newWorktreeTitle")) {
		t.Fatal("main window lost worktree form")
	}
}

func TestWorktreeFormPreservesPendingDraft(t *testing.T) {
	a := scopedWorktreeFormApp(t)
	a.openWorktreeForm()
	a.worktreeForm.path = filepath.Join(t.TempDir(), "edited destination")
	a.worktreeForm.branch = "edited-branch"
	a.worktreeForm.start = "chosen-start"
	before := a.worktreeForm
	a.openWorktreeForm()
	if a.worktreeForm != before {
		t.Fatal("later request replaced edited worktree form")
	}
	if len(a.notices[toastMain]) != 1 {
		t.Fatal("replacement rejection was silent")
	}
}

func TestWorktreeFormExitsOnRepositoryChange(t *testing.T) {
	a := scopedWorktreeFormApp(t)
	a.openWorktreeForm()
	a.worktreeForm.branch = "typed"
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.worktreeFormDialog(c) }, 640, 480)
	before := a.worktreeForm
	a.repo = &gitrepo.Repository{Root: a.repo.Root}
	tt.Frame()
	if a.worktreeForm != before {
		t.Fatal("same-root refresh lost draft")
	}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	tt.Frame()
	if a.worktreeForm.open || tt.HasText(a.t("newWorktreeTitle")) {
		t.Fatal("old worktree form survived root change")
	}
}

func TestStaleWorktreeFormCannotSubmitToNewRepository(t *testing.T) {
	a := scopedWorktreeFormApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a.ctx, a.runOnMain = ctx, func(func()) {}
	a.openWorktreeForm()
	a.worktreeForm.branch = "topic"
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.submitWorktreeForm()
	if a.busy || a.worktreeForm.open {
		t.Fatal("stale worktree form started against new repository")
	}
}

func TestWorktreeFailureCannotReplaceNewDraft(t *testing.T) {
	a, done, _ := worktreeApp(t)
	a.status.Branch = "main"
	a.openWorktreeForm()
	// A nonexistent start forces a real Git failure in the temporary fixture.
	a.worktreeForm.path = filepath.Join(t.TempDir(), "failed destination")
	a.worktreeForm.branch = "old-topic"
	a.worktreeForm.start = "missing-start"
	a.submitWorktreeForm()
	if !a.busy {
		t.Fatal("fixture did not start creation")
	}
	a.openWorktreeForm()
	a.worktreeForm.branch = "new-topic"
	before := a.worktreeForm
	runContinuation(t, done)
	if a.worktreeForm != before {
		t.Fatal("old failed operation overwrote newer worktree draft")
	}
}

func TestWorktreeFailureWaitsForOtherDialogWithoutLosingDraft(t *testing.T) {
	a, done, _ := worktreeApp(t)
	a.status.Branch = "main"
	a.openWorktreeForm()
	a.worktreeForm.path = filepath.Join(t.TempDir(), "failed destination")
	a.worktreeForm.branch = "failed-topic"
	a.worktreeForm.start = "missing-start"
	a.submitWorktreeForm()
	a.askPrompt("Keep this input", "Value", "current draft", "Continue", nil)
	runContinuation(t, done)
	if !a.prompt.open || a.prompt.value != "current draft" || a.worktreeForm.open {
		t.Fatal("failed creation overlaid current input")
	}
	a.prompt = promptState{}
	a.openWorktreeForm()
	if !a.worktreeForm.open || a.worktreeForm.branch != "failed-topic" || a.worktreeForm.start != "missing-start" {
		t.Fatal("deferred failure recovery lost original worktree draft")
	}
}

func TestWorktreeFailureCannotRestoreIntoNewRepository(t *testing.T) {
	a, done, _ := worktreeApp(t)
	a.status.Branch = "main"
	a.openWorktreeForm()
	a.worktreeForm.path = filepath.Join(t.TempDir(), "failed destination")
	a.worktreeForm.branch = "topic"
	a.worktreeForm.start = "missing-start"
	a.submitWorktreeForm()
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	runContinuation(t, done)
	if a.worktreeForm.open {
		t.Fatal("old failure restored form in another repository")
	}
}
