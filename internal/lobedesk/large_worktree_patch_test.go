package lobedesk

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestLargeWorkingFilePatchClipsRowsAndRetainsPressedGutter(t *testing.T) {
	a := &app{locale: localeEnglish, selectedPath: "large.go", linePressIndex: -1}
	a.diff = gitrepo.Diff{Text: "large working patch"}
	for i := range 1000 {
		a.diff.Lines = append(a.diff.Lines, gitrepo.DiffLine{Kind: "add", New: i + 1, Text: fmt.Sprintf("+line %d", i)})
	}
	tt := pageTester(a, a.diffBody)
	for range 3 {
		tt.Frame()
	}
	count := 0
	for _, node := range tt.AccessAll() {
		if node.Role == ui.RoleInsertion {
			count++
		}
	}
	if count == 0 || count > 100 {
		t.Fatalf("built %d working patch rows for a small viewport; want 1..100", count)
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "901 Added line"); ok {
		t.Fatal("distant gutter built before scrolling")
	}
	a.lineCursor, a.revealLine = 999, true
	for range 5 {
		tt.Frame()
	}
	last, ok := tt.AccessByRole(ui.RoleButton, "1000 Added line")
	if !ok || last.Bounds.Y < a.diffViewportRect.Y || last.Bounds.Y+last.Bounds.H > a.diffViewportRect.Y+a.diffViewportRect.H+1 {
		t.Fatalf("pending keyboard target not materialized in viewport: %+v", last)
	}
	if a.diffScroll.MaxY < 900*diffRowH {
		t.Fatal("working patch lost full scroll extent")
	}
	a.diffScroll.Y = 0
	for range 4 {
		tt.Frame()
	}
	first, ok := tt.AccessByRole(ui.RoleButton, "1 Added line")
	if !ok {
		t.Fatal("first gutter missing after scrolling back")
	}
	x := first.Bounds.X + first.Bounds.W/2
	tt.Press(x, first.Bounds.Y+first.Bounds.H/2)
	second, ok := tt.AccessByRole(ui.RoleButton, "2 Added line")
	if !ok {
		t.Fatal("second gutter missing")
	}
	tt.Move(x, second.Bounds.Y+second.Bounds.H/2)
	tt.Move(x, a.diffViewportRect.Y+a.diffViewportRect.H-2)
	for range 160 {
		tt.Frame()
	}
	if a.diffScroll.Y < 4*a.diffViewportRect.H || !a.lineDragHeld || a.linePressIndex != 0 || a.lineCursor < 100 {
		t.Fatalf("drag did not survive scrolling past overscan: scroll=%v viewport=%v held=%v press=%v cursor=%v", a.diffScroll.Y, a.diffViewportRect, a.lineDragHeld, a.linePressIndex, a.lineCursor)
	}
	tt.Release(x, a.diffViewportRect.Y+a.diffViewportRect.H-2)
	for range 3 {
		tt.Frame()
	}
	if a.lineDragHeld || a.linePressIndex >= 0 {
		t.Fatal("released working patch drag remains armed")
	}
}
