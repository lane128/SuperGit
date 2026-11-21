package lobedesk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveConflictBlocksRemainingMarkersWithLocalizedError(t *testing.T) {
	a, done, root := conflictEditorFixture(t)
	a.locale = localeEnglish
	before := splitCommitGit(t, root, "ls-files", "--stage")

	a.resolveConflict("base.txt")
	for a.busy || a.loading {
		runContinuation(t, done)
	}

	if splitCommitGit(t, root, "ls-files", "--stage") != before {
		t.Fatal("mark resolved staged a file that still contained conflict markers")
	}
	notices := a.notices[toastMain]
	if len(notices) == 0 || !strings.Contains(notices[len(notices)-1].description, a.t("resolveConflictMarkers")) {
		t.Fatalf("expected localized marker error notice, got %+v", notices)
	}

	// Cleanly edited working file resolves and stages normally.
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("resolved clean content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a.resolveConflict("base.txt")
	for a.busy || a.loading {
		runContinuation(t, done)
	}

	after := splitCommitGit(t, root, "ls-files", "--stage")
	if after == before {
		t.Fatal("mark resolved did not stage a cleanly resolved conflict file")
	}
	if strings.Contains(after, "\t1\t") || strings.Contains(after, "\t2\t") || strings.Contains(after, "\t3\t") {
		t.Fatalf("resolved file still has unmerged index stages:\n%s", after)
	}
}
