package lobedesk

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func waitVersionSwitchDiff(t *testing.T, a *app, callbacks <-chan func()) {
	t.Helper()
	for a.diffLoading {
		runContinuation(t, callbacks)
	}
	if a.diffErr != "" {
		t.Fatal(a.diffErr)
	}
}

func versionSwitchToggle(t *testing.T, tt *ui.Tester, label string) ui.AccessInfo {
	t.Helper()
	button, ok := tt.AccessByRole(ui.RoleToggleButton, label)
	if !ok {
		t.Fatalf("file diff header is missing toggle %q", label)
	}
	if button.Disabled || button.Offscreen || button.Bounds.W <= 0 || button.Bounds.H <= 0 {
		t.Fatalf("toggle %q is unusable: %+v", label, button)
	}
	return button
}

func TestWorktreeFileVersionSwitchRoundTrip(t *testing.T) {
	const path = "same.go"
	for _, start := range []struct {
		name   string
		staged bool
	}{{"from unstaged", false}, {"from staged", true}} {
		t.Run(start.name, func(t *testing.T) {
			a, callbacks := overviewFixture(t)
			a.page, a.locale = "history", localeEnglish
			root := a.repo.Root
			if got := splitCommitGit(t, root, "status", "--porcelain=v1"); got != "MM same.go\n" {
				t.Fatalf("expected one partially staged file, got %q", got)
			}
			type version struct {
				staged                bool
				label, addedLine, raw string
			}
			working := version{false, "Unstaged", "+working version", splitCommitGit(t, root, "diff", "--", path)}
			index := version{true, "Staged", "+index version", splitCommitGit(t, root, "diff", "--cached", "--", path)}
			first, other := working, index
			if start.staged {
				first, other = index, working
			}
			a.openDetailFile(detailFileKey{Path: path, Staged: first.staged})
			waitVersionSwitchDiff(t, a, callbacks)
			tt := pageTester(a, a.historyDiffPane)
			checkVersion := func(v version, count int) {
				t.Helper()
				want := detailFileKey{Path: path, Staged: v.staged}
				tabs := a.detailTabs()
				if len(tabs.OpenFiles) != count || tabs.Active != (detailTabKey{File: want}) || a.selectedPath != path || a.selectedStaged != v.staged {
					t.Fatalf("want %v active with %d file tabs, got active=%v open=%v path=%q staged=%t", want, count, tabs.Active, tabs.OpenFiles, a.selectedPath, a.selectedStaged)
				}
				if a.diff.Text != v.raw || !tt.HasText(v.addedLine) {
					t.Fatalf("want displayed %q and raw Git patch %q, got %q", v.addedLine, v.raw, a.diff.Text)
				}
				for _, side := range []version{working, index} {
					if got := versionSwitchToggle(t, tt, side.label).Checked; got != (side.staged == v.staged) {
						t.Fatalf("reading %s: %s checked=%t", v.label, side.label, got)
					}
				}
			}
			current := first
			for step, next := range []version{other, first} {
				gutter, ok := tt.AccessByRole(ui.RoleButton, "1 "+a.t("diffAddedLine"))
				if !ok {
					t.Fatalf("missing line-number button for %q", current.addedLine)
				}
				tt.ClickAt(gutter.Bounds.X+gutter.Bounds.W/2, gutter.Bounds.Y+gutter.Bounds.H/2)
				if len(a.lineSel) != 1 {
					t.Fatalf("clicking the gutter for %q left selection %v", current.addedLine, a.lineSel)
				}
				if step == 0 {
					checkVersion(first, 1)
				}
				button := versionSwitchToggle(t, tt, next.label)
				tt.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
				waitVersionSwitchDiff(t, a, callbacks)
				tt.Frame()
				checkVersion(next, 2)
				if len(a.lineSel) != 0 {
					t.Fatalf("selection from %s survived switch to %s: %v", current.label, next.label, a.lineSel)
				}
				if got := splitCommitGit(t, root, "show", ":"+path); got != "index version\n" {
					t.Fatalf("switch to %s changed the index: %q", next.label, got)
				}
				if got, err := os.ReadFile(filepath.Join(root, path)); err != nil || string(got) != "working version\n" {
					t.Fatalf("switch to %s changed the working file: %q, %v", next.label, got, err)
				}
				current = next
			}
		})
	}
}

func TestWorktreeFileVersionSwitchHiddenWithoutCounterpart(t *testing.T) {
	for _, tc := range []struct {
		name   string
		entry  gitrepo.Entry
		staged bool
	}{
		{"unstaged only", gitrepo.Entry{Path: "only.go", Index: '.', Worktree: 'M'}, false},
		{"staged only", gitrepo.Entry{Path: "only.go", Index: 'M', Worktree: '.'}, true},
		{"unmerged", gitrepo.Entry{Path: "only.go", Index: 'U', Worktree: 'U', Conflict: true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &app{page: "history", locale: localeEnglish, worktreeSelected: true, selectedPath: tc.entry.Path, selectedStaged: tc.staged,
				status: gitrepo.Status{Entries: []gitrepo.Entry{tc.entry, {Path: "other.go", Index: 'M', Worktree: 'M'}}}}
			a.detailTabs().Open(detailFileKey{Path: tc.entry.Path, Staged: tc.staged})
			tt := pageTester(a, func(c *ui.Context) { a.worktreeDiffHeader(c, theme.TokensOf(c)) })
			if !tt.HasText(tc.entry.Path) {
				t.Fatal("selected file header is missing")
			}
			for _, label := range []string{"Unstaged", "Staged"} {
				if _, ok := tt.AccessByRole(ui.RoleToggleButton, label); ok {
					t.Errorf("unexpected version toggle %q", label)
				}
			}
		})
	}
}

func TestWorktreeFileVersionSwitchNarrowKeyboard(t *testing.T) {
	previous := uiFontScaleValue()
	setUIFontScale(fontScaleLarge)
	t.Cleanup(func() { setUIFontScale(previous) })
	a, callbacks := overviewFixture(t)
	a.page, a.locale = "history", localeChinese
	a.openDetailFile(detailFileKey{Path: "same.go", Staged: true})
	waitVersionSwitchDiff(t, a, callbacks)
	const width, height = 300, 180
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Children(func() { a.worktreeDiffHeader(c, theme.TokensOf(c)) })
	}, width, height)
	left := versionSwitchToggle(t, tt, "未暂存").Bounds
	right := versionSwitchToggle(t, tt, "已暂存").Bounds
	path, ok := tt.Find("same.go")
	if !ok || min(left.Y, right.Y) < path.Y+path.H || left.X+left.W > right.X || left.Y >= right.Y+right.H || right.Y >= left.Y+left.H {
		t.Fatalf("expected a left-to-right version row below the file name: path=%v unstaged=%v staged=%v", path, left, right)
	}
	for _, b := range []ui.Rect{left, right} {
		if b.X < 0 || b.Y < 0 || b.X+b.W > width || b.Y+b.H > height {
			t.Fatalf("version toggle outside %dx%d viewport: %v", width, height, b)
		}
	}
	for range len(tt.AccessAll()) + 1 {
		tt.Key(0, ui.KeyTab)
		if versionSwitchToggle(t, tt, "已暂存").Focused {
			tt.Key(0, ui.KeyLeft)
		}
		if versionSwitchToggle(t, tt, "未暂存").Focused {
			break
		}
	}
	if !versionSwitchToggle(t, tt, "未暂存").Focused {
		t.Fatal("keyboard focus did not reach the unstaged version")
	}
	tt.Key(0, ui.KeySpace)
	waitVersionSwitchDiff(t, a, callbacks)
	tt.Frame()
	if a.selectedStaged || !versionSwitchToggle(t, tt, "未暂存").Checked || versionSwitchToggle(t, tt, "已暂存").Checked {
		t.Fatal("Space did not activate the focused unstaged version")
	}
}
