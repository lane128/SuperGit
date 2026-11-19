package lobedesk

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestDiscardPreparationDoesNotFollowHiddenSurface(t *testing.T) {
	for _, surface := range []string{"repository chooser", "historical selection", "other page"} {
		t.Run(surface, func(t *testing.T) {
			a, callbacks, tt := discardShortcutFixture(t)
			clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
			next := func() func() {
				t.Helper()
				select {
				case fn := <-callbacks:
					return fn
				case <-time.After(5 * time.Second):
					t.Fatal("callback missing")
					return nil
				}
			}
			for a.diffLoading {
				next()()
			}
			tt.Key(0, ui.KeyBackspace)
			prepared := next()
			switch surface {
			case "repository chooser":
				a.repositoryChooserOpen = true
			case "historical selection":
				a.worktreeSelected = false
			case "other page":
				a.page = "branches"
			}
			prepared()
			if a.confirm.open {
				t.Fatal("preparation opened discard confirmation after leaving its interaction surface")
			}
			assertDiscardBytes(t, a, "b.go", "working version\n")
		})
	}
}
