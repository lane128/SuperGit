package lobedesk

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestSidebarGroupCountStaysInsideScrollGutter(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		t.Run(string(language), func(t *testing.T) {
			a := &app{locale: language, page: "history", repo: &gitrepo.Repository{Root: t.TempDir()}, sidebarOpen: map[string]bool{"branches": false}, branches: make([]gitrepo.Branch, 12345)}
			tt := sidebarTester(a)
			count, ok := tt.Find("12345")
			if !ok {
				t.Fatal("missing branch count")
			}
			if path := os.Getenv("SUPERGIT_SIDEBAR_SCREENSHOT"); path != "" && language == localeEnglish {
				file, err := os.Create(path)
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(file, tt.Image())
				closeErr := file.Close()
				if err != nil {
					t.Fatal(err)
				}
				if closeErr != nil {
					t.Fatal(closeErr)
				}
			}
			// Sidebar has 6 DIP outer padding; ScrollArea overlays its last 12 DIP.
			right := a.workbenchSidebarWidth() - 6 - 12
			if count.W <= 0 || count.X+count.W > right {
				t.Fatalf("count overlaps scrollbar or clip: %+v, safe right=%v", count, right)
			}
		})
	}
}

func TestSidebarWorkspaceOpensHistory(t *testing.T) {
	for _, kind := range []string{"worktree", "submodule"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			a := &app{ctx: canceledContext(), runOnMain: func(func()) {}, locale: localeEnglish, page: "branches", repo: &gitrepo.Repository{Root: root}, sidebarOpen: map[string]bool{"worktrees": true, "submodules": true}}
			label := "child"
			if kind == "worktree" {
				a.worktrees = []gitrepo.Worktree{{Path: root + "/child"}}
			} else {
				a.submodules = []gitrepo.Submodule{{Name: label, Path: label, Initialized: true}}
			}
			tt := sidebarTester(a)
			row, ok := tt.AccessByRole(ui.RoleButton, label)
			if !ok {
				t.Fatal("missing workspace row")
			}
			tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
			if a.page != "history" || !a.loading {
				t.Fatalf("workspace opened page=%s loading=%v", a.page, a.loading)
			}
		})
	}
}

func TestSidebarSubmoduleLoadsItsOwnHistory(t *testing.T) {
	child, callbacks := overviewFixture(t)
	childRoot, err := filepath.EvalSymlinks(child.repo.Root)
	if err != nil {
		t.Fatal(err)
	}
	parentRoot := filepath.Dir(childRoot)
	child.repo = &gitrepo.Repository{Root: parentRoot}
	child.page = "branches"
	child.submodules = []gitrepo.Submodule{{Name: "child", Path: filepath.Base(childRoot), Initialized: true}}
	child.openSidebarWorkspace("submodule:" + filepath.Base(childRoot))
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	for child.loading {
		select {
		case fn := <-callbacks:
			fn()
		case <-timer.C:
			t.Fatal("submodule history did not finish loading")
		}
	}
	if child.repo.Root != childRoot || child.page != "history" || len(child.commits) != 1 || child.commits[0].Subject != "fixture" {
		t.Fatalf("wrong repository history: root=%s page=%s commits=%+v", child.repo.Root, child.page, child.commits)
	}
}

func TestSidebarUninitializedSubmoduleDoesNotOpenParent(t *testing.T) {
	a := &app{locale: localeEnglish, page: "history", repo: &gitrepo.Repository{Root: t.TempDir()}, sidebarOpen: map[string]bool{"submodules": true}, submodules: []gitrepo.Submodule{{Name: "child", Path: "child"}}}
	tt := sidebarTester(a)
	row, ok := tt.AccessByRole(ui.RoleButton, "child")
	if !ok {
		t.Fatal("missing submodule row")
	}
	tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	tt.Key(0, ui.KeyEnter)
	tt.Key(0, ui.KeySpace)
	if a.page != "history" || a.loading || a.busy {
		t.Fatalf("uninitialized module navigated or initialized: page=%s loading=%v busy=%v", a.page, a.loading, a.busy)
	}
}
