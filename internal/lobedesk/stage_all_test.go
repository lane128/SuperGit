package lobedesk

import (
	"testing"

	"supergit/internal/gitrepo"
)

// Stage all adds whole files; a conflict needs its resolution chosen first, so
// the count the header uses to enable the button leaves conflicts out. A list
// of nothing but conflicts therefore leaves the button disabled, with a tooltip
// that says why.
func TestStageableCountSkipsConflicts(t *testing.T) {
	unstaged := []gitrepo.Entry{
		{Path: "a.go", Index: '.', Worktree: 'M'},
		{Path: "both.go", Index: 'U', Worktree: 'U', Conflict: true},
	}
	untracked := []gitrepo.Entry{{Path: "new.go", Index: '?', Worktree: '?'}}
	if got := stageableCount(unstaged, untracked); got != 2 {
		t.Errorf("stageableCount = %d; want the changed and untracked files only", got)
	}
	if got := stageableCount([]gitrepo.Entry{{Path: "both.go", Conflict: true}}, nil); got != 0 {
		t.Errorf("a list of only conflicts counted %d stageable files", got)
	}
	if got := stageableCount(nil, nil); got != 0 {
		t.Errorf("an empty list counted %d", got)
	}
}
