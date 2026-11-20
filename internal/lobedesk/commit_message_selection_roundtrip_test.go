package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestMessageCommitSelectionRoundTripCancelsResolution(t *testing.T) {
	for _, phase := range []string{"resolved", "paging"} {
		for _, via := range []string{"other commit", "worktree", "same commit"} {
			t.Run(phase+"/"+via, func(t *testing.T) {
				var a *app
				var callbacks chan func()
				var source gitrepo.Commit
				var target string
				if phase == "paging" {
					a, callbacks, source, target = pagedMessageFixture(t)
				} else {
					a, callbacks, source, target = messageNavigationFixture(t)
				}
				draft := a.commitText
				other := a.commits[1].SHA
				tt := ui.NewTester(func(c *ui.Context) {
					theme.Apply(c)
					ui.Column(c).Fill().Children(func() {
						ui.Row(c).Children(func() {
							if workbenchButton(c, "Select other", buttonNormal, false).Clicked() {
								if via == "worktree" {
									a.selectWorktree()
								} else {
									a.selectCommit(other)
								}
							}
							if workbenchButton(c, "Select source", buttonNormal, false).Clicked() {
								a.selectCommit(source.SHA)
							}
						})
						a.commitInfoContent(c, source, theme.TokensOf(c))
					})
				}, 800, 400)
				chooseMessageCommit(t, tt, "See "+target, "Jump to Commit · "+target)
				done := historyCompletion(t, callbacks)
				if phase == "paging" {
					done()
					done = historyCompletion(t, callbacks)
				}
				click := func(label string) {
					t.Helper()
					item, ok := tt.AccessByRole(ui.RoleButton, label)
					if !ok {
						t.Fatal("selection button missing")
					}
					tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
				}
				if via != "same commit" {
					click("Select other")
					if via == "worktree" {
						if !a.worktreeSelected {
							t.Fatal("worktree click did not select worktree")
						}
					} else if a.selectedSHA != other {
						t.Fatal("other commit click did not select source")
					}
				}
				click("Select source")
				if a.selectedSHA != source.SHA || a.worktreeSelected {
					t.Fatal("return click did not restore source")
				}
				done()
				if a.selectedSHA != source.SHA || a.pendingJumpSHA != "" || a.historyNavigation.pending != nil || len(a.historyNavigation.positions) != 0 || len(a.notices[toastMain]) != 0 || a.commitText != draft {
					t.Fatalf("obsolete navigation revived after selection: selected=%s routes=%d", a.selectedSHA, len(a.historyNavigation.positions))
				}
			})
		}
	}
}

func TestMessageCommitSameSelectionInvalidatesMenuOrigin(t *testing.T) {
	a, _, source, target := messageNavigationFixture(t)
	origin := a.messageCommitOrigin()
	a.selectCommit(source.SHA)
	a.requestMessageCommit(origin, target)
	if a.messageCommitRequest != nil {
		t.Fatal("old menu ownership survived an explicit same-source selection")
	}
}
