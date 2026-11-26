package gitrepo

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestWorkspaceStashRoundTrip(t *testing.T) {
	dir, r := workspaceMixed(t)
	// Git stash cannot save intent-to-add; the ordinary staged addition covers
	// restoration of a partially staged new file in this round trip.
	git(t, dir, "add", "intent")
	before := git(t, dir, "ls-files", "--stage", "-z")
	head := git(t, dir, "rev-parse", "HEAD")
	if err := r.SaveWorkspaceStash(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "status", "--porcelain"); got != "" {
		t.Fatalf("dirty: %s", got)
	}
	requireWorkspaceBytes(t, dir, "new/preserve.keep", "ignored bytes")
	entries, err := r.Stashes(context.Background())
	if err != nil || len(entries) != 1 {
		t.Fatalf("stash: %v %v", entries, err)
	}
	git(t, dir, "stash", "apply", "--index", entries[0].SHA)
	if got := git(t, dir, "ls-files", "--stage", "-z"); got != before {
		t.Fatal("index not restored")
	}
	requireWorkspaceBytes(t, dir, "base.txt", "unstaged\n")
	requireWorkspaceBytes(t, dir, "added", "partially staged addition")
	requireWorkspaceBytes(t, dir, "binary", "\x00\xff\x80binary\x00")
	for _, name := range []string{"空 格", "line\nbreak", "-option", ":(glob)literal"} {
		requireWorkspaceBytes(t, dir, name, "new bytes")
	}
	if git(t, dir, "rev-parse", "HEAD") != head {
		t.Fatal("HEAD moved")
	}
}

func TestWorkspaceStashRefusesIntentToAddBeforeWrites(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, "intent", "pending")
	git(t, dir, "add", "-N", "intent")
	before := workspaceFingerprint(t, dir)
	err := openTest(t, dir).SaveWorkspaceStash(context.Background())
	var action *WorkspaceActionError
	if !errors.As(err, &action) || action.Partial || action.Phase != "preflight" {
		t.Fatalf("want preflight refusal, got %v", err)
	}
	if !reflect.DeepEqual(before, workspaceFingerprint(t, dir)) {
		t.Fatal("intent-to-add refusal wrote repository")
	}
}

func TestWorkspaceStashOwnOIDWithLinkedWorktree(t *testing.T) {
	for _, removeOwn := range []bool{false, true} {
		t.Run(map[bool]string{false: "other-pushed", true: "own-disappeared"}[removeOwn], func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			linked := filepath.Join(t.TempDir(), "linked")
			git(t, dir, "worktree", "add", "-b", "other", linked)
			write(t, dir, "base.txt", "our changes")
			write(t, linked, "base.txt", "their changes")
			realGit, err := exec.LookPath("git")
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("WORKSPACE_TEST_REAL_GIT", realGit)
			t.Setenv("WORKSPACE_TEST_LINKED", linked)
			t.Setenv("WORKSPACE_TEST_ROOT", dir)
			drop := ""
			if removeOwn {
				drop = "\"$WORKSPACE_TEST_REAL_GIT\" -C \"$WORKSPACE_TEST_ROOT\" stash drop 'stash@{0}' || exit $?\n"
			}
			workspaceGitShim(t, "for arg do\n if [ \"$arg\" = push ]; then\n  \"$WORKSPACE_TEST_REAL_GIT\" \"$@\" || exit $?\n"+drop+"  \"$WORKSPACE_TEST_REAL_GIT\" -C \"$WORKSPACE_TEST_LINKED\" stash push -u -m other-worktree || exit $?\n  exit 0\n fi\ndone")
			r := openTest(t, dir)
			err = r.SaveWorkspaceStash(context.Background())
			if removeOwn {
				var action *WorkspaceActionError
				if !errors.As(err, &action) || !action.Partial || action.StashOID != "" {
					t.Fatalf("claimed another worktree stash: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			entries, e := r.Stashes(context.Background())
			if e != nil {
				t.Fatal(e)
			}
			if len(entries) == 0 || !strings.Contains(entries[0].Message, "other-worktree") {
				t.Fatal("fixture did not replace latest stash")
			}
			if !removeOwn && (len(entries) != 2 || !strings.Contains(entries[1].Message, "SuperGit workspace")) {
				t.Fatal("own stash was lost")
			}
		})
	}
}

func TestWorkspaceStashPartialSaveRetainsOID(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, "base.txt", "saved bytes")
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("WORKSPACE_TEST_REAL_GIT", realGit)
	workspaceGitShim(t, "for arg do\n if [ \"$arg\" = push ]; then\n  \"$WORKSPACE_TEST_REAL_GIT\" \"$@\" || exit $?\n  exit 73\n fi\ndone")
	r := openTest(t, dir)
	err = r.SaveWorkspaceStash(context.Background())
	var action *WorkspaceActionError
	if !errors.As(err, &action) || !action.Partial || action.StashOID == "" {
		t.Fatalf("missing saved stash identity: %v", err)
	}
	entries, e := r.Stashes(context.Background())
	if e != nil || len(entries) != 1 || entries[0].SHA != action.StashOID {
		t.Fatalf("lost saved entry: %v %v", entries, e)
	}
	git(t, dir, "stash", "apply", action.StashOID)
	requireWorkspaceBytes(t, dir, "base.txt", "saved bytes")
}

func TestWorkspaceStashIgnoredDirectoryAndRuleExposure(t *testing.T) {
	for _, kind := range []string{"directory", "exposure"} {
		t.Run(kind, func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			write(t, dir, ".gitignore", "*.keep\n")
			if kind == "directory" {
				git(t, dir, "add", ".gitignore")
				git(t, dir, "commit", "-m", "ignore")
				git(t, dir, "rm", "base.txt")
				write(t, dir, "base.txt/file.keep", "ignored")
			} else {
				write(t, dir, "file.keep", "ignored")
			}
			before := workspaceFingerprint(t, dir)
			if err := openTest(t, dir).SaveWorkspaceStash(context.Background()); err == nil {
				t.Fatal("accepted ignored collision/exposure")
			}
			if !reflect.DeepEqual(before, workspaceFingerprint(t, dir)) {
				t.Fatal("unsafe stash wrote repository")
			}
		})
	}
}

func TestWorkspaceStashRefusesIgnoredCollision(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "rm", "base.txt")
	write(t, dir, ".git/info/exclude", "base.txt\n")
	write(t, dir, "base.txt", "ignored replacement")
	before := git(t, dir, "ls-files", "--stage", "-z")
	if err := openTest(t, dir).SaveWorkspaceStash(context.Background()); err == nil {
		t.Fatal("accepted ignored collision")
	}
	requireWorkspaceBytes(t, dir, "base.txt", "ignored replacement")
	if git(t, dir, "ls-files", "--stage", "-z") != before || git(t, dir, "stash", "list") != "" {
		t.Fatal("refused stash wrote index/ref")
	}
}
