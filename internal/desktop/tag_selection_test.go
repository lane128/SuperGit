package desktop

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"supergit/internal/gitrepo"
	"testing"
)

func tagSelectionApp() *app {
	return &app{repo: &gitrepo.Repository{Root: "/repo"}, page: "branches", historyQuery: "unrelated filter", data: snapshot{
		history: []gitrepo.Commit{{SHA: "loaded", Subject: "loaded history row"}},
		tags:    []gitrepo.Tag{{Name: "release/旧版本", ObjectSHA: "tag-object", TargetSHA: "older-commit", TargetType: "commit", Commit: gitrepo.Commit{SHA: "older-commit", Subject: "Older target", Parents: []string{"parent"}, Author: "Test"}}},
	}}
}

func TestTagSelectionPinsUnloadedCommitWithoutInventingGraphRows(t *testing.T) {
	a := tagSelectionApp()
	commit, ok := a.selectTagCommit("release/旧版本", "tag-object")
	if !ok || commit.SHA != "older-commit" || a.selectedSHA != "older-commit" || a.page != "history" || a.historyQuery != "" || !a.historyListReset || !a.focusHistoryList {
		t.Fatal("tag target did not become the exact independent history selection")
	}
	if a.historyIndex != -1 || len(a.data.history) != 1 || a.data.history[0].SHA != "loaded" {
		t.Fatal("tag selection fabricated/reordered a graph row")
	}
	if selected := a.selectedCommit(); selected == nil || selected.Subject != "Older target" {
		t.Fatal("unloaded commit metadata unavailable to details")
	}
	// The metadata pin must not borrow an incoming snapshot's parent slice.
	a.data.tags[0].Commit.Parents[0] = "changed-snapshot"
	if a.selectedCommit().Parents[0] != "parent" {
		t.Fatal("pinned metadata borrowed mutable snapshot data")
	}
	a.data = snapshot{history: []gitrepo.Commit{{SHA: "older-commit", Subject: "Now loaded metadata"}}}
	if a.selectedCommit().Subject != "Now loaded metadata" {
		t.Fatal("pin overrode matching loaded-history metadata")
	}
	a.setSelectedCommit(gitrepo.Commit{SHA: "another", Subject: "Explicit history choice"})
	if a.tagInspection != nil || a.selectedSHA != "another" {
		t.Fatal("explicit history selection retained old tag pin")
	}
}

func TestTagSelectionRejectsStaleAndBusyTargets(t *testing.T) {
	for _, condition := range []string{"name", "oid", "noncommit", "read error", "page", "closed", "opening", "writing", "no repo"} {
		t.Run(condition, func(t *testing.T) {
			a := tagSelectionApp()
			name, oid := "release/旧版本", "tag-object"
			switch condition {
			case "name":
				name = "wrong"
			case "oid":
				oid = "moved-tag-object"
			case "noncommit":
				a.data.tags[0].Commit = gitrepo.Commit{}
			case "read error":
				a.data.tagsErr = errors.New("read failed")
			case "page":
				a.page = "history"
			case "closed":
				a.closed = true
			case "opening":
				a.opening = true
			case "writing":
				a.guard.writing = true
			case "no repo":
				a.repo = nil
			}
			if _, ok := a.selectTagCommit(name, oid); ok || a.tagInspection != nil || a.selectedSHA != "" {
				t.Fatal("stale/busy/noncommit tag changed selection")
			}
		})
	}
	a := tagSelectionApp()
	a.loading = true
	if _, ok := a.selectTagCommit("release/旧版本", "tag-object"); !ok {
		t.Fatal("background snapshot blocked read-only current tag selection")
	}
}

func TestTagSearchFiltersNamesAndExactObjectTargets(t *testing.T) {
	tags := []gitrepo.Tag{{Name: "release/候选", ObjectSHA: "aabbcc", TargetSHA: "deadbeef"}, {Name: "other", ObjectSHA: "112233", TargetSHA: "445566"}}
	for _, query := range []string{"候选", " RELEASE/", "AABBCC", "DEADBEEF"} {
		got := filterTags(tags, query)
		if len(got) != 1 || got[0].Name != "release/候选" {
			t.Fatalf("search %q chose wrong tag: %+v", query, got)
		}
	}
	if len(filterTags(tags, "  ")) != 2 || len(filterTags(tags, "missing")) != 0 {
		t.Fatal("empty query/no-match handling wrong")
	}
	a := &app{}
	row := 2
	a.tagList.Selected = &row
	a.tagSearchChanged()
	if !a.tagListReset || a.tagList.Selected != &row {
		t.Fatal("search callback destroyed borrowed list state instead of deferring reset")
	}
}

func TestSnapshotTagFailurePreservesUsableWorkingStatus(t *testing.T) {
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	r, err := gitrepo.Init(context.Background(), filepath.Join(t.TempDir(), "repo"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.Root, "draft"), []byte("keep draft"), 0600); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", r.Root}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	git("config", "user.name", "Test")
	git("config", "user.email", "test@example.invalid")
	git("commit", "--allow-empty", "-m", "base")
	if err := os.WriteFile(filepath.Join(r.Root, ".git", "refs", "tags", "broken"), []byte(strings.Repeat("f", 40)+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := readSnapshot(context.Background(), r, 300)
	if err != nil || len(s.status.Entries) != 1 || s.tagsErr == nil || len(s.tags) != 0 {
		t.Fatalf("lost workspace or tag failure was empty success: %+v %v", s, err)
	}
	found := false
	for _, warning := range s.warnings {
		found = found || strings.HasPrefix(warning, "标签：")
	}
	if !found {
		t.Fatal("tag failure omitted from snapshot warnings")
	}
}
