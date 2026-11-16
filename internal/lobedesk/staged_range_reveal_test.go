package lobedesk

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

// A partially staged deletion is a short patch even when its unstaged source
// has thousands of rows. Extending the range must reveal the last selected row.
func TestStagedDeletionRangeRevealsItsKeyboardEndpoint(t *testing.T) {
	a := &app{page: "history", locale: localeEnglish, worktreeSelected: true}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.status.Entries = []gitrepo.Entry{{Path: "long.txt", Index: 'M', Worktree: 'M'}}
	a.diff = gitrepo.Diff{Text: "staged patch", CanStageHunks: true}
	a.diff.Lines = append(a.diff.Lines, gitrepo.DiffLine{Kind: "hunk", Text: "@@ -1,43 +1,3 @@"})
	for row := 1; row <= 40; row++ {
		a.diff.Lines = append(a.diff.Lines, gitrepo.DiffLine{Kind: "del", Old: row, Text: fmt.Sprintf("-after long patch line %04d", row-1)})
	}
	for row := 41; row <= 43; row++ {
		a.diff.Lines = append(a.diff.Lines, gitrepo.DiffLine{Kind: "context", Old: row, New: row - 40, Text: " unchanged"})
	}
	file := detailFileKey{Path: "long.txt", Staged: true}
	a.diffCache.put(a.repo.Root, a.loadSeq, file, a.diff)
	a.openDetailFile(file)
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.registerShortcuts(c)
		ui.Column(c).Fill().Children(func() { a.diffBody(c) })
	}, 310, 484)
	tt.Frame()
	handle, ok := tt.AccessByRole(ui.RoleButton, "1 Deleted line")
	if !ok {
		t.Fatal("first deletion handle missing")
	}
	tt.ClickAt(handle.Bounds.X+20, handle.Bounds.Y+10)
	for range 39 {
		tt.Key(ui.Shift, ui.KeyDown)
	}
	for range 4 {
		tt.Frame()
	}
	if a.lineCursor != 40 || len(a.lineSel) != 40 {
		t.Fatalf("range has cursor=%d, selected=%d; want 40/40", a.lineCursor, len(a.lineSel))
	}
	endpoint, ok := tt.AccessByRole(ui.RoleButton, "40 Deleted line")
	if !ok {
		t.Fatal("selected endpoint not built")
	}
	viewport := a.diffViewportRect
	if endpoint.Bounds.Y < viewport.Y || endpoint.Bounds.Y+endpoint.Bounds.H > viewport.Y+viewport.H {
		t.Fatalf("selected endpoint bounds=%+v outside viewport=%+v, scroll=%.1f", endpoint.Bounds, viewport, a.diffScroll.Y)
	}
}
