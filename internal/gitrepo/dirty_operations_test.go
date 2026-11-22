package gitrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitDelegatedDirtyOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("merge carries non-conflicting changes and rejects conflicting edits", func(t *testing.T) {
		dir := fixture(t)
		seed(t, dir)
		git(t, dir, "switch", "-c", "topic")
		write(t, dir, "topic.txt", "from topic\n")
		git(t, dir, "add", "--", "topic.txt")
		git(t, dir, "commit", "-m", "add topic")
		topicSHA := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
		git(t, dir, "switch", "main")
		write(t, dir, "local.txt", "keep local untracked\n")
		write(t, dir, "base.txt", "keep non-conflicting edit\n")

		repo := openTest(t, dir)
		if err := repo.Integrate(ctx, "merge", topicSHA); err != nil {
			t.Fatalf("non-conflicting merge failed: %v", err)
		}
		if b, err := os.ReadFile(filepath.Join(dir, "topic.txt")); err != nil || string(b) != "from topic\n" {
			t.Fatalf("merge did not apply topic.txt: %q %v", b, err)
		}
		if b, err := os.ReadFile(filepath.Join(dir, "base.txt")); err != nil || string(b) != "keep non-conflicting edit\n" {
			t.Fatalf("merge dropped non-conflicting edit: %q %v", b, err)
		}
		if b, err := os.ReadFile(filepath.Join(dir, "local.txt")); err != nil || string(b) != "keep local untracked\n" {
			t.Fatalf("merge dropped untracked file: %q %v", b, err)
		}

		// Conflicting modification on topic.txt must be refused by git without losing bytes.
		git(t, dir, "restore", "--", "base.txt")
		git(t, dir, "switch", "topic")
		write(t, dir, "topic.txt", "topic v2\n")
		git(t, dir, "commit", "-am", "topic v2")
		topicV2 := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
		git(t, dir, "switch", "main")
		write(t, dir, "topic.txt", "conflicting local edit\n")
		err := repo.Integrate(ctx, "merge", topicV2)
		if !errors.Is(err, ErrDirtyWorkingTree) {
			t.Fatalf("conflicting merge expected ErrDirtyWorkingTree, got: %v", err)
		}
		if b, readErr := os.ReadFile(filepath.Join(dir, "topic.txt")); readErr != nil || string(b) != "conflicting local edit\n" {
			t.Fatalf("conflicting merge changed local bytes: %q %v", b, readErr)
		}
	})

	t.Run("rebase allows untracked file and rejects unstaged tracked change via git", func(t *testing.T) {
		dir := fixture(t)
		seed(t, dir)
		git(t, dir, "switch", "-c", "topic")
		write(t, dir, "topic.txt", "topic commit\n")
		git(t, dir, "add", "--", "topic.txt")
		git(t, dir, "commit", "-m", "topic commit")
		git(t, dir, "switch", "main")
		write(t, dir, "main.txt", "main commit\n")
		git(t, dir, "add", "--", "main.txt")
		git(t, dir, "commit", "-m", "main commit")
		mainSHA := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
		git(t, dir, "switch", "topic")

		repo := openTest(t, dir)
		write(t, dir, "base.txt", "dirty tracked\n")
		if err := repo.Integrate(ctx, "rebase", mainSHA); !errors.Is(err, ErrDirtyWorkingTree) {
			t.Fatalf("rebase with unstaged tracked change expected ErrDirtyWorkingTree, got: %v", err)
		}
		git(t, dir, "restore", "--", "base.txt")

		write(t, dir, "untracked.txt", "keep me\n")
		if err := repo.Integrate(ctx, "rebase", mainSHA); err != nil {
			t.Fatalf("rebase with untracked file failed: %v", err)
		}
		if b, err := os.ReadFile(filepath.Join(dir, "untracked.txt")); err != nil || string(b) != "keep me\n" {
			t.Fatalf("rebase lost untracked file: %q %v", b, err)
		}
	})

	t.Run("cherry-pick and revert allow non-conflicting untracked file and reject conflicting edits", func(t *testing.T) {
		dir := fixture(t)
		seed(t, dir)
		git(t, dir, "switch", "-c", "topic")
		write(t, dir, "pick.txt", "picked\n")
		git(t, dir, "add", "--", "pick.txt")
		git(t, dir, "commit", "-m", "add pick")
		pickSHA := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
		git(t, dir, "switch", "main")

		repo := openTest(t, dir)
		write(t, dir, "notes.txt", "keep note\n")
		if err := repo.Integrate(ctx, "cherry-pick", pickSHA); err != nil {
			t.Fatalf("cherry-pick with untracked file failed: %v", err)
		}
		appliedSHA := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
		write(t, dir, "pick.txt", "local conflict\n")
		if err := repo.Integrate(ctx, "revert", appliedSHA); !errors.Is(err, ErrDirtyWorkingTree) {
			t.Fatalf("conflicting revert expected ErrDirtyWorkingTree, got: %v", err)
		}
		git(t, dir, "restore", "--", "pick.txt")
		if err := repo.Integrate(ctx, "revert", appliedSHA); err != nil {
			t.Fatalf("revert with untracked file failed: %v", err)
		}
		if b, err := os.ReadFile(filepath.Join(dir, "notes.txt")); err != nil || string(b) != "keep note\n" {
			t.Fatalf("revert lost untracked file: %q %v", b, err)
		}
	})

	t.Run("pull ff carries non-conflicting changes and rejects conflicting edits", func(t *testing.T) {
		source := fixture(t)
		seed(t, source)
		bare := filepath.Join(t.TempDir(), "remote.git")
		git(t, source, "clone", "--bare", source, bare)
		r, err := Clone(ctx, bare, filepath.Join(t.TempDir(), "clone"))
		if err != nil {
			t.Fatal(err)
		}

		// Push a new non-conflicting file from source.
		write(t, source, "upstream.txt", "from upstream\n")
		git(t, source, "add", "--", "upstream.txt")
		git(t, source, "commit", "-m", "upstream add")
		git(t, source, "push", bare, "main")

		write(t, r.Root, "base.txt", "local non-conflicting edit\n")
		if err := r.PullFF(ctx); err != nil {
			t.Fatalf("non-conflicting PullFF failed: %v", err)
		}
		if b, err := os.ReadFile(filepath.Join(r.Root, "upstream.txt")); err != nil || string(b) != "from upstream\n" {
			t.Fatalf("PullFF did not update upstream.txt: %q %v", b, err)
		}
		if b, err := os.ReadFile(filepath.Join(r.Root, "base.txt")); err != nil || string(b) != "local non-conflicting edit\n" {
			t.Fatalf("PullFF dropped local edit: %q %v", b, err)
		}

		// Now push an upstream edit to base.txt while local base.txt is dirty.
		write(t, source, "base.txt", "upstream base edit\n")
		git(t, source, "commit", "-am", "upstream base")
		git(t, source, "push", bare, "main")
		if err := r.PullFF(ctx); !errors.Is(err, ErrDirtyWorkingTree) {
			t.Fatalf("conflicting PullFF expected ErrDirtyWorkingTree, got: %v", err)
		}
		if b, err := os.ReadFile(filepath.Join(r.Root, "base.txt")); err != nil || string(b) != "local non-conflicting edit\n" {
			t.Fatalf("conflicting PullFF changed local bytes: %q %v", b, err)
		}
	})

	t.Run("create branch and checkout commit carry non-conflicting changes", func(t *testing.T) {
		dir := fixture(t)
		seed(t, dir)
		firstSHA := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
		write(t, dir, "second.txt", "second\n")
		git(t, dir, "add", "--", "second.txt")
		git(t, dir, "commit", "-m", "second")

		repo := openTest(t, dir)
		write(t, dir, "base.txt", "keep across create and detach\n")
		if err := repo.CreateBranch(ctx, "from-dirty", "HEAD", true); err != nil {
			t.Fatalf("CreateBranch with dirty tree failed: %v", err)
		}
		if got := strings.TrimSpace(git(t, dir, "branch", "--show-current")); got != "from-dirty" {
			t.Fatalf("branch = %q, want from-dirty", got)
		}
		if err := repo.CheckoutCommit(ctx, firstSHA); err != nil {
			t.Fatalf("CheckoutCommit with non-conflicting edit failed: %v", err)
		}
		if b, err := os.ReadFile(filepath.Join(dir, "base.txt")); err != nil || string(b) != "keep across create and detach\n" {
			t.Fatalf("CheckoutCommit dropped non-conflicting edit: %q %v", b, err)
		}
	})

	t.Run("untracked file collision returns ErrUntrackedOverwrite instead of ErrDirtyWorkingTree", func(t *testing.T) {
		dir := fixture(t)
		seed(t, dir)
		git(t, dir, "switch", "-c", "topic")
		write(t, dir, "collide.txt", "from topic\n")
		git(t, dir, "add", "--", "collide.txt")
		git(t, dir, "commit", "-m", "add collide")
		topicSHA := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
		git(t, dir, "switch", "main")

		repo := openTest(t, dir)
		write(t, dir, "collide.txt", "untracked local\n")

		if err := repo.SwitchBranch(ctx, "topic"); !errors.Is(err, ErrUntrackedOverwrite) || errors.Is(err, ErrDirtyWorkingTree) {
			t.Fatalf("switch untracked collision expected ErrUntrackedOverwrite only, got: %v", err)
		}
		if err := repo.Integrate(ctx, "merge", topicSHA); !errors.Is(err, ErrUntrackedOverwrite) || errors.Is(err, ErrDirtyWorkingTree) {
			t.Fatalf("merge untracked collision expected ErrUntrackedOverwrite only, got: %v", err)
		}
		if b, err := os.ReadFile(filepath.Join(dir, "collide.txt")); err != nil || string(b) != "untracked local\n" {
			t.Fatalf("untracked collision changed local bytes: %q %v", b, err)
		}
	})

	t.Run("ignored files do not block rebase or cherry-pick like native git", func(t *testing.T) {
		dir := fixture(t)
		seed(t, dir)
		write(t, dir, ".gitignore", "build/\nignored.txt\n")
		git(t, dir, "add", ".gitignore")
		git(t, dir, "commit", "-m", "ignore rules")
		git(t, dir, "switch", "-c", "topic")
		write(t, dir, "topic.txt", "topic\n")
		write(t, dir, "ignored.txt", "tracked in topic\n")
		git(t, dir, "add", "-f", "--", "topic.txt", "ignored.txt")
		git(t, dir, "commit", "-m", "topic commit")
		topicSHA := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
		git(t, dir, "switch", "main")
		write(t, dir, "main.txt", "main\n")
		git(t, dir, "add", "--", "main.txt")
		git(t, dir, "commit", "-m", "main commit")
		mainSHA := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))

		repo := openTest(t, dir)
		write(t, dir, "build/output.bin", "local build artifact\n")
		write(t, dir, "ignored.txt", "local ignored copy\n")

		if err := repo.Integrate(ctx, "cherry-pick", topicSHA); err != nil {
			t.Fatalf("cherry-pick should match native git with ignored files: %v", err)
		}
		git(t, dir, "switch", "topic")
		write(t, dir, "build/output.bin", "local build artifact\n")
		if err := repo.Integrate(ctx, "rebase", mainSHA); err != nil {
			t.Fatalf("rebase should match native git with unrelated ignored files: %v", err)
		}
	})
}
