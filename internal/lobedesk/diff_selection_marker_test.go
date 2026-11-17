package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestDiffSelectionHasGutterMarkWithoutMovingCode(t *testing.T) {
	for _, overview := range []bool{false, true} {
		t.Run(map[bool]string{false: "file", true: "overview"}[overview], func(t *testing.T) {
			lines := []gitrepo.DiffLine{{Kind: "del", Old: 23, Text: "-old"}, {Kind: "add", New: 24, Text: "+new"}, {Kind: "context", Old: 25, New: 25, Text: " context"}}
			a := &app{locale: localeEnglish, page: "changes", worktreeSelected: true, repo: &gitrepo.Repository{Root: t.TempDir()}, selectedPath: "file.go",
				status: gitrepo.Status{Entries: []gitrepo.Entry{{Path: "file.go", Worktree: 'M'}}},
				diff:   gitrepo.Diff{CanStageHunks: true, Lines: lines}, lineSel: map[int]bool{0: true, 1: true}, lineCursor: 1}
			target := overviewPatchTarget{Root: a.repo.Root, File: detailFileKey{Path: "file.go"}, Diff: a.diff}
			if overview {
				a.overviewLineTarget = &target
			}
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				ui.Column(c).Children(func() {
					for index, line := range lines {
						if overview {
							a.overviewPatchLine(c, target, index, 560)
						} else {
							a.diffLine(c, line, 560, index)
						}
					}
				})
			}, 560, 100)
			for index, line := range lines {
				handle, ok := tt.AccessByRole(ui.RoleButton, a.lineGutterLabel(line))
				if !ok {
					t.Fatal("line number selection handle disappeared")
				}
				want := index < 2
				if handle.Checked != want {
					t.Errorf("row %d checked semantics=%v want %v", index, handle.Checked, want)
				}
				// The existing right padding of the 92-DIP gutter must carry
				// actual ink, so selection remains visible without the blue fill.
				image := tt.Image()
				background := image.RGBAAt(int(handle.Bounds.X+90), int(handle.Bounds.Y+2))
				marked := false
				for y := int(handle.Bounds.Y + 5); y < int(handle.Bounds.Y+handle.Bounds.H-3); y++ {
					for x := int(handle.Bounds.X + 85); x < int(handle.Bounds.X+91); x++ {
						if image.RGBAAt(x, y) != background {
							marked = true
						}
					}
				}
				if marked != want {
					t.Errorf("row %d has visible gutter mark=%v want %v", index, marked, want)
				}
				if want && image.RGBAAt(int(handle.Bounds.X+89), int(handle.Bounds.Y+handle.Bounds.H/2-1)) == background {
					t.Errorf("row %d selection mark lacks its rising diagonal", index)
				}
				code, ok := tt.Find(line.Text)
				if !ok || code.X < handle.Bounds.X+handle.Bounds.W {
					t.Fatalf("row %d marker displaced or overlapped code", index)
				}
			}
		})
	}
}
