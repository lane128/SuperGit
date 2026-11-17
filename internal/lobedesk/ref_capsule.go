package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// A ref chip is drawn as a capsule: a tinted pill with a hairline border and
// the ref's own color as its label ink. The plain colored text used before read
// as a highlighted word, not as an object you could pick out of a row; a pill
// gives the label an edge without turning it into a loud solid block, which is
// what a whole row of refs would become.
//
// The tint always comes from the lane color and is mixed over the surface the
// capsule sits on, so the pill belongs to a selected row as much as to a plain
// one. The label ink is deepened against the capsule's own surface -- not
// against the row -- because the tint is the darkest thing the text sits on.
const (
	refCapsuleTint   = 0.14
	refCapsuleBorder = 0.30
	refCapsuleHeight = 20
	// capsuleBorderMin is the luminance separation a capsule's edge needs from
	// the row it sits on. A translucent edge in the lane color keeps the pill
	// quiet on an ordinary row, but on the default light scheme a yellow or
	// blue lane on the selected row's tint lands near 1.02–1.09:1, where the
	// edge is legible only by hue.
	capsuleBorderMin = 1.25
	// A history row shares its width with the subject and the other refs, so
	// its capsules stop at one short label; the details pane has the whole line
	// to itself and allows a longer one. Both are named because
	// docs/graph-colors.md states them and a test pairs the two.
	refCapsuleRowWidth    = 150
	refCapsuleDetailWidth = 320
)

// refCapsuleWidth scales a capsule's width cap with the interface text size.
// The cap keeps a long ref from pushing the subject aside, but the label inside
// is elided to a fixed number of runes that grows wider with the text scale: at
// the large preset a fixed cap cuts a name that already carries its own middle
// ellipsis, so one label shows two. Growing the cap in step with the text keeps
// the name whole at every preset.
func refCapsuleWidth(base float32) float32 {
	return base * uiFontScaleValue()
}

type refCapsuleStyle struct {
	Ink     ui.Color
	Surface ui.Color
	Border  ui.Color
}

// refCapsuleColors derives a capsule's three colors from one lane color and the
// ground it sits on.
func refCapsuleColors(color, ground ui.Color) refCapsuleStyle {
	surface := color.Alpha(refCapsuleTint).Over(ground)
	ink := capsuleInk(color, surface, 4.5)
	border := capsuleBorder(color.Alpha(refCapsuleBorder).Over(ground), ink, ground)
	return refCapsuleStyle{Ink: ink, Surface: surface, Border: border}
}

// capsuleInk is the label ink for a capsule. On a light surface it deepens the
// lane color, as deepInk does for every other label. On a dark surface deepInk
// leaves the color alone, which costs a saturated red or blue accent about a
// point of contrast against its own tint -- an 11px label needs all of it, so
// the search runs toward white and lifts the color into a lighter tone of the
// same hue. Lifting is only what the failing colors need: one that already
// clears the minimum is returned unchanged.
func capsuleInk(color, surface ui.Color, min float64) ui.Color {
	if contrastRatio(color, surface) >= min {
		return color
	}
	if surfaceLuminance(surface) >= 0.5 {
		return deepInk(color, surface, min)
	}
	white := ui.RGB(255, 255, 255)
	for step := float32(0.05); step <= 0.95; step += 0.05 {
		mixed := color.Mix(white, step)
		if contrastRatio(mixed, surface) >= min {
			return mixed
		}
	}
	return white
}

// neutralCapsule is the capsule for chips that have no lane to point at, such
// as the "+2" that stands in for refs a row could not show or a remote's name.
// It has no color to lean on, so its surface is a full Fill rather than the
// fainter FillSecondary: a tinted capsule is held by its hue, a neutral one
// would otherwise read as a border with nothing inside. The ink is deepened
// against that surface because a selected row darkens the ground under it.
func neutralCapsule(tk *theme.Tokens, ground ui.Color) refCapsuleStyle {
	surface := tk.Fill.Over(ground)
	ink := capsuleInk(tk.TextSecondary, surface, 4.5)
	return refCapsuleStyle{
		Ink:     ink,
		Surface: surface,
		Border:  capsuleBorder(tk.BorderSecondary.Over(ground), ink, ground),
	}
}

// capsuleBorder keeps a capsule's edge legible by luminance, not only by hue.
// A translucent edge in the lane color, or the scheme's neutral border,
// carries the row's tint onto a hovered or selected ground; where the two sit
// close in luminance the outline disappears. Pulling the edge toward the
// capsule's own ink until it clears the floor keeps schemes that already clear
// it untouched.
func capsuleBorder(border, ink, ground ui.Color) ui.Color {
	if contrastRatio(border, ground) >= capsuleBorderMin {
		return border
	}
	for mix := float32(0.1); mix <= 0.9001; mix += 0.1 {
		candidate := ink.Alpha(mix).Over(ground)
		if contrastRatio(candidate, ground) >= capsuleBorderMin {
			return candidate
		}
	}
	return ink
}

// refPill builds the capsule element. The caller owns width, tooltip and the
// context menu, so the same pills serve the history row, the details pane and
// the HEAD chip.
func refPill(c *ui.Context, style refCapsuleStyle, icon *ui.SVG, label string) ui.Element {
	pill := ui.Row(c).Height(refCapsuleHeight).Radius(refCapsuleHeight/2).PaddingX(8).Gap(4).
		Shrink(0).AlignItems(ui.Center).Background(style.Surface).Border(1, style.Border).
		TextColor(style.Ink)
	pill.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).FontSize(scaledFont(11))
		}
		display.Text(c, label, display.TextProps{FontSize: scaledFont(11)}).SingleLine()
	})
	return pill
}
