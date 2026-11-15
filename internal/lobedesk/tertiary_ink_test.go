package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The sidebar's group counts, "more" hints and per-row shortcut keys are
// 10-point tertiary text. Catppuccin Latte draws TextTertiary at 4.37:1 on the
// base surface and 3.65:1 on the selected-row tint, both below AA; the helper
// has to darken it on every ground a row can sit on, and leave dark flavors
// alone.
func TestTertiaryLabelsReadOnEverySidebarGround(t *testing.T) {
	for _, th := range appThemes {
		for _, dark := range []bool{false, true} {
			tk := th.config.Tokens(dark)
			grounds := map[string]ui.Color{
				"base":     tk.BgContainer,
				"hover":    tk.FillTertiary.Over(tk.BgContainer),
				"selected": selectionBg(tk, tk.BgContainer),
			}
			for name, ground := range grounds {
				if ratio := contrastRatio(tertiaryTextInk(tk, ground), ground); ratio < 4.5 {
					t.Errorf("%s dark=%v %s: tertiary label = %.2f:1; want at least 4.5:1",
						th.key, dark, name, ratio)
				}
			}
		}
	}
}

// The helper only moves a color that fails: a token already clearing AA has to
// come through unchanged, so the common case does not drift darker or lighter.
func TestTertiaryInkOnlyMovesFailingColors(t *testing.T) {
	for _, th := range appThemes {
		for _, dark := range []bool{false, true} {
			tk := th.config.Tokens(dark)
			for name, ground := range map[string]ui.Color{
				"base":     tk.BgContainer,
				"selected": selectionBg(tk, tk.BgContainer),
			} {
				if contrastRatio(tk.TextTertiary, ground) < 4.5 {
					continue
				}
				if got, want := tertiaryTextInk(tk, ground), tk.TextTertiary; got != want {
					t.Errorf("%s dark=%v %s: tertiary ink changed from %v to %v although it already reads",
						th.key, dark, name, want, got)
				}
			}
		}
	}
}
