package lobedesk

import (
	"context"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestResetFormOnlyDrawsInMainWindow(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	a.openResetDialog(gitrepo.Commit{SHA: strings.Repeat("a", 40), Subject: "First target"})
	settings := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.resetDialogFor(c, toastSettings) }, 640, 420)
	if settings.HasText(a.t("resetTitle")) {
		t.Fatal("reset form moved to Settings")
	}
	main := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.resetDialog(c) }, 640, 420)
	if !main.HasText(a.t("resetTitle")) {
		t.Fatal("main window lost reset form")
	}
}

func TestHardResetSecondConfirmationKeepsOriginalRoot(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	a.openResetDialog(gitrepo.Commit{SHA: strings.Repeat("a", 40), Subject: "First target"})
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.resetDialog(c) }, 640, 420)
	button, ok := tt.Find(a.t("resetHard"))
	if !ok {
		t.Fatal("hard reset choice unavailable")
	}
	tt.ClickAt(button.X+button.W/2, button.Y+button.H/2)
	if !a.confirm.open || !a.confirm.danger {
		t.Fatal("hard reset did not ask for explicit second confirmation")
	}
	a.repo = &gitrepo.Repository{Root: "second"}
	a.confirm.action()
	if a.busy {
		t.Fatal("old hard-reset confirmation started against a new repository")
	}
}

func TestResetTargetCannotBeReplacedWhileAwaitingChoice(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	first := gitrepo.Commit{SHA: strings.Repeat("a", 40), Subject: "First target"}
	a.openResetDialog(first)
	a.openResetDialog(gitrepo.Commit{SHA: strings.Repeat("b", 40), Subject: "Second target"})
	if a.reset.sha != first.SHA || a.reset.subject != first.Subject {
		t.Fatal("later reset request replaced target awaiting mode choice")
	}
	if len(a.notices[toastMain]) != 1 {
		t.Fatal("rejected replacement was silent")
	}
}

func TestResetDialogExitsOnRepositoryChange(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	a.openResetDialog(gitrepo.Commit{SHA: strings.Repeat("a", 40), Subject: "First target"})
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.resetDialog(c) }, 640, 420)
	if !a.reset.open {
		t.Fatal("fixture did not open reset")
	}
	a.repo = &gitrepo.Repository{Root: "first"}
	tt.Frame()
	if !a.reset.open {
		t.Fatal("same-root refresh removed selected reset target")
	}
	a.repo = &gitrepo.Repository{Root: "second"}
	tt.Frame()
	if a.reset.open {
		t.Fatal("old reset form survived repository switch")
	}
}

func TestResetCannotStartFromStaleRepositoryForm(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a := &app{ctx: ctx, repo: &gitrepo.Repository{Root: t.TempDir()}, runOnMain: func(func()) {}}
	a.openResetDialog(gitrepo.Commit{SHA: strings.Repeat("a", 40), Subject: "First target"})
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	if a.performReset("soft", a.reset.sha) {
		t.Fatal("stale reset form started a write against another repository")
	}
	if a.busy {
		t.Fatal("stale reset started an operation")
	}
}

func TestCurrentResetCanReplaceStaleForm(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	a.openResetDialog(gitrepo.Commit{SHA: strings.Repeat("a", 40)})
	a.repo = &gitrepo.Repository{Root: "second"}
	current := strings.Repeat("b", 40)
	a.openResetDialog(gitrepo.Commit{SHA: current})
	if a.reset.sha != current {
		t.Fatal("stale form blocked a current reset request")
	}
}
