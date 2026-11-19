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
		name, submodule   string
		conflict, allowed bool
	}{
		{name: "conflict", conflict: true},
		{name: "dirty-only child", submodule: "S.M."},
		{name: "changed gitlink", submodule: "SC..", allowed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a.status.Entries = []gitrepo.Entry{ordinary, {Path: "module", Index: '.', Worktree: 'M', Submodule: tc.submodule, Conflict: tc.conflict}}
			got, reason := a.batchFileTargets("stageTracked")
			if tc.allowed {
				if reason != "" || len(got) != 2 {
					t.Fatalf("supported parent pointer rejected: %s", reason)
				}
			} else if reason == "" || len(got) != 0 {
				t.Fatal("protected entry did not block entire batch")
			}
		})
	}
}
