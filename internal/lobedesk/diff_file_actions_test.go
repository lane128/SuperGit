package lobedesk

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestOverviewFileActionsKeepExactTarget(t *testing.T) {
	for _, staged := range []bool{false, true} {
		a, _ := overviewFixture(t)
		target, ok := a.fileActionTarget(a.status.Entries[0], staged)
		if !ok {
			t.Fatal("fixture target unavailable")
		}
		a.selectedPath, a.selectedStaged = "unrelated.txt", !staged
		action := a.prepareFileStage(target)
		if action == nil {
			t.Fatal("stage action missing")
		}
		if err := action(a.ctx, a.repo); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("git", "-C", a.repo.Root, "show", ":same.go")
		got, err := cmd.Output()
		want := "working version\n"
		if staged {
			want = "original\n"
		}
		if err != nil || string(got) != want {
			t.Fatalf("Index = %q, %v; want %q", got, err, want)
		}
		if a.selectedPath != "unrelated.txt" || a.selectedStaged != !staged {
			t.Fatal("header action moved background file cursor")
		}
		if got, _ := os.ReadFile(filepath.Join(a.repo.Root, "same.go")); string(got) != "working version\n" {
			t.Fatal("stage/unstage changed working bytes")
		}
	}
}

func TestCollapsedFileKeepsItsActions(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.locale, a.page = localeEnglish, "history"
	a.loadWorktreeOverview()
	waitOverview(t, a, callbacks)
	a.setOverviewCollapsed(true)
	tt := pageTester(a, a.worktreeOverview)
	for _, label := range []string{a.t("stageFile"), a.t("unstageFile"), a.t("discard")} {
		button, ok := tt.AccessByRole(ui.RoleButton, label)
		if !ok || button.Disabled {
			t.Fatalf("collapsed block lost %q action", label)
		}
	}
}

func TestHistoricalDiffHasNoWriteActions(t *testing.T) {
	a := &app{locale: localeEnglish, page: "history", selectedSHA: "aaaaaaaa", commitPath: "same.go", commitDiff: gitrepo.Diff{Lines: []gitrepo.DiffLine{{Kind: "add", Text: "historical payload", New: 1}}}}
	a.commitDiff, a.commitSections = buildCommitDiff([]gitrepo.FileDiff{{Path: "same.go", Diff: a.commitDiff}})
	tt := pageTester(a, func(c *ui.Context) { a.historyDiffFile(c, theme.TokensOf(c)) })
	tt.Frame()
	if !strings.Contains(strings.Join(tt.Texts(), "\n"), "historical payload") {
		t.Fatal("test did not render the historical patch")
	}
	for _, label := range []string{a.t("stageFile"), a.t("unstageFile"), a.t("discard"), a.t("stageAll"), a.t("discardAll")} {
		if _, ok := tt.AccessByRole(ui.RoleButton, label); ok {
			t.Fatalf("historical view exposes %q", label)
		}
	}
}

func TestOverviewDiscardButtonOpensConfirmation(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.locale = localeEnglish
	tt := pageTester(a, a.overviewBatchActions)
	button, ok := tt.AccessByRole(ui.RoleButton, a.t("discardAll"))
	if !ok || button.Disabled {
		t.Fatal("Discard All button unavailable")
	}
	tt.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
	tt.Frame()
	waitDiscardPreparation(t, callbacks)
	if !a.confirm.open || a.confirm.title != a.t("discardAll") {
		t.Fatal("Discard All click did not open its confirmation")
	}
}

func TestBatchActionsDeclareTheirScope(t *testing.T) {
	a, _ := overviewFixture(t)
	name := "-空 格\nnew.txt"
	if err := os.WriteFile(filepath.Join(a.repo.Root, name), []byte("new\n"), 0600); err != nil {
		t.Fatal(err)
	}
	status, err := a.repo.Status(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.status = status
	stage, reason := a.batchFileTargets("stage")
	if reason != "" || len(stage) != 2 {
		t.Fatalf("Stage All excludes untracked or modified: %v %q", stage, reason)
	}
	discard, reason := a.batchFileTargets("discard")
	if reason != "" || len(discard) != 1 || discard[0].Entry.Path != "same.go" || discard[0].Staged {
		t.Fatal("Discard All includes untracked or staged scope")
	}
	unstage, reason := a.batchFileTargets("unstage")
	if reason != "" || len(unstage) != 1 || !unstage[0].Staged {
		t.Fatal("Unstage All has wrong scope")
	}
	a.status.Entries = append(a.status.Entries, gitrepo.Entry{Path: "conflict", Conflict: true})
	for _, kind := range []string{"stage", "unstage", "discard"} {
		if targets, reason := a.batchFileTargets(kind); len(targets) != 0 || reason == "" {
			t.Fatalf("%s silently skips conflicting entry", kind)
		}
	}
}

func TestFileActionControlsShareProtectedState(t *testing.T) {
	a, _ := overviewFixture(t)
	a.locale = localeEnglish
	a.status.Entries[0].Submodule = "S..."
	entry := a.status.Entries[0]
	a.page, a.selectedPath = "history", entry.Path
	if a.canToggleSelectedStage() {
		t.Fatal("keyboard Stage is available for protected submodule")
	}
	tt := pageTester(a, func(c *ui.Context) { a.fileRow(c, entry, false) })
	button, ok := tt.AccessByRole(ui.RoleButton, a.t("stageFile"))
	if !ok || !button.Disabled {
		t.Fatal("file row offers enabled Stage for protected submodule")
	}
	for _, item := range a.fileMenuItems(entry, false) {
		if (item.Key == "stage" || item.Key == "discard") && !item.Disabled {
			t.Fatalf("menu offers enabled %s for protected submodule", item.Key)
		}
	}
}

func TestBatchActionsWriteOnlyTheirDeclaredScope(t *testing.T) {
	for _, kind := range []string{"stage", "unstage", "discard"} {
		t.Run(kind, func(t *testing.T) {
			a, callbacks := overviewFixture(t)
			a.locale = localeEnglish
			name := "-空 格\nnew.txt"
			if err := os.WriteFile(filepath.Join(a.repo.Root, name), []byte("keep untracked\n"), 0600); err != nil {
				t.Fatal(err)
			}
			status, err := a.repo.Status(a.ctx)
			if err != nil {
				t.Fatal(err)
			}
			a.status = status
			readGit := func(args ...string) string {
				t.Helper()
				out, err := exec.Command("git", append([]string{"-C", a.repo.Root}, args...)...).CombinedOutput()
				if err != nil {
					t.Fatalf("git %v: %v %s", args, err, out)
				}
				return string(out)
			}
			head, index := readGit("rev-parse", "HEAD"), readGit("ls-files", "--stage", "-z")
			switch kind {
			case "stage":
				a.stageAll()
			case "unstage":
				a.unstageAll()
			case "discard":
				a.confirmDiscardAll()
				waitDiscardPreparation(t, callbacks)
				if !a.confirm.open || !strings.Contains(a.confirm.message, "untracked") || !strings.Contains(a.confirm.message, "Index") {
					t.Fatal("Discard All confirmation omits exclusions")
				}
				action := a.confirm.action
				a.confirm = confirmState{}
				action()
			}
			select {
			case <-callbacks:
			case <-time.After(10 * time.Second):
				t.Fatal("batch did not finish")
			}
			if readGit("rev-parse", "HEAD") != head {
				t.Fatal("batch changed HEAD")
			}
			working, _ := os.ReadFile(filepath.Join(a.repo.Root, "same.go"))
			if kind == "discard" {
				if string(working) != "index version\n" || readGit("ls-files", "--stage", "-z") != index {
					t.Fatal("Discard All did not preserve Index")
				}
			} else {
				if string(working) != "working version\n" {
					t.Fatal("stage/unstage changed working bytes")
				}
				want := "working version\n"
				if kind == "unstage" {
					want = "original\n"
				}
				if readGit("show", ":same.go") != want {
					t.Fatal("batch has wrong Index result")
				}
			}
			if kind == "stage" && readGit("show", ":"+name) != "keep untracked\n" {
				t.Fatal("Stage All omitted special untracked path")
			}
			if got, _ := os.ReadFile(filepath.Join(a.repo.Root, name)); string(got) != "keep untracked\n" {
				t.Fatal("batch removed untracked file")
			}
		})
	}
}

func waitDiscardPreparation(t *testing.T, callbacks chan func()) {
	t.Helper()
	select {
	case fn := <-callbacks:
		fn()
	case <-time.After(10 * time.Second):
		t.Fatal("discard preparation did not finish")
	}
}

func TestDiscardConfirmationRejectsChangedContext(t *testing.T) {
	for _, change := range []string{"root", "generation", "entry", "pending root"} {
		t.Run(change, func(t *testing.T) {
			a, callbacks := overviewFixture(t)
			root := a.repo.Root
			a.confirmDiscard(a.status.Entries[0])
			if change == "pending root" {
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
				waitDiscardPreparation(t, callbacks)
				if a.confirm.open {
					t.Fatal("late preparation opened a stale-root confirmation")
				}
			} else {
				waitDiscardPreparation(t, callbacks)
				if !a.confirm.open || a.confirm.action == nil {
					t.Fatal("discard confirmation missing")
				}
				action := a.confirm.action
				a.confirm = confirmState{}
				switch change {
				case "root":
					a.repo = &gitrepo.Repository{Root: t.TempDir()}
				case "generation":
					a.loadSeq++
				case "entry":
					a.status.Entries[0].Conflict = true
				}
				action()
				if a.busy {
					t.Fatal("stale confirmation started a write")
				}
			}
			if got, _ := os.ReadFile(filepath.Join(root, "same.go")); string(got) != "working version\n" {
				t.Fatal("stale confirmation discarded original file")
			}
		})
	}
}

func TestDiscardConfirmationCancellationDoesNotWrite(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.confirmDiscard(a.status.Entries[0])
	waitDiscardPreparation(t, callbacks)
	tt := pageTester(a, a.confirmDialog)
	cancel, ok := tt.AccessByRole(ui.RoleButton, a.t("cancel"))
	if !ok {
		t.Fatal("discard confirmation has no Cancel")
	}
	tt.ClickAt(cancel.Bounds.X+cancel.Bounds.W/2, cancel.Bounds.Y+cancel.Bounds.H/2)
	if a.confirm.open || a.busy {
		t.Fatal("cancelling confirmation started an operation")
	}
	if got, _ := os.ReadFile(filepath.Join(a.repo.Root, "same.go")); string(got) != "working version\n" {
		t.Fatal("cancel changed file")
	}
}

func TestDiscardConfirmationRejectsExternalEdits(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.confirmDiscard(a.status.Entries[0])
	waitDiscardPreparation(t, callbacks)
	if !a.confirm.open {
		t.Fatal("confirmation missing")
	}
	path := filepath.Join(a.repo.Root, "same.go")
	if err := os.WriteFile(path, []byte("new external edit\n"), 0600); err != nil {
		t.Fatal(err)
	}
	action := a.confirm.action
	a.confirm = confirmState{}
	action()
	// Receiving the completion proves the backend finished. Do not execute the
	// UI reload continuation here: this assertion concerns the protected write.
	select {
	case <-callbacks:
	case <-time.After(10 * time.Second):
		t.Fatal("discard did not finish")
	}
	if got, _ := os.ReadFile(path); string(got) != "new external edit\n" {
		t.Fatal("confirmation overwrote external edit")
	}
}
