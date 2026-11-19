package lobedesk

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestSidebarSubmoduleMenuIsReachable(t *testing.T) {
	for _, initialized := range []bool{false, true} {
		name := "uninitialized"
		if initialized {
			name = "initialized"
		}
		t.Run(name, func(t *testing.T) {
			for _, keyboard := range []bool{false, true} {
				a := &app{locale: localeEnglish, page: "history", repo: &gitrepo.Repository{Root: t.TempDir()}, sidebarOpen: map[string]bool{"submodules": true}, submodules: []gitrepo.Submodule{{Name: "child", Path: "vendor/child", Initialized: initialized}}}
				tt := sidebarTester(a)
				row, ok := tt.AccessByRole(ui.RoleButton, "child")
				if !ok || row.Disabled || !row.Focusable {
					t.Fatal("submodule row must expose its actions even before initialization")
				}
				if !strings.Contains(row.Description, "vendor/child") {
					t.Fatal("submodule row does not identify its path")
				}
				if !initialized && !strings.Contains(row.Description, a.t("notInitialized")) {
					t.Fatal("missing uninitialized state")
				}
				if keyboard {
					for step := 0; step < 30 && !row.Focused; step++ {
						tt.Key(0, ui.KeyTab)
						row, _ = tt.AccessByRole(ui.RoleButton, "child")
					}
					if !row.Focused {
						t.Fatal("Tab cannot reach module row")
					}
					tt.Key(ui.Shift, ui.KeyF10)
				} else {
					tt.RightClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
				}
				label := a.t("initializeSubmodule")
				if initialized {
					label = a.t("updateSubmodule")
				}
				if _, ok := tt.AccessByRole(ui.RoleMenuItem, label); !ok {
					t.Fatalf("missing module action %q", label)
				}
				_, hasOpen := tt.AccessByRole(ui.RoleMenuItem, a.t("openSubmodule"))
				if hasOpen != initialized {
					t.Fatal("Open is not restricted to initialized modules")
				}
				tt.Key(0, ui.KeyEscape)
				if a.page != "history" || a.loading || a.busy {
					t.Fatal("opening or cancelling menu changed repository state")
				}
			}
		})
	}
}

func TestSidebarSubmoduleStateDoesNotConsumeNameWidth(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		a := &app{locale: language, page: "history", repo: &gitrepo.Repository{Root: t.TempDir()}, sidebarOpen: map[string]bool{"submodules": true}, submodules: []gitrepo.Submodule{{Name: "fixture-child", Path: "child"}}}
		tt := sidebarTester(a)
		row, ok := tt.AccessByRole(ui.RoleButton, "fixture-child")
		if !ok {
			t.Fatal("missing module row")
		}
		state, ok := tt.Find(a.t("notInitialized"))
		if !ok {
			t.Fatal("missing initialization state")
		}
		if row.Bounds.H < 38 || state.Y < row.Bounds.Y+row.Bounds.H/2-1 || state.X+state.W > row.Bounds.X+row.Bounds.W {
			t.Fatalf("state competes with module name or escapes sidebar: row=%+v state=%+v", row.Bounds, state)
		}
	}
}

func TestSidebarSubmoduleMenuDisablesWritesWhileLoading(t *testing.T) {
	for _, busy := range []bool{false, true} {
		a := &app{locale: localeEnglish, page: "history", repo: &gitrepo.Repository{Root: t.TempDir()}, busy: busy, loading: !busy, sidebarOpen: map[string]bool{"submodules": true}, submodules: []gitrepo.Submodule{{Name: "child", Path: "child"}}}
		tt := sidebarTester(a)
		row, ok := tt.AccessByRole(ui.RoleButton, "child")
		if !ok || row.Disabled {
			t.Fatal("module actions are undiscoverable while busy")
		}
		tt.RightClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
		action, ok := tt.AccessByRole(ui.RoleMenuItem, a.t("initializeSubmodule"))
		if !ok || !action.Disabled {
			t.Fatal("initialization must be visible and disabled while loading or busy")
		}
	}
}
