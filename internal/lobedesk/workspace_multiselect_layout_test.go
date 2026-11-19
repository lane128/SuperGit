package lobedesk

import (
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestWorkspaceMultiselectGroupReportsAccessibleCount(t *testing.T) {
	for _, locale := range []locale{localeEnglish, localeChinese} {
		for _, mode := range []string{"light", "dark"} {
			for _, width := range []int{220, 360} {
				t.Run(string(locale)+"/"+mode+"/"+strconv.Itoa(width), func(t *testing.T) {
					a := &app{page: "history", worktreeSelected: true, locale: locale, themeMode: mode, repo: &gitrepo.Repository{Root: "/owned-layout-fixture"}}
					a.status.Entries = []gitrepo.Entry{{Path: "b.go", Index: ' ', Worktree: 'M'}, {Path: "c.go", Index: ' ', Worktree: 'M'}}
					a.workspaceSelection = workspaceFileSelection{Root: a.repo.Root, Group: filesSectionUnstaged, Files: map[fileRef]bool{{"b.go", false}: true, {"c.go", false}: true}, Anchor: fileRef{"b.go", false}}
					title := a.t("fileGroupModified")
					tt := ui.NewTester(func(c *ui.Context) {
						a.applyTheme(c)
						if c.Theme().Dark != (mode == "dark") {
							t.Fatal("wrong applied theme")
						}
						ui.Column(c).Width(float32(width)).Height(300).Children(func() { a.fileSection(c, filesSectionUnstaged, title, a.unstagedEntries(), false) })
					}, width, 300)
					tt.Frame()
					tt.Frame()
					header, ok := tt.AccessByRole(ui.RoleButton, title)
					if !ok || !strings.Contains(header.Description, a.tf("filesSelectedCount", 2)) || !strings.Contains(header.Description, a.tf("fileGroupCount", 2)) {
						t.Fatalf("header lacks count description: %#v", header)
					}
					list, ok := tt.AccessByRole(ui.RoleList, title)
					if !ok || !list.Multiselectable {
						t.Fatalf("group not exposed as multiselectable: %#v", list)
					}
					count, ok := tt.Find(a.tf("filesSelectedCount", 2))
					if !ok || count.W <= 0 || count.X < 0 || count.X+count.W > float32(width)+1 {
						t.Fatalf("selected count outside narrow group: %#v", count)
					}
				})
			}
		}
	}
}
