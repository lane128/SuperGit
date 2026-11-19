package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestFileRowEnterRespectsActionGuards(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*app)
	}{
		{"loading", func(a *app) { a.loading = true }},
		{"busy", func(a *app) { a.busy = true }},
		{"confirmation", func(a *app) { a.confirm.open = true }},
		{"prompt", func(a *app) { a.prompt.open = true }},
		{"blame", func(a *app) { a.blameOpen = true }},
		{"worktree form", func(a *app) { a.worktreeForm.open = true }},
		{"reset", func(a *app) { a.reset.open = true }},
		{"history commit", func(a *app) { a.worktreeSelected = false }},
		{"diff selection", func(a *app) { a.lineSel = map[int]bool{1: true} }},
		{"conflict", func(a *app) { a.status.Entries[0].Conflict = true }},
		{"stale row", func(a *app) { a.status.Entries = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &app{ctx: canceledContext(), runOnMain: func(func()) {}, page: "history", locale: localeEnglish, worktreeSelected: true, repo: &gitrepo.Repository{Root: t.TempDir()}, selectedPath: "a.go"}
			entry := gitrepo.Entry{Path: "a.go", Worktree: 'M'}
			a.status.Entries = []gitrepo.Entry{entry}
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				ui.Column(c).Width(400).Children(func() { a.fileRow(c, entry, false) })
			}, 400, 100)
			row, ok := tt.AccessByRole(ui.RoleOption, "a.go")
			if !ok {
				t.Fatal("missing row")
			}
			tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
			tc.setup(a)
			busy := a.busy
			seq := a.loadSeq
			tt.Key(0, ui.KeyEnter)
			if a.busy != busy || a.loadSeq != seq {
				t.Fatal("guarded Enter started a file operation")
			}
		})
	}
}
