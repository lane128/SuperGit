package lobedesk

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/choice"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/field"
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

func (a *app) remoteActions(c *ui.Context) {
	blocked := a.busy || a.loading || a.repo == nil
	noRemote := len(a.remotes) == 0
	fetchHint := ""
	if noRemote {
		fetchHint = a.t("noRemote")
	}
	fetch := a.remoteActionButton(c, lucide.CloudDownload(), a.t("fetch"),
		blocked || noRemote, fetchHint, func() { a.fetchAll() })
	a.fetchButtonMenu(c, fetch)
	pullHint := ""
	if a.status.Upstream == "" {
		pullHint = a.t("noUpstream")
	}
	a.remoteActionButton(c, lucide.ArrowDown(), a.t("pull"),
		blocked || a.status.Upstream == "", pullHint, func() { a.pullFF() })
	pushHint := ""
	switch {
	case noRemote:
		pushHint = a.t("noRemote")
	case a.status.Branch == "":
		pushHint = a.t("pushNoBranch")
	}
	push := a.remoteActionButton(c, lucide.ArrowUp(), a.t("pushButton"),
		blocked || noRemote || a.status.Branch == "", pushHint, func() { a.confirmPush() })
	a.pushButtonMenu(c, push)
}

// remoteActionButton draws one toolbar icon. A disabled button hides its own
// tooltip, so the reason it is disabled sits on a wrapper Box, the way the
// branch details footer explains its locked actions. The busy state keeps no
// hint: it ends by itself, and it is not something the reader has to fix.
func (a *app) remoteActionButton(c *ui.Context, icon *ui.SVG, title string, disabled bool, hint string, onClick func()) ui.Element {
	var button ui.Element
	draw := func() {
		button = action.ActionIcon(c, icon, action.ActionIconProps{Size: theme.Small, Title: title}).Disabled(disabled)
		if hint != "" {
			// The wrapper explains why on hover; the button carries the same
			// reason to assistive technology, which a disabled element's own
			// tooltip would not reach.
			button.Description(hint)
		}
		if button.Clicked() && !disabled {
			onClick()
		}
	}
	if disabled && hint != "" {
		ui.Box(c).Shrink(0).Tooltip(hint).Children(draw)
	} else {
		draw()
	}
	return button
}

// fetchButtonMenu keeps the one-click fetch and puts the pruning variant
// behind a right click, the same shortcut Sublime Merge's fetch button uses.
func (a *app) fetchButtonMenu(c *ui.Context, button ui.Element) {
	if len(a.remotes) == 0 || a.busy || a.loading {
		return
	}
	switch overlay.ContextMenu(c, button, overlay.DropdownMenuProps{Items: []overlay.MenuItem{
		{Key: "fetch", Label: a.t("fetch"), Icon: lucide.CloudDownload()},
		{Key: "prune", Label: a.t("fetchPrune"), Icon: lucide.Scissors()},
	}}) {
	case "fetch":
		a.fetchAll()
	case "prune":
		a.fetchPruneAll()
	}
}

// pushButtonMenu adds the force variant behind a right click on the push
// button, so a normal push stays a single click.
func (a *app) pushButtonMenu(c *ui.Context, button ui.Element) {
	if len(a.remotes) == 0 || a.status.Branch == "" || a.busy || a.loading {
		return
	}
	switch overlay.ContextMenu(c, button, overlay.DropdownMenuProps{Items: []overlay.MenuItem{
		{Key: "push", Label: a.t("pushBranchAction"), Icon: lucide.Upload()},
		{Key: "force", Label: a.t("forcePushAction"), Icon: lucide.ShieldAlert(), Danger: true},
	}}) {
	case "push":
		a.confirmPush()
	case "force":
		a.confirmForcePush()
	}
}

func (a *app) branchSelector(c *ui.Context) {
	tk := theme.TokensOf(c)
	branch := a.status.Branch
	if branch == "" {
		branch = a.t("detached")
		a.statusTag(c, branch, display.TagProps{Variant: display.TagOutlined, Color: display.TagWarning, Size: theme.Small, Round: true})
		return
	}
	if len(a.branches) == 0 {
		a.statusTag(c, branch, display.TagProps{Variant: display.TagFilled, Color: display.TagInfo, Size: theme.Small, Round: true, Icon: lucide.GitBranch()})
		return
	}
	open := &a.branchOpen
	items := make([]overlay.MenuItem, 0, len(a.branches))
	for _, item := range a.branches {
		items = append(items, overlay.MenuItem{
			Key:      item.Name,
			Label:    item.Name,
			Icon:     lucide.GitBranch(),
			Extra:    item.Upstream,
			Disabled: item.Current,
		})
	}
	chosen := overlay.DropdownMenu(c, open, overlay.DropdownMenuProps{Items: items}, func(trigger ui.Element) {
		action.StyleButton(c, trigger, "", action.ButtonProps{Variant: action.Default, Size: theme.Small})
		// The trigger shows only the branch name and an icon; without a name
		// of its own a reader hears just the branch and cannot tell it is the
		// branch switch.
		trigger.Role(ui.RoleButton).Label(a.t("branches")).Description(branch)
		trigger.Gap(6).Children(func() {
			ui.Icon(c, lucide.GitBranch()).FontSize(scaledFont(13)).TextColor(tk.TextSecondary)
			ui.Text(c, branch).FontSize(scaledFont(12)).SingleLine()
			ui.Icon(c, lucide.ChevronDown()).FontSize(scaledFont(11)).TextColor(tk.TextTertiary)
		})
	})
	if chosen != "" {
		a.switchBranch(chosen)
	}
}

// filesToolbar holds the working tree's bulk stage actions. They sit in the
// history page's Files column header now that the working tree is its first
// row.
func (a *app) filesToolbar(c *ui.Context) {
	unstaged := a.unstagedEntries()
	staged := a.stagedEntries()
	untracked := a.untrackedEntries()
	stageable := stageableCount(unstaged, untracked)
	build := func() {
		stageAll := action.ActionIcon(c, lucide.Plus(), action.ActionIconProps{
			BlockSize: 24, GlyphSize: 14, Title: a.t("stageAll"),
		}).Disabled(stageable == 0 || a.busy || a.loading)
		if stageAll.Clicked() {
			a.stageAll()
		}
	}
	// A list of nothing but conflicts has nothing this button may stage; say
	// why rather than dimming it with no explanation.
	if stageable == 0 && len(unstaged) > 0 && !a.busy && !a.loading {
		ui.Box(c).Shrink(0).Tooltip(a.t("stageAllConflicts")).Children(build)
	} else {
		build()
	}
	unstageAll := action.ActionIcon(c, lucide.Minus(), action.ActionIconProps{
		BlockSize: 24, GlyphSize: 14, Title: a.t("unstageAll"),
	}).Disabled(len(staged) == 0 || a.busy || a.loading)
	if unstageAll.Clicked() {
		a.unstageAll()
	}
}

// workingFileSections draws the working tree's files in the order the commit
// form reads them: what is waiting (unstaged), what will be committed
// (staged), then new files. The history page's Files column shows the working
// tree through it whenever its first row is selected.
func (a *app) workingFileSections(c *ui.Context) {
	if len(a.status.Entries) == 0 {
		a.cleanFiles(c)
		return
	}
	unstaged := a.unstagedEntries()
	staged := a.stagedEntries()
	untracked := a.untrackedEntries()
	if len(unstaged) > 0 {
		a.fileSection(c, filesSectionUnstaged, a.t("unstaged"), unstaged, false)
	}
	if len(staged) > 0 {
		a.fileSection(c, filesSectionStaged, a.t("staged"), staged, true)
	}
	// New files come last: they are not part of the change the commit form
	// will take until they are staged, and keeping them below the two tracked
	// groups keeps the reading order from "what is waiting" to "what will be
	// committed".
	if len(untracked) > 0 {
		a.fileSection(c, filesSectionUntracked, a.t("untracked"), untracked, false)
	}
}

func (a *app) cleanFiles(c *ui.Context) {
	listEmptyState(c, lucide.CircleCheck(), theme.TokensOf(c).Success, a.t("clean"), a.t("cleanHint"))
}

// fileSection draws one group of files under a header that folds it. The count
// stays on a folded header, so the reader can still tell what is put away.
func (a *app) fileSection(c *ui.Context, key, title string, entries []gitrepo.Entry, staged bool) {
	tk := theme.TokensOf(c)
	collapsed := a.fileSectionCollapsed(key)
	ui.Column(c).FillWidth().Gap(1).Children(func() {
		header := ui.ButtonBase(c).Label(title).Row().FillWidth().Height(28).PaddingX(8).Gap(6).
			Radius(tk.BorderRadius).AlignItems(ui.Center).Expanded(!collapsed).
			// The header folds the section and its count keeps counting the
			// folded rows; say both, or the number reads as what is on screen.
			Tooltip(a.t("fileSectionHint"))
		if header.Hovered() {
			header.Background(tk.FillTertiary)
		}
		header.OnClick(func() { a.toggleFilesSection(key) })
		header.Children(func() {
			chevron := lucide.ChevronDown()
			if collapsed {
				chevron = lucide.ChevronRight()
			}
			ui.Icon(c, chevron).FontSize(scaledFont(12)).TextColor(tk.TextTertiary)
			display.Text(c, title, display.TextProps{Type: display.TextSecondary, Strong: true, FontSize: scaledFont(11)})
			display.Text(c, fmt.Sprint(len(entries)), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)})
			// A button centers what it holds; the spacer keeps the header
			// reading from the left edge like the rows below it.
			ui.Row(c).Grow(1).Basis(0)
		})
		if collapsed {
			return
		}
		// The rows are the section's options, with their place and the
		// section's name in the description: the same path can appear in
		// two sections as "unstaged" and "staged", and a screen reader
		// otherwise hears two identical buttons.
		ui.Column(c).FillWidth().Gap(1).Role(ui.RoleList).Children(func() {
			for i := range entries {
				row := a.fileRow(c, entries[i], staged)
				// The description names the section; the tooltip gives the full
				// path without replacing that status in the accessibility tree.
				row.PositionInSet(i+1, len(entries)).Description(title).Tooltip(entries[i].Path)
			}
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
	selected := a.selectedPath == entry.Path && a.selectedStaged == staged
	row := fileRowShell(c, tk, entry.Path, selected)
	// The row is a button, so it takes Tab, Shift+F10 and the accessibility
	// button role. A focused button also takes Enter and Space for itself,
	// which would only reselect the row; the row's own shortcuts claim those
	// keys first and run the page's actions (open the file, stage or unstage
	// it), so mouse and keyboard paths agree.
	row.OnShortcut(0, ui.KeyEnter, func() { a.openSelectedFile() })
	row.OnShortcut(0, ui.KeySpace, func() { a.toggleSelectedStage() })
	a.revealIfSelected(row, selected)
	row.Children(func() {
		a.statusTag(c, statusCode(entry, staged), display.TagProps{
			Variant: display.TagFilled,
			Color:   display.TagColor(statusTagColor(entry, staged)),
			Size:    theme.Small,
			Round:   true,
		})
		fileRowPath(c, row, entry.Path, entry.OriginalPath, selected)
		if entry.Conflict {
			a.statusTag(c, "!", display.TagProps{Variant: display.TagFilled, Color: display.TagError, Size: theme.Small, Round: true})
		} else {
			icon := lucide.Plus()
			title := a.t("stageFile")
			if staged {
				icon = lucide.Minus()
				title = a.t("unstageFile")
			}
			// The icon sits inside a 26-point row, so it stops short of the
			// row's edges rather than filling them.
			button := action.ActionIcon(c, icon, action.ActionIconProps{
				BlockSize: 22, GlyphSize: 12, Title: title,
			}).Disabled(a.busy || a.loading)
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
	if *ui.Local(row, "contextMenu", func() bool { return false }) && !selected {
		a.selectFile(entry.Path, staged)
	}
	if row.Clicked() {
		a.selectFile(entry.Path, staged)
	}
	// A double click opens the file in its default app, the way Sublime
	// Merge's file list does; the menu's open action stays for discovery.
	if row.DoubleClicked() {
		a.openFile(entry.Path, "open")
	}
	return row
}

func (a *app) diffBody(c *ui.Context) {
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
			if action.Button(c, a.t("retry"), action.ButtonProps{Variant: action.Default, Size: theme.Small}).Clicked() {
				a.loadDiff()
			}
		})
		return
	case a.diff.Text == "":
		a.diffPlaceholder(c, lucide.FileDiff(), a.t("diffEmpty"))
		return
	}
	width := diffWidth(a.diff)
	scroll := layout.ScrollArea(c, layout.ScrollAreaProps{Direction: layout.Both, State: &a.diffScroll}, func() {
		// One selectable container for the whole patch: a drag across rows then
		// selects the code as text, and Cmd+C copies exactly what is highlighted.
		ui.Column(c).Width(width).AlignSelf(ui.Start).Shrink(0).Selectable().Children(func() {
			for i, line := range a.diff.Lines {
				a.diffLine(c, line, width, i)
			}
		})
	})
	// Keep the box, not the element: elements expire at the end of a pass,
	// and the drag reads this while the next pass builds.
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

// menloAdvance is Menlo's advance as a fraction of its size, rounded up a
// little: over-measuring costs a few pixels of scroll range, under-measuring
// clips the end of a long line where no scroll can reach it.
const menloAdvance float32 = 0.62

// diffHoverActionReserve is the room a row keeps on its right for the hover
// action: the icon's box, the scroll bar the pane draws over its right edge,
// and a little clearance.
const diffHoverActionReserve float32 = 34

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

func diffWidth(diff gitrepo.Diff) float32 {
	longest := 0
	for _, line := range diff.Lines {
		n := len([]rune(strings.ReplaceAll(line.Text, "\t", "    ")))
		if n > longest {
			longest = n
		}
	}
	// The estimate has to follow the reader's text size: a fixed advance that
	// fits the standard preset leaves the large preset's longest lines short.
	width := float32(longest)*scaledFont(diffCodeSize)*menloAdvance + 110
	return min(max(width, 560), 20000)
}

func (a *app) diffLine(c *ui.Context, line gitrepo.DiffLine, width float32, index int) {
	a.diffLineFor(c, line, width, true, index)
}

func (a *app) diffLineFor(c *ui.Context, line gitrepo.DiffLine, width float32, interactive bool, index int) {
	tk := theme.TokensOf(c)
	if line.Kind == "hunk" {
		if interactive {
			a.diffHunkLine(c, line, width)
		} else {
			a.plainHunkLine(c, line, width)
		}
		return
	}
	bg := tk.BgContainer
	switch line.Kind {
	case "add":
		bg = tk.Success.Alpha(0.075).Over(tk.BgContainer)
	case "del":
		bg = tk.Error.Alpha(0.065).Over(tk.BgContainer)
	case "meta":
		bg = tk.FillQuaternary.Over(tk.BgContainer)
	}
	change := interactive && (line.Kind == "add" || line.Kind == "del") && index >= 0 && a.lineActionsAllowed()
	if change && a.lineSelected(index) {
		alpha := float32(0.16)
		if index == a.lineCursor {
			alpha = 0.24
		}
		bg = tk.Blue.Alpha(alpha).Over(bg)
	}
	gutter := gutterInk(tk, bg)
	if !interactive {
		plain := ui.Row(c).Width(width).Height(diffRowH).Shrink(0).Padding(0, 8, 0, 12).Gap(8).AlignItems(ui.Center).
			Background(bg)
		plain.Children(func() {
			ui.Text(c, lineNumber(line.Old)).Font("Menlo").FontSize(scaledFont(10)).Width(34).Shrink(0).
				TextAlign(ui.End).TextColor(gutter)
			ui.Text(c, lineNumber(line.New)).Font("Menlo").FontSize(scaledFont(10)).Width(34).Shrink(0).
				TextAlign(ui.End).TextColor(gutter)
			ui.Text(c, strings.ReplaceAll(line.Text, "\t", "    ")).Font("Menlo").FontSize(scaledFont(diffCodeSize)).
				SingleLine().TextColor(lineTextColor(tk, line.Kind)).Role(diffCodeRole(line.Kind))
		})
		return
	}
	// Every row of an interactive diff is built the same way, whether it is a
	// change, a context line or a patch header: the line numbers on the left are
	// the handle that selects the whole line, and the code beside them is a
	// paragraph a drag can select from. MyGo only lets a paragraph be selected
	// when no clickable element owns it, so the paragraph sits in the row's
	// content container instead of inside the numbers' button.
	row := ui.Row(c).Width(width).Height(diffRowH).Shrink(0).AlignItems(ui.Center).Background(bg)
	var numbers ui.Element
	row.Children(func() {
		inner := ui.Row(c).Height(diffRowH).Width(width-diffHoverActionReserve).Shrink(0).
			Padding(0, 12, 0, 0).AlignItems(ui.Center)
		inner.Children(func() {
			numbers = ui.ButtonBase(c).Label(a.lineGutterLabel(line)).Row().Width(92).Height(diffRowH).Shrink(0).
				Padding(0, 8).Gap(8).AlignItems(ui.Center)
			numbers.Children(func() {
				ui.Text(c, lineNumber(line.Old)).Font("Menlo").FontSize(scaledFont(10)).Width(34).Shrink(0).
					TextAlign(ui.End).TextColor(gutter)
				ui.Text(c, lineNumber(line.New)).Font("Menlo").FontSize(scaledFont(10)).Width(34).Shrink(0).
					TextAlign(ui.End).TextColor(gutter)
			})
			ui.Text(c, strings.ReplaceAll(line.Text, "\t", "    ")).Font("Menlo").FontSize(scaledFont(diffCodeSize)).
				SingleLine().TextColor(lineTextColor(tk, line.Kind)).Role(diffCodeRole(line.Kind))
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
		if offset := a.diffHoverActionLeft(width) - (width - diffHoverActionReserve); offset < 0 {
			button.Left(offset)
		}
		if button.Clicked() {
			// The hover button acts on its own row; the context menu is how a
			// multi-line selection is staged or discarded as a group.
			a.applyLines([]int{index})
		}
	})
	a.trackLinePress(c, index, numbers)
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
		Background(tk.Info.Alpha(0.06).Over(tk.BgContainer)).Children(func() {
		ui.Text(c, line.Text).Font("Menlo").FontSize(scaledFont(10.5)).SingleLine().
			TextColor(tk.Info)
	})
}

func (a *app) diffHunkLine(c *ui.Context, line gitrepo.DiffLine, width float32) {
	tk := theme.TokensOf(c)
	// A ButtonBase gives the row state, which the context menu needs to open;
	// a stateless Row would silently ignore the secondary click.
	// The header's text gets the band left of the action, so a long hunk
	// header and the hover button never fight over the same pixels.
	row := ui.ButtonBase(c).Label(line.Text).Row().Width(width).Height(24).Shrink(0).AlignItems(ui.Center).
		Justify(ui.Start).
		Background(tk.Info.Alpha(0.06).Over(tk.BgContainer))
	row.Children(func() {
		inner := ui.Row(c).Height(24).Width(width-diffHoverActionReserve).Shrink(0).
			Padding(0, 8, 0, 12).AlignItems(ui.Center)
		inner.Children(func() {
			ui.Text(c, line.Text).Font("Menlo").FontSize(scaledFont(10.5)).SingleLine().
				TextColor(tk.Info)
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
	return ok && entry.Index != '?' && entry.OriginalPath == "" && !entry.IsSubmodule()
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

func lineTextColor(tk *theme.Tokens, kind string) ui.Color {
	switch kind {
	case "add":
		return tk.Text
	case "del":
		return tk.TextSecondary
	case "meta":
		// Hunk headers are read like any other diff line, so they take the
		// secondary text token: TextDescription sits at 3.2:1 on the meta
		// row's fill in the light theme. The tone stays recessive by size and
		// position rather than by contrast.
		return tk.TextSecondary
	}
	return tk.Text
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
				cont := action.Button(c, a.t("continueOperation"), action.ButtonProps{Variant: action.Default, Size: theme.Small}).
					Disabled(a.busy || a.loading || a.operationConflicts())
				if a.operationConflicts() {
					cont.Tooltip(a.t("operationConflicts"))
				}
				if cont.Clicked() {
					a.continueOperation(kind)
				}
				if action.Button(c, a.t("skipOperation"), action.ButtonProps{Variant: action.Default, Size: theme.Small}).
					Disabled(a.busy || a.loading).Clicked() {
					a.confirmSkipOperation(kind)
				}
			}
			if action.Button(c, a.t("abortOperation"), action.ButtonProps{Variant: action.Default, Size: theme.Small}).
				Disabled(a.busy || a.loading).Clicked() {
				a.confirmAbortOperation(kind)
			}
		})
	})
}

func (a *app) commitForm(c *ui.Context) {
	staged := len(a.stagedEntries())
	branch := a.status.Branch
	if branch == "" {
		branch = a.t("detached")
	}
	layout.Block(c, layout.BlockProps{Variant: layout.BlockFilled}).FillWidth().
		Padding(14).Gap(10).Children(func() {
		ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
			display.Text(c, a.t("commit"), display.TextProps{Strong: true, FontSize: scaledFont(13)})
			display.Tag(c, a.stagedLabel(staged), display.TagProps{
				Variant: display.TagFilled, Color: display.TagColor(stagedTagColor(staged)), Size: theme.Small, Round: true,
			})
		})
		display.Text(c, a.tf("commitTo", branch), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)})
		ui.Column(c).FillWidth().Children(func() {
			field.TextArea(c, &a.commitText, field.TextAreaProps{
				Variant:     field.InputFilled,
				Placeholder: a.t("commitPlaceholder"),
				MinRows:     4,
				MaxRows:     10,
			}).FillWidth().FontSize(scaledFont(13)).LineHeight(1.45)
		})
		display.Text(c, a.t("commitHelp"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(10), Rows: 2})
		wasAmend := a.amend
		if a.headSubject != "" {
			choice.Checkbox(c, &a.amend, a.t("amendLast"), choice.CheckboxProps{Size: 14})
		}
		if a.amend != wasAmend {
			// Amend rewrites HEAD, so its prefill is HEAD's subject even while
			// the list is scoped to another branch.
			latest := a.headSubject
			switch {
			case a.amend && strings.TrimSpace(a.commitText) == "":
				a.commitText = latest
			case !a.amend && latest != "" && a.commitText == latest:
				a.commitText = ""
			}
		}
		label := a.t("commit")
		if a.amend {
			label = a.t("amend")
		}
		ready := a.canCommit()
		variant := action.Primary
		if !ready {
			variant = action.Text
		}
		button := action.Button(c, label, action.ButtonProps{Variant: variant, Size: theme.Large}).
			FillWidth().Disabled(!ready).KeyShortcuts(ui.Cmd, ui.KeyEnter)
		if button.Clicked() {
			a.commit()
		}
		switch {
		case a.operation.Kind != "" && !commitOperation(a.operation.Kind):
			// A rebase continues through its own button, not this box.
			style := display.TextSecondary
			if a.operationConflicts() {
				style = display.TextWarning
			}
			display.Text(c, a.tf("operationContinuePrompt", a.operation.Kind), display.TextProps{
				Type: style, FontSize: scaledFont(10),
			})
		case commitOperation(a.operation.Kind) && a.operationConflicts():
			// The commit finishes the operation, but only once every
			// conflicted file is resolved.
			display.Text(c, a.tf("commitOperationConflictPrompt", a.operation.Kind), display.TextProps{
				Type: display.TextWarning, FontSize: scaledFont(10),
			})
		case staged == 0 && !a.amend:
			prompt := a.t("stagePrompt")
			if commitOperation(a.operation.Kind) {
				prompt = a.tf("commitOperationPrompt", a.operation.Kind)
			}
			display.Text(c, prompt, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(10)})
		}
	})
}
