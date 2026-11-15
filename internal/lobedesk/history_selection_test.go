package lobedesk

import (
	"strings"
	"testing"

	"supergit/internal/gitrepo"
)

// The details pane describes the row the reader picked, and the history page
// can be listing a file's history instead of the loaded window: a commit that
// touched the file but sits outside the newest 200 still has to be found, or
// the pane shows "select a commit" while the row is selected on the left.
func TestSelectedCommitFollowsTheHistoryPageList(t *testing.T) {
	a := &app{
		commits:     []gitrepo.Commit{{SHA: "1111", Subject: "in the loaded window"}},
		fileCommits: []gitrepo.Commit{{SHA: "2222", Subject: "only in the file's history"}},
		fileFilter:  "pkg/x.go",
		selectedSHA: "2222",
	}
	commit, ok := a.selectedCommit()
	if !ok {
		t.Fatalf("selectedCommit did not find the row the file history is showing")
	}
	if commit.SHA != "2222" {
		t.Errorf("selectedCommit = %q; want the file history's row %q", commit.SHA, "2222")
	}
}

// The graph is laid out for the rows the list draws, which are the page's
// source narrowed by the search. The row index handed to the details pane has
// to count those rows; counting the unfiltered window instead points the color
// lookup at a different commit's line.
func TestSelectedCommitIndexCountsTheRowsTheListDraws(t *testing.T) {
	a := &app{
		commits: []gitrepo.Commit{
			{SHA: "aaaa", Subject: "first"},
			{SHA: "bbbb", Subject: "second"},
			{SHA: "cccc", Subject: "third"},
		},
		historyQuery: "third",
		selectedSHA:  "cccc",
	}
	if got := a.selectedCommitIndex(); got != 0 {
		t.Errorf("selectedCommitIndex = %d; want 0, the row the filtered list draws", got)
	}
}

// "Select a commit to inspect it" is the wrong sentence for a repository that
// has no commits, and so is "details appear once there is a commit" for a file
// that no commit touched, or for a history that failed to load.
func TestDetailsPlaceholderOnlyClaimsThereAreNoCommitsWhenThereAreNone(t *testing.T) {
	loaded := []gitrepo.Commit{{SHA: "aaaa"}}
	for _, test := range []struct {
		name string
		app  *app
		want string
	}{
		{"repository with no commits", &app{}, "detailsNoCommits"},
		{"file with no history", &app{commits: loaded, fileFilter: "pkg/x.go"}, "selectCommit"},
		{"history that failed to load", &app{historyErr: "boom"}, "selectCommit"},
		{"commits loaded, none picked", &app{commits: loaded}, "selectCommit"},
	} {
		if got := test.app.detailsPlaceholderKey(); got != test.want {
			t.Errorf("%s: detailsPlaceholderKey = %q; want %q", test.name, got, test.want)
		}
	}
}

// The details pane has the width for a longer label than a row, but it still
// has to keep the tail: two dependabot refs that share a prefix are only told
// apart by the end of the name.
func TestDetailRefBadgesKeepTheirTails(t *testing.T) {
	if refCapsuleDetailRunes <= refBadgeRunes {
		t.Fatalf("details budget %d is not longer than the row's %d", refCapsuleDetailRunes, refBadgeRunes)
	}
	long := "origin/dependabot/go_modules/create-pull-request-8.1.2"
	elided := elideMiddle(long, refCapsuleDetailRunes)
	if !strings.HasSuffix(elided, "8.1.2") {
		t.Errorf("elideMiddle(%q) = %q; the tail that tells the refs apart is gone", long, elided)
	}
	if len([]rune(elided)) > refCapsuleDetailRunes {
		t.Errorf("elideMiddle(%q) = %q; it is longer than the %d-rune budget", long, elided, refCapsuleDetailRunes)
	}
}
