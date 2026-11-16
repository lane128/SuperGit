package lobedesk

import (
	"context"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestCheckoutPickerPreservesPendingTargetAndSearch(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	a.openCheckoutBranchDialog(gitrepo.Commit{SHA: "aaaa", Subject: "Original"})
	a.checkoutBranch.query = "feature/"
	a.checkoutBranch.cursor = 3
	a.checkoutBranch.cursorQuery = "feature/"
	a.checkoutBranch.cursorPlaced = true
	a.checkoutBranch.reveal = false
	before := a.checkoutBranch
	a.openCheckoutBranchDialog(gitrepo.Commit{SHA: "bbbb", Subject: "Replacement"})
	if a.checkoutBranch != before {
		t.Fatal("later request replaced pending commit or search state")
	}
	if len(a.notices[toastMain]) != 1 {
		t.Fatal("rejected replacement was silent")
	}
}

func TestCheckoutPickerExitsOnRepositoryChange(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}, branches: []gitrepo.Branch{
		{Name: "feature/one", SHA: "aaaa"}, {Name: "feature/two", SHA: "aaaa"},
	}}
	a.openCheckoutBranchDialog(gitrepo.Commit{SHA: "aaaa"})
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.checkoutBranchDialog(c) }, 640, 480)
	tt.Type("feature/")
	tt.Frame()
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	before := a.checkoutBranch
	a.repo = &gitrepo.Repository{Root: "first"}
	tt.Frame()
	if a.checkoutBranch != before {
		t.Fatal("same-root refresh lost picker input or cursor")
	}
	a.repo = &gitrepo.Repository{Root: "second"}
	tt.Frame()
	if a.checkoutBranch.open || tt.HasText(a.t("checkoutBranchesTitle")) {
		t.Fatal("old picker survived repository switch")
	}
}

func TestCheckoutPickerEnterAfterRootSwitchCannotStartOperation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a := &app{ctx: ctx, runOnMain: func(func()) {}, repo: &gitrepo.Repository{Root: t.TempDir()}, branches: []gitrepo.Branch{{Name: "feature", SHA: "aaaa"}}}
	a.openCheckoutBranchDialog(gitrepo.Commit{SHA: "aaaa"})
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.checkoutBranchDialog(c) }, 640, 480)
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.busy || a.confirm.open {
		t.Fatal("old picker started branch switch in another repository")
	}
	if a.checkoutBranch.open {
		t.Fatal("stale picker remained actionable")
	}
}

func TestCheckoutPickerOnlyDrawsInMainWindow(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	a.openCheckoutBranchDialog(gitrepo.Commit{SHA: "aaaa"})
	settings := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.checkoutBranchDialogFor(c, toastSettings) }, 640, 480)
	if settings.HasText(a.t("checkoutBranchesTitle")) {
		t.Fatal("repository picker moved to Settings")
	}
	main := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.checkoutBranchDialog(c) }, 640, 480)
	if !main.HasText(a.t("checkoutBranchesTitle")) {
		t.Fatal("main window lost repository picker")
	}
}

func TestCurrentCheckoutPickerCanReplaceStaleTarget(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	a.openCheckoutBranchDialog(gitrepo.Commit{SHA: "aaaa"})
	a.repo = &gitrepo.Repository{Root: "second"}
	a.openCheckoutBranchDialog(gitrepo.Commit{SHA: "bbbb"})
	if a.checkoutBranch.sha != "bbbb" {
		t.Fatal("stale picker blocked current request")
	}
}
