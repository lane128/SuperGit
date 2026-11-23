package lobedesk

import (
	"supergit/internal/gitrepo"
	"testing"
)

func TestStageTrackedScopeUsesIndexTrackingAndExistingProtection(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: t.TempDir()}, locale: localeEnglish, status: gitrepo.Status{Entries: []gitrepo.Entry{
		{Path: "modified.txt", Index: '.', Worktree: 'M'},
		{Path: "added-then-edited.txt", Index: 'A', Worktree: 'M'},
		{Path: "intent.txt", Index: '.', Worktree: 'A'},
		{Path: "deleted.txt", Index: '.', Worktree: 'D'},
		{Path: "new.txt", Index: '?', Worktree: '?'},
		{Path: "staged-only.txt", Index: 'A', Worktree: '.'},
	}}}
	targets, reason := a.batchFileTargets("stageTracked")
	if reason != "" || len(targets) != 4 {
		t.Fatalf("wrong tracked scope: %+v %s", targets, reason)
	}
	for _, target := range targets {
		if target.Entry.Path == "new.txt" || target.Entry.Path == "staged-only.txt" {
			t.Fatal("wrong target included")
		}
	}
	ordinary := a.status.Entries[0]
	for _, tc := range []struct {
		name, submodule string
		conflict        bool
		want            int
	}{
		{name: "conflict", conflict: true},
		{name: "dirty-only child", submodule: "S.M.", want: 1},
		{name: "changed gitlink", submodule: "SC..", want: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a.status.Entries = []gitrepo.Entry{ordinary, {Path: "module", Index: '.', Worktree: 'M', Submodule: tc.submodule, Conflict: tc.conflict}}
			got, reason := a.batchFileTargets("stageTracked")
			if len(got) != tc.want || (reason != "") != (tc.want == 0) {
				t.Fatalf("want %d targets: got %+v, %q", tc.want, got, reason)
			}
			if tc.want == 1 && got[0].Entry != ordinary {
				t.Fatalf("dirty-only child displaced ordinary parent target: %+v", got)
			}
		})
	}
}
