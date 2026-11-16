package lobedesk

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func pendingBlameRead(t *testing.T) (*app, func()) {
	t.Helper()
	a, done, _ := worktreeApp(t)
	a.openBlameAt("base.txt", "HEAD")
	select {
	case fn := <-done:
		return a, fn
	case <-time.After(10 * time.Second):
		t.Fatal("blame did not complete")
		return nil, nil
	}
}

func TestBlameOnlyDrawsInMainWindow(t *testing.T) {
	a, finish := pendingBlameRead(t)
	finish()
	settings := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.blameDialogFor(c, toastSettings) }, 980, 640)
	if settings.HasText(a.t("blameTitle")) {
		t.Fatal("blame moved to Settings")
	}
	main := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.blameDialog(c) }, 980, 640)
	if !main.HasText(a.t("blameTitle")) {
		t.Fatal("main window lost blame")
	}
}

func TestBlamePendingRequestKeepsOriginalObject(t *testing.T) {
	a, finish := pendingBlameRead(t)
	finish()
	a.blameIgnoreWS = true
	a.openBlameAt("other.txt", "different-revision")
	if a.blamePath != "base.txt" || a.blameRev != "HEAD" || !a.blameIgnoreWS || len(a.blame) == 0 {
		t.Fatal("later request replaced current blame object or options")
	}
	if len(a.notices[toastMain]) != 1 {
		t.Fatal("replacement rejection was silent")
	}
}

func TestBlameReadCannotEnterAnotherRepository(t *testing.T) {
	a, finish := pendingBlameRead(t)
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	finish()
	if len(a.blame) != 0 {
		t.Fatal("old repository blame result entered current state")
	}
}

func TestBlameReadAfterCloseIsIgnored(t *testing.T) {
	a, finish := pendingBlameRead(t)
	a.blameOpen = false
	finish()
	if len(a.blame) != 0 {
		t.Fatal("closed blame accepted pending read")
	}
}

func TestBlameSameRootRefreshRetainsReadAndOptions(t *testing.T) {
	a, finish := pendingBlameRead(t)
	a.repo = &gitrepo.Repository{Root: a.repo.Root}
	finish()
	if len(a.blame) != 1 || a.blame[0].Text != "base" {
		t.Fatalf("same-root read lost lines: %+v", a.blame)
	}
	a.blameIgnoreWS = true
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.blameDialog(c) }, 980, 640)
	tt.Frame()
	if !a.blameOpen || !a.blameIgnoreWS {
		t.Fatal("same-root rendering lost blame state")
	}
}

func TestBlameDialogExitsOnRepositoryChange(t *testing.T) {
	a, finish := pendingBlameRead(t)
	finish()
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.blameDialog(c) }, 980, 640)
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	tt.Frame()
	if a.blameOpen || tt.HasText(a.t("blameTitle")) {
		t.Fatal("old blame dialog survived root change")
	}
}

func TestBlamePreviousFrameActionCannotReplaceReopenedObject(t *testing.T) {
	a, finish := pendingBlameRead(t)
	finish()
	action := a.blameCommitAction(a.blame[0].SHA)
	a.blameOpen = false
	a.openBlameAt("base.txt", "HEAD")
	a.page = "worktrees"
	action()
	if a.page != "worktrees" || !a.blameOpen {
		t.Fatal("old row callback acted on reopened blame object")
	}
}

func TestBlameOldRowCannotNavigateInNewRepository(t *testing.T) {
	a, finish := pendingBlameRead(t)
	finish()
	sha := a.blame[0].SHA
	a.page = "worktrees"
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.openBlameCommit(sha)
	if a.page != "worktrees" || a.selectedSHA == sha {
		t.Fatal("old blame row navigated in new repository")
	}
}
