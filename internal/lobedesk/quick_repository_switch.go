package lobedesk

import (
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type quickRepositorySwitchState struct {
	open, tabsOnly                    bool
	root                              string
	epoch, generation                 uint64
	query                             string
	cursor                            int
	cursorQuery                       string
	cursorPlaced, reveal, searchFocus bool
}

func quickRepositorySwitchShortcuts() []appShortcut {
	keys := "Ctrl+Shift+O"
	if runtime.GOOS == "darwin" {
		keys = "⌘⇧O"
	}
	primary := appShortcut{ID: "repo.quickSwitch", Mod: ui.Cmd | ui.Shift, Key: ui.KeyO, Keys: keys, Group: shortcutGroupGlobal, Label: "quickRepositorySwitchCommand", Available: func(a *app) bool { return a.canOpenQuickRepositorySwitch() }, Run: func(a *app) { a.openQuickRepositorySwitch() }}
	result := []appShortcut{primary}
	alias := primary
	alias.ID = "repo.quickSwitchAlias"
	alias.Key = ui.KeyP
	switch runtime.GOOS {
	case "darwin":
		alias.Mod = ui.Cmd | ui.Ctrl
		alias.Keys = "⌘⌃P"
		result = append(result, alias)
	case "linux":
		alias.Mod = ui.Cmd | ui.Alt
		alias.Keys = "Ctrl+Alt+P"
		result = append(result, alias)
	}
	return result
}

func (a *app) quickRepositorySwitchRoot() string {
	if a.repo != nil {
		return a.repo.Root
	}
	return ""
}

func (a *app) discardStaleQuickRepositorySwitch() {
	s := a.quickRepositorySwitch
	if s.open && (a.quickRepositorySwitchRoot() != s.root || a.loadSeq != s.epoch) {
		a.quickRepositorySwitch = quickRepositorySwitchState{}
	}
}

func (a *app) canOpenRepositoryPicker() bool {
	a.discardStaleQuickRepositorySwitch()
	if tabs := a.repositoryTabs; tabs != nil && (tabs.active != a || !tabs.canChange()) {
		return false
	}
	return !a.quickRepositorySwitch.open && !a.busy && !a.loading && !a.historyReloading && !a.repositoryChooserOpen && !a.historyRefOpen && !a.hasPendingDialog(toastMain)
}

func (a *app) canOpenQuickRepositorySwitch() bool {
	return a.canOpenRepositoryPicker() && (a.repo != nil || len(a.quickRepositorySwitchCandidates()) > 0)
}

func (a *app) openQuickRepositorySwitch() {
	if a.canOpenQuickRepositorySwitch() {
		a.openRepositoryPicker(false)
	}
}

func (a *app) openRepositoryPicker(tabsOnly bool) {
	a.quickRepositorySwitchGeneration++
	a.quickRepositorySwitch = quickRepositorySwitchState{open: true, tabsOnly: tabsOnly, root: a.quickRepositorySwitchRoot(), epoch: a.loadSeq, generation: a.quickRepositorySwitchGeneration, cursor: -1, reveal: true, searchFocus: true}
	a.invalidate()
}

func (a *app) quickRepositorySwitchCandidates() []string {
	if a.repo == nil && a.repositoryTabs == nil {
		return nil
	}
	var result []string
	query := strings.ToLower(strings.TrimSpace(a.quickRepositorySwitch.query))
	seen := map[string]bool{"": true, a.quickRepositorySwitchRoot(): true}
	add := func(root string) {
		if !seen[root] && strings.Contains(strings.ToLower(root), query) {
			seen[root] = true
			result = append(result, root)
		}
	}
	for _, root := range a.recentRepositories {
		add(root)
	}
	if tabs := a.repositoryTabs; tabs != nil {
		for _, session := range tabs.sessions {
			if session.repo != nil {
				add(session.repo.Root)
			}
		}
	}
	return result
}

func (a *app) chooseQuickRepositorySwitch(root string, generation uint64) {
	a.discardStaleQuickRepositorySwitch()
	s := a.quickRepositorySwitch
	if !s.open || s.tabsOnly || s.generation != generation || a.busy || a.loading || a.historyReloading || a.repositoryChooserOpen || a.historyRefOpen {
		return
	}
	found := false
	for _, candidate := range a.quickRepositorySwitchCandidates() {
		if candidate == root {
			found = true
			break
		}
	}
	if !found {
		return
	}
	// Close before checking existing modal guards; never let this picker block itself.
	a.quickRepositorySwitch = quickRepositorySwitchState{}
	allowed := a.canOpenRecentRepository()
	if tabs := a.repositoryTabs; tabs != nil {
		allowed = tabs.active == a && tabs.canChange()
	}
	if allowed {
		a.repositorySwitchTarget = root
		// The current picker results have already validated the root. An open
		// tab can remain valid after eviction from the bounded Recents list.
		a.openRepository(root, true)
		a.repositorySwitchTarget = ""
	}
	a.invalidate()
}

type repositoryPickerOption struct {
	name, root string
	tab        *app
	tabIndex   int
}

func (option repositoryPickerOption) label() string {
	if option.root != "" {
		return option.root
	}
	return option.name
}

func (a *app) repositoryPickerOptions() []repositoryPickerOption {
	var options []repositoryPickerOption
	if !a.quickRepositorySwitch.tabsOnly {
		for _, root := range a.quickRepositorySwitchCandidates() {
			options = append(options, repositoryPickerOption{name: filepath.Base(root), root: root})
		}
		return options
	}
	if a.repositoryTabs == nil {
		return nil
	}
	query := strings.ToLower(strings.TrimSpace(a.quickRepositorySwitch.query))
	for index, tab := range a.repositoryTabs.sessions {
		if tab == a {
			continue
		}
		option := repositoryPickerOption{name: a.t("welcomeTab"), tab: tab, tabIndex: index}
		if tab.repo != nil {
			option.root = tab.repo.Root
			option.name = filepath.Base(option.root)
			if tab.repositoryAlias != "" {
				option.name = tab.repositoryAlias
			}
		}
		if strings.Contains(strings.ToLower(option.name), query) || strings.Contains(strings.ToLower(option.root), query) {
			options = append(options, option)
		}
	}
	return options
}

func (a *app) chooseOpenTab(tab *app, generation uint64) {
	s := a.quickRepositorySwitch
	tabs := a.repositoryTabs
	if !s.open || !s.tabsOnly || s.generation != generation || tabs == nil || tabs.active != a || !slices.Contains(tabs.sessions, tab) {
		return
	}
	a.quickRepositorySwitch = quickRepositorySwitchState{}
	if tabs.canChange() && !a.historyReloading && !a.historyRefOpen {
		tabs.focusTab = tab
		tabs.activate(tab, false)
	}
	a.invalidate()
}

func (a *app) quickRepositorySwitchDialog(c *ui.Context) {
	a.discardStaleQuickRepositorySwitch()
	if !a.quickRepositorySwitch.open {
		return
	}
	matches := a.repositoryPickerOptions()
	first := func() int {
		if len(matches) > 0 {
			return 0
		}
		return -1
	}
	s := &a.quickRepositorySwitch
	if placePicker(&s.cursor, &s.cursorQuery, &s.cursorPlaced, s.query, first) || repairPicker(&s.cursor, len(matches), first) {
		s.reveal = true
	}
	tk := theme.TokensOf(c)
	width, height := c.Size()
	generation := s.generation
	var search ui.Element
	var chosen repositoryPickerOption
	title, searchLabel := a.t("quickRepositorySwitch"), a.t("quickRepositorySwitchSearch")
	if s.tabsOnly {
		title, searchLabel = a.t("switchRepositoryTab"), a.t("searchRepositoryTabs")
	}
	c.Key(commandPickerOverlayKey)
	ui.DialogBase(c, &a.quickRepositorySwitch.open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .4))
		panel.Column().Label(title).Width(min(520, max(0, width-32))).MaxHeight(max(0, height-32)).Padding(16).Gap(10).Background(tk.BgElevated).Radius(tk.BorderRadiusLG).Border(1, tk.Border).Children(func() {
			display.Text(c, title, display.TextProps{Strong: true, FontSize: scaledFont(14)})
			if !s.tabsOnly {
				display.Text(c, a.t("quickRepositorySwitchScope"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11), Rows: 2})
			}
			search = a.pickerSearchBar(c, &s.query, searchLabel, func() {
				i := s.cursor
				if i >= 0 && i < len(matches) {
					chosen = matches[i]
				}
			}, func(delta int) {
				s.cursor = stepPickerCursor(len(matches), s.cursor, delta, func(int) bool { return true })
				s.reveal = true
			}).Label(searchLabel)
			var active ui.Element
			hasActive := false
			layout.ScrollArea(c, layout.ScrollAreaProps{}, func() {
				ui.Column(c).FillWidth().Gap(1).Role(ui.RoleList).Children(func() {
					for i, option := range matches {
						highlighted := i == s.cursor
						if option.tab != nil {
							// Use the strip position, which is stable while the picker is open.
							c.Key(option.tabIndex)
						}
						row := pickerRowBase(c, option.label(), i+1, len(matches), highlighted)
						if option.tab != nil && option.root != "" && option.name != filepath.Base(option.root) {
							row.Description(option.name)
						}
						if highlighted {
							row.Background(selectionBg(tk, tk.BgElevated))
						} else if row.Hovered() {
							row.Background(tk.FillTertiary.Over(tk.BgElevated))
						}
						row.Children(func() {
							ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(2).Children(func() {
								display.Text(c, option.name, display.TextProps{FontSize: scaledFont(12)}).SingleLine()
								if option.root != "" {
									path := ui.Text(c, option.root).FillWidth().MinWidth(0).FontSize(scaledFont(11)).TextColor(tk.TextSecondary)
									if !highlighted {
										path.SingleLine()
									}
								}
							})
						})
						pickerReveal(row, highlighted, &s.reveal)
						if highlighted {
							active, hasActive = row, true
						}
						if row.Clicked() {
							chosen = option
						}
					}
					if len(matches) == 0 {
						key := "quickRepositorySwitchNoMatch"
						if s.tabsOnly {
							key = "repositoryTabsNoMatch"
						} else if strings.TrimSpace(s.query) == "" {
							key = "quickRepositorySwitchEmpty"
						}
						display.Text(c, a.t(key), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12), Rows: 2}).Padding(6, 8)
					}
				})
			}).MaxHeight(min(float32(pickerListMaxHeight), max(0, height-190)))
			if hasActive {
				search.ActiveDescendant(active)
			}
		})
	})
	if a.quickRepositorySwitch.open && s.searchFocus {
		search.Focus()
		s.searchFocus = false
	}
	if chosen.tab != nil {
		a.chooseOpenTab(chosen.tab, generation)
		return
	}
	if chosen.root != "" {
		a.chooseQuickRepositorySwitch(chosen.root, generation)
		return
	}
	if !a.quickRepositorySwitch.open {
		a.quickRepositorySwitch = quickRepositorySwitchState{}
	}
}
