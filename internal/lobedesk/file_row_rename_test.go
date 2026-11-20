package lobedesk

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestRenamedFileRowShowsDestinationPath(t *testing.T) {
	before := uiFontScaleValue()
	setUIFontScale(fontScaleLarge)
	t.Cleanup(func() { setUIFontScale(before) })
	longDir := strings.Repeat("long-directory/", 4) + "destination"
	cases := []struct{ label, path, original, name, dir string }{
		{"move", "new-folder/same.go", "old-folder/same.go", "same.go", "new-folder"},
		{"root", "same.go", "nested/old.go", "same.go", ""},
		{"long", longDir + "/target.go", strings.Repeat("source-directory/", 6) + "old.go", "target.go", longDir},
	}
	for _, mode := range []string{"light", "dark"} {
		for _, scope := range []string{"commit", "workspace"} {
			for _, width := range []int{500, 360, 260, 259, 220} {
				for _, tc := range cases {
					t.Run(fmt.Sprintf("%s/%s/%d/%s", mode, scope, width, tc.label), func(t *testing.T) {
						a := &app{locale: localeEnglish, themeMode: mode, commitFiles: []gitrepo.ChangedFile{{Path: tc.path, OriginalPath: tc.original, Status: "R100"}}}
						tt := ui.NewTester(func(c *ui.Context) {
							a.applyTheme(c)
							if c.Theme().Dark != (mode == "dark") {
								t.Fatal("incorrect actual theme")
							}
							ui.Column(c).Fill().Background(theme.TokensOf(c).BgContainer).Children(func() {
								if scope == "commit" {
									a.commitFileList(c)
								} else {
									a.fileRow(c, gitrepo.Entry{Path: tc.path, OriginalPath: tc.original, Index: 'R', Worktree: ' '}, true)
								}
							})
						}, width, 100)
						tt.Frame()
						tt.Frame()
						row, ok := tt.AccessByRole(ui.RoleOption, tc.path)
						if !ok {
							t.Fatal("rename lost complete current-path accessible label")
						}
						if row.Bounds.W <= 0 || row.Bounds.X < 0 || row.Bounds.X+row.Bounds.W > float32(width)+0.5 {
							t.Fatalf("path is outside the row: %+v", row.Bounds)
						}
						found := false
						for _, text := range tt.Texts() {
							if text == tc.path {
								found = true
							}
							if text == tc.original || text == "→" {
								t.Fatalf("extra visible rename decoration %q", text)
							}
						}
						if !found {
							t.Fatalf("missing destination path %s", tc.path)
						}
					})
				}
			}
		}
	}
}
