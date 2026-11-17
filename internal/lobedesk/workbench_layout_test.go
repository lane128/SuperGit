package lobedesk

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
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
			if sidebar < 159.99 || history < 259.99 || files < 179.99 || patch < 299.99 {
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
		a.historyDetailsPane(c)
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
	a := &app{page: "changes", worktreeSelected: true, themeMode: "light", locale: localeEnglish, commitText: "retained draft"}
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.changesPage(c) }, 1000, 600)
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
	b := &app{page: "changes"}
	b.applySettings(readSettings(path))
	b.filesSplit.rowWidth = 1000
	if math.Abs(float64(b.historyFilesWidth()-preferred)) > 1 {
		t.Fatal("Changes Files preference did not survive restart")
	}
	share := b.filesSplit.share
	b.filesSplit.rowWidth = changesFilesPaneMin + splitterWidth + historyDiffPaneMin
	files := b.historyFilesWidth()
	patch := b.filesSplit.rowWidth - splitterWidth - files
	if files < changesFilesPaneMin-0.01 || patch < historyDiffPaneMin-0.01 {
		t.Fatalf("narrow Changes lost minimum panes: %v/%v", files, patch)
	}
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
