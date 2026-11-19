package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestQAMinimumWindowCloseRevealsSuccessorAndPreservesDraft(t *testing.T) {
	a := &app{page: "history", worktreeSelected: true, locale: localeEnglish, repo: &gitrepo.Repository{Root: t.TempDir()}}
	first := detailFileKey{Path: "first-extremely-long-file-name-long-enough-to-fill-tab.go"}
	closing := detailFileKey{Path: "b.go"}
	next := detailFileKey{Path: "successor-extremely-long-file-name-long-enough-to-fill-tab.go"}
	last := detailFileKey{Path: "last-extremely-long-file-name-long-enough-to-fill-tab.go"}
	for _, file := range []detailFileKey{first, closing, next, last} {
		a.detailTabs().Open(file)
	}
	// The commit box is the Summary tab's content now, so the draft is typed
	// there before a file tab takes the pane.
	a.detailTabs().Activate(detailTabKey{Summary: true})
	a.selectedPath = closing.Path
	a.diffLoading = true // No asynchronous fixture read is needed to test strip layout.
	// Prime the neighbor's diff so the normal pointer close can activate it.
	a.diffCache.put(a.repo.Root, a.loadSeq, next, gitrepo.Diff{Text: "cached fixture patch"})
	view := ui.NewTester(a.root, minWindowWidth, minWindowHeight)
	view.SetTitleBar(ui.TitleBar{Height: 38, Left: 72})
	for range 5 {
		view.Frame()
	}
	focusCommitField(t, view, a.t("commitSummary"))
	view.Type("keep my summary")
	focusCommitField(t, view, a.t("commitDescription"))
	view.Type("keep my description")
	if a.commitText != "keep my summary\n\nkeep my description" {
		t.Fatalf("draft setup failed: %q", a.commitText)
	}
	a.activateDetailTab(detailTabKey{File: closing})
	for range 5 {
		view.Frame()
	}
	xLabel := func(file detailFileKey) string { return a.t("close") + " " + file.Path }
	if node, _ := view.AccessByRole(ui.RoleButton, xLabel(closing)); node.Offscreen || node.Bounds.W == 0 {
		t.Fatalf("close action setup failed: %+v", node)
	}
	clickRepositoryTabsControl(t, view, ui.RoleButton, xLabel(closing))
	for range 5 {
		view.Frame()
	}
	if a.detailTabs().Active.File != next || a.commitText != "keep my summary\n\nkeep my description" {
		t.Fatalf("active=%+v; draft=%q", a.detailTabs().Active, a.commitText)
	}
	strip, _ := view.AccessByRole(ui.RoleTabList, a.t("detailTabs"))
	node, ok := view.AccessByRole(ui.RoleButton, xLabel(next))
	if !ok || node.Offscreen || node.Bounds.X+node.Bounds.W > minWindowWidth+0.5 {
		t.Fatalf("closing an active tab hid its selected successor's Close action: x=%v max=%v strip=%+v node=%+v", a.detailTabStrip.X, a.detailTabStrip.MaxX, strip.Bounds, node)
	}
}
