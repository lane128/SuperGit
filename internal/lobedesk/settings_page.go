package lobedesk

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/choice"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// Settings use a fixed reading width, shrinking only when the window cannot
// fit that width plus the page gutters.
const settingsMaxWidth = 640
const settingsPagePadding = 24
const settingsNavigationWidth = 160
const settingsStackWidth = 688

// settingsFormRow uses a single reading order at both widths. Controls move as
// a group below their label; individual segmented options never wrap.
func settingsFormRow(c *ui.Context, label string, body func()) {
	width, _ := c.Size()
	compact := width < settingsStackWidth
	var row ui.Element
	if compact {
		row = ui.Column(c).FillWidth().Padding(12).Gap(8).MinHeight(44)
	} else {
		row = ui.Row(c).FillWidth().Padding(10, 12).Gap(16).MinHeight(44).AlignItems(ui.Center)
	}
	row.Children(func() {
		name := display.Text(c, label, display.TextProps{FontSize: scaledFont(13)})
		if !compact {
			name.Width(120).Shrink(0)
		}
		controls := ui.Row(c).Gap(8).Wrap().MinWidth(0).AlignItems(ui.Center)
		if compact {
			controls.FillWidth()
		} else {
			controls.Grow(1).Basis(0).Justify(ui.End)
		}
		controls.Children(body)
	})
	settingsDivider(c)
}

func settingsCheckRow(c *ui.Context, body func()) {
	ui.Column(c).FillWidth().Padding(12).Gap(6).MinHeight(44).Children(body)
}

func settingsCheckbox(c *ui.Context, value *bool, label, help string, disabled bool) {
	settingsCheckRow(c, func() {
		ui.Box(c).FillWidth().Tooltip(help).Children(func() {
			choice.Checkbox(c, value, label, choice.CheckboxProps{Size: 14}).Disabled(disabled).Tooltip(help)
		})
		if help != "" {
			settingsHelp(c, help)
		}
	})
	settingsDivider(c)
}

func settingsDivider(c *ui.Context) {
	ui.Box(c).FillWidth().Height(1).Background(theme.TokensOf(c).BorderSecondary)
}

// settingsHelpWidth keeps explanatory text at a readable measure.
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
			ui.Row(c).FillWidth().Padding(settingsPagePadding).Children(func() {
				ui.Column(c).Width(settingsMaxWidth).Shrink(1).MinWidth(0).Gap(24).Role(ui.RoleGroup).Label(a.t("settingsContent")).Children(func() {
					section := c.Key(a.activeSettingsSection())
					ui.Column(section).FillWidth().Gap(4).Children(func() {
						for _, item := range settingsSections {
							if item.key == a.activeSettingsSection() {
								display.Text(section, a.t(item.label), display.TextProps{FontSize: scaledFont(20), Strong: true})
							}
						}
						settingsHelp(section, a.t("settingsIntro_"+a.activeSettingsSection()))
					})
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
						a.updatesCard(section)
						ui.Column(section).FillWidth().Gap(12).Children(func() {
							settingsDivider(section)
							settingsFormRow(section, a.t("resetSettings"), func() {
								if workbenchButton(section, a.t("resetSettingsAction"), buttonDangerIcon, false).Clicked() {
									a.confirmResetSettings()
								}
							})
							settingsHelp(section, a.t("settingsResetHelp"))
						})
					}
				})
			})
		}).Grow(1).Basis(0).MinWidth(0).MinHeight(0).Background(theme.TokensOf(c).BgContainer)
	})
}

func (a *app) settingsCard(c *ui.Context, title string, body func()) {
	tk := theme.TokensOf(c)
	ui.Column(c).FillWidth().Gap(8).Role(ui.RoleGroup).Label(title).Children(func() {
		display.Text(c, title, display.TextProps{Strong: true, Type: display.TextSecondary, FontSize: scaledFont(12)})
		ui.Column(c).FillWidth().Radius(8).Border(1, tk.BorderSecondary).Children(body)
	})
}

func (a *app) appearanceCard(c *ui.Context) {
	a.settingsCard(c, a.t("settingsInterface"), func() {
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
		a.diffPaletteRow(c)
		a.choiceRow(c, a.t("language"), []choice.ToggleItem{
			{Value: string(localeChinese), Label: "中文"},
			{Value: string(localeEnglish), Label: "English"},
		}, string(a.locale), func(key string) {
			a.setLocale(locale(key))
		})
	})
	a.settingsCard(c, a.t("settingsTypography"), func() {
		a.uiFontFamilySetting(c)
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
		a.codeFontFamilySetting(c)
		a.codeFontSizeSetting(c)
	})
	ui.Column(c).FillWidth().Gap(8).Children(func() {
		display.Text(c, a.t("settingsReadingPreview"), display.TextProps{Strong: true, Type: display.TextSecondary, FontSize: scaledFont(12)})
		a.fontPreviewBox(c)
		settingsHelp(c, a.t("settingsFontHelp"))
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
		blockBox := func(value *bool, label string, disabled bool) {
			help := ""
			if disabled {
				help = a.t("detailsKeepOneBlock")
			}
			settingsCheckbox(c, value, label, help, disabled)
		}
		blockBox(&showInfo, a.t("detailsInfo"), lastSummary)
		blockBox(&showChanges, a.t("detailsChanges"), false)
		blockBox(&showDiff, a.t("detailsDiff"), lastPatch)
		settingsCheckRow(c, func() { settingsHelp(c, a.t("detailsLayoutHelp")) })
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
	settingsFormRow(c, a.t("palette"), func() {
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
			trigger.Gap(6).MaxWidth(252).Shrink(1).Children(func() {
				for _, dot := range current.dots {
					ui.Box(c).Size(6, 6).Radius(3).Background(dot)
				}
				ui.Text(c, a.t(current.label)).FontSize(scaledFont(12)).SingleLine().Shrink(1).MinWidth(0)
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
	settingsFormRow(c, label, func() {
		tk := theme.TokensOf(c)
		var optionWidth float32
		for _, item := range items {
			width, _ := c.MeasureText(0, ui.Span{Text: item.Label, Size: scaledFont(12), Weight: 600})
			width += 22
			if item.Icon != nil {
				width += scaledFont(14) + 6
			}
			optionWidth = max(optionWidth, float32(math.Ceil(float64(width))))
		}
		ui.Row(c).Shrink(0).Padding(3).Gap(2).
			Background(tk.BgLayout).Radius(6).Children(func() {
			for _, item := range items {
				selected := item.Value == current
				button := styleWorkbenchButton(c.Key(item.Value), ui.ButtonBase(c.Key(item.Value)), buttonNormal, item.Disabled, selected).
					Role(ui.RoleToggleButton).Label(item.Label).Checked(selected).Width(optionWidth).Shrink(0)
				if !selected && !item.Disabled && !button.Hovered() && !button.Pressed() {
					button.Background(ui.Transparent).Border(1, ui.Transparent)
				}
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
		settingsCheckbox(c, &a.autoFetch, a.t("autoFetch"), a.t("autoFetchHelp"), false)
		pruneHelp := a.t("autoFetchPruneHelp")
		if !a.autoFetch {
			pruneHelp = a.t("autoFetchPruneNeedsFetch")
		}
		settingsCheckbox(c, &a.autoFetchPrune, a.t("autoFetchPrune"), pruneHelp, !a.autoFetch)
		settingsFormRow(c, a.t("autoFetchEvery"), func() {
			var items []overlay.MenuItem
			for _, minutes := range []int{0, 5, 15, 30, 60} {
				item := overlay.MenuItem{Key: itoa(minutes), Label: a.fetchIntervalLabel(minutes)}
				if minutes == a.autoFetchEvery {
					item.Extra = "✓"
				}
				items = append(items, item)
			}
			key := overlay.DropdownMenu(c, &a.settingsIntervalOpen, overlay.DropdownMenuProps{Items: items}, func(trigger ui.Element) {
				styleWorkbenchButton(c, trigger, buttonNormal, false, a.settingsIntervalOpen)
				trigger.Role(ui.RoleButton).Label(a.t("autoFetchEvery")).Description(a.fetchIntervalLabel(a.autoFetchEvery)).Gap(6).Children(func() {
					ui.Text(c, a.fetchIntervalLabel(a.autoFetchEvery)).FontSize(scaledFont(12)).SingleLine()
					ui.Icon(c, lucide.ChevronDown()).FontSize(scaledFont(11))
				})
			})
			if key != "" && fetchIntervalValue(key) != a.autoFetchEvery {
				a.autoFetchEvery = fetchIntervalValue(key)
				// Start a new interval without performing a fetch on selection.
				a.lastAutoFetch = time.Now()
				a.saveSettingsFor(toastSettings)
				a.invalidate()
			}
		})
		settingsCheckRow(c, func() { settingsHelp(c, a.t("autoFetchEveryHelp")) })
		if a.autoFetch != wasFetch || a.autoFetchPrune != wasPrune {
			a.saveSettingsFor(toastSettings)
			a.invalidate()
		}
	})
}

func (a *app) graphCard(c *ui.Context) {
	a.settingsCard(c, a.t("historyGraph"), func() {
		was := a.showGraph
		settingsCheckbox(c, &a.showGraph, a.t("showHistoryGraph"), a.t("showHistoryGraphHelp"), false)
		if a.showGraph != was {
			a.saveSettingsFor(toastSettings)
			a.invalidate()
		}
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
		settingsCheckbox(c, &showAuthor, a.t("showAuthor"), "", false)
		settingsCheckbox(c, &showTime, a.t("showTime"), "", false)
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
	for _, group := range shortcutGroupOrder {
		var rows []appShortcut
		for _, s := range shortcutTable() {
			if s.Group == group {
				rows = append(rows, s)
			}
		}
		if len(rows) == 0 {
			continue
		}
		a.settingsCard(c, a.t(shortcutGroupLabel(group)), func() {
			for i, s := range rows {
				if i > 0 {
					settingsDivider(c)
				}
				ui.Row(c).FillWidth().Padding(8, 12).Gap(16).AlignItems(ui.Center).Children(func() {
					display.Text(c, a.t(s.Label), display.TextProps{FontSize: scaledFont(13)}).Grow(1).Basis(0).MinWidth(0)
					tk := theme.TokensOf(c)
					ui.Box(c).MaxWidth(140).Shrink(0).Padding(3, 6).Radius(4).Background(tk.BgContainerSecondary).Border(1, tk.BorderSecondary).Children(func() {
						display.Text(c, s.Keys, display.TextProps{Code: true, FontSize: scaledFont(11)}).Font(a.patchFontFamily())
					})
				})
			}
		})
	}
	settingsHelp(c, a.t("shortcutsHelp"))
}

func (a *app) aboutCard(c *ui.Context) {
	a.settingsCard(c, "SuperGit", func() {
		settingsFormRow(c, a.t("settingsVersion"), func() {
			ui.Row(c).MinWidth(0).Selectable().Children(func() {
				display.Text(c, currentAppVersionLabel(), display.TextProps{Code: true, FontSize: scaledFont(12)}).Font(a.patchFontFamily())
			})
		})
		if path, err := settingsPath(); err == nil {
			settingsFormRow(c, a.t("settingsPath"), func() {
				ui.Row(c).FillWidth().MinWidth(0).Selectable().Children(func() {
					display.Text(c, path, display.TextProps{Code: true, FontSize: scaledFont(11)}).Font(a.patchFontFamily()).FillWidth().Tooltip(path)
				})
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
