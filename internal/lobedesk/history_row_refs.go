package lobedesk

import (
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func historyRefPillWidth(c *ui.Context, label string, icon bool, gap float32) float32 {
	w, _ := c.MeasureText(0, ui.Span{Text: label, Size: scaledFont(11), Weight: 600})
	w += 2*refPillPadding + 2 // border
	if icon {
		w += scaledFont(11) + refPillGap
	}
	return w + gap
}

// Shorten the actual text before painting so the label keeps the end of the
// name instead of relying on the text control's trailing ellipsis.
func measuredRefPillLabel(c *ui.Context, label string, icon bool, limit float32, maxRunes int) string {
	for runes := min(utf8.RuneCountInString(label), maxRunes); runes > 7; runes-- {
		candidate := elideMiddle(label, runes)
		if historyRefPillWidth(c, candidate, icon, 0) <= limit {
			return candidate
		}
	}
	return elideMiddle(label, 7)
}

type historyRefLayout struct {
	shown, folded []string
	maxWidth      float32
	compactHead   bool
}

// fitHistoryRefs tries the existing ref priority order at each slot count.
// Each pill costs its measured width and one gap from the preceding element.
func (a *app) fitHistoryRefs(c *ui.Context, refs []string, head string, listWidth, budget, gap float32, metadata bool) historyRefLayout {
	// The ref budget is the only width limit: it already leaves the author
	// and the date their room, so one long name may use every point left.
	layout := historyRefLayout{maxWidth: budget, compactHead: true}
	slots := min(len(refs), refSlotsForWidth(listWidth))
	if listWidth <= 0 {
		if metadata {
			slots = 1 // Use the compact layout until the list has been measured.
		}
		// The first frame has no column width, so its budget says nothing.
		// Lay the capsule out at the old provisional width; the next frame
		// measures against the real budget.
		layout.maxWidth = max(budget, refCapsuleWidth(refCapsuleRowWidth))
		layout.shown, layout.folded = rowRefsLimit(refs, a.historyRef, head, slots)
		layout.compactHead = metadata
		return layout
	}
	for ; slots > 0; slots-- {
		shown, folded := rowRefsLimit(refs, a.historyRef, head, slots)
		var overflow float32
		if len(folded) > 0 {
			overflow = historyRefPillWidth(c, "+"+itoa(len(folded)), false, gap)
		}
		width, readable := overflow, true
		for _, ref := range shown {
			badge := a.refBadge(ref)
			icon := badge.Icon != nil
			minimum := a.historyNamedRefWidth(c, historyRefCandidate(badge.Label, 7), icon, ref == head)
			readable = readable && minimum <= layout.maxWidth
			// Fit the name as it can be drawn; a shortened estimate would let
			// two long names each pass the budget and overflow the row.
			width += a.historyNamedRefWidth(c, badge.Label, icon, ref == head) + gap
		}
		if readable && width <= budget {
			layout.shown, layout.folded = shown, folded
			layout.compactHead = true
			return layout
		}
		if slots == 1 {
			// A squeezed pill needs room for a recognizable name. Short
			// names must fit whole, and long ones need space for both ends.
			ref := shown[0]
			badge := a.refBadge(ref)
			width := budget - overflow - gap
			minimum := a.historyNamedRefWidth(c, historyRefCandidate(badge.Label, 7), badge.Icon != nil, ref == head)
			if width >= minimum {
				layout.shown, layout.folded = shown, folded
				layout.maxWidth = width
				return layout
			}
		}
	}
	layout.folded = refs
	return layout
}

func (a *app) commitRowRefs(c *ui.Context, commit gitrepo.Commit, graph *commitGraph, index int, rowBg ui.Color, head string, layout historyRefLayout) {
	tk := theme.TokensOf(c)
	for _, ref := range layout.shown {
		badge := a.refBadge(ref)
		// A ref can change position as more pills fit. Keep its open menu
		// attached to the ref rather than to that position.
		key := "history-ref:" + ref
		label := a.historyNamedRefLabel(c, badge.Label, badge.Icon != nil, ref == head, layout.maxWidth)
		kind, name, filterable := a.parseCommitRef(ref)
		color := tk.TextSecondary
		if filterable {
			color = refBadgeColor(a.lanePalette(tk), graph, index, kind, name)
		} else if family, ok := refColor(tk, badge.Color); ok {
			color = family
		}
		pill := a.historyNamedRefPill(c.Key(key), label, badge.Icon, ref == head, color, rowBg).MaxWidth(layout.maxWidth).Tooltip(badge.Label)
		if ref == head {
			pill.Tooltip(a.tf("headBadgeTip", head))
		}
		a.refBadgeMenu(c, pill, ref, commit.SHA)
	}
	if len(layout.folded) > 0 {
		labels := make([]string, 0, len(layout.folded))
		for _, ref := range layout.folded {
			labels = append(labels, a.refBadge(ref).Label)
		}
		// Dismiss the overflow menu when resizing changes the set it lists.
		key := "history-ref-overflow:" + strings.Join(layout.folded, "\x00")
		pill := refPill(c.Key(key), neutralCapsule(tk, rowBg), nil, "+"+itoa(len(layout.folded))).Tooltip(strings.Join(labels, "\n"))
		a.foldedRefsMenu(c, pill, layout.folded)
	}
}
