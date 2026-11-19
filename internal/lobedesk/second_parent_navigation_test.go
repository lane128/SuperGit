package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func secondParentShortcut(t *testing.T) appShortcut {
	t.Helper()
	for _, s := range shortcutTable() {
		if s.ID == "history.secondParent" {
			return s
		}
	}
	t.Fatal("missing second-parent shortcut")
	return appShortcut{}
}

func secondParentApp(t *testing.T) *app {
	a := parentNavigationApp(t)
	a.commits[0].Parents = []string{"first", "second", "third"}
	a.commits = append(a.commits, gitrepo.Commit{SHA: "third"})
	for i := range a.commits {
		a.commits[i].Subject = a.commits[i].SHA
	}
	a.commitFiles = []gitrepo.ChangedFile{{Path: "file.txt", Status: "M"}}
	return a
}

func TestSecondParentFromActualCommitRow(t *testing.T) {
	a := secondParentApp(t)
	s := secondParentShortcut(t)
	if s.Mod != ui.Alt|ui.Shift || s.Key != ui.KeyDown {
		t.Fatal("incorrect reference key")
	}
	tt := pageTester(a, func(c *ui.Context) { a.registerShortcuts(c); a.historyPage(c) })
	row, ok := tt.AccessByRole(ui.RoleOption, "merge")
	if !ok {
		t.Fatal("missing commit row")
	}
	tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	tt.Key(s.Mod, s.Key)
	if a.selectedSHA != "second" {
		t.Fatal("local key did not select exact second parent")
	}
	target, ok := tt.AccessByRole(ui.RoleOption, "second")
	if !ok || target.Bounds.Y < a.historyViewport.Y || target.Bounds.Y+target.Bounds.H > a.historyViewport.Y+a.historyViewport.H {
		t.Fatal("target was not revealed in commit viewport")
	}
	tt.Key(0, ui.KeyEnter)
	tt.Key(0, ui.KeySpace)
	if a.selectedSHA != "second" {
		t.Fatal("activation snapped back to invoking row")
	}
	tt.Key(0, ui.KeyUp)
	if a.selectedSHA != "unrelated" {
		t.Fatal("continuation used old focused row rather than selection")
	}
}

func TestSecondParentStaysInsideCommitList(t *testing.T) {
	for _, focus := range []string{"details tab", "file row", "search", "outside list"} {
		t.Run(focus, func(t *testing.T) {
			a := secondParentApp(t)
			s := secondParentShortcut(t)
			tt := pageTester(a, func(c *ui.Context) {
				a.registerShortcuts(c)
				if focus == "outside list" {
					ui.Column(c).Fill()
				} else {
					a.historyPage(c)
				}
			})
			switch focus {
			case "details tab":
				node, ok := tt.AccessByRole(ui.RoleTab, a.detailSummaryLabel())
				if !ok {
					t.Fatal("missing summary tab")
				}
				tt.ClickAt(node.Bounds.X+node.Bounds.W/2, node.Bounds.Y+node.Bounds.H/2)
			case "file row":
				node, ok := tt.AccessByRole(ui.RoleOption, "file.txt")
				if !ok {
					t.Fatal("missing details file row")
				}
				tt.ClickAt(node.Bounds.X+node.Bounds.W/2, node.Bounds.Y+node.Bounds.H/2)
			case "search":
				tt.Key(ui.Cmd, ui.KeyF)
			}
			tt.Key(s.Mod, s.Key)
			if a.selectedSHA != "merge" || a.pendingJumpSHA != "" {
				t.Fatal("second-parent key escaped its list focus scope")
			}
			if focus == "search" {
				tt.Type("X")
				if a.historyQuery != "X" {
					t.Fatal("local key stole search editing focus")
				}
			}
		})
	}
}

func TestSecondParentMissingTargetAndRetainedGuard(t *testing.T) {
	for _, reason := range []string{"root", "one parent", "worktree", "busy", "loading", "reload", "chooser", "reference", "dialog", "hidden"} {
		t.Run(reason, func(t *testing.T) {
			a := secondParentApp(t)
			s := secondParentShortcut(t)
			switch reason {
			case "root":
				a.commits[0].Parents = nil
			case "one parent":
				a.commits[0].Parents = a.commits[0].Parents[:1]
			case "worktree":
				a.worktreeSelected = true
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "reload":
				a.historyReloading = true
			case "chooser":
				a.repositoryChooserOpen = true
			case "reference":
				a.historyRefOpen = true
			case "dialog":
				a.confirm.open = true
			case "hidden":
				a.historyQuery = "nothing matches"
			}
			if s.Available == nil || s.Available(a) {
				t.Fatal("unavailable second-parent target offered")
			}
			s.Run(a)
			if a.selectedSHA != "merge" || a.pendingJumpSHA != "" {
				t.Fatal("retained local action ignored guard")
			}
		})
	}
}
