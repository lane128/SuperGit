package lobedesk

import (
	"testing"

	"supergit/internal/gitrepo"
)

// The line selection belongs to a patch the reader can see. Hiding the details
// pane, or the Diff block in Settings, must hand the arrow keys back to the
// commit list instead of moving a cursor nothing draws; showing the patch
// again brings the selection back, so the state itself stays.
func TestHiddenCommitDetailsReleaseTheLineKeys(t *testing.T) {
	newApp := func() *app {
		a := &app{
			page:       "history",
			detailTab:  detailTabFile,
			commitPath: "a.go",
			commitDiff: gitrepo.Diff{Lines: []gitrepo.DiffLine{
				{Kind: "hunk", Text: "@@ -1 +1 @@"},
				{Kind: "add", Text: "one", New: 1},
				{Kind: "add", Text: "two", New: 2},
			}},
			commitSections: []commitDiffSection{{Path: "a.go", Start: 0, Rows: []int{0, 1, 2}}},
		}
		a.selectCommitLines([]int{1})
		a.commitAnchor = 1
		a.commitCursor = 1
		return a
	}

	a := newApp()
	if !a.commitDiffShown() {
		t.Fatal("the details are open; the patch should count as shown")
	}
	if !a.moveCommitLineSelection(1, false) {
		t.Fatal("the line arrows did not move the selection while the patch is shown")
	}
	if a.commitCursor != 2 || !a.commitSel[2] {
		t.Errorf("after one down, cursor = %d selection = %v; want row 2", a.commitCursor, a.commitSel)
	}

	a = newApp()
	a.historyDetailsHidden = true
	if a.commitDiffShown() {
		t.Error("a closed details pane still counts as showing the patch")
	}
	if a.moveCommitLineSelection(1, false) {
		t.Error("the line arrows moved a selection the closed pane hides")
	}
	if a.commitCursor != 1 || !a.commitSel[1] {
		t.Errorf("a hidden patch's selection changed to cursor %d %v", a.commitCursor, a.commitSel)
	}
	a.historyDetailsHidden = false
	if !a.moveCommitLineSelection(1, false) {
		t.Error("reopening the pane did not restore the line arrows")
	}

	a = newApp()
	a.detailsDiffHidden = true
	if a.commitDiffShown() {
		t.Error("a Diff block hidden in Settings still counts as showing the patch")
	}
	if a.moveCommitLineSelection(1, false) {
		t.Error("the line arrows moved while Settings hides the Diff block")
	}

	// A patch that is still loading, failed, or came back empty has no rows
	// on screen either, so the keys belong to the commit list until it does.
	// The file tab gives way to the Summary when the reader goes back to it:
	// the patch is off screen then, so the keys belong to the list.
	a = newApp()
	a.detailTab = detailTabSummary
	if a.commitDiffShown() {
		t.Error("the Summary in front still counts as showing the patch")
	}
	if a.moveCommitLineSelection(1, false) {
		t.Error("the line arrows moved while the Summary covers the patch")
	}
	a.detailTab = detailTabFile
	if !a.commitDiffShown() {
		t.Error("going back to the file tab did not bring the patch back")
	}

	a = newApp()
	a.commitDiffLoading = true
	if a.commitDiffShown() {
		t.Error("a patch still loading counted as shown")
	}
	if a.moveCommitLineSelection(1, false) {
		t.Error("the line arrows moved while the patch was still loading")
	}
	a = newApp()
	a.commitDiffErr = "fatal: bad object"
	if a.commitDiffShown() {
		t.Error("a failed patch counted as shown")
	}
	a = newApp()
	a.commitDiff = gitrepo.Diff{}
	if a.commitDiffShown() {
		t.Error("an empty patch counted as shown")
	}
}

// A file the selected commit deleted has no version at that revision, so its
// Blame item cannot run. A submodule pointer is a mode, not text, so it gets
// the same treatment; the set is built once and both menus read it.
func TestCommitBlameDisabledCoversDeletedFilesAndSubmodules(t *testing.T) {
	files := []gitrepo.ChangedFile{
		{Path: "gone.go", Status: "D"},
		{Path: "kept.go", Status: "M"},
		{Path: "sub", Status: "M"},
	}
	a := &app{selectedSHA: "aaaa", commitNoBlameSHA: "aaaa", commitNoBlame: commitNoBlamePaths(files)}
	_, sections := buildCommitDiff([]gitrepo.FileDiff{
		{Path: "sub", Diff: gitrepo.Diff{Lines: []gitrepo.DiffLine{
			{Kind: "meta", Text: "diff --git a/sub b/sub"},
			{Kind: "meta", Text: "index 1111111..2222222 160000"},
		}}},
		{Path: "kept.go", Diff: gitrepo.Diff{Lines: []gitrepo.DiffLine{
			{Kind: "meta", Text: "diff --git a/kept.go b/kept.go"},
			{Kind: "meta", Text: "index 1111111..2222222 100644"},
		}}},
	})
	if !sections[0].Submodule || sections[1].Submodule {
		t.Fatalf("submodule flags = %v/%v; want true/false", sections[0].Submodule, sections[1].Submodule)
	}
	addSubmoduleNoBlame(a.commitNoBlame, sections)
	if !a.commitBlameDisabled("gone.go") {
		t.Error("a deleted file should not take blame at the commit")
	}
	if !a.commitBlameDisabled("sub") {
		t.Error("a submodule pointer should not take blame at the commit")
	}
	if a.commitBlameDisabled("kept.go") || a.commitBlameDisabled("other.go") {
		t.Error("a changed or unknown file should take blame")
	}
	// A stale map from another commit must not disable a path in this one.
	a.selectedSHA = "bbbb"
	if a.commitBlameDisabled("gone.go") {
		t.Error("another commit's map disabled blame for this one")
	}
}

// Opening another file's history drops the previous file's rows before the
// walk starts; otherwise the list keeps drawing the old file's commits under
// the new file's header until the walk returns.
func TestSwitchFileHistoryClearsTheOldRows(t *testing.T) {
	a := &app{
		fileFilter:     "old.go",
		fileCommits:    []gitrepo.Commit{{SHA: "aaaa", Subject: "old"}},
		fileHistoryErr: "stale failure",
	}
	// No repository, so the walk does not start; the state it would start
	// from is what this checks.
	a.switchFileHistory("new.go")
	if a.fileFilter != "new.go" {
		t.Errorf("filter = %q; want new.go", a.fileFilter)
	}
	if len(a.fileCommits) != 0 {
		t.Errorf("the old file's rows survived: %v", a.fileCommits)
	}
	if a.fileHistoryErr != "" {
		t.Errorf("the old file's error survived: %q", a.fileHistoryErr)
	}
}

// A path that ends the commit as a regular file can be blamed, even when the
// change replaced a gitlink: the deleted section is the old shape of the same
// path. The other direction stays disabled -- the path is a pointer now.
func TestSubmoduleBlameFollowsTheFinalShapeOfThePath(t *testing.T) {
	meta := func(texts ...string) gitrepo.Diff {
		lines := make([]gitrepo.DiffLine, 0, len(texts))
		for _, text := range texts {
			lines = append(lines, gitrepo.DiffLine{Kind: "meta", Text: text})
		}
		return gitrepo.Diff{Lines: lines}
	}
	for _, test := range []struct {
		name     string
		files    []gitrepo.FileDiff
		disabled bool
	}{
		{
			name: "a gitlink replaced by a file",
			files: []gitrepo.FileDiff{
				{Path: "sub", Diff: meta("deleted file mode 160000", "index 1111111..0000000")},
				{Path: "sub", Diff: meta("new file mode 100644", "index 0000000..2222222")},
			},
		},
		{
			name: "a file replaced by a gitlink",
			files: []gitrepo.FileDiff{
				{Path: "sub", Diff: meta("deleted file mode 100644", "index 1111111..0000000")},
				{Path: "sub", Diff: meta("new file mode 160000", "index 0000000..2222222")},
			},
			disabled: true,
		},
		{
			name: "a gitlink added",
			files: []gitrepo.FileDiff{
				{Path: "sub", Diff: meta("new file mode 160000", "index 0000000..2222222")},
			},
			disabled: true,
		},
		{
			name: "a gitlink pointer moved",
			files: []gitrepo.FileDiff{
				{Path: "sub", Diff: meta("index 1111111..2222222 160000")},
			},
			disabled: true,
		},
	} {
		_, sections := buildCommitDiff(test.files)
		noBlame := map[string]bool{}
		addSubmoduleNoBlame(noBlame, sections)
		if got := noBlame["sub"]; got != test.disabled {
			t.Errorf("%s: blame disabled = %v; want %v", test.name, got, test.disabled)
		}
	}
}

// A diff row's accessible handle is its line numbers, or a word for rows that
// have none; the word follows the interface language instead of always being
// Chinese.
func TestDiffLineAccessibilityNameFollowsTheLanguage(t *testing.T) {
	for _, test := range []struct {
		locale locale
		want   string
	}{
		{localeChinese, "行"},
		{localeEnglish, "Line"},
	} {
		a := &app{locale: test.locale}
		if got := a.lineGutterLabel(gitrepo.DiffLine{Kind: "meta", Text: "diff --git a/x b/x"}); got != test.want {
			t.Errorf("%v: a bare patch row reads %q; want %q", test.locale, got, test.want)
		}
		if got := a.lineGutterLabel(gitrepo.DiffLine{Old: 3, New: 4}); got != "3 4" {
			t.Errorf("%v: a numbered row reads %q; want the numbers", test.locale, got)
		}
	}
}
