package lobedesk

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

// Exercise actual gitlink writes; permitting a text-hunk or destructive child
// reset is not part of restoring these existing whole-file pointer operations.
func movedGitlinkFixture(t *testing.T) (*app, func(string, ...string) string, string, string) {
	t.Helper()
	git := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	parent, child := t.TempDir(), t.TempDir()
	for _, dir := range []string{parent, child} {
		git(dir, "init", "-q")
		git(dir, "config", "user.name", "Fixture")
		git(dir, "config", "user.email", "fixture@example.invalid")
	}
	put := func(path, text string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	put(filepath.Join(child, "file"), "one\n")
	git(child, "add", "file")
	git(child, "commit", "--no-gpg-sign", "-qm", "one")
	git(parent, "-c", "protocol.file.allow=always", "submodule", "add", "-q", child, "sub")
	git(parent, "commit", "--no-gpg-sign", "-qam", "base")
	old := git(parent, "rev-parse", ":sub")
	sub := filepath.Join(parent, "sub")
	git(sub, "config", "user.name", "Fixture")
	git(sub, "config", "user.email", "fixture@example.invalid")
	put(filepath.Join(sub, "file"), "two\n")
	git(sub, "commit", "--no-gpg-sign", "-qam", "two")
	next := git(sub, "rev-parse", "HEAD")
	repo, err := gitrepo.Open(context.Background(), parent)
	if err != nil {
		t.Fatal(err)
	}
	status, err := repo.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	a := &app{ctx: context.Background(), repo: repo, status: status, locale: localeEnglish, page: "history", worktreeSelected: true}
	return a, git, old, next
}

func TestMovedGitlinkWholeFileStageAndUnstage(t *testing.T) {
	a, git, old, next := movedGitlinkFixture(t)
	head := git(a.repo.Root, "rev-parse", "HEAD")
	for _, staged := range []bool{false, true} {
		status, err := a.repo.Status(a.ctx)
		if err != nil {
			t.Fatal(err)
		}
		a.status = status
		if len(status.Entries) != 1 {
			t.Fatalf("unexpected fixture status: %+v", status.Entries)
		}
		entry := status.Entries[0]
		target, ready := a.fileActionTarget(entry, staged)
		if !ready {
			t.Fatalf("valid gitlink disabled: staged=%v entry=%+v", staged, entry)
		}
		for _, item := range a.fileMenuItems(entry, staged) {
			if (item.Key == "stage" || item.Key == "unstage") && item.Disabled {
				t.Fatal("pointer menu action disabled")
			}
			if item.Key == "discard" && !item.Disabled {
				t.Fatal("destructive submodule discard enabled")
			}
		}
		tt := pageTester(a, func(c *ui.Context) { a.fileActionButtons(c, target) })
		label := a.t("stageFile")
		if staged {
			label = a.t("unstageFile")
		}
		if button, ok := tt.AccessByRole(ui.RoleButton, label); !ok || button.Disabled {
			t.Fatal("pointer header action disabled")
		}
		if !staged {
			if button, ok := tt.AccessByRole(ui.RoleButton, a.t("discard")); !ok || !button.Disabled {
				t.Fatal("header discard protection lost")
			}
		}
		a.selectedPath, a.selectedStaged = entry.Path, staged
		if !a.canToggleSelectedStage() {
			t.Fatal("keyboard pointer action disabled")
		}
		kind := "stage"
		if staged {
			kind = "unstage"
		}
		targets, reason := a.batchFileTargets(kind)
		if reason != "" || len(targets) != 1 {
			t.Fatalf("pointer excluded from %s: %q", kind, reason)
		}
		if err := validateFileActionTargets(a.ctx, a.repo, targets, fileTargetsBatch); err != nil {
			t.Fatal(err)
		}
		action := a.prepareFileStage(target)
		if action == nil {
			t.Fatal("missing pointer action")
		}
		if err := action(a.ctx, a.repo); err != nil {
			t.Fatal(err)
		}
		want := next
		if staged {
			want = old
		}
		if got := git(a.repo.Root, "rev-parse", ":sub"); got != want {
			t.Fatalf("Index gitlink=%s want %s", got, want)
		}
		if git(a.repo.Root, "rev-parse", "HEAD") != head || git(filepath.Join(a.repo.Root, "sub"), "rev-parse", "HEAD") != next {
			t.Fatal("pointer operation moved parent or child HEAD")
		}
	}
	if got, err := os.ReadFile(filepath.Join(a.repo.Root, "sub", "file")); err != nil || string(got) != "two\n" {
		t.Fatal("pointer operation changed child bytes")
	}
}

func TestBatchGitlinkStageAndUnstageWriteOnlyParentIndex(t *testing.T) {
	for _, staged := range []bool{false, true} {
		t.Run(fmt.Sprint(staged), func(t *testing.T) {
			a, git, old, next := movedGitlinkFixture(t)
			if staged {
				if err := a.repo.Stage(a.ctx, "sub"); err != nil {
					t.Fatal(err)
				}
			}
			status, err := a.repo.Status(a.ctx)
			if err != nil {
				t.Fatal(err)
			}
			a.status = status
			callbacks := make(chan func(), 4)
			a.runOnMain = func(fn func()) { callbacks <- fn }
			head := git(a.repo.Root, "rev-parse", "HEAD")
			a.runBatchStage(staged)
			// Receiving the write completion proves Git finished. Do not execute
			// it here: its subsequent repository refresh is a separate lifecycle.
			_ = historyCompletion(t, callbacks)
			want := next
			if staged {
				want = old
			}
			if git(a.repo.Root, "rev-parse", ":sub") != want {
				t.Fatal("batch did not write requested gitlink version")
			}
			if git(a.repo.Root, "rev-parse", "HEAD") != head || git(filepath.Join(a.repo.Root, "sub"), "rev-parse", "HEAD") != next {
				t.Fatal("batch changed a HEAD")
			}
			if got, err := os.ReadFile(filepath.Join(a.repo.Root, "sub", "file")); err != nil || string(got) != "two\n" {
				t.Fatal("batch changed child bytes")
			}
		})
	}
}

func TestDirtyOnlySubmoduleDoesNotBlockUnstageOfOtherFiles(t *testing.T) {
	a, git, _, _ := movedGitlinkFixture(t)
	git(a.repo.Root, "add", "sub") // Match the current child pointer in the Index.
	if err := os.WriteFile(filepath.Join(a.repo.Root, "sub", "file"), []byte("uncommitted\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(a.repo.Root, "other.txt"), []byte("staged\n"), 0600); err != nil {
		t.Fatal(err)
	}
	git(a.repo.Root, "add", "other.txt")
	// Commit only the gitlink so the submodule row is internal-dirty only.
	git(a.repo.Root, "commit", "--no-gpg-sign", "-qm", "pointer", "--", "sub")
	status, err := a.repo.Status(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.status = status
	targets, reason := a.batchFileTargets("unstage")
	if reason != "" || len(targets) != 1 || targets[0].Entry.Path != "other.txt" {
		t.Fatalf("dirty child blocks unrelated unstage: %v %q", targets, reason)
	}
	if err := validateFileActionTargets(a.ctx, a.repo, targets, fileTargetsBatch); err != nil {
		t.Fatal(err)
	}
	for _, e := range status.Entries {
		if e.IsSubmodule() {
			if _, ready := a.fileActionTarget(e, false); ready {
				t.Fatal("internal child edits advertised as stageable pointer")
			}
		}
	}
}
