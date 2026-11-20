package lobedesk

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/choice"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// settingsMaxWidth keeps a settings card from stretching across a wide
// window. Full-width cards push the label and the control metres apart and
// run help text past a comfortable line length; Sublime Merge's preferences
// hold a similar narrow measure.
const settingsMaxWidth = 760

// settingsHelpWidth keeps an explanatory line at a readable measure. A card
// is 760 wide at most, and a sentence run across its full width reads as one
// long line; capping it also keeps the help text from pushing the control it
// describes out of the eye's way.
const settingsHelpWidth = 520

// settingsHelp explains the controls without truncating localized wording.
// The settings content scrolls, so narrow windows can show the complete text.
func settingsHelp(c *ui.Context, text string) {
	display.Text(c, text, display.TextProps{
		Type: display.TextSecondary, FontSize: scaledFont(11),
	}).MaxWidth(settingsHelpWidth)
}

func (a *app) settingsPage(c *ui.Context) {
	windowTitleBar(c).Children(func() {
		display.Text(c, a.settingsWindowTitle(), display.TextProps{Heading: 5, Strong: true, FontSize: scaledFont(13)}).SingleLine()
	})
	ui.Row(c).Grow(1).Basis(0).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
		a.settingsNavigation(c)
		layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.settingsScroll}, func() {
			ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
				ui.Column(c).Grow(1).Basis(0).MinWidth(0).MaxWidth(settingsMaxWidth).Padding(16).Gap(14).Children(func() {
					section := c.Key(a.activeSettingsSection())
					switch a.activeSettingsSection() {
					case "appearance":
						a.appearanceCard(section)
					case "history":
						a.historyColumnsCard(section)
						a.graphCard(section)
						a.detailsCard(section)
					case "git":
						a.gitCard(section)
					case "shortcuts":
						a.shortcutsCard(section)
					case "about":
						a.aboutCard(section)
						ui.Row(section).FillWidth().Justify(ui.End).Children(func() {
							if workbenchButton(section, a.t("resetSettingsAction"), buttonNormal, false).Clicked() {
								a.confirmResetSettings()
							}
						})
					}
				})
			})
		}).Grow(1).Basis(0).MinWidth(0).MinHeight(0)
	})
}

func (a *app) settingsCard(c *ui.Context, title string, body func()) {
	tk := theme.TokensOf(c)
	ui.Column(c).FillWidth().Padding(16).Gap(12).
		Radius(tk.BorderRadiusLG).Background(tk.BgElevated).Border(1, tk.BorderSecondary).
		Children(func() {
			display.Text(c, title, display.TextProps{Strong: true, FontSize: scaledFont(13)})
			body()
		})
}

func (a *app) appearanceCard(c *ui.Context) {
	a.settingsCard(c, a.t("appearance"), func() {
		a.choiceRow(c, a.t("themeMode"), []choice.ToggleItem{
			{Value: "system", Label: a.t("themeSystem")},
			{Value: "light", Label: a.t("themeLight")},
			{Value: "dark", Label: a.t("themeDark")},
		}, a.themeMode, func(key string) {
			a.themeMode = key
			a.saveSettingsFor(toastSettings)
			a.invalidate()
		})
		// The palette is a separate choice from the mode: a scheme supplies a
		// light and a dark set, and system/light/dark picks which one shows.
		a.paletteRow(c)
		a.choiceRow(c, a.t("language"), []choice.ToggleItem{
			{Value: string(localeChinese), Label: "中文"},
			{Value: string(localeEnglish), Label: "English"},
		}, string(a.locale), func(key string) {
			a.setLocale(locale(key))
		})
		// Text size scales every size the app draws, so this row itself grows or
		// shrinks as soon as it is chosen, which is the clearest preview.
		a.choiceRow(c, a.t("fontSize"), []choice.ToggleItem{
			{Value: "small", Label: a.t("fontSizeSmall")},
			{Value: "standard", Label: a.t("fontSizeStandard")},
			{Value: "large", Label: a.t("fontSizeLarge")},
		}, fontScaleKey(uiFontScaleValue()), func(key string) {
			setUIFontScale(fontScaleOf(key))
			a.saveSettingsFor(toastSettings)
			a.invalidate()
		})
		a.codeFontSizeSetting(c)
		settingsHelp(c, a.t("appearanceHelp"))
	})
}

// detailsCard configures the commit details blocks. A hidden block is left
// out of the pane and the blocks around it take its room. The last block
// standing cannot be unchecked: an empty details pane would help no one.
func (a *app) detailsCard(c *ui.Context) {
	a.settingsCard(c, a.t("commitDetails"), func() {
		showInfo := !a.detailsInfoHidden
		showChanges := !a.detailsChangesHidden
		showDiff := !a.detailsDiffHidden
		was := [3]bool{showInfo, showChanges, showDiff}
		// The Summary and the patch are the panes the files column opens into;
		// with both hidden a click on a file could show nothing, so one of the
		// two always stays. The files column itself is free to go: the panes
		// keep the full width without it.
		lastSummary := showInfo && !showDiff
		lastPatch := showDiff && !showInfo
		// A disabled box with no reason reads as broken. MyGo's tooltip skips a
		// disabled control, so the reason sits on an enabled wrapper around the
		// box, where a hover reaches it; the box keeps it for assistive
		// technology.
		blockBox := func(value *bool, label string, disabled bool) {
			draw := func(c *ui.Context) {
				box := choice.Checkbox(c, value, label, choice.CheckboxProps{Size: 14})
				box.Disabled(disabled)
				box.Tooltip(a.t("detailsKeepOneBlock"))
			}
			if !disabled {
				draw(c)
				return
			}
			ui.Box(c).AlignSelf(ui.Start).Tooltip(a.t("detailsKeepOneBlock")).Children(func() {
				draw(c)
			})
		}
		blockBox(&showInfo, a.t("detailsInfo"), lastSummary)
		blockBox(&showChanges, a.t("detailsChanges"), false)
		blockBox(&showDiff, a.t("detailsDiff"), lastPatch)
		settingsHelp(c, a.t("detailsLayoutHelp"))
		if showInfo != was[0] || showChanges != was[1] || showDiff != was[2] {
			a.detailsInfoHidden = !showInfo
			a.detailsChangesHidden = !showChanges
			a.detailsDiffHidden = !showDiff
			a.normalizeDetailVisibility()
			a.saveSettingsFor(toastSettings)
			a.invalidate()
		}
	})
}

// currentTheme resolves the active color scheme, falling back to the first
// entry so a stale key still has a label and dots to show.
func (a *app) currentTheme() appTheme {
	for _, th := range appThemes {
		if th.key == a.palette {
			return th
		}
	}
	return appThemes[0]
}

// paletteRow shows the schemes as a dropdown. The trigger keeps the active
// scheme's three-dot preview; the menu lists the names with a check on the
// active one. A segmented row stops scaling once more schemes arrive, and the
// ToggleGroup behind choiceRow would mask the dots into one color anyway.
func (a *app) paletteRow(c *ui.Context) {
	tk := theme.TokensOf(c)
	current := a.currentTheme()
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		display.Text(c, a.t("palette"), display.TextProps{FontSize: scaledFont(12)}).Width(96)
		items := make([]overlay.MenuItem, 0, len(appThemes))
		for _, th := range appThemes {
			item := overlay.MenuItem{Key: th.key, Label: a.t(th.label)}
			if th.key == a.palette {
				item.Extra = "✓"
			}
			items = append(items, item)
		}
		chosen := overlay.DropdownMenu(c, &a.paletteOpen, overlay.DropdownMenuProps{Items: items}, func(trigger ui.Element) {
			styleWorkbenchButton(c, trigger, buttonNormal, false, a.paletteOpen)
			// The visual label is on the row to the left and inside the
			// trigger itself, so a reader reaching the button heard only its
			// dots and name. Name the control and say what it currently holds.
			trigger.Role(ui.RoleButton).Label(a.t("palette")).Description(a.t(current.label))
			trigger.Gap(6).Children(func() {
				for _, dot := range current.dots {
					ui.Box(c).Size(6, 6).Radius(3).Background(dot)
				}
				ui.Text(c, a.t(current.label)).FontSize(scaledFont(12)).SingleLine()
				ui.Icon(c, lucide.ChevronDown()).FontSize(scaledFont(11)).TextColor(tk.TextTertiary)
			})
		})
		if chosen != "" && chosen != a.palette {
			a.palette = chosen
			a.saveSettingsFor(toastSettings)
			a.invalidate()
		}
	})
}

// choiceRow renders one setting as a segmented control: a label and the
// choices joined on one track. A row of separate buttons reads as several
// independent switches; a segmented control shows they are one choice, which
// keeps the appearance card calm as more palettes arrive.
func (a *app) choiceRow(c *ui.Context, label string, items []choice.ToggleItem, current string, set func(string)) {
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		display.Text(c, label, display.TextProps{FontSize: scaledFont(12)}).Width(96)
		tk := theme.TokensOf(c)
		ui.Row(c).Grow(1).Basis(0).MinWidth(0).Wrap().Padding(3).Gap(4).
			Background(tk.BgLayout).Radius(6).Children(func() {
			for _, item := range items {
				selected := item.Value == current
				button := styleWorkbenchButton(c.Key(item.Value), ui.ButtonBase(c.Key(item.Value)), buttonNormal, item.Disabled, selected).
					Role(ui.RoleToggleButton).Label(item.Label).Checked(selected).Shrink(0)
				button.Children(func() {
					if item.Icon != nil {
						ui.Icon(c, item.Icon).FontSize(scaledFont(14))
					}
					ui.Text(c, item.Label).SingleLine()
				})
				if button.Clicked() && !selected && !item.Disabled {
					set(item.Value)
				}
			}
		})
	})
}

// gitCard holds the settings that change what the app runs against Git. The
// automatic fetch keeps ahead/behind fresh without a visible success toast;
// pruning only applies when that fetch runs.
func (a *app) gitCard(c *ui.Context) {
	a.settingsCard(c, a.t("gitBehavior"), func() {
		wasFetch := a.autoFetch
		wasPrune := a.autoFetchPrune
		choice.Checkbox(c, &a.autoFetch, a.t("autoFetch"), choice.CheckboxProps{Size: 14})
		settingsHelp(c, a.t("autoFetchHelp"))
		// Pruning rides on that fetch, so with fetching off the box would
		// save a choice that never applies: it reads unavailable instead of
		// pretending to be a setting of its own. The reason goes on an enabled
		// wrapper as well as the box, since a disabled control shows no
		// tooltip of its own.
		pruneBox := func(c *ui.Context) {
			box := choice.Checkbox(c, &a.autoFetchPrune, a.t("autoFetchPrune"), choice.CheckboxProps{Size: 14})
			box.Disabled(!a.autoFetch)
			box.Tooltip(a.t("autoFetchPruneNeedsFetch"))
		}
		if a.autoFetch {
			pruneBox(c)
		} else {
			ui.Box(c).AlignSelf(ui.Start).Tooltip(a.t("autoFetchPruneNeedsFetch")).Children(func() {
				pruneBox(c)
			})
		}
		settingsHelp(c, a.t("autoFetchPruneHelp"))
		a.choiceRow(c, a.t("autoFetchEvery"), []choice.ToggleItem{
			{Value: "0", Label: a.fetchIntervalLabel(0)},
			{Value: "5", Label: a.fetchIntervalLabel(5)},
			{Value: "15", Label: a.fetchIntervalLabel(15)},
			{Value: "30", Label: a.fetchIntervalLabel(30)},
			{Value: "60", Label: a.fetchIntervalLabel(60)},
		}, itoa(a.autoFetchEvery), func(key string) {
			next := fetchIntervalValue(key)
			if next == a.autoFetchEvery {
				return
			}
			a.autoFetchEvery = next
			// A newly chosen interval starts counting now: choosing 15
			// minutes should not fire a fetch the moment the choice is made,
			// and switching between presets should not fetch again and again.
			a.lastAutoFetch = time.Now()
			a.saveSettingsFor(toastSettings)
			a.invalidate()
		})
		settingsHelp(c, a.t("autoFetchEveryHelp"))
		if a.autoFetch != wasFetch || a.autoFetchPrune != wasPrune {
			a.saveSettingsFor(toastSettings)
			a.invalidate()
		}
	})
}

func (a *app) graphCard(c *ui.Context) {
	a.settingsCard(c, a.t("historyGraph"), func() {
		was := a.showGraph
		choice.Checkbox(c, &a.showGraph, a.t("showHistoryGraph"), choice.CheckboxProps{Size: 14})
		if a.showGraph != was {
			a.saveSettingsFor(toastSettings)
			a.invalidate()
		}
		settingsHelp(c, a.t("showHistoryGraphHelp"))
		line, node := a.graphStyle()
		a.choiceRow(c, a.t("graphLineWidth"), []choice.ToggleItem{
			{Value: "thin", Label: a.t("graphLineThin")},
			{Value: "standard", Label: a.t("graphLineStandard")},
			{Value: "thick", Label: a.t("graphLineThick")},
		}, graphLineKey(line), func(key string) {
			a.historyLine = graphLineOf(key)
			a.saveSettingsFor(toastSettings)
			a.invalidate()
		})
		a.choiceRow(c, a.t("graphNodeSize"), []choice.ToggleItem{
			{Value: "small", Label: a.t("graphNodeSmall")},
			{Value: "standard", Label: a.t("graphNodeStandard")},
			{Value: "large", Label: a.t("graphNodeLarge")},
		}, graphNodeKey(node), func(key string) {
			a.historyNode = graphNodeOf(key)
			a.saveSettingsFor(toastSettings)
			a.invalidate()
		})
	})
}

// historyColumnsCard controls the history row's second line. Hiding one
// column keeps the other; hiding both leaves a single-line list, and the row
// stays the same height so the graph keeps its alignment.
func (a *app) historyColumnsCard(c *ui.Context) {
	a.settingsCard(c, a.t("historyList"), func() {
		showAuthor := !a.hideAuthor
		showTime := !a.hideTime
		wasAuthor, wasTime := showAuthor, showTime
		choice.Checkbox(c, &showAuthor, a.t("showAuthor"), choice.CheckboxProps{Size: 14})
		choice.Checkbox(c, &showTime, a.t("showTime"), choice.CheckboxProps{Size: 14})
		settingsHelp(c, a.t("historyColumnsHelp"))
		if showAuthor != wasAuthor || showTime != wasTime {
			a.hideAuthor = !showAuthor
			a.hideTime = !showTime
			a.saveSettingsFor(toastSettings)
			a.invalidate()
		}
	})
}

// shortcutsCard lists every binding the table declares, grouped by page. It
// reads the same table the window dispatcher registers from, so the card
// cannot go stale: a key that works is on this card, and the dispatcher test
// fails if a window key is registered anywhere else.
func (a *app) shortcutsCard(c *ui.Context) {
	a.settingsCard(c, a.t("shortcuts"), func() {
		for _, group := range shortcutGroupOrder {
			var rows []appShortcut
			for _, s := range shortcutTable() {
				if s.Group == group {
					rows = append(rows, s)
				}
			}
			// A page with no keys of its own gets no empty heading.
			if len(rows) == 0 {
				continue
			}
			ui.Column(c).FillWidth().Gap(4).Children(func() {
				display.Text(c, a.t(shortcutGroupLabel(group)), display.TextProps{
					Type: display.TextSecondary, Strong: true, FontSize: scaledFont(11),
				})
				for _, s := range rows {
					// The label is the row's text and the key sits at the far
					// edge, so a reader scans one column of actions and one of
					// keys however long the wording is.
					ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Center).Children(func() {
						display.Text(c, a.t(s.Label), display.TextProps{FontSize: scaledFont(12)}).
							Grow(1).Basis(0).MinWidth(0).SingleLine()
						display.Text(c, s.Keys, display.TextProps{
							Type: display.TextSecondary, Code: true, FontSize: scaledFont(11),
						}).Shrink(0)
					})
				}
			})
		}
		settingsHelp(c, a.t("shortcutsHelp"))
	})
}

func (a *app) aboutCard(c *ui.Context) {
	a.settingsCard(c, a.t("about"), func() {
		display.Text(c, "SuperGit · Lobe MyGo", display.TextProps{FontSize: scaledFont(12), Strong: true})
		display.Text(c, a.t("settingsAboutText"), display.TextProps{
			Type: display.TextSecondary, FontSize: scaledFont(11), Rows: 2,
		})
		if path, err := settingsPath(); err == nil {
			// The row is the selectable scope: a paragraph on its own inside the
			// page's scroll area would not be selectable, since MyGo needs a
			// Selectable container between the text and the scroll controls.
			ui.Row(c).Gap(6).AlignItems(ui.Center).Selectable().Children(func() {
				display.Text(c, a.t("settingsPath"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)})
				display.Text(c, path, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11), Code: true}).
					SingleLine().Tooltip(path)
			})
		}
	})
}

// confirmResetSettings asks before overwriting the saved preferences. The
// dialog names the scope so "reset" does not sound like it might touch the
// repository.
func (a *app) confirmResetSettings() {
	// This entry belongs to the settings panel even if native focus changes
	// before its action is delivered. Its result stays in this window too;
	// divider preference writes on the main window have their own owner.
	a.askConfirmInWindow(toastSettings, "", a.t("resetSettingsTitle"), a.t("resetSettingsMessage"), a.t("resetSettings"), false, func() {
		a.resetSettingsFor(toastSettings)
	})
}
