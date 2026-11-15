package lobedesk

import (
	"context"
	"testing"

	"supergit/internal/gitrepo"
)

// A reload starts a new read, and any history walk still out belongs to the
// read it replaces: its callback is dropped as stale and never clears its own
// flags. Clearing them here is the only chance to do it; left set, they keep
// the list on "Rereading history…" with dead arrows and a dead "Load more".
func TestAReloadReplacesAHistoryWalkStillOut(t *testing.T) {
	a := &app{locale: localeChinese, page: "history", ctx: context.Background()}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.historyReloading = true
	a.historyLoadingMore = true
	a.pendingJumpSHA = "aaaa"
	a.reload()
	if a.historyReloading || a.historyLoadingMore {
		t.Errorf("the reload kept the replaced walk's flags (reloading=%t loadingMore=%t)",
			a.historyReloading, a.historyLoadingMore)
	}
	if a.pendingJumpSHA != "" {
		t.Errorf("the replaced walk's pending jump survived the reload: %q", a.pendingJumpSHA)
	}
}
