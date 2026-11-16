package lobedesk

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

// Large commits retain their complete extent, but must not build one native
// control subtree per offscreen path or section on every frame.
func TestLargeCommitFilesBuildOnlyNearbyRows(t *testing.T) {
	a := &app{page: "history", selectedSHA: "aaaa"}
	for i := range 1000 {
		a.commitFiles = append(a.commitFiles, gitrepo.ChangedFile{Path: fmt.Sprintf("files/%04d.txt", i), Status: "A"})
	}
	tt := pageTester(a, func(c *ui.Context) {
		ui.Row(c).Height(600).FillWidth().Children(func() { a.historyFilesPane(c) })
	})
	tt.Frame()
	count := 0
	for _, node := range tt.AccessAll() {
		if node.Role == ui.RoleOption {
			count++
		}
	}
	if count == 0 || count > 100 {
		t.Fatalf("built %d file rows for a small viewport; want 1..100", count)
	}
	if _, ok := tt.AccessByRole(ui.RoleOption, "files/0900.txt"); ok {
		t.Fatal("distant file row built before it was visible")
	}
	a.commitFilesScroll.Y = 900 * (fileRowHeight() + fileRowGap)
	for range 4 {
		tt.Frame()
	}
	if _, ok := tt.AccessByRole(ui.RoleOption, "files/0900.txt"); !ok {
		t.Fatal("scrolling did not materialize a distant file row")
	}
	if a.commitFilesScroll.MaxY < 900*fileRowHeight() {
		t.Fatal("virtual file list lost its full scroll extent")
	}
	a.commitFilesScroll.Y = 0
	for range 4 {
		tt.Frame()
	}
	first, ok := tt.AccessByRole(ui.RoleOption, "files/0000.txt")
	if !ok {
		t.Fatal("first row missing after returning to the top")
	}
	tt.ClickAt(first.Bounds.X+first.Bounds.W/2, first.Bounds.Y+first.Bounds.H/2)
	tt.Key(0, ui.KeyEnd)
	for range 4 {
		tt.Frame()
	}
	last, ok := tt.AccessByRole(ui.RoleOption, "files/0999.txt")
	if !ok || !last.Focused || last.PosInSet != 1000 || last.SetSize != 1000 {
		t.Fatalf("End did not reveal/focus the original last file position: %+v; pending=%+v scroll=%+v", last, a.detailFileFocus, a.commitFilesScroll)
	}
}

func TestLargeCommitPatchBuildsOnlyNearbySections(t *testing.T) {
	a := &app{page: "history", selectedSHA: "aaaa", commits: []gitrepo.Commit{{SHA: "aaaa", Subject: "many files"}}}
	var files []gitrepo.FileDiff
	for i := range 1000 {
		files = append(files, gitrepo.FileDiff{Path: fmt.Sprintf("files/%04d.txt", i), Diff: gitrepo.Diff{Lines: []gitrepo.DiffLine{{Kind: "add", Text: fmt.Sprintf("+section-%04d", i)}}}})
	}
	a.commitDiff, a.commitSections = buildCommitDiff(files)
	tt := pageTester(a, a.historyDiffPane)
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleButton, "files/0900.txt"); ok {
		t.Fatal("distant patch section header built before it was visible")
	}
	a.commitInfoScroll.Y = 900 * (diffFileHeaderH + diffRowH)
	for range 4 {
		tt.Frame()
	}
	found := false
	for i := 895; i < 905; i++ {
		if _, ok := tt.AccessByRole(ui.RoleButton, fmt.Sprintf("files/%04d.txt", i)); ok {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("scrolling did not materialize distant section headers")
	}
	if a.commitInfoScroll.MaxY < 900*(diffFileHeaderH+diffRowH) {
		t.Fatal("virtual sections lost full patch scroll extent")
	}
	a.commitInfoScroll.Y = 0
	for range 3 {
		tt.Frame()
	}
	index := a.commitSections[900].Rows[0]
	a.commitSel, a.commitCursor, a.commitReveal = map[int]bool{index: true}, index, true
	for range 5 {
		tt.Frame()
	}
	if a.commitInfoScroll.Y < 890*(diffFileHeaderH+diffRowH) {
		t.Fatal("keyboard patch target did not reveal its distant section")
	}
}
