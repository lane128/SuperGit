package lobedesk

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestConflictEditorLateReadStaysWithItsRepository(t *testing.T) {
	a, done, rootA := conflictEditorFixture(t)
	b, _, rootB := conflictEditorFixture(t)
	repoA, statusA := a.repo, a.status
	a.openConflictEditor()
	var late func()
	select {
	case late = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("first read missing")
	}
	a.repo, a.status = b.repo, b.status
	a.openConflictEditor()
	runContinuation(t, done)
	current := a.currentConflictEditor()
	current.Text = "repository B draft\n"
	late()
	if a.currentConflictEditor() != current || current.Text != "repository B draft\n" || current.Original.Root != rootB {
		t.Fatal("late read replaced current repository draft")
	}
	a.repo, a.status = repoA, statusA
	if s := a.currentConflictEditor(); s == nil || !s.Loaded || s.Original.Root != rootA || s.Text == current.Text {
		t.Fatal("original repository read was lost or mixed")
	}
}

func TestConflictEditorRejectsRemainingMarkers(t *testing.T) {
	a, done, root := conflictEditorFixture(t)
	a.openConflictEditor()
	runContinuation(t, done)
	s := a.currentConflictEditor()
	before := splitCommitGit(t, root, "ls-files", "--stage")
	a.saveConflictEditor(s)
	runContinuation(t, done)
	if s.Error == "" || splitCommitGit(t, root, "ls-files", "--stage") != before {
		t.Fatal("unresolved markers were staged")
	}
	if s.Text != s.Original.Working {
		t.Fatal("marker failure discarded draft")
	}
}

func TestConflictEditorUnsupportedBinaryReturnsToPatch(t *testing.T) {
	a, done, root := conflictEditorContentFixture(t, "current\x00bytes\n", "incoming\x00bytes\n")
	before := splitCommitGit(t, root, "ls-files", "--stage")
	tt := pageTester(a, a.diffBody)
	conflictEditorClick(t, tt, "Compare and edit result")
	runContinuation(t, done)
	tt.Frame()
	s := a.currentConflictEditor()
	if s.Loaded || s.Error == "" {
		t.Fatal("binary conflict offered text editing")
	}
	if _, ok := tt.AccessByRole(ui.RoleTextField, "Resolved contents"); ok {
		t.Fatal("binary result is editable")
	}
	conflictEditorClick(t, tt, "Show patch")
	if _, ok := tt.AccessByRole(ui.RoleButton, "Compare and edit result"); !ok {
		t.Fatal("binary fallback cannot return to patch")
	}
	if splitCommitGit(t, root, "ls-files", "--stage") != before {
		t.Fatal("binary inspection changed Index")
	}
}

func TestConflictEditorStagingFailureRecoversWithoutLosingDraft(t *testing.T) {
	a, done, root := conflictEditorFixture(t)
	a.openConflictEditor()
	runContinuation(t, done)
	s := a.currentConflictEditor()
	s.Text = "resolution retained after staging failure\n"
	before := splitCommitGit(t, root, "ls-files", "--stage")
	lock := filepath.Join(root, ".git", "index.lock")
	if err := os.WriteFile(lock, []byte("owned test lock"), 0600); err != nil {
		t.Fatal(err)
	}
	a.saveConflictEditor(s)
	runContinuation(t, done)
	data, err := os.ReadFile(filepath.Join(root, "base.txt"))
	if err != nil || string(data) != s.Text {
		t.Fatalf("expected written working result: %q %v", data, err)
	}
	if s.Error == "" || s.Text != "resolution retained after staging failure\n" || splitCommitGit(t, root, "ls-files", "--stage") != before {
		t.Fatal("staging failure lost draft or changed Index")
	}
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}
	for a.loading || a.diffLoading || a.commitFilesLoading || a.commitDiffLoading {
		runContinuation(t, done)
	}
	tt := pageTester(a, a.diffBody)
	if _, ok := tt.Find(a.t("conflictEditorRecoveryHint")); !ok {
		t.Fatal("failed save hides written-file recovery guidance")
	}
	conflictEditorClick(t, tt, "Reload versions, keep draft")
	for s.Loading {
		runContinuation(t, done)
	}
	tt.Frame()
	conflictEditorClick(t, tt, "Save and mark resolved")
	for a.busy {
		runContinuation(t, done)
	}
	if got := splitCommitGit(t, root, "show", ":base.txt"); got != "resolution retained after staging failure\n" {
		t.Fatalf("retry Index=%q", got)
	}
}

func TestConflictEditorRejectsExternalChangesAndKeepsDraft(t *testing.T) {
	a, done, root := conflictEditorFixture(t)
	a.openConflictEditor()
	runContinuation(t, done)
	s := a.currentConflictEditor()
	s.Text = "my combined draft\n"
	before := splitCommitGit(t, root, "ls-files", "--stage")
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("external saved contents\n"), 0644); err != nil {
		t.Fatal(err)
	}
	a.saveConflictEditor(s)
	runContinuation(t, done)
	if s.Error == "" || s.Text != "my combined draft\n" {
		t.Fatalf("lost failure/draft: %+v", s)
	}
	if splitCommitGit(t, root, "ls-files", "--stage") != before {
		t.Fatal("stale save changed Index")
	}
	data, err := os.ReadFile(filepath.Join(root, "base.txt"))
	if err != nil || string(data) != "external saved contents\n" {
		t.Fatalf("stale save overwrote file: %q %v", data, err)
	}
}

func TestConflictEditorSwitchingAndSourceReadingKeepDraft(t *testing.T) {
	a, done, _ := conflictEditorFixture(t)
	a.openConflictEditor()
	runContinuation(t, done)
	s := a.currentConflictEditor()
	s.Text = "draft to retain\n"
	a.selectedPath = "other.txt"
	if a.currentConflictEditor() != nil {
		t.Fatal("draft leaked to another file")
	}
	a.selectedPath = "base.txt"
	a.openConflictEditor()
	if a.currentConflictEditor() != s || s.Text != "draft to retain\n" {
		t.Fatal("switch discarded draft")
	}
	tt := pageTester(a, a.diffBody)
	for _, tc := range []struct{ label, text string }{{"Incoming (theirs)", "incoming content\n"}, {"Base", "base\n"}, {"Current (ours)", "current content\n"}} {
		conflictEditorClick(t, tt, tc.label)
		if _, ok := tt.Find(tc.text); !ok {
			t.Fatalf("missing source %q", tc.text)
		}
	}
	if s.Text != "draft to retain\n" {
		t.Fatal("source reading replaced result")
	}
}

func TestConflictEditorReloadsVersionsWithoutReplacingDraft(t *testing.T) {
	a, done, root := conflictEditorFixture(t)
	a.openConflictEditor()
	runContinuation(t, done)
	s := a.currentConflictEditor()
	s.Text = "retained resolution\n"
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("external updated working file\n"), 0644); err != nil {
		t.Fatal(err)
	}
	tt := pageTester(a, a.diffBody)
	conflictEditorClick(t, tt, "Reload versions, keep draft")
	runContinuation(t, done)
	tt.Frame()
	if s.Text != "retained resolution\n" || s.Original.Working != "external updated working file\n" {
		t.Fatalf("reload lost draft or kept stale snapshot: %+v", s)
	}
	a.saveConflictEditor(s)
	runContinuation(t, done)
	if got := splitCommitGit(t, root, "show", ":base.txt"); got != "retained resolution\n" {
		t.Fatalf("recovered Index = %q", got)
	}
}

func TestConflictEditorFailureOffersPatchWithoutLosingDraft(t *testing.T) {
	a, done, _ := conflictEditorFixture(t)
	a.openConflictEditor()
	runContinuation(t, done)
	s := a.currentConflictEditor()
	s.Text = "draft to retain\n"
	tt := pageTester(a, a.diffBody)
	conflictEditorClick(t, tt, "Show patch")
	if _, ok := tt.AccessByRole(ui.RoleTextField, "Resolved contents"); ok {
		t.Fatal("editor still covers patch")
	}
	conflictEditorClick(t, tt, "Compare and edit result")
	if a.currentConflictEditor() != s || s.Text != "draft to retain\n" {
		t.Fatal("returning to editor lost draft")
	}
}
