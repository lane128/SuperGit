package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestWorktreeDiffHeaderNamesOriginalPathAndComparison(t *testing.T) {
	before := uiFontScaleValue()
	setUIFontScale(fontScaleLarge)
	t.Cleanup(func() { setUIFontScale(before) })
	path := "very-long-directory/文件 space/original-name.go"
	for _, staged := range []bool{false, true} {
		label := "Index → Working tree"
		if staged {
			label = "HEAD → Index"
		}
		t.Run(label, func(t *testing.T) {
			a := &app{locale: localeEnglish, page: "changes", worktreeSelected: true, selectedPath: path, selectedStaged: staged,
				status: gitrepo.Status{Entries: []gitrepo.Entry{{Path: path, Index: 'M', Worktree: 'M'}}}}
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				ui.Column(c).Fill().Children(func() { a.worktreeDiffHeader(c, theme.TokensOf(c)) })
			}, 300, 100)
			if !tt.HasText(label) {
				t.Fatalf("missing exact comparison %q", label)
			}
			text, ok := tt.Find(path)
			if !ok || text.X < 0 || text.X+text.W > 300 {
				t.Fatalf("raw original path missing or overflowing: %+v", text)
			}
			if a.selectedPath != path || a.selectedStaged != staged {
				t.Fatal("header changed original target")
			}
		})
	}
}
