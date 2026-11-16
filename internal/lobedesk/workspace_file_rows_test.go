package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"supergit/internal/gitrepo"
)

// The same path can sit in the unstaged and the staged section at once. The
// rows used to read as two identical buttons, and the list never said which
// one the diff showed.
func TestWorkspaceFileSectionsAreListsOfDistinctOptions(t *testing.T) {
	a := &app{locale: localeChinese, page: "history", worktreeSelected: true}
	a.status = gitrepo.Status{Entries: []gitrepo.Entry{
		{Path: "dup.go", Index: 'M', Worktree: 'M'},
		{Path: "other.go", Worktree: 'M'},
	}}
	a.selectFile("dup.go", false)
	tt := pageTester(a, func(c *ui.Context) {
		ui.Column(c).FillWidth().Gap(2).Children(func() { a.workingFileSections(c) })
	})

	lists := 0
	for _, info := range tt.AccessAll() {
		if info.Role == ui.RoleList {
			lists++
		}
	}
	if lists < 2 {
		t.Fatalf("the file pane shows %d lists; want one per section", lists)
	}

	rows := 0
	selected := 0
	for _, info := range tt.AccessAll() {
		if info.Role != ui.RoleOption || info.Name != "dup.go" {
			continue
		}
		rows++
		if info.PosInSet == 0 || info.SetSize == 0 {
			t.Errorf("the %q row reports place %d of %d", info.Description, info.PosInSet, info.SetSize)
		}
		if info.Selected {
			selected++
		}
	}
	if rows != 2 {
		t.Fatalf("the pane shows %d rows for dup.go; want the unstaged and staged ones", rows)
	}
	if selected != 1 {
		t.Errorf("%d of the duplicated rows report themselves selected; want the one the diff shows", selected)
	}

	unselected, selectedRow := ui.AccessInfo{}, ui.AccessInfo{}
	for _, info := range tt.AccessAll() {
		if info.Role != ui.RoleOption || info.Name != "dup.go" {
			continue
		}
		if info.Selected {
			selectedRow = info
		} else {
			unselected = info
		}
	}
	if unselected.Description != a.t("fileGroupStaged") {
		t.Errorf("the staged row reads %q; want it to say %q", unselected.Description, a.t("fileGroupStaged"))
	}
	if selectedRow.Description != a.t("fileGroupModified") {
		t.Errorf("the unstaged row reads %q; want it to say %q", selectedRow.Description, a.t("fileGroupModified"))
	}
}
