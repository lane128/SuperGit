package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

func workspaceMixed(t *testing.T) (string, *Repository) {
	t.Helper()
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, ".gitignore", "*.keep\nignored/\n")
	write(t, dir, "old", "old content")
	write(t, dir, "gone", "gone content")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-m", "baseline")
	write(t, dir, "base.txt", "staged\n")
	git(t, dir, "add", "base.txt")
	write(t, dir, "base.txt", "unstaged\n")
	git(t, dir, "mv", "old", "renamed")
	git(t, dir, "rm", "gone")
	write(t, dir, "added", "staged addition")
	git(t, dir, "add", "added")
	write(t, dir, "added", "partially staged addition")
	write(t, dir, "intent", "intent")
	git(t, dir, "add", "-N", "intent")
	for _, name := range []string{"new/deep/file", "空 格", "line\nbreak", "-option", ":(glob)literal"} {
		write(t, dir, name, "new bytes")
	}
	write(t, dir, "binary", "\x00\xff\x80binary\x00")
	write(t, dir, "new/preserve.keep", "ignored bytes")
	if err := os.Mkdir(filepath.Join(dir, "empty"), 0755); err != nil {
		t.Fatal(err)
	}
	return dir, openTest(t, dir)
}

func requireWorkspaceBytes(t *testing.T, dir, name, want string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil || string(b) != want {
		t.Fatalf("%q = %q, %v; want %q", name, b, err, want)
	}
}

func TestWorkspaceCleanMixed(t *testing.T) {
	for _, deleteNew := range []bool{false, true} {
		t.Run(map[bool]string{false: "preserve", true: "delete"}[deleteNew], func(t *testing.T) {
			dir, r := workspaceMixed(t)
			outside := t.TempDir()
			write(t, outside, "target", "external")
			if err := os.Symlink(filepath.Join(outside, "target"), filepath.Join(dir, "link")); err != nil {
				t.Fatal(err)
			}
			head := git(t, dir, "rev-parse", "HEAD")
			branch := git(t, dir, "symbolic-ref", "HEAD")
			p, err := r.PrepareWorkspaceClean(context.Background(), deleteNew)
			if err != nil {
				t.Fatal(err)
			}
			if !p.CanExecute || p.Head != strings.TrimSpace(head) || !slices.Contains(p.RestorePaths, "old") || !slices.Contains(p.NewPaths, "renamed") {
				t.Fatalf("bad preview: %+v", p)
			}
			if err := r.CleanWorkspaceChecked(context.Background(), p); err != nil {
				t.Fatal(err)
			}
			requireWorkspaceBytes(t, dir, "base.txt", "original\n")
			requireWorkspaceBytes(t, dir, "old", "old content")
			requireWorkspaceBytes(t, dir, "gone", "gone content")
			requireWorkspaceBytes(t, dir, "new/preserve.keep", "ignored bytes")
			requireWorkspaceBytes(t, outside, "target", "external")
			if got := git(t, dir, "diff", "--cached", "--name-only"); got != "" {
				t.Fatalf("index dirty: %s", got)
			}
			if git(t, dir, "rev-parse", "HEAD") != head || git(t, dir, "symbolic-ref", "HEAD") != branch {
				t.Fatal("moved HEAD/branch")
			}
			if deleteNew {
				if s := git(t, dir, "status", "--porcelain"); s != "" {
					t.Fatalf("dirty: %s", s)
				}
				if _, err := os.Stat(filepath.Join(dir, "empty")); !os.IsNotExist(err) {
					t.Fatal("empty directory survived")
				}
			} else {
				requireWorkspaceBytes(t, dir, "added", "partially staged addition")
				requireWorkspaceBytes(t, dir, "renamed", "old content")
				requireWorkspaceBytes(t, dir, "intent", "intent")
			}
		})
	}
}

func TestWorkspaceCleanTrackedFileInsideIgnoredDirectory(t *testing.T) {
	for _, deleteNew := range []bool{false, true} {
		t.Run(fmt.Sprint(deleteNew), func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			write(t, dir, "vendor/config.txt", "original")
			git(t, dir, "add", ".")
			git(t, dir, "commit", "-m", "track config")
			write(t, dir, ".gitignore", "vendor/\n")
			git(t, dir, "add", ".gitignore")
			git(t, dir, "commit", "-m", "ignore vendor")
			write(t, dir, "vendor/config.txt", "edited")
			write(t, dir, "vendor/private.txt", "preserve ignored bytes")
			r := openTest(t, dir)
			p, err := r.PrepareWorkspaceClean(context.Background(), deleteNew)
			if err != nil {
				t.Fatal(err)
			}
			if !p.CanExecute || !slices.Contains(p.RestorePaths, "vendor/config.txt") {
				t.Fatalf("missing tracked restoration: %+v", p)
			}
			if err := r.CleanWorkspaceChecked(context.Background(), p); err != nil {
				t.Fatal(err)
			}
			requireWorkspaceBytes(t, dir, "vendor/config.txt", "original")
			requireWorkspaceBytes(t, dir, "vendor/private.txt", "preserve ignored bytes")
		})
	}
}

func TestWorkspaceCleanReplacement(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		for _, ignored := range []bool{false, true} {
			for _, del := range []bool{false, true} {
				t.Run(strings.Join([]string{map[bool]string{false: "file", true: "directory"}[reverse], map[bool]string{false: "new", true: "ignored"}[ignored], map[bool]string{false: "keep", true: "delete"}[del]}, "/"), func(t *testing.T) {
					dir := fixture(t)
					seed(t, dir)
					write(t, dir, ".gitignore", "*.keep\n")
					name := "p"
					if reverse {
						name = "p/old"
					}
					write(t, dir, name, "original")
					git(t, dir, "add", ".")
					git(t, dir, "commit", "-m", "tree")
					git(t, dir, "rm", name)
					replacement := "p/new"
					if ignored {
						replacement = "p/new.keep"
					}
					if reverse {
						replacement = "p"
						if ignored {
							write(t, dir, ".git/info/exclude", "/p\n")
						}
					}
					write(t, dir, replacement, "replacement")
					before := git(t, dir, "ls-files", "--stage")
					r := openTest(t, dir)
					p, err := r.PrepareWorkspaceClean(context.Background(), del)
					if !del || ignored {
						if err == nil {
							t.Fatal("accepted protected obstruction")
						}
						requireWorkspaceBytes(t, dir, replacement, "replacement")
						if git(t, dir, "ls-files", "--stage") != before {
							t.Fatal("index changed")
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					if err = r.CleanWorkspaceChecked(context.Background(), p); err != nil {
						t.Fatal(err)
					}
					requireWorkspaceBytes(t, dir, name, "original")
				})
			}
		}
	}
}

// Independently capture bytes, modes, symlink destinations, index and refs;
// rejected operations must leave all of these unchanged.
func workspaceFingerprint(t *testing.T, dir string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(dir, func(full string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name, _ := filepath.Rel(dir, full)
		if d.Name() == ".git" {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		value := fmt.Sprint(info.Mode())
		if info.Mode().IsRegular() {
			b, err := os.ReadFile(full)
			if err != nil {
				return err
			}
			value += string(b)
		} else if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(full)
			if err != nil {
				return err
			}
			value += link
		}
		result[name] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"rev-parse", "--verify", "HEAD"}, {"rev-parse", "--symbolic-full-name", "HEAD"}, {"ls-files", "--stage", "-z"}, {"stash", "list", "--format=%H"}} {
		result[strings.Join(args, " ")] = git(t, dir, args...)
	}
	indexPath := strings.TrimSpace(git(t, dir, "rev-parse", "--path-format=absolute", "--git-path", "index"))
	b, err := os.ReadFile(indexPath)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	result["index bytes"] = string(b)
	return result
}

func TestWorkspaceCleanRejectsStaleSnapshotWithoutWrites(t *testing.T) {
	changes := map[string]func(*testing.T, string){
		"bytes":            func(t *testing.T, d string) { write(t, d, "base.txt", "newer bytes") },
		"index":            func(t *testing.T, d string) { git(t, d, "add", "base.txt") },
		"same-head-branch": func(t *testing.T, d string) { git(t, d, "checkout", "-b", "other") },
		"head":             func(t *testing.T, d string) { git(t, d, "commit", "--allow-empty", "-m", "advance") },
		"ignore":           func(t *testing.T, d string) { write(t, d, ".git/info/exclude", "irrelevant-pattern\n") },
		"child":            func(t *testing.T, d string) { write(t, d, "new/arrived", "keep") },
		"missing": func(t *testing.T, d string) {
			if err := os.Remove(filepath.Join(d, "new/file")); err != nil {
				t.Fatal(err)
			}
		},
		"inode": func(t *testing.T, d string) {
			if err := os.Rename(filepath.Join(d, "new/file"), filepath.Join(t.TempDir(), "old")); err != nil {
				t.Fatal(err)
			}
			write(t, d, "new/file", "pending")
		},
		"parent": func(t *testing.T, d string) {
			if err := os.Rename(filepath.Join(d, "new"), filepath.Join(d, "moved")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("moved", filepath.Join(d, "new")); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			write(t, dir, "base.txt", "pending")
			write(t, dir, "new/file", "pending")
			r := openTest(t, dir)
			p, err := r.PrepareWorkspaceClean(context.Background(), true)
			if err != nil {
				t.Fatal(err)
			}
			change(t, dir)
			before := workspaceFingerprint(t, dir)
			if err = r.CleanWorkspaceChecked(context.Background(), p); err == nil {
				t.Fatal("accepted stale preview")
			}
			if !reflect.DeepEqual(before, workspaceFingerprint(t, dir)) {
				t.Fatal("rejected cleanup wrote repository")
			}
		})
	}
}

func TestWorkspaceCleanIgnoreTransitions(t *testing.T) {
	for _, kind := range []string{"restore-rule", "expose-ignored", "nested-rule", "global-exclude"} {
		t.Run(kind, func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			switch kind {
			case "restore-rule":
				write(t, dir, ".gitignore", "*.keep\n")
				git(t, dir, "add", ".")
				git(t, dir, "commit", "-m", "ignore")
				write(t, dir, ".gitignore", "")
			case "expose-ignored":
				write(t, dir, ".gitignore", "*.keep\n")
			case "nested-rule":
				write(t, dir, "new/.gitignore", "*.keep\n")
				git(t, dir, "add", ".")
				git(t, dir, "commit", "-m", "ignore")
				write(t, dir, "new/.gitignore", "")
			case "global-exclude":
				exclude := filepath.Join(t.TempDir(), "ignore")
				write(t, filepath.Dir(exclude), "ignore", "*.keep\n")
				git(t, dir, "config", "core.excludesFile", exclude)
			}
			write(t, dir, "new/file.keep", "preserve")
			write(t, dir, "new/remove", "remove")
			r := openTest(t, dir)
			before := workspaceFingerprint(t, dir)
			p, err := r.PrepareWorkspaceClean(context.Background(), true)
			if kind == "expose-ignored" {
				if err == nil {
					t.Fatal("accepted exposure of ignored file")
				}
				if !reflect.DeepEqual(before, workspaceFingerprint(t, dir)) {
					t.Fatal("preflight wrote repository")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if err = r.CleanWorkspaceChecked(context.Background(), p); err != nil {
				t.Fatal(err)
			}
			requireWorkspaceBytes(t, dir, "new/file.keep", "preserve")
			if git(t, dir, "status", "--porcelain") != "" {
				t.Fatal("not clean")
			}
		})
	}
}

func TestWorkspaceCleanRefusesUnsupportedStates(t *testing.T) {
	cases := map[string]func(*testing.T, string){
		"sparse":           func(t *testing.T, d string) { git(t, d, "config", "core.sparseCheckout", "true") },
		"skip-worktree":    func(t *testing.T, d string) { git(t, d, "update-index", "--skip-worktree", "base.txt") },
		"assume-unchanged": func(t *testing.T, d string) { git(t, d, "update-index", "--assume-unchanged", "base.txt") },
		"nested":           func(t *testing.T, d string) { git(t, d, "init", filepath.Join(d, "nested")) },
		"bare":             func(t *testing.T, d string) { git(t, d, "init", "--bare", filepath.Join(d, "bare")) },
	}
	for _, name := range []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD", "rebase-merge/head-name", "rebase-apply/applying", "sequencer/todo"} {
		cases[name] = func(t *testing.T, d string) {
			write(t, d, ".git/"+name, strings.TrimSpace(git(t, d, "rev-parse", "HEAD"))+"\n")
		}
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			write(t, dir, "base.txt", "pending")
			setup(t, dir)
			r := openTest(t, dir)
			before := workspaceFingerprint(t, dir)
			if _, err := r.PrepareWorkspaceClean(context.Background(), true); err == nil {
				t.Fatal("accepted unsupported clean")
			}
			if err := r.SaveWorkspaceStash(context.Background()); err == nil {
				t.Fatal("accepted unsupported stash")
			}
			if !reflect.DeepEqual(before, workspaceFingerprint(t, dir)) {
				t.Fatal("unsupported action wrote repository")
			}
		})
	}
}

func TestWorkspaceCleanRefusesConflictAndSpecialFile(t *testing.T) {
	for _, kind := range []string{"conflict", "fifo"} {
		t.Run(kind, func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			if kind == "fifo" {
				if err := syscall.Mkfifo(filepath.Join(dir, "pipe"), 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				git(t, dir, "checkout", "-b", "other")
				write(t, dir, "base.txt", "theirs")
				git(t, dir, "commit", "-am", "theirs")
				git(t, dir, "checkout", "main")
				write(t, dir, "base.txt", "ours")
				git(t, dir, "commit", "-am", "ours")
				if _, err := run(context.Background(), dir, "", "merge", "other"); err == nil {
					t.Fatal("fixture did not conflict")
				}
			}
			before := workspaceFingerprint(t, dir)
			r := openTest(t, dir)
			if _, err := r.PrepareWorkspaceClean(context.Background(), true); err == nil {
				t.Fatal("accepted unsupported state")
			}
			if err := r.SaveWorkspaceStash(context.Background()); err == nil {
				t.Fatal("stashed unsupported state")
			}
			if !reflect.DeepEqual(before, workspaceFingerprint(t, dir)) {
				t.Fatal("preflight mutated repository")
			}
		})
	}
}

func TestWorkspaceCleanSubmoduleBoundaries(t *testing.T) {
	for _, kind := range []string{"clean", "dirty", "gitlink", "new-gitlink"} {
		t.Run(kind, func(t *testing.T) {
			source := fixture(t)
			seed(t, source)
			write(t, source, "second", "second")
			git(t, source, "add", ".")
			git(t, source, "commit", "-m", "second")
			dir := fixture(t)
			seed(t, dir)
			git(t, dir, "-c", "protocol.file.allow=always", "submodule", "add", source, "module")
			if kind != "new-gitlink" {
				git(t, dir, "commit", "-am", "module")
			}
			write(t, dir, "base.txt", "pending")
			switch kind {
			case "dirty":
				write(t, dir, "module/base.txt", "inside changes")
				git(t, dir, "config", "submodule.module.ignore", "all")
			case "gitlink":
				git(t, filepath.Join(dir, "module"), "checkout", "HEAD~1")
				git(t, dir, "add", "module")
			}
			r := openTest(t, dir)
			before := workspaceFingerprint(t, dir)
			p, err := r.PrepareWorkspaceClean(context.Background(), true)
			if kind == "clean" {
				if err != nil {
					t.Fatal(err)
				}
				if err = r.CleanWorkspaceChecked(context.Background(), p); err != nil {
					t.Fatal(err)
				}
				requireWorkspaceBytes(t, dir, "module/base.txt", "original\n")
				write(t, dir, "base.txt", "stash pending")
				if err = r.SaveWorkspaceStash(context.Background()); err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("accepted submodule change")
			}
			if err = r.SaveWorkspaceStash(context.Background()); err == nil {
				t.Fatal("stashed submodule change")
			}
			if !reflect.DeepEqual(before, workspaceFingerprint(t, dir)) {
				t.Fatal("submodule rejection wrote files")
			}
		})
	}
}

func TestWorkspaceCleanDetachedAndLinkedWorktree(t *testing.T) {
	for _, kind := range []string{"detached", "linked"} {
		t.Run(kind, func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			if kind == "detached" {
				git(t, dir, "checkout", "--detach")
			} else {
				linked := filepath.Join(t.TempDir(), "linked")
				git(t, dir, "worktree", "add", "-b", "linked", linked)
				dir = linked
			}
			head := git(t, dir, "rev-parse", "HEAD")
			branch := git(t, dir, "rev-parse", "--symbolic-full-name", "HEAD")
			write(t, dir, "base.txt", "pending")
			write(t, dir, "new", "pending")
			r := openTest(t, dir)
			p, err := r.PrepareWorkspaceClean(context.Background(), true)
			if err != nil {
				t.Fatal(err)
			}
			if err = r.CleanWorkspaceChecked(context.Background(), p); err != nil {
				t.Fatal(err)
			}
			if git(t, dir, "rev-parse", "HEAD") != head || git(t, dir, "rev-parse", "--symbolic-full-name", "HEAD") != branch {
				t.Fatal("identity changed")
			}
			write(t, dir, "base.txt", "stash pending")
			if err = r.SaveWorkspaceStash(context.Background()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestWorkspaceCleanRechecksDeletionAfterRestore(t *testing.T) {
	for _, kind := range []string{"bytes", "child", "parent"} {
		t.Run(kind, func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			write(t, dir, "base.txt", "pending")
			write(t, dir, "new/file", "pending")
			r := openTest(t, dir)
			p, err := r.PrepareWorkspaceClean(context.Background(), true)
			if err != nil {
				t.Fatal(err)
			}
			realGit, err := exec.LookPath("git")
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("WORKSPACE_TEST_REAL_GIT", realGit)
			t.Setenv("WORKSPACE_TEST_ROOT", dir)
			mutate := map[string]string{"bytes": "printf newer > \"$WORKSPACE_TEST_ROOT/new/file\"", "child": "printf keep > \"$WORKSPACE_TEST_ROOT/new/arrived\"", "parent": "mv \"$WORKSPACE_TEST_ROOT/new\" \"$WORKSPACE_TEST_ROOT/moved\"\nln -s moved \"$WORKSPACE_TEST_ROOT/new\""}[kind]
			workspaceGitShim(t, "for arg do\n if [ \"$arg\" = restore ]; then\n  \"$WORKSPACE_TEST_REAL_GIT\" \"$@\" || exit $?\n"+mutate+"\n  exit 0\n fi\ndone")
			err = r.CleanWorkspaceChecked(context.Background(), p)
			var action *WorkspaceActionError
			if !errors.As(err, &action) || action.Phase != "delete" || !action.Partial {
				t.Fatalf("want delete failure: %v", err)
			}
			switch kind {
			case "bytes":
				requireWorkspaceBytes(t, dir, "new/file", "newer")
			case "child":
				requireWorkspaceBytes(t, dir, "new/arrived", "keep")
			case "parent":
				requireWorkspaceBytes(t, dir, "moved/file", "pending")
			}
		})
	}
}

func TestWorkspaceCleanRefusesUnborn(t *testing.T) {
	dir := fixture(t)
	write(t, dir, "new", "keep")
	git(t, dir, "add", "new")
	before := git(t, dir, "ls-files", "--stage")
	r := openTest(t, dir)
	if _, err := r.PrepareWorkspaceClean(context.Background(), true); err == nil {
		t.Fatal("accepted unborn")
	}
	if err := r.SaveWorkspaceStash(context.Background()); err == nil {
		t.Fatal("stashed unborn")
	}
	if git(t, dir, "ls-files", "--stage") != before {
		t.Fatal("index changed")
	}
	requireWorkspaceBytes(t, dir, "new", "keep")
}

func TestWorkspaceCleanRejectsEditedPreviewAndCancellation(t *testing.T) {
	for _, kind := range []string{"option", "paths", "cancel", "other-repo"} {
		t.Run(kind, func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			write(t, dir, "base.txt", "pending")
			write(t, dir, "new", "keep")
			r := openTest(t, dir)
			p, err := r.PrepareWorkspaceClean(context.Background(), true)
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			switch kind {
			case "option":
				p.DeleteNew = false
			case "paths":
				p.NewPaths[0] = "base.txt"
			case "cancel":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "other-repo":
				other := fixture(t)
				seed(t, other)
				r = openTest(t, other)
			}
			before := workspaceFingerprint(t, dir)
			if err = r.CleanWorkspaceChecked(ctx, p); err == nil {
				t.Fatal("accepted invalid confirmation")
			}
			if !reflect.DeepEqual(before, workspaceFingerprint(t, dir)) {
				t.Fatal("invalid confirmation wrote files")
			}
		})
	}
}

// A command-boundary shim exercises a real successful read-tree followed by a
// restore failure. It changes no production code and runs only inside fixtures.
func workspaceGitShim(t *testing.T, body string) {
	t.Helper()
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	script := "#!/bin/sh\n" + body + "\nexec \"" + realGit + "\" \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestWorkspaceCleanRechecksBeforeRestoringAndDeleting(t *testing.T) {
	for _, target := range []string{"base.txt", "new/file"} {
		t.Run(target, func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			write(t, dir, "base.txt", "pending")
			write(t, dir, "new/file", "pending")
			r := openTest(t, dir)
			p, err := r.PrepareWorkspaceClean(context.Background(), true)
			if err != nil {
				t.Fatal(err)
			}
			realGit, err := exec.LookPath("git")
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("WORKSPACE_TEST_REAL_GIT", realGit)
			t.Setenv("WORKSPACE_TEST_TARGET", filepath.Join(dir, target))
			workspaceGitShim(t, "for arg do\n if [ \"$arg\" = read-tree ]; then\n  \"$WORKSPACE_TEST_REAL_GIT\" \"$@\" || exit $?\n  printf newer > \"$WORKSPACE_TEST_TARGET\"\n  exit 0\n fi\ndone")
			err = r.CleanWorkspaceChecked(context.Background(), p)
			var action *WorkspaceActionError
			if !errors.As(err, &action) || !action.Partial {
				t.Fatalf("did not report concurrent mutation: %v", err)
			}
			requireWorkspaceBytes(t, dir, target, "newer")
		})
	}
}

func TestWorkspaceCleanAndStashCancellationAfterMutation(t *testing.T) {
	for _, kind := range []string{"clean", "stash"} {
		t.Run(kind, func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			write(t, dir, "base.txt", "saved bytes")
			git(t, dir, "add", "base.txt")
			r := openTest(t, dir)
			p, err := r.PrepareWorkspaceClean(context.Background(), true)
			if err != nil {
				t.Fatal(err)
			}
			realGit, err := exec.LookPath("git")
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("WORKSPACE_TEST_REAL_GIT", realGit)
			signal := filepath.Join(t.TempDir(), "started")
			t.Setenv("WORKSPACE_TEST_SIGNAL", signal)
			command := "read-tree"
			if kind == "stash" {
				command = "push"
			}
			workspaceGitShim(t, "for arg do\n if [ \"$arg\" = "+command+" ]; then\n  \"$WORKSPACE_TEST_REAL_GIT\" \"$@\" || exit $?\n  printf started > \"$WORKSPACE_TEST_SIGNAL\"\n  exec /bin/sleep 30\n fi\ndone")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() {
				if kind == "clean" {
					done <- r.CleanWorkspaceChecked(ctx, p)
				} else {
					done <- r.SaveWorkspaceStash(ctx)
				}
			}()
			ticker := time.NewTicker(5 * time.Millisecond)
			defer ticker.Stop()
			timeout := time.NewTimer(5 * time.Second)
			defer timeout.Stop()
		wait:
			for {
				select {
				case <-ticker.C:
					if _, err := os.Stat(signal); err == nil {
						cancel()
						break wait
					}
				case <-timeout.C:
					cancel()
					t.Fatal("command did not reach mutation boundary")
				case err := <-done:
					t.Fatalf("command ended before cancellation: %v", err)
				}
			}
			err = <-done
			var action *WorkspaceActionError
			if !errors.Is(err, context.Canceled) || !errors.As(err, &action) || !action.Partial {
				t.Fatalf("missing cancellation/partial warning: %v", err)
			}
			if kind == "stash" {
				if action.StashOID == "" {
					t.Fatal("saved stash not identified after cancellation")
				}
				git(t, dir, "stash", "apply", action.StashOID)
			}
			requireWorkspaceBytes(t, dir, "base.txt", "saved bytes")
		})
	}
}

func TestWorkspaceCleanReportsRestoreFailure(t *testing.T) {
	dir, r := workspaceMixed(t)
	p, err := r.PrepareWorkspaceClean(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	workspaceGitShim(t, "for arg do\n if [ \"$arg\" = restore ]; then exit 71; fi\ndone")
	err = r.CleanWorkspaceChecked(context.Background(), p)
	var failure *WorkspaceActionError
	if !errors.As(err, &failure) || failure.Phase != "restore" || !failure.Partial {
		t.Fatalf("expected partial restore error: %v", err)
	}
	if git(t, dir, "diff", "--cached", "--name-only") != "" {
		t.Fatal("index stage did not run")
	}
	requireWorkspaceBytes(t, dir, "base.txt", "unstaged\n")
	requireWorkspaceBytes(t, dir, "added", "partially staged addition")
}

func TestWorkspaceCleanPreviewsMissingStagedAddition(t *testing.T) {
	for _, deleteNew := range []bool{false, true} {
		t.Run(fmt.Sprint(deleteNew), func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			write(t, dir, "added", "index-only bytes")
			git(t, dir, "add", "added")
			if err := os.Remove(filepath.Join(dir, "added")); err != nil {
				t.Fatal(err)
			}
			r := openTest(t, dir)
			p, err := r.PrepareWorkspaceClean(context.Background(), deleteNew)
			if err != nil {
				t.Fatal(err)
			}
			if !p.CanExecute || !slices.Equal(p.NewPaths, []string{"added"}) || !slices.Equal(p.IndexOnlyPaths, []string{"added"}) {
				t.Fatalf("index-only loss not disclosed exactly once: %+v", p)
			}
			tampered := p
			tampered.IndexOnlyPaths = nil
			before := workspaceFingerprint(t, dir)
			if err = r.CleanWorkspaceChecked(context.Background(), tampered); err == nil {
				t.Fatal("accepted changed loss disclosure")
			}
			if !reflect.DeepEqual(before, workspaceFingerprint(t, dir)) {
				t.Fatal("invalid preview wrote repository")
			}
			if err = r.CleanWorkspaceChecked(context.Background(), p); err != nil {
				t.Fatal(err)
			}
			if git(t, dir, "status", "--porcelain") != "" {
				t.Fatal("index addition survived")
			}
			if _, err := os.Lstat(filepath.Join(dir, "added")); !os.IsNotExist(err) {
				t.Fatalf("missing working file unexpectedly recreated: %v", err)
			}
		})
	}
}

func TestWorkspaceCleanEligibility(t *testing.T) {
	for _, kind := range []string{"untracked", "staged", "intent"} {
		t.Run(kind, func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			write(t, dir, "new", "bytes")
			if kind == "staged" {
				git(t, dir, "add", "new")
			}
			if kind == "intent" {
				git(t, dir, "add", "-N", "new")
			}
			r := openTest(t, dir)
			p, err := r.PrepareWorkspaceClean(context.Background(), false)
			if err != nil {
				t.Fatal(err)
			}
			if p.CanExecute != (kind != "untracked") {
				t.Fatalf("eligibility: %+v", p)
			}
			if p.CanExecute {
				if err = r.CleanWorkspaceChecked(context.Background(), p); err != nil {
					t.Fatal(err)
				}
				requireWorkspaceBytes(t, dir, "new", "bytes")
				if git(t, dir, "diff", "--cached", "--name-only") != "" {
					t.Fatal("staged content remains")
				}
			}
		})
	}
}
