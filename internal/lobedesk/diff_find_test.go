package lobedesk

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

// Exercise the actual history field, patch focus, text editor and both virtual lists.
func TestDiffFindInFocusedPatch(t *testing.T) {
	for _, worktree := range []bool{false, true} {
		for _, summary := range []bool{false, true} {
			for _, split := range []bool{false, true} {
				t.Run(fmt.Sprintf("worktree=%v/summary=%v/split=%v", worktree, summary, split), func(t *testing.T) {
					a := worktreePageApp()
					a.repo = &gitrepo.Repository{Root: "/review"}
					a.locale, a.worktreeSelected, a.splitDiff = localeEnglish, worktree, split
					a.selectedSHA, a.selectedPath = "aaaa", "a.go"
					file := detailFileKey{Path: "a.go"}
					if !summary {
						a.detailTabs().Open(file)
					}
					patch := gitrepo.Diff{Text: "fixture", Lines: []gitrepo.DiffLine{
						{Kind: "context", Old: 1, New: 1, Text: " retryDelay retryDELAY"},
						{Kind: "del", Old: 2, Text: "-retryDelay"},
					}}
					for i := range 320 {
						patch.Lines = append(patch.Lines, gitrepo.DiffLine{Kind: "add", New: i + 2, Text: fmt.Sprintf("+line %d", i)})
					}
					patch.Lines = append(patch.Lines, gitrepo.DiffLine{Kind: "add", New: 322, Text: "+last " + strings.Repeat("x", 3800) + " retryDelay"})
					a.diff = patch
					a.overviewRoot = a.repo.Root
					a.overviewEntries = []worktreeOverviewEntry{{Key: file, Diff: patch}}
					a.commitDiff, a.commitSections = buildCommitDiff([]gitrepo.FileDiff{{Path: file.Path, Diff: patch}})
					view := pageTester(a, func(c *ui.Context) {
						a.registerShortcuts(c)
						a.historyPage(c)
					})
					first, ok := view.Find(" retryDelay retryDELAY")
					if !ok {
						t.Fatal("first line missing")
					}
					view.ClickAt(first.X+4, first.Y+first.H/2)
					view.Key(ui.Cmd, ui.KeyF)
					view.Type("retrydelay")
					if !view.HasText("1/4") || a.historyQuery != "" {
						t.Fatalf("focused patch did not find four occurrences; history query=%q", a.historyQuery)
					}
					view.Key(0, ui.KeyEnter)
					if !view.HasText("2/4") {
						t.Fatal("Enter did not advance within the first line")
					}
					view.Key(ui.Shift, ui.KeyEnter)
					view.Key(ui.Shift, ui.KeyEnter)
					if !view.HasText("4/4") {
						t.Fatal("reverse did not wrap")
					}
					for range 3 {
						view.Frame()
					}
					r, ok := view.AccessByRole(ui.RoleInsertion, patch.Lines[len(patch.Lines)-1].Text)
					viewport := a.commitDiffViewport
					if worktree && !summary {
						viewport = a.diffViewportRect
					}
					if !ok || r.Bounds.Y < viewport.Y-1 || r.Bounds.Y+r.Bounds.H > viewport.Y+viewport.H+1 {
						t.Fatalf("distant result is outside viewport %+v: %+v", viewport, r)
					}
					// Only the active occurrence paints with the solid amber ink.
					// Inline text is part of one accessible paragraph, so check the
					// rendered pixels to catch both clipping and missing highlights.
					pixels := view.Image()
					left, right := int(viewport.X+viewport.W), -1
					for y := int(r.Bounds.Y); y < int(r.Bounds.Y+r.Bounds.H); y++ {
						for x := int(viewport.X); x < int(viewport.X+viewport.W); x++ {
							color := pixels.RGBAAt(x, y)
							if color.R == 255 && color.G == 224 && color.B == 138 {
								left = min(left, x)
								right = max(right, x)
							}
						}
					}
					if right-left < 44 {
						t.Errorf("last match not visibly highlighted on long line: x=%d..%d, row=%+v viewport=%+v", left, right, r.Bounds, viewport)
					}
					view.Key(0, ui.KeyEscape)
					if view.HasText("4/4") || a.selectedPath != "a.go" || a.selectedSHA != "aaaa" || a.detailTabs().Active.Summary != summary {
						t.Fatal("closing Find changed the reading target")
					}
					view.Key(ui.Cmd, ui.KeyF)
					view.Type("absent")
					if !view.HasText("0/0") || a.historyQuery != "" {
						t.Fatal("Escape did not return focus to patch")
					}
				})
			}
		}
	}
}

func TestDiffFindLeavesHistorySearchAndCommitEditorScoped(t *testing.T) {
	a := worktreePageApp()
	a.locale, a.worktreeSelected = localeEnglish, true
	view := pageTester(a, a.historyPage)
	view.Key(ui.Cmd, ui.KeyF)
	view.Type("first")
	if a.historyQuery != "first" {
		t.Fatal("history search lost its existing shortcut")
	}
	editor, ok := view.AccessByRole(ui.RoleTextField, a.t("commitMessage"))
	if !ok {
		t.Fatal("missing commit editor")
	}
	view.ClickAt(editor.Bounds.X+5, editor.Bounds.Y+5)
	view.Key(ui.Cmd, ui.KeyF)
	view.Type("draft")
	if a.historyQuery != "first" || a.commitText != "draft" {
		t.Fatalf("Find took over the commit editor: query=%q, draft=%q", a.historyQuery, a.commitText)
	}
}

func TestDiffFindTracksDisplayedOverviewAcrossReadsAndFileTabs(t *testing.T) {
	a := worktreePageApp()
	a.worktreeSelected, a.locale = true, localeChinese
	a.repo = &gitrepo.Repository{Root: "/review"}
	a.overviewRoot = a.repo.Root
	a.selectedPath = "same.go"
	selected := detailFileKey{Path: a.selectedPath}
	other := detailFileKey{Path: a.selectedPath, Staged: true}
	makePatch := func(text string) gitrepo.Diff {
		return gitrepo.Diff{Text: text, Lines: []gitrepo.DiffLine{{Kind: "add", New: 1, Text: text}}}
	}
	a.overviewEntries = []worktreeOverviewEntry{{Key: selected, Diff: makePatch("+开头 [重试]")}}
	for i := range 100 {
		a.overviewEntries = append(a.overviewEntries, worktreeOverviewEntry{Key: detailFileKey{Path: fmt.Sprintf("p/%03d", i)}, Diff: makePatch("+other")})
	}
	a.overviewEntries = append(a.overviewEntries, worktreeOverviewEntry{Key: other, Loading: true})
	view := pageTester(a, a.historyPage)
	r, ok := view.Find("+开头 [重试]")
	if !ok {
		t.Fatal("missing overview")
	}
	view.ClickAt(r.X+3, r.Y+r.H/2)
	view.Key(ui.Cmd, ui.KeyF)
	view.Type("[重试]")
	if !view.HasText("1/1") {
		t.Fatal("query did not treat bracketed Chinese text literally")
	}
	a.overviewEntries[len(a.overviewEntries)-1] = worktreeOverviewEntry{Key: other, Diff: makePatch("+结尾 [重试]")}
	view.Frame()
	if !view.HasText("1/2") {
		t.Fatal("newly loaded patch failed to join the overview search")
	}
	clickRepositoryTabsControl(t, view, ui.RoleButton, a.t("nextDiffMatch"))
	last, ok := view.AccessByRole(ui.RoleInsertion, "+结尾 [重试]")
	if !view.HasText("2/2") || !ok || last.Offscreen || last.Bounds.Y < a.commitDiffViewport.Y || last.Bounds.Y+last.Bounds.H > a.commitDiffViewport.Y+a.commitDiffViewport.H+1 || a.selectedPath != "same.go" || a.selectedStaged {
		t.Fatalf("next overview target missing: result=%+v, scroll=%v/%v, active=%d/%d, reveal=%v, field=%v, focus=%v, textsTail=%v", last.Bounds, a.commitInfoScroll.Y, a.commitInfoScroll.MaxY, a.diffFind.active, len(a.diffFind.matches), a.diffFind.reveal, a.diffFind.matches[a.diffFind.active], view.Focused(a.t("findInDiff")), view.Texts()[max(0, len(view.Texts())-10):])
	}
	clickRepositoryTabsControl(t, view, ui.RoleButton, a.t("previousDiffMatch"))
	if !view.HasText("1/2") {
		t.Fatal("previous control did not navigate")
	}
	a.detailContexts.Worktree.Collapsed = map[detailFileKey]bool{other: true}
	view.Frame()
	if !view.HasText("1/1") {
		t.Fatal("collapsed patch still counted")
	}
	a.overviewEntries[0].Diff = makePatch("+替换内容")
	view.Frame()
	if !view.HasText("0/0") {
		t.Fatal("old content from a replaced patch still counted")
	}
	a.detailTabs().Open(selected)
	view.Frame()
	if _, ok := view.AccessByRole(ui.RoleTextField, a.t("findInDiff")); ok {
		t.Fatal("opening a new tab retained old search")
	}
}
