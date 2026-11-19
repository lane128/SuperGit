package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestHistorySearchPlaceholderPrecedence(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		for _, scenario := range []string{"error", "reload", "file-loading", "empty", "file-empty", "file-error"} {
			t.Run(string(language)+"/"+scenario, func(t *testing.T) {
				a := worktreePageApp()
				a.locale, a.historyQuery, a.historyMore = language, "no match", true
				expected := ""
				switch scenario {
				case "error":
					a.commits = nil
					a.historyErr = "read failed"
					expected = "read failed"
				case "reload":
					a.historyReloading = true
					expected = a.t("rereadingHistory")
				case "file-loading":
					a.fileFilter = "owned.txt"
					a.fileHistoryLoading = true
					expected = a.t("diffLoading")
				case "empty":
					a.commits = nil
					expected = a.t("noHistory")
				case "file-empty":
					a.fileFilter = "owned.txt"
					expected = a.t("noFileHistory")
				case "file-error":
					a.fileFilter = "owned.txt"
					a.fileHistoryErr = "read failed"
					expected = a.t("historyLoadFailed") + ": read failed"
				}
				tt := pageTester(a, a.commitList)
				if !tt.HasText(expected) || tt.HasText(a.t("searchNoMatch")) || tt.HasText(a.t("searchNoMatchPartial")) {
					t.Fatal("search feedback replaced higher-priority list state")
				}
			})
		}
	}
}

func TestHistorySearchPartialNoMatch(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		for _, scenario := range []string{"ordinary-more", "ordinary-loading", "ordinary-paused", "ordinary-page-error", "ordinary-complete", "file-more", "file-loading", "file-paused", "file-error", "file-complete"} {
			t.Run(string(language)+"/"+scenario, func(t *testing.T) {
				a := worktreePageApp()
				a.locale = language
				a.historyQuery = "not-a-matching-commit"
				a.commitText = "draft\n\nbody"
				a.amend = true
				partial := true
				switch scenario {
				case "ordinary-more":
					a.historyMore = true
				case "ordinary-loading":
					a.historyMore = true
					a.historyLoadingMore = true
				case "ordinary-paused":
					a.historyMore = true
					a.historyPaused = true
				case "ordinary-page-error":
					a.historyMore = true
					a.historyPageErr = "read failed"
				case "ordinary-complete":
					partial = false
					a.fileHistoryMore = true
					a.fileHistoryLoading, a.fileHistoryPaused, a.fileHistoryErr = true, true, "inactive source error"
				default:
					a.fileFilter = "owned.txt"
					a.fileCommits = a.commits
					a.historyMore = true // Ordinary traversal does not describe file-history completeness.
					switch scenario {
					case "file-more":
						a.fileHistoryMore = true
					case "file-loading":
						a.fileHistoryLoading = true
					case "file-paused":
						a.fileHistoryPaused = true
					case "file-error":
						a.fileHistoryErr = "read failed"
					case "file-complete":
						partial = false
						a.historyLoadingMore, a.historyPaused, a.historyPageErr = true, true, "inactive source error"
					}
				}
				beforeSHA, beforeQuery, beforeFilter := a.selectedSHA, a.historyQuery, a.fileFilter
				tt := pageTester(a, a.commitList)
				partialText := "No matches in loaded history. History is incomplete."
				if language == localeChinese {
					partialText = "已读取的历史中暂无匹配，历史尚未读完。"
				}
				if partial {
					if !tt.HasText(partialText) || tt.HasText(a.t("searchNoMatch")) {
						t.Fatal("incomplete search presented a final no-match result")
					}
				} else if !tt.HasText(a.t("searchNoMatch")) || tt.HasText(partialText) {
					t.Fatal("complete search presented an incomplete result")
				}
				if _, ok := tt.AccessByRole(ui.RoleOption, a.changesLabel(1)); !ok {
					t.Fatal("working tree row missing")
				}
				if a.selectedSHA != beforeSHA || a.historyQuery != beforeQuery || a.fileFilter != beforeFilter || a.commitText != "draft\n\nbody" || !a.amend {
					t.Fatal("render changed browsing or draft state")
				}
			})
		}
	}
}
