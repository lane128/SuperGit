package lobedesk

import (
	"fmt"
	"testing"

	"supergit/internal/gitrepo"
)

func TestDetailPatchesBuildOnlyNearbyRows(t *testing.T) {
	for _, worktree := range []bool{false, true} {
		t.Run(fmt.Sprint("worktree=", worktree), func(t *testing.T) {
			lines := make([]gitrepo.DiffLine, 2000)
			for i := range lines {
				lines[i] = gitrepo.DiffLine{Kind: "add", Text: fmt.Sprintf("+virtual-row-%04d", i)}
			}
			diff := gitrepo.Diff{Lines: lines}
			a := &app{page: "history", worktreeSelected: worktree, selectedSHA: "aaaa", commits: []gitrepo.Commit{{SHA: "aaaa", Subject: "large patch"}}}
			if worktree {
				a.overviewEntries = []worktreeOverviewEntry{{Key: detailFileKey{Path: "large.go"}, Diff: diff}}
			} else {
				a.commitDiff, a.commitSections = buildCommitDiff([]gitrepo.FileDiff{{Path: "large.go", Diff: diff}})
			}
			tt := pageTester(a, a.historyDiffPane)
			tt.Frame()
			if _, ok := tt.Find("+virtual-row-1500"); ok {
				t.Fatal("offscreen patch row built before it was visible")
			}
			a.commitInfoScroll.Y = 1500 * diffRowH
			for range 4 {
				tt.Frame()
			}
			found := false
			for i := 1480; i < 1530; i++ {
				if _, ok := tt.Find(fmt.Sprintf("+virtual-row-%04d", i)); ok {
					found = true
					break
				}
			}
			if !found {
				t.Fatal("scrolling did not materialize rows near the reading position")
			}
			if _, ok := tt.Find("+virtual-row-0000"); ok {
				t.Fatal("first row remains built after scrolling far away")
			}
			if a.commitInfoScroll.MaxY < 1900*diffRowH {
				t.Fatal("virtualization lost full patch scroll extent")
			}
			// A keyboard selection can jump past every currently built row.
			a.commitInfoScroll.Y = 0
			for range 3 {
				tt.Frame()
			}
			if worktree {
				a.overviewLineTarget = &overviewPatchTarget{File: detailFileKey{Path: "large.go"}, Diff: diff}
				a.lineSel = map[int]bool{1500: true}
				a.lineCursor, a.revealLine = 1500, true
			} else {
				a.commitSel = map[int]bool{1500: true}
				a.commitCursor, a.commitReveal = 1500, true
			}
			for range 5 {
				tt.Frame()
			}
			if a.commitInfoScroll.Y < 1400*diffRowH {
				t.Fatal("offscreen keyboard target did not scroll into view")
			}
		})
	}
}
