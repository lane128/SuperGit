package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

// A sidebar ref is a way to find a place in History, not a way to change what
// History lists. Clicking one selects the commit the ref points at and leaves
// the ref filter, the loaded rows and their count alone.
func TestSidebarRefsLocateTheirTipWithoutChangingTheHistoryScope(t *testing.T) {
	for _, kind := range []string{"branch", "remote", "tag"} {
		t.Run(kind, func(t *testing.T) {
			a := &app{
				ctx: canceledContext(), runOnMain: func(func()) {},
				locale: localeEnglish, page: "branches",
				repo: &gitrepo.Repository{Root: t.TempDir()},
				sidebarOpen: map[string]bool{
					"branches": true, "remotes": true, "tags": true,
				},
				branches:       []gitrepo.Branch{{Name: "main", SHA: "main-tip"}, {Name: "feature", SHA: "feature-tip"}},
				remoteBranches: []gitrepo.Branch{{Name: "origin/main", SHA: "remote-tip"}},
				tags:           []gitrepo.Tag{{Name: "v1", TargetSHA: "tag-tip", TargetType: "commit"}},
				commits: []gitrepo.Commit{
					{SHA: "head"}, {SHA: "main-tip"}, {SHA: "feature-tip"},
					{SHA: "remote-tip"}, {SHA: "tag-tip"},
				},
				selectedSHA: "head",
			}
			label, want := "feature", "feature-tip"
			switch kind {
			case "remote":
				label, want = "origin/main", "remote-tip"
			case "tag":
				label, want = "v1", "tag-tip"
			}
			tt := sidebarTester(a)
			row, ok := tt.AccessByRole(ui.RoleButton, label)
			if !ok {
				t.Fatalf("the sidebar does not list %q", label)
			}
			tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
			if a.page != "history" || a.selectedSHA != want || a.worktreeSelected {
				t.Fatalf("clicking %q left page=%q sha=%q worktree=%t", label, a.page, a.selectedSHA, a.worktreeSelected)
			}
			if a.historyRef != (historyRef{}) {
				t.Fatalf("clicking %q changed the history scope to %+v", label, a.historyRef)
			}
			if a.historyReloading || a.historyPinned != "" {
				t.Fatalf("clicking %q reloaded the list: reloading=%t pinned=%q", label, a.historyReloading, a.historyPinned)
			}
			if len(a.commits) != 5 {
				t.Fatalf("clicking %q changed the loaded rows: %d", label, len(a.commits))
			}
			if !a.revealSelection {
				t.Fatalf("clicking %q did not ask the row to be revealed", label)
			}
		})
	}
}

// The target's commit may be older than the loaded window. The click keeps
// the list's scope and pages until the commit appears instead of re-walking
// the list from the ref's tip.
func TestSidebarRefLocatesThroughPagingWithoutRescoping(t *testing.T) {
	a := &app{
		ctx: canceledContext(), runOnMain: func(func()) {},
		locale: localeEnglish, page: "history",
		repo:        &gitrepo.Repository{Root: t.TempDir()},
		sidebarOpen: map[string]bool{"tags": true},
		tags:        []gitrepo.Tag{{Name: "v1", TargetSHA: "tag-tip", TargetType: "commit"}},
		commits:     []gitrepo.Commit{{SHA: "head"}},
		historyMore: true,
		selectedSHA: "head",
	}
	tt := sidebarTester(a)
	row, ok := tt.AccessByRole(ui.RoleButton, "v1")
	if !ok {
		t.Fatal("the sidebar does not list the tag")
	}
	tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	if a.selectedSHA != "head" || a.pendingJumpSHA != "tag-tip" {
		t.Fatalf("the jump did not wait for paging: sha=%q pending=%q", a.selectedSHA, a.pendingJumpSHA)
	}
	if a.historyRef != (historyRef{}) || len(a.commits) != 1 || a.historyPinned != "" {
		t.Fatalf("paging reclaimed the scope: ref=%+v rows=%d pinned=%q", a.historyRef, len(a.commits), a.historyPinned)
	}
}

// A search is part of what the reader sees. A locate keeps it while the
// target is one of its results, and clears it only when it would hide the
// commit the click asked for.
func TestSidebarLocateKeepsASearchThatStillShowsTheTarget(t *testing.T) {
	a := &app{
		ctx: canceledContext(), runOnMain: func(func()) {},
		locale: localeEnglish, page: "history",
		repo:        &gitrepo.Repository{Root: t.TempDir()},
		sidebarOpen: map[string]bool{"tags": true},
		tags:        []gitrepo.Tag{{Name: "v1", TargetSHA: "tag-tip", TargetType: "commit"}},
		commits: []gitrepo.Commit{
			{SHA: "head", Subject: "head work"},
			{SHA: "tag-tip", Subject: "tag target"},
		},
		historyQuery: "tag",
		selectedSHA:  "head",
	}
	tt := sidebarTester(a)
	row, ok := tt.AccessByRole(ui.RoleButton, "v1")
	if !ok {
		t.Fatal("the sidebar does not list the tag")
	}
	tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	if a.selectedSHA != "tag-tip" || a.historyQuery != "tag" {
		t.Fatalf("locate changed the search: sha=%q query=%q", a.selectedSHA, a.historyQuery)
	}
	if len(a.commits) != 2 || len(a.filteredHistoryRows()) != 1 {
		t.Fatalf("locate changed the rows: loaded=%d shown=%d", len(a.commits), len(a.filteredHistoryRows()))
	}
}

func TestSidebarLocateClearsASearchThatHidesTheTarget(t *testing.T) {
	a := &app{
		ctx: canceledContext(), runOnMain: func(func()) {},
		locale: localeEnglish, page: "history",
		repo:        &gitrepo.Repository{Root: t.TempDir()},
		sidebarOpen: map[string]bool{"tags": true},
		tags:        []gitrepo.Tag{{Name: "v1", TargetSHA: "tag-tip", TargetType: "commit"}},
		commits: []gitrepo.Commit{
			{SHA: "head", Subject: "head work"},
			{SHA: "tag-tip", Subject: "tag target"},
		},
		historyQuery: "zzz",
		selectedSHA:  "head",
	}
	tt := sidebarTester(a)
	row, ok := tt.AccessByRole(ui.RoleButton, "v1")
	if !ok {
		t.Fatal("the sidebar does not list the tag")
	}
	tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	if a.selectedSHA != "tag-tip" || a.historyQuery != "" {
		t.Fatalf("locate did not reveal the hidden target: sha=%q query=%q", a.selectedSHA, a.historyQuery)
	}
}

// Selecting a stash from the sidebar selects its row in History and reveals
// it, without touching the history scope.
func TestSidebarStashSelectsAndRevealsTheSameRowInHistory(t *testing.T) {
	a := &app{
		ctx: canceledContext(), runOnMain: func(func()) {},
		locale: localeEnglish, page: "branches",
		repo:        &gitrepo.Repository{Root: t.TempDir()},
		sidebarOpen: map[string]bool{"stashes": true},
		stashes: []gitrepo.Stash{
			{SHA: "first-stash", Ref: "stash@{0}", Message: "first"},
			{SHA: "second-stash", Ref: "stash@{1}", Message: "second"},
		},
		commits:     []gitrepo.Commit{{SHA: "head"}},
		selectedSHA: "head",
	}
	tt := sidebarTester(a)
	row, ok := tt.AccessByRole(ui.RoleButton, "second")
	if !ok {
		t.Fatal("the sidebar does not list the stash")
	}
	tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	if a.page != "history" || a.selectedStash != "second-stash" || a.selectedSHA != "" {
		t.Fatalf("stash click left page=%q stash=%q sha=%q", a.page, a.selectedStash, a.selectedSHA)
	}
	if a.historyRef != (historyRef{}) || len(a.commits) != 1 || a.historyReloading {
		t.Fatalf("stash click changed the history scope: ref=%+v rows=%d reloading=%t", a.historyRef, len(a.commits), a.historyReloading)
	}
	if !a.revealSelection {
		t.Fatal("stash click did not ask the row to be revealed")
	}
}

func TestSidebarLocateAcrossHistoryScopes(t *testing.T) {
	for _, reloading := range []bool{false, true} {
		a, callbacks, shas := automaticHistoryFixture(t, 230)
		// The generated history and main have disjoint roots.
		target := shas[len(shas)-1]
		a.branches = []gitrepo.Branch{{Name: "main", SHA: a.status.Head}}
		a.tags = []gitrepo.Tag{{Name: "old", TargetSHA: target, TargetType: "commit"}}
		a.page = "history"
		a.applyHistoryRef(historyRef{Kind: historyRefLocal, Name: "main"})
		if !reloading {
			for a.historyReloading {
				historyCompletion(t, callbacks)()
			}
		}
		a.locateRef(historyRefTag, "old")
		for a.historyReloading || a.historyMore {
			historyCompletion(t, callbacks)()
		}
		if a.historyRef.Kind != "" || a.selectedSHA != target || a.worktreeSelected || !a.revealSelection {
			t.Fatalf("reloading=%v: locate tag outside main scope: ref=%v got=%q want=%q", reloading, a.historyRef, a.selectedSHA, target)
		}
	}
}
