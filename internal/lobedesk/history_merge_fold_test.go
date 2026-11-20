package lobedesk

import (
	"fmt"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func mergeHistoryFixture() []gitrepo.Commit {
	return []gitrepo.Commit{
		{SHA: "merge", Subject: "Merge topic", Parents: []string{"main", "tip"}},
		{SHA: "main", Subject: "Mainline change", Parents: []string{"base"}},
		{SHA: "tip", Subject: "Topic tip", Parents: []string{"side"}},
		{SHA: "side", Subject: "Earlier topic change", Parents: []string{"base"}},
		{SHA: "base", Subject: "Shared base"},
	}
}

func TestHistoryMergeFoldKeepsReferencedAndSharedCommitsVisible(t *testing.T) {
	for name, change := range map[string]func([]gitrepo.Commit) []gitrepo.Commit{
		"side has a ref": func(rows []gitrepo.Commit) []gitrepo.Commit {
			rows[3].Refs = "tag: release"
			return rows
		},
		"side shared by another head": func(rows []gitrepo.Commit) []gitrepo.Commit {
			return append([]gitrepo.Commit{{SHA: "other", Parents: []string{"side"}}}, rows...)
		},
		"side has a nested merge": func(rows []gitrepo.Commit) []gitrepo.Commit {
			rows[2].Parents = []string{"side", "base"}
			return rows
		},
		"octopus merge": func(rows []gitrepo.Commit) []gitrepo.Commit {
			rows[0].Parents = []string{"main", "tip", "base"}
			return rows
		},
	} {
		t.Run(name, func(t *testing.T) {
			branches, _ := findHistoryMergeBranches(change(mergeHistoryFixture()), "")
			if branches["merge"] {
				t.Fatal("merge must keep its side visible")
			}
		})
	}
}

func TestHistoryMergeFoldRevealsSearchedOrSelectedCommit(t *testing.T) {
	a := &app{showGraph: true, commits: mergeHistoryFixture()}
	a.historyMore = true
	if rows := a.projectHistoryMerges(a.commits); len(rows) != 5 {
		t.Fatalf("partial History must not hide commits: %+v", rows)
	}
	a.historyMore = false
	if len(a.projectHistoryMerges(a.commits)) != 3 {
		t.Fatal("expected a complete, eligible branch to start folded")
	}
	a.selectedSHA = "main"
	a.moveHistorySelection(1)
	if a.selectedSHA != "base" || len(a.projectHistoryMerges(a.commits)) != 3 {
		t.Fatalf("list navigation should skip folded commits, got %q", a.selectedSHA)
	}
	a.historyReloading = true
	a.projectHistoryMerges(a.commits[4:]) // Old scoped rows while All refs loads.
	a.historyReloading = false
	if len(a.projectHistoryMerges(a.commits)) != 3 {
		t.Fatal("reloading from a ref scope lost the fold state")
	}
	a.historyQuery = "Earlier topic"
	if rows := a.projectHistoryMerges(a.filteredHistoryRows()); len(rows) != 1 || rows[0].SHA != "side" {
		t.Fatalf("search must still find the folded commit: %+v", rows)
	}
	a.historyQuery = ""
	a.selectedSHA = "side"
	if rows := a.projectHistoryMerges(a.commits); !slices.EqualFunc(rows, a.commits, func(first, second gitrepo.Commit) bool { return first.SHA == second.SHA }) {
		t.Fatalf("navigating to a folded commit did not unfold its branch: %+v", rows)
	}
}

func TestHistoryMergeKeepsExpansionAcrossTemporarySideRef(t *testing.T) {
	a := &app{showGraph: true, commits: mergeHistoryFixture(), selectedSHA: "merge"}
	a.projectHistoryMerges(a.commits)
	a.toggleHistoryMerge("merge")
	withRef := slices.Clone(a.commits)
	withRef[2].Refs = "tag: release"
	a.projectHistoryMerges(withRef)
	if rows := a.projectHistoryMerges(a.commits); len(rows) != 5 {
		t.Fatalf("removing the side ref should preserve the explicit expansion: %+v", rows)
	}
}

func TestHistoryMergeDefaultFoldPreservesLastPageScrollAnchor(t *testing.T) {
	a := &app{showGraph: true, historyMore: true, commits: mergeHistoryFixture(), selectedSHA: "main"}
	rowHeight := a.historyRowHeight()
	// Start partway down the shared base, below two rows that will fold.
	a.historyStashExtent = 92 // Two stash rows ahead of the commits.
	firstCommit := historyListPadding + rowHeight + a.historyStashExtent
	a.historyScroll.Y = firstCommit + 4*rowHeight + 6
	a.projectHistoryMerges(a.commits)
	a.historyMore = false
	if rows := a.projectHistoryMerges(a.commits); len(rows) != 3 {
		t.Fatalf("expected the complete history to fold the merged side: %+v", rows)
	}
	if want := firstCommit + 2*rowHeight + 6; a.historyScroll.Y != want {
		t.Fatalf("scroll should stay on the shared base: got %v, want %v", a.historyScroll.Y, want)
	}
}

func TestExpandingSimpleMergeKeepsCommitSubjectsAligned(t *testing.T) {
	for _, width := range []int{980, 1600} {
		a := &app{locale: localeEnglish, page: "history", showGraph: true, themeMode: "light",
			repo: &gitrepo.Repository{Root: t.TempDir()}, commits: mergeHistoryFixture(), selectedSHA: "merge", commitFilesLoaded: true,
			status: gitrepo.Status{Head: "merge", Entries: []gitrepo.Entry{{Path: "notes.txt"}}}}
		view := ui.NewTester(a.root, width, 640)
		// Both the working-tree and commit subjects start after this viewport.
		before := a.historyGraph.Width
		for _, disclosure := range []string{"Expand merged commits", "Collapse merged commits"} {
			clickRepositoryTabsControl(t, view, ui.RoleDisclosure, disclosure)
			view.Frame()
			if _, ok := view.AccessByRole(ui.RoleOption, "Topic tip"); ok != (disclosure == "Expand merged commits") {
				t.Fatalf("%d: %s did not change the visible side rows", width, disclosure)
			}
			if got := a.historyGraph.Width; got < before-.5 || got > before+.5 {
				t.Errorf("%d: %s shifted the subject column by %.1f points", width, disclosure, got-before)
			}
			if a.historyGraphScroll.MaxX > 0 || a.selectedSHA != "merge" {
				t.Fatal("toggling the merge added unnecessary Graph panning or moved the selection")
			}
		}
	}
}

func TestHistoryMergeDisclosureStaysUnderPointerWithParallelBranches(t *testing.T) {
	for _, tc := range []struct{ width, peers int }{{980, 1}, {980, 2}, {1320, 7}, {1600, 7}} {
		t.Run(fmt.Sprintf("%d-points-%d-peers", tc.width, tc.peers), func(t *testing.T) {
			commits := mergeHistoryFixture()
			for i := 0; i < tc.peers; i++ {
				parent := fmt.Sprintf("parent%d", i)
				tip := fmt.Sprintf("peer%d", i)
				commits = append([]gitrepo.Commit{{SHA: tip, Subject: tip, Parents: []string{parent}, Refs: tip}}, commits...)
				commits = append(commits, gitrepo.Commit{SHA: parent, Subject: parent})
			}
			a := &app{locale: localeEnglish, page: "history", showGraph: true, themeMode: "light",
				repo: &gitrepo.Repository{Root: t.TempDir()}, commits: commits, selectedSHA: "merge", commitFilesLoaded: true}
			view := ui.NewTester(a.root, tc.width, 720)
			view.Frame()
			view.Frame()
			disclosure, ok := view.AccessByRole(ui.RoleDisclosure, "Expand merged commits")
			if !ok {
				t.Fatal("merge disclosure missing")
			}
			initial := a.historyGraph.Width
			bounds := disclosure.Bounds
			click := func() {
				view.ClickAt(bounds.X+bounds.W/2, bounds.Y+bounds.H/2)
				view.Frame()
				view.Frame()
			}
			click()
			if _, ok := view.AccessByRole(ui.RoleOption, "Topic tip"); !ok {
				t.Fatal("click did not expand the merge")
			}
			if got := a.historyGraph.Width; got != initial {
				t.Errorf("commit subject column moved by %.2f points", got-initial)
			}
			click()
			if _, ok := view.AccessByRole(ui.RoleOption, "Topic tip"); ok || a.selectedSHA != "merge" {
				t.Fatal("click at the original position did not collapse the merge and retain selection")
			}
		})
	}
}

func TestHistoryMergeDisclosureFitsGraphViewport(t *testing.T) {
	commits := mergeHistoryFixture()
	for i := range 16 {
		parent := fmt.Sprintf("parent%d", i)
		tip := fmt.Sprintf("peer%d", i)
		row := gitrepo.Commit{SHA: tip, Subject: tip, Parents: []string{parent}, Refs: tip}
		if i < 3 {
			commits = slices.Insert(commits, 0, row)
		} else {
			commits = slices.Insert(commits, 6, row)
		}
		commits = append(commits, gitrepo.Commit{SHA: parent, Subject: parent})
	}
	a := &app{locale: localeEnglish, page: "history", showGraph: true, themeMode: "light",
		repo: &gitrepo.Repository{Root: t.TempDir()}, commits: commits, selectedSHA: "merge", commitFilesLoaded: true}
	view := ui.NewTester(a.root, 1600, 800)
	settle := func() {
		for range 4 {
			view.Frame()
		}
	}
	settle()
	const expand = "Expand merged commits"
	rail, railOK := view.AccessByRole(ui.RoleScroll, a.t("scrollCommitGraph"))
	button, buttonOK := view.AccessByRole(ui.RoleDisclosure, expand)
	if !railOK || !buttonOK {
		t.Fatal("fixture needs an overflowing graph and visible merge button")
	}
	// Place the button's center 7.5 points from the graph edge. An 18-point
	// button cannot fit there, even though its node can still be visible.
	pan := button.Bounds.X + button.Bounds.W/2 - rail.Bounds.X - 7.5
	view.Scroll(rail.Bounds.X+3, rail.Bounds.Y+6, pan, 0)
	settle()
	view.SetSize(980, 800)
	settle()
	if got := a.historyGraphScroll.X; got != pan {
		t.Fatalf("narrowing lost the manual graph position: got %.2f, want %.2f", got, pan)
	}
	if node, ok := view.AccessByRole(ui.RoleDisclosure, expand); ok && !node.Offscreen {
		t.Errorf("a partially clipped disclosure should not be offered: %+v", node.Bounds)
	}
	row, ok := view.AccessByRole(ui.RoleOption, "Merge topic")
	if !ok || row.Offscreen {
		t.Fatal("selected merge row is unavailable")
	}
	view.ClickAt(row.Bounds.X+row.Bounds.W-30, row.Bounds.Y+row.Bounds.H/2)
	settle()
	rail, railOK = view.AccessByRole(ui.RoleScroll, a.t("scrollCommitGraph"))
	button, buttonOK = view.AccessByRole(ui.RoleDisclosure, expand)
	if !railOK || !buttonOK || button.Bounds.X < rail.Bounds.X || button.Bounds.X+button.Bounds.W > rail.Bounds.X+a.historyGraph.Width {
		t.Fatalf("clicking the merge did not reveal the whole disclosure: graph=%+v disclosure=%+v", rail.Bounds, button.Bounds)
	}
	view.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
	settle()
	if _, ok := view.AccessByRole(ui.RoleOption, "Topic tip"); !ok || a.selectedSHA != "merge" {
		t.Fatal("revealed disclosure did not expand the merge while retaining selection")
	}
}

func TestHistoryMergeNodeFoldsAndRestoresSideRows(t *testing.T) {
	for _, width := range []int{980, 1320, 1600} {
		a := &app{locale: localeEnglish, page: "history", showGraph: true, commits: mergeHistoryFixture(), selectedSHA: "merge", commitFilesLoaded: true}
		tt := ui.NewTester(func(c *ui.Context) {
			theme.Apply(c)
			a.frameSeq++
			a.historyPage(c)
		}, width, 640)
		tt.Frame()
		tt.Frame()
		if _, ok := tt.AccessByRole(ui.RoleOption, "Topic tip"); ok {
			t.Fatalf("at width %d the merged side should start folded", width)
		}
		for _, title := range []string{"Merge topic", "Mainline change", "Shared base"} {
			if _, ok := tt.AccessByRole(ui.RoleOption, title); !ok {
				t.Fatalf("at width %d folding hid %q", width, title)
			}
		}
		commit, ok := a.selectedCommit()
		if !ok || commit.SHA != "merge" || len(commit.Parents) != 2 || commit.Parents[1] != "tip" || !a.commitFilesLoaded {
			t.Fatalf("folding changed the selection or commit data: %+v", commit)
		}
		if a.historyGraph == nil || len(a.historyGraph.Rows) != 3 || a.historyGraph.Rows[0].Boundary {
			t.Fatalf("folded History graph should have only the three visible rows and no dangling merge edge")
		}
		clickRepositoryTabsControl(t, tt, ui.RoleDisclosure, "Expand merged commits")
		tt.Frame()
		for _, title := range []string{"Topic tip", "Earlier topic change"} {
			if _, ok := tt.AccessByRole(ui.RoleOption, title); !ok {
				t.Fatalf("at width %d expanding did not restore %q", width, title)
			}
		}
		a.commits = slices.Clone(a.commits) // A refresh supplies a fresh snapshot.
		tt.Frame()
		if _, ok := tt.AccessByRole(ui.RoleOption, "Topic tip"); !ok {
			t.Fatalf("at width %d refreshing refolded an explicitly opened branch", width)
		}
		clickRepositoryTabsControl(t, tt, ui.RoleDisclosure, "Collapse merged commits")
		tt.Frame()
		if _, ok := tt.AccessByRole(ui.RoleOption, "Topic tip"); ok {
			t.Fatalf("at width %d clicking did not fold the side again", width)
		}
	}
}
