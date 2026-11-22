package lobedesk

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"supergit/internal/gitrepo"
)

func TestDirtyWorkingTreeRecoveryUsesCurrentLanguage(t *testing.T) {
	a, _, root := worktreeApp(t)
	first := strings.TrimSpace(pullFixtureGit(t, root, "rev-parse", "HEAD"))
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("second commit\n"), 0600); err != nil {
		t.Fatal(err)
	}
	pullFixtureGit(t, root, "commit", "-am", "advance base")
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("preserved local edit\n"), 0600); err != nil {
		t.Fatal(err)
	}
	err := a.repo.CreateBranch(context.Background(), "must-not-checkout", first, true)
	if !errors.Is(err, gitrepo.ErrDirtyWorkingTree) {
		t.Fatalf("conflicting dirty checkout did not return typed dirty guard: %v", err)
	}
	for _, tc := range []struct {
		language locale
		want     string
	}{
		{localeChinese, "工作区或暂存区有改动，请先提交或保存 Stash"},
		{localeEnglish, "The working tree or Index has changes. Commit or stash them before retrying."},
	} {
		a.locale = tc.language
		got := a.operationErrorDescription(err)
		if !strings.HasPrefix(got, tc.want) || !strings.Contains(got, "base.txt") {
			t.Errorf("%s recovery=%q; want prefix %q and conflicting file base.txt", tc.language, got, tc.want)
		}
	}
	data, readErr := os.ReadFile(filepath.Join(root, "base.txt"))
	if readErr != nil || string(data) != "preserved local edit\n" {
		t.Fatalf("guard changed local bytes: %q, %v", data, readErr)
	}
}
