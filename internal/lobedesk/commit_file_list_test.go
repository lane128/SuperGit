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
func TestCommitFileHistoryMenuCanReplaceListDuringRender(t *testing.T) {
	a := &app{ctx: canceledContext(), runOnMain: func(func()) {}, repo: &gitrepo.Repository{Root: t.TempDir()}, locale: localeEnglish, page: "history", selectedSHA: "selected-commit", commitFilesLoaded: true, commitFiles: []gitrepo.ChangedFile{{Path: "first.txt", Status: "A"}, {Path: "second.txt", Status: "A"}}}
	t.Cleanup(func() { a.resetFileHistory() })
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(500).Height(250).Children(func() { a.commitFileList(c) })
	}, 500, 250)
	first, ok := tt.AccessByRole(ui.RoleOption, "first.txt")
	if !ok {
		t.Fatal("missing first file")
	}
	tt.RightClickAt(first.Bounds.X+first.Bounds.W/2, first.Bounds.Y+first.Bounds.H/2)
	item, ok := tt.AccessByRole(ui.RoleMenuItem, a.t("fileHistory"))
	if !ok {
		t.Fatal("missing file history action")
	}
	tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
	tt.Frame()
	if a.fileFilter != "first.txt" || len(a.commitFiles) != 0 || a.selectedSHA != "" {
		t.Fatal("file history action did not replace selected commit context")
	}
}

func TestNarrowHistoryCommitFilesKeepShortBasenamesReadable(t *testing.T) {
	oldScale := uiFontScaleValue()
	setUIFontScale(fontScaleLarge)
	t.Cleanup(func() { setUIFontScale(oldScale) })
	const name = "fixture.txt"
	const path = "src/" + name
	const sha = "selected-commit"
	a := &app{page: "history", themeMode: "light", locale: localeEnglish,
		repo: &gitrepo.Repository{Root: t.TempDir()}, selectedSHA: sha, commitFilesLoaded: true,
		commits:     []gitrepo.Commit{{SHA: sha, Subject: "Update fixture"}},
		commitFiles: []gitrepo.ChangedFile{{Path: path, Status: "M"}}}
	var nameWidth float32
	view := ui.NewTester(func(c *ui.Context) {
		a.root(c)
		nameWidth, _ = c.MeasureText(0, ui.Span{Text: name, Size: scaledFont(12)})
	}, 980, 640)
	for _, width := range []int{980, 1320, 980} {
		view.SetSize(width, 640)
		view.Frame()
		if file, ok := view.AccessByRole(ui.RoleOption, path); !ok || file.Offscreen {
			t.Fatalf("width %d: full path missing from the Files list", width)
		}
		bounds, ok := view.Find(name)
		if !ok || bounds.W+0.5 < nameWidth {
			t.Errorf("width %d: %s has %.1f points, needs %.1f to avoid truncation", width, name, bounds.W, nameWidth)
		}
	}
}

func TestCommitFilesReadAsAListOfOptions(t *testing.T) {
	a := &app{
		locale: localeChinese,
		commitFiles: []gitrepo.ChangedFile{
			{Path: "dir/a.go", Status: "M"},
			{Path: "dir/b.go", Status: "A"},
		},
		commitPath: "dir/a.go",
	}
	a.detailTabs().Open(detailFileKey{Path: "dir/a.go"})
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
