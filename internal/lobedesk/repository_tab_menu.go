package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func repositoryColorDot(dark bool, preset string) (bg, ink ui.Color) {
	dots := map[string][2]string{
		"blue": {"#476EAE", "#8FAFE5"}, "green": {"#4B806B", "#94BFAE"},
		"amber": {"#96783B", "#D2B67C"}, "rose": {"#986E85", "#D0A5BC"},
		"violet": {"#7C6CA4", "#BAAFE0"}, "gray": {"#738294", "#ACB8C7"},
	}
	if preset == "default" {
		bg, ink, _ = repositoryTabColors(dark, preset, false)
		return
	}
	index := 0
	ink = ui.Hex("#FFFFFF")
	if dark {
		index = 1
		ink = ui.Hex("#202B38")
	}
	return ui.Hex(dots[preset][index]), ink
}

func (tabs *repositoryTabs) tabMenu(c *ui.Context, target ui.Element, a, session *app) (closeTab, editAlias bool) {
	tk := theme.TokensOf(c)
	open := ui.Local(target, "repository-menu", func() bool { return false })
	root := ui.Local(target, "repository-menu-root", func() string { return "" })
	if !*open {
		*root = ""
		if session.repo != nil {
			*root = session.repo.Root
		}
	}
	m := ui.ContextMenuBase(c, target, open)
	m.Popup(func(panel ui.Element) {
		panel.Padding(8).Width(244).Radius(8).Background(tk.BgElevated).Border(1, tk.BorderSecondary)
		tk.Elevate(panel, theme.BoxShadowSecondary)
		// Resolve Escape within the popup before the workbench's global
		// clear-selection shortcut can consume it.
		panel.OnShortcut(0, ui.KeyEscape, func() {
			*open = false
			tabs.focusTab = session
			c.After(0)
		})
		panel.Children(func() {
			if session.repo != nil {
				ui.Text(c, a.t("repositoryTabColor")).FontSize(scaledFont(12)).TextColor(tk.TextSecondary).Padding(3, 7, 5, 7)
				m.Group(func(group ui.Element) {
					group.Gap(4).Padding(2, 3).Label(a.t("repositoryTabColor"))
					for _, color := range repositoryColorPresets {
						selected := normalizedRepositoryColor(session.repositoryColor) == color
						label := a.t("repositoryColor_" + color)
						dot := m.RadioItem(selected).Size(28, 28).Padding(0).Radius(14).Label(label).Tooltip(label).
							Disabled(!tabs.canColorRepository(session, *root)).Justify(ui.Center).AlignItems(ui.Center)
						if dot.Highlighted() {
							dot.Background(tk.FillTertiary)
						}
						bg, ink := repositoryColorDot(tk.Dark, color)
						dot.Children(func() {
							swatch := ui.Box(c).Size(18, 18).Radius(9).Background(bg).Justify(ui.Center).AlignItems(ui.Center)
							if color == "default" {
								swatch.Border(1, tk.Border)
							}
							swatch.Children(func() {
								if selected {
									ui.Icon(c, lucide.Check()).FontSize(12).TextColor(ink)
								} else if color == "default" {
									ui.Text(c, "∅").FontSize(12).TextColor(ink)
								}
							})
						})
						if dot.Clicked() {
							tabs.setRepositoryColor(session, *root, color)
						}
					}
				})
				ui.Box(c).Height(1).Background(tk.BorderSecondary).Margin(7, 0)
			}
			action := func(label string, icon *ui.SVG, disabled bool) bool {
				item := m.Item().Label(label).Disabled(disabled).Height(30).Padding(0, 7).Radius(5).FontSize(scaledFont(13)).TextColor(tk.Text)
				if item.Highlighted() {
					item.Background(tk.FillTertiary)
				}
				item.Children(func() {
					ui.Icon(c, icon).FontSize(14).TextColor(tk.TextSecondary)
					ui.Text(c, label).Margin(0, 0, 0, 8).SingleLine()
				})
				return item.Clicked() && !disabled
			}
			if session.repo != nil {
				editAlias = action(a.t("setRepositoryAlias"), lucide.Pencil(), !tabs.canColorRepository(session, *root))
			}
			closeTab = action(a.t("closeTab"), lucide.X(), !tabs.canClose(session))
		})
	})
	return
}
