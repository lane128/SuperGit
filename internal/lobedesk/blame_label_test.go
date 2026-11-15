package lobedesk

import (
	"testing"

	"supergit/internal/gitrepo"
)

// A file the commit cannot blame (added there, a submodule pointer, or a
// deletion) keeps its blame item, dimmed. A dimmed item with no reason reads as
// a broken menu, so the label says why instead of leaving the reader guessing.
func TestBlameMenuLabelSaysWhyTheItemIsDisabled(t *testing.T) {
	for _, test := range []struct {
		name        string
		sha         string
		mapSHA      string
		noBlame     map[string]bool
		path        string
		unavailable bool
	}{
		{"the commit cannot blame the file", "bbb", "bbb", map[string]bool{"gone.go": true}, "gone.go", true},
		{"another file of the same commit", "bbb", "bbb", map[string]bool{"gone.go": true}, "kept.go", false},
		{"a map left over from another commit", "ccc", "bbb", map[string]bool{"gone.go": true}, "gone.go", false},
		{"no map at all", "bbb", "", nil, "gone.go", false},
	} {
		a := &app{
			locale:           localeChinese,
			selectedSHA:      test.sha,
			commitNoBlame:    test.noBlame,
			commitNoBlameSHA: test.mapSHA,
		}
		want := a.t("blame")
		if test.unavailable {
			want = a.t("blameUnavailable")
		}
		if got := a.blameMenuLabel(test.path); got != want {
			t.Errorf("%s: label = %q; want %q", test.name, got, want)
		}
		if got := a.commitBlameDisabled(test.path); got != test.unavailable {
			t.Errorf("%s: disabled = %v; want %v", test.name, got, test.unavailable)
		}
	}
}

// The workspace menu's blame item is dimmed for a submodule, an untracked
// file, a file added to the index, and a deleted file. Each of those says why,
// as the commit menu does.
func TestWorkspaceBlameMenuSaysWhyItIsDisabled(t *testing.T) {
	a := &app{locale: localeChinese}
	for _, entry := range []gitrepo.Entry{
		{Path: "sub", Submodule: "S.M."},
		{Path: "new.txt", Index: '?', Worktree: '?'},
		{Path: "added.go", Index: 'A'},
		{Path: "gone.go", Index: ' ', Worktree: 'D'},
	} {
		if got := a.workspaceBlameLabel(entry); got != a.t("blameUnavailableFile") {
			t.Errorf("%q reads %q; want the reason for the disabled item", entry.Path, got)
		}
	}
	if got := a.workspaceBlameLabel(gitrepo.Entry{Path: "kept.go", Index: ' ', Worktree: 'M'}); got != a.t("blame") {
		t.Errorf("a changed file reads %q; want the plain blame label", got)
	}
}
