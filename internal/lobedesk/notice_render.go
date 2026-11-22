package lobedesk

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

var noticeStackAnimation = ui.Animation{Duration: 400 * time.Millisecond, Ease: motion.ToastEase}
var noticeFadeAnimation = ui.Animation{Duration: 200 * time.Millisecond, Ease: motion.EaseOut}
var noticeSizeAnimation = ui.Animation{Duration: 150 * time.Millisecond, Ease: motion.CSSEase}

type noticeCardLayout struct {
	y, scale, front float32
	collapsed       bool
}

// The engine still owns expiry, hover/focus pause, actions and announcements.
// Only the card rendering is local, including progress from ToastParts.Left.
func (a *app) noticeViewport(c *ui.Context) ui.Element {
	tk := theme.TokensOf(c)
	return ui.ToastViewportBase(c, func(viewport ui.Element, toasts []ui.Toast) {
		viewport.AlignItems(ui.End).Justify(ui.End)
		stack := ui.Box(c).Width(360).MaxWidthPercent(100)
		heights := *ui.Local(stack, "heights", func() map[string]float32 { return map[string]float32{} })
		expanded := stack.Hovered() || stack.FocusWithin()
		front := float32(0)
		if len(toasts) > 0 {
			front = heights[toasts[len(toasts)-1].ID]
		}
		total := front + 12*float32(max(len(toasts)-1, 0))
		if expanded {
			total = 12 * float32(max(len(toasts)-1, 0))
			for _, toast := range toasts {
				total += heights[toast.ID]
			}
		}
		stack.Height(total)
		layouts := make([]noticeCardLayout, len(toasts))
		offset := float32(0)
		for k := len(toasts) - 1; k >= 0; k-- {
			depth := len(toasts) - 1 - k
			scale := 1 - 0.05*float32(depth)
			l := noticeCardLayout{y: float32(depth)*12 + (1-scale)*front, scale: scale, collapsed: !expanded && depth > 0, front: front}
			if expanded {
				l.y, l.scale = offset, 1
			}
			offset += heights[toasts[k].ID] + 12
			layouts[k] = l
		}
		stack.Children(func() {
			for k, toast := range toasts {
				a.noticeCard(c, tk, toast, heights, layouts[k])
			}
		})
	})
}

func (a *app) noticeCard(c *ui.Context, tk *theme.Tokens, toast ui.Toast, heights map[string]float32, l noticeCardLayout) {
	parts := ui.ToastBase(c, toast)
	root := parts.Root.Absolute().Left(0).Right(0).Bottom(0).Radius(10).Clip().
		Background(tk.BgElevated).TextColor(tk.Text).FontSize(scaledFont(12)).TransformOrigin(0.5, 1)
	tk.Elevate(root, theme.BoxShadowSecondary)
	enter := motion.Toast(motion.Bottom)
	enter.Colors = true
	root.Transition(enter)
	height := heights[toast.ID]
	if l.collapsed && l.front > 0 {
		height = l.front
	}
	if height > 0 {
		root.Height(root.AnimateTo("height", height, noticeSizeAnimation))
	}
	root.Translate(0, root.AnimateTo("y", -l.y, noticeStackAnimation))
	scale := root.AnimateTo("scale", l.scale, noticeStackAnimation)
	root.Scale(scale, scale)
	shown := float32(1)
	if l.collapsed {
		shown = 0
	}
	var body ui.Element
	root.Children(func() {
		body = ui.Column(c).Shrink(0).Opacity(root.AnimateTo("content", shown, noticeFadeAnimation))
		body.Children(func() {
			ui.Row(c).Padding(12, 16).Gap(10).AlignItems(ui.Start).Children(func() {
				// Align the title and status glyph to the close control, not to
				// the middle of the whole card when a description or action follows.
				headerHeight := scaledFont(workbenchButtonBaseHeight)
				icon, ink := noticeIcon(tk, toast.Type)
				ui.Column(c).Height(headerHeight).Shrink(0).Justify(ui.Center).Children(func() {
					ui.Icon(c, icon).FontSize(scaledFont(16)).TextColor(ink).AccessHidden(true)
				})
				ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(6).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Start).Children(func() {
						text := toast.Title
						if text == "" {
							text = toast.Description
						}
						ui.Column(c).MinHeight(headerHeight).Grow(1).Basis(0).MinWidth(0).Justify(ui.Center).Children(func() {
							ui.Text(c, text).FontWeight(500)
						})
						close := styleWorkbenchButton(c, parts.CloseButton(), buttonNormal, false, false).
							Width(headerHeight).PaddingX(0).Label(a.t("close")).Shrink(0)
						close.Children(func() { ui.Icon(c, lucide.X()).FontSize(scaledFont(14)) })
					})
					if toast.Title != "" && toast.Description != "" {
						ui.Text(c, toast.Description).FontSize(scaledFont(11)).TextColor(tk.TextSecondary)
					}
					if toast.Action != "" {
						ui.Row(c).Justify(ui.End).Children(func() {
							button := styleWorkbenchButton(c, parts.ActionButton(), buttonNormal, false, false)
							button.Children(func() { ui.Text(c, toast.Action).SingleLine() })
						})
					}
				})
			})
			if toast.Timeout > 0 && toast.OnAction != nil && toast.Action != "" {
				remaining := max(0, min(1, float64(parts.Left())/float64(toast.Timeout)))
				ui.Box(c).FillWidth().Height(3).Role(ui.RoleProgress).Label(a.t("noticeTimeRemaining")).Range(0, 1, remaining).
					Background(tk.FillSecondary).DrawOver(func(p *ui.Painter, r ui.Rect) {
					r.W *= float32(remaining)
					p.Fill(r, tk.Blue, 0)
				})
				// Refresh from the engine clock without keeping the window in an
				// endless animation; input and automation can settle between ticks.
				c.After(100 * time.Millisecond)
			}
		})
	})
	if h := body.Bounds().H; h > 0 && h != heights[toast.ID] {
		heights[toast.ID] = h
		c.Invalidate()
	}
}

func noticeIcon(tk *theme.Tokens, kind string) (*ui.SVG, ui.Color) {
	switch kind {
	case "success":
		return lucide.CircleCheck(), tk.Success
	case "warning":
		return lucide.TriangleAlert(), tk.Warning
	case "error":
		return lucide.CircleX(), tk.Error
	default:
		return lucide.Info(), tk.Info
	}
}
