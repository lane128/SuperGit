package lobedesk

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

// A historical patch has one combined line index. Working overview entries
// each start at zero, including the two staged versions of the same path.
type diffFindLine struct {
	file  detailFileKey
	index int
}

type diffFindMatch struct {
	line       diffFindLine
	start, end int // byte offsets in the displayed text
}

type diffFindSection struct {
	file    detailFileKey
	lines   []gitrepo.DiffLine
	visible []int
	split   bool
}

type diffFindExtent struct {
	line        *gitrepo.DiffLine
	end         int
	font        string
	size, width float32
}

type diffFindState struct {
	context             detailContextKey
	tab                 detailTabKey
	query, indexedQuery string
	sections            []diffFindSection
	matches             []diffFindMatch
	byLine              map[diffFindLine]int // first match in the line
	active              int
	focus, reveal       bool
	revealSeq           uint64
	extent              diffFindExtent
}

func (a *app) canFindInDiff() bool {
	return a.detailPaneVisible() && (a.worktreeSelected || a.currentDetailContext().CommitSHA != "" && !a.detailsDiffHidden) &&
		!a.historyRefOpen && !a.repositoryChooserOpen && !a.conflictEditorTakesThePane() && !a.hasPendingDialog(toastMain)
}

func (a *app) discardStaleDiffFind() {
	if f := a.diffFind; f != nil && (!a.canFindInDiff() || f.context != a.currentDetailContext() || f.tab != a.detailTabs().Active) {
		a.diffFind = nil
	}
}

func (a *app) openDiffFind() {
	if !a.canFindInDiff() {
		return
	}
	a.discardStaleDiffFind()
	if a.diffFind == nil {
		a.diffFind = &diffFindState{context: a.currentDetailContext(), tab: a.detailTabs().Active}
	}
	a.diffFind.focus = true
	a.invalidate()
}

func (a *app) closeDiffFind() {
	if a.diffFind == nil {
		return
	}
	a.diffFind = nil
	a.diffFindPane.Focus()
	a.invalidate()
}

// Describe expanded, loaded patch sections only. The renderers own their
// original slices; a replacement Git read supplies new immutable line data.
func (a *app) diffFindSections() []diffFindSection {
	var out []diffFindSection
	if !a.worktreeSelected {
		if a.commitDiffLoading || a.commitDiffErr != "" {
			return nil
		}
		for _, s := range a.visibleCommitSections() {
			if !a.commitCollapsed[s.Path] && len(s.Rows) > 0 {
				out = append(out, diffFindSection{lines: a.commitDiff.Lines, visible: s.Rows, split: a.splitDiff})
			}
		}
		return out
	}
	if !a.detailTabs().Active.Summary {
		if a.diffLoading || a.diffErr != "" || len(a.diff.Lines) == 0 {
			return nil
		}
		_, conflict := a.conflictPreviewEntry(a.selectedPath)
		return []diffFindSection{{lines: a.diff.Lines, visible: worktreeVisiblePatchRows(a.diff.Lines), split: a.splitDiff && !conflict}}
	}
	conflicts := make(map[string]bool)
	if a.splitDiff {
		for _, e := range a.status.Entries {
			if e.Conflict {
				conflicts[e.Path] = true
			}
		}
	}
	for _, e := range a.overviewEntries {
		if !e.Loading && e.Err == "" && !a.detailContexts.Worktree.Collapsed[e.Key] && len(e.Diff.Lines) > 0 {
			out = append(out, diffFindSection{file: e.Key, lines: e.Diff.Lines, visible: worktreeVisiblePatchRows(e.Diff.Lines), split: a.splitDiff && !conflicts[e.Key.Path]})
		}
	}
	return out
}

func sameFindSection(x, y diffFindSection) bool {
	return x.file == y.file && x.split == y.split && len(x.lines) == len(y.lines) && &x.lines[0] == &y.lines[0] &&
		slices.Equal(x.visible, y.visible)
}

func (f *diffFindState) update(sections []diffFindSection) {
	sameQuery := f.indexedQuery == f.query
	if sameQuery && slices.EqualFunc(f.sections, sections, sameFindSection) {
		return
	}
	var previous diffFindMatch
	if len(f.matches) > 0 {
		previous = f.matches[f.active]
	}
	f.sections, f.indexedQuery = sections, f.query
	f.matches, f.byLine, f.active, f.reveal = nil, nil, 0, false
	if f.query == "" {
		return
	}
	pattern, err := regexp.Compile("(?i)" + regexp.QuoteMeta(f.query))
	if err != nil {
		return
	}
	f.byLine = make(map[diffFindLine]int)
	for _, s := range sections {
		rows := newPatchRows(s.lines, s.visible, s.split)
		for i := range rows.len() {
			r := rows.at(i)
			for slot, index := range [...]int{r.Shared, r.Left, r.Right} {
				if index < 0 || slot == 2 && index == r.Left {
					continue
				}
				line := s.lines[index]
				offsets := pattern.FindAllStringIndex(diffFindText(line), -1)
				if len(offsets) == 0 {
					continue
				}
				id := diffFindLine{s.file, index}
				f.byLine[id] = len(f.matches)
				for _, pair := range offsets {
					f.matches = append(f.matches, diffFindMatch{id, pair[0], pair[1]})
				}
			}
		}
	}
	if sameQuery {
		if at := slices.Index(f.matches, previous); at >= 0 {
			f.active = at
			return
		}
	}
	f.reveal = len(f.matches) > 0
	f.revealSeq++
}

func diffFindText(line gitrepo.DiffLine) string {
	if line.Kind == "hunk" {
		return line.Text
	}
	return strings.ReplaceAll(line.Text, "\t", "    ")
}

func (f *diffFindState) move(delta int) {
	if n := len(f.matches); n > 0 {
		f.active = (f.active + n + delta) % n
		f.reveal = true
		f.revealSeq++
	}
}

// Working patches normally cap the horizontal canvas. Extend just the
// active section when its result lies beyond that limit, including the
// divider and both equal-width columns in a split replacement.
func (f *diffFindState) fitMatch(c *ui.Context, file detailFileKey, lines []gitrepo.DiffLine, widths patchWidths, size float32) patchWidths {
	if f == nil || len(f.matches) == 0 {
		return widths
	}
	m := f.matches[f.active]
	if m.line.file != file || m.line.index >= len(lines) {
		return widths
	}
	line := &lines[m.line.index]
	if line.Kind == "hunk" {
		size = max(size, scaledFont(10.5))
	}
	font := codeFontFamily()
	if f.extent.line != line || f.extent.end != m.end || f.extent.font != font || f.extent.size != size {
		text := diffFindText(*line)
		if m.end > len(text) {
			return widths
		}
		width, _ := c.MeasureText(0, ui.Span{Text: text[:m.end], Font: font, Size: size})
		f.extent = diffFindExtent{line: line, end: m.end, font: font, size: size, width: width}
	}
	if widths.columns > 0 && splitPatchCodeLine(line.Kind) {
		widths.columns = max(widths.columns, 2*(f.extent.width+splitPatchReserve))
		widths.canvas = max(widths.canvas, widths.columns)
	} else {
		widths.canvas = max(widths.canvas, f.extent.width+diffCodeReserve)
	}
	return widths
}

// Ask both levels of virtualization to materialize the active occurrence.
// Scroll the inline word itself after layout, including horizontal ancestors.
func (f *diffFindState) revealLine(file detailFileKey) int {
	if f != nil && f.reveal {
		if line := f.matches[f.active].line; line.file == file {
			return line.index
		}
	}
	return -1
}

func (f *diffFindState) lineText(c *ui.Context, id diffFindLine, text string, size float32, mirrored ...bool) ui.Element {
	font := codeFontFamily()
	drawPlain := func() ui.Element { return ui.Text(c, text).Font(font).FontSize(size).SingleLine() }
	if f == nil {
		return drawPlain()
	}
	first, ok := f.byLine[id]
	if !ok {
		return drawPlain()
	}
	tk := theme.TokensOf(c)
	var spans []ui.Span
	end := 0
	for i := first; i < len(f.matches) && f.matches[i].line == id; i++ {
		m := f.matches[i]
		// A header action can change the displayed patch mid-build.
		if m.end > len(text) {
			return drawPlain()
		}
		if end < m.start {
			spans = append(spans, ui.Span{Text: text[end:m.start]})
		}
		bg, ink := diffSearchColors(tk.Dark, i == f.active)
		span := ui.Span{Text: text[m.start:m.end], Background: bg, Color: ink, Underline: i == f.active}
		spans = append(spans, span)
		end = m.end
	}
	if end < len(text) {
		spans = append(spans, ui.Span{Text: text[end:]})
	}
	var paragraph ui.Element
	ui.Row(c.Key("matched-line")).Shrink(0).Children(func() {
		paragraph = ui.RichText(c, spans...).Font(font).FontSize(size).SingleLine()
		if !f.reveal || f.matches[f.active].line != id || len(mirrored) > 0 && mirrored[0] {
			return
		}
		m := f.matches[f.active]
		// MyGo reveals an inline span's entire paragraph. Give the glyphs a
		// layout anchor so nested split and overview scrollers can expose a
		// word at either end of a long line.
		measure := func(part string) float32 {
			w, _ := c.MeasureText(0, ui.Span{Text: part, Font: font, Size: size})
			return w
		}
		anchor := ui.Box(c.Key("find-reveal").Key(f.revealSeq)).Absolute().Left(measure(text[:m.start])).Top(0).
			Width(measure(text[m.start:m.end]) + 2).Height(size).AccessHidden(true)
		if anchor.Bounds().W > 0 {
			f.reveal = false
		} else {
			// A click can rebuild the tree several times before layout. Keep
			// both the section and its row materialized until the anchor was
			// laid out at least once.
			anchor.ScrollIntoView(ui.ScrollIntoViewOptions{Block: ui.ScrollCenter})
			c.After(0)
		}
	})
	return paragraph
}

func (a *app) wireDiffFind(c *ui.Context, pane ui.Element) ui.Element {
	pane.TabStop(false).Bind(&a.diffFindPane)
	a.discardStaleDiffFind()
	if a.canFindInDiff() && (!c.FocusTakesText() || a.diffFind != nil && pane.FocusWithin()) {
		pane.OnShortcut(ui.Cmd, ui.KeyF, a.openDiffFind)
	}
	if a.diffFind != nil {
		pane.OnShortcut(0, ui.KeyEscape, a.closeDiffFind)
	}
	return pane
}

func (a *app) drawDiffFindBar(c *ui.Context, f *diffFindState, sections func() []diffFindSection, closeFind func()) {
	if f == nil {
		return
	}
	tk := theme.TokensOf(c)
	bar := ui.Row(c).FillWidth().Shrink(0).Height(scaledFont(38)).Padding(0, 8).Gap(5).AlignItems(ui.Center).
		Background(tk.BgContainer).BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Role(ui.RoleSearch).Label(a.t("findInDiff"))
	bar.OnShortcut(0, ui.KeyEscape, closeFind)
	bar.Children(func() {
		ui.Icon(c, lucide.Search()).FontSize(scaledFont(14)).TextColor(tk.TextSecondary)
		var input ui.Element
		ui.Box(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
			input = field.Input(c, &f.query, field.InputProps{Variant: field.InputFilled, Size: theme.Small, Placeholder: a.t("findInDiff")}).
				FillWidth().Label(a.t("findInDiff"))
		})
		input.OnTextKey(func(k ui.TextKey) bool {
			switch {
			case k.Mods == 0 && k.Key == ui.KeyEscape:
				closeFind()
			case k.Key == ui.KeyEnter && (k.Mods == 0 || k.Mods == ui.Shift):
				f.query = k.Text()
				f.update(sections())
				delta := 1
				if k.Mods == ui.Shift {
					delta = -1
				}
				f.move(delta)
				a.invalidate()
			default:
				return f == a.diffFind && a.fileTabTextKey(k)
			}
			return true
		})
		if f.focus {
			input.Focus().SetTextSelection(0, utf8.RuneCountInString(f.query))
			f.focus = false
		}
		f.update(sections())
		n := len(f.matches)
		active := 0
		if n > 0 {
			active = f.active + 1
		}
		ui.Text(c, fmt.Sprintf("%d/%d", active, n)).Font(a.patchFontFamily()).FontSize(scaledFont(10)).TextColor(tk.TextSecondary).Shrink(0)
		for _, b := range []struct {
			icon  *ui.SVG
			label string
			delta int
		}{
			{lucide.ChevronUp(), "previousDiffMatch", -1}, {lucide.ChevronDown(), "nextDiffMatch", 1},
		} {
			if workbenchIconButton(c, b.icon, a.t(b.label), buttonNormal, n == 0).Shrink(0).Clicked() {
				f.move(b.delta)
				input.Focus()
				a.invalidate()
			}
		}
		if workbenchIconButton(c, lucide.X(), a.t("closeDiffFind"), buttonNormal, false).Shrink(0).Clicked() {
			closeFind()
		}
	})
}
