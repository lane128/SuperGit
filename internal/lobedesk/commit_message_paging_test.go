package lobedesk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"supergit/internal/gitrepo"
)

func TestMessageCommitReportsObjectAndTraversalOutcomes(t *testing.T) {
	for _, scenario := range []string{"missing", "blob", "outside traversal", "paused", "read error"} {
		t.Run(scenario, func(t *testing.T) {
			a, callbacks, source, target := messageNavigationFixture(t)
			a.commits = a.commits[:2]
			a.historyQuery = "Source"
			switch scenario {
			case "missing":
				target = strings.Repeat("0", 40)
			case "blob":
				target = pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD:same.go")
			case "outside traversal":
				target = pullFixtureGit(t, a.repo.Root, "commit-tree", "HEAD^{tree}", "-m", "Outside pinned main traversal")
			case "paused":
				a.historyMore = true
				a.historyPaused = true
			case "read error":
				a.historyMore = true
				a.historyPageErr = "owned read failure"
			}
			a.commitMessage = source.Subject + "\n\nSee " + target
			tt := messageNavigationTester(a, source)
			chooseMessageCommit(t, tt, "See "+target, "Jump to Commit · "+target)
			historyCompletion(t, callbacks)()
			if a.selectedSHA != source.SHA || a.historyQuery != "Source" || a.pendingJumpSHA != "" || a.historyNavigation.pending != nil || a.canNavigateHistory(-1) || a.historyRef.Name != "main" || a.historyPinned != source.SHA {
				t.Fatal("failed jump changed source, query, route or ref")
			}
			notices := a.notices[toastMain]
			if len(notices) != 1 || !strings.Contains(notices[0].description, target) {
				t.Fatal("outcome lost complete target")
			}
			switch scenario {
			case "missing", "blob":
				if notices[0].kind != "error" || notices[0].description != a.tf("messageCommitUnavailable", target) {
					t.Fatal("unresolvable object reported as traversal exhaustion")
				}
			case "outside traversal":
				if notices[0].kind != "warning" || notices[0].description != a.tf("messageCommitOutsideHistory", target) {
					t.Fatal("existing object was described as missing")
				}
			default:
				if notices[0].kind != "warning" || !strings.Contains(notices[0].description, a.t("commitNavigationBlocked")) || strings.Contains(notices[0].description, "outside") {
					t.Fatal("interrupted read reported as EOF")
				}
			}
		})
	}
}

func pagedMessageFixture(t *testing.T) (*app, chan func(), gitrepo.Commit, string) {
	t.Helper()
	a, callbacks, all := automaticHistoryFixture(t, 420)
	source, target := a.commits[0], all[350]
	if target == source.SHA || len(source.Parents) > 0 && source.Parents[0] == target {
		t.Fatal("paging target is source or parent")
	}
	a.locale, a.page, a.worktreeSelected, a.selectedSHA = localeEnglish, "history", false, source.SHA
	a.commitMessage = source.Subject + "\n\nSee " + target
	a.commitFilesLoaded = true
	a.commitFields.Summary = "keep paging draft"
	t.Cleanup(a.resetHistoryTraversal)
	return a, callbacks, source, target
}

func TestMessageCommitPagesWithinCurrentScope(t *testing.T) {
	a, callbacks, source, target := pagedMessageFixture(t)
	beforeRef, beforeTips := a.historyRef, strings.Join(a.historyTips, " ")
	tt := messageNavigationTester(a, source)
	chooseMessageCommit(t, tt, "See "+target, "Jump to Commit · "+target)
	historyCompletion(t, callbacks)()
	if a.pendingJumpSHA != target || a.selectedSHA != source.SHA {
		t.Fatal("unloaded commit did not enter guarded paging")
	}
	for a.pendingJumpSHA != "" || a.historyLoadingMore {
		historyCompletion(t, callbacks)()
	}
	if a.selectedSHA != target || !a.revealSelection || !a.canNavigateHistory(-1) || a.commitFields.Summary != "keep paging draft" || a.historyRef != beforeRef || strings.Join(a.historyTips, " ") != beforeTips || len(a.notices[toastMain]) != 0 {
		t.Fatal("paged message jump failed or changed traversal")
	}
	a.navigateHistory(-1)
	if a.selectedSHA != source.SHA {
		t.Fatal("paged jump could not return")
	}
}

func TestMessageCommitPagingReadFailureAndStaleCompletion(t *testing.T) {
	for _, change := range []string{"none", "page", "source", "query", "dialog", "new menu", "ref scope", "pinned"} {
		t.Run(change, func(t *testing.T) {
			a, callbacks, source, target := pagedMessageFixture(t)
			tt := messageNavigationTester(a, source)
			chooseMessageCommit(t, tt, "See "+target, "Jump to Commit · "+target)
			resolve := historyCompletion(t, callbacks)
			// ResolveRef has completed. Hide only this disposable repository's metadata
			// before the genuine history reader starts, to obtain a real Git read error.
			gitDir := filepath.Join(a.repo.Root, ".git")
			hidden := filepath.Join(a.repo.Root, ".git-owned-read-error")
			if err := os.Rename(gitDir, hidden); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := os.Rename(hidden, gitDir); err != nil {
					t.Error(err)
				}
			}()
			resolve()
			done := historyCompletion(t, callbacks)
			switch change {
			case "page":
				a.page = "changes"
			case "source":
				a.selectedSHA = a.commits[1].SHA
			case "query":
				a.historyQuery = "new query"
			case "dialog":
				a.confirm.open = true
			case "new menu":
				r, ok := tt.Find("See " + target)
				if !ok {
					t.Fatal("source message disappeared")
				}
				tt.RightClickAt(r.X+5, r.Y+r.H/2)
			case "ref scope":
				a.historyRef = historyRef{Kind: historyRefLocal, Name: "different"}
			case "pinned":
				a.historyPinned = "changed"
			}
			selected, query := a.selectedSHA, a.historyQuery
			done()
			if a.selectedSHA != selected || a.historyQuery != query || a.pendingJumpSHA != "" || a.historyNavigation.pending != nil {
				t.Fatal("failed read changed current selection or retained stale jump")
			}
			notices := a.notices[toastMain]
			if change != "none" {
				if len(notices) != 0 {
					t.Fatal("stale read posted navigation feedback into new view")
				}
				return
			}
			if a.historyPageErr == "" || len(notices) != 1 || !strings.Contains(notices[0].description, target) || !strings.Contains(notices[0].description, a.t("commitNavigationBlocked")) {
				t.Fatal("real read failure lost its complete target or was reported as EOF")
			}
		})
	}
}
