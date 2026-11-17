package lobedesk

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestFileRowSeparatesStagingStateFromVersionAction(t *testing.T) {
	for _, test := range []struct {
		name                string
		index, worktree     byte
		staged              bool
		state, code, action string
	}{
		{"modified", '.', 'M', false, "Unstaged", "M", "Stage file"},
		{"untracked", '?', '?', false, "Unstaged", "?", "Stage file"},
		{"staged modification", 'M', '.', true, "Staged", "M", "Unstage file"},
		{"staged addition", 'A', '.', true, "Staged", "A", "Unstage file"},
		{"partial working version", 'M', 'M', false, "Partially staged", "M", "Stage file"},
		{"partial index version", 'M', 'M', true, "Partially staged", "M", "Unstage file"},
		{"partial rename index", 'R', 'M', true, "Partially staged", "R", "Unstage file"},
		{"partial deletion working", 'M', 'D', false, "Partially staged", "D", "Stage file"},
	} {
		t.Run(test.name, func(t *testing.T) {
			entry := gitrepo.Entry{Path: "dir/文件 space.go", Index: test.index, Worktree: test.worktree}
			a := &app{locale: localeEnglish, page: "changes", worktreeSelected: true}
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				ui.Column(c).Fill().Children(func() { a.fileRow(c, entry, test.staged) })
			}, 280, 60)
			action, ok := tt.AccessByRole(ui.RoleButton, test.action)
			if !ok || !strings.Contains(action.Description, test.state) {
				t.Fatalf("%s lost separate staging state %q: %+v", test.action, test.state, action)
			}
			if !tt.HasText(test.code) {
				t.Fatalf("staging indicator replaced file status %s", test.code)
			}
			if action.Bounds.X < 0 || action.Bounds.X+action.Bounds.W > 280 {
				t.Fatalf("staging action outside narrow file pane: %+v", action.Bounds)
			}
			if _, ok := tt.AccessByRole(ui.RoleOption, entry.Path); !ok {
				t.Fatal("original path identity changed")
			}
		})
	}
}
