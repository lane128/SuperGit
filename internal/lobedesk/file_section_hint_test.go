package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"supergit/internal/gitrepo"
)

// A folded section keeps its count, and the count includes the rows that are
// put away. The header now says both to the pointer and to a screen reader, so
// the number does not read as "what is visible".
func TestFoldedSectionHeaderExplainsItsCount(t *testing.T) {
	a := &app{
		locale:           localeChinese,
		page:             "history",
		worktreeSelected: true,
		status: gitrepo.Status{Entries: []gitrepo.Entry{
			{Path: "a.go", Worktree: 'M'},
		}},
		filesCollapsed: map[string]bool{filesSectionUnstaged: true},
	}
	tt := pageTester(a, func(c *ui.Context) {
		ui.Column(c).FillWidth().Gap(2).Children(func() { a.workingFileSections(c) })
	})
	header, ok := tt.Access(a.t("unstaged"))
	if !ok {
		t.Fatal("the folded section header is not in the accessibility tree")
	}
	if header.Description != a.t("fileSectionHint") {
		t.Errorf("the folded header reads %q; want the fold/count hint", header.Description)
	}
	if header.Expanded {
		t.Error("the folded header reports itself expanded")
	}
}
