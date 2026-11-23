package lobedesk

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

type childNavigationState struct {
	open                              bool
	root, source, query               string
	epoch, generation                 uint64
	reload, cursor                    int
	cursorQuery                       string
	cursorPlaced, reveal, searchFocus bool
}

// The ordinary stream is a topological prefix of fixed tips. If source occurs
// in that prefix, every in-scope direct child precedes it. UI search and file
// --follow history cannot be used to infer these relationships.
func (a *app) childNavigationCandidates(source string) []gitrepo.Commit {
	found := false
	for _, commit := range a.commits {
		if commit.SHA == source {
			found = true
			break
		}
	}
	if !found || a.fileFilter != "" {
		return nil
	}
	var result []gitrepo.Commit
	for _, commit := range a.commits {
		for _, parent := range commit.Parents {
			if parent == source {
				result = append(result, commit)
				break
			}
		}
	}
	return result
}

func (a *app) childNavigationSource() string {
	if a.repo == nil || a.page != "history" || a.worktreeSelected || a.exclusiveBusy() || a.loading || a.historyReloading || a.repositoryChooserOpen || a.historyRefOpen {
		return ""
	}
	for _, commit := range a.historyRows() {
		if commit.SHA == a.selectedSHA {
			return commit.SHA
		}
	}
	return ""
}

func (a *app) canNavigateToChild() bool {
	return a.childNavigationSource() != "" && !a.hasPendingDialog(toastMain)
}

func (a *app) navigateToChild() {
	if !a.canNavigateToChild() {
		return
	}
	source := a.selectedSHA
	present := false
	for _, commit := range a.commits {
		if commit.SHA == source {
			present = true
			break
		}
	}
	if a.fileFilter != "" || !present {
		a.queue("warning", a.t("jumpToChild"), a.t("childNavigationIncomplete"))
		a.invalidate()
		return
	}
	children := a.childNavigationCandidates(source)
	switch len(children) {
	case 0:
		a.queue("warning", a.t("jumpToChild"), a.t("childNavigationNone"))
	case 1:
		a.jumpWithHistory(children[0].SHA)
	default:
		a.childNavigationGeneration++
		a.childNavigation = childNavigationState{open: true, root: a.repo.Root, source: source, epoch: a.loadSeq, reload: a.historyReload, generation: a.childNavigationGeneration, cursor: -1, reveal: true, searchFocus: true}
	}
	a.invalidate()
}

func (a *app) discardStaleChildNavigation() {
	s := a.childNavigation
	if s.open && (a.repo == nil || a.repo.Root != s.root || a.loadSeq != s.epoch || a.historyReload != s.reload || a.childNavigationSource() != s.source || a.fileFilter != "") {
		a.childNavigation = childNavigationState{}
	}
}

func (a *app) chooseChildNavigation(sha string, generation uint64) {
	a.discardStaleChildNavigation()
	s := a.childNavigation
	if !s.open || s.generation != generation {
		return
	}
	for _, commit := range a.childNavigationCandidates(s.source) {
		if commit.SHA == sha {
			a.childNavigation = childNavigationState{}
			if a.canNavigateToChild() {
				a.jumpWithHistory(sha)
			}
			a.invalidate()
			return
		}
	}
}

func (a *app) childNavigationDialog(c *ui.Context) {
	a.discardStaleChildNavigation()
	if !a.childNavigation.open {
		return
	}
	var matches []gitrepo.Commit
	query := strings.ToLower(strings.TrimSpace(a.childNavigation.query))
	for _, commit := range a.childNavigationCandidates(a.childNavigation.source) {
		if strings.Contains(strings.ToLower(commit.SHA+" "+commit.Subject), query) {
			matches = append(matches, commit)
		}
	}
	first := func() int {
		if len(matches) > 0 {
			return 0
		}
		return -1
	}
	if placePicker(&a.childNavigation.cursor, &a.childNavigation.cursorQuery, &a.childNavigation.cursorPlaced, a.childNavigation.query, first) || repairPicker(&a.childNavigation.cursor, len(matches), first) {
		a.childNavigation.reveal = true
	}
	tk := theme.TokensOf(c)
	width, height := c.Size()
	generation := a.childNavigation.generation
	var search ui.Element
	chosen := ""
	ui.DialogBase(c, &a.childNavigation.open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .4))
		panel.Column().Label(a.t("childNavigationTitle")).Width(min(520, max(0, width-32))).MaxHeight(max(0, height-32)).Padding(16).Gap(10).Background(tk.BgElevated).Radius(tk.BorderRadiusLG).Border(1, tk.Border).Children(func() {
			display.Text(c, a.t("childNavigationTitle"), display.TextProps{Strong: true, FontSize: scaledFont(14)})
			display.Text(c, a.t("childNavigationScope"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11), Rows: 2})
			search = a.pickerSearchBar(c, &a.childNavigation.query, a.t("childNavigationSearch"), func() {
				i := a.childNavigation.cursor
				if i >= 0 && i < len(matches) {
					chosen = matches[i].SHA
				}
			}, func(delta int) {
				a.childNavigation.cursor = stepPickerCursor(len(matches), a.childNavigation.cursor, delta, func(int) bool { return true })
				a.childNavigation.reveal = true
			}).Label(a.t("childNavigationSearch"))
			var active ui.Element
			hasActive := false
			layout.ScrollArea(c, layout.ScrollAreaProps{}, func() {
				ui.Column(c).FillWidth().Gap(1).Role(ui.RoleList).Children(func() {
					for i, commit := range matches {
						highlighted := i == a.childNavigation.cursor
						row := pickerRowBase(c, commit.SHA+" "+commit.Subject, i+1, len(matches), highlighted)
						if highlighted {
							row.Background(selectionBg(tk, tk.BgElevated))
						} else if row.Hovered() {
							row.Background(tk.FillTertiary.Over(tk.BgElevated))
						}
						row.Children(func() {
							ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(2).Children(func() {
								display.Text(c, commit.Subject, display.TextProps{FontSize: scaledFont(12)}).SingleLine()
								display.Text(c, shortSHA(commit.SHA), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).SingleLine()
							})
						})
						pickerReveal(row, highlighted, &a.childNavigation.reveal)
						if highlighted {
							active, hasActive = row, true
						}
						if row.Clicked() {
							chosen = commit.SHA
						}
					}
					if len(matches) == 0 {
						display.Text(c, a.t("searchNoMatch"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12)}).Padding(6, 8)
					}
				})
			}).MaxHeight(min(float32(pickerListMaxHeight), max(0, height-190)))
			if hasActive {
				search.ActiveDescendant(active)
			}
		})
	})
	if a.childNavigation.open && a.childNavigation.searchFocus {
		search.Focus()
		a.childNavigation.searchFocus = false
	}
	if chosen != "" {
		a.chooseChildNavigation(chosen, generation)
		return
	}
	if !a.childNavigation.open {
		a.childNavigation = childNavigationState{}
	}
}

// Copy the established first-parent keyboard policy, including its input guard.
func childNavigationShortcut(parent appShortcut) appShortcut {
	parent.ID = "history.child"
	parent.Key = ui.KeyUp
	parent.Keys = "Alt+↑"
	parent.Label = "jumpToChild"
	parent.Available = func(a *app) bool { return a.canNavigateToChild() }
	parent.Run = func(a *app) { a.navigateToChild() }
	return parent
}
