package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

func commitFormTester(a *app) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(420).Height(560).Children(func() { a.commitForm(c) })
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
		headSubject: "head subject",
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

// A failed history read leaves the list empty, but HEAD is still there and can
// still be amended: the checkbox must not disappear with the list.
func TestAmendStaysAvailableWithoutALoadedHistory(t *testing.T) {
	a := &app{
		locale:      localeChinese,
		repo:        &gitrepo.Repository{Root: t.TempDir()},
		headSubject: "head subject",
		historyErr:  "history read failed",
	}
	tt := commitFormTester(a)
	if _, ok := tt.AccessByRole(ui.RoleCheckBox, a.t("amendLast")); !ok {
		t.Error("the amend checkbox needs a loaded history row to appear")
	}
}
