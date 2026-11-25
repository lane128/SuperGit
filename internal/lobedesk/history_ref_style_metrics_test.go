package lobedesk

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestHistoryRefTextContrastOnRowStates(t *testing.T) {
	for _, palette := range appThemes {
		for _, dark := range []bool{false, true} {
			cfg := themeConfig(palette.key)
			tk := cfg.Tokens(dark)
			for _, ground := range []ui.Color{graphNodeGround(tk, false, false), graphNodeGround(tk, false, true), graphNodeGround(tk, true, false)} {
				for _, accent := range []ui.Color{tk.Blue, tk.Gold, tk.Purple, tk.TextSecondary} {
					for _, head := range []bool{false, true} {
						colors := historyRefColorsFor(tk, accent, ground, head)
						for _, pair := range [][2]ui.Color{{colors.ink, colors.surface}, {colors.muted, colors.surface}, {colors.mark, colors.segment}, {colors.mark, colors.surface}} {
							if ratio := contrastRatio(pair[0], pair[1]); ratio < 4.5 {
								t.Errorf("palette=%s dark=%v head=%v contrast %.2f", palette.key, dark, head, ratio)
							}
						}
					}
				}
			}
		}
	}
}

func TestHistoryRefMeasurementsMatchRenderedText(t *testing.T) {
	old := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(old) })
	for _, style := range []string{"segmented", "stripe"} {
		for _, scale := range []float32{.9, 1, 1.15} {
			for _, head := range []bool{false, true} {
				setUIFontScale(scale)
				a := &app{historyRefStyle: style}
				for _, full := range []string{"main", "origin/feature/history/reference-layout", "feature/历史/修复键盘焦点", "very-long-unbroken-reference-name-01234567890123456789"} {
					var label string
					view := ui.NewTester(func(c *ui.Context) {
						a.applyTheme(c)
						tk := theme.TokensOf(c)
						label = a.historyNamedRefLabel(c, full, true, head, 150)
						a.historyNamedRefPill(c, label, lucide.GitBranch(), head, tk.Blue, tk.BgContainer).Role(ui.RoleGroup).Label("Reference")
					}, 400, 70)
					pill, _ := view.AccessByRole(ui.RoleGroup, "Reference")
					// The label renders as a muted prefix and an ink leaf so the
					// last path component keeps the emphasis; both are text nodes.
					prefix, leaf := "", label
					if i := strings.LastIndex(label, "/"); i >= 0 {
						prefix, leaf = label[:i+1], label[i+1:]
					}
					node, ok := view.AccessByRole(ui.RoleText, leaf)
					if !ok || label == "" {
						t.Fatalf("missing label %q", label)
					}
					if prefix != "" {
						if _, ok := view.AccessByRole(ui.RoleText, prefix); !ok {
							t.Fatalf("missing muted prefix %q of %q", prefix, label)
						}
					}
					if node.Bounds.X+node.Bounds.W > pill.Bounds.X+pill.Bounds.W-4*scale || node.Bounds.Y+node.Bounds.H > pill.Bounds.Y+pill.Bounds.H {
						t.Errorf("%s scale=%v head=%v text clipped %q: text=%+v pill=%+v", style, scale, head, label, node.Bounds, pill.Bounds)
					}
					if pill.Bounds.W > 150.1 {
						t.Errorf("%s label exceeded fitting budget: %v", style, pill.Bounds)
					}
				}
			}
		}
	}
	if got := historyRefCandidate("origin/feature/search", 12); !strings.HasSuffix(got, "search") || utf8.RuneCountInString(got) > 12 {
		t.Fatalf("lost last component: %q", got)
	}
}
