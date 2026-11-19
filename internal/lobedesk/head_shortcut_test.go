package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func headShortcutApp(t *testing.T) *app {
	t.Helper()
	return &app{ctx: canceledContext(), runOnMain: func(func()) {}, locale: localeEnglish, page: "history", repo: &gitrepo.Repository{Root: t.TempDir()}, status: gitrepo.Status{Head: "head-sha"}, commits: []gitrepo.Commit{{SHA: "newer-other-branch"}, {SHA: "head-sha"}}, selectedSHA: "head-sha", commitFilesLoaded: true, worktreeSelected: true, commitText: "retained draft"}
}

func pressHeadShortcut(a *app) {
	tt := ui.NewTester(func(c *ui.Context) { a.registerShortcuts(c); ui.Column(c).Fill() }, 400, 200)
	tt.Key(ui.Cmd, ui.Key0)
}

func TestHeadShortcutSelectsActualHeadAndReveals(t *testing.T) {
	a := headShortcutApp(t)
	a.page = "stash"
	a.historyQuery = "hides HEAD"
	pressHeadShortcut(a)
	if a.page != "history" || a.selectedSHA != a.status.Head || a.worktreeSelected || !a.revealSelection || a.historyQuery != "" || a.commitText != "retained draft" {
		t.Fatal("HEAD navigation did not select/reveal actual HEAD while retaining draft")
	}
}

func TestHeadShortcutWorksFromTextInput(t *testing.T) {
	a := headShortcutApp(t)
	text := "draft text"
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.registerShortcuts(c)
		field.Input(c, &text, field.InputProps{}).Label("editor")
	}, 400, 200)
	row, ok := tt.AccessByRole(ui.RoleTextField, "editor")
	if !ok {
		t.Fatal("missing editor")
	}
	tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	tt.Key(ui.Cmd, ui.Key0)
	if !a.revealSelection || text != "draft text" || !tt.Focused("editor") {
		t.Fatal("HEAD navigation was swallowed by input or changed input/focus")
	}
}

func TestHeadShortcutRespectsGuards(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*app)
	}{
		{"no repo", func(a *app) { a.repo = nil }},
		{"empty HEAD", func(a *app) { a.status.Head = "" }},
		{"busy", func(a *app) { a.busy = true }},
		{"loading", func(a *app) { a.loading = true }},
		{"history reload", func(a *app) { a.historyReloading = true }},
		{"chooser", func(a *app) { a.repositoryChooserOpen = true }},
		{"ref picker", func(a *app) { a.historyRefOpen = true }},
		{"confirmation", func(a *app) { a.confirm.open = true; a.confirm.owner = toastMain }},
		{"prompt", func(a *app) { a.prompt.open = true }},
		{"reset", func(a *app) { a.reset.open = true }},
		{"checkout", func(a *app) { a.checkoutBranch.open = true }},
		{"worktree", func(a *app) { a.worktreeForm.open = true }},
		{"blame", func(a *app) { a.blameOpen = true }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := headShortcutApp(t)
			a.page = "stash"
			a.selectedSHA = "original"
			tc.setup(a)
			pressHeadShortcut(a)
			if a.page != "stash" || a.selectedSHA != "original" || a.revealSelection || a.pendingJumpSHA != "" {
				t.Fatal("guarded HEAD navigation changed context")
			}
		})
	}
}

func TestHeadShortcutLeavesFileHistoryThatPausedFullTraversal(t *testing.T) {
	a := headShortcutApp(t)
	a.fileFilter = "old-only.txt"
	a.fileCommits = []gitrepo.Commit{{SHA: "old-file-commit"}}
	a.historyPaused = true // switchFileHistory pauses full traversal as part of opening the filter.
	pressHeadShortcut(a)
	if a.fileFilter != "" || a.historyPaused || a.selectedSHA != a.status.Head || !a.revealSelection || len(a.notices[toastMain]) != 0 {
		t.Fatal("file-history pause prevented navigation to the loaded full-history HEAD")
	}
}

func TestHeadShortcutCanRevealLoadedPartialHistory(t *testing.T) {
	for _, paused := range []bool{false, true} {
		a := headShortcutApp(t)
		a.historyPaused = paused
		if !paused {
			a.historyPageErr = "partial read failed"
		}
		pressHeadShortcut(a)
		if a.selectedSHA != a.status.Head || a.worktreeSelected || !a.revealSelection || a.historyPaused != paused || !paused && a.historyPageErr != "partial read failed" || len(a.notices[toastMain]) != 0 {
			t.Fatal("loaded HEAD navigation reset partial-history state or failed to reveal")
		}
	}
}

func TestHeadShortcutBlockedHistoryDoesNotLeavePendingJump(t *testing.T) {
	for _, reason := range []string{"paused", "page error", "read error"} {
		t.Run(reason, func(t *testing.T) {
			a := headShortcutApp(t)
			a.commits = nil
			a.historyMore = true
			a.pendingJumpSHA = "old-jump"
			switch reason {
			case "paused":
				a.historyPaused = true
			case "page error":
				a.historyPageErr = "read failed"
			case "read error":
				a.historyErr = "read failed"
			}
			pressHeadShortcut(a)
			if a.pendingJumpSHA != "" || a.historyLoadingMore || len(a.notices[toastMain]) != 1 {
				t.Fatal("blocked HEAD jump lacks feedback or retains an unfulfillable jump")
			}
			if reason == "paused" && !a.historyPaused || reason == "page error" && a.historyPageErr != "read failed" || reason == "read error" && a.historyErr != "read failed" {
				t.Fatal("HEAD jump reset history failure state")
			}
		})
	}
}
