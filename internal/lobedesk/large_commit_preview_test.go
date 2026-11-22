package lobedesk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestLargeCommitPreviewsFilesIndividually(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.page, a.locale = "history", localeEnglish
	for path, data := range map[string]string{
		"large.txt": strings.Repeat("x", (4<<20)+1024),
		"small.txt": "small file marker\n",
	} {
		if err := os.WriteFile(filepath.Join(a.repo.Root, path), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	selectionGit(t, a.repo.Root, "add", "--", "large.txt", "small.txt")
	selectionGit(t, a.repo.Root, "commit", "-m", "Large preview fixture")
	sha := strings.TrimSpace(selectionGit(t, a.repo.Root, "rev-parse", "HEAD"))
	a.selectCommit(sha)
	settle := func() {
		t.Helper()
		for a.commitDiffLoading {
			runContinuation(t, callbacks)
		}
	}
	settle()
	checkNotice := func(want string) {
		t.Helper()
		view := pageTester(a, a.commitDiffView)
		if !view.HasText(want) {
			t.Fatalf("missing %q in %v", want, view.Texts())
		}
		if _, ok := view.AccessByRole(ui.RoleButton, a.t("retry")); ok {
			t.Fatal("size limit offered a retry")
		}
	}
	if len(a.commitFiles) != 3 || !strings.Contains(a.commitMessage, "Large preview fixture") {
		t.Fatal("large patch hid the file list or commit message")
	}
	const summary = "Select a file to view its changes."
	checkNotice(summary)
	a.selectCommitFile("large.txt")
	settle()
	checkNotice("This file’s diff is too large to preview.")
	a.selectCommitFile("small.txt")
	settle()
	if len(a.commitSections) != 1 || a.commitSections[0].Path != "small.txt" || !strings.Contains(a.commitDiff.Text, "+small file marker") {
		t.Fatal("file tab lost its scoped patch")
	}
	a.selectCommitFile("large.txt")
	a.activateDetailTab(detailTabKey{Summary: true})
	a.selectCommitFile("small.txt")
	settle()
	if !strings.Contains(a.commitDiff.Text, "+small file marker") || a.commitDiffErr != "" {
		t.Fatal("late completion replaced the active file")
	}
	a.activateDetailTab(detailTabKey{Summary: true})
	checkNotice(summary)
	if len(a.commitSections) != 0 || a.commitDiff.Text != "" {
		t.Fatal("summary retained a single-file patch")
	}
}
