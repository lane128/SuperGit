package lobedesk

import (
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

// historyRef scopes the history list to the commits reachable from one ref,
// the way Sublime Merge's branch selector at the top of history does. The zero
// value shows every loaded ref.
type historyRef struct {
	Kind, Name string
}

const (
	historyRefCommit = "commit"
	historyRefLocal  = "local"
	historyRefRemote = "remote"
	historyRefTag    = "tag"
)

// menuCloseDelay is how long a context menu gets to finish its exit animation
// before the page that opened it is replaced. See showRefHistory.
const menuCloseDelay = 150 * time.Millisecond

func (r historyRef) key() string {
	if r.Kind == "" {
		return "all"
	}
	return r.Kind + ":" + r.Name
}

// historyRefTip resolves the ref to the commit the list should walk back
// from. Refs move between reloads, so the tip is read from the latest
// branch/tag snapshot instead of being cached with the choice.
func (a *app) historyRefTip(ref historyRef) string {
	switch ref.Kind {
	case historyRefCommit:
		return ref.Name
	case historyRefLocal:
		for _, branch := range a.branches {
			if branch.Name == ref.Name {
				return branch.SHA
			}
		}
	case historyRefRemote:
		for _, branch := range a.remoteBranches {
			if branch.Name == ref.Name {
				return branch.SHA
			}
		}
	case historyRefTag:
		for _, tag := range a.tags {
			if tag.Name == ref.Name && tag.TargetType == "commit" {
				return tag.TargetSHA
			}
		}
	}
	return ""
}

// reachableCommits keeps the commits a walk from tip can reach, preserving the
// list's order. Parents outside the loaded window are simply not in the
// result, so filtering a page never invents rows. An unknown tip, such as a
// deleted branch, falls back to the full list instead of an empty one.
func reachableCommits(commits []gitrepo.Commit, tip string) []gitrepo.Commit {
	if tip == "" {
		return commits
	}
	index := make(map[string]int, len(commits))
	for i, commit := range commits {
		index[commit.SHA] = i
	}
	if _, ok := index[tip]; !ok {
		return commits
	}
	reachable := make(map[string]bool, len(commits))
	queue := []string{tip}
	for len(queue) > 0 {
		sha := queue[0]
		queue = queue[1:]
		if reachable[sha] {
			continue
		}
		reachable[sha] = true
		i, ok := index[sha]
		if !ok {
			continue
		}
		for _, parent := range commits[i].Parents {
			if !reachable[parent] {
				queue = append(queue, parent)
			}
		}
	}
	result := make([]gitrepo.Commit, 0, len(commits))
	for _, commit := range commits {
		if reachable[commit.SHA] {
			result = append(result, commit)
		}
	}
	return result
}

func (a *app) historyRefLabel() string {
	if a.historyRef.Kind == "" {
		return a.t("allRefs")
	}
	if a.historyRef.Kind == historyRefCommit {
		return shortSHA(a.historyRef.Name)
	}
	return a.historyRef.Name
}

// refChoice is one row of the ref selector: a ref, how it is drawn, and the
// color of its line in the graph.
type refChoice struct {
	ref   historyRef
	label string
	icon  *ui.SVG
	color ui.Color
	// head marks the checked-out branch, so it can be told apart from the
	// other refs without opening the branch page.
	head bool
}

// historyRefChoices lists every ref the selector offers, in the order the
// popup shows them: all refs, local branches, remote branches, tags.
func (a *app) historyRefChoices(tk *theme.Tokens) []refChoice {
	palette := a.lanePalette(tk)
	colorOf := func(seed string) ui.Color {
		// The dot is drawn on the popover and on the selected row, both of
		// which can be light; laneInk is the same ink the graph and the
		// branch icons use, so the dot matches them exactly.
		return laneInk(tk, a.refGraphColor(palette, seed))
	}
	choices := []refChoice{{label: a.t("allRefs"), icon: lucide.Layers()}}
	for _, branch := range a.branches {
		if a.branchHidden("refs/heads/" + branch.Name) {
			continue
		}
		choices = append(choices, refChoice{
			ref:   historyRef{Kind: historyRefLocal, Name: branch.Name},
			label: branch.Name,
			icon:  lucide.GitBranch(),
			color: colorOf(branch.Name),
			head:  branch.Name == a.status.Branch,
		})
	}
	for _, branch := range a.remoteBranches {
		if a.branchHidden("refs/remotes/" + branch.Name) {
			continue
		}
		choices = append(choices, refChoice{
			ref:   historyRef{Kind: historyRefRemote, Name: branch.Name},
			label: branch.Name,
			icon:  lucide.Cloud(),
			color: colorOf(branch.Name),
		})
	}
	for _, tag := range a.tags {
		if tag.TargetType != "commit" {
			continue
		}
		choices = append(choices, refChoice{
			ref:   historyRef{Kind: historyRefTag, Name: tag.Name},
			label: tag.Name,
			icon:  lucide.Tag(),
			color: colorOf(tag.Name),
		})
	}
	return choices
}

// historyRefErr is the first ref list that failed to load. The selector says
// so next to a retry instead of showing an empty list that reads as "this
// repository has no refs"; a git failure and an empty repository must not look
// the same.
func (a *app) historyRefErr() string {
	for _, err := range []string{a.branchesErr, a.remoteBranchesErr, a.tagsErr} {
		if err != "" {
			return err
		}
	}
	return ""
}

// refCursorTarget is the row the popup highlights when its list is rebuilt:
// the ref in use when the new query still shows it, and otherwise the first
// match. -1 means the query hides every ref.
func refCursorTarget(matches []refChoice, current historyRef) int {
	for index, choice := range matches {
		if choice.ref == current {
			return index
		}
	}
	if len(matches) > 0 {
		return 0
	}
	return -1
}

// stepRefCursor moves the highlight within matches, wrapping at either end so
// a short list never traps the keys. Every ref can be chosen, so nothing is
// skipped.
func stepRefCursor(matches []refChoice, cursor, delta int) int {
	return stepPickerCursor(len(matches), cursor, delta, nil)
}

// refChoiceAt is the row Enter takes: the highlighted one, or the first match
// when no row is highlighted or the list shrank under it.
func refChoiceAt(matches []refChoice, cursor int) (refChoice, bool) {
	index := pickerCursorAt(len(matches), cursor, nil)
	if index < 0 {
		return refChoice{}, false
	}
	return matches[index], true
}

// historyRefHeaderIcon mirrors the active filter in the selector's own
// button: the same icon and graph color the chosen ref has in the list, so
// the header reads at a glance without opening the popup.
func (a *app) historyRefHeaderIcon(tk *theme.Tokens) (*ui.SVG, ui.Color) {
	palette := a.lanePalette(tk)
	switch a.historyRef.Kind {
	case historyRefLocal:
		return lucide.GitBranch(), laneInk(tk, a.refGraphColor(palette, a.historyRef.Name))
	case historyRefRemote:
		return lucide.Cloud(), laneInk(tk, a.refGraphColor(palette, a.historyRef.Name))
	case historyRefTag:
		return lucide.Tag(), laneInk(tk, a.refGraphColor(palette, a.historyRef.Name))
	}
	return lucide.GitFork(), tk.TextSecondary
}

// refGraphColor is the color the graph drew a ref's line with, when that ref is
// on screen. A name whose hash slot is already taken by another live branch
// steps to a free slot, so the hash color alone would show a dot in a color the
// line beside it does not have. Off screen -- or before the first layout -- the
// hash color is the best answer and the one the branch page shows too.
func (a *app) refGraphColor(palette []ui.Color, name string) ui.Color {
	if a.historyGraph != nil {
		if color, ok := a.historyGraph.RefColor(name); ok {
			return color
		}
	}
	return refLaneColor(palette, name)
}

// historyRefMenu is the history header's ref selector. It opens a searchable
// list instead of a menu: a long-lived repository has hundreds of tags, and
// scrolling a menu to find one is slow. Each branch and tag shows the color
// its line has in the graph, so a list entry and its lane can be matched by
// sight.
func (a *app) historyRefMenu(c *ui.Context) {
	tk := theme.TokensOf(c)
	label := a.historyRefLabel()
	anchor := ui.ButtonBase(c).Label(label)
	styleWorkbenchButton(c, anchor, buttonNormal, false, a.historyRefOpen)
	if a.historyRef.Kind != "" {
		anchor.Tooltip(label)
	}
	anchor.Gap(6).Children(func() {
		icon, color := a.historyRefHeaderIcon(tk)
		ui.Icon(c, icon).FontSize(scaledFont(14)).TextColor(color)
		ui.Text(c, a.historyRefLabel()).FontSize(scaledFont(12)).SingleLine().MaxWidth(a.historyHeaderLabelWidth())
		ui.Icon(c, lucide.ChevronDown()).FontSize(scaledFont(14)).TextColor(tk.TextSecondary)
	})
	// The palette phase owns the shared modal; the header remains its normal
	// anchor without constructing another overlay behind it.
	if a.historyRefPalette.active {
		return
	}
	wasOpen := a.historyRefOpen
	overlay.Popover(c, anchor, &a.historyRefOpen, overlay.PopoverProps{Placement: overlay.Bottom}, func() {
		if !wasOpen {
			a.initializeHistoryRefPicker()
		}
		a.historyRefPickerContent(c, 300, pickerListMaxHeight, true, func(ref historyRef) {
			a.historyRefOpen = false
			a.historyRefQuery = ""
			a.applyHistoryRef(ref)
		})
	})
}

func (a *app) initializeHistoryRefPicker() {
	a.historyRefQuery = ""
	a.historyRefCursorPlaced = false
	a.historyRefReveal = true
}

// Both entry points use the same search, choices, highlight and error recovery.
// The caller owns overlay focus so a palette handoff keeps its original opener.
func (a *app) historyRefPickerContent(c *ui.Context, width, listHeight float32, focus bool, choose func(historyRef)) ui.Element {
	tk := theme.TokensOf(c)
	q := strings.ToLower(strings.TrimSpace(a.historyRefQuery))
	choices := a.historyRefChoices(tk)
	matchesQuery := func(label string) bool {
		return q == "" || strings.Contains(strings.ToLower(label), q)
	}
	var filtered []refChoice
	for _, choice := range choices {
		if matchesQuery(choice.label) {
			filtered = append(filtered, choice)
		}
	}
	if placePicker(&a.historyRefCursor, &a.historyRefCursorQuery, &a.historyRefCursorPlaced,
		a.historyRefQuery, func() int { return refCursorTarget(filtered, a.historyRef) }) {
		a.historyRefReveal = true
	}
	// A refresh can shorten the list under a highlight already placed.
	// Put it back on a ref that still exists and scroll to it, or Enter
	// would take a row the reader cannot see.
	if repairPicker(&a.historyRefCursor, len(filtered), func() int { return refCursorTarget(filtered, a.historyRef) }) {
		a.historyRefReveal = true
	}
	shown := 0
	var search ui.Element
	ui.Column(c).Width(width).Gap(6).Children(func() {
		// Enter takes the highlighted ref, or the first match when no
		// row is marked, so a ref can be picked without the mouse.
		search = a.pickerSearchBar(c, &a.historyRefQuery, a.t("searchRefs"), func() {
			if choice, ok := refChoiceAt(filtered, a.historyRefCursor); ok {
				choose(choice.ref)
			}
		}, func(delta int) {
			a.historyRefCursor = stepRefCursor(filtered, a.historyRefCursor, delta)
			a.historyRefReveal = true
		})
		// A failed ref read leaves the list short; say why and offer the
		// same reload the rest of the app uses, instead of letting the
		// reader mistake it for a repository without branches.
		if err := a.historyRefErr(); err != "" {
			// The rest of the app says when a retry is already running;
			// this retry is a full reread too, so it must not start a
			// second one behind the first.
			retrying := a.loading || a.busy
			label := a.t("retry")
			if a.loading {
				label = a.t("retrying")
			}
			ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
				display.Text(c, err, display.TextProps{
					Type: display.TextDanger, FontSize: scaledFont(10), Rows: 2,
				}).Grow(1).Basis(0).MinWidth(0)
				button := workbenchButton(c, label, buttonNormal, retrying)
				if !retrying && button.Clicked() {
					a.reload()
				}
			})
		}
		// The box holds the keyboard while the popup is open, so Tab and
		// row clicks cannot strand the arrows and the typing; asking each
		// frame outlasts a focus that moves. While the read has failed its
		// Retry is the only useful control, so the box takes the focus once
		// and lets Tab reach it; the force returns when the error clears.
		if focus {
			if a.historyRefErr() != "" {
				search.AutoFocus()
			} else {
				search.Focus()
			}
		}
		var active ui.Element
		hasActive := false
		pickerRows(c, func() {
			group := ""
			for _, choice := range choices {
				if choice.ref.Kind != group {
					group = choice.ref.Kind
					// Headers only help in the unfiltered list; a
					// search should read as one flat result set.
					if q == "" && hasRefGroup(choices, group) {
						if title := refGroupLabel(a.t, group); title != "" {
							display.Text(c, title, display.TextProps{
								Type: display.TextSecondary, FontSize: scaledFont(10), Strong: true,
							}).Padding(6, 8, 2, 8)
						}
					}
				}
				if !matchesQuery(choice.label) {
					continue
				}
				highlighted := shown == a.historyRefCursor
				shown++
				row := a.historyRefRow(c, choice, shown, len(filtered), a.historyRef == choice.ref, highlighted, choose)
				// The box keeps the keyboard, so the row under the arrows is
				// its active descendant; Selected stays with the ref in use.
				if highlighted {
					active, hasActive = row, true
				}
			}
			if shown == 0 {
				empty := "searchBranchesNoMatch"
				if a.historyRefErr() != "" {
					// The error above names the failure; blaming the
					// query for an empty list would hide it.
					empty = "refsUnavailable"
				}
				display.Text(c, a.t(empty), display.TextProps{
					Type: display.TextSecondary, FontSize: scaledFont(11),
				}).Padding(6, 8)
			}
		}).MaxHeight(listHeight)
		if hasActive {
			search.ActiveDescendant(active)
		}
	})
	return search
}

// refGroupLabel names a group in the ref selector; all refs have none.
func refGroupLabel(t func(string) string, kind string) string {
	switch kind {
	case historyRefLocal:
		return t("localBranches")
	case historyRefRemote:
		return t("remoteBranches")
	case historyRefTag:
		return t("tags")
	}
	return ""
}

func hasRefGroup(choices []refChoice, kind string) bool {
	for _, choice := range choices {
		if choice.ref.Kind == kind {
			return true
		}
	}
	return false
}

// historyRefRow draws one ref in the selector, with its graph color, and
// chooses it when clicked.
func (a *app) historyRefRow(c *ui.Context, choice refChoice, pos, size int, selected, highlighted bool, choose func(historyRef)) ui.Element {
	tk := theme.TokensOf(c)
	row := pickerRowBase(c, choice.label, pos, size, selected)
	// Exactly one row carries a background: the one the keyboard would take.
	// The applied ref is marked by its check and strong label instead, so the
	// two states never read as two selections side by side.
	switch {
	case highlighted:
		row.Background(selectionBg(tk, tk.BgElevated))
	case row.Hovered():
		row.Background(tk.FillTertiary.Over(tk.BgElevated))
	}
	pickerReveal(row, highlighted, &a.historyRefReveal)
	row.Children(func() {
		// The icon names the kind of ref and its color ties the row to the
		// lane, the same way the branch and tag lists do. A plain dot showed
		// the color but made a tag and a branch with the same name look alike
		// once a search hid the group headers.
		ink := tk.TextTertiary
		if choice.color.A > 0 {
			ink = choice.color
		}
		ui.Icon(c, choice.icon).FontSize(scaledFont(12)).TextColor(ink)
		display.Text(c, choice.label, display.TextProps{FontSize: scaledFont(12), Strong: selected || highlighted}).
			SingleLine().Grow(1).MinWidth(0)
		if choice.head {
			// Secondary rather than tertiary: this label can land on the
			// selected row's tint, where tertiary text drops below AA.
			ui.Text(c, a.t("current")).FontSize(scaledFont(10)).TextColor(tk.TextSecondary)
		}
		if selected {
			ui.Icon(c, lucide.Check()).FontSize(scaledFont(12)).TextColor(tk.Blue)
		}
	})
	row.Tooltip(choice.label)
	if row.Clicked() {
		choose(choice.ref)
	}
	return row
}

// applyHistoryRef scopes the history list to ref and walks from its tip. A
// branch or tag older than the loaded pages is not in them, so filtering the
// loaded rows would silently show everything; walking from the tip shows that
// ref's own history instead. Choosing all refs walks the whole set again.
func (a *app) applyHistoryRef(ref historyRef) {
	a.historyRef = ref
	a.historyQuery = ""
	// A new scope lists different commits, so the old offset would open the
	// list somewhere arbitrary. The scroll is kept for page switches, which
	// keep the same scope.
	a.historyScroll.Y = 0
	// A ref filter and a file filter both scope the list; the newest choice
	// wins, and keeping the file filter would make the ref appear ignored.
	a.resetFileHistory()
	a.fileFilter = ""
	a.fileCommits = nil
	a.fileHistoryErr = ""
	tip := ""
	if ref.Kind != "" {
		tip = a.historyRefTip(ref)
	}
	// A deleted ref has no tip: walking every ref keeps the list usable
	// instead of empty behind a filter that selects nothing.
	a.historyPinned = tip
	a.pendingJumpSHA = tip
	a.reloadHistoryFromTip(tip)
	a.invalidate()
}

// failHistoryReload settles a walk that could not read. The rows in a.commits
// belong to the walk this request replaced, and the header now names a
// different filter, so keeping them would let the details pane describe a
// commit the selected ref does not contain. Dropping the rows and the
// selection leaves "read failed" as the only thing the page claims.
func (a *app) failHistoryReload(message string) {
	a.historyErr = message
	a.pendingJumpSHA = ""
	a.commits = nil
	a.historyTips = nil
	a.historyMore = false
	a.selectedSHA = ""
	a.worktreeSelected = false
	a.commitFiles = nil
	a.commitFilesLoaded = false
	a.commitPath = ""
	a.commitSeq++
	a.resetCommitDiff()
	a.commitDiffErr = ""
	a.clearCommitSelection()
	a.invalidate()
}

// reloadHistoryFromTip replaces the loaded pages with the first page walked
// from tip, or from every ref when tip is empty.
func (a *app) reloadHistoryFromTip(tip string) {
	if a.repo == nil {
		return
	}
	repo, seq := a.repo, a.loadSeq
	a.resetHistoryTraversal()
	reload, ctx := a.historyReload, a.historyTraversalContext()
	// A page of the previous walk is no longer wanted; its callback checks the
	// same counter and drops out, so the flag is cleared here rather than
	// waiting for a callback that will not touch it.
	a.historyLoadingMore = false
	// The rows still in a.commits belong to the walk being replaced. The list
	// draws a placeholder until this walk answers, so a filter change never
	// shows commits the header says are filtered out.
	a.historyReloading = true
	// A new walk replaces whatever the last one said, whether it succeeded or
	// failed; keeping the old error would leave the error line up while the
	// placeholder says the history is being reread.
	a.historyErr = ""
	var tips []string
	if tip != "" {
		tips = []string{tip}
	}
	hidden := a.hiddenBranches[repo.Root]
	go func() {
		var page gitrepo.HistoryPage
		var err error
		if tips == nil {
			page, err = repo.ReadVisibleHistory(ctx, 200, hidden)
		} else {
			page, err = repo.HistoryPage(ctx, tips, 0, 200)
		}
		a.hopToMain(func() {
			if a.historyPageStale(seq, repo, reload) {
				return
			}
			// This walk is the current one: whatever the answer, it ends the
			// placeholder. A stale callback left the flag alone because a
			// newer walk owns it now.
			a.historyReloading = false
			if err != nil {
				a.failHistoryReload(err.Error())
				return
			}
			a.commits = page.Commits
			a.historyTips = page.Tips
			a.historyMore = page.HasMore
			a.historyErr = ""
			if !page.HasMore && len(page.Commits) == 0 && a.fileFilter == "" && a.selectedStash == "" && a.pendingJumpSHA == "" {
				a.selectWorktree()
			}
			// A jump that made the filter give way waits for this walk: the
			// commit it wants may only be reachable from the other refs.
			if sha := a.pendingJumpSHA; sha != "" {
				a.pendingJumpSHA = ""
				a.jumpToCommitKeeping(sha, true)
			}
			a.invalidate()
			a.loadMoreHistory()
		})
	}()
}

// locateRef selects the tip in History. Keep a compatible scope and search,
// and let a filter give way if it would hide the requested commit.
func (a *app) locateRef(kind, name string) {
	if a.repo == nil || name == "" {
		return
	}
	tip := a.historyRefTip(historyRef{Kind: kind, Name: name})
	if tip == "" {
		return
	}
	if a.page != "history" {
		a.openPage("history")
	}
	a.showCommitKeeping(tip, true)
}

// showRefHistory scopes the history page to one ref, the same state the
// header's ref selector sets. Branch, remote and tag menus use it to jump
// from a ref to the commits it reaches.
func (a *app) showRefHistory(kind, name string) {
	if a.repo == nil || name == "" {
		return
	}
	a.applyHistoryRef(historyRef{Kind: kind, Name: name})
	a.selectedSHA = ""
	a.commitFiles = nil
	a.commitFilesLoaded = false
	a.commitPath = ""
	a.resetCommitDiff()
	a.commitDiffErr = ""
	// The page switch waits for the context menu to finish closing. A menu
	// belongs to its row; replacing the page while the popup is still
	// animating out drops the row that owns it and leaves the popup's layer
	// in the window. That layer then redraws on every frame as soon as the
	// next dialog opens and keeps the keyboard until it is dismissed. The
	// popup exits in 75ms (motion.Popup); the margin below covers a slow
	// frame, and 150ms is short enough that the jump still feels immediate.
	repo := a.repo
	go func() {
		time.Sleep(menuCloseDelay)
		mygo.RunOnMain(func() {
			if a.repo != repo {
				return
			}
			// Already on history (a ref badge in the list): only the filter
			// changed, so refresh without replacing the page or its rows.
			if a.page != "history" {
				a.page = "history"
			}
			a.invalidate()
		})
	}()
}

// refBadgeMenu gives the ref labels in the history list their own menu, so a
// branch, remote or tag can be opened as history without leaving the commit
// list for the branch page.
func (a *app) refBadgeMenu(c *ui.Context, target ui.Element, ref string) {
	kind, name, ok := a.parseCommitRef(ref)
	if !ok {
		return
	}
	copyLabel := a.t("copyName")
	if kind == historyRefTag {
		copyLabel = a.t("copyTagName")
	}
	// Copying the name is what a reader reaches for most on a capsule whose
	// label may be elided; it sits first, as it does for the other rows' menus.
	switch overlay.ContextMenu(c, target, overlay.DropdownMenuProps{Items: []overlay.MenuItem{
		{Key: "copy", Label: copyLabel, Icon: lucide.Copy()},
		{Key: "history", Label: a.t("viewHistory"), Icon: lucide.History()},
	}}) {
	case "history":
		a.showRefHistory(kind, name)
	case "copy":
		a.copyText(name)
	}
}

// foldedRefsMenu lists the refs a history row folded into its +N capsule. Each
// entry opens that ref's history the way its own capsule would, and the last
// item copies every folded name at once, since the capsule's tooltip is the
// only other place they can be read.
func (a *app) foldedRefsMenu(c *ui.Context, target ui.Element, refs []string) {
	items := make([]overlay.MenuItem, 0, len(refs)+2)
	targets := make(map[string][2]string, len(refs))
	names := make([]string, 0, len(refs))
	for i, ref := range refs {
		kind, name, ok := a.parseCommitRef(ref)
		if !ok {
			continue
		}
		badge := a.refBadge(ref)
		key := "folded-" + itoa(i)
		targets[key] = [2]string{kind, name}
		names = append(names, badge.Label)
		items = append(items, overlay.MenuItem{Key: key, Label: badge.Label, Icon: badge.Icon})
	}
	if len(items) == 0 {
		return
	}
	items = append(items,
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{Key: "copy", Label: a.t("copyRefNames"), Icon: lucide.Copy()},
	)
	switch chosen := overlay.ContextMenu(c, target, overlay.DropdownMenuProps{Items: items}); chosen {
	case "copy":
		a.copyText(strings.Join(names, "\n"))
	default:
		if pair, ok := targets[chosen]; ok {
			a.showRefHistory(pair[0], pair[1])
		}
	}
}

func splitHistoryRefKey(key string) (kind, name string, ok bool) {
	for _, k := range []string{historyRefLocal, historyRefRemote, historyRefTag} {
		prefix := k + ":"
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			return k, key[len(prefix):], true
		}
	}
	return "", "", false
}
