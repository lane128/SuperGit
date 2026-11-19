package lobedesk

import (
	"fmt"
	"image"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

func TestHistoryDetailTabOverflowStaysVisibleAndDraggable(t *testing.T) {
	const name = "configuration-file-with-a-long-name-for-two-worktree-versions.go"
	unstaged := detailFileKey{Path: name}
	staged := detailFileKey{Path: name, Staged: true}
	a := &app{
		page: "history", locale: localeEnglish, worktreeSelected: true,
		repo: &gitrepo.Repository{Root: t.TempDir()}, selectedPath: name, selectedStaged: true,
		status:       gitrepo.Status{Entries: []gitrepo.Entry{{Path: name, Index: 'M', Worktree: 'M'}}},
		detailReveal: true, diff: gitrepo.Diff{Text: "staged patch"},
	}
	a.detailTabs().Open(unstaged)
	a.detailTabs().Open(staged)
	a.diffCache.put(a.repo.Root, a.loadSeq, unstaged, gitrepo.Diff{Text: "unstaged patch"})
	paneWidth := float32(300)
	colors := *themeConfig("").Theme(false)
	colors.Scrollbar = ui.RGB(0, 160, 80)
	view := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		c.SetTheme(&colors)
		ui.Row(c).Width(paneWidth).Height(300).Children(func() { a.historyDiffPane(c) })
	}, 720, 300)
	view.Move(20, 200)
	view.Frame()
	view.Advance(time.Second)
	view.Frame()
	if a.detailTabStrip.MaxX <= 0 || a.detailTabStrip.X <= 0 {
		t.Fatalf("expected overflow and an active tab revealed on the right, got %+v", a.detailTabStrip)
	}
	// Find the painted thumb with the pointer away from any of the tabs.
	thumbX, thumbY := -1, -1
	paint := view.Image()
	for y := 28; y < 38 && thumbX < 0; y++ {
		for x := 8; x < int(paneWidth)-8; x++ {
			p := paint.RGBAAt(x, y)
			if p.R == 0 && p.G == 160 && p.B == 80 {
				thumbX, thumbY = x, y
				break
			}
		}
	}
	if thumbX < 0 {
		t.Fatal("overflow thumb is hidden at rest")
	}
	view.Press(float32(thumbX), float32(thumbY))
	view.Move(2, float32(thumbY))
	view.Release(2, float32(thumbY))
	view.Frame()
	if a.detailTabStrip.X > 0.5 || a.detailTabs().Active.File != staged || a.diff.Text != "staged patch" {
		t.Fatalf("drag should reveal earlier tabs without changing the patch: x=%v active=%+v patch=%q", a.detailTabStrip.X, a.detailTabs().Active, a.diff.Text)
	}
	clickRepositoryTabsControl(t, view, ui.RoleTab, name+" · Unstaged")
	if a.detailTabs().Active.File != unstaged || a.diff.Text != "unstaged patch" {
		t.Fatalf("click did not switch to the revealed version: active=%+v patch=%q", a.detailTabs().Active, a.diff.Text)
	}
	paneWidth = 720
	view.Move(20, 200)
	view.Frame()
	view.Advance(time.Second)
	view.Frame()
	if a.detailTabStrip.MaxX != 0 {
		t.Fatalf("all tabs should fit in the wider pane: %+v", a.detailTabStrip)
	}
	paint = view.Image()
	for y := 28; y < 38; y++ {
		for x := range int(paneWidth) {
			p := paint.RGBAAt(x, y)
			if p.R == 0 && p.G == 160 && p.B == 80 {
				t.Fatalf("an overflow thumb is still painted at %d,%d on a wide pane", x, y)
			}
		}
	}
}

func TestHistoryDetailTabsKeepStageVisibleDuringPaneDrag(t *testing.T) {
	const name = "dual-version-verification-rollout-configuration-validation-0123456789.go"
	const path = "configs/" + name
	for _, width := range []int{980, 1320, 1600} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			var captions [2]*image.RGBA
			var origins [2]image.Point
			for i, stage := range []string{"Staged", "Unstaged"} {
				staged := i == 0
				line := "+" + stage + " version content"
				a := &app{
					page: "history", locale: localeEnglish, themeMode: "light", showGraph: true,
					worktreeSelected: true, repo: &gitrepo.Repository{Root: t.TempDir()}, commits: manyLaneHistory(14),
					commitFilesLoaded: true, status: gitrepo.Status{Entries: []gitrepo.Entry{{Path: path, Index: 'M', Worktree: 'M'}}},
					selectedPath: path, selectedStaged: staged, detailReveal: true,
					historySplit: splitDrag{share: 0.34},
					diff:         gitrepo.Diff{Text: line, Lines: []gitrepo.DiffLine{{Kind: "add", Text: line}}},
				}
				// Compare the selected versions at the same position and weight.
				a.detailTabs().Open(detailFileKey{Path: path, Staged: !staged})
				a.detailTabs().Open(detailFileKey{Path: path, Staged: staged})
				view := ui.NewTester(a.root, width, 640)
				view.Frame()
				view.Frame()
				listBefore := a.commitListWidth()
				x := a.workbenchSidebarWidth() + splitterWidth + listBefore + splitterWidth/2
				view.Press(x, 300)
				view.Move(x+400, 300)
				// Keep the pointer held so this checks the live layout without
				// persisting a splitter preference when the pointer is released.
				view.Frame()
				view.Frame()
				if a.commitListWidth() <= listBefore || a.diffViewportRect.W < historyDiffPaneMin-0.5 {
					t.Fatalf("unexpected pane widths during drag: History %.1f → %.1f, patch %.1f", listBefore, a.commitListWidth(), a.diffViewportRect.W)
				}
				tab, ok := view.AccessByRole(ui.RoleTab, name+" · "+stage)
				if !ok || tab.Offscreen || !tab.Selected || tab.Bounds.X+tab.Bounds.W > float32(width) {
					t.Fatalf("selected %s tab is outside the workbench: %+v", stage, tab)
				}
				captions[i] = view.Image()
				origins[i] = image.Pt(int(math.Round(float64(tab.Bounds.X))), int(tab.Bounds.Y))
				if !strings.Contains(tab.Description, path) || !strings.Contains(tab.Description, stage) {
					t.Errorf("tooltip should identify %s and its full path, got %q", stage, tab.Description)
				}
			}
			ink, differences := 0, 0
			for y := 8; y < 30; y++ {
				for x := 10; x < 214; x++ {
					first := captions[0].RGBAAt(origins[0].X+x, origins[0].Y+y)
					second := captions[1].RGBAAt(origins[1].X+x, origins[1].Y+y)
					if first.R < 100 {
						ink++
					}
					if first != second {
						differences++
					}
				}
			}
			if ink < 50 || differences == 0 {
				t.Fatalf("selected version captions must be visibly different: %d ink pixels, %d differences", ink, differences)
			}
		})
	}
}
