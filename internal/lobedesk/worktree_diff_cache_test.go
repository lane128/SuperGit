package lobedesk

import (
	"os"
	"path/filepath"
	"strings"
	"supergit/internal/gitrepo"
	"testing"
	"time"
)

func TestWorktreeDiffCacheBoundsAndIdentity(t *testing.T) {
	cache := worktreeDiffCache{limit: 1024}
	a, b := detailFileKey{Path: "same.go"}, detailFileKey{Path: "same.go", Staged: true}
	diff := gitrepo.Diff{Text: strings.Repeat("x", 700)}
	cache.put("root", 1, a, diff)
	if _, ok := cache.get("root", 1, a); !ok {
		t.Fatal("missing inserted cache entry")
	}
	if _, ok := cache.get("root", 1, b); ok {
		t.Fatal("cache merged staged identities")
	}
	cache.put("root", 1, b, diff)
	if _, ok := cache.get("root", 1, a); ok {
		t.Fatal("cache did not evict oldest entry at budget")
	}
	if _, ok := cache.get("root", 1, b); !ok {
		t.Fatal("cache lost new entry")
	}
	if _, ok := cache.get("other-root", 1, b); ok {
		t.Fatal("cache crossed repository identity")
	}
	if _, ok := cache.get("root", 2, b); ok {
		t.Fatal("cache crossed refresh epoch")
	}
	cache.put("root", 2, a, gitrepo.Diff{})
	if _, ok := cache.get("root", 2, a); !ok {
		t.Fatal("successful empty diff was treated as a miss")
	}
	cache.put("root", 2, b, gitrepo.Diff{Text: strings.Repeat("z", 1025)})
	if _, ok := cache.get("root", 2, b); ok {
		t.Fatal("oversize diff exceeded cache budget")
	}
	cache.put("root", 3, b, diff)
	if _, ok := cache.get("root", 2, a); ok {
		t.Fatal("old generation survived replacement")
	}
}

func TestWorktreeFileTabReusesOverviewUntilRefresh(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.page = "history"
	a.loadWorktreeOverview()
	waitOverview(t, a, callbacks)
	key := detailFileKey{Path: "same.go"}
	var want string
	for _, entry := range a.overviewEntries {
		if entry.Key == key {
			want = entry.Diff.Text
		}
	}
	if want == "" {
		t.Fatal("fixture lacks working diff")
	}
	if err := os.WriteFile(filepath.Join(a.repo.Root, key.Path), []byte("external edit\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a.openDetailFile(key)
	if a.diffLoading || a.diff.Text != want {
		t.Fatal("file tab did not reuse overview's current snapshot")
	}
	// A refresh epoch must discard cached bytes, even with identical status codes.
	a.loadSeq++
	a.activateDetailTab(detailTabKey{Summary: true})
	a.loadWorktreeOverview()
	waitOverview(t, a, callbacks)
	a.openDetailFile(key)
	if a.diffLoading || !strings.Contains(a.diff.Text, "+external edit") {
		t.Fatalf("refresh reused an old patch: %q", a.diff.Text)
	}
}

func TestWorktreeOverviewReusesSuccessfulSingleFileRead(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.page = "history"
	a.selectedPath = "same.go"
	a.loadDiff()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for a.diffLoading {
		select {
		case fn := <-callbacks:
			fn()
		case <-timer.C:
			t.Fatal("single-file diff did not finish")
		}
	}
	want := a.diff.Text
	if want == "" || a.diffErr != "" {
		t.Fatal("fixture single-file diff failed")
	}
	if err := os.WriteFile(filepath.Join(a.repo.Root, "same.go"), []byte("later edit\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a.loadWorktreeOverview()
	waitOverview(t, a, callbacks)
	found := false
	for _, entry := range a.overviewEntries {
		if entry.Key == (detailFileKey{Path: "same.go"}) {
			found = true
			if entry.Err != "" || entry.Diff.Text != want {
				t.Fatal("overview reread the same working snapshot instead of reusing the file patch")
			}
		}
	}
	if !found {
		t.Fatal("missing cached file in overview")
	}
}
