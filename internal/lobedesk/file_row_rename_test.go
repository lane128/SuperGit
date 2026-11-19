package lobedesk

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestRenamedFileRowShowsSourceBeforeCurrentNameAndFolder(t *testing.T) {
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
						name, ok := tt.Find(tc.name)
						if tc.path == tc.name {
							// Find sees the row label first for a root-level file.
							// Both that label and the separately drawn filename must exist.
							count := 0
							for _, text := range tt.Texts() {
								if text == tc.name {
									count++
								}
							}
							if count != 2 {
								t.Fatal("root filename was not drawn beside its row label")
							}
						}
						if !ok || name.W <= 0 || name.X < 0 || name.X+name.W > float32(width) {
							t.Fatal("current name is not visible inside row")
						}
						tt.Frame()
						stable, _ := tt.Find(tc.name)
						if !nearlyEqual(name.X, stable.X) || !nearlyEqual(name.W, stable.W) {
							t.Fatal("settled file-name geometry changed between frames")
						}
						if row.Bounds.W < fileRowFolderMin {
							_, hasSource := tt.Find(tc.original)
							_, hasArrow := tt.Find("→")
							if hasSource || hasArrow {
								t.Fatal("narrow row displaced primary name with source")
							}
							if tc.dir != "" {
								if _, hasFolder := tt.Find(tc.dir); hasFolder {
									t.Fatal("narrow row kept destination folder")
								}
							}
							return
						}
						source, hasSource := tt.Find(tc.original)
						arrow, hasArrow := tt.Find("→")
						if !hasSource || !hasArrow || source.W <= 0 || arrow.W <= 0 || source.X+source.W > arrow.X || arrow.X+arrow.W > row.Bounds.X+row.Bounds.W || tc.path != tc.name && arrow.X+arrow.W > name.X {
							t.Fatalf("rename direction unclear: source=%+v arrow=%+v name=%+v", source, arrow, name)
						}
						if tc.dir != "" {
							folder, hasFolder := tt.Find(tc.dir)
							if !hasFolder || folder.W <= 0 || name.X+name.W > folder.X || folder.X+folder.W > float32(width) {
								t.Fatalf("missing or overlapping destination folder: %+v", folder)
							}
						}
						for _, decorative := range []string{tc.original, "→", tc.name, tc.dir} {
							if decorative == "" || decorative == tc.path {
								continue
							}
							if _, ok := tt.Access(decorative); ok {
								t.Fatalf("decorative text %q duplicates row accessibility", decorative)
							}
						}
					})
				}
			}
		}
	}
}
