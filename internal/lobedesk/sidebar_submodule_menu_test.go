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

func TestSidebarSubmoduleShowsCheckedOutRef(t *testing.T) {
	const current = "0a4e5979ba02aaff9debfabb9ad4a41012aa6474"
	const recorded = "d534878fd8bc2b0e8469ef19d389722f1ad26c14"
	const longBranch = "feature/long-submodule-branch-that-must-fit-the-sidebar"
	for _, tc := range []struct {
		name, branch, current string
		initialized           bool
		label, fullRef        string
	}{
		{name: "branch", initialized: true, branch: longBranch, current: current, label: longBranch, fullRef: longBranch},
		{name: "detached", initialized: true, current: current, label: "0a4e5979", fullRef: current},
		{name: "uninitialized"},
		{name: "unavailable", initialized: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const name = "modules/Audio"
			a := &app{
				locale: localeEnglish, page: "history", repo: &gitrepo.Repository{Root: t.TempDir()},
				sidebarOpen: map[string]bool{"submodules": true},
				submodules:  []gitrepo.Submodule{{Name: name, Path: name, Branch: tc.branch, Current: tc.current, Recorded: recorded, Index: recorded, Initialized: tc.initialized}},
			}
			tt := sidebarTester(a)
			row, ok := tt.AccessByRole(ui.RoleButton, name)
			if !ok {
				t.Fatal("missing submodule row")
			}
			if strings.Contains(row.Description, recorded) {
				t.Fatalf("row reported the parent's recorded commit as the current ref: %q", row.Description)
			}
			if _, ok := tt.Find("d534878f"); ok {
				t.Fatal("row displayed the parent's recorded commit")
			}
			if tc.fullRef == "" {
				return
			}
			if !strings.Contains(row.Description, tc.fullRef) {
				t.Fatalf("row description %q does not include %q", row.Description, tc.fullRef)
			}
			label, ok := tt.Find(tc.label)
			if !ok {
				t.Fatalf("missing visible ref %q", tc.label)
			}
			if label.X < row.Bounds.X+row.Bounds.W/2 || label.Y >= row.Bounds.Y+row.Bounds.H/2 ||
				label.W < 1 || label.X+label.W > row.Bounds.X+row.Bounds.W {
				t.Fatalf("ref should fit on the right half of the row: row=%+v ref=%+v", row.Bounds, label)
			}
			leaf, ok := tt.Find("Audio")
			if !ok || leaf.W < 1 || leaf.X+leaf.W > label.X {
				t.Fatalf("module name should keep its leaf visible before the ref: leaf=%+v ref=%+v", leaf, label)
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
