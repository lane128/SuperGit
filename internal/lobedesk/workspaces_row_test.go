package lobedesk

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

func workspaceRowsApp() *app {
	a := &app{
		locale: localeChinese,
		page:   "worktrees",
		worktrees: []gitrepo.Worktree{
			{Path: "/repos/main", Branch: "main"},
			{Path: "/repos/feat", Branch: "feat"},
		},
		submodules: []gitrepo.Submodule{
			{Name: "lib", Path: "vendor/lib", Initialized: true},
			{Name: "docs", Path: "docs", Initialized: true},
		},
	}
	a.worktreeSel = "worktree:/repos/main"
	return a
}

func worktreeRowsTester(a *app) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(700).Height(500).Children(func() {
			for i, worktree := range a.worktrees {
				a.worktreeRow(c, worktree, i+1, len(a.worktrees))
			}
			for i, module := range a.submodules {
				a.submoduleRow(c, module, i+1, len(a.submodules))
			}
		})
	}, 700, 500)
}

// The worktree rows were plain rows: the menu key and Shift+F10 could not
// reach their menu, and a screen reader got loose text. They now read as the
// option the list picked, named by the folder, described by the full path.
func TestWorktreeRowIsASelectableOption(t *testing.T) {
	a := workspaceRowsApp()
	tt := worktreeRowsTester(a)

	mainRow, ok := tt.AccessByRole(ui.RoleOption, "main")
	if !ok {
		t.Fatal("a worktree row is not an option of the list")
	}
	if !mainRow.Selected {
		t.Errorf("the chosen worktree reads %+v; want it selected", mainRow)
	}
	if !strings.Contains(mainRow.Description, "/repos/main") {
		t.Errorf("the worktree describes itself as %q; want the full path", mainRow.Description)
	}
	feat, ok := tt.AccessByRole(ui.RoleOption, "feat")
	if !ok || feat.Selected {
		t.Fatalf("the other worktree reads %+v; want an option that is not selected", feat)
	}

	tt.ClickAt(feat.Bounds.X+feat.Bounds.W/2, feat.Bounds.Y+feat.Bounds.H/2)
	if a.worktreeSel != "worktree:/repos/feat" {
		t.Errorf("clicking the row selected %q; want worktree:/repos/feat", a.worktreeSel)
	}
	tt.Key(0, ui.KeyContextMenu)
	if _, ok := tt.Find(a.t("copyAbsolutePath")); !ok {
		t.Errorf("the menu key on a worktree row shows no %q item", a.t("copyAbsolutePath"))
	}
}

func TestSubmoduleRowIsASelectableOption(t *testing.T) {
	a := workspaceRowsApp()
	tt := worktreeRowsTester(a)
	lib, ok := tt.AccessByRole(ui.RoleOption, "lib")
	if !ok {
		t.Fatal("a submodule row is not an option of the list")
	}
	if lib.Selected {
		t.Errorf("an unselected submodule reads %+v; want it not selected", lib)
	}
	if lib.Description != "vendor/lib" {
		t.Errorf("the submodule describes itself as %q; want its path", lib.Description)
	}
	tt.ClickAt(lib.Bounds.X+lib.Bounds.W/2, lib.Bounds.Y+lib.Bounds.H/2)
	if a.worktreeSel != "submodule:vendor/lib" {
		t.Errorf("clicking the row selected %q; want submodule:vendor/lib", a.worktreeSel)
	}
	if lib, _ := tt.AccessByRole(ui.RoleOption, "lib"); !lib.Selected {
		t.Errorf("lib reads %+v after the click; want it selected", lib)
	}
	tt.Key(0, ui.KeyContextMenu)
	if _, ok := tt.Find(a.t("copyAbsolutePath")); !ok {
		t.Errorf("the menu key on a submodule row shows no %q item", a.t("copyAbsolutePath"))
	}
}
