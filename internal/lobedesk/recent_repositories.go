package lobedesk

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// Roots come from accepted repository snapshots, so symlinks and nested paths
// have already been resolved to their checkout root by gitrepo.Open.
func (a *app) rememberRepository(root string) {
	if root == "" {
		return
	}
	roots := appendRecentRepositories([]string{root}, a.recentRepositories)
	if tabs := a.repositoryTabs; tabs != nil && tabs.active == a {
		state := &tabs.sessionState
		if state.path != "" && !state.restoring && state.savedRecentHead != root {
			roots = tabs.saveRecentRepository(root, roots)
		}
		a.recentRepositories = roots
		tabs.refreshRepositoryRecents()
		return
	}
	a.recentRepositories = roots
}

func (a *app) canOpenRecentRepository() bool {
	if tabs := a.repositoryTabs; tabs != nil {
		return tabs.active == a && tabs.canChange()
	}
	return a.repo != nil && !a.busy && !a.loading && !a.repositoryChooserOpen && !a.hasPendingDialog(toastMain)
}

func (a *app) openRecentRepository(root string) {
	if root == "" || !a.canOpenRecentRepository() || (a.repo != nil && root == a.repo.Root) || !slices.Contains(a.recentRepositories, root) {
		return
	}
	a.repositorySwitchTarget = root
	a.openRepository(root, true)
	a.repositorySwitchTarget = ""
}

func (a *app) welcomeRecentRepositories(c *ui.Context) {
	if a.repositoryTabs == nil || len(a.recentRepositories) == 0 {
		return
	}
	tk := theme.TokensOf(c)
	allowed := a.canOpenRecentRepository()
	chosen := ""
	c.Key("welcome-recent-repositories")
	ui.Column(c).FillWidth().MaxWidth(420).Margin(8, 0, 0).Gap(4).Shrink(1).MinHeight(0).Children(func() {
		display.Text(c, a.t("sessionRepositories"), display.TextProps{Strong: true, FontSize: scaledFont(12)}).Shrink(0)
		layout.ScrollArea(c, layout.ScrollAreaProps{}, func() {
			ui.Column(c).FillWidth().Gap(2).Children(func() {
				for _, root := range a.recentRepositories {
					name := filepath.Base(root)
					pathLabel := recentCheckoutPathLabel(root, a.recentRepositories)
					row := ui.ButtonBase(c.Key(root)).Row().FillWidth().AlignItems(ui.Center).Gap(10).
						Padding(6, 8).Radius(tk.BorderRadiusSM).Label(name).Description(root).Tooltip(root).Disabled(!allowed)
					if row.Hovered() || row.Focused() {
						row.Background(tk.FillTertiary)
					}
					row.Children(func() {
						ui.Icon(c, lucide.FolderGit2()).FontSize(scaledFont(14)).TextColor(tk.TextSecondary).Shrink(0)
						ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(2).Children(func() {
							display.Text(c, name, display.TextProps{FontSize: scaledFont(12)}).SingleLine()
							display.Text(c, pathLabel, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).SingleLine()
						})
					})
					if allowed && row.Clicked() {
						chosen = root
					}
				}
			})
		}).MaxHeight(180).MinHeight(0).Shrink(1)
	})
	if chosen != "" {
		a.openRecentRepository(chosen)
	}
}

// The folder name is already the title. For same-named checkouts, start the
// subtitle at the distinguishing parent so truncation does not hide it.
func recentCheckoutPathLabel(root string, recent []string) string {
	name := filepath.Base(root)
	parent := filepath.Dir(root)
	var peers []string
	for _, other := range recent {
		if other != root && filepath.Base(other) == name {
			peers = append(peers, filepath.Dir(other))
		}
	}
	if suffix := distinguishingParentSuffix(parent, peers); suffix != "" && suffix != parent {
		return elidedPrefix(string(filepath.Separator) + suffix)
	}
	return root
}

// The bounded menu shows the highlighted checkout's complete path inline.
// This avoids competing hover/focus tooltips intercepting Escape or showing
// a different checkout from the item the keyboard is about to activate.
func repositoryContextMenu(c *ui.Context, target ui.Element, root string, items []overlay.MenuItem) string {
	tk := theme.TokensOf(c)
	open := ui.Local(target, "repositoryMenu", func() bool { return false })
	if target.Clicked() {
		*open = true
	}
	m := ui.ContextMenuBase(c, target, open)
	if !m.Open() {
		target.Tooltip(root)
	}
	chosen := ""
	m.Popup(func(panel ui.Element) {
		panel.Padding(4).Radius(tk.BorderRadius).Width(480).MaxWidthPercent(95).Background(tk.BgElevated).AttachedOrigin()
		tk.Elevate(panel, theme.BoxShadowSecondary)
		panel.Children(func() {
			preview := ""
			for i, it := range items {
				if it.Divider {
					ui.Box(c.Key(i)).Height(1).Margin(4, 0).AlignSelf(ui.Stretch).Background(tk.Border)
					continue
				}
				item := m.Item().Label(it.Label).Disabled(it.Disabled).
					PaddingY(6).PaddingX(12).MinHeight(32).Radius(tk.BorderRadiusSM).AlignSelf(ui.Stretch).FontSize(14)
				fg := tk.Text
				switch {
				case it.Disabled:
					fg = tk.TextDisabled
				case item.Pressed():
					item.Background(tk.FillSecondary)
				case item.Highlighted():
					item.Background(tk.FillTertiary)
				}
				if strings.HasPrefix(it.Key, "recent:") && item.Highlighted() {
					preview = it.Label
				}
				item.TextColor(fg).Children(func() {
					ui.Row(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
						if it.Icon != nil {
							ui.Icon(c, it.Icon).FontSize(14).Margin(0, 12, 0, 0)
						}
						ui.Text(c, it.Label).Grow(1).Basis(0).MinWidth(0).SingleLine()
					})
				})
				if item.Clicked() && !it.Disabled {
					chosen = it.Key
				}
			}
			if preview != "" {
				ui.Box(c).Height(1).Margin(4, 0).AlignSelf(ui.Stretch).Background(tk.Border)
				ui.Text(c, preview).FillWidth().MinWidth(0).Padding(6, 12).FontSize(12).TextColor(tk.TextSecondary)
			}
		})
	})
	return chosen
}
