package lobedesk

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestGroupedFileSpaceUsesFocusedVersion(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.page, a.locale = "history", localeEnglish
	a.selectedPath, a.selectedStaged = "same.go", false
	tt := pageTester(a, func(c *ui.Context) {
		ui.Row(c).Height(500).FillWidth().Children(func() { a.historyFilesPane(c) })
	})
	tt.Frame()
	tt.Frame()
	header, ok := tt.AccessByRole(ui.RoleButton, a.t("fileGroupStaged"))
	if !ok {
		t.Fatal("missing staged header")
	}
	// Fold and reopen using the same keyboard-accessible header.
	tt.ClickAt(header.Bounds.X+header.Bounds.W/2, header.Bounds.Y+header.Bounds.H/2)
	tt.Key(0, ui.KeySpace)
	tt.Key(0, ui.KeyRight) // Unstage all
	tt.Key(0, ui.KeyRight) // staged same.go
	focused := false
	for _, node := range tt.AccessAll() {
		if node.Role == ui.RoleOption && node.Name == "same.go" && node.Description == a.t("fileGroupStaged") && node.Focused {
			focused = true
		}
	}
	if !focused {
		t.Fatal("Right did not reach staged file row")
	}
	if a.selectedStaged {
		t.Fatal("fixture must retain different selected version before Space")
	}
	tt.Key(0, ui.KeySpace)
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for a.busy {
		select {
		case fn := <-callbacks:
			fn()
		case <-timer.C:
			t.Fatal("file stage operation did not finish")
		}
	}
	status, err := a.repo.Status(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range status.Entries {
		if entry.Path == "same.go" && entry.Staged() {
			t.Fatalf("Space acted on selected unstaged version instead of focused staged version: %+v", entry)
		}
	}
}

func TestHistoryFileListHasOneTabStop(t *testing.T) {
	for _, worktree := range []bool{false, true} {
		name := "commit"
		if worktree {
			name = "worktree"
		}
		t.Run(name, func(t *testing.T) {
			a := &app{page: "history", worktreeSelected: worktree, locale: localeEnglish}
			a.commitFiles = []gitrepo.ChangedFile{{Path: "a.go", Status: "M"}, {Path: "b.go", Status: "M"}}
			a.status.Entries = []gitrepo.Entry{{Path: "a.go", Index: ' ', Worktree: 'M'}, {Path: "b.go", Index: 'M', Worktree: ' '}}
			tt := pageTester(a, func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					ui.ButtonBase(c).Label("before files").Height(24)
					ui.Row(c).Height(400).FillWidth().Children(func() { a.historyFilesPane(c) })
					ui.ButtonBase(c).Label("after files").Height(24)
				})
			})
			tt.Frame()
			tt.Frame()
			// Starting at a real row avoids the separate file toolbar Tab stops.
			row, ok := tt.AccessByRole(ui.RoleOption, "a.go")
			if !ok {
				t.Fatal("missing file row")
			}
			tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
			tt.Key(0, ui.KeyTab)
			after, ok := tt.AccessByRole(ui.RoleButton, "after files")
			if !ok || !after.Focused {
				for _, node := range tt.AccessAll() {
					if node.Focused {
						t.Logf("focused after Tab: %+v", node)
					}
				}
				t.Fatalf("Tab from file list did not leave in one step (row bounds=%+v, after=%+v)", row.Bounds, after)
			}
			tt.Key(ui.Shift, ui.KeyTab)
			row, ok = tt.AccessByRole(ui.RoleOption, "a.go")
			if !ok || !row.Focused {
				t.Fatal("Shift+Tab did not return to remembered file row")
			}
			// A second row and its action must not create more Tab stops either.
			row, ok = tt.AccessByRole(ui.RoleOption, "b.go")
			if !ok {
				t.Fatal("missing second file row")
			}
			tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
			tt.Key(0, ui.KeyTab)
			after, ok = tt.AccessByRole(ui.RoleButton, "after files")
			if !ok || !after.Focused {
				t.Fatal("Tab from final file row did not leave in one step")
			}
		})
	}
}
