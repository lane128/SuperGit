package lobedesk

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"supergit/internal/gitrepo"
)

// line selection lets a diff row choose one or several changed lines, then act
// on all of them at once. Context and meta lines never take part.

func (a *app) clearLineSelection() {
	a.overviewLineTarget = nil
	a.lineSel = nil
	a.lineAnchor = -1
	a.lineCursor = -1
	a.revealLine = false
	a.linePressIndex = -1
	a.lineDragHeld = false
	a.lineDragging = false
	a.lineDragMoved = false
}

// lineIndices lists the changed rows of the current diff in order, which is
// the set the keyboard walks and the mouse extends over.
func (a *app) lineIndices() []int {
	diff := a.activeWorktreeDiff()
	if diff.Lines == nil {
		return nil
	}
	var out []int
	for i, line := range diff.Lines {
		if line.Kind == "add" || line.Kind == "del" {
			out = append(out, i)
		}
	}
	return out
}

// moveLineSelection walks changed rows once a line selection exists. With
// extend it keeps the anchor and grows or shrinks the range, as Shift+arrows
// do in an editor; without it the new row becomes the only selection.
func (a *app) moveLineSelection(delta int, extend bool) bool {
	if a.page == "history" && !a.worktreeDiffShown() {
		return false
	}
	if len(a.lineSel) == 0 {
		return false
	}
	indices := a.lineIndices()
	if len(indices) == 0 {
		return false
	}
	current := a.lineCursor
	if current < 0 {
		current = a.lineAnchor
	}
	position := 0
	for i, index := range indices {
		if index == current {
			position = i
			break
		}
	}
	if current >= 0 {
		position = clampIndex(position+delta, len(indices))
	}
	target := indices[position]
	if extend {
		lo, hi := a.lineAnchor, target
		if lo < 0 || lo >= len(a.activeWorktreeDiff().Lines) {
			lo = target
		}
		if lo > hi {
			lo, hi = hi, lo
		}
		next := make(map[int]bool, hi-lo+1)
		for _, index := range indices {
			if index >= lo && index <= hi {
				next[index] = true
			}
		}
		a.lineSel = next
	} else {
		a.lineSel = map[int]bool{target: true}
		a.lineAnchor = target
	}
	a.lineCursor = target
	a.revealLine = true
	a.invalidate()
	return true
}

func (a *app) lineSelected(index int) bool {
	return a.lineSel[index]
}

// lineTargets returns the rows an action on index applies to: the whole
// selection when index is part of it, otherwise just index.
func (a *app) lineTargets(index int) []int {
	if !a.lineSel[index] || len(a.lineSel) == 0 {
		return []int{index}
	}
	targets := make([]int, 0, len(a.lineSel))
	for i := range a.lineSel {
		targets = append(targets, i)
	}
	sort.Ints(targets)
	return targets
}

const (
	// lineAutoScrollBand is how close to the diff's top or bottom edge a
	// drag has to hold before the view scrolls on its own.
	lineAutoScrollBand = 36
	// lineAutoScrollTopSpeed is how far a frame scrolls at the very edge,
	// in DIPs; the speed falls off towards the inner side of the band.
	lineAutoScrollTopSpeed = 24
)

// autoScrollStep is how far a drag scrolls per frame with the pointer at
// pointerY in a view spanning top to bottom: nothing away from the edges,
// and faster the closer it holds to one.
func autoScrollStep(pointerY, top, bottom float32) float32 {
	if bottom-top <= 0 {
		return 0
	}
	var depth float32
	dir := float32(0)
	switch {
	case pointerY < top+lineAutoScrollBand:
		depth, dir = pointerY-top, -1
	case pointerY > bottom-lineAutoScrollBand:
		depth, dir = bottom-pointerY, 1
	default:
		return 0
	}
	if depth < 0 {
		depth = 0
	}
	if depth > lineAutoScrollBand {
		depth = lineAutoScrollBand
	}
	return dir * lineAutoScrollTopSpeed * (1 - depth/lineAutoScrollBand)
}

// trackLinePress runs the mouse half of the selection: a press arms a line,
// and while the button stays down, crossing another changed line extends the
// range from where the press started, as dragging in an editor does.
func (a *app) trackLinePress(c *ui.Context, index int, body ui.Element) {
	btn, pressed := body.PressedButton()
	if pressed && btn == 0 && body.PressedInside() {
		a.linePressIndex = index
		a.lineDragHeld = true
		a.lineDragging = false
		a.lineDragMoved = false
		// Ask for a frame on every move from here: without a tracked press
		// the engine only rebuilds when the pointer's hover changes.
		body.PointerPosition()
		return
	}
	if a.linePressIndex < 0 {
		return
	}
	// The engine delivers the moves of a press to the pressed row alone and
	// keeps the hover chain from the press's start, so Hovered tells nothing
	// about a row the pointer crosses. Every row asks the pointer where it
	// is instead; the pressed row doing it also asks for a frame on each
	// move, which is what lets the crossed rows rebuild during the drag.
	_, py, over := body.PointerPosition()
	if index == a.linePressIndex {
		// PressedButton speaks for the pressed row alone, the one the press
		// started on: it turns false there once the button lets go.
		if !pressed {
			a.lineReleasePress()
			return
		}
		a.lineDragHeld = true
		a.autoScrollLineDrag(c, body, py)
		return
	}
	if !a.lineDragHeld || !over {
		return
	}
	if !a.lineDragging {
		a.lineDragging = true
		a.lineAnchor = a.linePressIndex
		a.lineSel = map[int]bool{a.linePressIndex: true}
	}
	a.dragToLine(index)
}

// lineReleasePress forgets the press once the button let go.
func (a *app) lineReleasePress() {
	a.linePressIndex = -1
	a.lineDragHeld = false
	a.lineDragging = false
}

// autoScrollLineDrag scrolls the diff while a line drag holds near its top
// or bottom edge, so the selection can reach lines outside the view. It asks
// for the next frame itself: a still pointer sends no moves to rebuild on, and
// a timer is serviced too late to scroll smoothly.
func (a *app) autoScrollLineDrag(c *ui.Context, body ui.Element, py float32) {
	vp := a.diffViewportRect
	if vp.H <= 0 {
		return
	}
	pointerY := body.Bounds().Y + py
	step := autoScrollStep(pointerY, vp.Y, vp.Y+vp.H)
	if step == 0 || a.diffScroll.MaxY <= 0 {
		return
	}
	next := a.diffScroll.Y + step
	if next < 0 {
		next = 0
	}
	if next > a.diffScroll.MaxY {
		next = a.diffScroll.MaxY
	}
	if next == a.diffScroll.Y {
		return
	}
	a.diffScroll.Y = next
	c.AnimationFrame()
}

// dragToLine grows the drag selection to index without dropping the anchor.
func (a *app) dragToLine(index int) {
	if index < 0 || index >= len(a.diff.Lines) {
		return
	}
	switch a.diff.Lines[index].Kind {
	case "add", "del":
	default:
		return
	}
	if a.lineAnchor < 0 {
		a.lineAnchor = index
	}
	lo, hi := a.lineAnchor, index
	if lo > hi {
		lo, hi = hi, lo
	}
	next := make(map[int]bool, hi-lo+1)
	for i := lo; i <= hi; i++ {
		switch a.diff.Lines[i].Kind {
		case "add", "del":
			next[i] = true
		}
	}
	same := len(next) == len(a.lineSel)
	if same {
		for i := range next {
			if !a.lineSel[i] {
				same = false
				break
			}
		}
	}
	if same && a.lineCursor == index {
		return
	}
	a.lineSel = next
	a.lineCursor = index
	a.lineDragMoved = true
	a.invalidate()
}

// clickLine applies Finder-like selection rules: a plain click selects one
// line, Shift extends from the anchor over changed lines, and Cmd toggles.
func (a *app) clickLine(index int, mods ui.Modifiers) {
	diff := a.activeWorktreeDiff()
	if index < 0 || index >= len(diff.Lines) {
		return
	}
	if diff.Lines[index].Kind != "add" && diff.Lines[index].Kind != "del" {
		return
	}
	if a.lineSel == nil {
		a.lineSel = make(map[int]bool)
	}
	switch {
	case mods&ui.Shift != 0 && a.lineAnchor >= 0 && a.lineAnchor < len(diff.Lines):
		lo, hi := a.lineAnchor, index
		if lo > hi {
			lo, hi = hi, lo
		}
		next := make(map[int]bool, hi-lo+1)
		for i := lo; i <= hi; i++ {
			if diff.Lines[i].Kind == "add" || diff.Lines[i].Kind == "del" {
				next[i] = true
			}
		}
		a.lineSel = next
	case mods&ui.Cmd != 0 || mods&ui.Super != 0:
		if a.lineSel[index] {
			delete(a.lineSel, index)
		} else {
			a.lineSel[index] = true
		}
		a.lineAnchor = index
		if len(a.lineSel) == 0 {
			a.lineSel = nil
		}
	default:
		a.lineSel = map[int]bool{index: true}
		a.lineAnchor = index
	}
	a.lineCursor = index
	a.revealLine = false
	a.invalidate()
}

func (a *app) discardLinesAllowed() bool {
	if a.overviewLineTarget != nil {
		return false
	}
	if a.selectedStaged || !a.lineActionsAllowed() {
		return false
	}
	entry, ok := a.selectedEntry()
	return ok && entry.Worktree != 'D'
}

// applyLines stages or unstages every chosen line through one partial patch.
func (a *app) applyLines(indexes []int) {
	if a.overviewLineTarget != nil {
		a.applyOverviewLines(*a.overviewLineTarget, indexes)
		return
	}
	if !a.lineActionsAllowed() {
		return
	}
	choice := make([]int, 0, len(indexes))
	for _, index := range indexes {
		if index < 0 || index >= len(a.diff.Lines) {
			continue
		}
		switch a.diff.Lines[index].Kind {
		case "add", "del":
			choice = append(choice, index)
		}
	}
	if len(choice) == 0 {
		return
	}
	path, diff, staged := a.selectedPath, a.diff, a.selectedStaged
	label, success := a.t("stageLineFailed"), a.t("stageLineSuccess")
	if len(choice) > 1 {
		label, success = a.t("stageLinesFailed"), a.t("stageLinesSuccess")
	}
	if staged {
		label, success = a.t("unstageLineFailed"), a.t("unstageLineSuccess")
		if len(choice) > 1 {
			label, success = a.t("unstageLinesFailed"), a.t("unstageLinesSuccess")
		}
	}
	a.perform(label, success, func(ctx context.Context, repo *gitrepo.Repository) error {
		return repo.ApplyLines(ctx, path, diff, choice, staged)
	})
}

func (a *app) confirmDiscardLines(indexes []int) {
	if !a.discardLinesAllowed() || len(indexes) == 0 {
		return
	}
	path, diff := a.selectedPath, a.diff
	choice := append([]int(nil), indexes...)
	a.askConfirm(a.t("discardLinesTitle"), a.discardLinesMessage(len(choice)), a.t("discard"), true, func() {
		a.discardLines(path, diff, choice)
	})
}

// discardLines runs the worktree discard itself instead of going through
// perform so the applied patch can be kept for the undo action.
func (a *app) discardLines(path string, diff gitrepo.Diff, choice []int) {
	if a.repo == nil || a.busy || a.loading {
		return
	}
	repo := a.repo
	ctx, cancel := a.beginOperation("")
	go func() {
		defer cancel()
		patch, err := repo.DiscardLines(ctx, path, diff, choice)
		mygo.RunOnMain(func() {
			a.finishOperation()
			if operationCancelled(ctx, err) {
				a.queue("info", a.t("operationCancelled"), "")
			} else if err != nil {
				a.queue("error", a.t("discardLinesFailed"), err.Error())
			} else {
				a.undoDiscard = undoDiscardState{path: path, patch: patch}
				a.pushNotice(toastMain, &toastNotice{
					kind:     "success",
					title:    a.t("discardLinesSuccess"),
					action:   a.t("undo"),
					onAction: a.undoLastDiscard,
					timeout:  8 * time.Second,
				})
			}
			a.reload()
			a.invalidate()
		})
	}()
}

// undoLastDiscard reverses the most recent line discard. Git's context check
// refuses the undo if the file changed after the discard.
func (a *app) undoLastDiscard() {
	undo := a.undoDiscard
	a.undoDiscard = undoDiscardState{}
	if a.repo == nil || undo.patch == "" || a.busy || a.loading {
		return
	}
	repo := a.repo
	ctx, cancel := a.beginOperation("")
	go func() {
		defer cancel()
		err := repo.UndoDiscard(ctx, undo.path, undo.patch)
		mygo.RunOnMain(func() {
			a.finishOperation()
			if operationCancelled(ctx, err) {
				a.queue("info", a.t("operationCancelled"), "")
			} else if err != nil {
				a.queue("error", a.t("undoDiscardFailed"), err.Error())
			} else {
				a.queue("success", a.t("undoDiscardSuccess"), "")
			}
			a.reload()
			a.invalidate()
		})
	}()
}

// diffLinesText joins selected changed lines as the file has them, in row
// order, separated by newlines.
func diffLinesText(diff gitrepo.Diff, indexes []int) string {
	var b strings.Builder
	for _, index := range indexes {
		if index < 0 || index >= len(diff.Lines) {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(diffLineText(diff.Lines[index]))
	}
	return b.String()
}

// diffHunkText returns a hunk header and its body with the diff's prefixes
// kept, as the diff shows them.
func diffHunkText(diff gitrepo.Diff, hunk int) string {
	var b strings.Builder
	for _, line := range diff.Lines {
		if line.Hunk != hunk {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line.Text)
	}
	return b.String()
}

// hunkChangedLines lists the changed rows of a hunk, the set staging and
// discarding it apply to.
func hunkChangedLines(diff gitrepo.Diff, hunk int) []int {
	var out []int
	for i, line := range diff.Lines {
		if line.Hunk != hunk {
			continue
		}
		if line.Kind == "add" || line.Kind == "del" {
			out = append(out, i)
		}
	}
	return out
}

// sortedSelection lists the chosen row indexes in row order.
func sortedSelection(set map[int]bool) []int {
	out := make([]int, 0, len(set))
	for index := range set {
		out = append(out, index)
	}
	sort.Ints(out)
	return out
}

// lineIndexesWith lists the rows a selection may cover, in row order. The
// working tree selects added and removed lines; a committed diff also lets
// context lines be copied.
func lineIndexesWith(diff gitrepo.Diff, withContext bool) []int {
	var out []int
	for i, line := range diff.Lines {
		switch line.Kind {
		case "add", "del":
			out = append(out, i)
		case "context":
			if withContext {
				out = append(out, i)
			}
		}
	}
	return out
}

func changedLineIndexes(diff gitrepo.Diff) []int { return lineIndexesWith(diff, false) }

func commitLineIndexes(diff gitrepo.Diff) []int { return lineIndexesWith(diff, true) }

// selectionSet turns row indexes into a selection set; no rows leaves the
// set nil, which is how the app spells "nothing selected".
func selectionSet(indexes []int) map[int]bool {
	if len(indexes) == 0 {
		return nil
	}
	set := make(map[int]bool, len(indexes))
	for _, index := range indexes {
		set[index] = true
	}
	return set
}

// selectionEdges returns the first and last row of a selection, the anchor
// and cursor the keyboard extends from; no rows give -1, -1.
func selectionEdges(indexes []int) (anchor, cursor int) {
	if len(indexes) == 0 {
		return -1, -1
	}
	return min(indexes[0], indexes[len(indexes)-1]), max(indexes[0], indexes[len(indexes)-1])
}

// invertIndexes complements the selection over the rows it may cover.
func invertIndexes(candidates []int, selected map[int]bool) []int {
	var out []int
	for _, index := range candidates {
		if !selected[index] {
			out = append(out, index)
		}
	}
	return out
}

// selectWorkspaceLines replaces the working diff's selection with indexes.
func (a *app) selectWorkspaceLines(indexes []int) {
	if len(indexes) == 0 {
		a.clearLineSelection()
	} else {
		a.lineSel = selectionSet(indexes)
		a.lineAnchor, a.lineCursor = selectionEdges(indexes)
		a.revealLine = false
	}
	a.invalidate()
}

// invertWorkspaceLines selects the changed lines the current selection
// leaves out, as the diff's context menu offers.
func (a *app) invertWorkspaceLines() {
	a.selectWorkspaceLines(invertIndexes(changedLineIndexes(a.diff), a.lineSel))
}

// selectCommitLines replaces the committed diff's selection with indexes.
func (a *app) selectCommitLines(indexes []int) {
	visible := selectionSet(a.commitLineIndices())
	filtered := make([]int, 0, len(indexes))
	for _, index := range indexes {
		if visible[index] {
			filtered = append(filtered, index)
		}
	}
	indexes = filtered
	if len(indexes) == 0 {
		a.clearCommitSelection()
	} else {
		a.commitSel = selectionSet(indexes)
		a.commitAnchor, a.commitCursor = selectionEdges(indexes)
		a.commitReveal = false
	}
	a.invalidate()
}

// invertCommitLines selects the committed rows the current selection leaves
// out; context rows count here because they can be copied.
func (a *app) invertCommitLines() {
	a.selectCommitLines(invertIndexes(a.commitLineIndices(), a.commitSel))
}

func (a *app) lineMenu(c *ui.Context, row ui.Element, index int) {
	if !a.lineActionsAllowed() {
		return
	}
	// Only changed rows can be staged or discarded; a context or header row has
	// no menu of its own.
	if index < 0 || index >= len(a.diff.Lines) {
		return
	}
	switch a.diff.Lines[index].Kind {
	case "add", "del":
	default:
		return
	}
	targets := a.lineTargets(index)
	if len(targets) == 0 {
		return
	}
	label, icon := a.t("stageLine"), lucide.Plus()
	if a.selectedStaged {
		label, icon = a.t("unstageLine"), lucide.Minus()
	}
	if len(targets) > 1 {
		label = a.tf("stageLines", len(targets))
		if a.selectedStaged {
			label = a.tf("unstageLines", len(targets))
		}
	}
	items := []overlay.MenuItem{{Key: "apply", Label: label, Icon: icon}}
	if a.discardLinesAllowed() {
		discard := a.t("discardLine")
		if len(targets) > 1 {
			discard = a.tf("discardLines", len(targets))
		}
		items = append(items,
			overlay.MenuItem{Divider: true},
			overlay.MenuItem{Key: "discard", Label: discard, Icon: lucide.Undo2(), Danger: true},
		)
	}
	items = append(items,
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{Key: "copy", Label: a.t("copyLine"), Icon: lucide.Copy()},
	)
	if len(targets) > 1 {
		items[len(items)-1].Label = a.copyLinesLabel(len(targets))
	}
	if len(changedLineIndexes(a.diff)) > 1 {
		items = append(items,
			overlay.MenuItem{Divider: true},
			overlay.MenuItem{Key: "select-all", Label: a.t("selectAllLines"), Icon: lucide.ListChecks()},
		)
		if len(a.lineSel) > 0 {
			items = append(items,
				overlay.MenuItem{Key: "invert", Label: a.t("invertLines"), Icon: lucide.FlipHorizontal()},
			)
		}
	}
	switch overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: items}) {
	case "apply":
		a.applyLines(targets)
	case "discard":
		a.confirmDiscardLines(targets)
	case "copy":
		a.copyText(diffLinesText(a.diff, targets))
	case "select-all":
		a.selectWorkspaceLines(changedLineIndexes(a.diff))
	case "invert":
		a.invertWorkspaceLines()
	}
}
