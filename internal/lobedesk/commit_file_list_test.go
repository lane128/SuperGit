package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// The commit's changed files read as a list of options: a screen reader hears
// each path once, with its place in the list and which one the diff shows.
// The drawn name and folder used to be read again beside the row's label.
func TestCommitFilesReadAsAListOfOptions(t *testing.T) {
	a := &app{
		locale: localeChinese,
		commitFiles: []gitrepo.ChangedFile{
			{Path: "dir/a.go", Status: "M"},
			{Path: "dir/b.go", Status: "A"},
		},
		commitPath: "dir/a.go",
	}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(500).Height(200).Children(func() {
			a.commitFileList(c)
		})
	}, 500, 200)

	if _, ok := tt.AccessByRole(ui.RoleList, ""); !ok {
		t.Fatal("the changed files are not inside a list")
	}
	first, ok := tt.AccessByRole(ui.RoleOption, "dir/a.go")
	if !ok {
		t.Fatal("the changed file is not an option")
	}
	if first.PosInSet != 1 || first.SetSize != 2 {
		t.Errorf("the first file reads place %d of %d; want 1 of 2", first.PosInSet, first.SetSize)
	}
	if !first.Selected {
		t.Errorf("the shown file reads %+v; want it selected", first)
	}
	second, ok := tt.AccessByRole(ui.RoleOption, "dir/b.go")
	if !ok {
		t.Fatal("the second changed file is not an option")
	}
	if second.PosInSet != 2 || second.SetSize != 2 {
		t.Errorf("the second file reads place %d of %d; want 2 of 2", second.PosInSet, second.SetSize)
	}
	if second.Selected {
		t.Error("a file the diff does not show reads selected")
	}

	// The row's label carries the whole path; the separate name and folder
	// drawn inside it must not be read a second time.
	if info, ok := tt.Access("a.go"); ok {
		t.Errorf("the drawn file name is read again: %+v", info)
	}
	if info, ok := tt.Access("dir"); ok {
		t.Errorf("the drawn folder is read again: %+v", info)
	}
}
