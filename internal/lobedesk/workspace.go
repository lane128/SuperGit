package lobedesk

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	unicodewidth "golang.org/x/text/width"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/choice"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

// The working tree and the repository's long-lived panes share this file:
// the file list and patch that the history page draws for the working-tree
// row, the commit form, and the cards for an operation in progress. The
// separate workspace page is gone; its first row lives on in history.

func (a *app) branchSelector(c *ui.Context, iconWidth float32) {
	tk := theme.TokensOf(c)
	branch := a.status.Branch
	if branch == "" {
		branch = a.t("detached")
	}
	caption := branch
	items, spoken := a.branchTrackingStatus(tk)
	fontSize := scaledFont(12)
	const padding, groupGap, iconGap = 10, 8, 2
	available := toolbarCommandWidth - iconWidth - 2*(padding+1)
	if spoken != "" {
		caption += " (" + spoken + ")"
	}
	// Counts follow a hairline divider so they belong to the branch instead
	// of floating at the end of the field; numbers use the code font.
	counts := len(items) > 0 && items[0].icon != nil
	if counts {
		available -= 1 + groupGap
	}
	for _, item := range items {
		font := ""
		if item.icon != nil {
			font = a.patchFontFamily()
		}
		width, _ := c.MeasureText(0, ui.Span{Text: item.text, Font: font, Size: fontSize, Weight: 500})
		available -= width + groupGap
		if item.icon != nil {
			available -= fontSize + iconGap
		}
	}
	name := fitToolbarBranchName(c, branch, fontSize, available)
	disabled := !a.canOpenQuickCheckout() || a.repositoryOpenPending() || a.historyRefPalette.active
	progress := a.progressText()
	tooltip := caption + "\n" + a.t("quickCheckout") + " (" + shortcutKeys("branch.quickCheckout") + ")"
	if progress != "" {
		status := progress
		if a.opProgress.Active && !a.opProgress.Indeterminate && a.opProgress.Fraction > 0 {
			status = fmt.Sprintf("%s %d%%", progress, int(a.opProgress.Fraction*100+0.5))
		}
		tooltip = status + "\n" + tooltip
	}
	ui.Box(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
		button := styleWorkbenchButton(c, ui.ButtonBase(c), buttonNormal, disabled, false).
			Label(a.t("branches")).Description(caption).
			Tooltip(tooltip).
			FillWidth().MinWidth(0).PaddingX(padding).Gap(groupGap).Radius(0, 6, 6, 0)
		button.Children(func() {
			label := ui.Text(c.Key("branch-name"), name).FontSize(fontSize).SingleLine().MinWidth(0).Grow(1).Basis(0)
			if a.status.Branch == "" && !disabled {
				label.TextColor(tk.Warning)
			}
			bg, _ := workbenchButtonColors(tk, buttonNormal, button.Hovered(), button.Pressed(), button.IsDisabled(), false)
			if counts {
				ui.Box(c).Width(1).Height(fontSize + 2).Shrink(0).Background(tk.Border).AccessHidden(true)
			}
			for _, item := range items {
				ui.Row(c).Shrink(0).AlignItems(ui.Center).Gap(iconGap).TextColor(capsuleInk(item.ink, bg, 4.5)).AccessHidden(true).Children(func() {
					if item.icon == nil {
						ui.Text(c, item.text).FontSize(fontSize).SingleLine()
						return
					}
					ui.Icon(c, item.icon).Size(fontSize, fontSize)
					ui.Text(c, item.text).Font(a.patchFontFamily()).FontSize(fontSize).SingleLine()
				})
			}
		})
		if button.Clicked() {
			button.Focus()
			a.openQuickCheckout()
		}
		if progress != "" {
			a.branchProgressBar(c, tk, progress)
		}
	})
}

const branchProgressHeight float32 = 2

var branchProgressAnimation = ui.Animation{Duration: 180 * time.Millisecond, Ease: ui.EaseOut}

func (a *app) branchProgressBar(c *ui.Context, tk *theme.Tokens, label string) {
	bar := ui.Box(c.Key("branch-progress")).Absolute().Left(1).Right(1).Bottom(1).Height(branchProgressHeight).
		Radius(0, 0, 5, 0).Clip().Role(ui.RoleProgress).Label(label).Background(tk.FillSecondary)
	if a.opProgress.Active && !a.opProgress.Indeterminate && a.opProgress.Fraction > 0 {
		fraction := min(float32(1), max(float32(0), a.opProgress.Fraction))
		bar.Range(0, 1, float64(fraction))
		shown := bar.AnimateTo("progress", fraction, branchProgressAnimation)
		bar.DrawOver(func(p *ui.Painter, r ui.Rect) {
			if shown <= 0 {
				return
			}
			r.W *= shown
			p.Fill(r, tk.Blue, 0)
		})
		return
	}
	phase := bar.Loop("indeterminate", 1250*time.Millisecond, ui.Linear)
	bar.DrawOver(func(p *ui.Painter, r ui.Rect) {
		segW := r.W * 0.34
		x := r.X - segW + (r.W+segW)*phase
		left := max(r.X, x)
		right := min(r.X+r.W, x+segW)
		if right > left {
			p.Fill(ui.Rect{X: left, Y: r.Y, W: right - left, H: r.H}, tk.Blue, 1)
		}
	})
}

// Fit the visible name to the space left after reserving the complete tracking
// status. The same prefix and suffix strategy is used by the other ref labels.
func fitToolbarBranchName(c *ui.Context, name string, size, available float32) string {
	width, _ := c.MeasureText(0, ui.Span{Text: name, Size: size, Weight: 500})
	if width <= available {
		return name
	}
	fit := elidedPrefix("")
	for low, high := 5, utf8.RuneCountInString(name)-1; low <= high; {
		mid := low + (high-low)/2
		candidate := elideMiddle(name, mid)
		width, _ := c.MeasureText(0, ui.Span{Text: candidate, Size: size, Weight: 500})
		if width <= available {
			fit, low = candidate, mid+1
		} else {
			high = mid - 1
		}
	}
	return fit
}

// All workspace-wide file actions live in the Files header. Each uses the same
// target checks as its keyboard and menu commands.
func (a *app) filesToolbar(c *ui.Context) {
	for _, item := range []struct {
		kind, label, scope string
		icon               *ui.SVG
		run                func()
	}{
		{"stage", "stageAll", "stageAllScope", lucide.Plus(), a.stageAll},
		{"unstage", "unstageAll", "unstageAllScope", lucide.Minus(), a.unstageAll},
		{"discard", "discardAll", "discardAllScope", lucide.Undo2(), a.confirmDiscardAll},
	} {
		_, reason := a.batchFileTargets(item.kind)
		hint := a.t(item.scope)
		if reason != "" {
			hint += "\n" + reason
		}
		role := buttonNormal
		if item.kind == "discard" {
			role = buttonDangerIcon
		}
		button := compactActionIcon(c.Key(item.kind), item.icon, a.t(item.label), role, reason != "").Description(hint).Tooltip(a.t(item.label) + "\n" + hint)
		if button.Clicked() {
			item.run()
		}
	}
}

// The same three sections remain visible even when all counts are zero.
func (a *app) workingFileSections(c *ui.Context) {
	for _, group := range a.workingFileGroups() {
		a.fileSection(c.Key(group.Key), group.Key, group.Title, group.Entries, group.Staged)
	}
	if !a.status.HasWorkspaceChanges() {
		a.cleanFiles(c)
	}
}

func (a *app) cleanFiles(c *ui.Context) {
	listEmptyState(c, lucide.CircleCheck(), theme.TokensOf(c).Success, a.t("clean"), a.t("cleanHint"))
}

func fileSectionAccent(tk *theme.Tokens, key string, staged bool) ui.Color {
	switch {
	case staged || key == filesSectionStaged:
		return tk.Success
	case key == filesSectionUntracked:
		return tk.Blue
	default:
		return tk.Warning
	}
}

func fileSectionHeaderColors(tk *theme.Tokens, hovered bool) (bg, divider, badgeBg ui.Color) {
	return foldBandColors(tk, hovered)
}

// fileSection draws one group of files under a header that folds it. The count
// stays on a folded header, so the reader can still tell what is put away.
func (a *app) fileSection(c *ui.Context, key, title string, entries []gitrepo.Entry, staged bool) {
	tk := theme.TokensOf(c)
	collapsed := a.fileSectionCollapsed(key)
	selectionCount := 0
	if s := a.workspaceSelection; a.repo != nil && s.Root == a.repo.Root && s.Group == key {
		selectionCount = len(s.Files)
	}
	description := a.t("fileSectionHint")
	if selectionCount > 0 {
		description += "\n" + a.tf("fileGroupCount", len(entries)) + " · " + a.tf("filesSelectedCount", selectionCount)
	}
	compact := a.page == "history" && a.historyFilesWidth() < scaledFont(130)
	topBorder := float32(1)
	if key == filesSectionUnstaged {
		topBorder = 0
	}
	bottomBorder := float32(0)
	if !collapsed || key == filesSectionUntracked {
		bottomBorder = 1
	}
	ui.Column(c).FillWidth().Gap(0).Children(func() {
		header := ui.ButtonBase(c).Label(title).Row().FillWidth().Height(scaledFont(29)).Padding(0, 10, 0, 0).Gap(0).
			Radius(0).AlignItems(ui.Center).Expanded(!collapsed).Cursor(ui.CursorPointer).
			// The header folds the section and its count keeps counting the
			// folded rows; say both, or the number reads as what is on screen.
			Description(description).Tooltip(description)
		if compact {
			header.Padding(0, 4, 0, 0).Tooltip(title + "\n" + description)
		}
		bg, divider, badgeBg := fileSectionHeaderColors(tk, header.Hovered())
		header.Background(bg).BorderWidth(topBorder, 0, bottomBorder, 0).BorderColor(divider)
		header.OnClick(func() { a.toggleFilesSection(key) })
		header.Children(func() {
			accent := fileSectionAccent(tk, key, staged).Mix(bg, 0.22)
			ui.Box(c).Width(3).AlignSelf(ui.Stretch).Background(accent).Shrink(0)
			inner := ui.Row(c).Grow(1).Basis(0).MinWidth(0).Padding(0, 0, 0, 8).Gap(6).AlignItems(ui.Center)
			if compact {
				inner.Padding(0, 0, 0, 4).Gap(4)
			}
			inner.Children(func() {
				chevron := lucide.ChevronDown()
				if collapsed {
					chevron = lucide.ChevronRight()
				}
				ui.Icon(c, chevron).FontSize(scaledFont(12)).TextColor(tk.TextSecondary).Shrink(0)
				ui.Text(c, title).FontSize(scaledFont(11.5)).FontWeight(600).TextColor(tk.Text).SingleLine().Grow(1).Basis(0).MinWidth(0)
				if s := a.workspaceSelection; a.repo != nil && s.Root == a.repo.Root && s.Group == key && len(s.Files) > 0 {
					display.Text(c, a.tf("filesSelectedCount", len(s.Files)), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(10.5)}).SingleLine().Shrink(0)
				}
				ui.Row(c).Height(scaledFont(17)).MinWidth(scaledFont(20)).PaddingX(6).Radius(999).
					Center().Shrink(0).Background(badgeBg).Border(1, divider).Children(func() {
					ui.Text(c, fmt.Sprint(len(entries))).FontSize(scaledFont(10.5)).FontWeight(600).TextColor(tk.TextSecondary).SingleLine()
				})
			})
		})
		if collapsed {
			return
		}
		// The rows are the section's options, with their place and the
		// section's name in the description: the same path can appear in
		// two sections as "unstaged" and "staged", and a screen reader
		// otherwise hears two identical buttons.
		heights := make([]float32, len(entries))
		reveal, remembered := -1, -1
		context := a.currentDetailContext()
		for i, entry := range entries {
			heights[i] = fileRowHeight() + fileRowGap
			file := detailFileKey{Path: entry.Path, Staged: staged}
			if a.detailFileRemembered != nil && a.detailFileRememberedContext == context && *a.detailFileRemembered == file {
				remembered = i
			}
			if (a.detailFileFocus != nil && a.detailFileFocusContext == context && *a.detailFileFocus == file) ||
				(a.revealSelection && a.selectedPath == entry.Path && a.selectedStaged == staged) {
				reveal = i
			}
		}
		listCol := ui.Column(c).FillWidth().Role(ui.RoleList).Label(title).Multiselectable(true)
		if len(entries) > 0 {
			if compact {
				listCol.Padding(3, 3, 4, 3)
			} else {
				listCol.Padding(4, 6, 6, 6)
			}
		}
		listCol.Children(func() {
			virtualPatchRows(c, a.commitFilesViewport, &a.commitFilesScroll, heights, reveal, func(rowContext *ui.Context, i int) {
				ui.Column(rowContext).FillWidth().Height(heights[i]).Shrink(0).Children(func() {
					row := a.fileRow(rowContext, entries[i], staged)
					// Keep full group positions and the Index/worktree identity.
					// The badge names the change type; the tooltip stays the path.
					row.PositionInSet(i+1, len(entries)).Description(title).Tooltip(entries[i].Path)
					wasFocused := ui.Local(row, "file-was-focused", func() bool { return false })
					if row.Focused() {
						file := detailFileKey{Path: entries[i].Path, Staged: staged}
						a.detailFileRemembered = &file
						a.detailFileRememberedContext = context
						if !*wasFocused {
							row.ScrollIntoView()
						}
					}
					*wasFocused = row.Focused()
				})
			}, remembered)
		})
	})
}

// filesFoldSel is what folding one section did to the file selection: the row
// it moved from and the row it moved to. The memory only counts while the
// selection is still the row the fold chose, so a reader who picks another
// file keeps it when the section opens again.
type filesFoldSel struct {
	Was     fileRef
	MovedTo fileRef
}

// toggleFilesSection folds or unfolds a section. Folding the section that holds
// the selection moves the selection to the first row still on screen, the way
// the arrows would: a selected row that is not drawn would leave the details
// pane reading a file with no way to point at it. Unfolding that section puts
// the selection back, so folding is a peek rather than a way to lose the row.
func (a *app) toggleFilesSection(key string) {
	if a.filesCollapsed == nil {
		a.filesCollapsed = map[string]bool{}
	}
	if a.filesCollapsed[key] {
		a.expandFilesSection(key)
	} else {
		a.foldFilesSection(key)
	}
	// Folding outlives the session: the next start opens the workspace the
	// way the reader left it, and another repository keeps its own folds.
	a.rememberFilesFolds()
}

func (a *app) foldFilesSection(key string) {
	if a.workspaceSelection.Group == key {
		a.workspaceSelection = workspaceFileSelection{}
	}
	a.filesCollapsed[key] = true
	refs := a.fileRefs()
	if len(refs) == 0 {
		return
	}
	for _, ref := range refs {
		if ref.path == a.selectedPath && ref.staged == a.selectedStaged {
			return
		}
	}
	if a.selectedPath != "" {
		if a.filesFoldSel == nil {
			a.filesFoldSel = map[string]filesFoldSel{}
		}
		a.filesFoldSel[key] = filesFoldSel{
			Was:     fileRef{path: a.selectedPath, staged: a.selectedStaged},
			MovedTo: refs[0],
		}
	}
	a.revealSelection = true
	a.selectFile(refs[0].path, refs[0].staged)
}

// expandFilesSection opens one section and, when the selection is still the
// row the fold moved it to, returns it to the row the fold hid.
func (a *app) expandFilesSection(key string) {
	delete(a.filesCollapsed, key)
	saved, ok := a.filesFoldSel[key]
	if !ok {
		return
	}
	delete(a.filesFoldSel, key)
	if a.selectedPath != saved.MovedTo.path || a.selectedStaged != saved.MovedTo.staged {
		return
	}
	for _, ref := range a.fileRefs() {
		if ref.path == saved.Was.path && ref.staged == saved.Was.staged {
			a.revealSelection = true
			a.selectFile(ref.path, ref.staged)
			return
		}
	}
}

func (a *app) fileRow(c *ui.Context, entry gitrepo.Entry, staged bool) ui.Element {
	tk := theme.TokensOf(c)
	primary := a.selectedPath == entry.Path && a.selectedStaged == staged
	selected := a.workspaceFileSelected(fileRef{entry.Path, staged})
	if a.page == "history" {
		// Explicit list membership survives Summary; a legacy preview-only
		// cursor is selected only while its file tab is visible.
		explicitSelection := a.repo != nil && a.workspaceSelection.Root == a.repo.Root && a.workspaceSelection.Files != nil
		selected = selected && a.worktreeSelected && (explicitSelection || a.detailTabShown() == detailTabFile)
		primary = primary && a.worktreeSelected && a.detailTabShown() == detailTabFile
	}
	row := fileRowShell(c, tk, entry.Path, selected)
	if a.page == "history" && a.historyFilesWidth() < scaledFont(130) {
		row.Padding(0, 3).Gap(4)
	}
	a.wireHistoryFileKeys(row, detailFileKey{Path: entry.Path, Staged: staged})
	a.wireFileDiscardKey(row, entry, staged)
	// The row is a button, so it takes Tab, Shift+F10 and the accessibility
	// button role. A focused button also takes Enter and Space for itself,
	// which would only reselect the row; the row's own shortcuts claim those
	// keys first and stage or unstage the focused path/version, matching
	// Sublime Merge's file-list Enter behavior.
	selectForKey := func() bool {
		if !a.worktreeFilesShown() || !a.detailNavigationAllowed() || a.busy || a.repo == nil || len(a.lineSel) > 0 {
			return false
		}
		// Roving focus outside the set acts on its own row; focus inside the
		// set retains membership while making that version the preview.
		a.selectWorkspaceFile(fileRef{entry.Path, staged}, 0, true)
		return true
	}
	row.OnShortcut(0, ui.KeyEnter, func() {
		if selectForKey() {
			a.runWorkspaceStage(a.workspaceStageTargets(entry, staged))
		}
	})
	row.OnShortcut(0, ui.KeySpace, func() {
		if selectForKey() {
			a.runWorkspaceStage(a.workspaceStageTargets(entry, staged))
		}
	})
	row.OnShortcut(ui.Cmd, ui.KeyA, func() { a.selectWorkspaceGroup(fileRef{entry.Path, staged}) })
	a.revealIfSelected(row, primary)
	row.Children(func() {
		a.fileStatusMarker(c, tk, statusCode(entry, staged))
		fileRowPath(c, entry.Path, primary)
		if !entry.Conflict {
			icon, state := lucide.Square(), "unstaged"
			if entry.Staged() {
				icon, state = lucide.SquareCheck(), "staged"
				if entry.WorkspaceUnstaged() {
					icon, state = lucide.SquareMinus(), "partiallyStaged"
				}
			}
			title, version := a.t("stageFile"), a.t("unstaged")
			if staged {
				title, version = a.t("unstageFile"), a.t("staged")
			}
			// State describes the whole path; the action still targets this
			// row's exact Index/worktree version, including either partial row.
			hint := a.t(state) + " · " + version + " · " + title
			if entry.IsSubmodule() {
				hint += "\n" + a.t("fileActionsProtected")
			}
			// The icon sits inside a 26-point row, so it stops short of the
			// row's edges rather than filling them. ActionIcon's Title owns
			// the Lobe tooltip relation; keep the short action name as Label.
			_, ready := a.fileActionTarget(entry, staged)
			button := action.ActionIcon(c, icon, action.ActionIconProps{
				BlockSize: 22, GlyphSize: 12, Title: hint,
			}).Label(title).Disabled(!ready)
			if button.Clicked() {
				a.toggleStage(entry, staged)
				return
			}
		}
	})
	a.fileMenu(c, row, entry, staged)
	// The context menu owns the right click, so a row never sees RightClicked;
	// its open state selects the row instead. The diff and the menu then
	// describe the same file, as the history and branch lists do.
	if *ui.Local(row, "contextMenu", func() bool { return false }) && !primary {
		a.selectWorkspaceFile(fileRef{entry.Path, staged}, 0, true)
	}
	if row.Clicked() {
		a.selectWorkspaceFile(fileRef{entry.Path, staged}, row.ClickModifiers(), false)
	}
	// A double click opens the file in its default app, the way Sublime
	// Merge's file list does; the menu's open action stays for discovery.
	if row.DoubleClicked() {
		a.openFile(entry.Path, "open")
	}
	return row
}

func (a *app) diffBody(c *ui.Context) {
	if entry, conflict := a.conflictPreviewEntry(a.selectedPath); conflict {
		if editor := a.currentConflictEditor(); editor != nil && !editor.Preview {
			a.conflictEditorBody(c, editor)
			return
		}
		a.conflictPreviewGuidance(c, entry)
	}
	switch {
	case a.selectedPath == "":
		a.diffPlaceholder(c, lucide.FileDiff(), a.t("selectFile"))
		return
	case a.diffLoading:
		a.diffPlaceholder(c, lucide.LoaderCircle(), a.t("diffLoading"))
		return
	case a.diffErr != "":
		tk := theme.TokensOf(c)
		ui.Column(c).Fill().Center().Gap(8).Padding(24).Children(func() {
			ui.Icon(c, lucide.AlertTriangle()).FontSize(scaledFont(22)).TextColor(tk.TextTertiary)
			display.Text(c, a.tf("diffFailed", a.diffErr), display.TextProps{Type: display.TextDanger, FontSize: scaledFont(12), Rows: 4}).
				MaxWidth(520).TextAlign(ui.Center)
			if workbenchButton(c, a.t("retry"), buttonNormal, false).Clicked() {
				a.loadDiff()
			}
		})
		return
	case a.diff.Text == "":
		a.diffPlaceholder(c, lucide.FileDiff(), a.t("diffEmpty"))
		return
	}
	_, conflict := a.conflictPreviewEntry(a.selectedPath)
	split := a.splitDiff && !conflict
	diff := a.diff
	visible := worktreeVisiblePatchRows(diff.Lines)
	rows := newPatchRows(diff.Lines, visible, split)
	var widths patchWidths
	if split {
		widths = a.splitPatchWidthsFor(diff.Lines, visible, a.diffViewportRect.W)
	} else {
		widths.canvas = a.diffWidthForRows(diff.Lines, visible)
	}
	widths = a.diffFind.fitMatch(c, detailFileKey{}, diff.Lines, widths, a.patchFontSize())
	heights := make([]float32, rows.len())
	reveal, pressed := -1, -1
	findLine := a.diffFind.revealLine(detailFileKey{})
	for i := range heights {
		row := rows.at(i)
		heights[i] = rows.height(a, i)
		if rowContainsLine(row, findLine) || a.revealLine && rowContainsLine(row, a.lineCursor) {
			reveal = i
		}
		if a.lineDragHeld && rowContainsLine(row, a.linePressIndex) {
			pressed = i
		}
	}
	scroll := layout.ScrollArea(c, layout.ScrollAreaProps{Direction: layout.Both, State: &a.diffScroll}, func() {
		// Keep one selectable container and full extent. Retain the pressed
		// gutter even after it scrolls out, so it owns drag/release events.
		ui.Column(c).Width(widths.canvas).AlignSelf(ui.Start).Shrink(0).Selectable().Children(func() {
			a.paintPatchRows(c, rows, widths, a.diffViewportRect, &a.diffScroll, heights, reveal, pressed, func(cell *ui.Context, index int, cellWidth float32, side patchSide) {
				a.diffLineForSide(cell, diff.Lines[index], cellWidth, true, index, side)
			})
		})
	})
	// Keep the box, not the element: elements expire at the end of a pass,
	// and the drag reads this while the next pass builds.
	if split && a.diffViewportRect.W != scroll.Bounds().W {
		c.After(0)
	}
	a.diffViewportRect = scroll.Bounds()
	scroll.Grow(1).Basis(0).MinHeight(0)
	if a.syncDiffHoverScroll() {
		a.invalidate()
	}
}

// diffPlaceholder fills a pane that has nothing to show with an icon and a
// sentence. The optional hint says how the pane gets content, for the cases
// where the reader has an action to take rather than a wait to sit through.
func (a *app) diffPlaceholder(c *ui.Context, icon *ui.SVG, text string, hint ...string) {
	tk := theme.TokensOf(c)
	ui.Column(c).Fill().Center().Gap(8).Padding(24).Children(func() {
		ui.Icon(c, icon).FontSize(scaledFont(22)).TextColor(tk.TextTertiary)
		display.Text(c, text, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12), Rows: 3}).
			MaxWidth(480).TextAlign(ui.Center)
		if len(hint) > 0 && hint[0] != "" {
			display.Text(c, hint[0], display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11), Rows: 2}).
				MaxWidth(480).TextAlign(ui.Center)
		}
	})
}

// diffCodeSize is the size the patch's code text asks for; the scroll width is
// computed before layout, so it has to estimate the glyphs from it.
const diffCodeSize float32 = 11.5

// diffHoverActionReserve is the room a row keeps on its right for the hover
// action: the icon's box, the scroll bar the pane draws over its right edge,
// and a little clearance.
const diffHoverActionReserve float32 = 34

// diffCodeReserve includes the interactive row's gutter, right padding and
// hover action. The scroll extent must leave all of these outside code text.
const diffCodeReserve float32 = 92 + 12 + diffHoverActionReserve

// diffHoverActionLeft is where a row's hover action sits in the patch's own
// coordinates. The action belongs at the right edge of the visible band: a
// patch wider than the pane would otherwise park it past the horizontal
// scroll, where no click can reach it. Before the pane has been measured the
// row's own end is the best answer.
func (a *app) diffHoverActionLeft(patchWidth float32) float32 {
	left := patchWidth - diffHoverActionReserve
	if viewport := a.diffViewportRect.W; viewport > 0 {
		if visible := a.diffScroll.X + viewport - diffHoverActionReserve; visible < left {
			left = visible
		}
	}
	if left < 0 {
		return 0
	}
	return left
}

// syncDiffHoverScroll records the horizontal scroll the hover action was last
// placed from, and reports whether it moved since. A wheel changes the scroll
// during the pass, after the rows have placed their actions from the old
// value; one more frame lets the action catch up instead of trailing the
// pointer for a frame.
func (a *app) syncDiffHoverScroll() bool {
	if a.diffScroll.X == a.diffHoverSeenX {
		return false
	}
	a.diffHoverSeenX = a.diffScroll.X
	return true
}

func diffEstimatedAdvance(text string) float32 {
	return diffEstimatedAdvanceWith(text, codeFontAdvance())
}

func diffEstimatedAdvanceWith(text string, glyphAdvance float32) float32 {
	advance := float32(0)
	for _, r := range text {
		switch {
		case r == '\t':
			advance += 4 * glyphAdvance // The renderer uses four spaces.
		case r < 0x80:
			advance += glyphAdvance
		default:
			// CJK/fullwidth fallback glyphs use one em, rather than
			// monospace's narrower ordinary advance or two such advances.
			kind := unicodewidth.LookupRune(r).Kind()
			if kind == unicodewidth.EastAsianWide || kind == unicodewidth.EastAsianFullwidth {
				advance++
			} else {
				advance += glyphAdvance
			}
		}
	}
	return advance
}

func (a *app) diffWidth(diff gitrepo.Diff) float32 {
	return a.diffWidthForRows(diff.Lines, nil)
}

func (a *app) diffWidthForRows(lines []gitrepo.DiffLine, visible []int) float32 {
	glyphAdvance := codeFontAdvanceFor(a.codeFontKey())
	var longest float32
	if visible != nil {
		for _, i := range visible {
			longest = max(longest, diffEstimatedAdvanceWith(lines[i].Text, glyphAdvance))
		}
	} else {
		for _, line := range lines {
			longest = max(longest, diffEstimatedAdvanceWith(line.Text, glyphAdvance))
		}
	}
	// The estimate has to follow the reader's text size: a fixed advance that
	// fits the standard preset leaves the large preset's longest lines short.
	size := a.patchFontSize()
	width := longest*size + diffCodeReserve
	// Keep the same maximum code advance when a fixed size enlarges the glyphs.
	limit := float32(20000)
	if a.codeFontSize > diffCodeSize {
		limit = (limit-diffCodeReserve)*size/diffCodeSize + diffCodeReserve
	}
	return min(max(width, 560), limit)
}

func (a *app) diffLine(c *ui.Context, line gitrepo.DiffLine, width float32, index int) {
	a.diffLineFor(c, line, width, true, index)
}

func (a *app) diffLineFor(c *ui.Context, line gitrepo.DiffLine, width float32, interactive bool, index int) {
	a.diffLineForSide(c, line, width, interactive, index, patchBoth)
}

func (a *app) diffLineForSide(c *ui.Context, line gitrepo.DiffLine, width float32, interactive bool, index int, side patchSide) {
	tk := theme.TokensOf(c)
	if line.Kind == "hunk" {
		if interactive {
			a.diffHunkLine(c, line, width, index)
		} else {
			a.plainHunkLine(c, line, width)
		}
		return
	}
	colors := a.diffColors(tk, line.Kind)
	bg, gutter := colors.background, colors.gutter
	change := interactive && (line.Kind == "add" || line.Kind == "del") && index >= 0 && a.lineActionsAllowed()
	if !interactive {
		plain := ui.Row(c).Width(width).Height(a.patchRowHeight()).Shrink(0).Padding(0, 8, 0, 12).Gap(8).AlignItems(ui.Center).
			Background(bg)
		a.paintDiffLine(plain, tk, line.Kind, false, false)
		plain.Children(func() {
			font := a.patchFontFamily()
			ui.Text(c, lineNumber(line.Old)).Font(font).FontSize(scaledFont(10)).Width(34).Shrink(0).
				TextAlign(ui.End).TextColor(gutter)
			ui.Text(c, lineNumber(line.New)).Font(font).FontSize(scaledFont(10)).Width(34).Shrink(0).
				TextAlign(ui.End).TextColor(gutter)
			ui.Text(c, strings.ReplaceAll(line.Text, "\t", "    ")).Font(font).FontSize(a.patchFontSize()).
				SingleLine().TextColor(colors.text).Role(diffCodeRole(line.Kind))
		})
		return
	}
	// Every row of an interactive diff is built the same way, whether it is a
	// change, a context line or a patch header: the line numbers on the left are
	// the handle that selects the whole line, and the code beside them is a
	// paragraph a drag can select from. MyGo only lets a paragraph be selected
	// when no clickable element owns it, so the paragraph sits in the row's
	// content container instead of inside the numbers' button.
	row := ui.Row(c).Width(width).Height(a.patchRowHeight()).Shrink(0).AlignItems(ui.Center)
	a.paintDiffLine(row, tk, line.Kind, change && a.lineSelected(index), index == a.lineCursor)
	var numbers ui.Element
	row.Children(func() {
		inner := ui.Row(c).Height(a.patchRowHeight()).Width(width-diffHoverActionReserve).Shrink(0).
			Padding(0, 12, 0, 0).AlignItems(ui.Center)
		inner.Children(func() {
			numbers = a.patchGutter(c, line, side, gutter)
			markDiffGutter(numbers, change && a.lineSelected(index), gutter)
			a.diffFind.lineText(c, diffFindLine{index: index}, diffFindText(line), a.patchFontSize(), side == patchNew && line.Kind == "context").TextColor(colors.text).Role(diffCodeRole(line.Kind))
		})
		if !change || !row.Hovered() {
			return
		}
		icon, label := lucide.Plus(), a.t("stageLine")
		if a.selectedStaged {
			icon, label = lucide.Minus(), a.t("unstageLine")
		}
		// A filled chip reads above the code the button covers: the row's
		// right end is unpainted space, and the pointer can be anywhere on
		// the row while the action sits further right.
		button := action.ActionIcon(c, icon, action.ActionIconProps{
			BlockSize: 18, GlyphSize: 11, Title: label, Variant: action.ActionIconFilled,
		})
		// The action belongs at the right edge of the pane, not at the end of
		// a long patch row it would sit past the visible area on.
		cellStart := float32(0)
		if side == patchNew {
			cellStart = width + 2 // Left half and the center divider.
		}
		if offset := a.diffHoverActionLeft(cellStart+width) - cellStart - (width - diffHoverActionReserve); offset < 0 {
			button.Left(max(offset, diffHoverActionReserve-width))
		}
		if button.Clicked() {
			// The hover button acts on its own row; the context menu is how a
			// multi-line selection is staged or discarded as a group.
			a.applyLines([]int{index})
		}
	})
	if side == patchBoth || change {
		a.trackLinePress(c, index, numbers)
	}
	if numbers.Clicked() {
		if a.lineDragMoved {
			a.lineDragMoved = false
			a.lineDragging = false
			a.lineReleasePress()
		} else {
			a.lineReleasePress()
			a.clickLine(index, numbers.ClickModifiers())
		}
	}
	if index == a.lineCursor && a.revealLine {
		row.ScrollIntoView()
		a.revealLine = false
	}
	a.lineMenu(c, row, index)
	// Right-clicking outside the current selection selects that line, so the
	// menu's stage/discard/copy acts on what the user pointed at, as Sublime
	// Merge does. A right click inside the selection keeps the whole range.
	if change && *ui.Local(row, "contextMenu", func() bool { return false }) && !a.lineSel[index] {
		a.selectWorkspaceLines([]int{index})
	}
}

// lineGutterLabel names a diff row's line-number handle. It is the numbers, not
// the code: the code is a paragraph beside the handle, and naming the button
// after it would read every changed line twice to a screen reader. A row with
// no numbers (a patch header, a binary notice) falls back to a word that
// follows the interface's language.
// diffCodeRole marks added and deleted code as an insertion or a deletion,
// so assistive technology hears the direction instead of an ordinary
// paragraph whose text happens to start with + or -.
func diffCodeRole(kind string) ui.Role {
	switch kind {
	case "add":
		return ui.RoleInsertion
	case "del":
		return ui.RoleDeletion
	}
	return ui.RoleAuto
}

func (a *app) lineGutterLabel(line gitrepo.DiffLine) string {
	label := strings.TrimSpace(lineNumber(line.Old) + " " + lineNumber(line.New))
	if label == "" {
		label = a.t("diffLineLabel")
	}
	// A deletion and an insertion can share the same numbers in one hunk;
	// the direction keeps their handles apart, as RoleDeletion and
	// RoleInsertion keep their text apart.
	switch line.Kind {
	case "add":
		return label + " " + a.t("diffAddedLine")
	case "del":
		return label + " " + a.t("diffDeletedLine")
	}
	return label
}

func (a *app) plainHunkLine(c *ui.Context, line gitrepo.DiffLine, width float32) {
	tk := theme.TokensOf(c)
	ui.Row(c).Width(width).Height(24).Shrink(0).Padding(0, 12).AlignItems(ui.Center).
		Background(a.diffColors(tk, "hunk").background).Children(func() {
		ui.Text(c, line.Text).Font(a.patchFontFamily()).FontSize(scaledFont(10.5)).SingleLine().
			TextColor(a.diffColors(tk, "hunk").text)
	})
}

func (a *app) diffHunkLine(c *ui.Context, line gitrepo.DiffLine, width float32, index int) {
	tk := theme.TokensOf(c)
	// A ButtonBase gives the row state, which the context menu needs to open;
	// a stateless Row would silently ignore the secondary click.
	// The header's text gets the band left of the action, so a long hunk
	// header and the hover button never fight over the same pixels.
	row := ui.ButtonBase(c.Key(a.hunkFocusKey(line.Hunk))).Label(line.Text).Row().Width(width).Height(24).Shrink(0).AlignItems(ui.Center).
		Justify(ui.Start).
		Background(a.diffColors(tk, "hunk").background)
	a.wireHunkEnter(row, line.Hunk)
	row.Children(func() {
		inner := ui.Row(c).Height(24).Width(width-diffHoverActionReserve).Shrink(0).
			Padding(0, 8, 0, 12).AlignItems(ui.Center)
		inner.Children(func() {
			a.diffFind.lineText(c, diffFindLine{index: index}, line.Text, scaledFont(10.5)).
				TextColor(a.diffColors(tk, "hunk").text)
		})
		ready := a.hunkActionsAllowed() &&
			line.Hunk >= 0 && line.Hunk < len(a.diff.Hunks)
		if !ready || !row.Hovered() {
			return
		}
		icon, label := lucide.Plus(), a.t("stageHunk")
		if a.selectedStaged {
			icon, label = lucide.Minus(), a.t("unstageHunk")
		}
		button := action.ActionIcon(c, icon, action.ActionIconProps{
			BlockSize: 20, GlyphSize: 12, Title: label, Variant: action.ActionIconFilled,
		})
		if offset := a.diffHoverActionLeft(width) - (width - diffHoverActionReserve); offset < 0 {
			button.Left(offset)
		}
		if button.Clicked() {
			a.applyHunk(line.Hunk)
		}
	})
	a.hunkMenu(c, row, line.Hunk)
}

func (a *app) hunkActionsAllowed() bool {
	if a.repo == nil || a.busy || a.loading || !a.diff.CanStageHunks {
		return false
	}
	entry, ok := a.selectedEntry()
	// A dirty submodule prints a "Subproject commit X-dirty" pseudo patch that
	// git apply accepts without changing anything; staging goes through the
	// file row, which updates the pointer or the submodule itself. A staged
	// rename is refused by the backend for the same reason: applying a hunk
	// into an index that also carries the rename would split one change across
	// two paths, so the file row stages it whole.
	return ok && !entry.Conflict && !entry.IsSubmodule() && entry.OriginalPath == ""
}

func (a *app) lineActionsAllowed() bool {
	if !a.hunkActionsAllowed() {
		return false
	}
	entry, ok := a.selectedEntry()
	return ok && entry.OriginalPath == "" && !entry.IsSubmodule()
}

func (a *app) applyHunk(index int) {
	if a.repo == nil || !a.hunkActionsAllowed() || index < 0 || index >= len(a.diff.Hunks) {
		return
	}
	patch := a.diff.Hunks[index].Patch
	staged := a.selectedStaged
	label, success := a.t("stageHunkFailed"), a.t("stageHunkSuccess")
	if staged {
		label, success = a.t("unstageHunkFailed"), a.t("unstageHunkSuccess")
	}
	a.perform(label, success, func(ctx context.Context, repo *gitrepo.Repository) error {
		return repo.ApplyHunk(ctx, patch, staged)
	})
}

func (a *app) hunkMenu(c *ui.Context, row ui.Element, index int) {
	if index < 0 || index >= len(a.diff.Hunks) {
		return
	}
	var items []overlay.MenuItem
	if a.hunkActionsAllowed() {
		label, key := a.t("stageHunk"), "stage"
		icon := lucide.Plus()
		if a.selectedStaged {
			label, key, icon = a.t("unstageHunk"), "unstage", lucide.Minus()
		}
		items = append(items, overlay.MenuItem{Key: key, Label: label, Icon: icon})
	}
	if a.discardLinesAllowed() {
		items = append(items,
			overlay.MenuItem{Divider: true},
			overlay.MenuItem{Key: "discard", Label: a.t("discardHunk"), Icon: lucide.Undo2(), Danger: true},
		)
	}
	items = append(items,
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{Key: "copy", Label: a.t("copyHunk"), Icon: lucide.Copy()},
	)
	switch overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: items}) {
	case "stage", "unstage":
		a.applyHunk(index)
	case "discard":
		a.confirmDiscardHunk(index)
	case "copy":
		a.copyText(diffHunkText(a.diff, index))
	}
}

// confirmDiscardHunk asks before dropping a whole hunk from the working tree,
// then discards its changed lines through the shared line discard, which
// keeps the patch for the undo action.
func (a *app) confirmDiscardHunk(index int) {
	if !a.discardLinesAllowed() || index < 0 || index >= len(a.diff.Hunks) {
		return
	}
	choice := hunkChangedLines(a.diff, index)
	if len(choice) == 0 {
		return
	}
	path, diff := a.selectedPath, a.diff
	a.askConfirm(a.t("discardHunkTitle"), a.discardHunkMessage(len(choice)), a.t("discard"), true, func() {
		a.discardLines(path, diff, choice)
	})
}

func lineNumber(n int) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprint(n)
}

func (a *app) operationCard(c *ui.Context) {
	if a.operation.Kind == "" {
		return
	}
	kind := a.operation.Kind
	layout.Block(c, layout.BlockProps{Variant: layout.BlockOutlined}).Padding(12).Gap(8).Children(func() {
		display.Text(c, a.tf("operation", kind), display.TextProps{Type: display.TextWarning, Strong: true, FontSize: scaledFont(12)})
		ui.Row(c).Gap(6).Children(func() {
			// A merge finishes through the commit box above, which
			// carries the message typed there; the other operations
			// continue through git's own command.
			if kind != "merge" {
				cont := workbenchButton(c, a.t("continueOperation"), buttonNormal,
					a.busy || a.loading || a.operationConflicts())
				if a.operationConflicts() {
					cont.Tooltip(a.t("operationConflicts"))
				}
				if cont.Clicked() {
					a.continueOperation(kind)
				}
				if workbenchButton(c, a.t("skipOperation"), buttonNormal, a.busy || a.loading).Clicked() {
					a.confirmSkipOperation(kind)
				}
			}
			if workbenchButton(c, a.t("abortOperation"), buttonDanger, a.busy || a.loading).Clicked() {
				a.confirmAbortOperation(kind)
			}
		})
	})
}

func (a *app) canShowAmendControl() bool {
	return a.headMessage != "" || a.status.Head != ""
}

func (a *app) commitAmendControl(c *ui.Context) {
	wasAmend := a.amend
	if a.canShowAmendControl() {
		choice.Checkbox(c, &a.amend, a.t("amendLast"), choice.CheckboxProps{Size: 14})
	}
	if a.amend != wasAmend {
		// Always prefill HEAD, even when browsing another branch.
		latest := a.headMessage
		switch {
		case a.amend && strings.TrimSpace(a.commitText) == "":
			a.commitText = latest
		case !a.amend && latest != "" && a.commitText == latest:
			a.commitText = ""
		}
	}
}

// commitStatusPill shows the staged count or Amend HEAD state beside the
// compact Commit button so the action button keeps a short, fixed label.
func (a *app) commitStatusPill(c *ui.Context, staged int) {
	tk := theme.TokensOf(c)
	text := a.t("noStagedChanges")
	bg := tk.BgContainer
	border := tk.BorderSecondary
	ink := tk.TextTertiary
	dot := tk.TextTertiary
	switch {
	case a.amend:
		text = a.t("amendHead")
		bg = tk.PrimaryBg
		ink = tk.PrimaryText
		dot = tk.PrimaryText
		border = tk.PrimaryText.Alpha(0.28)
	case staged > 0:
		text = a.stagedLabel(staged)
		bg = tk.SuccessBg
		ink = tk.SuccessText
		dot = tk.SuccessText
		border = tk.SuccessText.Alpha(0.28)
	}
	ui.Row(c).Height(scaledFont(24)).Padding(0, 9).Gap(5).Radius(999).
		AlignItems(ui.Center).Shrink(0).
		Background(bg).Border(1, border).Children(func() {
		ui.Box(c).Size(6, 6).Radius(3).Background(dot).Shrink(0)
		ui.Text(c, text).FontSize(scaledFont(11)).FontWeight(500).TextColor(ink).SingleLine()
	})
}

// Both layouts share the same readiness and operation guards. Changes keeps
// this footer outside its scrolling editor so a long message cannot hide it.
func (a *app) commitAction(c *ui.Context, showCount bool) {
	staged := len(a.stagedEntries())
	label := a.t("commit")
	if a.amend {
		label = a.t("amend")
	}
	ready := a.canCommit()
	reason := a.commitDisabledReason()
	description := reason
	if showCount {
		if reason != "" {
			description = a.stagedLabel(staged) + ". " + reason
		} else {
			description = a.stagedLabel(staged)
		}
	}
	ui.Row(c).Shrink(0).AlignItems(ui.Center).Gap(10).Children(func() {
		if showCount {
			a.commitStatusPill(c, staged)
		}
		button := workbenchButton(c, label, buttonPrimary, !ready).
			Label(label).MinWidth(scaledFont(76)).PaddingX(14).Shrink(0).
			KeyShortcuts(ui.Cmd, ui.KeyEnter).
			Description(description).Tooltip(reason)
		if button.Clicked() {
			a.commit()
		}
	})
}

// commitActionReason surfaces blocking operational errors below the footer row
// without stretching the compact Commit button.
func (a *app) commitActionReason(c *ui.Context) {
	reason := a.commitDisabledReason()
	if reason == "" || reason == a.t("stagePrompt") {
		return
	}
	display.Text(c, reason, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(10.5), Rows: 2}).FillWidth()
}
