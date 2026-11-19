package lobedesk

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestRepositoryMenuBoundsPathsAndRestoresKeyboardFocus(t *testing.T) {
	for _, width := range []float32{260, 980} {
		for _, keyboard := range []bool{false, true} {
			a := headShortcutApp(t)
			a.rememberRepository(a.repo.Root)
			for i := 0; i < 7; i++ {
				a.rememberRepository("/fixtures/" + strings.Repeat("long-parent/", 12) + itoa(i) + "/same-name")
			}
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				ui.Row(c).Width(184).Height(600).Children(func() { a.sidebar(c) })
			}, int(width), 640)
			title, ok := tt.AccessByRole(ui.RoleButton, a.repoName())
			if !ok {
				t.Fatal("repository title is not a button")
			}
			if keyboard {
				for i := 0; i < 40 && !title.Focused; i++ {
					tt.Key(0, ui.KeyTab)
					title, _ = tt.AccessByRole(ui.RoleButton, a.repoName())
				}
				if !title.Focused {
					t.Fatal("title is not keyboard reachable")
				}
				tt.Key(ui.Shift, ui.KeyF10)
			} else {
				tt.ClickAt(title.Bounds.X+title.Bounds.W/2, title.Bounds.Y+title.Bounds.H/2)
			}
			if _, ok := tt.AccessByRole(ui.RoleMenuItem, a.repo.Root); ok {
				t.Fatal("current checkout was listed")
			}
			for _, path := range a.recentRepositories {
				if path == a.repo.Root {
					continue
				}
				item, ok := tt.AccessByRole(ui.RoleMenuItem, path)
				if !ok || item.Disabled {
					t.Fatalf("missing enabled path %q", path)
				}
				if item.Bounds.X < 0 || item.Bounds.X+item.Bounds.W > width+1 || item.Bounds.Y+item.Bounds.H > 640 {
					t.Fatalf("menu exceeds viewport %v: %+v", width, item.Bounds)
				}
			}
			if keyboard {
				tt.Key(0, ui.KeyEnd)
				path := a.recentRepositories[len(a.recentRepositories)-2] // last displayed candidate; current root is hidden
				preview, ok := tt.AccessByRole(ui.RoleText, path)
				if !ok || preview.Bounds.H <= 20 {
					t.Fatalf("keyboard selection lacks wrapped full path: %+v", preview)
				}
			}
			tt.Key(0, ui.KeyEscape)
			if _, open := tt.AccessByRole(ui.RoleMenuItem, a.t("copyRepoPath")); open {
				t.Fatal("one Escape must close the repository menu")
			}
			if keyboard {
				title, _ = tt.AccessByRole(ui.RoleButton, a.repoName())
				if !title.Focused {
					t.Fatalf("Escape lost title focus at width %v", width)
				}
			}
		}
	}
}

func TestRecentRepositoriesKeepCheckoutRecencyAndBound(t *testing.T) {
	a := &app{}
	for _, root := range []string{"parent", "child", "linked", "parent"} {
		a.rememberRepository(root)
	}
	if !reflect.DeepEqual(a.recentRepositories, []string{"parent", "linked", "child"}) {
		t.Fatalf("wrong checkout recency: %v", a.recentRepositories)
	}
	for i := 0; i < 15; i++ {
		a.rememberRepository("checkout-" + itoa(i))
	}
	if len(a.recentRepositories) != 12 || a.recentRepositories[0] != "checkout-14" || a.recentRepositories[11] != "checkout-3" {
		t.Fatalf("wrong bounded checkout history: %v", a.recentRepositories)
	}
}

func TestRecentRepositoryOpenRespectsCurrentListAndGuards(t *testing.T) {
	for _, reason := range []string{"current", "unknown", "busy", "loading", "chooser", "dialog"} {
		t.Run(reason, func(t *testing.T) {
			a := headShortcutApp(t)
			a.recentRepositories = []string{a.repo.Root, "other"}
			path := "other"
			switch reason {
			case "current":
				path = a.repo.Root
			case "unknown":
				path = "not in list"
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "chooser":
				a.repositoryChooserOpen = true
			case "dialog":
				a.prompt.open = true
			}
			a.openRecentRepository(path)
			if a.loadSeq != 0 || a.commitText != "retained draft" {
				t.Fatal("guarded recent action started an open or changed draft")
			}
		})
	}
}

func TestRecentRepositoryMenuOpensItsExactCheckout(t *testing.T) {
	a, _, root := worktreeApp(t)
	_, _, other := worktreeApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	a.ctx = ctx
	done := make(chan func(), 64)
	a.runOnMain = func(fn func()) { done <- fn }
	t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
	for _, path := range []string{root, other, root} {
		a.openRepository(path, true)
		for a.loading {
			runContinuation(t, done)
		}
	}
	a.commitText = "draft belonging to parent"
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Row(c).Width(184).Height(600).Children(func() { a.sidebar(c) })
	}, 980, 640)
	title, _ := tt.AccessByRole(ui.RoleButton, a.repoName())
	tt.RightClickAt(title.Bounds.X+title.Bounds.W/2, title.Bounds.Y+title.Bounds.H/2)
	item, ok := tt.AccessByRole(ui.RoleMenuItem, other)
	if !ok {
		t.Fatal("missing checkout action")
	}
	tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
	if !a.loading {
		t.Fatal("menu action did not open a repository")
	}
	for a.loading {
		runContinuation(t, done)
	}
	if a.repo.Root != other || a.commitText != "" || !reflect.DeepEqual(a.recentRepositories, []string{other, root}) {
		t.Fatalf("menu opened wrong checkout or retained draft: root=%s draft=%q recent=%v", a.repo.Root, a.commitText, a.recentRepositories)
	}
}

func TestRecentRepositoryIgnoresSupersededSuccessfulOpen(t *testing.T) {
	a, _, root := worktreeApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	a.ctx = ctx
	done := make(chan func(), 64)
	a.runOnMain = func(fn func()) { done <- fn }
	t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
	a.commitText = "retained draft"
	a.openRepository(root, true)
	// Hold the successful snapshot while a newer open supersedes it.
	var accepted func()
	select {
	case accepted = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("snapshot did not finish")
	}
	a.openRepository(t.TempDir(), true)
	accepted()
	if len(a.recentRepositories) != 0 || a.commitText != "retained draft" || !a.loading {
		t.Fatal("stale success was accepted")
	}
	for a.loading {
		runContinuation(t, done)
	}
	if len(a.recentRepositories) != 0 || a.repo.Root != root || a.commitText != "retained draft" {
		t.Fatal("failed newer open or stale success changed checkout context")
	}
}

func TestRecentRepositoryOnlyRecordsSuccessfulAcceptedOpen(t *testing.T) {
	a, _, root := worktreeApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	a.ctx = ctx
	done := make(chan func(), 64)
	a.runOnMain = func(fn func()) { done <- fn }
	t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
	a.openRepository(root, true)
	for a.loading {
		runContinuation(t, done)
	}
	if !reflect.DeepEqual(a.recentRepositories, []string{root}) {
		t.Fatalf("initial checkout not recorded: %v", a.recentRepositories)
	}
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "checkout-alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{nested, alias} {
		a.openRepository(path, true)
		for a.loading {
			runContinuation(t, done)
		}
		if a.loadErr != "" || a.repo.Root != root {
			t.Fatalf("canonical open failed for %q: root=%q error=%q", path, a.repo.Root, a.loadErr)
		}
	}
	if !reflect.DeepEqual(a.recentRepositories, []string{root}) {
		t.Fatalf("canonical checkout duplicated: %v", a.recentRepositories)
	}
	a.commitText = "keep after failed open"
	a.openRepository(t.TempDir(), true)
	for a.loading {
		runContinuation(t, done)
	}
	if a.repo.Root != root || a.commitText != "keep after failed open" || !reflect.DeepEqual(a.recentRepositories, []string{root}) {
		t.Fatal("failed checkout open replaced context or recency")
	}
}
