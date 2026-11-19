package lobedesk

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
)

const codeFontMin, codeFontMax = 11, 19

func validCodeFontSize(size float32) bool {
	return size >= codeFontMin && size <= codeFontMax
}

func (a *app) patchFontSize() float32 {
	if a.codeFontSize > 0 {
		return a.codeFontSize
	}
	return scaledFont(diffCodeSize)
}

func (a *app) patchRowHeight() float32 {
	return max(diffRowH, a.patchFontSize()+8)
}

func (a *app) codeFontSizeSetting(c *ui.Context) {
	size := a.patchFontSize()
	down := max(codeFontMin, float32(math.Ceil(float64(size)))-1)
	up := min(codeFontMax, float32(math.Floor(float64(size)))+1)
	next := a.codeFontSize
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		display.Text(c, a.t("codeFontSize"), display.TextProps{FontSize: scaledFont(12)}).Width(96).Shrink(0)
		ui.Row(c).Grow(1).Basis(0).MinWidth(0).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Shrink(0).Children(func() {
				if workbenchIconButton(c, lucide.Minus(), a.t("decreaseCodeFont"), buttonNormal, size <= codeFontMin).Clicked() {
					next = down
				}
				display.Text(c, fmt.Sprintf("%.1f pt", size), display.TextProps{FontSize: scaledFont(12)}).Width(58).TextAlign(ui.Center)
				if workbenchIconButton(c, lucide.Plus(), a.t("increaseCodeFont"), buttonNormal, size >= codeFontMax).Clicked() {
					next = up
				}
			})
			if workbenchButton(c, a.t("codeFontFollowInterface"), buttonNormal, next == 0).Clicked() {
				next = 0
			}
		})
	})
	if next != a.codeFontSize {
		a.codeFontSize = next
		a.saveSettingsFor(toastSettings)
		a.invalidate()
	}
}
