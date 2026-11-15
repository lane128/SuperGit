package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// The ref selector and the checkout-branch picker are the same control: a
// search box that keeps the keyboard, a highlighted row the arrows move, Enter
// taking the highlight (or the first usable row), and a capped list that
// scrolls the highlight into view once when it moves. The pieces below are
// what the two popups share; what differs — the rows themselves and which
// rows can be taken — stays with each popup.

const pickerListMaxHeight = 320

// stepPickerCursor moves the highlight inside size rows, wrapping at either
// end so a short list never traps the keys. selectable rejects rows that exist
// but cannot be chosen; nil accepts every row.
func stepPickerCursor(size, cursor, delta int, selectable func(int) bool) int {
	if size == 0 || delta == 0 {
		return -1
	}
	index := cursor
	if index < 0 || index >= size {
		// No highlight yet: Down starts at the top, Up at the bottom.
		if delta > 0 {
			index = -1
		} else {
			index = size
		}
	}
	for i := 0; i < size; i++ {
		index = (index + delta + size) % size
		if selectable == nil || selectable(index) {
			return index
		}
	}
	return -1
}

// pickerCursorAt is the row Enter takes: the highlighted one when it can be
// taken, and otherwise the first row that can. -1 means the list offers no
// choice, so Enter has nothing to do.
func pickerCursorAt(size, cursor int, selectable func(int) bool) int {
	take := func(index int) bool { return selectable == nil || selectable(index) }
	if cursor >= 0 && cursor < size && take(cursor) {
		return cursor
	}
	for index := 0; index < size; index++ {
		if take(index) {
			return index
		}
	}
	return -1
}

// placePicker puts the highlight where target says when the popup opens or the
// query changes, and reports whether the caller should scroll to the row.
func placePicker(cursor *int, placedQuery *string, placed *bool, query string, target func() int) bool {
	if *placed && *placedQuery == query {
		return false
	}
	*placedQuery, *placed = query, true
	*cursor = target()
	return true
}

// repairPicker pulls a highlight a refresh left outside the list back onto
// target, and reports whether the caller should scroll to it. Enter must never
// act on a row the reader cannot see.
func repairPicker(cursor *int, size int, target func() int) bool {
	if *cursor >= 0 && *cursor < size {
		return false
	}
	if size == 0 {
		*cursor = -1
		return false
	}
	*cursor = target()
	return *cursor >= 0
}

// pickerSearchBar is the search box shared by the pickers. Enter takes the
// highlighted row through enter, and Up/Down move the highlight through step;
// the box keeps the focus while that happens so the reader can type, step and
// type again.
func (a *app) pickerSearchBar(c *ui.Context, query *string, placeholder string, enter func(), step func(delta int)) ui.Element {
	search := field.SearchBar(c, query, field.SearchBarProps{
		Placeholder: placeholder,
		Size:        theme.Small,
		Variant:     field.InputFilled,
		OnSearch:    func(string) { enter() },
	})
	search.OnShortcut(0, ui.KeyDown, func() {
		step(1)
		a.invalidate()
	})
	search.OnShortcut(0, ui.KeyUp, func() {
		step(-1)
		a.invalidate()
	})
	return search
}

// pickerRowBase is the row shell both pickers draw: one option of the list,
// with the height, padding and pointer the two share. The caller paints the
// chosen and highlighted states on top.
func pickerRowBase(c *ui.Context, label string, pos, size int, selected bool) ui.Element {
	tk := theme.TokensOf(c)
	return ui.ButtonBase(c).Label(label).Row().AlignSelf(ui.Stretch).
		MinHeight(30).Padding(5, 8).Gap(8).AlignItems(ui.Center).
		Role(ui.RoleOption).PositionInSet(pos, size).AccessSelected(selected).
		Radius(tk.BorderRadiusSM).Cursor(ui.CursorPointer)
}

// pickerRows lays the picker's rows out in the capped scroll area both popups
// use; a list longer than the cap scrolls instead of pushing the dialog past
// the window.
func pickerRows(c *ui.Context, rows func()) {
	layout.ScrollArea(c, layout.ScrollAreaProps{}, func() {
		ui.Column(c).FillWidth().Gap(1).Role(ui.RoleList).Children(rows)
	}).MaxHeight(pickerListMaxHeight)
}

// pickerReveal scrolls the highlighted row into view once when it moves. A
// scroll every frame would pull the list back from the reader's wheel.
func pickerReveal(row ui.Element, highlighted bool, reveal *bool) {
	if highlighted && *reveal {
		row.ScrollIntoView()
		*reveal = false
	}
}
