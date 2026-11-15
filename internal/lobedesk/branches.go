package lobedesk

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func (a *app) branchesPage(c *ui.Context) {
	tk := theme.TokensOf(c)
	// A search that hides the selected row moves the selection to the first
	// match, so Enter never acts on a row the list no longer shows.
	a.syncBranchSelection()
	// A new search lists different rows, so it starts at the top; otherwise
	// the list picks up where the reader left it.
	if a.branchesScrollQuery != a.branchQuery {
		a.branchesScrollQuery = a.branchQuery
		a.branchesScroll.Y = 0
	}
	branches := a.filteredBranches()
	a.pageHeader(c, a.t("branches"), a.branchesSubtitle(
		len(branches), len(a.filteredRemoteBranches()), len(a.filteredTags()),
		a.branchesErr != "", a.remoteBranchesErr != "", a.tagsErr != ""), func() {
		ui.Box(c).Width(260).Shrink(0).Children(func() {
			field.SearchBar(c, &a.branchQuery, field.SearchBarProps{
				Placeholder: a.t("searchBranches"),
				ShortKey:    "mod+f",
				Size:        theme.Small,
				Variant:     field.InputFilled,
				// Enter while the search holds the focus acts on the row the
				// search selected, as entering the list would.
				OnSearch: func(query string) {
					a.branchQuery = query
					a.syncBranchSelection()
					a.activateBranchSelection()
					a.invalidate()
				},
			})
		})
		button := action.Button(c, a.t("newBranchAction"), action.ButtonProps{Variant: action.Default, Size: theme.Small}).
			Disabled(a.busy || a.loading || a.repo == nil || a.status.Head == "")
		if button.Clicked() {
			a.promptNewBranch(a.status.Head)
		}
		// A detached HEAD has no branch name; an empty pill would name nothing,
		// while the history page's first row and the title bar both say "detached".
		if a.repo != nil {
			branch := a.status.Branch
			if branch == "" {
				branch = a.t("detached")
			}
			a.statusTag(c, branch, display.TagProps{
				Variant: display.TagFilled, Color: display.TagSuccess, Size: theme.Small, Round: true,
			})
		}
	})
	row := ui.Row(c).Grow(1).Basis(0).MinHeight(0).AlignItems(ui.Stretch)
	// The row has no width on the page's first frame. The opening room stands
	// in so the two panes get their default widths instead of a zero-width
	// list; from the second frame the drag divides the real width.
	room := row.Bounds().W
	if room <= 0 {
		// Reuse the last measured row when the page is rebuilt, so switching
		// back to it does not open with the placeholder width for one frame.
		room = a.branchSplit.rowWidth
	}
	if room <= 0 {
		room = branchOpeningRoom
	}
	a.branchSplit.rowWidth = room
	row.Children(func() {
		list := ui.Column(c).Grow(a.branchShare()).Basis(0).MinWidth(0).AlignSelf(ui.Stretch).Focusable()
		// The border always takes the same room: a thin ring shows where the
		// arrows act once Tab left the search, as Sublime Merge outlines the
		// focused list, without the rows shifting when the focus moves.
		ring := ui.Transparent
		if list.FocusWithin() {
			ring = tk.Blue.Alpha(0.35)
		}
		list.Border(1, ring)
		list.Children(func() {
			ui.Row(c).Height(34).Shrink(0).Padding(0, 16).Gap(8).AlignItems(ui.Center).
				BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
				display.Text(c, a.t("branchHelp"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)})
			})
			layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.branchesScroll}, func() {
				ui.Column(c).FillWidth().Padding(10).Gap(2).Children(func() {
					querying := strings.TrimSpace(a.branchQuery) != ""
					refs := a.filteredRemoteBranches()
					tags := a.filteredTags()
					failed := a.branchesErr != "" || a.remoteBranchesErr != "" || a.tagsErr != ""
					if querying && len(branches) == 0 && len(refs) == 0 && len(tags) == 0 && !failed {
						display.Text(c, a.t("searchBranchesNoMatch"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12)}).Padding(12)
						return
					}
					// Each ref group owns its failure, under its own title: two
					// bare "Retry" rows stacked at the top of the page gave no
					// way to tell which read had failed when the other group
					// still opened its retry too.
					if a.branchesErr != "" {
						a.refSectionHeader(c, a.t("localBranches"), 0, false)
						a.failedWithNamedRetry(c, a.branchesErr, 12, a.t("retryLocalBranches"))
					} else {
						// A failed read leaves the list empty without meaning
						// "no branches": the error says why, and offering to
						// create one here would mislead.
						if len(branches) == 0 && !querying {
							sectionEmptyState(c, a.t("noBranches"), a.t("noBranchesHint"), 12)
						}
						if len(branches) > 0 {
							// The rows are one list, and each says its place in it:
							// a reader hears "3 of 12" with the chosen branch.
							ui.Column(c).FillWidth().Gap(2).Role(ui.RoleList).Children(func() {
								for i, branch := range branches {
									a.branchRow(c, branch, i+1, len(branches))
								}
							})
						}
					}
					a.remoteBranchSection(c, refs)
					a.tagSection(c, tags)
				})
			}).Grow(1).Basis(0).MinHeight(0)
		})
		a.splitHandle(c, &a.branchSplit, a.branchDefaultShare(),
			branchSplitMinList, branchSplitMinDetail, a.t("resizeBranchPanes"))
		a.branchDetail(c)
	})
}

func (a *app) remoteBranchSection(c *ui.Context, refs []gitrepo.Branch) {
	if a.remoteBranchesErr != "" {
		a.refSectionHeader(c, a.t("remoteBranches"), 0, false)
		a.failedWithNamedRetry(c, a.remoteBranchesErr, 12, a.t("retryRemoteBranches"))
		return
	}
	if len(refs) == 0 {
		// An empty section under a search is noise: the query already scoped
		// the page, so a "0" header only adds a line to read past.
		if strings.TrimSpace(a.branchQuery) == "" {
			a.refSectionHeader(c, a.t("remoteBranches"), 0, false)
			sectionEmptyState(c, a.t("noRemoteBranches"), a.t("noRemoteBranchesHint"), 11)
		}
		return
	}
	a.refSectionHeader(c, a.t("remoteBranches"), len(refs), true)
	ui.Column(c).FillWidth().Gap(2).Role(ui.RoleList).Children(func() {
		for i, ref := range refs {
			a.remoteBranchRow(c, ref, i+1, len(refs))
		}
	})
}

// refSectionHeader titles one of the branch page's ref groups; the count is
// left out where a section is shown empty or failed to load.
func (a *app) refSectionHeader(c *ui.Context, title string, count int, showCount bool) {
	ui.Row(c).FillWidth().Height(32).Margin(0, 12, 0, 2).PaddingX(4).Gap(6).AlignItems(ui.Center).Children(func() {
		display.Text(c, title, display.TextProps{Type: display.TextSecondary, Strong: true, FontSize: scaledFont(11)})
		if showCount {
			display.Text(c, fmt.Sprint(count), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)})
		}
	})
}

func (a *app) remoteBranchRow(c *ui.Context, ref gitrepo.Branch, pos, size int) {
	tk := theme.TokensOf(c)
	remote, name := a.remoteBranchParts(ref.Name)
	selected := a.branchSel == "remote:"+ref.Name
	// The local and tag rows are options of the same list; the remote row is
	// one too, so Tab reaches it, the menu key opens its menu and a screen
	// reader hears which remote branch the list has chosen. Its name is the
	// short branch name the row draws first; the tooltip keeps the full ref.
	row := ui.ButtonBase(c).Label(name).Row().FillWidth().MinHeight(46).Padding(5, 8).
		Gap(10).AlignItems(ui.Center).Radius(tk.BorderRadius).Border(1, ui.Transparent).
		Role(ui.RoleOption).AccessSelected(selected).PositionInSet(pos, size)
	switch {
	case selected:
		row.Background(selectionBg(tk, tk.BgContainer))
	case row.Hovered():
		row.Background(tk.FillTertiary)
	}
	ground := graphNodeGround(tk, selected, row.Hovered())
	row.Children(func() {
		lane := refLaneColor(a.lanePalette(tk), ref.Name)
		ui.Box(c).Size(30, 30).Shrink(0).Radius(9).Center().
			Background(tk.FillTertiary).Children(func() {
			ui.Icon(c, lucide.Cloud()).FontSize(scaledFont(15)).TextColor(laneInk(tk, lane))
		})
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(2).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				display.Text(c, name, display.TextProps{FontSize: scaledFont(12)}).SingleLine().AccessHidden(true)
				if remote != "" {
					// The remote's name is a plain label, so its capsule stays
					// neutral; the lane color already belongs to the icon.
					refPill(c, neutralCapsule(tk, ground), nil, remote)
				}
			})
			// The second line used to repeat the full ref the first line
			// already spells out. What the row cannot show on its own is the
			// local branch that tracks this one, the mirror of the local rows'
			// upstream line.
			if tracker := a.trackingBranch(ref.Name); tracker != "" {
				display.Text(c, a.tf("trackedBy", tracker), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(10)}).SingleLine()
			}
		})
		display.Text(c, shortSHA(ref.SHA), display.TextProps{Code: true, FontSize: scaledFont(10), Type: display.TextSecondary}).Shrink(0)
	})
	row.Tooltip(ref.Name)
	row.OnShortcut(0, ui.KeyEnter, func() { a.activateBranchSelection() })
	// Space is how a button activates, so without a claim it would only
	// reselect the row the pointer chose and undo the arrow selection.
	row.OnShortcut(0, ui.KeySpace, func() { a.activateBranchSelection() })
	row.OnClick(func() { a.selectBranchKey("remote:" + ref.Name) })
	// A double click goes through the same checkout prompt as Enter on the
	// selected remote row.
	if row.DoubleClicked() {
		a.selectBranchKey("remote:" + ref.Name)
		a.activateBranchSelection()
	}
	a.revealBranchRow(row, selected)
	a.remoteBranchMenu(c, row, ref)
	// Opening the row's menu also selects the row, so Enter and the menu act
	// on the same branch.
	if *ui.Local(row, "contextMenu", func() bool { return false }) {
		a.selectBranchKey("remote:" + ref.Name)
	}
}

// trackingBranch names the local branch that tracks a remote branch, so a
// remote row can show which local work it feeds. Several local branches may
// track the same ref; the first in git's sorted branch list wins, which keeps
// the row stable between reloads.
func (a *app) trackingBranch(remoteRef string) string {
	for _, branch := range a.branches {
		if branch.Upstream == remoteRef {
			return branch.Name
		}
	}
	return ""
}

func (a *app) remoteBranchParts(full string) (string, string) {
	best := ""
	for _, remote := range a.remotes {
		if strings.HasPrefix(full, remote.Name+"/") && len(remote.Name) > len(best) {
			best = remote.Name
		}
	}
	if best != "" {
		return best, strings.TrimPrefix(full, best+"/")
	}
	if i := strings.IndexByte(full, '/'); i > 0 {
		return full[:i], full[i+1:]
	}
	return "", full
}

func (a *app) tagSection(c *ui.Context, tags []gitrepo.Tag) {
	if a.tagsErr != "" {
		// The tag read failed, so the section keeps its title and says so:
		// moving the error above the local branches buried which group it
		// belonged to.
		a.refSectionHeader(c, a.t("tags"), 0, false)
		a.failedWithNamedRetry(c, a.tagsErr, 12, a.t("retryTags"))
		return
	}
	if len(tags) == 0 {
		if strings.TrimSpace(a.branchQuery) == "" {
			a.refSectionHeader(c, a.t("tags"), 0, false)
			sectionEmptyState(c, a.t("noTags"), a.t("noTagsHint"), 11)
		}
		return
	}
	a.refSectionHeader(c, a.t("tags"), len(tags), true)
	ui.Column(c).FillWidth().Gap(2).Role(ui.RoleList).Children(func() {
		for i, tag := range tags {
			a.tagRow(c, tag, i+1, len(tags))
		}
	})
}

// filteredBranches narrows the local, remote and tag lists with one case
// insensitive search shared by the branches page.
func (a *app) filteredBranches() []gitrepo.Branch {
	query := strings.ToLower(strings.TrimSpace(a.branchQuery))
	if query == "" {
		return a.branches
	}
	var out []gitrepo.Branch
	for _, branch := range a.branches {
		if strings.Contains(strings.ToLower(branch.Name), query) || strings.Contains(strings.ToLower(branch.Upstream), query) {
			out = append(out, branch)
		}
	}
	return out
}

func (a *app) filteredRemoteBranches() []gitrepo.Branch {
	query := strings.ToLower(strings.TrimSpace(a.branchQuery))
	if query == "" {
		return a.remoteBranches
	}
	var out []gitrepo.Branch
	for _, ref := range a.remoteBranches {
		if strings.Contains(strings.ToLower(ref.Name), query) {
			out = append(out, ref)
		}
	}
	return out
}

func (a *app) filteredTags() []gitrepo.Tag {
	query := strings.ToLower(strings.TrimSpace(a.branchQuery))
	if query == "" {
		return a.tags
	}
	var out []gitrepo.Tag
	for _, tag := range a.tags {
		if strings.Contains(strings.ToLower(tag.Name), query) || strings.Contains(strings.ToLower(tag.Commit.Subject), query) {
			out = append(out, tag)
		}
	}
	return out
}

func (a *app) tagRow(c *ui.Context, tag gitrepo.Tag, pos, size int) {
	tk := theme.TokensOf(c)
	selected := a.branchSel == "tag:"+tag.Name
	row := ui.ButtonBase(c).Label(tag.Name).Row().FillWidth().MinHeight(42).Padding(5, 8).
		Gap(10).Radius(tk.BorderRadius).AlignItems(ui.Center).Tooltip(tag.Name).
		Role(ui.RoleOption).AccessSelected(selected).PositionInSet(pos, size)
	// The row is an option of the list, so Tab, Shift+F10 and the menu key
	// reach it, and the selected tag is announced as the list's choice. A
	// focused option also takes Enter and Space for itself, which would only
	// reselect the row; the row's own shortcuts claim those keys first and run
	// the page's action, so mouse and keyboard paths agree.
	row.OnShortcut(0, ui.KeyEnter, func() { a.activateBranchSelection() })
	row.OnShortcut(0, ui.KeySpace, func() { a.activateBranchSelection() })
	if selected {
		row.Background(selectionBg(tk, tk.BgContainer))
	} else if row.Hovered() {
		row.Background(tk.FillTertiary)
	}
	ground := graphNodeGround(tk, selected, row.Hovered())
	row.Children(func() {
		lane := refLaneColor(a.lanePalette(tk), tag.Name)
		ui.Box(c).Size(28, 28).Shrink(0).Radius(8).Center().
			Background(tk.FillTertiary).Children(func() {
			ui.Icon(c, lucide.Tag()).FontSize(scaledFont(14)).TextColor(laneInk(tk, lane))
		})
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(2).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				display.Text(c, tag.Name, display.TextProps{FontSize: scaledFont(12)}).SingleLine().AccessHidden(true)
				if tag.ObjectType == "tag" {
					// An annotated tag is a Git object, not a point on a branch,
					// so it wears the tag family's gold rather than the row's
					// lane color.
					refPill(c, refCapsuleColors(tk.Gold, ground), nil, a.t("annotatedTag"))
				}
			})
			detail := shortSHA(tag.TargetSHA)
			if tag.Commit.Subject != "" {
				detail += " · " + tag.Commit.Subject
			}
			display.Text(c, detail, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(10)}).SingleLine()
		})
		// The trailing type only earns its place when the tag does not point
		// at a commit: repeating "commit" on every row is noise, and the raw
		// Git object name would read as an untranslated string.
		if tag.TargetType != "" && tag.TargetType != "commit" {
			display.Text(c, a.tagTargetLabel(tag.TargetType), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(10)}).Shrink(0)
		}
	})
	if row.Clicked() {
		a.selectBranchKey("tag:" + tag.Name)
	}
	// Double-clicking a tag opens its commit, as Enter on the selected row
	// does.
	if row.DoubleClicked() {
		a.selectBranchKey("tag:" + tag.Name)
		a.activateBranchSelection()
	}
	a.revealBranchRow(row, selected)
	a.tagMenu(c, row, tag)
	// Opening the row's menu also selects the row, so Enter and the menu act
	// on the same tag.
	if *ui.Local(row, "contextMenu", func() bool { return false }) {
		a.selectBranchKey("tag:" + tag.Name)
	}
}

// tagTargetLabel names the Git object a tag points at when it is not a
// commit. An unknown type keeps its raw name so a new object kind is visible
// instead of being hidden.
func (a *app) tagTargetLabel(kind string) string {
	switch kind {
	case "tree":
		return a.t("tagTargetTree")
	case "blob":
		return a.t("tagTargetBlob")
	}
	return kind
}

// branchKeys lists local, remote and tag rows in render order so the arrow
// keys can walk one continuous selection across the three sections.
func (a *app) branchKeys() []string {
	var keys []string
	for _, branch := range a.filteredBranches() {
		keys = append(keys, "local:"+branch.Name)
	}
	for _, ref := range a.filteredRemoteBranches() {
		keys = append(keys, "remote:"+ref.Name)
	}
	for _, tag := range a.filteredTags() {
		keys = append(keys, "tag:"+tag.Name)
	}
	return keys
}

func (a *app) moveBranchSelection(delta int) {
	keys := a.branchKeys()
	if len(keys) == 0 {
		return
	}
	index := -1
	for i, key := range keys {
		if key == a.branchSel {
			index = i
			break
		}
	}
	if index < 0 {
		index = listStartIndex(delta, len(keys))
	} else {
		index = clampIndex(index+delta, len(keys))
	}
	if keys[index] == a.branchSel {
		return
	}
	a.branchSel = keys[index]
	a.revealBranch = true
	a.branchDetailScroll.Y = 0
	a.invalidate()
}

// syncBranchSelection keeps the selection inside the rows the page lists: a
// branch that was deleted, merged away or renamed leaves no row to highlight,
// so the selection moves to the first row instead of pointing at a name that
// is gone (Enter would then do nothing). An empty search keeps the selection
// where it was; no rows at all only clear it while a search hides everything,
// because a reload briefly empties the list too.
func (a *app) syncBranchSelection() {
	searching := strings.TrimSpace(a.branchQuery) != ""
	keys := a.branchKeys()
	if len(keys) == 0 {
		if searching {
			a.branchSel = ""
			a.revealBranch = false
			a.branchDetailScroll.Y = 0
		}
		return
	}
	for _, key := range keys {
		if key == a.branchSel {
			return
		}
	}
	if !searching && a.branchSel == "" {
		return
	}
	a.branchSel = keys[0]
	a.revealBranch = false
	// The details describe one row; a different row starts its own scroll.
	a.branchDetailScroll.Y = 0
}

// selectBranchKey marks a row selected from a click. The keyboard reveals a
// selection it just moved; a click needs no scroll.
func (a *app) selectBranchKey(key string) {
	if a.branchSel == key {
		return
	}
	a.branchSel = key
	a.revealBranch = false
	a.branchDetailScroll.Y = 0
	a.invalidate()
}

// activateBranchSelection runs the default action of the selected row, as
// Enter does in Sublime Merge: check out a local branch, check out and track
// a remote one, or open a tag's commit.
func (a *app) activateBranchSelection() {
	if a.repo == nil || a.busy || a.loading || a.page != "branches" || a.branchSel == "" {
		return
	}
	if a.confirm.open || a.prompt.open || a.blameOpen || a.worktreeForm.open || a.reset.open {
		return
	}
	kind, name, ok := strings.Cut(a.branchSel, ":")
	if !ok {
		return
	}
	// The search may have hidden the selected row since it was chosen.
	visible := false
	for _, key := range a.branchKeys() {
		if key == a.branchSel {
			visible = true
			break
		}
	}
	if !visible {
		return
	}
	switch kind {
	case "local":
		for _, branch := range a.branches {
			if branch.Name != name {
				continue
			}
			if !branch.Current {
				a.switchBranch(branch.Name)
			}
			return
		}
	case "remote":
		_, short := a.remoteBranchParts(name)
		a.promptCheckoutRemoteBranch(name, short)
	case "tag":
		for _, tag := range a.tags {
			if tag.Name != name {
				continue
			}
			if tag.TargetSHA != "" && tag.TargetType == "commit" {
				a.openPage("history")
				a.showCommit(tag.TargetSHA)
			}
			return
		}
	}
}

// revealBranchRow scrolls the keyboard selection into view once per move;
// mouse selection never raises the flag, so lists do not jump under the cursor.
func (a *app) revealBranchRow(row ui.Element, selected bool) {
	if !selected || !a.revealBranch {
		return
	}
	row.ScrollIntoView()
	a.revealBranch = false
}

func (a *app) branchRow(c *ui.Context, branch gitrepo.Branch, pos, size int) {
	tk := theme.TokensOf(c)
	selected := a.branchSel == "local:"+branch.Name
	// The row is a choice of the branch list, so it takes the keyboard
	// focus, the menu key and Shift+F10 the way the tag rows do. It reports
	// itself as an option rather than a plain button: the list picks one row
	// at a time, and only an option can say which one through aria-selected.
	// The shortcut claims Enter before the button does, so Enter on a focused
	// row checks the branch out instead of only reselecting it.
	row := ui.ButtonBase(c).Label(branch.Name).Row().FillWidth().MinHeight(46).Padding(5, 8).
		Gap(10).AlignItems(ui.Center).Radius(tk.BorderRadius).Border(1, ui.Transparent).
		Role(ui.RoleOption).AccessSelected(selected).PositionInSet(pos, size)
	if branch.Current {
		// The "当前" chip says it on screen; a screen reader gets it after
		// the name, so the checked-out branch is never just another option.
		row.Description(a.t("current"))
	}
	row.OnShortcut(0, ui.KeyEnter, func() {
		// The shortcut covers the row's focus subtree, and the switch button
		// sits inside it. Focus on that button means Enter presses it: the
		// branch it names, not the row the list has selected. Space already
		// presses it there, so letting Enter do anything else would make the
		// two keys disagree on the same focused control.
		if !row.Focused() {
			if !branch.Current && !a.busy && !a.loading {
				a.switchBranch(branch.Name)
			}
			return
		}
		a.activateBranchSelection()
	})
	// Space activates a button, so the row has to claim it too: left to the
	// row button, Space would only reselect the last-clicked row. The claim is
	// only taken while the row itself holds the focus, so the switch button
	// inside it keeps Space to press it, as the Enter comment above describes.
	if row.Focused() {
		row.OnShortcut(0, ui.KeySpace, func() { a.activateBranchSelection() })
	}
	// Space activates a button, so the row has to claim it too: left to the
	// row button, Space would only reselect the last-clicked row. The claim is
	// only taken while the row itself holds the focus, so the switch button
	// inside it keeps Space to press it, as the Enter comment above describes.
	// Selection comes first so the row under the keyboard always reads as
	// selected: the checked-out branch keeps its own tint only while the
	// selection is elsewhere, and its "当前" chip marks it either way.
	switch {
	case selected:
		row.Background(selectionBg(tk, tk.BgContainer))
	case branch.Current:
		row.Background(tk.FillSecondary)
	case row.Hovered():
		row.Background(tk.FillTertiary)
	}
	row.Children(func() {
		lane := refLaneColor(a.lanePalette(tk), branch.Name)
		// The tile stays neutral: the lane color lives in the icon, and a
		// tile tinted toward it would lower the icon's contrast.
		ui.Box(c).Size(30, 30).Shrink(0).Radius(9).Center().
			Background(tk.FillTertiary).Children(func() {
			ui.Icon(c, lucide.GitBranch()).FontSize(scaledFont(15)).TextColor(laneInk(tk, lane))
		})
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(2).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				display.Text(c, branch.Name, display.TextProps{FontSize: scaledFont(12), Strong: branch.Current}).SingleLine().AccessHidden(true)
				if branch.Current {
					a.statusTag(c, a.t("current"), display.TagProps{
						Variant: display.TagFilled, Color: display.TagSuccess, Size: theme.Small, Round: true,
					})
				}
			})
			upstream := branch.Upstream
			switch {
			case upstream == "":
				upstream = a.t("upstream") + " —"
			case branch.UpstreamGone:
				upstream += " · " + a.t("upstreamGone")
			default:
				// Ahead/behind counts match Sublime Merge's branch list, so a
				// branch that needs a pull or push is visible without opening
				// the menu.
				var counts []string
				if branch.Ahead > 0 {
					counts = append(counts, fmt.Sprintf("↑%d", branch.Ahead))
				}
				if branch.Behind > 0 {
					counts = append(counts, fmt.Sprintf("↓%d", branch.Behind))
				}
				if len(counts) > 0 {
					upstream += " · " + strings.Join(counts, " ")
				}
			}
			display.Text(c, upstream, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(10)}).SingleLine()
		})
		display.Text(c, shortSHA(branch.SHA), display.TextProps{Code: true, FontSize: scaledFont(10), Type: display.TextSecondary}).Shrink(0)
		if !branch.Current {
			button := action.Button(c, a.t("switch"), action.ButtonProps{Variant: action.Default, Size: theme.Small}).
				Disabled(a.busy || a.loading)
			if button.Clicked() {
				a.switchBranch(branch.Name)
			}
		}
	})
	row.OnClick(func() { a.selectBranchKey("local:" + branch.Name) })
	// A double click checks the branch out, as Enter on the selected row
	// does in Sublime Merge; the current branch has nothing to switch to.
	if row.DoubleClicked() && !branch.Current {
		a.switchBranch(branch.Name)
	}
	a.revealBranchRow(row, selected)
	a.branchMenu(c, row, branch)
	// Opening the row's menu also selects the row, so Enter and the menu act
	// on the same branch.
	if *ui.Local(row, "contextMenu", func() bool { return false }) {
		a.selectBranchKey("local:" + branch.Name)
	}
}
