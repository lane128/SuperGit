package lobedesk

import (
	"context"
	"testing"

	"github.com/egoist/mygo/ui"

	"supergit/internal/gitrepo"
)

// A folder that is not a repository fails on open; the empty page used to
// call that a diff failure, while the toast named it correctly.
func TestOpeningANonRepositorySaysOpenFailed(t *testing.T) {
	a := &app{locale: localeChinese, loadErr: "not a git repository"}
	tt := pageTester(a, a.emptyPage)
	if _, ok := tt.Find(a.t("openFailed")); !ok {
		t.Fatal("the page does not say the repository could not be opened")
	}
	if _, ok := tt.Find(a.tf("diffFailed", a.loadErr)); ok {
		t.Error("the page calls an open failure a diff failure")
	}
}

// The branch and tag reads can fail on their own; the page must offer the
// way to read them again instead of leaving recovery to the title bar.
func TestBranchReadFailuresOfferRetry(t *testing.T) {
	a := &app{
		locale: localeChinese, page: "branches",
		branchesErr: "fatal: cannot read refs",
		tagsErr:     "fatal: cannot read tags",
	}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.ctx = context.Background()
	tt := pageTester(a, a.branchesPage)
	for _, name := range []string{a.t("retryLocalBranches"), a.t("retryTags")} {
		if _, ok := tt.AccessByRole(ui.RoleButton, name); !ok {
			t.Fatalf("the failed reads do not offer %q", name)
		}
	}
	// Two buttons doing the same reread must be told apart; a second "Retry"
	// gives no clue which group it belongs to.
	if _, ok := tt.AccessByRole(ui.RoleButton, a.t("retry")); ok {
		t.Error("the page still shows a plain Retry beside the named ones")
	}
	retry, _ := tt.AccessByRole(ui.RoleButton, a.t("retryLocalBranches"))
	before := a.loadSeq
	tt.ClickAt(retry.Bounds.X+retry.Bounds.W/2, retry.Bounds.Y+retry.Bounds.H/2)
	if a.loadSeq == before {
		t.Error("Retry did not start a new read")
	}
}

// The history list reads its own commits; a failure there gets the same
// retry as the other lists.
func TestHistoryReadFailureOffersRetry(t *testing.T) {
	a := &app{
		locale: localeChinese, page: "history",
		historyErr: "fatal: bad revision",
	}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.ctx = context.Background()
	tt := pageTester(a, a.historyPage)
	retry, ok := tt.AccessByRole(ui.RoleButton, a.t("retry"))
	if !ok {
		t.Fatal("the failed history read offers no retry")
	}
	before := a.loadSeq
	tt.ClickAt(retry.Bounds.X+retry.Bounds.W/2, retry.Bounds.Y+retry.Bounds.H/2)
	if a.loadSeq == before {
		t.Error("Retry did not start a new read")
	}
}
