package lobedesk

import (
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

// A preferred wide sidebar or Files pane must not consume the patch's minimum
// width, and a narrow window must not overwrite the saved preference.
func TestWorkbenchDividersRespectMinimumWidths(t *testing.T) {
	for _, width := range []float32{980, 1320, 1600} {
		for _, share := range []float32{0, 0.01, 0.9} {
			a := &app{}
			a.sidebarSplit.rowWidth = width
			a.sidebarSplit.share = share
			sidebar := a.workbenchSidebarWidth()
			a.historySplit.rowWidth = width - sidebar - splitterWidth
			a.historySplit.share = share
			history := (a.historySplit.rowWidth - splitterWidth) * a.historyShare()
			details := a.historySplit.rowWidth - splitterWidth - history
			a.filesSplit.rowWidth = details
			a.filesSplit.share = share
			files := a.historyFilesWidth()
			patch := details - splitterWidth - files
			minHistory, minFiles := float32(260), float32(180)
			if width == 980 {
				minHistory, minFiles = 350, 96
			}
			if sidebar < 159.99 || history < minHistory-0.01 || files < minFiles-0.01 || patch < 299.99 {
				t.Fatalf("width %v share %v: sidebar/history/files/patch = %v/%v/%v/%v", width, share, sidebar, history, files, patch)
			}
			if math.Abs(float64(sidebar+history+files+patch+3*splitterWidth-width)) > 0.01 {
				t.Fatal("column budget does not match window")
			}
			if a.sidebarSplit.share != share || a.filesSplit.share != share {
				t.Fatal("effective clamping overwrote preferred width")
			}
		}
	}
}

func TestHistoryResizeWithFilesHiddenKeepsCommitListStable(t *testing.T) {
	a := &app{page: "history", detailsChangesHidden: true}
	a.sidebarSplit.share = 0.30
	var previous float32
	for _, width := range []float32{1084, 1085, 1084} {
		a.sidebarSplit.rowWidth = width
		a.historySplit.rowWidth = width - a.workbenchSidebarWidth() - splitterWidth
		list := a.commitListWidth()
		details := a.historySplit.rowWidth - splitterWidth - list
		if list < 259.99 || details < 299.99 {
			t.Fatalf("at %.0f points: list=%.1f, details=%.1f", width, list, details)
		}
		if previous != 0 && math.Abs(float64(previous-list)) > 1 {
			t.Errorf("one-point resize changed commit list from %.2f to %.2f", previous, list)
		}
		previous = list
	}
}

func TestHistoryControlsStayReachableWithDetailsHiddenAndWideSidebar(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := &app{page: "history", locale: localeEnglish, themeMode: "light", historyDetailsHidden: true,
		repo: &gitrepo.Repository{Root: t.TempDir()}, worktreeSelected: true}
	a.sidebarSplit.share = .95
	tt := changesLayoutTester(a)
	assertControls := func(windowWidth int, filter string) {
		t.Helper()
		for _, name := range []string{filter, a.t("showCommitDetails")} {
			node, ok := tt.AccessByRole(ui.RoleButton, name)
			if !ok || node.Offscreen || node.Bounds.W < 24 || node.Bounds.X+node.Bounds.W > float32(windowWidth)-5 {
				t.Errorf("%d: %s must be reachable with a wide sidebar: %+v", windowWidth, name, node.Bounds)
			}
		}
		var search ui.AccessInfo
		for _, node := range tt.AccessAll() {
			if node.Role == ui.RoleTextField && node.Placeholder == a.t("searchHistory") {
				search = node
				break
			}
		}
		if search.Offscreen || search.Bounds.W < 90 || search.Bounds.X+search.Bounds.W > float32(windowWidth)-5 {
			t.Errorf("%d: History search should fit with a wide sidebar: %+v", windowWidth, search.Bounds)
		}
	}
	for _, width := range []int{980, 1320, 1600, 980} {
		tt.SetSize(width, 640)
		tt.Frame()
		tt.Frame()
		assertControls(width, a.t("allRefs"))
	}
	longName := strings.Repeat("跨平台-history/", 12) + "changes.go"
	a.historyRef = historyRef{Kind: historyRefLocal, Name: longName}
	tt.Frame()
	assertControls(980, longName)
	a.historyRef, a.fileFilter = historyRef{}, longName
	tt.Frame()
	assertControls(980, a.t("showAllHistory"))
	clickRepositoryTabsControl(t, tt, ui.RoleButton, a.t("showCommitDetails"))
	if a.historyDetailsHidden {
		t.Fatal("clicking Show Commit Details did not restore the pane")
	}
}

func TestHistoryDefaultFilesWidthLeavesRoomForPatch(t *testing.T) {
	for _, tc := range []struct {
		width, files, minPatch float32
	}{{980, 105, 300}, {1320, 240, 370}, {1600, 240, 520}} {
		a := &app{page: "history"}
		a.sidebarSplit.rowWidth = tc.width
		a.historySplit.rowWidth = tc.width - a.workbenchSidebarWidth() - splitterWidth
		listWidth := (a.historySplit.rowWidth - splitterWidth) * a.historyShare()
		a.filesSplit.rowWidth = a.historySplit.rowWidth - splitterWidth - listWidth
		files := a.historyFilesWidth()
		patch := a.filesSplit.rowWidth - splitterWidth - files
		if math.Abs(float64(files-tc.files)) > 0.5 || patch < tc.minPatch-0.5 {
			t.Errorf("window %.0f: Files %.1f / patch %.1f, want %.0f / ≥%.0f", tc.width, files, patch, tc.files, tc.minPatch)
		}
	}
	// The working tree's row carries the commit box under its files, so its
	// Files column opens wider than a commit's.
	a := &app{page: "history", worktreeSelected: true}
	a.filesSplit.rowWidth = 800
	if got := a.historyFilesWidth(); math.Abs(float64(got-300)) > 0.5 {
		t.Errorf("working tree Files width = %.1f, want 300", got)
	}
}

func TestSavedHistorySplitLeavesMorePatchRoomInWiderWindows(t *testing.T) {
	a := &app{page: "history"}
	a.sidebarSplit.share, a.historySplit.share, a.filesSplit.share = .30, .40, .65
	var previousWindow, previousList float32
	for _, tc := range []struct{ window, minPatch float32 }{
		{980, 300}, {1000, 300}, {1100, 300}, {1160, 330}, {1320, 350}, {1600, 400}, {980, 300},
	} {
		a.sidebarSplit.rowWidth = tc.window
		a.historySplit.rowWidth = tc.window - a.workbenchSidebarWidth() - splitterWidth
		list := a.commitListWidth()
		a.filesSplit.rowWidth = a.historySplit.rowWidth - list - splitterWidth
		files := a.historyFilesWidth()
		patch := a.filesSplit.rowWidth - files - splitterWidth
		if tc.window > previousWindow && list < previousList-0.5 {
			t.Errorf("widening from %.0f to %.0f DIP reduced History from %.1f to %.1f", previousWindow, tc.window, previousList, list)
		}
		previousWindow, previousList = tc.window, list
		minFiles := float32(96)
		if tc.window >= 1160 {
			minFiles = 180
		}
		if list < 349.5 || files < minFiles-.5 || patch < tc.minPatch-.5 || tc.window == 980 && math.Abs(float64(patch-300)) > .5 {
			t.Errorf("%.0f DIP: commit list %.1f, Files %.1f, patch %.1f, want patch at least %.0f", tc.window, list, files, patch, tc.minPatch)
		}
	}
	if a.sidebarSplit.share != .30 || a.historySplit.share != .40 || a.filesSplit.share != .65 {
		t.Fatal("changing the window width replaced the saved split preferences")
	}
}

func TestHistoryResizeKeepsPatchWidthOnFirstFrame(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := &app{page: "history", locale: localeEnglish, themeMode: "light",
		repo: &gitrepo.Repository{Root: t.TempDir()}, worktreeSelected: true,
		status: gitrepo.Status{Entries: []gitrepo.Entry{{Path: "staged.txt", Index: 'M'}, {Path: "edited.txt", Worktree: 'M'}}}}
	a.sidebarSplit.share, a.historySplit.share, a.filesSplit.share = .30, .40, .65

	var frames []float32
	view := ui.NewTester(func(c *ui.Context) {
		a.root(c)
		// Files is the last fixed-width pane. The patch stretches across the
		// remainder, so a stale Files budget narrows it on this very frame.
		width, _ := c.Size()
		frames = append(frames, width-a.workbenchSidebarWidth()-a.commitListWidth()-a.historyFilesWidth()-3*splitterWidth)
	}, 980, 640)
	for _, tc := range []struct {
		window int
		patch  float64
	}{{1320, 366}, {1600, 420}, {980, 300}} {
		frames = frames[:0]
		view.SetSize(tc.window, 640)
		if len(frames) == 0 || math.Abs(float64(frames[0])-tc.patch) > 1 {
			t.Errorf("%d DIP: patch budgets across resize frames = %v, want %.0f from the first frame", tc.window, frames, tc.patch)
		}
		if tab, ok := view.AccessByRole(ui.RoleTab, a.detailSummaryLabel()); !ok || math.Abs(float64(float32(tc.window)-tab.Bounds.X)-tc.patch) > 10 {
			t.Errorf("%d DIP: patch tab starts at %.1f, want near %.0f", tc.window, tab.Bounds.X, float64(tc.window)-tc.patch)
		}
		filesLeft := a.workbenchSidebarWidth() + a.commitListWidth() + 2*splitterWidth
		for _, label := range []string{a.t("stageAll"), a.t("unstageAll"), a.t("discardAll")} {
			button, ok := view.AccessByRole(ui.RoleButton, label)
			if !ok || button.Offscreen || button.Disabled || button.Bounds.W < 23 ||
				button.Bounds.X < filesLeft-0.5 || button.Bounds.X+button.Bounds.W > filesLeft+a.historyFilesWidth()+0.5 {
				t.Errorf("%d DIP: %s must remain usable inside Files, got %+v", tc.window, label, button.Bounds)
			}
		}
	}
	if a.sidebarSplit.share != .30 || a.historySplit.share != .40 || a.filesSplit.share != .65 {
		t.Fatal("resizing replaced the saved split preferences")
	}
}

func TestShowingHistoryDetailsUsesWideEnoughRowOnFirstFrame(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := &app{page: "history", locale: localeEnglish, themeMode: "light", historyDetailsHidden: true,
		repo: &gitrepo.Repository{Root: t.TempDir()}, worktreeSelected: true}
	a.sidebarSplit.share = .95
	var firstPatchWidth float32
	view := ui.NewTester(func(c *ui.Context) {
		a.root(c)
		if firstPatchWidth == 0 && a.filesSplit.rowWidth > 0 {
			firstPatchWidth = a.filesSplit.rowWidth - a.historyFilesWidth() - splitterWidth
		}
	}, 980, 640)
	clickRepositoryTabsControl(t, view, ui.RoleButton, a.t("showCommitDetails"))
	if firstPatchWidth < 299.5 {
		t.Fatalf("newly shown patch was %.1f points wide, want at least 300", firstPatchWidth)
	}
	if a.sidebarSplit.share != .95 {
		t.Fatal("showing commit details replaced the saved sidebar preference")
	}
}

func TestWorkbenchWidthsSurviveRestartAndNarrowing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := &app{themeMode: "dark", locale: localeEnglish}
	a.sidebarSplit.share = 0.3
	a.filesSplit.share = 0.65
	a.historySplit.share = 0.4
	if !a.saveSettings() {
		t.Fatal("save failed")
	}
	path, err := settingsPath()
	if err != nil {
		t.Fatal(err)
	}
	b := &app{}
	b.applySettings(readSettings(path))
	if b.sidebarSplit.share != 0.3 || b.filesSplit.share != 0.65 || b.historySplit.share != 0.4 {
		t.Fatal("new widths did not survive restart")
	}
	b.sidebarSplit.rowWidth = 980
	_ = b.workbenchSidebarWidth()
	b.filesSplit.rowWidth = 487
	_ = b.historyFilesWidth()
	if !b.saveSettings() {
		t.Fatal("saving after resize failed")
	}
	c := &app{}
	c.applySettings(readSettings(path))
	if c.sidebarSplit.share != 0.3 || c.filesSplit.share != 0.65 {
		t.Fatal("narrowing permanently lost saved preferences")
	}
	c.filesSplit.rowWidth = 900
	if got := c.historyFilesWidth(); got < 579 || got > 582 {
		t.Fatalf("wide Files preference not restored: %v", got)
	}
	// Old settings need neither new field.
	oldPath := filepath.Join(t.TempDir(), "settings.json")
	if err := writeSettings(oldPath, persistedSettings{Theme: "dark"}); err != nil {
		t.Fatal(err)
	}
	d := &app{}
	d.applySettings(readSettings(oldPath))
	if d.sidebarSplit.share != 0 || d.filesSplit.share != 0 {
		t.Fatal("old settings gained explicit widths")
	}
}

func TestWorkbenchFilesDividerRespondsToPointerDrag(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := &app{page: "history", themeMode: "light"}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		width, _ := c.Size()
		a.historyDetailsPane(c, 1, width)
	}, 1000, 600)
	tt.Frame()
	before := a.historyFilesWidth()
	x := before + splitterWidth/2
	tt.Press(x, 300)
	tt.Move(x+80, 300)
	tt.Release(x+80, 300)
	if got := a.historyFilesWidth(); got < before+60 {
		t.Fatalf("pointer drag did not resize Files: before %v, after %v", before, got)
	}
	path, err := settingsPath()
	if err != nil {
		t.Fatal(err)
	}
	saved := readSettings(path)
	if saved.FilesSplit == nil {
		t.Fatal("completed drag did not persist Files width")
	}
	// A hidden Files column must remove its divider and give all room to Diff.
	a.detailsChangesHidden = true
	tt.Frame()
	a.detailsChangesHidden = false
	tt.Frame()
	if got := a.historyFilesWidth(); got < before+60 {
		t.Fatal("hide/show lost preferred Files width")
	}
}

func TestChangesDividerDragSurvivesRestartAndNarrowing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := &app{page: "history", worktreeSelected: true, themeMode: "light", locale: localeEnglish, commitText: "retained draft"}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		width, _ := c.Size()
		a.historyDetailsPane(c, 1, width)
	}, 1000, 600)
	tt.Frame()
	before := a.historyFilesWidth()
	x := before + splitterWidth/2
	tt.Press(x, 150)
	tt.Move(x+80, 150)
	tt.Release(x+80, 150)
	preferred := a.historyFilesWidth()
	if preferred < before+60 {
		t.Fatalf("Changes divider did not resize: %v → %v", before, preferred)
	}
	path, err := settingsPath()
	if err != nil {
		t.Fatal(err)
	}
	b := &app{page: "history", worktreeSelected: true}
	b.applySettings(readSettings(path))
	b.filesSplit.rowWidth = 1000
	if math.Abs(float64(b.historyFilesWidth()-preferred)) > 1 {
		t.Fatal("Changes Files preference did not survive restart")
	}
	share := b.filesSplit.share
	// At this width the Files column gives width back so the patch keeps its
	// own minimum; both floors hold even with a split saved wide.
	b.sidebarSplit.rowWidth = minWindowWidth
	filesMin, patchMin := b.historyFilesMinWidth(), b.historyPatchMinWidth()
	b.filesSplit.rowWidth = filesMin + splitterWidth + patchMin
	files := b.historyFilesWidth()
	patch := b.filesSplit.rowWidth - splitterWidth - files
	if files < filesMin-0.01 || patch < patchMin-0.01 {
		t.Fatalf("narrow Changes lost minimum panes: %v/%v", files, patch)
	}
	b.sidebarSplit.rowWidth = 0
	b.filesSplit.rowWidth = 1000
	if b.filesSplit.share != share || math.Abs(float64(b.historyFilesWidth()-preferred)) > 1 || a.commitText != "retained draft" {
		t.Fatal("narrowing lost preferred width or draft")
	}
}

func TestWorkbenchDividerResetKeepsOtherPreferences(t *testing.T) {
	a := &app{themeMode: "dark", locale: localeEnglish, commitText: "retained draft"}
	a.sidebarSplit.share = 0.3
	a.filesSplit.share = 0.6
	a.historySplit.share = 0.4
	a.resetLayout()
	if a.sidebarSplit.share != 0 || a.filesSplit.share != 0 || a.historySplit.share != 0 {
		t.Fatal("layout reset left new widths behind")
	}
	if a.themeMode != "dark" || a.locale != localeEnglish || a.commitText != "retained draft" {
		t.Fatal("layout reset changed unrelated preferences or draft")
	}
}
