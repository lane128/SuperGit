package desktop

import (
	"context"
	"os"
	"path/filepath"
	"supergit/internal/gitrepo"
	"testing"
)

func TestConfirmationCapturesArgumentsAndRejectsRepositorySwitch(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "/repo/a"}}
	a.guard.Switch()
	draft := actionDraft{Kind: actionPush, First: "origin", Second: "main", Third: "published", Flag: true}
	target := a.capture(draft)
	draft.First, draft.Second = "other", "wrong"
	if target.Draft.First != "origin" || target.Draft.Second != "main" {
		t.Fatal("confirmation target changed")
	}
	if !target.valid(a.repo, &a.guard) {
		t.Fatal("valid target rejected")
	}
	a.guard.Switch()
	if target.valid(a.repo, &a.guard) {
		t.Fatal("old repository generation accepted")
	}
	a.repo = &gitrepo.Repository{Root: "/repo/b"}
	if target.valid(a.repo, &a.guard) {
		t.Fatal("different repo accepted")
	}
}

func TestPreferencesRoundTripAndRecentLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	p := preferences{Dark: true, FontSize: 16}
	for i := range 20 {
		p.remember(filepath.Join("/repo", string(rune('a'+i))))
	}
	p.remember("/repo/t")
	if len(p.Recent) != 12 || p.Recent[0] != "/repo/t" {
		t.Fatalf("recents %+v", p.Recent)
	}
	if err := savePreferences(path, p); err != nil {
		t.Fatal(err)
	}
	got, err := loadPreferences(path)
	if err != nil || !got.Dark || got.FontSize != 16 || len(got.Recent) != 12 {
		t.Fatalf("roundtrip %+v %v", got, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("preferences should be private")
	}
	if err := os.WriteFile(path, []byte(`{"FontSize":999}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err = loadPreferences(path)
	if err != nil || got.FontSize != 13 {
		t.Fatal("invalid size not normalized")
	}
}

func TestHistoryFilterKeepsOriginalSelectionAndParentBoundary(t *testing.T) {
	commits := []gitrepo.Commit{{SHA: "abc", Parents: []string{"def"}, Subject: "Fix conflict", Author: "Lin"}, {SHA: "def", Subject: "Initial", Author: "Sam"}}
	matches, indices := filterHistory(commits, "FIX")
	if len(matches) != 1 || indices[0] != 0 || matches[0].SHA != "abc" {
		t.Fatal("wrong filtered selection")
	}
	rows := layoutGraph(matches)
	if !rows[0].Boundary {
		t.Fatal("missing parent presented as loaded")
	}
}

func TestAuxiliaryFailurePreservesWorkingStatus(t *testing.T) {
	r, err := gitrepo.Init(context.Background(), filepath.Join(t.TempDir(), "repo"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.Root, "new.txt"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.Root, ".gitmodules"), []byte("[broken"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := readSnapshot(context.Background(), r, 300)
	if err != nil || len(s.status.Entries) != 2 {
		t.Fatalf("lost working status %+v %v", s.status, err)
	}
	if len(s.warnings) == 0 {
		t.Fatal("auxiliary error hidden")
	}
}
