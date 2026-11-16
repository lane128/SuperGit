package lobedesk

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestLargeWorktreeOverviewBuildsOnlyNearbyReadySections(t *testing.T) {
	for _, collapsed := range []bool{false, true} {
		t.Run(fmt.Sprint("collapsed=", collapsed), func(t *testing.T) {
			a := &app{page: "history", worktreeSelected: true, locale: localeEnglish}
			a.detailContexts.Worktree.Collapsed = make(map[detailFileKey]bool)
			for i := range 1000 {
				key := detailFileKey{Path: fmt.Sprintf("files/%04d.txt", i)}
				a.overviewEntries = append(a.overviewEntries, worktreeOverviewEntry{Key: key, Diff: gitrepo.Diff{
					Lines: []gitrepo.DiffLine{{Kind: "add", Text: fmt.Sprintf("+section-%04d", i)}},
				}})
				a.detailContexts.Worktree.Collapsed[key] = collapsed
			}
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				ui.Column(c).Width(500).Height(240).Children(func() {
					scroll := layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.commitInfoScroll}, func() { a.worktreeOverview(c) }).Grow(1).Basis(0)
					a.commitDiffViewport = scroll.Bounds()
				})
			}, 500, 240)
			for range 4 {
				tt.Frame()
			}
			if _, ok := tt.AccessByRole(ui.RoleButton, "files/0900.txt · Unstaged"); ok {
				t.Fatal("distant worktree section header built before it was visible")
			}
			first, ok := tt.AccessByRole(ui.RoleButton, "files/0000.txt · Unstaged")
			if !ok {
				t.Fatal("first worktree section missing")
			}
			second, ok := tt.AccessByRole(ui.RoleButton, "files/0001.txt · Unstaged")
			if !ok {
				t.Fatal("second worktree section missing")
			}
			height := second.Bounds.Y - first.Bounds.Y
			if height <= 0 {
				t.Fatal("worktree section height is not positive")
			}
			if a.commitInfoScroll.MaxY < 990*height {
				t.Fatal("clipping lost the complete overview scroll extent")
			}
			a.commitInfoScroll.Y = 900 * height
			for range 4 {
				tt.Frame()
			}
			found := false
			for i := 895; i < 905; i++ {
				if _, ok := tt.AccessByRole(ui.RoleButton, fmt.Sprintf("files/%04d.txt · Unstaged", i)); ok {
					found = true
					break
				}
			}
			if !found {
				t.Fatal("scrolling did not materialize distant worktree sections")
			}
			a.commitInfoScroll.Y = 0
			for range 4 {
				tt.Frame()
			}
			if _, ok := tt.AccessByRole(ui.RoleButton, "files/0000.txt · Unstaged"); !ok {
				t.Fatal("returning to the top lost the first worktree section")
			}
			if !collapsed {
				entry := a.overviewEntries[900]
				a.overviewLineTarget = &overviewPatchTarget{File: entry.Key, Diff: entry.Diff}
				a.lineCursor, a.revealLine = 0, true
				for range 5 {
					tt.Frame()
				}
				if a.revealLine || a.commitInfoScroll.Y < 850*height {
					t.Fatal("distant selected patch line was not revealed through section clipping")
				}
			}
		})
	}
}
