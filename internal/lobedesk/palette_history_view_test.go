package lobedesk

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"supergit/internal/gitrepo"
)

func TestPaletteHistoryViewExpandDefaultAndOverrides(t *testing.T) {
	a := &app{commits: mergeHistoryFixture(), showGraph: true, expandMerges: true}
	if rows := a.projectHistoryMerges(a.commits); len(rows) != 5 {
		t.Fatalf("expand default: %v", rows)
	}
	a.toggleHistoryMerge("merge")
	if rows := a.projectHistoryMerges(a.commits); len(rows) != 3 {
		t.Fatalf("manual collapse: %v", rows)
	}
	if rows := a.projectHistoryMerges(slices.Clone(a.commits)); len(rows) != 3 {
		t.Fatalf("reload lost manual override: %v", rows)
	}
	a.expandMerges = false
	if rows := a.projectHistoryMerges(a.commits); len(rows) != 3 {
		t.Fatalf("fold default: %v", rows)
	}
	a.selectedSHA = "side"
	if rows := a.projectHistoryMerges(a.commits); len(rows) != 5 {
		t.Fatalf("selection hidden: %v", rows)
	}
}

func TestPaletteHistoryViewBranchTargetsAreScoped(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "one"}, hiddenBranches: map[string][]string{"one": {"refs/remotes/team/topic"}}, branches: []gitrepo.Branch{{Name: "team/topic", SHA: "a"}}, remoteBranches: []gitrepo.Branch{{Name: "team/topic", SHA: "b"}}}
	show := a.historyBranchChoices("show")
	hide := a.historyBranchChoices("hide")
	if len(show) != 1 || show[0].sha != "b" || len(hide) != 1 || hide[0].sha != "a" {
		t.Fatalf("qualified refs: show %+v hide %+v", show, hide)
	}
	a.repo = &gitrepo.Repository{Root: "two"}
	if rows := a.historyBranchChoices("hide"); len(rows) != 2 {
		t.Fatalf("other repository inherited hidden refs: %v", rows)
	}
}

func TestPaletteHistoryViewAsyncParentAndCancellation(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		a := &app{ctx: context.Background(), repo: &gitrepo.Repository{Root: t.TempDir()}, locale: localeEnglish}
		callbacks := make(chan func(), 1)
		a.runOnMain = func(fn func()) { callbacks <- fn }
		a.choosePalette("parent", "", nil, func(paletteChoice) {})
		parent := a.commandPalette.step
		continued := false
		readHistoryPalette(a, "reading", func(context.Context, *gitrepo.Repository) (string, error) { return "ok", nil }, func(result string) { continued = true; a.choosePalette(result, "", nil, func(paletteChoice) {}) })
		if cancel {
			a.commandPalette = commandPaletteState{}
		}
		select {
		case fn := <-callbacks:
			fn()
		case <-time.After(3 * time.Second):
			t.Fatal("worker timed out")
		}
		if cancel {
			if continued || a.commandPalette.open {
				t.Fatal("cancelled picker reopened")
			}
			continue
		}
		if !continued || a.commandPalette.step.parent != parent || a.commandPalette.step.title != "ok" {
			t.Fatal("Back should skip the loading placeholder")
		}
	}
}

func TestPaletteHistoryViewBrowseDirectoryAndBlob(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.locale = localeEnglish
	root := a.repo.Root
	splitCommitGit(t, root, "mv", "same.go", "folder-name")
	splitCommitGit(t, root, "commit", "-m", "move file")
	tree := strings.TrimSpace(splitCommitGit(t, root, "rev-parse", "HEAD^{tree}"))
	// Reuse this committed directory as a subtree without following worktree paths.
	splitCommitGit(t, root, "read-tree", "--prefix=nested/", tree)
	splitCommitGit(t, root, "commit", "-m", "nested tree")
	rev := strings.TrimSpace(splitCommitGit(t, root, "rev-parse", "HEAD"))
	a.status.Head = rev
	runRead := func() {
		t.Helper()
		select {
		case fn := <-callbacks:
			fn()
		case <-time.After(5 * time.Second):
			t.Fatal("Git read timed out")
		}
	}
	choose := func(id string) {
		t.Helper()
		s := a.commandPalette.step
		if s == nil {
			t.Fatal("missing file picker")
		}
		for _, r := range s.options {
			if r.id == id {
				a.finishPaletteStep(a.commandPalette.generation, func() { s.choose(r) })
				return
			}
		}
		t.Fatalf("file %q missing: %+v", id, s.options)
	}
	a.openSnapshotFilePicker("tree")
	runRead()
	top := a.commandPalette.step
	choose("nested")
	runRead()
	folder := a.commandPalette.step
	if folder.parent != top || !strings.Contains(folder.title, "nested") {
		t.Fatal("directory lost its path or parent")
	}
	choose("folder-name")
	runRead()
	v := a.snapshotView
	if !v.open || v.path != "nested/folder-name" || v.rev != rev || v.text != "index version\n" || v.parent != folder || a.commandPalette.open {
		t.Fatalf("file preview lost its contents or browsing context: %+v", v)
	}
}

func TestPaletteHistoryViewThemeModesAndColorsAreIndependent(t *testing.T) {
	path := isolateSettings(t)
	a := &app{locale: localeEnglish, themeMode: "system", palette: themePaletteSuperGit}
	choose := func(id string) {
		t.Helper()
		a.chooseHistoryTheme()
		s := a.commandPalette.step
		if len(s.options) != 3+len(appThemes) {
			t.Fatalf("theme choices: %+v", s.options)
		}
		for _, row := range s.options {
			active := row.id == a.palette || row.id == "appearance."+a.themeMode
			if (row.prefix == "✓") != active {
				t.Fatalf("incorrect active state: %+v", row)
			}
		}
		for _, row := range s.options {
			if row.id == id {
				a.finishPaletteStep(a.commandPalette.generation, func() { s.choose(row) })
				return
			}
		}
		t.Fatalf("missing theme option %q", id)
	}
	for _, mode := range []string{"dark", "light", "system"} {
		choose("appearance." + mode)
		if a.palette != themePaletteSuperGit || a.themeMode != mode {
			t.Fatalf("appearance changed color scheme: %q %q", a.palette, a.themeMode)
		}
	}
	choose("appearance.dark")
	choose(themePaletteCatppuccinFrappe)
	if a.palette != themePaletteCatppuccinFrappe || a.themeMode != "dark" {
		t.Fatalf("color scheme changed appearance: %q %q", a.palette, a.themeMode)
	}
	saved := readSettings(path)
	if saved.Theme != a.themeMode || saved.Palette != a.palette {
		t.Fatalf("theme choices not persisted: %+v", saved)
	}
}
