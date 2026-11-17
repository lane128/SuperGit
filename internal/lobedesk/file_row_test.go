package lobedesk

import (
	"fmt"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// A file row is one line: the name, and the folder after it when there is room.
// The folder is dropped below the threshold rather than shortening the name,
// because the name is what the row is for; at the two window widths the app
// opens at, the file pane is narrower than the threshold, so the rows read as
// names only and the diff header keeps the full path.
func TestFileRowDropsTheFolderWhenItIsNarrow(t *testing.T) {
	if fileRowShowsFolder(fileRowFolderMin - 1) {
		t.Errorf("a row %v wide showed its folder; want it dropped below %v", fileRowFolderMin-1, fileRowFolderMin)
	}
	if !fileRowShowsFolder(fileRowFolderMin) {
		t.Errorf("a row %v wide dropped its folder; want it shown from %v", fileRowFolderMin, fileRowFolderMin)
	}
	// The pane at both supported window sizes sits under the threshold, so the
	// rule has to leave the name alone there rather than splitting the row.
	for _, room := range []float32{180, 215, 240} {
		if fileRowShowsFolder(room) {
			t.Errorf("a %v wide file pane kept the folder; the name would lose its room", room)
		}
	}
}

// The working tree's files and a commit's files sit in the same 300-point
// pane, so they are drawn by the same code. They had grown two densities --
// a fixed 30-point row against one that followed the font preset -- which
// meant the two lists looked different at every preset and the working tree's
// got tighter than the other's at the large one.
func TestBothFileListsDrawOneKindOfRow(t *testing.T) {
	rows := func(a *app) *ui.Tester {
		return ui.NewTester(func(c *ui.Context) {
			theme.Apply(c)
			ui.Column(c).Width(300).Height(120).Children(func() { a.fileRow(c, gitrepo.Entry{Path: "dir/a.go", Worktree: 'M'}, false) })
		}, 300, 120)
	}
	commits := func(a *app) *ui.Tester {
		return ui.NewTester(func(c *ui.Context) {
			theme.Apply(c)
			ui.Column(c).Width(300).Height(120).Children(func() { a.commitFileList(c) })
		}, 300, 120)
	}
	for _, scale := range []float32{fontScaleSmall, fontScaleStandard, fontScaleLarge} {
		t.Run(fmt.Sprintf("scale %v", scale), func(t *testing.T) {
			setUIFontScale(scale)
			t.Cleanup(func() { setUIFontScale(fontScaleStandard) })

			work := rows(&app{locale: localeChinese, page: "history"})
			commit := commits(&app{locale: localeChinese, page: "history",
				commitFiles: []gitrepo.ChangedFile{{Path: "dir/a.go", Status: "M"}}})
			// The folder is drawn from the row's laid-out width, so it appears
			// on the frame after the first, as it does in the window.
			work.Frame()
			commit.Frame()
			wrow, ok := work.AccessByRole(ui.RoleOption, "dir/a.go")
			if !ok {
				t.Fatal("the working tree's file row is not laid out")
			}
			crow, ok := commit.AccessByRole(ui.RoleOption, "dir/a.go")
			if !ok {
				t.Fatal("the commit's file row is not laid out")
			}
			if want := scaledFont(29); !nearlyEqual(wrow.Bounds.H, want) || !nearlyEqual(crow.Bounds.H, want) {
				t.Errorf("rows are %v and %v high; want both %v at this preset", wrow.Bounds.H, crow.Bounds.H, want)
			}
			reference := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				ui.Text(c, "a.go").FontSize(scaledFont(12))
			}, 300, 120)
			nameReference, _ := reference.Find("a.go")
			for _, text := range []struct{ name, label string }{{"a.go", "name"}, {"dir", "folder"}} {
				w, ok := work.Find(text.name)
				if !ok {
					t.Fatalf("the working tree's %s is not drawn", text.label)
				}
				c, ok := commit.Find(text.name)
				if !ok {
					t.Fatalf("the commit's %s is not drawn", text.label)
				}
				if text.label == "name" && !nearlyEqual(w.H, nameReference.H) {
					t.Errorf("file name height %v differs from 12pt reference %v", w.H, nameReference.H)
				}
				if !nearlyEqual(w.H, c.H) {
					t.Errorf("the %s is %v high in the working tree and %v in the commit; want one size", text.label, w.H, c.H)
				}
			}
		})
	}
}

// nearlyEqual compares laid-out sizes, which reach the same point by two
// rounding paths and can differ in the last bits.
func nearlyEqual(a, b float32) bool { return math.Abs(float64(a-b)) < 0.01 }
