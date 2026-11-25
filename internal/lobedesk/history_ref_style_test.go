package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

// A pill's label must sit where an ordinary centered text sits. The old
// rich-text box carried its own taller line metrics and drew visibly low.
func TestHistoryRefLabelsSitOnThePillMidline(t *testing.T) {
	inkRows := func(view *ui.Tester, box ui.Rect, ground ui.Color) (int, int) {
		img := view.Image()
		// Skip the border and the stripe's accent bar; the ink threshold
		// keeps the faint capsule edge out of the measurement.
		x0, x1 := int((box.X+scaledFont(10))*2), int((box.X+box.W-2)*2)
		y0, y1 := int(box.Y*2), int((box.Y+box.H)*2)
		top, bottom := -1, -1
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				p := img.RGBAAt(x, y)
				if contrastRatio(ui.RGB(p.R, p.G, p.B), ground) > 3 {
					if top < 0 {
						top = y
					}
					bottom = y
					break
				}
			}
		}
		return top, bottom
	}
	for _, style := range []string{"segmented", "stripe"} {
		t.Run(style, func(t *testing.T) {
			const label = "History"
			a := &app{historyRefStyle: style}
			var pill, reference ui.Rect
			var ground ui.Color
			view := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				tk := theme.TokensOf(c)
				ground = tk.BgLayout
				ui.Column(c).Gap(8).Children(func() {
					ui.Row(c).FillWidth().Children(func() {
						pill = a.historyNamedRefPill(c, label, nil, false, tk.Blue, ground).Bounds()
					})
					ui.Row(c).FillWidth().Children(func() {
						el := ui.Row(c).Height(scaledFont(20)).Width(a.historyNamedRefWidth(c, label, false, false)).
							Radius(scaledFont(5)).AlignItems(ui.Center).Border(1, tk.BorderSecondary)
						el.Children(func() {
							ui.Text(c, label).FontSize(scaledFont(11)).FontWeight(500).TextColor(tk.Text).SingleLine()
						})
						reference = el.Bounds()
					})
				})
			}, 240, 80)
			view.SetScale(2)
			view.Frame()
			top, bottom := inkRows(view, pill, ground)
			refTop, refBottom := inkRows(view, reference, ground)
			if top < 0 || refTop < 0 {
				t.Fatal("no label ink rendered")
			}
			// The reference row has the same height and border, so the label
			// rows line up one-for-one when the pill's text is centered.
			pillTop, pillBottom := top-int(pill.Y*2), bottom-int(pill.Y*2)
			refTop, refBottom = refTop-int(reference.Y*2), refBottom-int(reference.Y*2)
			if pillTop-refTop < -1 || pillTop-refTop > 1 || pillBottom-refBottom < -1 || pillBottom-refBottom > 1 {
				t.Errorf("%s: label ink sits %d..%d in the pill but %d..%d in a plain row of the same height", style, pillTop, pillBottom, refTop, refBottom)
			}
		})
	}
}

// The capsule grows with the ref name as long as the row's ref budget allows:
// no fixed point cap and no rune cap may shorten a name that would fit.
func TestHistoryRefLabelUsesTheAvailableWidth(t *testing.T) {
	const label = "feature/history/reference-layout"
	old := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(old) })
	for _, style := range []string{"segmented", "stripe"} {
		for _, scale := range []float32{1, 1.15} {
			setUIFontScale(scale)
			t.Run(style, func(t *testing.T) {
				a := &app{historyRefStyle: style}
				var got string
				var full float32
				view := ui.NewTester(func(c *ui.Context) {
					a.applyTheme(c)
					layout := a.fitHistoryRefs(c, []string{label}, label, 900, 500, 8, true)
					full = a.historyNamedRefWidth(c, label, true, false)
					if len(layout.shown) != 1 {
						t.Errorf("%s: one ref showed %d capsules", style, len(layout.shown))
						return
					}
					got = a.historyNamedRefLabel(c, label, true, false, layout.maxWidth)
				}, 900, 100)
				view.Frame()
				if got != label {
					t.Errorf("%s at scale %v: a %.0f-point label shows %q in a 500-point budget (needs %.0f); the row must not cap its width", style, scale, full, got, full)
				}
			})
		}
	}
}

func TestHistoryStyledLabelsExposeHeadAndStash(t *testing.T) {
	for _, style := range []string{"segmented", "stripe"} {
		a := &app{historyRefStyle: style, hideAuthor: true, hideTime: true, historyDetailsHidden: true}
		commit := gitrepo.Commit{SHA: "a", Subject: "Commit", Refs: "HEAD -> main, origin/main, tag: v1.0"}
		stash := gitrepo.Stash{SHA: "s", Ref: "stash@{12}", Message: "WIP: sample"}
		view := ui.NewTester(func(c *ui.Context) {
			a.applyTheme(c)
			a.historySplit.rowWidth, _ = c.Size()
			ui.Column(c).FillWidth().Children(func() { a.commitRow(c, commit, 0, 0, 2, nil); a.stashRow(c, stash, 1, 2) })
		}, 600, 140)
		for _, label := range []string{"HEAD", "main", "stash@{12}"} {
			if !view.HasText(label) {
				t.Errorf("%s: missing %s", style, label)
			}
		}
	}
}

func TestHistoryStyledLabelsStayWithinRows(t *testing.T) {
	old := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(old) })
	commit := gitrepo.Commit{SHA: "a", Subject: "A long title for a narrow history", Author: "Someone", Date: "2026-11-25T10:00:00Z", Refs: "HEAD -> main, feature/历史/分支名称很长, origin/feature/history-layout, tag: v1.2, tag: latest"}
	for _, style := range []string{"segmented", "stripe"} {
		for _, scale := range []float32{1, 1.15} {
			for _, compact := range []bool{false, true} {
				setUIFontScale(scale)
				a := &app{historyRefStyle: style, hideAuthor: compact, hideTime: compact, historyDetailsHidden: true}
				view := ui.NewTester(func(c *ui.Context) {
					a.applyTheme(c)
					a.historySplit.rowWidth, _ = c.Size()
					a.commitRow(c, commit, 0, 0, 1, nil)
				}, 600, 120)
				for _, width := range []int{360, 600, 900, 220} {
					view.SetSize(width, 120)
					view.Frame()
					view.Frame()
					row, ok := view.AccessByRole(ui.RoleOption, commit.Subject)
					if !ok {
						t.Fatal("row missing")
					}
					for _, n := range view.AccessAll() {
						if n.Role == ui.RoleText && !n.Offscreen && n.Bounds.Y >= row.Bounds.Y && n.Bounds.Y+n.Bounds.H <= row.Bounds.Y+row.Bounds.H {
							if n.Bounds.X < row.Bounds.X-0.5 || n.Bounds.X+n.Bounds.W > row.Bounds.X+row.Bounds.W+0.5 {
								t.Errorf("%s scale %v compact %v width %d: %s outside row: %+v", style, scale, compact, width, n.Name, n.Bounds)
							}
						}
					}
				}
			}
		}
	}
}
