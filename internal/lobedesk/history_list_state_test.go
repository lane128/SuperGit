package lobedesk

import (
	"testing"

	"supergit/internal/gitrepo"
)

// The list shows one of seven states. Their order is the rule: a failed load
// prints its own line and outranks the placeholders under it; a walk still out
// hides the previous ref's rows; a file whose history is still loading is not
// a file without history. Adding a state means fitting it into this chain.
func TestHistoryListStatePriority(t *testing.T) {
	loaded := []gitrepo.Commit{{SHA: "aaaa", Subject: "first"}}
	for _, test := range []struct {
		name string
		app  *app
		rows int
		want historyListState
	}{
		{"a load error outranks everything", &app{historyErr: "boom", historyReloading: true, fileHistoryLoading: true}, 0, historyListError},
		{"a file history error is an error too", &app{fileHistoryErr: "boom", historyReloading: true}, 0, historyListError},
		{"a stale walk outranks the loading placeholders", &app{historyReloading: true, fileHistoryLoading: true}, 0, historyListReloading},
		{"a file history still loading", &app{fileHistoryLoading: true}, 0, historyListFileLoading},
		{"a file no commit touched", &app{commits: loaded, fileFilter: "pkg/x.go"}, 0, historyListNoFileHistory},
		{"a repository with no commits", &app{}, 0, historyListEmpty},
		{"a search that hides every row", &app{commits: loaded}, 0, historyListNoMatch},
		{"rows to draw", &app{commits: loaded}, 1, historyListRows},
	} {
		if got := test.app.historyListState(test.rows); got != test.want {
			t.Errorf("%s: historyListState(%d) = %v; want %v", test.name, test.rows, got, test.want)
		}
	}
}
