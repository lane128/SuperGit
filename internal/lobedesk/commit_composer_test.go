package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestCommitComposerLayoutSpacingCompactButtonAndEmptyMessage(t *testing.T) {
	a := &app{
		locale:         localeEnglish,
		repo:           &gitrepo.Repository{Root: "fixture"},
		headMessage:    "previous commit",
		commitIdentity: readyCommitIdentity("fixture"),
		status: gitrepo.Status{
			Head:    "0123456789abcdef0123456789abcdef01234567",
			Entries: []gitrepo.Entry{{Path: "main.go", Index: 'M'}},
		},
	}

	// Empty commit message is allowed when staged changes exist.
	if !a.canCommit() {
		t.Fatalf("empty commit message should be allowed with staged changes: %q", a.commitDisabledReason())
	}

	// Narrow layout: author and Amend checkbox stack with >= 8px vertical gap,
	// message field is taller (4 rows), button stays compact and separate from staged pill.
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(420).Height(560).Children(func() { a.worktreeCommitPane(c) })
	}, 420, 560)
	for range 2 {
		tt.Frame()
	}

	msgField, ok := tt.AccessByRole(ui.RoleTextField, a.t("commitMessage"))
	if !ok {
		t.Fatal("missing commit message field")
	}
	if msgField.Bounds.H < 64 {
		t.Fatalf("commit message field height = %.1f; want taller multi-row input (>= 64)", msgField.Bounds.H)
	}

	authorBounds, ok := tt.Find("Author: Fixture Author <author@example.invalid>")
	if !ok {
		t.Fatal("missing author text")
	}
	amendBox, ok := tt.AccessByRole(ui.RoleCheckBox, a.t("amendLast"))
	if !ok {
		t.Fatal("missing Amend last commit checkbox")
	}
	verticalGap := amendBox.Bounds.Y - (authorBounds.Y + authorBounds.H)
	if verticalGap < 7.5 {
		t.Fatalf("stacked vertical gap between author and amend = %.1f; want >= 8", verticalGap)
	}

	if !tt.HasText("1 staged") {
		t.Fatal("missing independent staged status pill")
	}
	commitBtn, ok := tt.AccessByRole(ui.RoleButton, a.t("commit"))
	if !ok || commitBtn.Disabled {
		t.Fatalf("commit button should be enabled for empty message with staged changes: %+v", commitBtn)
	}
	if commitBtn.Bounds.W > 140 {
		t.Fatalf("commit button stretched too wide: %.1f (want compact <= 140)", commitBtn.Bounds.W)
	}

	// Wide layout: author and Amend checkbox sit side by side with generous spacing,
	// and commit button remains compact.
	wide := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(680).Height(560).Children(func() { a.worktreeCommitPane(c) })
	}, 680, 560)
	for range 3 {
		wide.Frame()
	}
	wideAuthor, ok := wide.Find("Author: Fixture Author <author@example.invalid>")
	if !ok {
		t.Fatal("missing wide author text")
	}
	wideAmend, ok := wide.AccessByRole(ui.RoleCheckBox, a.t("amendLast"))
	if !ok {
		t.Fatal("missing wide amend checkbox")
	}
	if wideAmend.Bounds.X <= wideAuthor.X {
		t.Fatalf("wide composer should place amend beside author: author=%+v amend=%+v", wideAuthor, wideAmend.Bounds)
	}
	wideBtn, ok := wide.AccessByRole(ui.RoleButton, a.t("commit"))
	if !ok || wideBtn.Bounds.W > 140 {
		t.Fatalf("wide commit button should remain compact: %+v", wideBtn)
	}

	// Amend mode switches the independent pill to Amend HEAD and button to Amend,
	// and stays committable even when HEAD had an empty message and draft is empty.
	a.amend = true
	a.headMessage = ""
	a.status.Entries = nil
	wide.Frame()
	if !a.canCommit() {
		t.Fatalf("amend with empty message should be allowed: %q", a.commitDisabledReason())
	}
	if !wide.HasText(a.t("amendHead")) {
		t.Fatal("missing Amend HEAD status pill")
	}
	amendBtn, ok := wide.AccessByRole(ui.RoleButton, a.t("amend"))
	if !ok || amendBtn.Disabled || amendBtn.Bounds.W > 150 {
		t.Fatalf("amend button invalid: %+v", amendBtn)
	}
}
