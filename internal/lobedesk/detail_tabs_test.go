package lobedesk

import (
	"supergit/internal/gitrepo"
	"testing"
)

func TestDetailTabsDeduplicateByPathAndStage(t *testing.T) {
	var tabs detailTabsState
	a := detailFileKey{Path: "src/main.go"}
	b := detailFileKey{Path: "src/main.go", Staged: true}
	c := detailFileKey{Path: "cmd/main.go"}
	tabs.Open(a)
	tabs.Open(b)
	tabs.Open(c)
	tabs.Open(a)
	if len(tabs.OpenFiles) != 3 {
		t.Fatalf("tabs = %v; want three independent versions", tabs.OpenFiles)
	}
	if tabs.OpenFiles[0] != a || tabs.OpenFiles[1] != b || tabs.OpenFiles[2] != c {
		t.Fatalf("opening again changed order: %v", tabs.OpenFiles)
	}
	if tabs.Active.File != a || tabs.Active.Summary {
		t.Fatalf("reopening did not activate original file: %+v", tabs.Active)
	}
	if tabs.Activate(detailTabKey{File: detailFileKey{Path: "missing.go"}}) {
		t.Fatal("activated a missing file")
	}
	if tabs.Active.File != a {
		t.Fatal("failed activation changed active file")
	}
	if !tabs.Activate(detailTabKey{Summary: true}) || !tabs.Active.Summary {
		t.Fatal("cannot return to summary")
	}
}

func TestClosingDetailTabsChoosesTheExpectedNeighbor(t *testing.T) {
	a, b, c := detailFileKey{Path: "a"}, detailFileKey{Path: "b"}, detailFileKey{Path: "c"}
	var tabs detailTabsState
	tabs.Open(a)
	tabs.Open(b)
	tabs.Open(c)
	tabs.Activate(detailTabKey{File: b})
	tabs.Close(a)
	if tabs.Active.File != b {
		t.Fatal("closing background tab changed the active file")
	}
	tabs.Close(b)
	if tabs.Active.File != c {
		t.Fatalf("want right neighbor c, got %+v", tabs.Active)
	}
	tabs.Open(a)
	tabs.Close(a)
	if tabs.Active.File != c {
		t.Fatal("last tab did not return to left neighbor")
	}
	tabs.Close(c)
	if !tabs.Active.Summary || len(tabs.OpenFiles) != 0 {
		t.Fatalf("last close must return to summary: %+v", tabs)
	}
	tabs.Close(detailFileKey{Path: "missing"})
	if !tabs.Active.Summary {
		t.Fatal("closing unknown file changed summary")
	}
}

func TestDetailContextsFollowSelectionEntryPoints(t *testing.T) {
	a := &app{}
	a.selectCommit("aaaaaaaa")
	a.detailTabs().Open(detailFileKey{Path: "a.go"})
	a.selectCommit("bbbbbbbb")
	// Switch back before asking for tab state. Selection must invalidate the
	// previous historical context even when no frame was drawn in between.
	a.selectCommit("aaaaaaaa")
	if len(a.detailTabs().OpenFiles) != 0 {
		t.Fatal("commit navigation revived a discarded context")
	}
}

func TestDetailContextsDoNotLeakAcrossTargets(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "/repo/a"}, worktreeSelected: true}
	file := detailFileKey{Path: "same.go", Staged: true}
	a.detailTabs().Open(file)
	a.worktreeSelected = false
	a.selectedSHA = "aaaaaaaa"
	if !a.detailTabs().Active.Summary || len(a.detailTabs().OpenFiles) != 0 {
		t.Fatal("worktree leaked into history")
	}
	a.detailTabs().Open(detailFileKey{Path: "same.go"})
	a.worktreeSelected = true
	if a.detailTabs().Active.File != file {
		t.Fatal("worktree tabs were lost during history browsing")
	}
	a.worktreeSelected = false
	if len(a.detailTabs().OpenFiles) != 1 {
		t.Fatal("current history context was lost")
	}
	a.selectedSHA = "bbbbbbbb"
	if !a.detailTabs().Active.Summary || len(a.detailTabs().OpenFiles) != 0 {
		t.Fatal("previous commit tabs leaked")
	}
	a.repo = &gitrepo.Repository{Root: "/repo/b"}
	a.worktreeSelected = true
	if !a.detailTabs().Active.Summary || len(a.detailTabs().OpenFiles) != 0 {
		t.Fatal("other repository inherited worktree tabs")
	}
}
