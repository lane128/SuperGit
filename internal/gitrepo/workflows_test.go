package gitrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStashRoundTripAndStaleDropProtection(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	write(t, dir, "base.txt", "first\n")
	git(t, dir, "add", "base.txt")
	write(t, dir, "new.txt", "untracked\n")
	for _, path := range []string{"*.txt", "空 格\nfile", "-option"} {
		write(t, dir, path, "special\n")
	}
	if err := r.SaveStash(ctx, "first stash", true); err != nil {
		t.Fatal(err)
	}
	list, err := r.Stashes(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("stashes %v %v", list, err)
	}
	first := list[0]
	if s, err := r.Status(ctx); err != nil || len(s.Entries) != 0 {
		t.Fatalf("save did not clean working tree: %+v %v", s, err)
	}
	d, err := r.StashDiff(ctx, first.SHA)
	if err != nil || !strings.Contains(d.Text, "untracked") {
		t.Fatalf("stash diff missing untracked: %s %v", d.Text, err)
	}
	write(t, dir, "base.txt", "second\n")
	if err := r.SaveStash(ctx, "second stash", false); err != nil {
		t.Fatal(err)
	}
	if err := r.DropStash(ctx, first.Ref, first.SHA); err == nil {
		t.Fatal("stale index dropped wrong stash")
	}
	if err := r.ApplyStash(ctx, first.SHA); err != nil {
		s, statusErr := r.Status(ctx)
		t.Fatalf("apply: %v; status=%+v (%v); porcelain=%q; stashes=%q", err, s, statusErr, git(t, dir, "status", "--porcelain=v2"), git(t, dir, "stash", "list"))
	}
	if got := git(t, dir, "show", ":base.txt"); got != "first\n" {
		t.Fatal("index not restored")
	}
	if b, err := os.ReadFile(filepath.Join(dir, "new.txt")); err != nil || string(b) != "untracked\n" {
		t.Fatal("untracked not restored")
	}
	if list, _ := r.Stashes(ctx); len(list) != 2 {
		t.Fatal("apply removed stash")
	}
}
func TestStashPopConflictPreservesEntry(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	write(t, dir, "base.txt", "stashed\n")
	if err := r.SaveStash(ctx, "keep on conflict", false); err != nil {
		t.Fatal(err)
	}
	list, err := r.Stashes(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("stashes %v %v", list, err)
	}
	write(t, dir, "base.txt", "new head\n")
	git(t, dir, "commit", "-am", "new head")
	if err := r.PopStash(ctx, list[0].Ref, list[0].SHA); err == nil {
		t.Fatal("expected conflict")
	}
	remaining, err := r.Stashes(ctx)
	if err != nil || len(remaining) != 1 || remaining[0].SHA != list[0].SHA {
		t.Fatalf("stash lost: %v %v", remaining, err)
	}
	s, err := r.Status(ctx)
	if err != nil || len(s.Entries) != 1 || !s.Entries[0].Conflict {
		t.Fatalf("missing conflict: %+v %v", s, err)
	}
}

func TestStashApplyOntoDirtyWorktree(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, "keep.txt", "keep original\n")
	git(t, dir, "add", "--", "keep.txt")
	git(t, dir, "commit", "-m", "keep")
	r := openTest(t, dir)
	ctx := context.Background()
	write(t, dir, "base.txt", "stashed\n")
	if err := r.SaveStash(ctx, "over dirty tree", false); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "keep.txt", "keep dirty\n")
	list, err := r.Stashes(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("stashes %v %v", list, err)
	}
	if err := r.ApplyStash(ctx, list[0].SHA); err != nil {
		t.Fatalf("apply onto a dirty worktree: %v", err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "base.txt")); err != nil || string(b) != "stashed\n" {
		t.Fatalf("stash not applied: %q %v", b, err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "keep.txt")); err != nil || string(b) != "keep dirty\n" {
		t.Fatalf("unrelated edit lost: %q %v", b, err)
	}
	if list, _ := r.Stashes(ctx); len(list) != 1 {
		t.Fatal("apply removed the stash")
	}
}
func TestStashPopOntoDirtyWorktree(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, "keep.txt", "keep original\n")
	git(t, dir, "add", "--", "keep.txt")
	git(t, dir, "commit", "-m", "keep")
	r := openTest(t, dir)
	ctx := context.Background()
	write(t, dir, "base.txt", "stashed\n")
	if err := r.SaveStash(ctx, "pop over dirty tree", false); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "keep.txt", "keep dirty\n")
	list, err := r.Stashes(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("stashes %v %v", list, err)
	}
	if err := r.PopStash(ctx, list[0].Ref, list[0].SHA); err != nil {
		t.Fatalf("pop onto a dirty worktree: %v", err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "base.txt")); err != nil || string(b) != "stashed\n" {
		t.Fatalf("stash not applied: %q %v", b, err)
	}
	if list, _ := r.Stashes(ctx); len(list) != 0 {
		t.Fatal("pop kept the entry")
	}
}

func TestStashApplyErrorHints(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	if got := r.stashApplyError(ctx, nil, "应用"); got != nil {
		t.Errorf("nil error changed: %v", got)
	}
	overlap := errors.New("git stash: exit status 1\nerror: Your local changes to the following files would be overwritten by merge:")
	if got := r.stashApplyError(ctx, overlap, "弹出"); got == nil || !strings.Contains(got.Error(), "重叠") || !strings.Contains(got.Error(), "弹出") {
		t.Errorf("overlap hint = %v", got)
	}
	conflict := errors.New("git stash: exit status 1\nCONFLICT (content): Merge conflict in base.txt")
	if got := r.stashApplyError(ctx, conflict, "应用"); got == nil || !strings.Contains(got.Error(), "冲突") {
		t.Errorf("conflict hint = %v", got)
	}
	other := errors.New("git stash: exit status 128\nfatal: bad revision")
	if got := r.stashApplyError(ctx, other, "应用"); got != other {
		t.Errorf("unrelated error = %v; want it unchanged", got)
	}

	// A pop that lands with conflicts reports "Index was not unstashed";
	// the conflicted status is what turns that into the conflict hint.
	conflictDir := conflictFixture(t)
	conflictRepo := openTest(t, conflictDir)
	if err := conflictRepo.Integrate(ctx, "merge", "feature"); err == nil {
		t.Fatal("expected a merge conflict")
	}
	notUnstashed := errors.New("git stash: exit status 1\nIndex was not unstashed.")
	if got := conflictRepo.stashApplyError(ctx, notUnstashed, "弹出"); got == nil || !strings.Contains(got.Error(), "冲突") {
		t.Errorf("conflicted status hint = %v", got)
	}
}

func TestConflictRejectsSymlinkWorkingFile(t *testing.T) {
	dir := conflictFixture(t)
	r := openTest(t, dir)
	ctx := context.Background()
	if err := r.Integrate(ctx, "merge", "feature"); err == nil {
		t.Fatal("expected conflict")
	}
	c, err := r.Conflict(ctx, "base.txt")
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.txt")
	write(t, filepath.Dir(outside), filepath.Base(outside), "preserve\n")
	if err := os.Remove(filepath.Join(dir, "base.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "base.txt")); err != nil {
		t.Fatal(err)
	}
	if err := r.ResolveConflict(ctx, c, "replacement\n"); err == nil {
		t.Fatal("symlink accepted")
	}
	b, err := os.ReadFile(outside)
	if err != nil || string(b) != "preserve\n" {
		t.Fatal("outside content changed")
	}
}

func conflictFixture(t *testing.T) string {
	t.Helper()
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "checkout", "-b", "feature")
	write(t, dir, "base.txt", "theirs\n")
	git(t, dir, "commit", "-am", "theirs")
	git(t, dir, "checkout", "main")
	write(t, dir, "base.txt", "ours\n")
	git(t, dir, "commit", "-am", "ours")
	return dir
}
func TestCheckoutCommitDetachesAndGuardsDirtyTree(t *testing.T) {
	ctx := context.Background()
	dir := fixture(t)
	seed(t, dir)
	first := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	write(t, dir, "base.txt", "second\n")
	git(t, dir, "commit", "-am", "second")
	second := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	r := openTest(t, dir)
	if err := r.CheckoutCommit(ctx, "not-a-sha"); err == nil {
		t.Fatal("invalid SHA accepted")
	}
	if err := r.CheckoutCommit(ctx, first); err != nil {
		t.Fatal(err)
	}
	s, err := r.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if s.Branch != "" || s.Head != first {
		t.Fatalf("status branch=%q head=%q; want detached at %s", s.Branch, s.Head, first)
	}
	write(t, dir, "base.txt", "dirty\n")
	if err := r.CheckoutCommit(ctx, second); err == nil {
		t.Fatal("dirty checkout accepted")
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "base.txt")); string(b) != "dirty\n" {
		t.Fatal("dirty work lost")
	}
	git(t, dir, "restore", "--", "base.txt")
	if err := r.SwitchBranch(ctx, "main"); err != nil {
		t.Fatal(err)
	}
	if s, _ := r.Status(ctx); s.Branch != "main" {
		t.Fatalf("branch = %q; want main", s.Branch)
	}
}

func TestOperationMessageReadsPreparedMergeMessage(t *testing.T) {
	ctx := context.Background()
	idle := openTest(t, fixture(t))
	if message, err := idle.OperationMessage(ctx); err != nil || message != "" {
		t.Fatalf("idle repository message = %q, %v; want empty", message, err)
	}
	dir := conflictFixture(t)
	r := openTest(t, dir)
	if err := r.Integrate(ctx, "merge", "feature"); err == nil {
		t.Fatal("expected merge conflict")
	}
	message, err := r.OperationMessage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(message, "Merge commit '") {
		t.Fatalf("message = %q; want Git's prepared merge message", message)
	}
	if strings.Contains(message, "# Conflicts") {
		t.Fatalf("message kept Git's comment lines: %q", message)
	}

	pick := fixture(t)
	seed(t, pick)
	git(t, pick, "checkout", "-b", "side")
	write(t, pick, "base.txt", "side change\n")
	git(t, pick, "commit", "-am", "side change")
	git(t, pick, "checkout", "main")
	write(t, pick, "base.txt", "main change\n")
	git(t, pick, "commit", "-am", "main change")
	pr := openTest(t, pick)
	if err := pr.Integrate(ctx, "cherry-pick", "side"); err == nil {
		t.Fatal("expected cherry-pick conflict")
	}
	if message, err := pr.OperationMessage(ctx); err != nil || message != "side change" {
		t.Fatalf("cherry-pick message = %q, %v; want the picked subject", message, err)
	}
}

func TestMergeConflictSaveContinueAndAbort(t *testing.T) {
	dir := conflictFixture(t)
	r := openTest(t, dir)
	ctx := context.Background()
	if err := r.Integrate(ctx, "merge", "feature"); err == nil {
		t.Fatal("expected merge conflict")
	}
	op, err := r.Operation(ctx)
	if err != nil || op.Kind != "merge" {
		t.Fatalf("operation %v %v", op, err)
	}
	c, err := r.Conflict(ctx, "base.txt")
	if err != nil {
		t.Fatal(err)
	}
	if c.Base != "original\n" || c.Ours != "ours\n" || c.Theirs != "theirs\n" {
		t.Fatalf("stages %+v", c)
	}
	if err := r.ResolveConflict(ctx, c, "<<<<<<< still unresolved\n"); err == nil {
		t.Fatal("conflict markers accepted")
	}
	write(t, dir, "base.txt", "external edit\n")
	if err := r.ResolveConflict(ctx, c, "resolved\n"); err == nil {
		t.Fatal("stale conflict overwrote external edit")
	}
	c, err = r.Conflict(ctx, "base.txt")
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ResolveConflict(ctx, c, "combined result\n"); err != nil {
		t.Fatal(err)
	}
	if err := r.Continue(ctx, "merge"); err != nil {
		t.Fatal(err)
	}
	if op, _ := r.Operation(ctx); op.Kind != "" {
		t.Fatal("merge not finished")
	}
	if len(strings.Fields(git(t, dir, "show", "-s", "--format=%P", "HEAD"))) != 2 {
		t.Fatal("resolution did not create merge commit")
	}
	other := conflictFixture(t)
	ro := openTest(t, other)
	before := git(t, other, "rev-parse", "HEAD")
	ro.Integrate(ctx, "merge", "feature")
	if err := ro.Abort(ctx, "merge"); err != nil {
		t.Fatal(err)
	}
	if got := git(t, other, "rev-parse", "HEAD"); got != before {
		t.Fatal("abort changed original HEAD")
	}
}
func TestSkipEmptyCherryPick(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "checkout", "-b", "side")
	write(t, dir, "base.txt", "same\n")
	git(t, dir, "commit", "-am", "side change")
	git(t, dir, "checkout", "main")
	write(t, dir, "base.txt", "same\n")
	git(t, dir, "commit", "-am", "same change")
	before := git(t, dir, "rev-parse", "HEAD")
	r := openTest(t, dir)
	ctx := context.Background()
	if err := r.Integrate(ctx, "cherry-pick", "side"); err == nil {
		t.Fatal("an already-applied change should stop the cherry-pick")
	}
	if op, _ := r.Operation(ctx); op.Kind != "cherry-pick" {
		t.Fatalf("operation = %+v; want cherry-pick", op)
	}
	if err := r.Skip(ctx, "merge"); err == nil {
		t.Fatal("skip should reject a kind that is not running")
	}
	if err := r.Skip(ctx, "cherry-pick"); err != nil {
		t.Fatal(err)
	}
	if op, _ := r.Operation(ctx); op.Kind != "" {
		t.Fatal("skip did not end the cherry-pick")
	}
	if got := git(t, dir, "rev-parse", "HEAD"); got != before {
		t.Fatal("skip created or moved a commit")
	}
}

func TestTakeConflictSideHandlesDeletion(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "checkout", "-b", "deleted")
	git(t, dir, "rm", "-q", "--", "base.txt")
	git(t, dir, "commit", "-m", "delete base")
	git(t, dir, "checkout", "main")
	write(t, dir, "base.txt", "modified\n")
	git(t, dir, "commit", "-am", "modify base")
	r := openTest(t, dir)
	ctx := context.Background()
	if err := r.Integrate(ctx, "merge", "deleted"); err == nil {
		t.Fatal("expected a delete/modify conflict")
	}
	if err := r.TakeConflictSide(ctx, "base.txt", "theirs"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "base.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("accepting the deleting side kept the file")
	}
	s, err := r.Status(ctx)
	if err != nil || len(s.Entries) != 1 || s.Entries[0].Conflict || s.Entries[0].Index != 'D' {
		t.Fatalf("staged deletion missing: %+v %v", s, err)
	}
	if err := r.Continue(ctx, "merge"); err != nil {
		t.Fatal(err)
	}
	if op, _ := r.Operation(ctx); op.Kind != "" {
		t.Fatal("merge not finished")
	}

	// The modifying side still checks out as before.
	other := fixture(t)
	seed(t, other)
	git(t, other, "checkout", "-b", "deleted")
	git(t, other, "rm", "-q", "--", "base.txt")
	git(t, other, "commit", "-m", "delete base")
	git(t, other, "checkout", "main")
	write(t, other, "base.txt", "modified\n")
	git(t, other, "commit", "-am", "modify base")
	ro := openTest(t, other)
	if err := ro.Integrate(ctx, "merge", "deleted"); err == nil {
		t.Fatal("expected a delete/modify conflict")
	}
	if err := ro.TakeConflictSide(ctx, "base.txt", "ours"); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(other, "base.txt")); err != nil || string(b) != "modified\n" {
		t.Fatalf("keeping side lost the file: %q %v", b, err)
	}
	// Keeping our side leaves the tree as HEAD has it, so nothing is left
	// to show, but the conflict is gone.
	if s, err := ro.Status(ctx); err != nil || len(s.Entries) != 0 {
		t.Fatalf("conflict not resolved: %+v %v", s, err)
	}
}

func TestIntegrationAndWorktreeGuards(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	if err := r.CreateBranch(ctx, "feature", "HEAD", false); err != nil {
		t.Fatal(err)
	}
	wt := filepath.Join(t.TempDir(), "review")
	if err := r.AddWorktree(ctx, wt, "review", "HEAD"); err != nil {
		t.Fatal(err)
	}
	write(t, wt, "untracked.txt", "keep me\n")
	if err := r.RemoveWorktree(ctx, wt); err == nil {
		t.Fatal("dirty worktree removed")
	}
	if _, err := os.Stat(filepath.Join(wt, "untracked.txt")); err != nil {
		t.Fatal("worktree content lost")
	}
	if err := r.DeleteBranch(ctx, "review"); err == nil {
		t.Fatal("checked-out branch deleted")
	}
	write(t, dir, "base.txt", "dirty\n")
	if err := r.Integrate(ctx, "rebase", "feature"); err == nil {
		t.Fatal("dirty rebase accepted")
	}
	if err := r.RemoveWorktree(ctx, dir); err == nil {
		t.Fatal("current worktree removed")
	}
}
func TestSubmoduleInitAndDirtyUpdateGuard(t *testing.T) {
	child := fixture(t)
	seed(t, child)
	parent := fixture(t)
	seed(t, parent)
	git(t, parent, "-c", "protocol.file.allow=always", "submodule", "add", child, "module")
	git(t, parent, "commit", "-am", "module")
	r := openTest(t, parent)
	ctx := context.Background()
	write(t, filepath.Join(parent, "module"), "base.txt", "keep dirty\n")
	if err := r.UpdateSubmodule(ctx, "module"); err == nil {
		t.Fatal("dirty module updated")
	}
	if err := r.UpdateSubmodule(ctx, "../outside"); err == nil {
		t.Fatal("external module path allowed")
	}
	git(t, parent, "submodule", "deinit", "-f", "module")
	git(t, parent, "config", "protocol.file.allow", "always")
	t.Setenv("GIT_ALLOW_PROTOCOL", "file")
	if err := r.UpdateSubmodule(ctx, "module"); err != nil {
		t.Fatal(err)
	}
	mods, err := r.Submodules(ctx)
	if err != nil || !mods[0].Initialized {
		t.Fatalf("not initialized %v %v", mods, err)
	}
}

func TestResetToModes(t *testing.T) {
	ctx := context.Background()
	dir := fixture(t)
	seed(t, dir)
	first := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	write(t, dir, "base.txt", "second\n")
	git(t, dir, "commit", "-am", "second")
	second := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	r := openTest(t, dir)
	if err := r.ResetTo(ctx, first, "keep"); err == nil {
		t.Fatal("unknown mode accepted")
	}
	if err := r.ResetTo(ctx, "not-a-sha", "mixed"); err == nil {
		t.Fatal("invalid SHA accepted")
	}

	// Soft keeps the index and worktree, so the rolled-back commit stays
	// staged.
	if err := r.ResetTo(ctx, first, "soft"); err != nil {
		t.Fatal(err)
	}
	if head := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD")); head != first {
		t.Fatalf("head = %s; want %s", head, first)
	}
	if out := git(t, dir, "status", "--porcelain"); strings.TrimSpace(out) != "M  base.txt" {
		t.Fatalf("soft status = %q; want staged change", out)
	}

	// Mixed resets the index only, so the same change becomes unstaged.
	if err := r.ResetTo(ctx, first, "mixed"); err != nil {
		t.Fatal(err)
	}
	if out := git(t, dir, "status", "--porcelain"); strings.TrimSpace(out) != "M base.txt" {
		t.Fatalf("mixed status = %q; want unstaged change", out)
	}

	// Hard discards both, even a dirty file.
	if err := r.ResetTo(ctx, second, "hard"); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "base.txt", "dirty\n")
	if err := r.ResetTo(ctx, first, "hard"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "base.txt")); string(b) != "original\n" {
		t.Fatalf("hard reset kept worktree content %q", b)
	}
	if out := git(t, dir, "status", "--porcelain"); strings.TrimSpace(out) != "" {
		t.Fatalf("hard status = %q; want clean", out)
	}

	// An interrupted operation keeps the repository busy.
	conflict := conflictFixture(t)
	busy := openTest(t, conflict)
	if err := busy.Integrate(ctx, "merge", "feature"); err == nil {
		t.Fatal("expected merge conflict")
	}
	target := strings.TrimSpace(git(t, conflict, "rev-parse", "HEAD~1"))
	if err := busy.ResetTo(ctx, target, "hard"); err == nil {
		t.Fatal("reset ran while merging")
	}
}

func TestSaveStashPathsStashesOnlySelectedFile(t *testing.T) {
	ctx := context.Background()
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	write(t, dir, "base.txt", "changed\n")
	write(t, dir, "notes.txt", "new\n")
	write(t, dir, "wild[1].txt", "wild\n")

	if err := r.SaveStashPaths(ctx, "base only", false, []string{"base.txt"}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "base.txt")); string(b) != "original\n" {
		t.Fatalf("base.txt = %q; want the change stashed", b)
	}
	for _, path := range []string{"notes.txt", "wild[1].txt"} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Fatalf("%s disappeared with an unrelated path stash: %v", path, err)
		}
	}
	list, err := r.Stashes(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("stashes %v %v", list, err)
	}
	if d, err := r.StashDiff(ctx, list[0].SHA); err != nil || !strings.Contains(d.Text, "base.txt") || strings.Contains(d.Text, "notes.txt") {
		t.Fatalf("stash diff = %q (%v); want only base.txt", d.Text, err)
	}

	// Untracked paths need --include-untracked, and the wildcard in a
	// filename has to stay literal.
	for i, path := range []string{"notes.txt", "wild[1].txt"} {
		if err := r.SaveStashPaths(ctx, path, true, []string{path}); err != nil {
			t.Fatalf("stash %s: %v", path, err)
		}
		if _, err := os.Stat(filepath.Join(dir, path)); !os.IsNotExist(err) {
			t.Fatalf("%s still in the work tree after stashing", path)
		}
		if list, _ := r.Stashes(ctx); len(list) != i+2 {
			t.Fatalf("stashes after %s = %d; want %d", path, len(list), i+2)
		}
	}

	if err := r.SaveStashPaths(ctx, "", false, []string{"gone.txt"}); err == nil {
		t.Fatal("path without changes accepted")
	}
	if err := r.SaveStashPaths(ctx, "", false, []string{"../outside"}); err == nil {
		t.Fatal("path outside the repository accepted")
	}
}
