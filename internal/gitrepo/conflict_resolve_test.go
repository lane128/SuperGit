package gitrepo

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGitTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Tester",
		"GIT_AUTHOR_EMAIL=tester@example.com",
		"GIT_COMMITTER_NAME=Tester",
		"GIT_COMMITTER_EMAIL=tester@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}

func setupMergeConflictRepo(t *testing.T, filename string, base, ours, theirs []byte) *Repository {
	t.Helper()
	dir := t.TempDir()
	runGitTest(t, dir, "init", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, filename), base, 0o644); err != nil {
		t.Fatal(err)
	}
	runGitTest(t, dir, "add", filename)
	runGitTest(t, dir, "commit", "-m", "base")

	runGitTest(t, dir, "checkout", "-b", "topic")
	if err := os.WriteFile(filepath.Join(dir, filename), theirs, 0o644); err != nil {
		t.Fatal(err)
	}
	runGitTest(t, dir, "add", filename)
	runGitTest(t, dir, "commit", "-m", "theirs")

	runGitTest(t, dir, "checkout", "main")
	if err := os.WriteFile(filepath.Join(dir, filename), ours, 0o644); err != nil {
		t.Fatal(err)
	}
	runGitTest(t, dir, "add", filename)
	runGitTest(t, dir, "commit", "-m", "ours")

	// Trigger merge conflict (expected non-zero exit).
	cmd := exec.Command("git", "merge", "topic")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Tester",
		"GIT_AUTHOR_EMAIL=tester@example.com",
		"GIT_COMMITTER_NAME=Tester",
		"GIT_COMMITTER_EMAIL=tester@example.com",
	)
	if out, err := cmd.CombinedOutput(); err == nil {
		t.Fatalf("expected merge conflict, got clean merge:\n%s", out)
	}

	repo, err := Open(context.Background(), dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return repo
}

func TestMarkConflictResolvedBlocksRemainingMarkersAndAllowsCleanFile(t *testing.T) {
	ctx := context.Background()
	repo := setupMergeConflictRepo(t, "notes.txt", []byte("base\n"), []byte("ours\n"), []byte("theirs\n"))

	err := repo.MarkConflictResolved(ctx, "notes.txt")
	if !errors.Is(err, ErrConflictMarkers) {
		t.Fatalf("MarkConflictResolved with markers = %v, want ErrConflictMarkers", err)
	}
	status, err := repo.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(status.Entries) != 1 || !status.Entries[0].Conflict {
		t.Fatalf("conflict must remain unmerged after rejection: %+v", status.Entries)
	}

	// Partial edit that leaves a lone divider marker must still be blocked.
	if err := os.WriteFile(filepath.Join(repo.Root, "notes.txt"), []byte("ours\n=======\ntheirs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkConflictResolved(ctx, "notes.txt"); !errors.Is(err, ErrConflictMarkers) {
		t.Fatalf("MarkConflictResolved with stray ======= = %v, want ErrConflictMarkers", err)
	}

	// Cleanly resolved file (including a long line before normal content) succeeds.
	resolved := strings.Repeat("a", 8192) + "\nresolved content\n"
	if err := os.WriteFile(filepath.Join(repo.Root, "notes.txt"), []byte(resolved), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkConflictResolved(ctx, "notes.txt"); err != nil {
		t.Fatalf("MarkConflictResolved on clean file: %v", err)
	}
	status, err = repo.Status(ctx)
	if err != nil {
		t.Fatalf("Status after resolve: %v", err)
	}
	if len(status.Entries) != 1 || status.Entries[0].Conflict || status.Entries[0].Index != 'M' {
		t.Fatalf("expected clean staged entry after resolve, got %+v", status.Entries)
	}
}

func TestMarkConflictResolvedAllowsBinaryAndDeletedFiles(t *testing.T) {
	ctx := context.Background()

	// Binary file containing NUL bytes and marker-like bytes should not be falsely blocked.
	binRepo := setupMergeConflictRepo(t, "blob.bin", []byte{0x00, 0x01}, []byte{0x00, 0x02}, []byte{0x00, 0x03})
	binaryResolved := append([]byte{0x00, 0x04, '\n'}, []byte("<<<<<<< binary payload\n")...)
	if err := os.WriteFile(filepath.Join(binRepo.Root, "blob.bin"), binaryResolved, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := binRepo.MarkConflictResolved(ctx, "blob.bin"); err != nil {
		t.Fatalf("MarkConflictResolved rejected binary file: %v", err)
	}

	// Deleted working file (resolved by removing the path) should stage deletion cleanly.
	delRepo := setupMergeConflictRepo(t, "drop.txt", []byte("base\n"), []byte("ours\n"), []byte("theirs\n"))
	if err := os.Remove(filepath.Join(delRepo.Root, "drop.txt")); err != nil {
		t.Fatal(err)
	}
	if err := delRepo.MarkConflictResolved(ctx, "drop.txt"); err != nil {
		t.Fatalf("MarkConflictResolved rejected deleted file: %v", err)
	}
	status, err := delRepo.Status(ctx)
	if err != nil {
		t.Fatalf("Status after delete resolve: %v", err)
	}
	if len(status.Entries) != 1 || status.Entries[0].Conflict || status.Entries[0].Index != 'D' {
		t.Fatalf("expected staged deletion, got %+v", status.Entries)
	}
}
