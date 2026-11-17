package lobedesk

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestDiffRowsUseDesignHeightAndContainLargeText(t *testing.T) {
	before := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(before) })
	for _, scale := range []float32{fontScaleStandard, fontScaleLarge} {
		setUIFontScale(scale)
		for _, mode := range []string{"file", "overview", "historical"} {
			t.Run(fmt.Sprintf("%s/%v", mode, scale), func(t *testing.T) {
				lines := []gitrepo.DiffLine{{Kind: "del", Old: 1, Text: "-旧的文本"}, {Kind: "add", New: 1, Text: "+new text"}}
				a := &app{locale: localeEnglish, page: "changes", worktreeSelected: true,
					repo: &gitrepo.Repository{Root: t.TempDir()}, selectedPath: "file.go",
					status: gitrepo.Status{Entries: []gitrepo.Entry{{Path: "file.go", Worktree: 'M'}}},
					diff:   gitrepo.Diff{CanStageHunks: true, Lines: lines}}
				target := overviewPatchTarget{Root: a.repo.Root, File: detailFileKey{Path: "file.go"}, Diff: a.diff}
				tt := ui.NewTester(func(c *ui.Context) {
					theme.Apply(c)
					ui.Column(c).Children(func() {
						for index, line := range lines {
							switch mode {
							case "file":
								a.diffLine(c, line, 560, index)
							case "overview":
								a.overviewPatchLine(c, target, index, 560)
							case "historical":
								a.commitDiffLine(c, line, "file.go", 560, index)
							}
						}
					})
				}, 560, 100)
				for index, line := range lines {
					gutter, ok := tt.AccessByRole(ui.RoleButton, a.lineGutterLabel(line))
					if !ok || !nearlyEqual(gutter.Bounds.H, 23) || !nearlyEqual(gutter.Bounds.Y, float32(index)*23) {
						t.Fatalf("patch row does not use 23pt geometry: %+v", gutter.Bounds)
					}
					code, ok := tt.Find(line.Text)
					if !ok || code.Y < gutter.Bounds.Y || code.Y+code.H > gutter.Bounds.Y+gutter.Bounds.H {
						t.Fatalf("patch text escapes row: code=%+v row=%+v", code, gutter.Bounds)
					}
				}
			})
		}
	}
}
