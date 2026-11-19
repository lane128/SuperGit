package lobedesk

import (
	"path"
	"sort"
	"strings"
	"time"

	"supergit/internal/gitrepo"
)

func splitRepoPath(value string) (name, dir string) {
	value = strings.TrimSuffix(value, "/")
	if value == "" {
		return "", ""
	}
	name = path.Base(value)
	dir = path.Dir(value)
	if dir == "." || dir == "/" {
		dir = ""
	}
	return name, dir
}

func statusCode(entry gitrepo.Entry, staged bool) string {
	switch {
	case entry.Conflict:
		return "!"
	case entry.IsSubmodule():
		return "S"
	}
	code := entry.Worktree
	if staged {
		code = entry.Index
	}
	if code == 0 || code == ' ' || code == '.' {
		return "?"
	}
	return string(code)
}

// statusTagColor maps a porcelain status to the closest Lobe UI tag color.
func statusTagColor(entry gitrepo.Entry, staged bool) string {
	switch {
	case entry.Conflict:
		return "error"
	case entry.IsSubmodule():
		return "purple"
	}
	code := entry.Worktree
	if staged {
		code = entry.Index
	}
	switch code {
	case 'A':
		return "success"
	case '?':
		return "info"
	case 'D':
		return "error"
	case 'R', 'C':
		return "geekblue"
	case 'M', 'T':
		return "warning"
	default:
		return "gray"
	}
}

// isUntracked reports a file Git does not know yet. It has no index entry, so
// it is Unstaged too; the workspace lists it apart from the tracked edits that
// are waiting, because that is the one group the reader has never committed.
func isUntracked(entry gitrepo.Entry) bool {
	return entry.Index == '?' && entry.Worktree == '?'
}

func (a *app) stagedEntries() []gitrepo.Entry {
	var result []gitrepo.Entry
	for _, entry := range a.status.Entries {
		if entry.Staged() && !entry.Conflict {
			result = append(result, entry)
		}
	}
	return result
}

// unstagedEntries lists the tracked files with changes not in the index.
// Untracked files are Unstaged as well, but they get their own section.
func (a *app) unstagedEntries() []gitrepo.Entry {
	var result []gitrepo.Entry
	for _, entry := range a.status.Entries {
		if entry.Unstaged() && !isUntracked(entry) {
			result = append(result, entry)
		}
	}
	return result
}

// stageableCount is how many rows the stage-all button would actually stage.
// Conflicts are left out: each one needs its resolution chosen, not a plain
// add, so a list of nothing but conflicts leaves the button with nothing to do.
func stageableCount(unstaged, untracked []gitrepo.Entry) int {
	count := len(untracked)
	for _, entry := range unstaged {
		if !entry.Conflict {
			count++
		}
	}
	return count
}

func (a *app) untrackedEntries() []gitrepo.Entry {
	var result []gitrepo.Entry
	for _, entry := range a.status.Entries {
		if isUntracked(entry) {
			result = append(result, entry)
		}
	}
	return result
}

// branchesAt lists the local branches whose tip is this commit, in Git's own
// sorted order. The commit's decorations already name some of them, but the
// branch list knows which one is checked out, and it still works for a commit
// the loaded window did not decorate (a file's history reaches older ones).
func (a *app) branchesAt(sha string) []gitrepo.Branch {
	var result []gitrepo.Branch
	if sha == "" {
		return nil
	}
	for _, branch := range a.branches {
		if branch.SHA == sha {
			result = append(result, branch)
		}
	}
	return result
}

// filterCommits narrows a history list to the commits a search matches: the
// SHA, subject, author, refs and calendar day. Matching the day lets a reader
// type "2026-11-08" and see that day's commits, and a "from..to" range keeps
// that day, a week or a month; the commit menu fills in an author, a day, the
// last week or this month so the filter is discoverable. The text lands in the
// search box, so what was searched for is always on screen and one click from
// being cleared.
func filterCommits(commits []gitrepo.Commit, query string) []gitrepo.Commit {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return commits
	}
	// A range is the only query that cannot be a substring. It is checked
	// first so "2026-11-09..2026-11-15" never falls through to the text
	// search and quietly matches nothing.
	from, to, ranged := dayRangeBounds(query)
	result := make([]gitrepo.Commit, 0, len(commits))
	for _, commit := range commits {
		if ranged {
			if commit.Day != "" && (from == "" || commit.Day >= from) && (to == "" || commit.Day <= to) {
				result = append(result, commit)
			}
			continue
		}
		haystack := strings.ToLower(commit.SHA + " " + commit.Subject + " " + commit.Author + " " + commit.Refs + " " + commit.Day)
		if strings.Contains(haystack, query) {
			result = append(result, commit)
		}
	}
	return result
}

// dayRangeBounds reads a search written as "from..to", Git's own spelling for
// a range. A bound may be a whole day ("2026-11-08"), a month ("2026-11") or a
// year ("2026"), and either side may be left out, so "2026-11.." is everything
// from November on. It reports false when a non-empty side is not a date, so a
// query the reader typed for its text still gets the substring search.
func dayRangeBounds(query string) (from, to string, ok bool) {
	left, right, found := strings.Cut(query, "..")
	if !found {
		return "", "", false
	}
	left, right = strings.TrimSpace(left), strings.TrimSpace(right)
	from, fromOK := dayBoundStart(left)
	to, toOK := dayBoundEnd(right)
	if left != "" && !fromOK || right != "" && !toOK {
		return "", "", false
	}
	if from == "" && to == "" {
		return "", "", false
	}
	return from, to, true
}

func dayBoundStart(value string) (string, bool) {
	switch len(value) {
	case 0:
		return "", true
	case 4:
		if _, err := time.Parse("2006", value); err != nil {
			return "", false
		}
		return value + "-01-01", true
	case 7:
		if _, err := time.Parse("2006-01", value); err != nil {
			return "", false
		}
		return value + "-01", true
	case 10:
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return "", false
		}
		return value, true
	}
	return "", false
}

func dayBoundEnd(value string) (string, bool) {
	switch len(value) {
	case 0:
		return "", true
	case 4:
		if _, err := time.Parse("2006", value); err != nil {
			return "", false
		}
		return value + "-12-31", true
	case 7:
		if _, err := time.Parse("2006-01", value); err != nil {
			return "", false
		}
		// Day strings sort lexicographically, and "-31" is past every real
		// last day of a month, so this clips the whole month without asking
		// the calendar how long it is.
		return value + "-31", true
	case 10:
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return "", false
		}
		return value, true
	}
	return "", false
}

// recentWeekQuery is the search the menu's "last 7 days" entry writes: both
// ends spelled out, so the box says exactly which days are on screen, and the
// range still means the same thing after a reload.
func recentWeekQuery(now time.Time) string {
	return now.AddDate(0, 0, -6).Format("2006-01-02") + ".." + now.Format("2006-01-02")
}

// thisMonthQuery is what "this month" writes. A month prefix is already a
// substring of every day in it, so it needs no range syntax.
func thisMonthQuery(now time.Time) string {
	return now.Format("2006-01")
}

// preferredCommitPath picks the file a commit's details should show: the
// file already chosen in that commit, else the file whose history is being
// browsed, else the commit's first file. None of the preferred paths are in
// the commit, so the first changed file is the fallback.
func preferredCommitPath(files []gitrepo.ChangedFile, keep, filter string) string {
	if len(files) == 0 {
		return ""
	}
	for _, wanted := range []string{keep, filter} {
		if wanted == "" {
			continue
		}
		for _, file := range files {
			if file.Path == wanted {
				return wanted
			}
		}
	}
	return files[0].Path
}

// historySelection picks the commit a filtered history should show: the
// selected one while it still matches, otherwise the first match. An empty
// result means the selection stays as it is (no matches to move to).
func historySelection(commits []gitrepo.Commit, selected string) string {
	if len(commits) == 0 {
		return ""
	}
	for _, commit := range commits {
		if commit.SHA == selected {
			return selected
		}
	}
	return commits[0].SHA
}

func diffStats(diff gitrepo.Diff) (additions, deletions, hunks int) {
	for _, line := range diff.Lines {
		switch line.Kind {
		case "add":
			additions++
		case "del":
			deletions++
		}
	}
	return additions, deletions, len(diff.Hunks)
}

func shortSHA(sha string) string {
	if len(sha) <= 8 {
		return sha
	}
	return sha[:8]
}

// elideMiddle shortens text to max runes while keeping both ends, so two refs
// that share a long prefix (origin/dependabot/... on a fetched repository) stay
// apart by their tails. The head keeps the remote or namespace prefix; the tail
// is longer, because that is where such names differ. A badge's tooltip keeps
// the full name.
func elideMiddle(text string, max int) string {
	runes := []rune(text)
	if max < 5 || len(runes) <= max {
		return text
	}
	head := max * 2 / 5
	tail := max - 1 - head
	return string(runes[:head]) + "…" + string(runes[len(runes)-tail:])
}

// elidedPrefix marks the omitted beginning of a display string.
func elidedPrefix(visible string) string {
	return "…" + visible
}

// elideAround keeps the text near index and marks any omitted ends.
func elideAround(text string, index int) string {
	runes := []rune(text)
	start, end := max(0, index-1), min(len(runes), index+4)
	visible := string(runes[start:end])
	if start > 0 {
		visible = "…" + visible
	}
	if end < len(runes) {
		visible += "…"
	}
	return visible
}

// formatDateTime renders the local calendar date and clock down to the second.
func formatDateTime(value string) string {
	return formatDateTimeLayout(value, "2006-01-02 15:04:05")
}

// An unparseable timestamp is shown as it came from Git.
func formatDateTimeLayout(value, layout string) string {
	if value == "" {
		return ""
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.Local().Format(layout)
	}
	return value
}

func commitRefs(refs string) []string {
	if strings.TrimSpace(refs) == "" {
		return nil
	}
	parts := strings.Split(refs, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.TrimPrefix(part, "HEAD -> ")
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// rowRefs splits a row's refs into the ones it prints beside the subject and
// the tail it folds into the +N capsule. A row has space for three. Which three
// follow the history's context rather than Git's decoration order: the ref the
// page is filtered to, then the branch HEAD points at, then the rest in their
// original order. A crowded row therefore keeps the refs the reader is looking
// at and folds the tags and extra branches that would push the subject aside.
//
// The chosen three then print in Git's order, not in the order they were
// picked. Switching a filter should change which capsule holds which ref, not
// slide the capsules that were already there to new slots: a reader who just
// looked at "main, v1, v2" and filters to b3 sees "main, v1, b3" rather than
// "b3, main, v1".
func rowRefs(refs []string, focus historyRef, head string) (shown, folded []string) {
	return rowRefsLimit(refs, focus, head, 3)
}

// rowRefsLimit is rowRefs with a slot count the row's width can afford. A
// narrow list keeps one or two capsules so the subject keeps its room; the
// rest fold into the +N menu under the same ranking.
func rowRefsLimit(refs []string, focus historyRef, head string, limit int) (shown, folded []string) {
	limit = max(limit, 1)
	if len(refs) <= limit {
		return refs, nil
	}
	rank := func(ref string) int {
		if focus.Kind != "" && (ref == focus.Name || ref == "tag: "+focus.Name) {
			return 0
		}
		if head != "" && ref == head {
			return 1
		}
		return 2
	}
	order := make([]int, len(refs))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return rank(refs[order[a]]) < rank(refs[order[b]]) })
	keep := make([]bool, len(refs))
	for _, i := range order[:limit] {
		keep[i] = true
	}
	for i, ref := range refs {
		if keep[i] {
			shown = append(shown, ref)
		} else {
			folded = append(folded, ref)
		}
	}
	return shown, folded
}

// commitRefOrder lists every ref of a commit in the order the history row
// prints them: the row's shown capsules first, then the ones it folded into
// +N. The details pane has room for all of them and uses this so the same
// commit never lists its refs two different ways.
func commitRefOrder(refs []string, focus historyRef, head string) []string {
	shown, folded := rowRefs(refs, focus, head)
	ordered := make([]string, 0, len(shown)+len(folded))
	ordered = append(ordered, shown...)
	return append(ordered, folded...)
}

// currentBranch names the checked-out branch, or "" when HEAD is detached.
func (a *app) currentBranch() string {
	for _, branch := range a.branches {
		if branch.Current {
			return branch.Name
		}
	}
	return ""
}

// commitMenuSource names a commit for the menu's merge, rebase and reset
// labels. A remote branch tells the reader which line the commit is on, so it
// wins over a local one; the current branch is skipped because the label
// already names it as the destination. A commit no other ref decorates falls
// back to its short SHA, as Sublime Merge's "Reset main to This Commit" does.
func commitMenuSource(commit gitrepo.Commit, current string) string {
	var candidates []string
	for _, ref := range commitRefs(commit.Refs) {
		if strings.HasPrefix(ref, "tag: ") || ref == current {
			continue
		}
		candidates = append(candidates, ref)
	}
	for _, ref := range candidates {
		if strings.Contains(ref, "/") {
			return ref
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	return shortSHA(commit.SHA)
}

// headBranch names the branch HEAD points to, as "HEAD -> name" prints in
// Git's %D; empty when HEAD is detached or the commit is not HEAD.
func headBranch(refs string) string {
	for _, part := range strings.Split(refs, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "HEAD -> ") {
			return strings.TrimPrefix(part, "HEAD -> ")
		}
	}
	return ""
}

// commitBody returns the message after its subject line. A message with no
// blank separator has no body, as in Git's own interpretation.
func commitBody(message string) string {
	if i := strings.Index(message, "\n\n"); i >= 0 {
		return strings.TrimSpace(message[i+2:])
	}
	return ""
}

func (a *app) selectedEntry() (gitrepo.Entry, bool) {
	for _, entry := range a.status.Entries {
		if entry.Path == a.selectedPath {
			return entry, true
		}
	}
	return gitrepo.Entry{}, false
}

// historyRows is the list the history page draws this frame: the page's source
// -- the loaded window, one ref's reachable commits, or a file's history --
// narrowed by the search. The row colors, the lane layout and the details pane
// all index into these rows, so they agree on what "row 3" means. Counting the
// unfiltered window instead would point the details pane at another commit.
func (a *app) historyRows() []gitrepo.Commit {
	// historyPage built these rows at the start of this frame; callers outside
	// that build (a test, a handler between frames) get their own.
	if a.historyRowsSet && a.historyRowsFrame == a.frameSeq {
		return a.historyRowsCache
	}
	return a.filteredHistoryRows()
}

// selectedCommit is the commit the reader picked, looked up in the list they
// picked it from. A file's history can reach commits outside the loaded
// window, and a selection can outlive a filter that hid it, so the window
// itself is the fallback rather than the only place.
func (a *app) selectedCommit() (gitrepo.Commit, bool) {
	for _, commit := range a.historyRows() {
		if commit.SHA == a.selectedSHA {
			return commit, true
		}
	}
	for _, commit := range a.commits {
		if commit.SHA == a.selectedSHA {
			return commit, true
		}
	}
	if a.selectedSHA != "" && a.refreshSelectedCommit.SHA == a.selectedSHA {
		return a.refreshSelectedCommit, true
	}
	return gitrepo.Commit{}, false
}

// selectedCommitIndex is the selected commit's row in the list the history
// page draws, or -1 when it is not on it. The graph is laid out for those same
// rows, so the details pane can ask which line the commit was drawn on.
func (a *app) selectedCommitIndex() int {
	for i, commit := range a.historyRows() {
		if commit.SHA == a.selectedSHA {
			return i
		}
	}
	return -1
}

func (a *app) canCommit() bool { return a.commitDisabledReason() == "" }

// The button, keyboard shortcut and write entry point share the same guards.
func (a *app) commitDisabledReason() string {
	if a.busy {
		return a.t("commitWaiting")
	}
	if a.loading || a.repo == nil {
		return a.t("loadingRepo")
	}
	if a.loadErr != "" {
		return a.t("reloadFailed")
	}
	summary, _, _ := strings.Cut(a.commitText, "\n")
	if strings.TrimSpace(summary) == "" {
		return a.t("commitTitleRequired")
	}
	if !a.commitIdentityReady() {
		if !a.commitIdentity.Loaded || a.commitIdentity.Root != a.repo.Root {
			return a.t("commitIdentityLoading")
		}
		return a.t("commitIdentityFailed")
	}
	// Rebase/am continue through their own command. All unresolved entries
	// block both Commit and Amend, even when other files are already staged.
	if a.operation.Kind != "" && !commitOperation(a.operation.Kind) {
		return a.tf("operationContinuePrompt", a.operation.Kind)
	}
	for _, entry := range a.status.Entries {
		if entry.Conflict {
			return a.t("commitConflictsHint")
		}
	}
	if a.amend || len(a.stagedEntries()) > 0 {
		return ""
	}
	// Merge/cherry-pick/revert may finish with a tree identical to HEAD.
	if commitOperation(a.operation.Kind) {
		return ""
	}
	return a.t("stagePrompt")
}

// commitOperation reports whether an interrupted operation is finished by a
// plain commit. A rebase continues through its own command instead.
func commitOperation(kind string) bool {
	switch kind {
	case "merge", "cherry-pick", "revert":
		return true
	}
	return false
}

// operationConflicts reports whether unresolved files still block the
// operation from continuing.
func (a *app) operationConflicts() bool {
	for _, entry := range a.status.Entries {
		if entry.Conflict {
			return true
		}
	}
	return false
}

func (a *app) repoName() string {
	if a.repo == nil {
		return ""
	}
	return path.Base(a.repo.Root)
}
