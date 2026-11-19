package lobedesk

import (
	"context"
	"testing"
)

// Scope changes are allowed while a refresh retains the old rows on screen.
// They must not cancel the first history page belonging to that refresh.
func TestRepositoryRefreshSurvivesFileScopeChanges(t *testing.T) {
	for _, enter := range []bool{false, true} {
		name := "leave file history"
		if enter {
			name = "enter file history"
		}
		t.Run(name, func(t *testing.T) {
			a, _, root := worktreeApp(t)
			ctx, cancel := context.WithCancel(context.Background())
			a.ctx = ctx
			// Follow-up detail/file reads can finish after the snapshot callback.
			done := make(chan func(), 64)
			a.runOnMain = func(fn func()) { done <- fn }
			t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
			if !enter {
				a.fileFilter = "base.txt"
			}
			a.openRepository(root, false)
			if !a.loading {
				t.Fatal("refresh did not start")
			}
			if enter {
				a.showFileHistory("base.txt")
			} else {
				a.clearFileFilter()
			}
			runContinuation(t, done)
			if a.loading || a.historyErr != "" || len(a.commits) != 1 {
				t.Fatalf("scope change corrupted repository refresh: loading=%v historyErr=%q commits=%d", a.loading, a.historyErr, len(a.commits))
			}
			want := ""
			if enter {
				want = "base.txt"
			}
			if a.fileFilter != want {
				t.Fatalf("refresh lost current scope: %q, want %q", a.fileFilter, want)
			}
		})
	}
}
