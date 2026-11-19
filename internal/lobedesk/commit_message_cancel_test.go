package lobedesk

import (
	"context"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestMessageCommitCancelDuringPagingRestoresSourceQuery(t *testing.T) {
	a, callbacks, source, target := pagedMessageFixture(t)
	a.historyQuery = source.Subject
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Children(func() {
			a.commitInfoContent(c, source, theme.TokensOf(c))
			if a.historyLoadingMore {
				a.historyReadStatus(c)
			}
		})
	}, 800, 500)
	chooseMessageCommit(t, tt, "See "+target, "Jump to Commit · "+target)
	historyCompletion(t, callbacks)()
	late := historyCompletion(t, callbacks)
	tt.Frame()
	cancel, ok := tt.AccessByRole(ui.RoleButton, "Cancel")
	if !ok {
		t.Fatal("active paging cancellation missing")
	}
	tt.ClickAt(cancel.Bounds.X+cancel.Bounds.W/2, cancel.Bounds.Y+cancel.Bounds.H/2)
	if !a.historyPaused || a.historyQuery != source.Subject || a.selectedSHA != source.SHA || a.pendingJumpSHA != "" || a.historyNavigation.pending != nil || a.canNavigateHistory(-1) {
		t.Fatal("paging cancellation lost source search or retained the jump")
	}
	notices := a.notices[toastMain]
	if len(notices) != 1 || !strings.Contains(notices[0].description, target) || !strings.Contains(notices[0].description, a.t("commitNavigationBlocked")) {
		t.Fatal("cancelled navigation lost its full-target outcome")
	}
	late()
	if a.selectedSHA != source.SHA || a.historyQuery != source.Subject || len(a.notices[toastMain]) != 1 {
		t.Fatal("late cancelled page changed restored view")
	}
}

func TestMessageCommitCancelledResolutionDoesNotNavigate(t *testing.T) {
	a, callbacks, source, target := messageNavigationFixture(t)
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()
	a.ctx = ctx
	a.historyQuery = "Source"
	tt := messageNavigationTester(a, source)
	chooseMessageCommit(t, tt, "See "+target, "Jump to Commit · "+target)
	done := historyCompletion(t, callbacks)
	cancel()
	done()
	if a.selectedSHA != source.SHA || a.historyQuery != "Source" || a.pendingJumpSHA != "" || len(a.historyNavigation.positions) != 0 || len(a.notices[toastMain]) != 0 {
		t.Fatal("cancelled read applied a queued successful result")
	}
}
