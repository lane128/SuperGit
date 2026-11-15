package lobedesk

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"supergit/internal/gitrepo"
)

func menuItem(t *testing.T, items []overlay.MenuItem, key string) overlay.MenuItem {
	t.Helper()
	i := menuKeyIndex(items, key)
	if i < 0 {
		t.Fatalf("menu has no %q", key)
	}
	return items[i]
}

func menuKeyIndex(items []overlay.MenuItem, key string) int {
	for i, item := range items {
		if item.Key == key {
			return i
		}
	}
	return -1
}

// A row menu mixes two kinds of action: copies, which take something away
// with you, and jumps, which move the page. Every menu the app has already
// separates them with a divider, except the three ref menus, where "view
// history" sat directly under the copy entries. The divider is the only
// signal of that difference, so it is checked rather than left to the eye.
func TestRefMenusSeparateCopiesFromHistory(t *testing.T) {
	a := &app{remotes: []gitrepo.Remote{{Name: "origin"}}}
	menus := map[string][]overlay.MenuItem{
		"branch": a.branchMenuItems(gitrepo.Branch{Name: "main"}),
		"remote": a.remoteBranchMenuItems(gitrepo.Branch{Name: "origin/main"}),
		"tag":    a.tagMenuItems(gitrepo.Tag{Name: "v1.0.0", TargetType: "commit"}),
	}
	for name, items := range menus {
		copies := menuKeyIndex(items, "copy")
		history := menuKeyIndex(items, "history")
		if copies < 0 || history < 0 {
			t.Fatalf("%s menu: copy at %d, history at %d; want both", name, copies, history)
		}
		if copies >= history {
			t.Errorf("%s menu: history at %d comes before copy at %d", name, history, copies)
		}
		if history == 0 || !items[history-1].Divider {
			t.Errorf("%s menu: history is not separated from the copy group", name)
		}
	}
}

// The commit menu keeps four groups: copies, moves of the current branch,
// ref creation, and the commits that get rewritten. Reset sits alone at the
// bottom in the danger style because it rewrites what the branch points at.
// Every action that asks something first ends with an ellipsis; the copies
// act at once and do not.
func TestCommitMenuKeepsItsGroupsAndItsEllipses(t *testing.T) {
	a := &app{locale: localeChinese}
	items := a.commitMenuItems(gitrepo.Commit{SHA: "0123456789abcdef0123456789abcdef01234567"})
	for _, key := range []string{"merge", "rebase", "checkout", "tag", "annotated-tag", "new-branch", "cherry-pick", "revert", "reset"} {
		item := menuItem(t, items, key)
		if !strings.HasSuffix(item.Label, "…") {
			t.Errorf("%s label %q lacks the ellipsis of a dialog", key, item.Label)
		}
	}
	for _, key := range []string{"copy-sha", "copy-subject"} {
		item := menuItem(t, items, key)
		if strings.HasSuffix(item.Label, "…") {
			t.Errorf("%s label %q should act at once", key, item.Label)
		}
	}
	last := items[len(items)-1]
	if last.Key != "reset" || !last.Danger {
		t.Fatalf("last item is %q danger=%v; want the danger reset", last.Key, last.Danger)
	}
	if !items[len(items)-2].Divider {
		t.Fatal("reset is not separated from the group above it")
	}
	for _, key := range []string{"checkout", "tag", "cherry-pick"} {
		if i := menuKeyIndex(items, key); i < 1 || !items[i-1].Divider {
			t.Errorf("%s does not open a new group", key)
		}
	}
	if menuKeyIndex(items, "copy-message") >= 0 {
		t.Fatal("copy-message showed without the full message loaded")
	}
}

// A commit's menu can put HEAD on any branch that points at it, which keeps
// HEAD attached. Those items open the move group, before the detached-HEAD
// checkout; the branch already checked out stays listed but dimmed so the menu
// says where HEAD is. Branches on another commit do not appear.
func TestCommitMenuListsTheOneBranchOnIt(t *testing.T) {
	const sha, other = "0123456789abcdef0123456789abcdef01234567", "fedcba9876543210fedcba9876543210fedcba98"
	a := &app{locale: localeChinese, branches: []gitrepo.Branch{
		{Name: "main", SHA: sha, Current: true},
		{Name: "old", SHA: other},
	}}
	items := a.commitMenuItems(gitrepo.Commit{SHA: sha})
	current := menuItem(t, items, commitBranchCheckoutPrefix+"main")
	if !current.Disabled || current.Extra != a.t("current") {
		t.Errorf("current branch item disabled=%v extra=%q; want dimmed and marked %q", current.Disabled, current.Extra, a.t("current"))
	}
	if menuKeyIndex(items, commitBranchCheckoutPrefix+"old") >= 0 {
		t.Error("a branch on another commit was offered")
	}
	if menuKeyIndex(items, commitBranchPickerKey) >= 0 {
		t.Error("a commit with one branch opened the picker")
	}
	first := menuKeyIndex(items, commitBranchCheckoutPrefix+"main")
	checkout := menuKeyIndex(items, "checkout")
	if first < 1 || !items[first-1].Divider {
		t.Error("the branch item does not open the move group")
	}
	if first >= checkout {
		t.Error("the branch item does not come before the detached-HEAD checkout")
	}
	if menuKeyIndex(a.commitMenuItems(gitrepo.Commit{SHA: other}), commitBranchCheckoutPrefix+"main") >= 0 {
		t.Error("another commit's menu offered a branch that is not on it")
	}
}

// A commit can sit on a dozen branches. Listing every one of them grows the
// menu past a small window, and the rows after the list cannot be reached: the
// menu does not scroll. So several branches take one row, which opens the
// picker, and the menu's height stays the same whatever the count.
func TestCommitMenuCollapsesManyBranchesIntoThePicker(t *testing.T) {
	const sha = "0123456789abcdef0123456789abcdef01234567"
	branches := make([]gitrepo.Branch, 0, 12)
	for i := 0; i < 12; i++ {
		branches = append(branches, gitrepo.Branch{Name: fmt.Sprintf("topic-%d", i), SHA: sha})
	}
	a := &app{locale: localeChinese, branches: branches}
	items := a.commitMenuItems(gitrepo.Commit{SHA: sha})
	picker := menuItem(t, items, commitBranchPickerKey)
	if picker.Extra != "12" || picker.Disabled {
		t.Errorf("picker item extra=%q disabled=%v; want the count and enabled", picker.Extra, picker.Disabled)
	}
	for _, item := range items {
		if strings.HasPrefix(item.Key, commitBranchCheckoutPrefix) {
			t.Fatalf("the menu still lists %q beside the picker", item.Label)
		}
	}
	index := menuKeyIndex(items, commitBranchPickerKey)
	if index < 1 || !items[index-1].Divider {
		t.Error("the picker item does not open the move group")
	}
	if index >= menuKeyIndex(items, "checkout") {
		t.Error("the picker item does not come before the detached-HEAD checkout")
	}
	// Busy and loading are the states that hold every other action back, so
	// the picker waits for them too.
	a.busy = true
	if !menuItem(t, a.commitMenuItems(gitrepo.Commit{SHA: sha}), commitBranchPickerKey).Disabled {
		t.Error("the picker stayed enabled while the app was busy")
	}
}

// The commit menu can narrow the history to one author or one day. Both write
// into the search box, so they act at once and carry no ellipsis, and they get
// a group of their own between the copies and the actions that move HEAD. A
// commit whose author or date could not be read leaves its item disabled.
func TestCommitMenuOffersTheAuthorAndDayFilters(t *testing.T) {
	a := &app{locale: localeChinese}
	items := a.commitMenuItems(gitrepo.Commit{SHA: "0123456789abcdef0123456789abcdef01234567", Author: "Ada", Day: "2026-11-08"})
	author := menuItem(t, items, "filter-author")
	day := menuItem(t, items, "filter-day")
	for _, item := range []overlay.MenuItem{author, day} {
		if strings.HasSuffix(item.Label, "…") {
			t.Errorf("%s label %q asks something first; the filter applies at once", item.Key, item.Label)
		}
		if item.Disabled {
			t.Errorf("%s is disabled although the commit has the value", item.Key)
		}
	}
	if i := menuKeyIndex(items, "filter-author"); i < 1 || !items[i-1].Divider {
		t.Error("the filters do not open a group after the copies")
	}
	if i := menuKeyIndex(items, "filter-month"); i+1 >= len(items) || !items[i+1].Divider {
		t.Error("the filters are not separated from the actions below")
	}
	if menuKeyIndex(items, "filter-author") < menuKeyIndex(items, "copy-subject") {
		t.Error("a filter comes before the copies")
	}

	bare := a.commitMenuItems(gitrepo.Commit{SHA: "0123456789abcdef0123456789abcdef01234567"})
	if !menuItem(t, bare, "filter-author").Disabled || !menuItem(t, bare, "filter-day").Disabled {
		t.Error("a commit without an author or a day offered the filters")
	}
}

// The week and the month are the searches a reader would otherwise spell out
// by hand, so they sit in the same group as the author and the day and write
// the same kind of text: a range the box shows and clears like any other
// query. They are relative to the clock rather than to the commit, so they
// stay enabled when the commit's own date could not be read.
func TestTheMenuFiltersTheLastWeekAndThisMonth(t *testing.T) {
	now := time.Date(2026, 11, 15, 9, 0, 0, 0, time.Local)
	a := &app{locale: localeChinese}
	items := a.commitMenuItems(gitrepo.Commit{SHA: "0123456789abcdef0123456789abcdef01234567"})
	recent := menuItem(t, items, "filter-recent")
	month := menuItem(t, items, "filter-month")
	for _, item := range []overlay.MenuItem{recent, month} {
		if item.Disabled {
			t.Errorf("%s is disabled on a commit without a date; it filters by the clock", item.Key)
		}
		if strings.HasSuffix(item.Label, "…") {
			t.Errorf("%s label %q asks something first; the filter applies at once", item.Key, item.Label)
		}
	}
	if menuKeyIndex(items, "filter-recent") < menuKeyIndex(items, "filter-day") {
		t.Error("the week filter is not in the filter group with the day")
	}

	commits := []gitrepo.Commit{
		{SHA: "old", Day: "2026-11-08"},
		{SHA: "first", Day: "2026-11-09"},
		{SHA: "today", Day: "2026-11-15"},
		{SHA: "later", Day: "2026-11-16"},
	}
	week, ok := historyFilterFor("filter-recent", gitrepo.Commit{}, now)
	if !ok || week != "2026-11-09..2026-11-15" {
		t.Fatalf("the week filter writes %q; want the seven days ending today", week)
	}
	if got := filterCommits(commits, week); len(got) != 2 || got[0].SHA != "first" || got[1].SHA != "today" {
		t.Errorf("the week filter showed %d commits; want first and today", len(got))
	}
	monthQuery, ok := historyFilterFor("filter-month", gitrepo.Commit{}, now)
	if !ok || monthQuery != "2026-11" {
		t.Fatalf("the month filter writes %q; want the month prefix", monthQuery)
	}
	if got := filterCommits(commits, monthQuery); len(got) != 4 {
		t.Errorf("the month filter showed %d commits; want all four", len(got))
	}
}

// The last-commit card and its menu went away with the workspace page. Undo
// still needs a home: the checked-out commit's row carries it, and only that
// row, because undoing an older commit would also drop the commits above it,
// which is what reset is for.
func TestOnlyTheHeadCommitOffersUndo(t *testing.T) {
	a := &app{locale: localeChinese}
	head := a.commitMenuItems(gitrepo.Commit{SHA: "0123456789abcdef0123456789abcdef01234567", Refs: "HEAD -> main"})
	undo := menuItem(t, head, "undo")
	if !strings.HasSuffix(undo.Label, "…") {
		t.Errorf("undo label %q asks first; it wants the ellipsis", undo.Label)
	}
	if undoIdx, revertIdx := menuKeyIndex(head, "undo"), menuKeyIndex(head, "revert"); undoIdx < revertIdx {
		t.Errorf("undo at %d is not with the rewriting actions (revert at %d)", undoIdx, revertIdx)
	}
	if undoIdx, resetIdx := menuKeyIndex(head, "undo"), menuKeyIndex(head, "reset"); undoIdx >= resetIdx {
		t.Error("undo is not above the danger reset")
	}
	older := a.commitMenuItems(gitrepo.Commit{SHA: "fedcba9876543210fedcba9876543210fedcba98", Refs: "main"})
	if menuKeyIndex(older, "undo") >= 0 {
		t.Error("a commit that is not HEAD offered undo")
	}
}

// The file menu stages first, keeps the destructive discard beside it, opens
// a group for stash, and ends with the two jumps that open a page. An
// untracked file's discard is a delete and says so.
func TestFileMenuKeepsItsGroupsAndItsEllipses(t *testing.T) {
	a := &app{locale: localeChinese}
	untracked := gitrepo.Entry{Path: "new.txt", Index: '?', Worktree: '?'}
	items := a.fileMenuItems(untracked, false)
	if items[0].Key != "stage" {
		t.Fatalf("first item is %q; want stage", items[0].Key)
	}
	discardIdx := menuKeyIndex(items, "discard")
	stashIdx := menuKeyIndex(items, "stash")
	if discardIdx < 0 || stashIdx < 0 {
		t.Fatalf("unstaged file menu: discard at %d, stash at %d; want both", discardIdx, stashIdx)
	}
	discard := items[discardIdx]
	if !discard.Danger || discard.Label != a.t("deleteFile") {
		t.Fatalf("untracked discard = %q danger=%v; want the delete label in danger", discard.Label, discard.Danger)
	}
	stash := items[stashIdx]
	if !strings.HasSuffix(stash.Label, "…") || !items[stashIdx-1].Divider {
		t.Fatalf("stash = %q; want its own group and an ellipsis", stash.Label)
	}
	if last := items[len(items)-1].Key; last != "blame" {
		t.Fatalf("last item is %q; want blame", last)
	}

	tracked := a.fileMenuItems(gitrepo.Entry{Path: "a.txt", Index: 'M', Worktree: ' '}, false)
	if got := menuItem(t, tracked, "discard").Label; got != a.t("discardChanges") {
		t.Fatalf("tracked discard = %q; want %q", got, a.t("discardChanges"))
	}

	staged := a.fileMenuItems(gitrepo.Entry{Path: "a.txt", Index: 'M', Worktree: ' '}, true)
	if staged[0].Key != "unstage" {
		t.Fatalf("first item for a staged file is %q; want unstage", staged[0].Key)
	}
	if menuKeyIndex(staged, "discard") >= 0 {
		t.Fatal("a staged file offered discard; unstaging comes first")
	}
}

// A conflicted file cannot be staged until the conflict is settled, so its
// menu offers the three ways to settle it before anything else.
func TestConflictFileMenuStartsWithTheThreeResolutions(t *testing.T) {
	a := &app{locale: localeChinese}
	items := a.conflictFileMenuItems()
	want := []string{"resolve", "ours", "theirs"}
	for i, key := range want {
		if items[i].Key != key {
			t.Fatalf("item %d is %q; want %q", i, items[i].Key, key)
		}
	}
	if menuKeyIndex(items, "stage") >= 0 || menuKeyIndex(items, "stash") >= 0 {
		t.Fatal("a conflicted file offered stage or stash")
	}
	if !items[len(want)].Divider {
		t.Fatal("the resolution group is not separated from the rest")
	}
}

// The copy entries belong together: "copy name" and "copy SHA" are the same
// kind of action and nothing may sit between them.
func TestRefMenusKeepCopyEntriesTogether(t *testing.T) {
	a := &app{remotes: []gitrepo.Remote{{Name: "origin"}}}
	menus := map[string][]overlay.MenuItem{
		"branch": a.branchMenuItems(gitrepo.Branch{Name: "main"}),
		"remote": a.remoteBranchMenuItems(gitrepo.Branch{Name: "origin/main"}),
		"tag":    a.tagMenuItems(gitrepo.Tag{Name: "v1.0.0", TargetType: "commit"}),
	}
	for name, items := range menus {
		copyIdx := menuKeyIndex(items, "copy")
		shaIdx := menuKeyIndex(items, "copy-sha")
		if shaIdx != copyIdx+1 {
			t.Errorf("%s menu: copy-sha at %d is not right after copy at %d", name, shaIdx, copyIdx)
		}
	}
}

// Merge, rebase and reset name both ends, the way Sublime Merge's commit menu
// does: the ref that tells the reader which line the commit is on, and the
// branch the action changes. The current branch is the destination, so it is
// never shown as the source, and a commit no other ref decorates falls back to
// its short SHA.
func TestTheCommitMenuNamesTheBranchItActsOn(t *testing.T) {
	sha := "0123456789abcdef0123456789abcdef01234567"
	a := &app{locale: localeChinese, branches: []gitrepo.Branch{{Name: "main", Current: true}, {Name: "feat"}}}
	items := a.commitMenuItems(gitrepo.Commit{SHA: sha, Refs: "HEAD -> main, feat, tag: v1, origin/feat/song"})

	if got, want := menuItem(t, items, "merge").Label, a.tf("mergeCommitInto", "origin/feat/song", "main"); got != want {
		t.Errorf("the merge item reads %q; want %q", got, want)
	}
	if got, want := menuItem(t, items, "rebase").Label, a.tf("rebaseBranchOnto", "main", "origin/feat/song"); got != want {
		t.Errorf("the rebase item reads %q; want %q", got, want)
	}
	if got, want := menuItem(t, items, "reset").Label, a.tf("resetBranchToCommit", "main"); got != want {
		t.Errorf("the reset item reads %q; want %q", got, want)
	}

	bare := a.commitMenuItems(gitrepo.Commit{SHA: sha})
	if got, want := menuItem(t, bare, "merge").Label, a.tf("mergeCommitInto", shortSHA(sha), "main"); got != want {
		t.Errorf("a commit with no other ref reads %q; want its short SHA (%q)", got, want)
	}

	// A detached HEAD has no branch to name, so the generic wording stays.
	detached := &app{locale: localeChinese}
	if got := menuItem(t, detached.commitMenuItems(gitrepo.Commit{SHA: sha}), "merge").Label; got != detached.t("mergeIntoCurrentAction") {
		t.Errorf("a detached HEAD reads %q; want the generic label", got)
	}
}
