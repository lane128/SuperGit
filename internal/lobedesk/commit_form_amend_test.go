package lobedesk

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

func commitFormTester(a *app) *ui.Tester {
	a.worktreeSelected = true
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(420).Height(560).Children(func() { a.worktreeCommitPane(c) })
	}, 420, 560)
}

// Amend rewrites HEAD, but the commit list is whatever ref the reader is
// browsing. Prefilling from the first row meant that after looking at another
// branch, ticking "amend" offered that branch's newest subject and committing
// it would rewrite HEAD's message with it.
func TestAmendPrefillsHeadNotTheBrowsedBranch(t *testing.T) {
	a := &app{
		locale:      localeChinese,
		repo:        &gitrepo.Repository{Root: t.TempDir()},
		headMessage: "head subject",
		commits:     []gitrepo.Commit{{SHA: "aaaa", Subject: "other branch subject"}},
	}
	tt := commitFormTester(a)
	checkbox, ok := tt.AccessByRole(ui.RoleCheckBox, a.t("amendLast"))
	if !ok {
		t.Fatal("the amend checkbox is not on the commit form")
	}
	tt.ClickAt(checkbox.Bounds.X+checkbox.Bounds.W/2, checkbox.Bounds.Y+checkbox.Bounds.H/2)
	if !a.amend {
		t.Fatal("clicking amend did not tick it")
	}
	if a.commitText != "head subject" {
		t.Errorf("amend prefilled %q; want HEAD's subject", a.commitText)
	}
}

func TestAmendPrefillsCompleteHeadMessageAfterReload(t *testing.T) {
	a, callbacks := overviewFixture(t)
	message := "Original title\n\nFirst paragraph\n\nSecond paragraph\n\nRefs: fixture"
	if err := a.repo.AmendCommit(a.ctx, message); err != nil {
		t.Fatal(err)
	}
	a.openRepository(a.repo.Root, false)
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	for a.loading {
		select {
		case fn := <-callbacks:
			fn()
		case <-timer.C:
			t.Fatal("repository reload timed out")
		}
	}
	if a.loadErr != "" {
		t.Fatal(a.loadErr)
	}
	a.commitText = ""
	tt := commitFormTester(a)
	checkbox, ok := tt.AccessByRole(ui.RoleCheckBox, a.t("amendLast"))
	if !ok {
		t.Fatal("Amend checkbox missing after reload")
	}
	tt.ClickAt(checkbox.Bounds.X+checkbox.Bounds.W/2, checkbox.Bounds.Y+checkbox.Bounds.H/2)
	if strings.TrimSuffix(a.commitText, "\n") != message {
		t.Fatalf("Amend lost HEAD message: got %q; want %q", a.commitText, message)
	}
}

// A failed history read leaves the list empty, but HEAD is still there and can
// still be amended: the checkbox must not disappear with the list.
func TestAmendStaysAvailableWithoutALoadedHistory(t *testing.T) {
	a := &app{
		locale:      localeChinese,
		repo:        &gitrepo.Repository{Root: t.TempDir()},
		headMessage: "head subject",
		historyErr:  "history read failed",
	}
	tt := commitFormTester(a)
	if _, ok := tt.AccessByRole(ui.RoleCheckBox, a.t("amendLast")); !ok {
		t.Error("the amend checkbox needs a loaded history row to appear")
	}
}
