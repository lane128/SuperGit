package lobedesk

import (
	"fmt"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

func TestFileRowsShowRelativePathsInNarrowColumns(t *testing.T) {
	for _, width := range []int{160, 240, 360} {
		view := ui.NewTester(func(c *ui.Context) {
			theme.Apply(c)
			ui.Row(c).FillWidth().Children(func() { fileRowPath(c, "api/config.go", false) })
		}, width, 100)
		if _, ok := view.Find("api/config.go"); !ok {
			t.Fatalf("path missing at width %d", width)
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
			if !work.HasText("dir/a.go") || !commit.HasText("dir/a.go") {
				t.Fatal("both lists must render the repository-relative path")
			}
		})
	}
}

// nearlyEqual compares laid-out sizes, which reach the same point by two
// rounding paths and can differ in the last bits.
func nearlyEqual(a, b float32) bool { return math.Abs(float64(a-b)) < 0.01 }
