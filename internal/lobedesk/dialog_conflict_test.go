package lobedesk

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func dialogConflictApp(t *testing.T) *app {
	t.Helper()
	return &app{ctx: t.Context(), repo: &gitrepo.Repository{Root: t.TempDir()}, status: gitrepo.Status{Branch: "main"}, runOnMain: func(func()) {}}
}

var dialogOpeners = map[string]func(*app){
	"confirm":  func(a *app) { a.askConfirm("Confirm original", "Object", "Continue", false, nil) },
	"prompt":   func(a *app) { a.askPrompt("Original input", "Value", "typed draft", "Continue", nil) },
	"reset":    func(a *app) { a.openResetDialog(gitrepo.Commit{SHA: "aaaa", Subject: "Original reset"}) },
	"checkout": func(a *app) { a.openCheckoutBranchDialog(gitrepo.Commit{SHA: "aaaa", Subject: "Original checkout"}) },
	"worktree": func(a *app) { a.openWorktreeForm() },
	"blame":    func(a *app) { a.openBlameAt("base.txt", "aaaa") },
	"details": func(a *app) {
		a.openNoticeDetails(toastMain, &toastNotice{root: a.repo.Root, description: "New error"})
	},
}

func setPendingDialog(a *app, kind string) {
	switch kind {
	case "confirm":
		a.confirm = confirmState{owner: toastMain, root: a.repo.Root, open: true, title: "Pending confirm"}
	case "prompt":
		a.prompt = promptState{root: a.repo.Root, open: true, value: "pending draft"}
	case "reset":
		a.reset = resetState{root: a.repo.Root, head: a.status.Head, branch: a.status.Branch, open: true, sha: "pending"}
	case "checkout":
		a.checkoutBranch = checkoutBranchState{root: a.repo.Root, open: true, sha: "pending", query: "edited query"}
	case "worktree":
		a.worktreeForm = worktreeFormState{root: a.repo.Root, open: true, branch: "pending draft"}
	case "blame":
		a.blameRoot, a.blameOpen, a.blamePath = a.repo.Root, true, "pending.txt"
	case "details":
		a.noticeDetails[toastMain] = noticeDetailsState{open: true, notice: &toastNotice{root: a.repo.Root, description: "Pending error"}}
	}
}

func dialogIsOpen(a *app, kind string) bool {
	switch kind {
	case "confirm":
		return a.confirm.open
	case "prompt":
		return a.prompt.open
	case "reset":
		return a.reset.open
	case "checkout":
		return a.checkoutBranch.open
	case "worktree":
		return a.worktreeForm.open
	case "blame":
		return a.blameOpen
	case "details":
		return a.noticeDetails[toastMain].open
	}
	return false
}

func TestDifferentDialogRequestsPreserveFirstModal(t *testing.T) {
	for _, first := range []string{"confirm", "prompt", "reset", "checkout", "worktree", "blame", "details"} {
		for second, open := range dialogOpeners {
			if first == second {
				continue
			}
			t.Run(first+"-then-"+second, func(t *testing.T) {
				a := dialogConflictApp(t)
				setPendingDialog(a, first)
				open(a)
				if !dialogIsOpen(a, first) || dialogIsOpen(a, second) {
					t.Fatal("later dialog overlapped or replaced the first modal")
				}
				if len(a.notices[toastMain]) != 1 {
					t.Fatal("later request rejection was silent")
				}
			})
		}
	}
}

func TestRequestedErrorDetailsWaitForCurrentInput(t *testing.T) {
	a := dialogConflictApp(t)
	setPendingDialog(a, "prompt")
	notice := &toastNotice{root: a.repo.Root, title: "Failure", description: "Original complete output"}
	a.openNoticeDetails(toastMain, notice)
	if a.noticeDetails[toastMain].open {
		t.Fatal("details covered current input")
	}
	a.prompt = promptState{}
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.noticeDetailsDialog(c, toastMain) }, 980, 640)
	if !tt.HasText(notice.description) {
		t.Fatal("requested details were lost while input was open")
	}
}

func TestMainInputDoesNotBlockSettingsConfirmation(t *testing.T) {
	a := dialogConflictApp(t)
	a.settingsWin.Store(&mygo.Window{})
	setPendingDialog(a, "prompt")
	a.askConfirmInWindow(toastSettings, "", "Settings reset", "Settings only", "Reset", false, nil)
	if !a.confirm.open || a.confirm.owner != toastSettings || a.prompt.value != "pending draft" {
		t.Fatal("independent settings confirmation was blocked or changed main draft")
	}
}

func TestSettingsDetailsDoNotBlockMainInput(t *testing.T) {
	a := dialogConflictApp(t)
	a.settingsWin.Store(&mygo.Window{})
	a.noticeDetails[toastSettings] = noticeDetailsState{open: true, notice: &toastNotice{description: "Settings error"}}
	dialogOpeners["prompt"](a)
	if !a.prompt.open || !a.noticeDetails[toastSettings].open {
		t.Fatal("independent settings details blocked main input")
	}
}

func TestNewRootDialogIsNotBlockedByStaleOtherType(t *testing.T) {
	a := dialogConflictApp(t)
	setPendingDialog(a, "prompt")
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	dialogOpeners["reset"](a)
	if !a.reset.open || a.prompt.open {
		t.Fatal("stale input blocked current reset dialog")
	}
}
