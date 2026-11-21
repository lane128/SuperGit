package gitrepo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func git(t testing.TB, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %q: %v\n%s", args, err, out)
	}
	return string(out)
}
func fixture(t testing.TB) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	dir := t.TempDir()
	git(t, dir, "init", "-b", "main")
	git(t, dir, "config", "user.name", "Test Developer")
	git(t, dir, "config", "user.email", "test@example.invalid")
	return dir
}
func write(t testing.TB, dir, path, text string) {
	t.Helper()
	full := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
}
func seed(t testing.TB, dir string) {
	t.Helper()
	write(t, dir, "base.txt", "original\n")
	git(t, dir, "add", "--", "base.txt")
	git(t, dir, "commit", "-m", "initial")
}
func openTest(t testing.TB, dir string) *Repository {
	t.Helper()
	r, e := Open(context.Background(), dir)
	if e != nil {
		t.Fatal(e)
	}
	return r
}

func TestStatusPreservesSpecialPathsAndPartialStaging(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	names := []string{"空 格.txt", "line\nbreak.txt", "-option.txt"}
	for _, name := range names {
		write(t, dir, name, "hello\n")
	}
	r := openTest(t, dir)
	ctx := context.Background()
	if err := r.Stage(ctx, names[0]); err != nil {
		t.Fatal(err)
	}
	write(t, dir, names[0], "hello\nmore\n")
	status, err := r.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.Branch != "main" {
		t.Fatalf("branch=%q", status.Branch)
	}
	got := map[string]Entry{}
	for _, e := range status.Entries {
		got[e.Path] = e
	}
	for _, name := range names {
		if _, ok := got[name]; !ok {
			t.Errorf("missing exact path %q: %#v", name, status.Entries)
		}
	}
	if !got[names[0]].Staged() || !got[names[0]].Unstaged() {
		t.Fatal("partial staging must appear in both groups")
	}
	if err := r.Stage(ctx, "-option.txt"); err != nil {
		t.Fatal(err)
	}
}
func TestUnstageBeforeFirstCommitPreservesFile(t *testing.T) {
	dir := fixture(t)
	write(t, dir, "new.txt", "keep me\n")
	r := openTest(t, dir)
	ctx := context.Background()
	if err := r.Stage(ctx, "new.txt"); err != nil {
		t.Fatal(err)
	}
	if err := r.Unstage(ctx, "new.txt"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "new.txt"))
	if err != nil || string(b) != "keep me\n" {
		t.Fatalf("working file lost: %q %v", b, err)
	}
	if out := git(t, dir, "ls-files"); out != "" {
		t.Fatalf("index not empty: %q", out)
	}
	history, err := r.History(ctx, 20)
	if err != nil || len(history) != 0 {
		t.Fatalf("unborn history: %v %v", history, err)
	}
}
func TestCommitIncludesOnlyIndexAndKeepsOtherChanges(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, "base.txt", "unstaged\n")
	write(t, dir, "selected.txt", "selected\n")
	r := openTest(t, dir)
	ctx := context.Background()
	if err := r.Stage(ctx, "selected.txt"); err != nil {
		t.Fatal(err)
	}
	if err := r.Commit(ctx, "selected only\n\nDetailed body"); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "show", "HEAD:base.txt"); got != "original\n" {
		t.Fatalf("unstaged content committed: %q", got)
	}
	if got := git(t, dir, "show", "HEAD:selected.txt"); got != "selected\n" {
		t.Fatal(got)
	}
	status, err := r.Status(ctx)
	if err != nil || len(status.Entries) != 1 || status.Entries[0].Path != "base.txt" {
		t.Fatalf("remaining work: %#v %v", status, err)
	}
	// Empty commit messages are allowed when staged changes exist.
	write(t, dir, "empty-msg.txt", "ok\n")
	if err := r.Stage(ctx, "empty-msg.txt"); err != nil {
		t.Fatal(err)
	}
	if err := r.Commit(ctx, "  \n"); err != nil {
		t.Fatalf("empty message rejected: %v", err)
	}
	if got := git(t, dir, "log", "-1", "--format=%B"); strings.TrimSpace(got) != "" {
		t.Fatalf("empty commit message = %q; want empty", got)
	}
}
func TestCommitHookFailureIsReportedAndIndexPreserved(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, "selected.txt", "keep staged\n")
	r := openTest(t, dir)
	ctx := context.Background()
	if err := r.Stage(ctx, "selected.txt"); err != nil {
		t.Fatal(err)
	}
	hook := filepath.Join(dir, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nprintf 'hook rejected test commit\\n' >&2\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}
	err := r.Commit(ctx, "blocked")
	if err == nil || !strings.Contains(err.Error(), "hook rejected test commit") {
		t.Fatalf("hook error missing: %v", err)
	}
	if got := git(t, dir, "diff", "--cached", "--name-only"); got != "selected.txt\n" {
		t.Fatalf("index changed: %q", got)
	}
}
func TestRenameKeepsOriginalAndDestinationPaths(t *testing.T) {
	dir := fixture(t)
	write(t, dir, "old name.txt", "rename content\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-m", "initial")
	git(t, dir, "mv", "old name.txt", "新\nname.txt")
	r := openTest(t, dir)
	status, err := r.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Entries) != 1 {
		t.Fatalf("entries=%#v", status.Entries)
	}
	e := status.Entries[0]
	if e.Path != "新\nname.txt" || e.OriginalPath != "old name.txt" {
		t.Fatalf("rename=%#v", e)
	}
}
func TestHunkStageAndReverseDoNotChangeWorktree(t *testing.T) {
	dir := fixture(t)
	lines := make([]string, 40)
	for i := range lines {
		lines[i] = "unchanged line"
	}
	write(t, dir, "many.txt", strings.Join(lines, "\n")+"\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-m", "initial")
	lines[1] = "first edit"
	lines[35] = "second edit"
	modified := strings.Join(lines, "\n") + "\n"
	write(t, dir, "many.txt", modified)
	r := openTest(t, dir)
	ctx := context.Background()
	d, err := r.Diff(ctx, "many.txt", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Hunks) != 2 {
		t.Fatalf("hunks=%d\n%s", len(d.Hunks), d.Text)
	}
	if err := r.ApplyHunk(ctx, d.Hunks[0].Patch, false); err != nil {
		t.Fatal(err)
	}
	cached := git(t, dir, "diff", "--cached")
	if !strings.Contains(cached, "first edit") || strings.Contains(cached, "second edit") {
		t.Fatalf("wrong hunk staged: %s", cached)
	}
	staged, err := r.Diff(ctx, "many.txt", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ApplyHunk(ctx, staged.Hunks[0].Patch, true); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "diff", "--cached"); got != "" {
		t.Fatalf("unstage failed: %s", got)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "many.txt"))
	if string(b) != modified {
		t.Fatal("working file modified")
	}
}
func TestUntrackedDiffIsBoundedAndDoesNotFollowSymlink(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, "new file.txt", "new content\n")
	r := openTest(t, dir)
	ctx := context.Background()
	d, err := r.Diff(ctx, "new file.txt", false)
	if err != nil || !strings.Contains(d.Text, "new content") {
		t.Fatalf("untracked diff=%q %v", d.Text, err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	os.WriteFile(outside, []byte("private outside body"), 0600)
	if err := os.Symlink(outside, filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	d, err = r.Diff(ctx, "link", false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(d.Text, "private outside body") {
		t.Fatal("symlink target read")
	}
	if _, err := r.Diff(ctx, "../outside", false); err == nil {
		t.Fatal("outside repository path accepted")
	}
}
func TestHistoryExposesParentsFilesAndDiff(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "checkout", "-b", "feature")
	write(t, dir, "feature.txt", "feature\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-m", "feature")
	git(t, dir, "checkout", "main")
	write(t, dir, "main.txt", "main\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-m", "main change")
	git(t, dir, "merge", "--no-ff", "feature", "-m", "merge feature")
	r := openTest(t, dir)
	ctx := context.Background()
	history, err := r.History(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 4 || len(history[0].Parents) != 2 {
		t.Fatalf("history=%#v", history)
	}
	files, err := r.CommitFiles(ctx, history[0].SHA)
	if err != nil || len(files) != 1 || files[0].Path != "feature.txt" {
		t.Fatalf("files=%#v %v", files, err)
	}
	d, err := r.Show(ctx, history[0].SHA, "feature.txt")
	if err != nil || !strings.Contains(d.Text, "+feature") {
		t.Fatalf("show=%q %v", d.Text, err)
	}
	branches, err := r.Branches(ctx)
	if err != nil || len(branches) != 2 {
		t.Fatalf("branches=%#v %v", branches, err)
	}
}
func TestWorktreesAndSubmodulePointers(t *testing.T) {
	child := fixture(t)
	seed(t, child)
	parent := fixture(t)
	seed(t, parent)
	git(t, parent, "-c", "protocol.file.allow=always", "submodule", "add", child, "modules/core kit")
	git(t, parent, "commit", "-am", "add submodule")
	modulePath := filepath.Join(parent, "modules/core kit")
	git(t, modulePath, "config", "user.name", "Test Developer")
	git(t, modulePath, "config", "user.email", "test@example.invalid")
	write(t, modulePath, "base.txt", "new child\n")
	git(t, modulePath, "commit", "-am", "child update")
	r := openTest(t, parent)
	ctx := context.Background()
	mods, err := r.Submodules(ctx)
	if err != nil || len(mods) != 1 {
		t.Fatalf("modules=%#v %v", mods, err)
	}
	m := mods[0]
	if !m.Initialized || m.Recorded == m.Current || m.Index != m.Recorded {
		t.Fatalf("pointer separation=%#v", m)
	}
	if err := r.Stage(ctx, "modules/core kit"); err != nil {
		t.Fatal(err)
	}
	mods, err = r.Submodules(ctx)
	if err != nil || mods[0].Index != mods[0].Current || mods[0].Recorded == mods[0].Index {
		t.Fatalf("staged pointer=%#v %v", mods, err)
	}
	write(t, modulePath, "base.txt", "uncommitted inside\n")
	mods, err = r.Submodules(ctx)
	if err != nil || !mods[0].Dirty {
		t.Fatalf("child dirty=%#v %v", mods, err)
	}
	wt := filepath.Join(t.TempDir(), "review worktree")
	git(t, parent, "worktree", "add", "-b", "review", wt)
	worktrees, err := r.Worktrees(ctx)
	if err != nil || len(worktrees) != 2 {
		t.Fatalf("worktrees=%#v %v", worktrees, err)
	}
	git(t, parent, "submodule", "deinit", "-f", "--", "modules/core kit")
	mods, err = r.Submodules(ctx)
	if err != nil || mods[0].Initialized {
		t.Fatalf("uninitialized discovered as parent: %#v %v", mods, err)
	}
}

func TestResolveRefRejectsFlagsAndMissingRefs(t *testing.T) {
	ctx := context.Background()
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	sha, err := r.ResolveRef(ctx, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD")); sha != want {
		t.Fatalf("HEAD = %q; want %q", sha, want)
	}
	for _, ref := range []string{"", "--help", "-x", "missing-branch"} {
		if _, err := r.ResolveRef(ctx, ref); err == nil {
			t.Errorf("ResolveRef(%q) succeeded; want an error", ref)
		}
	}
}
