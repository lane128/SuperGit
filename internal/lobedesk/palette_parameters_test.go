package lobedesk

import (
	"strings"
	"testing"
	"time"

	"supergit/internal/gitrepo"
)

func TestPaletteMergeIdenticalNamesAndMovedRef(t *testing.T) {
	a := headShortcutApp(t)
	a.status.Branch = "main"
	a.branches = []gitrepo.Branch{{Name: "main", SHA: a.status.Head, Current: true}, {Name: "origin/topic", SHA: "1111111111"}}
	a.remoteBranches = []gitrepo.Branch{{Name: "origin/topic", SHA: "2222222222"}}
	a.openCommandPalette()
	a.runPaletteCommand("branch.merge", a.commandPalette.generation)
	rows := a.paletteChoices()
	if len(rows) != 2 || rows[0].id != "refs/heads/origin/topic" || rows[0].prefix != "L" ||
		rows[1].id != "refs/remotes/origin/topic" || rows[1].prefix != "R" {
		t.Fatalf("ambiguous or missing merge targets: %+v", rows)
	}
	generation := a.commandPalette.generation
	a.remoteBranches[0].SHA = "3333333333"
	a.acceptPaletteChoice(rows[1], generation)
	if a.confirm.open || !a.commandPalette.open {
		t.Fatal("a previous frame confirmed a ref that moved")
	}
	a.acceptPaletteChoice(a.paletteChoices()[1], generation)
	if !a.confirm.open || a.commandPalette.open || !strings.Contains(a.confirm.message, "refs/remotes/origin/topic") ||
		!strings.Contains(a.confirm.message, shortSHA(a.remoteBranches[0].SHA)) {
		t.Fatalf("wrong confirmation for selected remote: %+v", a.confirm)
	}
}

func TestPaletteMergeStateChangePreventsAcceptance(t *testing.T) {
	for _, change := range []string{"detached", "conflict", "operation", "read error", "repository epoch"} {
		t.Run(change, func(t *testing.T) {
			a := headShortcutApp(t)
			a.status.Branch = "main"
			a.branches = []gitrepo.Branch{{Name: "topic", SHA: "1111111111"}}
			a.setPaletteStage(paletteMergeRef, "")
			row := a.paletteChoices()[0]
			generation := a.commandPalette.generation
			switch change {
			case "detached":
				a.status.Branch = ""
			case "conflict":
				a.status.Entries = []gitrepo.Entry{{Path: "file", Conflict: true}}
			case "operation":
				a.operation.Kind = "rebase"
			case "read error":
				a.remoteBranchesErr = "unreadable"
			case "repository epoch":
				a.loadSeq++
			}
			a.acceptPaletteChoice(row, generation)
			if a.confirm.open || a.busy {
				t.Fatalf("stale choice acted after %s", change)
			}
		})
	}
}

func TestPaletteMergeAllowedWithNonConflictChanges(t *testing.T) {
	a := headShortcutApp(t)
	a.status.Branch = "main"
	a.status.Entries = []gitrepo.Entry{{Path: "notes.txt", Worktree: 'M'}}
	a.branches = []gitrepo.Branch{{Name: "main", SHA: a.status.Head, Current: true}, {Name: "topic", SHA: "1111111111"}}
	if why := a.paletteMergeUnavailable(); why != "" {
		t.Fatalf("palette merge should defer dirty checks to git, got %q", why)
	}
	a.setPaletteStage(paletteMergeRef, "")
	rows := a.paletteChoices()
	if len(rows) != 1 || rows[0].reason != "" {
		t.Fatalf("expected enabled merge row with dirty worktree, got %+v", rows)
	}
}

func TestPaletteFetchParameterGeneration(t *testing.T) {
	a := headShortcutApp(t)
	a.remotes = []gitrepo.Remote{{Name: "origin"}, {Name: "upstream"}}
	a.fetchFromCommand()
	generation := a.commandPalette.generation
	option := a.paletteChoices()[1]
	a.acceptPaletteChoice(option, generation)
	if a.commandPalette.stage != paletteFetchRemote || a.commandPalette.fetchOption != "fetch --prune" {
		t.Fatal("single-remote option must prompt for a remote")
	}
	a.acceptPaletteChoice(option, generation)
	remote, generation := a.paletteChoices()[0], a.commandPalette.generation
	a.setPaletteStage(paletteFetchOptions, "")
	a.acceptPaletteChoice(remote, generation)
	if a.fetching || !a.lastAutoFetch.IsZero() || a.commandPalette.stage != paletteFetchOptions || !a.commandPalette.open {
		t.Fatal("old fetch option or remote advanced a different stage")
	}
}

func TestPaletteFetchRemoteSurvivesBackgroundRefresh(t *testing.T) {
	a := headShortcutApp(t)
	a.autoFetch, a.autoFetchEvery = true, 1
	a.remotes = []gitrepo.Remote{{Name: "origin"}, {Name: "upstream"}}
	now := time.Now()
	probe, ok := a.beginLocalProbe(now)
	if !ok {
		t.Fatal("expected an idle repository probe")
	}
	a.setPaletteStage(paletteFetchRemote, "fetch --tags")
	if a.acceptLocalProbe(probe, "changed", nil, now) {
		t.Fatal("in-flight probe would close the remote picker")
	}
	a.tickAutoFetch()
	if _, ok := a.beginLocalProbe(now); ok || a.fetching || !a.lastAutoFetch.IsZero() {
		t.Fatal("background work started while selecting a remote")
	}
	a.commandPalette = commandPaletteState{}
	retry, ok := a.beginLocalProbe(now)
	if !ok || !a.acceptLocalProbe(retry, "changed", nil, now) {
		t.Fatal("deferred repository change lost after dismissal")
	}
}
