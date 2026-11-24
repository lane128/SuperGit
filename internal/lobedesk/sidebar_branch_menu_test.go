package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestSidebarBranchMenuKeepsHistoryScope(t *testing.T) {
	for _, keyboard := range []bool{false, true} {
		name := "mouse"
		if keyboard {
			name = "keyboard"
		}
		t.Run(name, func(t *testing.T) {
			scope := historyRef{Kind: historyRefLocal, Name: "main"}
			a := &app{locale: localeEnglish, page: "history", repo: &gitrepo.Repository{Root: t.TempDir()}, historyRef: scope, branchSel: "local:main", branches: []gitrepo.Branch{{Name: "main", Current: true}, {Name: "topic", SHA: "bbbbbbbb"}}}
			a.toggleSidebarGroup("branches")
			tt := sidebarTester(a)
			row, ok := tt.AccessByRole(ui.RoleButton, "topic")
			if !ok {
				t.Fatal("missing topic sidebar row")
			}
			if keyboard {
				for step := 0; step < 30 && !row.Focused; step++ {
					tt.Key(0, ui.KeyTab)
					row, _ = tt.AccessByRole(ui.RoleButton, "topic")
				}
				if !row.Focused {
					t.Fatal("Tab cannot reach topic")
				}
				tt.Key(ui.Shift, ui.KeyF10)
			} else {
				// Mouse menus restore the prior focus, independently of their target.
				for step := 0; step < 30; step++ {
					prior, _ := tt.AccessByRole(ui.RoleButton, "main")
					if prior.Focused {
						break
					}
					tt.Key(0, ui.KeyTab)
				}
				prior, _ := tt.AccessByRole(ui.RoleButton, "main")
				if !prior.Focused {
					t.Fatal("Tab cannot reach prior focus")
				}
				tt.RightClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
			}
			if _, ok := tt.AccessByRole(ui.RoleMenuItem, a.t("renameBranch")); !ok {
				t.Fatal("sidebar branch menu is missing")
			}
			if a.historyRef != scope || a.branchSel != "local:main" || a.busy || a.loading {
				t.Fatal("opening branch menu changed history or started an action")
			}
			tt.Key(0, ui.KeyEscape)
			if a.historyRef != scope || a.branchSel != "local:main" || a.busy || a.loading {
				t.Fatal("cancelling branch menu changed repository state")
			}
			focusName := "main"
			if keyboard {
				focusName = "topic"
			}
			row, ok = tt.AccessByRole(ui.RoleButton, focusName)
			if !ok || !row.Focused {
				t.Fatal("Escape did not restore focus from before the menu opened")
			}
		})
	}
}

func TestSidebarBranchMenuDisablesWritesWhileLoading(t *testing.T) {
	for _, busy := range []bool{false, true} {
		a := &app{locale: localeEnglish, page: "history", repo: &gitrepo.Repository{Root: t.TempDir()}, busy: busy, loading: !busy, branches: []gitrepo.Branch{{Name: "topic"}}}
		a.toggleSidebarGroup("branches")
		tt := sidebarTester(a)
		row, ok := tt.AccessByRole(ui.RoleButton, "topic")
		if !ok || row.Disabled {
			t.Fatal("branch row must remain discoverable")
		}
		tt.RightClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
		for _, key := range []string{"switch", "mergeIntoCurrentAction", "rebaseOntoAction", "renameBranch", "deleteBranch"} {
			item, ok := tt.AccessByRole(ui.RoleMenuItem, a.t(key))
			if !ok || !item.Disabled {
				t.Fatalf("write action %s must be visible and disabled", key)
			}
		}
		item, ok := tt.AccessByRole(ui.RoleMenuItem, a.t("copyName"))
		if !ok || item.Disabled {
			t.Fatal("copy must remain available")
		}
	}
}
