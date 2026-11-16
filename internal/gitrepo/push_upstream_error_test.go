package gitrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckedPushReportsAlreadyPublishedWhenUpstreamConfigIsLocked(t *testing.T) {
	ctx := context.Background()
	dir := fixture(t)
	seed(t, dir)
	bare := filepath.Join(t.TempDir(), "remote.git")
	git(t, dir, "init", "--bare", bare)
	git(t, dir, "remote", "add", "origin", bare)
	repo := openTest(t, dir)
	remotes, err := repo.Remotes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	sha := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	if err := os.WriteFile(filepath.Join(dir, ".git", "config.lock"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	err = repo.PushChecked(ctx, remotes[0], Branch{Name: "main", SHA: sha}, "main", true)
	if !errors.Is(err, ErrPushUpstreamFailed) || !strings.Contains(err.Error(), "git config:") {
		t.Fatalf("partial completion lost stage or original Git failure: %v", err)
	}
	if strings.TrimSpace(git(t, bare, "rev-parse", "refs/heads/main")) != sha {
		t.Fatal("fixture did not publish before upstream setup failed")
	}
}
