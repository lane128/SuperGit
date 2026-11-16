package lobedesk

import (
	"slices"
	"testing"
	"time"

	"supergit/internal/gitrepo"
)

func TestSplitRepoPath(t *testing.T) {
	tests := []struct {
		path, name, dir string
	}{
		{"README.md", "README.md", ""},
		{"cmd/supergit/main.go", "main.go", "cmd/supergit"},
		{"a/b/", "b", "a"},
		{"", "", ""},
	}
	for _, tt := range tests {
		name, dir := splitRepoPath(tt.path)
		if name != tt.name || dir != tt.dir {
			t.Errorf("splitRepoPath(%q) = %q, %q; want %q, %q", tt.path, name, dir, tt.name, tt.dir)
		}
	}
}

func TestStatusPresentation(t *testing.T) {
	tests := []struct {
		entry  gitrepo.Entry
		staged bool
		code   string
		color  string
	}{
		{gitrepo.Entry{Path: "new.txt", Index: '?', Worktree: '?'}, false, "?", "info"},
		{gitrepo.Entry{Path: "add.txt", Index: 'A', Worktree: '.'}, true, "A", "success"},
		{gitrepo.Entry{Path: "gone.txt", Index: '.', Worktree: 'D'}, false, "D", "error"},
		{gitrepo.Entry{Path: "conflict.txt", Index: 'U', Worktree: 'U', Conflict: true}, true, "!", "error"},
		{gitrepo.Entry{Path: "mod.txt", Index: 'M', Worktree: 'M'}, false, "M", "warning"},
	}
	for _, tt := range tests {
		if got := statusCode(tt.entry, tt.staged); got != tt.code {
			t.Errorf("statusCode(%q, %t) = %q; want %q", tt.entry.Path, tt.staged, got, tt.code)
		}
		if got := statusTagColor(tt.entry, tt.staged); got != tt.color {
			t.Errorf("statusTagColor(%q, %t) = %q; want %q", tt.entry.Path, tt.staged, got, tt.color)
		}
	}
}

func TestFilterCommits(t *testing.T) {
	commits := []gitrepo.Commit{
		{SHA: "abcdef123456", Subject: "Fix staging", Author: "Ada", Refs: "main", Day: "2026-11-08"},
		{SHA: "123456abcdef", Subject: "Add history", Author: "Ben", Refs: "origin/feature", Day: "2026-11-09"},
	}
	// A day matches its own commits, which is what the commit menu's "filter to
	// this day" writes into the search box.
	for _, query := range []string{"staging", "ADA", "origin/feature", "123456abc", "2026-11-08"} {
		if got := filterCommits(commits, query); len(got) != 1 {
			t.Errorf("filterCommits(%q) returned %d commits; want 1", query, len(got))
		}
	}
	if got := filterCommits(commits, "2026-11"); len(got) != 2 {
		t.Errorf("filterCommits(month) returned %d commits; want 2", len(got))
	}
	if got := filterCommits(commits, "missing"); len(got) != 0 {
		t.Errorf("filterCommits(missing) returned %d commits; want 0", len(got))
	}
	if got := filterCommits(commits, ""); len(got) != 2 {
		t.Errorf("filterCommits(empty) returned %d commits; want 2", len(got))
	}
}

// A range is the one date filter a substring cannot express. Both ends are
// inclusive, a bound may be a day, a month or a year, and a missing side
// leaves that end open. Text that only looks like a range stays a text
// search, and a commit whose date could not be read has no day to place.
func TestADayRangeNarrowsTheHistory(t *testing.T) {
	commits := []gitrepo.Commit{
		{SHA: "a", Subject: "October", Day: "2026-10-31"},
		{SHA: "b", Subject: "First", Day: "2026-11-08"},
		{SHA: "c", Subject: "Middle", Day: "2026-11-09"},
		{SHA: "d", Subject: "Today", Day: "2026-11-15"},
		{SHA: "e", Subject: "Later", Day: "2026-11-16"},
		{SHA: "f", Subject: "Undated"},
	}
	tests := []struct {
		query string
		want  []string
	}{
		{"2026-11-08..2026-11-15", []string{"b", "c", "d"}},
		{"2026-11-08..2026-11-08", []string{"b"}},
		{"2026-11..", []string{"b", "c", "d", "e"}},
		{"..2026-10", []string{"a"}},
		{"2026-11..2026-11", []string{"b", "c", "d", "e"}},
		{"2026..2026", []string{"a", "b", "c", "d", "e"}},
		{"..", nil},
		{"Middle..today", nil},
	}
	for _, tt := range tests {
		var got []string
		for _, commit := range filterCommits(commits, tt.query) {
			got = append(got, commit.SHA)
		}
		if !slices.Equal(got, tt.want) {
			t.Errorf("filterCommits(%q) = %v; want %v", tt.query, got, tt.want)
		}
	}
}

func TestDiffStats(t *testing.T) {
	diff := gitrepo.Diff{
		Hunks: []gitrepo.Hunk{{Header: "@@ -1 +1 @@"}},
		Lines: []gitrepo.DiffLine{
			{Kind: "meta"},
			{Kind: "hunk"},
			{Kind: "del"},
			{Kind: "add"},
			{Kind: "add"},
		},
	}
	additions, deletions, hunks := diffStats(diff)
	if additions != 2 || deletions != 1 || hunks != 1 {
		t.Fatalf("diffStats() = %d, %d, %d; want 2, 1, 1", additions, deletions, hunks)
	}
}

func TestCommitRefs(t *testing.T) {
	got := commitRefs("HEAD -> main, tag: v1.0, origin/main")
	want := []string{"main", "tag: v1.0", "origin/main"}
	if len(got) != len(want) {
		t.Fatalf("commitRefs() = %#v; want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("commitRefs()[%d] = %q; want %q", i, got[i], want[i])
		}
	}
	if refs := commitRefs("  "); refs != nil {
		t.Fatalf("commitRefs(blank) = %#v; want nil", refs)
	}
}

// A row prints three refs and folds the rest. Which three follow the page's
// context, but they print in Git's order. Whatever the choice, no ref may
// vanish between the row and the +N capsule.
func TestRowRefs(t *testing.T) {
	for _, test := range []struct {
		count         int
		shown, folded int
	}{
		{0, 0, 0},
		{1, 1, 0},
		{3, 3, 0},
		{4, 3, 1},
		{5, 3, 2},
		{7, 3, 4},
	} {
		refs := make([]string, test.count)
		for i := range refs {
			refs[i] = "ref-" + itoa(i)
		}
		shown, folded := rowRefs(refs, historyRef{}, "")
		if len(shown) != test.shown || len(folded) != test.folded {
			t.Errorf("rowRefs(%d refs) = %d shown, %d folded; want %d, %d",
				test.count, len(shown), len(folded), test.shown, test.folded)
			continue
		}
		if len(shown)+len(folded) != test.count {
			t.Errorf("rowRefs(%d refs) lost a ref between the row and +N", test.count)
		}
		if len(refs) > 0 && len(shown) > 0 && shown[0] != refs[0] {
			t.Errorf("rowRefs(%d refs) should keep the first ref when nothing is focused", test.count)
		}
	}
}

// Choosing by context must not move the capsules that were already on the row.
// The filtered ref and HEAD's branch are picked into the three either way, but
// the shown three keep Git's order, so filtering to b3 changes the last
// capsule from v2 to b3 instead of sliding main and v1 one slot right.
func TestRowRefsKeepsGitOrderWhileChoosingByContext(t *testing.T) {
	refs := []string{"main", "tag: v1", "tag: light1", "b1", "b2", "b3"}
	for _, test := range []struct {
		name  string
		focus historyRef
		head  string
		want  []string
	}{
		{"no context", historyRef{}, "", []string{"main", "tag: v1", "tag: light1"}},
		{"filtered tag", historyRef{Kind: historyRefTag, Name: "v1"}, "", []string{"main", "tag: v1", "tag: light1"}},
		{"filtered branch joins last", historyRef{Kind: historyRefLocal, Name: "b3"}, "", []string{"main", "tag: v1", "b3"}},
		{"head branch", historyRef{}, "main", []string{"main", "tag: v1", "tag: light1"}},
		{"filter then head", historyRef{Kind: historyRefTag, Name: "light1"}, "main", []string{"main", "tag: v1", "tag: light1"}},
	} {
		shown, folded := rowRefs(refs, test.focus, test.head)
		if !slices.Equal(shown, test.want) {
			t.Errorf("%s: shown = %v; want %v", test.name, shown, test.want)
		}
		seen := map[string]int{}
		for _, ref := range append(append([]string{}, shown...), folded...) {
			seen[ref]++
		}
		for _, ref := range refs {
			if seen[ref] != 1 {
				t.Errorf("%s: ref %q appears %d times across the row and +N", test.name, ref, seen[ref])
			}
		}
		// The capsules keep Git's order, so the shown refs are a subsequence
		// of the row's refs in their original positions.
		next := 0
		for _, ref := range refs {
			if next < len(shown) && shown[next] == ref {
				next++
			}
		}
		if next != len(shown) {
			t.Errorf("%s: shown %v is not in Git order", test.name, shown)
		}
	}
}

func TestHeadBranch(t *testing.T) {
	for _, test := range []struct{ refs, want string }{
		{"HEAD -> main, origin/main", "main"},
		{"tag: v1, HEAD -> release/1.0", "release/1.0"},
		{"origin/main", ""},
		{"HEAD", ""},
		{"", ""},
	} {
		if got := headBranch(test.refs); got != test.want {
			t.Errorf("headBranch(%q) = %q; want %q", test.refs, got, test.want)
		}
	}
}

func TestLocaleLabels(t *testing.T) {
	zh := &app{locale: localeChinese}
	if got := zh.changesLabel(2); got != "2 个改动" {
		t.Errorf("changesLabel(2) = %q", got)
	}
	en := &app{locale: localeEnglish}
	if got := en.changesLabel(1); got != "1 change" {
		t.Errorf("changesLabel(1) = %q", got)
	}
	if got := en.changesLabel(2); got != "2 changes" {
		t.Errorf("changesLabel(2) = %q", got)
	}
	if got := en.diffSummary(2, 1, 1); got != "2 additions · 1 deletion · 1 hunk" {
		t.Errorf("diffSummary() = %q", got)
	}
}

// Both tables carry the same keys: t returns the key itself when a language
// lacks one, so a label added in one language and forgotten in the other shows
// up on screen as a raw key such as "localBranches".
func TestMessageTablesCoverTheSameKeys(t *testing.T) {
	zh, en := messages[localeChinese], messages[localeEnglish]
	if len(zh) == 0 || len(en) == 0 {
		t.Fatal("a locale table is missing")
	}
	for key := range zh {
		if _, ok := en[key]; !ok {
			t.Errorf("%q is in the Chinese table but missing from the English one", key)
		}
	}
	for key := range en {
		if _, ok := zh[key]; !ok {
			t.Errorf("%q is in the English table but missing from the Chinese one", key)
		}
	}
	// The palette row resolves its labels through the same tables.
	for _, th := range appThemes {
		if _, ok := zh[th.label]; !ok {
			t.Errorf("palette %q: label key %q is missing from the Chinese table", th.key, th.label)
		}
		if _, ok := en[th.label]; !ok {
			t.Errorf("palette %q: label key %q is missing from the English table", th.key, th.label)
		}
	}
}

// A remote row names the local branch that tracks it, mirroring the upstream
// line under a local row; with no tracking branch it shows no second line at
// all rather than repeating the ref the first line already spells out.
func TestTrackingBranchFindsTheLocalCounterpart(t *testing.T) {
	a := &app{branches: []gitrepo.Branch{
		{Name: "feature/login", Upstream: "origin/feature/login"},
		{Name: "main", Upstream: "origin/main"},
		{Name: "local-only"},
	}}
	for ref, want := range map[string]string{
		"origin/main":            "main",
		"origin/feature/login":   "feature/login",
		"origin/local-only":      "",
		"origin/not-in-the-repo": "",
	} {
		if got := a.trackingBranch(ref); got != want {
			t.Errorf("trackingBranch(%q) = %q; want %q", ref, got, want)
		}
	}
}

// Every time in the app is the exact local moment, to the second. The list,
// the details pane, the stash rows and the blame gutter all call this, so one
// test pins the shape they share.
func TestFormatDateTime(t *testing.T) {
	value := "2026-11-12T08:30:05+08:00"
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := formatDateTime(value), parsed.Local().Format("2006-01-02 15:04:05"); got != want {
		t.Errorf("formatDateTime() = %q; want %q", got, want)
	}
	if got := formatDateTime("2020-01-02T03:04:05Z"); len(got) != len("2006-01-02 15:04:05") {
		t.Errorf("formatDateTime() = %q; want a full timestamp", got)
	}
	if got := formatDateTime(""); got != "" {
		t.Errorf("formatDateTime(empty) = %q", got)
	}
	if got := formatDateTime("not a date"); got != "not a date" {
		t.Errorf("formatDateTime(unparseable) = %q; want it shown as it came", got)
	}
}

func TestWorktreePathFromParent(t *testing.T) {
	tests := []struct {
		parent, current, root, want string
	}{
		{"/tmp", "/tmp/project-worktree", "/tmp/project", "/tmp/project-worktree"},
		{"/tmp", "", "/tmp/project", "/tmp/project-worktree"},
		{"/tmp", "/", "/tmp/project", "/tmp/project-worktree"},
		{"/tmp", "/tmp/feature/a", "/tmp/project", "/tmp/a"},
	}
	for _, tt := range tests {
		if got := worktreePathFromParent(tt.parent, tt.current, tt.root); got != tt.want {
			t.Errorf("worktreePathFromParent(%q, %q, %q) = %q; want %q", tt.parent, tt.current, tt.root, got, tt.want)
		}
	}
}

func TestDiffLineText(t *testing.T) {
	tests := []struct {
		kind, text, want string
	}{
		{"add", "+line 05", "line 05"},
		{"del", "-line 05", "line 05"},
		{"context", " line 05", "line 05"},
		{"add", "+", ""},
		{"meta", "diff --git a/x b/x", "diff --git a/x b/x"},
	}
	for _, tt := range tests {
		line := gitrepo.DiffLine{Kind: tt.kind, Text: tt.text}
		if got := diffLineText(line); got != tt.want {
			t.Errorf("diffLineText(%q, %q) = %q; want %q", tt.kind, tt.text, got, tt.want)
		}
	}
}

func TestDiffLinesText(t *testing.T) {
	diff := gitrepo.Diff{Lines: []gitrepo.DiffLine{
		{Kind: "meta", Text: "diff --git a/x b/x", Hunk: -1},
		{Kind: "hunk", Text: "@@ -1,2 +1,2 @@", Hunk: 0},
		{Kind: "del", Text: "-one", Hunk: 0},
		{Kind: "add", Text: "+ONE", Hunk: 0},
		{Kind: "context", Text: " two", Hunk: 0},
		{Kind: "add", Text: "+three", Hunk: 0},
	}}
	if got, want := diffLinesText(diff, []int{2, 3, 5}), "one\nONE\nthree"; got != want {
		t.Errorf("diffLinesText = %q; want %q", got, want)
	}
	if got, want := diffLinesText(diff, []int{3}), "ONE"; got != want {
		t.Errorf("diffLinesText single = %q; want %q", got, want)
	}
	if got := diffLinesText(diff, []int{-1, 99}); got != "" {
		t.Errorf("diffLinesText out of range = %q; want empty", got)
	}
}

func TestDiffHunkText(t *testing.T) {
	diff := gitrepo.Diff{Lines: []gitrepo.DiffLine{
		{Kind: "meta", Text: "diff --git a/x b/x", Hunk: -1},
		{Kind: "hunk", Text: "@@ -1,2 +1,2 @@", Hunk: 0},
		{Kind: "del", Text: "-one", Hunk: 0},
		{Kind: "add", Text: "+ONE", Hunk: 0},
		{Kind: "context", Text: " two", Hunk: 0},
		{Kind: "hunk", Text: "@@ -8,2 +8,2 @@", Hunk: 1},
		{Kind: "add", Text: "+eight", Hunk: 1},
	}}
	want := "@@ -1,2 +1,2 @@\n-one\n+ONE\n two"
	if got := diffHunkText(diff, 0); got != want {
		t.Errorf("diffHunkText(0) = %q; want %q", got, want)
	}
	if got, want := diffHunkText(diff, 1), "@@ -8,2 +8,2 @@\n+eight"; got != want {
		t.Errorf("diffHunkText(1) = %q; want %q", got, want)
	}
	if got := diffHunkText(diff, 9); got != "" {
		t.Errorf("diffHunkText(9) = %q; want empty", got)
	}
}

func TestHunkChangedLines(t *testing.T) {
	diff := gitrepo.Diff{Lines: []gitrepo.DiffLine{
		{Kind: "meta", Text: "diff --git a/x b/x", Hunk: -1},
		{Kind: "hunk", Text: "@@ -1,2 +1,2 @@", Hunk: 0},
		{Kind: "del", Text: "-one", Hunk: 0},
		{Kind: "add", Text: "+ONE", Hunk: 0},
		{Kind: "context", Text: " two", Hunk: 0},
		{Kind: "hunk", Text: "@@ -8,2 +8,2 @@", Hunk: 1},
		{Kind: "add", Text: "+eight", Hunk: 1},
	}}
	if got := hunkChangedLines(diff, 0); !slices.Equal(got, []int{2, 3}) {
		t.Errorf("hunkChangedLines(0) = %v; want [2 3]", got)
	}
	if got := hunkChangedLines(diff, 1); !slices.Equal(got, []int{6}) {
		t.Errorf("hunkChangedLines(1) = %v; want [6]", got)
	}
	if got := hunkChangedLines(diff, 9); len(got) != 0 {
		t.Errorf("hunkChangedLines(9) = %v; want empty", got)
	}
}

func TestCommitLineTargets(t *testing.T) {
	a := &app{commitSel: map[int]bool{9: true, 2: true, 5: true}}
	if got := a.commitLineTargets(5); !slices.Equal(got, []int{2, 5, 9}) {
		t.Errorf("commitLineTargets in selection = %v; want [2 5 9]", got)
	}
	if got := a.commitLineTargets(3); !slices.Equal(got, []int{3}) {
		t.Errorf("commitLineTargets outside selection = %v; want [3]", got)
	}
	a.commitSel = nil
	if got := a.commitLineTargets(4); !slices.Equal(got, []int{4}) {
		t.Errorf("commitLineTargets empty selection = %v; want [4]", got)
	}
}

func TestAutoScrollStep(t *testing.T) {
	const top, bottom = 100, 500
	tests := []struct {
		y, want float32
	}{
		{300, 0},   // away from both edges
		{100, -24}, // the top edge
		{50, -24},  // above the view keeps the top speed
		{118, -12}, // half way into the band
		{136, 0},   // the inner side of the band
		{500, 24},  // the bottom edge
		{600, 24},  // below the view keeps the bottom speed
		{482, 12},  // half way into the band
		{464, 0},   // the inner side of the band
	}
	for _, tt := range tests {
		if got := autoScrollStep(tt.y, top, bottom); absFloat(got-tt.want) > 0.01 {
			t.Errorf("autoScrollStep(%v) = %v; want %v", tt.y, got, tt.want)
		}
	}
	if got := autoScrollStep(50, 100, 100); got != 0 {
		t.Errorf("autoScrollStep in an empty view = %v; want 0", got)
	}
}

func absFloat(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func TestMoveCommitLineSelection(t *testing.T) {
	diff := gitrepo.Diff{Lines: []gitrepo.DiffLine{
		{Kind: "meta", Hunk: -1},
		{Kind: "hunk", Hunk: 0},
		{Kind: "del", Hunk: 0},
		{Kind: "add", Hunk: 0},
		{Kind: "context", Hunk: 0},
		{Kind: "meta", Hunk: -1},
		{Kind: "add", Hunk: 1},
	}}
	a := &app{commitDiff: diff, commitSections: []commitDiffSection{{Path: "a.go", Rows: []int{1, 2, 3, 4, 6}}}, commitSel: map[int]bool{2: true}, commitAnchor: 2, commitCursor: 2, commitPath: "a.go"}
	a.detailTabs().Open(detailFileKey{Path: "a.go"})
	if !a.moveCommitLineSelection(1, false) {
		t.Fatal("plain move did not take")
	}
	if !a.commitSel[3] || len(a.commitSel) != 1 || a.commitAnchor != 3 {
		t.Errorf("plain move: sel=%v anchor=%d; want only 3", a.commitSel, a.commitAnchor)
	}
	a.commitSel, a.commitAnchor, a.commitCursor = map[int]bool{2: true}, 2, 2
	a.moveCommitLineSelection(2, true)
	if !a.commitSel[2] || !a.commitSel[3] || !a.commitSel[4] || len(a.commitSel) != 3 {
		t.Errorf("extend: sel=%v; want 2,3,4", a.commitSel)
	}
	a.commitSel, a.commitAnchor, a.commitCursor = nil, -1, -1
	if a.moveCommitLineSelection(1, false) {
		t.Error("move without a selection should do nothing")
	}
}

func TestLineIndexesWith(t *testing.T) {
	diff := gitrepo.Diff{Lines: []gitrepo.DiffLine{
		{Kind: "meta"},
		{Kind: "hunk"},
		{Kind: "del"},
		{Kind: "context"},
		{Kind: "add"},
		{Kind: "meta"},
	}}
	if got := changedLineIndexes(diff); !slices.Equal(got, []int{2, 4}) {
		t.Errorf("changedLineIndexes = %v; want [2 4]", got)
	}
	if got := commitLineIndexes(diff); !slices.Equal(got, []int{2, 3, 4}) {
		t.Errorf("commitLineIndexes = %v; want [2 3 4]", got)
	}
}

func TestInvertIndexes(t *testing.T) {
	candidates := []int{2, 4, 6}
	if got := invertIndexes(candidates, map[int]bool{4: true}); !slices.Equal(got, []int{2, 6}) {
		t.Errorf("invertIndexes one chosen = %v; want [2 6]", got)
	}
	if got := invertIndexes(candidates, nil); !slices.Equal(got, candidates) {
		t.Errorf("invertIndexes empty selection = %v; want %v", got, candidates)
	}
	if got := invertIndexes(candidates, map[int]bool{2: true, 4: true, 6: true}); len(got) != 0 {
		t.Errorf("invertIndexes all chosen = %v; want empty", got)
	}
}

func TestSelectionSetAndEdges(t *testing.T) {
	if selectionSet(nil) != nil {
		t.Error("selectionSet(nil) should be nil")
	}
	set := selectionSet([]int{5, 2})
	if !set[2] || !set[5] || len(set) != 2 {
		t.Errorf("selectionSet = %v; want 2 and 5", set)
	}
	if anchor, cursor := selectionEdges([]int{5, 2}); anchor != 2 || cursor != 5 {
		t.Errorf("selectionEdges = %d, %d; want 2, 5", anchor, cursor)
	}
	if anchor, cursor := selectionEdges(nil); anchor != -1 || cursor != -1 {
		t.Errorf("selectionEdges(nil) = %d, %d; want -1, -1", anchor, cursor)
	}
}

func TestCanCommitOperation(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{}, commitText: "merge it", operation: gitrepo.Operation{Kind: "merge"}, commitIdentity: readyCommitIdentity("")}
	if !a.canCommit() {
		t.Error("a finished merge should be committable without staged files")
	}
	a.status = gitrepo.Status{Entries: []gitrepo.Entry{{Path: "x", Conflict: true}}}
	if a.canCommit() {
		t.Error("unresolved conflicts must block the commit")
	}
	a.status = gitrepo.Status{}
	a.operation.Kind = "rebase"
	if a.canCommit() {
		t.Error("a rebase continues through its own command, not a commit")
	}
	a.status = gitrepo.Status{Entries: []gitrepo.Entry{{Path: "x", Index: 'M', Worktree: '.'}}}
	if a.canCommit() {
		t.Error("staged files must not enable a commit during a rebase")
	}
	a.amend = true
	if a.canCommit() {
		t.Error("amend must not enable a commit during a rebase")
	}
	a.amend = false
	a.status = gitrepo.Status{}
	if commitOperation("") || commitOperation("rebase") {
		t.Error("commitOperation should accept only merge, cherry-pick and revert")
	}
	if !commitOperation("cherry-pick") || !commitOperation("revert") {
		t.Error("commitOperation should accept cherry-pick and revert")
	}
}

func TestBranchPushTarget(t *testing.T) {
	a := &app{remotes: []gitrepo.Remote{{Name: "origin"}, {Name: "upstream"}}}
	if remote, destination := a.branchPushTarget(gitrepo.Branch{Name: "topic"}); remote != "origin" || destination != "topic" {
		t.Errorf("no upstream = %q, %q; want origin, topic", remote, destination)
	}
	if remote, destination := a.branchPushTarget(gitrepo.Branch{Name: "topic", Upstream: "upstream/topic"}); remote != "upstream" || destination != "topic" {
		t.Errorf("tracked branch = %q, %q; want upstream, topic", remote, destination)
	}
	if remote, destination := a.branchPushTarget(gitrepo.Branch{Name: "release", Upstream: "origin/release/v1"}); remote != "origin" || destination != "release/v1" {
		t.Errorf("slashed upstream = %q, %q; want origin, release/v1", remote, destination)
	}
	if remote, destination := (&app{}).branchPushTarget(gitrepo.Branch{Name: "topic"}); remote != "" || destination != "" {
		t.Errorf("no remotes = %q, %q; want empty", remote, destination)
	}
}

func TestApplyOperationMessage(t *testing.T) {
	a := &app{operation: gitrepo.Operation{Kind: "merge"}}
	a.applyOperationMessage("Merge branch 'feature'")
	if a.commitText != "Merge branch 'feature'" || a.pendingMessage == "" {
		t.Fatalf("seed: text=%q pending=%q", a.commitText, a.pendingMessage)
	}
	a.applyOperationMessage("Merge branch 'feature'")
	if a.commitText != "Merge branch 'feature'" {
		t.Fatalf("reload changed the seeded text to %q", a.commitText)
	}
	a.applyOperationMessage("Merge branch 'renamed'")
	if a.commitText != "Merge branch 'renamed'" {
		t.Fatalf("a changed pending message did not replace the seeded text: %q", a.commitText)
	}
	a.commitText = "my own subject"
	a.applyOperationMessage("Merge branch 'other'")
	if a.commitText != "my own subject" {
		t.Fatalf("a typed message was overwritten: %q", a.commitText)
	}
	a.operation = gitrepo.Operation{}
	a.commitText = a.pendingMessage
	a.applyOperationMessage("")
	if a.commitText != "" || a.pendingMessage != "" {
		t.Fatalf("ending the operation left text=%q pending=%q", a.commitText, a.pendingMessage)
	}
	rebase := &app{operation: gitrepo.Operation{Kind: "rebase"}, commitText: "typed"}
	rebase.applyOperationMessage("rebase message")
	if rebase.commitText != "typed" {
		t.Fatalf("rebase seeded the field with %q", rebase.commitText)
	}
}

func TestBeginCommitDragResetsAnchor(t *testing.T) {
	a := &app{
		commitPressIndex: 5,
		commitAnchor:     2,
		commitSel:        map[int]bool{2: true, 3: true},
	}
	a.beginCommitDrag()
	if !a.commitDragging {
		t.Error("beginCommitDrag did not mark the drag")
	}
	if a.commitAnchor != 5 {
		t.Errorf("anchor = %d; want the pressed row 5", a.commitAnchor)
	}
	if len(a.commitSel) != 1 || !a.commitSel[5] {
		t.Errorf("selection = %v; want only row 5", a.commitSel)
	}
}

func TestPreferredCommitPath(t *testing.T) {
	files := []gitrepo.ChangedFile{{Path: "a.txt"}, {Path: "b.txt"}, {Path: "c.txt"}}
	if got := preferredCommitPath(files, "c.txt", "b.txt"); got != "c.txt" {
		t.Errorf("kept path = %q; want c.txt", got)
	}
	if got := preferredCommitPath(files, "", "b.txt"); got != "b.txt" {
		t.Errorf("filtered path = %q; want b.txt", got)
	}
	if got := preferredCommitPath(files, "", ""); got != "a.txt" {
		t.Errorf("fallback = %q; want a.txt", got)
	}
	if got := preferredCommitPath(files, "gone.txt", "gone2.txt"); got != "a.txt" {
		t.Errorf("missing preferred paths = %q; want a.txt", got)
	}
	if got := preferredCommitPath(nil, "", "x"); got != "" {
		t.Errorf("no files = %q; want empty", got)
	}
}

func TestHistorySelection(t *testing.T) {
	commits := []gitrepo.Commit{{SHA: "aaa"}, {SHA: "bbb"}}
	if got := historySelection(commits, "aaa"); got != "aaa" {
		t.Errorf("visible selection = %q; want aaa", got)
	}
	if got := historySelection(commits, "ccc"); got != "aaa" {
		t.Errorf("hidden selection = %q; want the first match aaa", got)
	}
	if got := historySelection(nil, "aaa"); got != "" {
		t.Errorf("no matches = %q; want empty", got)
	}
}

func TestCommitBody(t *testing.T) {
	tests := []struct {
		message, want string
	}{
		{"subject\n\nbody line one\nbody line two", "body line one\nbody line two"},
		{"subject only", ""},
		{"subject\n\n  padded body  \n", "padded body"},
		{"subject\n\n\n", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := commitBody(tt.message); got != tt.want {
			t.Errorf("commitBody(%q) = %q; want %q", tt.message, got, tt.want)
		}
	}
}

func TestCanToggleSelectedStage(t *testing.T) {
	a := &app{
		page:             "history",
		worktreeSelected: true,
		repo:             &gitrepo.Repository{},
		status:           gitrepo.Status{Entries: []gitrepo.Entry{{Path: "a.txt", Index: '.', Worktree: 'M'}}},
		selectedPath:     "a.txt",
	}
	if !a.canToggleSelectedStage() {
		t.Error("a selected unstaged file should toggle")
	}
	a.status.Entries[0].Conflict = true
	if a.canToggleSelectedStage() {
		t.Error("a conflicted file must not toggle through Space")
	}
	a.status.Entries[0].Conflict = false
	a.lineSel = map[int]bool{1: true}
	if a.canToggleSelectedStage() {
		t.Error("an active line selection must keep Space inert")
	}
	a.lineSel = nil
	a.worktreeSelected = false
	if a.canToggleSelectedStage() {
		t.Error("Space stages only while the working tree's row is selected")
	}
	a.worktreeSelected = true
	a.selectedPath = "missing.txt"
	if a.canToggleSelectedStage() {
		t.Error("no selected entry should do nothing")
	}
}

// The working tree's line handles select changes, not context: a drag that
// starts on a context row and crosses changed rows selects the changed rows in
// between, and a context row never joins the selection. The rule is written
// down in docs/selection-model.md.
func TestAContextAnchorDragSelectsOnlyChangedLines(t *testing.T) {
	a := &app{
		diff: gitrepo.Diff{Lines: []gitrepo.DiffLine{
			{Kind: "context"}, {Kind: "del"}, {Kind: "context"}, {Kind: "add"}, {Kind: "context"},
		}},
		lineAnchor: 0,
		lineCursor: 0,
	}
	a.dragToLine(3)
	if len(a.lineSel) != 2 || !a.lineSel[1] || !a.lineSel[3] {
		t.Fatalf("drag selection = %v; want the two changed rows only", a.lineSel)
	}
	// Resting on a context row does not extend the selection to it.
	a.dragToLine(4)
	if len(a.lineSel) != 2 || a.lineCursor != 3 {
		t.Fatalf("a context row joined the selection: %v (cursor %d)", a.lineSel, a.lineCursor)
	}
}

// The working tree's patch lives on the history page now, so a line selection
// made there has to walk with the arrows as it did on the workspace page. The
// commit patch keeps its own selection on the same page.
func TestTheWorkingTreeDiffLinesWalkOnTheHistoryPage(t *testing.T) {
	a := &app{
		page:             "history",
		worktreeSelected: true,
		repo:             &gitrepo.Repository{Root: t.TempDir()},
		selectedPath:     "a.go",
		diff: gitrepo.Diff{Lines: []gitrepo.DiffLine{
			{Kind: "del"}, {Kind: "add"}, {Kind: "ctx"}, {Kind: "add"},
		}},
		lineSel:    map[int]bool{0: true},
		lineAnchor: 0,
		lineCursor: 0,
	}
	a.detailTabs().Open(detailFileKey{Path: "a.go"})
	a.moveSelection(1)
	if !a.lineSel[1] || a.lineSel[0] {
		t.Fatalf("the arrow left the selection at %v; want line 1 alone", a.lineSel)
	}
	a.moveSelection(1)
	if !a.lineSel[3] {
		t.Fatalf("the second arrow left the selection at %v; want line 3, the next change", a.lineSel)
	}
}

func TestCanOpenSelectedFile(t *testing.T) {
	a := &app{page: "history", worktreeSelected: true, repo: &gitrepo.Repository{}, selectedPath: "a.txt"}
	if !a.canOpenSelectedFile() {
		t.Error("a selected file on the workspace should open on Enter")
	}
	a.lineSel = map[int]bool{1: true}
	if a.canOpenSelectedFile() {
		t.Error("an active line selection must keep Enter from opening")
	}
	a.lineSel = nil
	a.prompt.open = true
	if a.canOpenSelectedFile() {
		t.Error("an open prompt must keep Enter from opening")
	}
	a.prompt.open = false
	a.selectedPath = ""
	if a.canOpenSelectedFile() {
		t.Error("no selected file should do nothing")
	}
	a.selectedPath = "a.txt"
	a.worktreeSelected = false
	if a.canOpenSelectedFile() {
		t.Error("Enter opens files only while the working tree's row is selected")
	}
}

func TestOperationConflicts(t *testing.T) {
	a := &app{status: gitrepo.Status{Entries: []gitrepo.Entry{{Path: "ok"}, {Path: "x", Conflict: true}}}}
	if !a.operationConflicts() {
		t.Error("an unresolved file should block continuing")
	}
	a.status = gitrepo.Status{Entries: []gitrepo.Entry{{Path: "ok"}}}
	if a.operationConflicts() {
		t.Error("a resolved tree should allow continuing")
	}
	if (&app{}).operationConflicts() {
		t.Error("an empty status has no conflicts")
	}
}

func TestCanCommitShortcut(t *testing.T) {
	a := &app{page: "history", worktreeSelected: true, repo: &gitrepo.Repository{Root: t.TempDir()}}
	if !a.canCommitShortcut() {
		t.Error("the working tree with no dialog should take ⌘↩")
	}
	a.worktreeSelected = false
	if a.canCommitShortcut() {
		t.Error("⌘↩ commits only from the working tree")
	}
	a.worktreeSelected = true
	a.confirm.open = true
	if a.canCommitShortcut() {
		t.Error("a confirm dialog keeps ⌘↩")
	}
	a.confirm.open = false
	a.prompt.open = true
	if a.canCommitShortcut() {
		t.Error("a prompt keeps ⌘↩")
	}
}

func TestSelectedStashObject(t *testing.T) {
	a := &app{
		stashes:       []gitrepo.Stash{{SHA: "aaa", Message: "one"}, {SHA: "bbb"}},
		selectedStash: "bbb",
	}
	if stash, ok := a.selectedStashObject(); !ok || stash.SHA != "bbb" {
		t.Errorf("selectedStashObject() = %+v, %t; want bbb", stash, ok)
	}
	a.selectedStash = "missing"
	if _, ok := a.selectedStashObject(); ok {
		t.Error("a missing stash should not resolve")
	}
}

func TestWorkspaceSelection(t *testing.T) {
	a := &app{
		page:       "worktrees",
		worktrees:  []gitrepo.Worktree{{Path: "/tmp/a"}, {Path: "/tmp/b"}},
		submodules: []gitrepo.Submodule{{Path: "mod"}},
	}
	want := []string{"worktree:/tmp/a", "worktree:/tmp/b", "submodule:mod"}
	if got := a.workspaceKeys(); !slices.Equal(got, want) {
		t.Fatalf("workspaceKeys() = %v; want %v", got, want)
	}
	a.moveWorkspaceSelection(1)
	if a.worktreeSel != "worktree:/tmp/a" || !a.revealSelection {
		t.Errorf("first move: sel=%q reveal=%t; want the first row revealed", a.worktreeSel, a.revealSelection)
	}
	a.revealSelection = false
	a.moveWorkspaceSelection(1)
	if a.worktreeSel != "worktree:/tmp/b" {
		t.Errorf("second move: sel=%q; want worktree:/tmp/b", a.worktreeSel)
	}
	a.moveWorkspaceSelection(1 << 30)
	if a.worktreeSel != "submodule:mod" {
		t.Errorf("end: sel=%q; want submodule:mod", a.worktreeSel)
	}
	a.moveWorkspaceSelection(-1 << 30)
	if a.worktreeSel != "worktree:/tmp/a" {
		t.Errorf("home: sel=%q; want worktree:/tmp/a", a.worktreeSel)
	}
	locked := &app{page: "worktrees", worktreeSel: "worktree:/gone"}
	locked.moveWorkspaceSelection(1)
	if locked.worktreeSel != "worktree:/gone" {
		t.Errorf("empty list changed the selection to %q", locked.worktreeSel)
	}
}

func TestSyncBranchSelection(t *testing.T) {
	a := &app{
		branches:  []gitrepo.Branch{{Name: "main"}, {Name: "other"}},
		tags:      []gitrepo.Tag{{Name: "v1"}},
		branchSel: "local:main",
	}
	a.branchQuery = "oth"
	a.syncBranchSelection()
	if a.branchSel != "local:other" {
		t.Errorf("hidden selection: got %q; want local:other", a.branchSel)
	}
	a.branchQuery = "v1"
	a.syncBranchSelection()
	if a.branchSel != "tag:v1" {
		t.Errorf("tag match: got %q; want tag:v1", a.branchSel)
	}
	a.branchQuery = "zzz"
	a.syncBranchSelection()
	if a.branchSel != "" {
		t.Errorf("no match: got %q; want empty", a.branchSel)
	}
	a.branchSel = "local:main"
	a.branchQuery = ""
	a.syncBranchSelection()
	if a.branchSel != "local:main" {
		t.Errorf("empty search: got %q; want local:main", a.branchSel)
	}
}

// A count of one takes the singular in English. "1 hunks" and "Copy 1
// selected lines" read as bugs, and the Chinese strings have no plural form.
func TestCountLabelsUseSingularForOne(t *testing.T) {
	zh := &app{locale: localeChinese}
	en := &app{locale: localeEnglish}
	if got := en.hunksLabel(1); got != "1 hunk" {
		t.Errorf("en hunk = %q; want %q", got, "1 hunk")
	}
	if got := en.hunksLabel(3); got != "3 hunks" {
		t.Errorf("en hunks = %q; want %q", got, "3 hunks")
	}
	if got := en.blameLinesLabel(1); got != "1 line" {
		t.Errorf("en blame line = %q; want %q", got, "1 line")
	}
	if got := en.copyLinesLabel(1); got != "Copy 1 selected line" {
		t.Errorf("en copy line = %q; want %q", got, "Copy 1 selected line")
	}
	if got := en.copyLinesLabel(2); got != "Copy 2 selected lines" {
		t.Errorf("en copy lines = %q; want %q", got, "Copy 2 selected lines")
	}
	if got := zh.hunksLabel(1); got != "1 个片段" {
		t.Errorf("zh hunk = %q; want %q", got, "1 个片段")
	}
	if got := zh.blameLinesLabel(2); got != "2 行" {
		t.Errorf("zh blame = %q; want %q", got, "2 行")
	}
	if got := en.workspacesLabel(1, 0); got != "1 worktree · 0 submodules" {
		t.Errorf("en single worktree = %q", got)
	}
	if got := en.workspacesLabel(2, 1); got != "2 worktrees · 1 submodule" {
		t.Errorf("en worktrees = %q", got)
	}
	if got := en.discardLinesMessage(1); got != "Discard changes to the 1 selected line in the working tree. You can undo this once from the toast." {
		t.Errorf("en discard one line = %q", got)
	}
	if got := en.discardHunkMessage(1); got != "Discard changes to the 1 changed line in this hunk in the working tree. You can undo this once from the toast." {
		t.Errorf("en discard one hunk line = %q", got)
	}
	if got := zh.workspacesLabel(1, 0); got != "1 个工作树 · 0 个子模块" {
		t.Errorf("zh worktree = %q", got)
	}
	if got := en.branchesLabel(1, 59, 252); got != "1 local branch · 59 remote branches · 252 tags" {
		t.Errorf("en branches = %q", got)
	}
	if got := en.branchesLabel(0, 1, 1); got != "0 local branches · 1 remote branch · 1 tag" {
		t.Errorf("en branches singular = %q", got)
	}
	if got := zh.branchesLabel(1, 59, 252); got != "1 个本地分支 · 59 个远端分支 · 252 个标签" {
		t.Errorf("zh branches = %q", got)
	}
}

func TestTagTargetLabelNamesNonCommitTargets(t *testing.T) {
	zh := &app{locale: localeChinese}
	en := &app{locale: localeEnglish}
	tests := []struct {
		kind   string
		zh, en string
	}{
		{"tree", "树对象", "Tree object"},
		{"blob", "文件对象", "File object"},
		// An unknown object kind stays visible rather than being hidden.
		{"sparse", "sparse", "sparse"},
	}
	for _, tt := range tests {
		if got := zh.tagTargetLabel(tt.kind); got != tt.zh {
			t.Errorf("zh %s = %q; want %q", tt.kind, got, tt.zh)
		}
		if got := en.tagTargetLabel(tt.kind); got != tt.en {
			t.Errorf("en %s = %q; want %q", tt.kind, got, tt.en)
		}
	}
}

func TestElideMiddleKeepsBothEnds(t *testing.T) {
	tests := []struct {
		in   string
		max  int
		want string
	}{
		// Short names are left alone.
		{"master", 20, "master"},
		{"feature/login", 20, "feature/login"},
		// Two dependabot branches share a long prefix; the tails must stay.
		{"origin/dependabot/github_actions/peter-evans/create-pull-request-8.1.2", 20, "origin/d…quest-8.1.2"},
		{"origin/dependabot/go_modules/github.com/xo/terminfo-1.3.1", 20, "origin/d…minfo-1.3.1"},
		// Runes, not bytes: a cut through a multi-byte name stays valid.
		{"一个很长的分支名字用来测试中间省略", 10, "一个很长…试中间省略"},
		// A cap too small to keep both ends returns the whole name rather than
		// a lone ellipsis.
		{"master", 4, "master"},
	}
	for _, tt := range tests {
		if got := elideMiddle(tt.in, tt.max); got != tt.want {
			t.Errorf("elideMiddle(%q, %d) = %q; want %q", tt.in, tt.max, got, tt.want)
		}
	}
}

// The details pane lists the row's three capsules first, then the refs the
// row folded into +N, so the same commit never shows one order in the list
// and another below it.
func TestCommitRefOrderMatchesTheRow(t *testing.T) {
	refs := []string{"main", "tag: v1", "tag: light1", "b1", "b2", "b3"}
	for _, test := range []struct {
		name  string
		focus historyRef
		head  string
	}{
		{"no context", historyRef{}, ""},
		{"filtered tag", historyRef{Kind: historyRefTag, Name: "v1"}, ""},
		{"head branch", historyRef{}, "main"},
		{"filter then head", historyRef{Kind: historyRefTag, Name: "light1"}, "main"},
	} {
		ordered := commitRefOrder(refs, test.focus, test.head)
		shown, folded := rowRefs(refs, test.focus, test.head)
		want := append(append([]string{}, shown...), folded...)
		if len(ordered) != len(refs) {
			t.Fatalf("%s: commitRefOrder listed %d refs; want %d", test.name, len(ordered), len(refs))
		}
		for i := range want {
			if ordered[i] != want[i] {
				t.Errorf("%s: commitRefOrder[%d] = %q; want %q", test.name, i, ordered[i], want[i])
			}
		}
		for i := range shown {
			if ordered[i] != shown[i] {
				t.Errorf("%s: row capsule %d = %q but details lists %q first", test.name, i, shown[i], ordered[i])
			}
		}
	}
}

// A commit with three refs or fewer is not reordered by the row, so the
// details pane must not reorder it either.
func TestCommitRefOrderKeepsShortListsAsTheRowDoes(t *testing.T) {
	refs := []string{"tag: v1", "main"}
	ordered := commitRefOrder(refs, historyRef{Kind: historyRefLocal, Name: "main"}, "")
	for i := range refs {
		if ordered[i] != refs[i] {
			t.Fatalf("commitRefOrder(%v) = %v; want the Git order unchanged", refs, ordered)
		}
	}
}

// A narrow row keeps the ref the reader is looking at, then the branch HEAD
// points at, and folds the rest. The shown refs still print in Git's order.
func TestANarrowRowFoldsRefsByRank(t *testing.T) {
	refs := []string{"main", "tag: v1", "tag: v2", "b1"}
	shown, folded := rowRefsLimit(refs, historyRef{}, "main", 1)
	if !slices.Equal(shown, []string{"main"}) || !slices.Equal(folded, []string{"tag: v1", "tag: v2", "b1"}) {
		t.Errorf("one slot = %v shown, %v folded; want main shown", shown, folded)
	}
	shown, folded = rowRefsLimit(refs, historyRef{Kind: historyRefTag, Name: "v2"}, "main", 2)
	if !slices.Equal(shown, []string{"main", "tag: v2"}) || !slices.Equal(folded, []string{"tag: v1", "b1"}) {
		t.Errorf("two slots with v2 focused = %v shown, %v folded; want main and v2 shown", shown, folded)
	}
	// A caller that asks for no room still gets one capsule, or the row would
	// lose the only ref it has space for.
	shown, _ = rowRefsLimit(refs, historyRef{}, "", 0)
	if !slices.Equal(shown, []string{"main"}) {
		t.Errorf("a zero slot count shows %v; want the first ref", shown)
	}
	shown, folded = rowRefsLimit(refs[:2], historyRef{}, "", 2)
	if len(folded) != 0 || len(shown) != 2 {
		t.Errorf("refs that fit = %v shown, %v folded; want no fold", shown, folded)
	}
}
