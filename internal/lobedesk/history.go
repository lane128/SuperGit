package lobedesk

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

// The history page shares its width between the commit list and the two
// columns right of it: the changed files and the patch. The default gives the
// list the smaller half, as Sublime Merge does; the divider drags any other
// share, and neither side shrinks below its minimum.
const (
	historySplitDefault float32 = 0.45
	historySplitMinList float32 = 260
)

// loadMoreHistory advances the immutable traversal one bounded page at a time.
// Each completion schedules the next page; rendering never starts a read.
func (a *app) loadMoreHistory() {
	if a.repo == nil || a.loading || a.historyLoadingMore || a.historyReloading || a.historyErr != "" || a.historyPageErr != "" || a.historyPaused || a.fileFilter != "" || !a.historyMore {
		return
	}
	ctx := a.historyTraversalContext()
	repo, tips, offset, seq := a.repo, append([]string(nil), a.historyTips...), len(a.commits), a.loadSeq
	reload := a.historyReload
	hidden := a.hiddenBranches[repo.Root]
	stream := a.historyStream
	a.historyLoadingMore = true
	a.invalidate()
	go func() {
		var err error
		if stream == nil {
			stream, err = repo.OpenVisibleHistoryStream(ctx, tips, offset, hidden)
		}
		var page gitrepo.HistoryPage
		if err == nil {
			page, err = stream.Next(200)
		}
		a.hopToMain(func() {
			if a.historyPageStale(seq, repo, reload) || a.fileFilter != "" {
				if stream != nil {
					stream.Close()
				}
				return
			}
			a.historyStream = stream
			if err != nil || !page.HasMore {
				if stream != nil {
					stream.Close()
				}
				a.historyStream = nil
			}
			a.historyLoadingMore = false
			if err != nil {
				// Partial results remain browsable. HasMore stays true so an
				// interrupted read cannot be mistaken for the traversal's end.
				a.historyPageErr = err.Error()
				if p := a.historyNavigation.pending; p != nil && p.messageOrigin != nil {
					a.discardStaleHistoryNavigation()
					if a.historyNavigationBlocked() {
						a.cancelPendingHistoryNavigation()
					}
				}
				if a.pendingJumpSHA != "" {
					a.commitJumpNotice(a.pendingJumpSHA, true)
					a.pendingJumpSHA = ""
				}
				a.invalidate()
				return
			}
			a.commits = append(a.commits, page.Commits...)
			a.historyTips = page.Tips
			a.historyMore = page.HasMore
			if !a.historyMore {
				previous := a.selectedSHA
				a.reconcileHistorySelection()
				a.refreshSelectedCommit = gitrepo.Commit{}
				if a.selectedSHA != previous && a.selectedSHA != "" {
					a.loadCommitDetail(true)
				}
			}
			a.resumePendingCommitJump()
			a.invalidate()
			a.loadMoreHistory()
		})
	}()
}

func (a *app) historyTraversalContext() context.Context {
	if a.historyReadCtx == nil {
		parent := a.ctx
		if parent == nil {
			parent = context.Background()
		}
		a.historyReadCtx, a.historyReadCancel = context.WithCancel(parent)
	}
	return a.historyReadCtx
}

// resetHistoryTraversal cancels workers as well as rejecting their callbacks.
// Loaded rows and pinned tips remain available until their replacement arrives.
func (a *app) resetHistoryTraversal() {
	a.clearHistoryNavigation()
	if a.historyStream != nil {
		a.historyStream.Close()
		a.historyStream = nil
	}
	if a.historyReadCancel != nil {
		a.historyReadCancel()
	}
	a.historyReadCtx, a.historyReadCancel = nil, nil
	a.historyReload++
	a.historyLoadingMore = false
	a.historyPaused = false
	a.historyPageErr = ""
}

func (a *app) pauseHistoryTraversal() {
	// Report an owned message jump before reset discards its source query and
	// full target. Replaced views and other navigation commands retain their
	// existing cancellation behavior.
	a.discardStaleHistoryNavigation()
	if p := a.historyNavigation.pending; p != nil && p.messageOrigin != nil && !a.historyNavigationBlocked() {
		a.commitJumpNotice(p.target, true)
	}
	a.resetHistoryTraversal()
	a.historyPaused = true
	a.pendingJumpSHA = ""
	a.invalidate()
}

// historyPageStale reports whether a page that came back from the repository
// no longer describes the list on screen: the repository was reopened, or the
// history was re-walked from another tip. A load-more started before that walk
// would otherwise append the old ref's commits to the new list and hand the
// next page its old tips.
func (a *app) historyPageStale(seq uint64, repo *gitrepo.Repository, reload int) bool {
	return seq != a.loadSeq || a.repo != repo || reload != a.historyReload
}

// showCommit brings one commit into view from another page: a tag's commit, a
// blame line or a parent link. A ref filter that does not contain the commit
// would page after a row it can never list, so it gives way and the history is
// walked again from every ref; otherwise the filter stays. The jump then pages
// and scrolls until the row shows.
func (a *app) showCommit(sha string) {
	a.showCommitKeeping(sha, false)
}

func (a *app) showCommitKeeping(sha string, keepSearch bool) {
	if a.repo == nil || sha == "" {
		return
	}
	if !keepSearch {
		a.historyQuery = ""
	}
	// While a replacement walk is starting, its old rows cannot establish
	// membership. Only the pinned tip is known to be in the new scope.
	if a.historyRef.Kind != "" && sha != a.historyPinned && (a.historyReloading || !historyLists(a.historySource(), sha)) {
		a.historyRef = historyRef{}
		a.historyPinned = ""
		a.pendingJumpSHA = sha
		a.reloadHistoryFromTip("")
		a.invalidate()
		return
	}
	if a.historyReloading {
		a.pendingJumpSHA = sha
		a.invalidate()
		return
	}
	a.jumpToCommitKeeping(sha, keepSearch)
}

// jumpToCommit selects a commit in the history list, loading older pages
// until it appears. A parent link can point past the loaded window, so the
// jump keeps paging instead of stopping at 200 rows.
func (a *app) jumpToCommit(sha string) {
	a.jumpToCommitKeeping(sha, false)
}

// jumpToCommitKeeping is jumpToCommit with the option to keep the reader's
// search. A jump is about one commit, so a search that hides it would make the
// link look broken; but a locate from the sidebar should not change the rows
// on screen, so keepSearch survives as long as the filtered rows still
// contain the target. Clearing the query also makes paging report the real
// end of history.
func (a *app) jumpToCommitKeeping(sha string, keepSearch bool) {
	if a.repo == nil || sha == "" {
		return
	}
	if !keepSearch || a.historyQuery != "" && !historyLists(a.filteredHistoryRows(), sha) {
		a.historyQuery = ""
	}
	a.historyRowsSet = false
	if a.fileFilter != "" && !historyLists(a.fileCommits, sha) {
		// A parent of a file's commit may be older than the file itself, so
		// the jump continues in the full history.
		a.resetFileHistory()
		a.fileFilter = ""
		a.fileCommits = nil
		a.fileHistoryErr = ""
		a.resetHistoryTraversal()
		a.loadMoreHistory()
	}
	if historyLists(a.historySource(), sha) {
		a.pendingJumpSHA = ""
		a.revealSelection = true
		a.selectJumpCommit(sha)
		a.invalidate()
		return
	}
	if a.historyErr != "" || a.historyPageErr != "" || a.historyPaused {
		a.pendingJumpSHA = ""
		a.commitJumpNotice(sha, true)
		a.invalidate()
		return
	}
	if a.historyLoadingMore {
		a.pendingJumpSHA = sha
		return
	}
	if !a.historyMore {
		a.pendingJumpSHA = ""
		a.commitJumpNotice(sha, false)
		a.invalidate()
		return
	}
	a.pendingJumpSHA = sha
	a.loadMoreHistory()
}

// historyLists reports whether the rows the reader is looking at contain sha.
func historyLists(commits []gitrepo.Commit, sha string) bool {
	for _, commit := range commits {
		if commit.SHA == sha {
			return true
		}
	}
	return false
}

// historyReadStatus distinguishes a pending, paused or failed partial traversal.
func (a *app) historyReadStatus(c *ui.Context) {
	ui.Column(c).FillWidth().Padding(10).Gap(4).Children(func() {
		label := a.t("loadingMoreHistory")
		if a.historyPaused {
			label = a.t("historyReadPaused")
		} else if a.historyPageErr != "" {
			label = a.t("historyLoadFailed") + ": " + a.historyPageErr
		}
		display.Text(c, label, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11), Rows: 3})
		if a.historyLoadingMore {
			if workbenchButton(c, a.t("cancel"), buttonNormal, false).Clicked() {
				a.pauseHistoryTraversal()
			}
		} else if a.historyPaused || a.historyPageErr != "" {
			if workbenchButton(c, a.t("retry"), buttonNormal, a.loading).Clicked() {
				a.resetHistoryTraversal()
				a.loadMoreHistory()
			}
		}
	})
}

// The sidebar divider also reserves room for the History toolbar when the
// detail panes are hidden and their own minimum widths no longer apply.
const historyHeaderMinWidth float32 = 560

func (a *app) historyHeaderWidth() float32 {
	if a.historySplit.rowWidth > 0 {
		return a.historySplit.rowWidth
	}
	return defaultWindowWidth - sidebarWidth - splitterWidth
}

func (a *app) historyHeaderLabelWidth() float32 {
	// Leave room for the search and the fixed title, filter and pane controls.
	return min(220, max(140, a.historyHeaderWidth()-540))
}

func (a *app) historyPageAtWidth(c *ui.Context, width float32) {
	a.historySplit.rowWidth = width
	rowHeight := a.historyRowHeight()
	if previous := a.historyRowExtent; previous > 0 && previous != rowHeight && a.historyScroll.Y > historyListPadding {
		// Keep the same top row and fraction into it across density changes.
		// Apply this before merge folding interprets the offset in rows.
		a.historyScroll.Y = historyListPadding + (a.historyScroll.Y-historyListPadding)/previous*rowHeight
	}
	a.historyRowExtent = rowHeight
	source := a.historySource()
	subtitle := a.commitsLabel(len(source))
	filtered := a.filteredHistoryRows()
	commits := a.projectHistoryMerges(a.historyWithStashes(filtered))
	if a.selectedStash != "" && a.stashTimeline.bySHA[a.selectedStash] == nil && !a.loading && !a.historyReloading {
		a.clearStashDetail()
	}
	// The list, the selection and the details pane all index into these rows;
	// cache them for the frame instead of walking and filtering three times.
	a.historyRowsCache, a.historyRowsFrame, a.historyRowsSet = commits, a.frameSeq, true
	if strings.TrimSpace(a.historyQuery) != "" {
		subtitle = a.commitsLabel(len(filtered))
	}
	if a.historyErr != "" {
		// The count would describe the walk that failed, and after a ref
		// switch its rows are not even from the ref the header names.
		subtitle = a.t("commitsReadFailed")
	}
	// A search that hides the selected commit moves the selection to the
	// first match, so the details never show a row the list no longer has,
	// as the branches page does. The working tree's row is not part of the
	// search results and keeps the selection while it is chosen.
	more := a.historyMore
	if a.fileFilter != "" {
		more = a.fileHistoryMore
	}
	// Missing from a replacement page does not mean hidden by the search.
	// Reconcile only once the selected row is loaded or the walk is complete.
	// A chosen stash is the details' subject: picking a commit for it would
	// replace the stash's patch with a commit nobody chose.
	if !a.historyReloading && a.pendingJumpSHA == "" && a.selectedStash == "" && !a.worktreeSelected && (a.selectedSHA == "" || a.hasSelectedIn(source) || !more) {
		if sha := historySelection(filtered, a.selectedSHA); sha != "" && sha != a.selectedSHA {
			a.selectCommit(sha)
		}
	}
	toggleDetails := false
	a.pageHeader(c, a.t("history"), subtitle, func() {
		if a.fileFilter == "" {
			a.historyRefMenu(c)
		}
		if a.historyErr != "" {
			a.statusTag(c, a.t("history"), display.TagProps{Variant: display.TagFilled, Color: display.TagError, Size: theme.Small, Round: true})
		}
		if a.fileFilter != "" {
			a.statusTag(c, a.fileFilter, display.TagProps{
				Variant: display.TagFilled, Color: display.TagInfo, Size: theme.Small, Round: true,
				Icon: lucide.FileDiff(),
			}).MaxWidth(a.historyHeaderLabelWidth()).Tooltip(a.fileFilter)
			clear := compactActionIcon(c, lucide.X(), a.t("showAllHistory"), buttonNormal, false)
			if clear.Clicked() {
				a.clearFileFilter()
			}
		}
		toggleDetails = a.historyDetailsToggle(c)
		searchWidth := min(292, max(176, a.historyHeaderWidth()-400))
		ui.Box(c).Width(searchWidth).Shrink(0).Children(func() {
			queryBefore := a.historyQuery
			key := "mod+f"
			if c.FocusTakesText() || a.historyRefOpen || a.repositoryChooserOpen || a.hasPendingDialog(toastMain) {
				key = ""
			}
			field.SearchBar(c, &a.historyQuery, field.SearchBarProps{
				Placeholder: a.t("searchHistory"),
				ShortKey:    key,
				Size:        theme.Small,
				Variant:     field.InputFilled,
			}).OnTextKey(a.fileTabTextKey)
			if a.historyQuery != queryBefore {
				a.cancelPendingHistoryNavigation()
			}
		})
	})
	row := ui.Row(c).Grow(1).Basis(0).MinHeight(0).AlignItems(ui.Stretch)
	row.Children(func() {
		share := a.historyShare()
		detailsWidth := width - splitterWidth - a.commitListWidth()
		a.commitList(c)
		// Hiding the details gives the list the whole row: no divider, no
		// details, and the header's toggle brings the pane back.
		if a.historyDetailsHidden {
			return
		}
		a.splitHandle(c, &a.historySplit, a.historyDefaultShare(),
			a.historyListMinWidth(), a.historyDetailMinWidth(), a.t("resizeHistoryPanes"))
		// The divider can update the preferred share during the build. Use
		// the share already allocated to the list for both sides of this frame.
		a.historyDetailsPane(c, 1-share, detailsWidth)
	})
	if toggleDetails {
		// The sidebar already reserved room for the current visibility. Apply
		// the toggle after building this row so all panes change together.
		a.setHistoryDetailsHidden(!a.historyDetailsHidden)
		a.persistSettings()
		c.After(0)
	}
}

// historyShare is how much of the pane row the commit list takes. The minimum
// widths win over the stored share when the window is too narrow for both,
// and the details get the rest.
func (a *app) historyShare() float32 {
	return a.historySplit.fraction(a.historyDefaultShare(), a.historyListMinWidth(), a.historyDetailMinWidth())
}

func (a *app) historyDefaultShare() float32 {
	if a.detailsChangesHidden {
		return historySplitDefault
	}
	const narrowListTarget = 370
	compact := shareForWidth(a.historySplit.rowWidth-splitterWidth, narrowListTarget)
	return historySplitDefault + a.historyWidthPressure()*max(0, compact-historySplitDefault)
}

func (a *app) historyListMinWidth() float32 {
	if a.detailsChangesHidden || a.sidebarSplit.rowWidth <= 0 {
		return historySplitMinList
	}
	// The working tree's commit editor lives under the Files column, so when
	// the window narrows below the comfortable 980 points the list yields part
	// of its floor instead of the editor being squeezed to a sliver. Keep the
	// same floor as the window widens; relaxing it there would let a saved
	// sidebar or Files preference take space back from History.
	return 350 - 50*a.historyNarrowPressure()
}

func (a *app) historyDetailMinWidth() float32 {
	patch := a.historyPatchMinWidth()
	if a.detailsChangesHidden {
		return patch
	}
	return a.historyFilesMinWidth() + splitterWidth + patch
}

// historyDetailsToggle collapses or restores the details pane. The control
// lives in the page header because that is the one strip that exists in both
// states; a close button on the pane itself would vanish with the pane.
func (a *app) historyDetailsToggle(c *ui.Context) bool {
	icon, title := lucide.PanelRightClose(), a.t("hideCommitDetails")
	if a.historyDetailsHidden {
		icon, title = lucide.PanelRightOpen(), a.t("showCommitDetails")
	}
	return compactActionIcon(c, icon, title, buttonNormal, false).Clicked()
}

// historyListState tells the commit list what it draws: rows, a placeholder
// for a walk still out, or one of the empty states. A stale walk outranks the
// old rows: after a filter change a.commits still holds the previous ref's
// commits, and showing them under the new header reads as if the filter did
// nothing. Errors print their own line above the list and suppress the
// placeholders, so they come first.
type historyListState int

const (
	historyListRows historyListState = iota
	historyListError
	historyListReloading
	historyListFileLoading
	historyListNoFileHistory
	historyListEmpty
	historyListNoMatch
)

func (a *app) historyListState(rows int) historyListState {
	source := a.historySource()
	switch {
	case a.fileFilter == "" && a.historyErr != "" || a.fileHistoryErr != "" && len(source) == 0:
		return historyListError
	case a.historyReloading:
		return historyListReloading
	case a.fileHistoryLoading && len(source) == 0:
		return historyListFileLoading
	case a.fileHistoryPaused && len(source) == 0:
		return historyListRows // The footer describes the interrupted read.
	case len(source) == 0 && a.fileFilter != "":
		return historyListNoFileHistory
	case len(source) == 0 && rows == 0:
		return historyListEmpty
	case rows == 0:
		return historyListNoMatch
	}
	return historyListRows
}

func (a *app) commitList(c *ui.Context) {
	tk := theme.TokensOf(c)
	commits := a.historyRows()
	ui.Column(c).Grow(a.historyShare()).Basis(0).MinWidth(0).AlignSelf(ui.Stretch).Children(func() {
		// The list takes a Tab stop of its own, and the ring shows where the
		// arrows act once Tab left the search field. The ring sits inside the
		// pane so the divider to the details keeps its place.
		list := ui.Column(c).Fill().Focusable()
		for _, s := range shortcutTable() {
			if s.ID == "history.secondParent" && s.Local && s.Available(a) {
				run := s.Run
				list.OnShortcut(s.Mod, s.Key, func() { run(a) })
			}
		}
		ring := ui.Transparent
		if list.FocusWithin() {
			ring = tk.Blue.Alpha(0.35)
		}
		list.Border(historyListBorder, ring)
		list.Children(func() {
			if a.historyErr != "" {
				a.failedWithRetry(c, a.historyErr, 12)
			}
			a.historyGraph = nil
			state := a.historyListState(len(commits))
			// With the graph hidden nothing points at a lane, so the list
			// carries no graph: labels fall back to their name colors, as the
			// branch and tag lists draw them, and the lanes are not laid out at
			// all. The layout is built before the rows so the working tree's
			// empty graph column is exactly as wide as the commits' lanes.
			var graph *commitGraph
			if state == historyListRows && a.showGraph {
				graph = a.cachedCommitGraph(commits, a.lanePalette(tk), a.commitListWidth()*0.26)
				// The details pane, built after this list in the same frame,
				// reads the same graph so its badges match.
				a.historyGraph = graph
			}
			switch state {
			case historyListError:
				// The error line above says why there is nothing to list; an
				// empty-state placeholder under it would contradict it.
			case historyListReloading:
				a.diffPlaceholder(c, lucide.LoaderCircle(), a.t("rereadingHistory"))
			case historyListFileLoading:
				a.diffPlaceholder(c, lucide.FileClock(), a.t("diffLoading"))
			case historyListNoFileHistory:
				a.diffPlaceholder(c, lucide.FileClock(), a.t("noFileHistory"))
			default:
				scroll := layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.historyScroll}, func() {
					ui.Column(c).FillWidth().Padding(historyListPadding).Role(ui.RoleList).Children(func() {
						total := 1 + len(commits)
						a.historyWorktreeRow(c, 0, total)
						a.historyStashExtent = a.historyStashErrorBanner(c)
						switch state {
						case historyListEmpty:
							a.diffPlaceholder(c, lucide.History(), a.t("noHistory"), a.t("noHistoryHint"))
						case historyListNoMatch:
							message := "searchNoMatch"
							partial := a.historyMore
							if a.fileFilter != "" {
								partial = a.fileHistoryMore || a.fileHistoryLoading || a.fileHistoryPaused || a.fileHistoryErr != ""
							}
							if partial {
								message = "searchNoMatchPartial"
							}
							a.diffPlaceholder(c, lucide.SearchSlash(), a.t(message))
						default:
							heights := make([]float32, len(commits))
							// A click can rebuild the view before layout applies
							// ScrollIntoView. Keep its target in the virtual list
							// across those passes, even after the request is queued.
							selected := -1
							for i := range commits {
								heights[i] = a.historyRowHeight()
								if commits[i].SHA == a.selectedSHA || commits[i].SHA == a.selectedStash {
									selected = i
								}
							}
							cache := a.prepareHistoryCounts()
							var missing []gitrepo.Commit
							virtualPatchRows(c, a.historyViewport, &a.historyScroll, heights, selected, func(rowContext *ui.Context, i int) {
								commit := commits[i]
								stash := a.stashTimeline.bySHA[commit.SHA]
								cache.queue(&missing, commit)
								if extra := stashUntrackedSnapshot(stash); extra != "" {
									cache.queue(&missing, gitrepo.Commit{SHA: extra})
								}
								a.historyEntryRow(rowContext.Key(commit.SHA), commit, stash, i, i+2, total, graph)
							})
							a.loadHistoryFileCounts(cache, missing)
						}
					})
				}).Grow(1).Basis(0).MinHeight(0)
				a.historyViewport = scroll.Bounds()
			}
			// Keep progress, cancellation and retry outside the scroll area
			// so they remain reachable throughout a partial traversal.
			if a.fileFilter == "" && a.historyMore && a.historyErr == "" {
				a.historyReadStatus(c)
			}
			if a.fileFilter != "" && (a.fileHistoryLoading || a.fileHistoryPaused || a.fileHistoryErr != "") {
				a.fileHistoryReadStatus(c)
			}
		})
	})
}

// Only the read-error banner has a non-row height.
func (a *app) historyStashErrorBanner(c *ui.Context) float32 {
	if a.stashesErr == "" {
		return 0
	}
	group := ui.Column(c).FillWidth().Shrink(0)
	group.PaddingX(12).PaddingY(8).Gap(4)
	a.continueWorktreeThrough(group, theme.TokensOf(c), 12, 8)
	group.Children(func() {
		display.Text(c, a.stashesLabel(0, true), display.TextProps{Strong: true, FontSize: scaledFont(11)})
		display.Text(c, a.stashesErr, display.TextProps{Type: display.TextDanger, FontSize: scaledFont(11), Rows: 2})
		if workbenchButton(c, a.t("retry"), buttonNormal, false).Clicked() {
			a.reload()
		}
	})
	return group.Bounds().H
}

func (a *app) showFileHistory(path string) { a.showFileHistoryAt(path, "") }

func (a *app) showFileHistoryAt(path, rev string) {
	if a.repo == nil || path == "" {
		return
	}
	a.historyQuery = ""
	a.historyScroll.Y = 0
	a.page = "history"
	// A file's history is a commit list; the working tree's row is not part of
	// it, and keeping its selection would leave the files column describing
	// changes this list does not contain.
	a.worktreeSelected = false
	a.setSelectedCommit("")
	a.commitFiles = nil
	a.commitFilesLoaded = false
	a.commitPath = ""
	a.resetCommitDiff()
	a.commitDiffErr = ""
	a.selectedStash = ""
	a.fileHistoryRevision = rev
	a.switchFileHistory(path)
	a.invalidate()
}

// switchFileHistory points the list at one file's history. The previous
// file's rows and its error go with the old filter: keeping them would draw
// another file's commits under this file's header until the walk returns.
func (a *app) switchFileHistory(path string) {
	a.pauseHistoryTraversal()
	a.fileFilter = path
	a.fileCommits = nil
	a.fileHistoryErr = ""
	a.resetFileHistory()
	a.fileHistoryTips = a.startFileHistoryTips()
	a.fileHistoryMore = true
	a.loadFileHistory()
}

func (a *app) loadFileHistory() {
	if a.repo == nil || a.fileFilter == "" || a.loading || a.fileHistoryLoading || a.fileHistoryPaused || a.fileHistoryErr != "" || a.fileHistoryTips != nil && !a.fileHistoryMore {
		return
	}
	if a.fileHistoryCtx == nil {
		parent := a.ctx
		if parent == nil {
			parent = context.Background()
		}
		a.fileHistoryCtx, a.fileHistoryCancel = context.WithCancel(parent)
	}
	seq, generation := a.fileHistorySeq, a.loadSeq
	repo, path, ctx := a.repo, a.fileFilter, a.fileHistoryCtx
	tips := a.fileHistoryTips
	if tips != nil {
		tips = append([]string{}, tips...)
	}
	offset := len(a.fileCommits)
	a.fileHistoryLoading = true
	a.invalidate()
	go func() {
		page, err := repo.FileHistoryPage(ctx, path, tips, offset, 200)
		a.hopToMain(func() {
			if seq != a.fileHistorySeq || generation != a.loadSeq || a.repo != repo || a.fileFilter != path {
				return
			}
			a.fileHistoryLoading = false
			if err != nil {
				a.fileHistoryErr = err.Error()
				a.invalidate()
				return
			}
			a.fileCommits = append(a.fileCommits, page.Commits...)
			a.fileHistoryTips, a.fileHistoryMore = page.Tips, page.HasMore
			if len(a.fileCommits) > 0 {
				if a.selectedSHA == "" || !page.HasMore && !a.hasSelectedIn(a.fileCommits) {
					a.selectCommit(a.fileCommits[0].SHA)
				}
			} else {
				a.setSelectedCommit("")
				a.commitFiles = nil
				a.commitPath = ""
				a.resetCommitDiff()
			}
			a.invalidate()
			a.loadFileHistory()
		})
	}()
}

// resetFileHistory cancels workers and rejects queued completions. Retain the
// cursor and partial rows so a paused traversal can resume from its pinned HEAD.
func (a *app) resetFileHistory() {
	if a.fileHistoryCancel != nil {
		a.fileHistoryCancel()
	}
	a.fileHistoryCtx, a.fileHistoryCancel = nil, nil
	a.fileHistorySeq++
	a.fileHistoryLoading = false
	a.fileHistoryPaused = false
}

func (a *app) pauseFileHistory() {
	a.resetFileHistory()
	a.fileHistoryPaused = true
	a.invalidate()
}

func (a *app) fileHistoryReadStatus(c *ui.Context) {
	ui.Column(c).FillWidth().Padding(10).Gap(4).Children(func() {
		label := a.t("loadingMoreHistory")
		if a.fileHistoryPaused {
			label = a.t("historyReadPaused")
		} else if a.fileHistoryErr != "" {
			label = a.t("historyLoadFailed") + ": " + a.fileHistoryErr
		}
		display.Text(c, label, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11), Rows: 3})
		if a.fileHistoryLoading {
			if workbenchButton(c, a.t("cancel"), buttonNormal, false).Clicked() {
				a.pauseFileHistory()
			}
		} else if a.fileHistoryPaused || a.fileHistoryErr != "" {
			if workbenchButton(c, a.t("retry"), buttonNormal, a.loading).Clicked() {
				a.resetFileHistory()
				a.fileHistoryErr = ""
				a.loadFileHistory()
			}
		}
	})
}

func (a *app) clearFileFilter() {
	if a.fileFilter == "" {
		return
	}
	a.resetFileHistory()
	a.fileFilter = ""
	a.fileCommits = nil
	a.fileHistoryErr = ""
	a.historyQuery = ""
	a.historyScroll.Y = 0
	a.selectedSHA = ""
	a.commitFiles = nil
	a.commitFilesLoaded = false
	a.commitPath = ""
	a.resetCommitDiff()
	if len(a.commits) > 0 {
		a.selectCommit(a.commits[0].SHA)
	}
	a.resetHistoryTraversal()
	a.loadMoreHistory()
	a.invalidate()
}

// refBadgeRunes is how many runes a row's ref badge shows before it is elided
// in the middle. A badge is capped at 150 points; this keeps the label inside
// that cap while leaving room for its icon. The details pane has a 320-point
// capsule, so its badges get a longer budget -- enough that most names survive
// whole, still elided in the middle so two refs sharing a prefix stay apart by
// their tails.
const (
	refBadgeRunes         = 20
	refCapsuleDetailRunes = 40
)

// commitListWidth is the commit column's share of the history row this frame,
// or zero before the row has been laid out.
func (a *app) commitListWidth() float32 {
	row := a.historySplit.rowWidth
	if row <= 0 {
		return 0
	}
	if a.historyDetailsHidden {
		return row
	}
	return (row - splitterWidth) * a.historyShare()
}

// refSlotsForWidth keeps a narrow row's refs from crowding out the subject:
// three capsules can take 450 points, which is wider than the whole column at
// 980. The rest fold into +N with the same focus/HEAD ranking.
func refSlotsForWidth(width float32) int {
	switch {
	case width <= 0:
		return 3
	case width < 340:
		return 1
	case width < 480:
		return 2
	default:
		return 3
	}
}

// rowRefsNarrow reports whether the column is too narrow for the full HEAD
// chip and the wide capsule cap. At 980 the commit column is under 310, where
// the word "HEAD" alone costs the subject a word of its title.
func rowRefsNarrow(width float32) bool {
	return width > 0 && width < 480
}

// rowCapsuleWidthForList caps one row capsule by the column's width. The
// 150-point cap suits a wide list; at 300 it would leave the subject a few
// characters.
func rowCapsuleWidthForList(width float32) float32 {
	if width <= 0 {
		return refCapsuleRowWidth
	}
	return min(refCapsuleRowWidth, max(76, width*0.24))
}

// historyRowHeight is shared by rendered rows, virtual spacers and graph
// positioning. The 34pt single-line design grows to 44pt when metadata needs
// a second line; both follow the text preset to preserve vertical breathing room.
func (a *app) historyRowHeight() float32 {
	if a.hideAuthor && a.hideTime {
		return scaledFont(34)
	}
	// Ref capsules keep their 20pt height even at the small text preset.
	// Leave room for that fixed first line, metadata, the gap and padding.
	return max(scaledFont(44), refCapsuleHeight+scaledFont(24))
}

const (
	historyListBorder  = 1
	historyListPadding = 6
	historyRowPadding  = 8
	historyRowGraphGap = 4
)

// lead is how many rows sit above the commits in the same list -- the working
// tree's row -- so the place a reader hears counts that row too.
func (a *app) commitRow(c *ui.Context, commit gitrepo.Commit, index, lead, total int, graph *commitGraph) {
	a.historyEntryRow(c, commit, nil, index, index+lead+1, total, graph)
}

func (a *app) historyEntryRow(c *ui.Context, commit gitrepo.Commit, stash *gitrepo.Stash, index, pos, total int, graph *commitGraph) {
	tk := theme.TokensOf(c)
	selected := a.selectedSHA == commit.SHA
	if stash != nil {
		selected = a.selectedStash == stash.SHA
	}
	rowHeight := a.historyRowHeight()
	listWidth := a.commitListWidth()
	// No tooltip on the row itself: the subject is already on the row, and a
	// bubble repeating it only covers the next commit under the pointer. The
	// ref capsules keep theirs -- those names can be elided.
	// The list chooses one commit at a time and the details follow it, so the
	// row reports itself as a selectable option rather than a plain button:
	// only an option can say which row the arrows are on through
	// aria-selected, as the branch list does.
	row := ui.ButtonBase(c).Label(commit.Subject).Row().FillWidth().Height(rowHeight).Padding(0, historyRowPadding).
		Gap(historyRowGraphGap).Radius(tk.BorderRadius).AlignItems(ui.Center).
		Role(ui.RoleOption).AccessSelected(selected).PositionInSet(pos, total)
	// A merge commit's shape says it merged two histories, and a screen reader
	// hears only the title; the description carries that one state the row
	// cannot show as text.
	if stash != nil {
		row.Description(stash.Ref)
	} else if len(commit.Parents) > 1 {
		row.Description(a.t("mergeCommitRow"))
	}
	// The node fills and the selected ring's halo paint the surface the node
	// sits on. Taking the row's own background keeps them invisible against a
	// selected or hovered row, instead of punching a page-coloured disc
	// through its tint.
	rowBg := graphNodeGround(tk, selected, row.Hovered())
	switch {
	case selected:
		row.Background(rowBg)
	case row.Hovered():
		row.Background(tk.FillTertiary)
	}
	a.revealIfSelected(row, selected)
	row.Children(func() {
		contentWidth := listWidth - 2*(historyListBorder+historyListPadding+historyRowPadding)
		if a.showGraph && graph != nil {
			width := graph.Width
			if width <= 0 {
				width = 24
			}
			contentWidth -= width + historyRowGraphGap
			lane := ui.Box(c).Width(width).Height(rowHeight).Shrink(0).ClipX()
			if graph.contentWidth() > width {
				lane.HandleInput(a.panHistoryGraph)
			}
			if folds := &a.historyMergeFolds; folds.active && folds.branches[commit.SHA] && index < len(graph.Rows) {
				x := laneXGap(0, graph.Rows[index].Lane, graph.LaneGap()) - a.historyGraphScroll.X
				if x >= historyMergeDisclosureInset && x <= width-historyMergeDisclosureInset {
					lane.Children(func() {
						a.historyMergeDisclosure(c, commit.SHA, x, rowHeight, rowBg, graphInk(tk, graph.LaneColor(graph.Rows[index].Color)))
					})
				}
			}
			lane.Draw(func(p *ui.Painter, r ui.Rect) {
				if index >= len(graph.Rows) {
					return
				}
				p.Clip(r, 0, func() {
					center := r.Y + r.H/2
					r.X -= a.historyGraphScroll.X
					x0 := int(r.X)
					gap := graph.LaneGap()
					rowTop, rowBottom := r.Y, r.Y+r.H
					graphRow := graph.Rows[index]
					line, node := a.graphStyle()
					for span := range graph.spans(index) {
						col := graphInk(tk, graph.LaneColor(span.Color))
						sourceX, parentX, viaX := laneXGap(x0, span.SourceLane, gap), laneXGap(x0, span.ParentLane, gap), laneXGap(x0, span.ViaLane, gap)
						// Row heights are uniform, so the two nodes sit a whole
						// number of rows above and below this row's own centre.
						sourceY := center - float32(index-span.Start)*r.H
						parentY := center + float32(span.End-index)*r.H
						points := graphEdgePoints(sourceX, sourceY, viaX, parentX, parentY, r.H, rowTop, rowBottom)
						if len(points) > 0 {
							p.StrokePolyline(points, line, col)
						}
					}
					if link := graph.Worktree; link != nil {
						a.strokeWorktreeDash(p, laneX(x0, 0), r.Y, link.dashBottom(index, r), line, graphInk(tk, graph.LaneColor(link.Color)))
					}
					x := laneXGap(x0, graphRow.Lane, gap)
					col := graphInk(tk, graph.LaneColor(graphRow.Color))
					shape := graphNodeShape(commit, a.status.Head, graphRow)
					inner, ring := graphNodeRing(node, line)
					mergeSize, dot := graphNodeMerge(node, line), graphNodeDot(node, line)
					if selected {
						p.Fill(ui.Rect{X: x - ring, Y: center - ring, W: ring * 2, H: ring * 2}, rowBg, ring)
						p.Stroke(ui.Rect{X: x - inner, Y: center - inner, W: inner * 2, H: inner * 2}, col, inner, 2)
					}
					if stash != nil {
						shape = "stash"
					}
					switch shape {
					case "stash":
						box := ui.Rect{X: x - mergeSize, Y: center - mergeSize, W: 2 * mergeSize, H: 2 * mergeSize}
						p.Fill(box, rowBg, 1)
						p.Stroke(box, col, 1, max(1.5, line))
					case "head", "merge":
						p.Fill(ui.Rect{X: x - mergeSize, Y: center - mergeSize, W: mergeSize * 2, H: mergeSize * 2}, rowBg, mergeSize)
						p.Stroke(ui.Rect{X: x - mergeSize, Y: center - mergeSize, W: mergeSize * 2, H: mergeSize * 2}, col, mergeSize, max(2, line))
						if shape == "merge" {
							core := max(float32(1), node*1.4)
							p.Fill(ui.Rect{X: x - core, Y: center - core, W: core * 2, H: core * 2}, col, core)
						}
					case "fork":
						diamond := (&ui.Path{}).MoveTo(x, center-mergeSize).
							LineTo(x+mergeSize, center).LineTo(x, center+mergeSize).
							LineTo(x-mergeSize, center).Close()
						p.FillPath(diamond, col)
					default:
						p.Fill(ui.Rect{X: x - dot, Y: center - dot, W: dot * 2, H: dot * 2}, col, dot)
					}
					if graphRow.Boundary {
						tail := boundaryInk(tk, rowBg)
						for n := float32(8); n < r.H/2; n += 6 {
							p.Line(x, center+n, x, center+n+3, 1, tail)
						}
					}
				})
			})
		}
		titleGap := float32(6)
		if rowRefsNarrow(listWidth) {
			titleGap = 4
		}
		title := ui.Span{Text: commit.Subject, Size: scaledFont(12)}
		if selected {
			title.Weight = 700
		}
		metadataGap := float32(8)
		dateSize := scaledFont(11.5)
		fullDate := formatDateTime(commit.Date)
		date := formatHistoryDateTime(commit.Date, time.Now())
		if !a.hideTime && !a.hideAuthor && listWidth > 0 && contentWidth < scaledFont(175) && len(date) > 11 {
			// Keep the same date format for every row at this width. The
			// compact stamp leaves room for a recognizable author and refs.
			date = formatDateTimeLayout(commit.Date, "01-02 15:04")
		}
		refs := commitRefs(commit.Refs)
		if stash != nil {
			refs = nil
		}
		head := headBranch(commit.Refs)
		var refLayout historyRefLayout
		refsOnTitle := len(refs) > 0 && a.hideAuthor && a.hideTime
		var overflowOnTitle bool
		if len(refs) > 0 && !refsOnTitle {
			metadataWidth := contentWidth
			if listWidth > 0 && !a.hideTime {
				dateWidth, _ := c.MeasureText(0, ui.Span{Text: date, Size: dateSize})
				metadataWidth -= dateWidth + metadataGap
			}
			refBudget := metadataWidth
			var authorWidth float32
			if listWidth > 0 && !a.hideAuthor {
				authorWidth, _ = c.MeasureText(0, ui.Span{Text: commit.Author, Size: dateSize})
				// A ref promoted from +N on widening must not clip an already readable author.
				refBudget -= authorWidth
			}
			refLayout = a.fitHistoryRefs(c, refs, head, listWidth, refBudget, metadataGap, true)
			if len(refLayout.shown) == 0 {
				// A folded count can share spare title space. Named badges stay
				// with the author and time whenever either column is shown.
				const compactGap = 4
				wideAuthorRoom := metadataWidth
				if !a.hideTime {
					metadataWidth += metadataGap - compactGap
				}
				overflowLabel := "+" + itoa(len(refs))
				authorRoom := metadataWidth - historyRefPillWidth(c, overflowLabel, false, compactGap)
				overflowOnTitle = authorRoom < 0
				if !overflowOnTitle && listWidth > 0 && !a.hideAuthor && authorRoom < scaledFont(24) {
					// Move only an already folded count, and only if the whole subject
					// and a useful amount of the author can then be shown together.
					minAuthor := min(authorWidth, scaledFont(24))
					if authorRoom < minAuthor && wideAuthorRoom >= minAuthor {
						titleWidth, _ := c.MeasureText(0, title)
						overflowOnTitle = titleWidth+historyRefPillWidth(c, overflowLabel, false, titleGap) <= contentWidth
					}
				}
				if !overflowOnTitle {
					metadataGap = compactGap
				}
			}
		}
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(scaledFont(2)).Children(func() {
			if refsOnTitle {
				titleBudget := contentWidth
				if listWidth > 0 {
					// Reserve room for the selected title in both states so clicking a
					// row cannot fold or move its refs at the same width.
					layoutTitle := title
					layoutTitle.Weight = 700
					titleWidth, _ := c.MeasureText(0, layoutTitle)
					titleBudget -= min(scaledFont(84), titleWidth)
				}
				refLayout = a.fitHistoryRefs(c, refs, head, listWidth, titleBudget, titleGap, false)
			}
			ui.Row(c).FillWidth().Gap(titleGap).AlignItems(ui.Center).Children(func() {
				fontSize := scaledFont(12)
				titleText := display.Text(c, commit.Subject, display.TextProps{FontSize: fontSize, Strong: selected, Rows: 1}).Grow(1).MinWidth(0).
					AccessHidden(true)
				// Measure only the hovered title against its laid-out width, after
				// graph lanes and refs have taken their actual space.
				if titleText.Hovered() {
					if textWidth, _ := c.MeasureText(0, title); textWidth > titleText.Bounds().W+0.5 {
						titleText.Tooltip(commit.Subject)
					}
				}
				if a.repo != nil && a.historyCounts != nil && a.historyCounts.root == a.repo.Root {
					if n, ok := a.historyCounts.fileCount(commit.SHA, stash); ok {
						display.Text(c, itoa(n), display.TextProps{FontSize: scaledFont(10), Type: display.TextSecondary}).
							SingleLine().Shrink(0).Padding(0, 4).Radius(3).Background(tk.FillSecondary).
							Tooltip(a.changedFilesLabel(n))
					}
				}
				if stash != nil && a.hideTime && a.hideAuthor {
					a.stashBadge(c, *stash)
				}
				if refsOnTitle || overflowOnTitle {
					a.commitRowRefs(c, commit, graph, index, rowBg, head, refLayout)
				}
			})
			if !a.hideAuthor || !a.hideTime {
				ui.Row(c).FillWidth().Gap(metadataGap).AlignItems(ui.Center).Description(fullDate).Children(func() {
					// The SHA stays out of the row: the details pane shows it, the
					// row's menu copies it, and printing it here only competed with
					// the author and date for the reader's eye.
					if a.hideAuthor {
						// Keep the time against the right edge where the column
						// normally ends; without the author there is nothing on
						// the left to push it.
						ui.Row(c).Grow(1).Basis(0)
					} else {
						display.Text(c, commit.Author, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11.5)}).SingleLine().
							Grow(1).Basis(0).MinWidth(0)
					}
					if stash != nil {
						a.stashBadge(c, *stash)
					}
					if !overflowOnTitle && (len(refLayout.shown) > 0 || len(refLayout.folded) > 0) {
						a.commitRowRefs(c, commit, graph, index, rowBg, head, refLayout)
					}
					if !a.hideTime {
						stamp := display.Text(c, date, display.TextProps{Type: display.TextSecondary, FontSize: dateSize}).SingleLine().Shrink(0)
						if date != fullDate {
							stamp.Description(fullDate).Tooltip(fullDate)
						}
						if listWidth > 0 {
							// Malformed Git dates are displayed verbatim and may be longer.
							stamp.MaxWidth(max(0, contentWidth-metadataGap))
						}
					}
				})
			}
		})
	})
	if stash != nil {
		a.stashMenu(c, row, *stash)
	} else {
		a.commitMenu(c, row, commit)
	}
	// Opening a row's context menu selects the row, so the details on the
	// right describe the commit the menu acts on, as Sublime Merge does. The
	// context menu owns the right click, so this reads the menu's open state
	// rather than target.RightClicked().
	menuOpened := *ui.Local(row, "contextMenu", func() bool { return false })
	// The arrows move the selection while the row the pointer chose keeps the
	// focus; activating that row on Enter or Space would snap the selection
	// back to it. History has no action of its own for either key, so it
	// claims them and leaves the arrow selection alone.
	row.OnShortcut(0, ui.KeyEnter, func() {})
	row.OnShortcut(0, ui.KeySpace, func() {})
	if row.Clicked() || (menuOpened && !selected) {
		if a.showGraph {
			// A deliberate click can recover the node after horizontal panning.
			// Leave vertical scrolling alone when the pointer chose a visible row.
			a.revealGraphSelection = true
			c.After(0)
		}
		if stash != nil {
			a.selectStash(stash.SHA)
		} else {
			a.selectCommit(commit.SHA)
		}
	}
}

const (
	historyMergeDisclosureSize  float32 = 18
	historyMergeDisclosureInset         = historyMergeDisclosureSize / 2
)

func (a *app) historyMergeDisclosure(c *ui.Context, sha string, x, rowHeight float32, ground, ink ui.Color) {
	tk := theme.TokensOf(c)
	expanded := a.historyMergeFolds.isExpanded(sha)
	label, icon := a.t("expandMergedCommits"), lucide.ChevronRight()
	if expanded {
		label, icon = a.t("collapseMergedCommits"), lucide.ChevronDown()
	}
	button := ui.ButtonBase(c.Key("merge-disclosure")).Absolute().Left(x-historyMergeDisclosureInset).Top((rowHeight-historyMergeDisclosureSize)/2).
		Size(historyMergeDisclosureSize, historyMergeDisclosureSize).AlignItems(ui.Center).Justify(ui.Center).Radius(5).Background(ground).
		Role(ui.RoleDisclosure).Label(label).Expanded(expanded).Tooltip(label)
	border := ink
	if button.Hovered() || button.Focused() {
		button.Background(tk.PrimaryBg)
		border = tk.Primary
	}
	button.Border(1, border).Children(func() {
		ui.Icon(c, icon).FontSize(12).TextColor(ink).AccessHidden(true)
	})
	if button.Clicked() {
		a.toggleHistoryMerge(sha)
		c.After(0)
	}
}

// panHistoryGraph moves the pinned row's scroll state, which the commit lanes
// also read when drawing. Vertical gestures still reach the list scroller.
func (a *app) panHistoryGraph(ev ui.InputEvent) bool {
	if ev.Kind != ui.InputScroll {
		return false
	}
	dx, dy := ev.DX, ev.DY
	if dx == 0 && ev.Mods&ui.Shift != 0 {
		dx, dy = dy, 0
	}
	if dx == 0 || max(dx, -dx) < max(dy, -dy) {
		return false
	}
	scroll := &a.historyGraphScroll
	if scroll.MaxX <= 0 {
		return false
	}
	scroll.X = min(scroll.MaxX, max(0, scroll.X+dx))
	return true
}

// worktreeRow is the list's first row: the working tree itself. Sublime Merge
// keeps it above the commits so the changes a new commit would take are one
// row from the history they would extend. Dirty worktrees use a dashed
// connection to the checked-out commit in the first graph lane.
func (a *app) historyWorktreeRow(c *ui.Context, index, total int) {
	tk := theme.TokensOf(c)
	selected := a.worktreeSelected
	rowHeight := a.historyRowHeight()
	title := a.t("noChangesShort")
	if n := a.status.WorkspaceFileCount(); n > 0 {
		title = a.changesLabel(n)
	}
	row := ui.ButtonBase(c).Label(title).Row().FillWidth().Height(rowHeight).Padding(0, 8).
		Gap(4).Radius(tk.BorderRadius).AlignItems(ui.Center).
		Role(ui.RoleOption).AccessSelected(selected).PositionInSet(index+1, total)
	if selected {
		row.Background(selectionBg(tk, tk.BgContainer))
	} else if row.Hovered() {
		row.Background(tk.FillTertiary)
	}
	revealNode := selected && a.revealSelection
	a.revealIfSelected(row, selected)
	row.Children(func() {
		if a.showGraph {
			width := float32(24)
			if a.historyGraph != nil && a.historyGraph.Width > width {
				width = a.historyGraph.Width
			}
			contentWidth := width
			if a.historyGraph != nil {
				contentWidth = a.historyGraph.contentWidth()
			}
			link, dirty := a.worktreeGraphConnection(nil, nil)
			if a.historyGraph != nil && a.historyGraph.Worktree != nil {
				link = *a.historyGraph.Worktree
			}
			scroller := ui.ScrollHorizontal(c).Width(width).Height(rowHeight).Shrink(0).
				KeepScrollbars(true).TrackScroll(&a.historyGraphScroll)
			if contentWidth > width+0.5 {
				scroller.Role(ui.RoleScroll).Label(a.t("scrollCommitGraph")).Focusable()
				focused := scroller.Focused()
				scroller.Draw(func(p *ui.Painter, r ui.Rect) {
					// Align the rail with MyGo's horizontal thumb. MyGo keeps
					// ownership of thumb drawing, paging and drag input.
					height := p.Theme().ScrollbarWidth
					if height <= 0 {
						height = 6
					}
					p.Fill(ui.Rect{X: r.X + 2, Y: r.Y + r.H - height - 3, W: r.W - 4, H: height}, tk.BorderSecondary, height/2)
					if focused {
						p.Stroke(r, tk.Primary, 0, 1)
					}
				})
			}
			scroller.Children(func() {
				lane := ui.Box(c).Width(contentWidth).Height(rowHeight).Shrink(0)
				if dirty {
					if link.HasHead {
						lane.Tooltip(a.t("worktreeBasedOnHead"))
						row.Description(a.t("worktreeBasedOnHead"))
					}
					lane.Draw(func(p *ui.Painter, r ui.Rect) { a.drawWorktreeNode(p, r, tk, link) })
				}
			})
		}
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(scaledFont(2)).Children(func() {
			display.Text(c, title, display.TextProps{FontSize: scaledFont(12), Strong: selected, Rows: 1})
			if !a.hideAuthor || !a.hideTime {
				display.Text(c, a.t("commitChanges"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11), Rows: 1})
			}
		})
	})
	// The arrows move the selection while this row keeps the focus. Enter and
	// Space still have to work once the reader clicks the row to open the
	// commit box: they act on the working tree's selected file, as the window
	// shortcuts promise. While a commit row is the selection they are
	// swallowed, or the focused row would act on a file the reader is no
	// longer looking at.
	row.OnShortcut(0, ui.KeyEnter, func() {
		if a.worktreeSelected {
			a.openSelectedFile()
		}
	})
	row.OnShortcut(0, ui.KeySpace, func() {
		if a.worktreeSelected {
			a.toggleSelectedStage()
		}
	})
	if row.Clicked() {
		a.selectWorktree()
		revealNode = true
	}
	if revealNode && a.showGraph {
		if graph := a.historyGraph; graph != nil && graph.Worktree != nil {
			// The square sits near the left edge. Move back only if it is clipped.
			const margin = 5 // Four-DIP half-width plus raster clearance.
			left := max(float32(0), laneX(0, 0)-margin)
			if a.historyGraphScroll.X > left {
				a.historyGraphScroll.X = left
				c.After(0)
			}
		}
	}
}

// headTag marks the commit HEAD points to. Git prints the branch as
// "HEAD -> name", which commitRefs strips, so without this chip the checked
// out branch looks like any other and HEAD only shows in the title bar. The
// chip is a capsule like the refs beside it, tinted from the accent blue
// rather than from a lane, because HEAD is not a line on the graph.
// headTag marks the commit HEAD points at. A narrow history row drops the
// "HEAD" word and keeps the check chip: the branch capsule beside it already
// names the branch, and the tooltip still says which one.
func (a *app) headTag(c *ui.Context, refs string, ground ui.Color, compact bool) {
	head := headBranch(refs)
	if head == "" {
		return
	}
	style := refCapsuleColors(theme.TokensOf(c).Blue, ground)
	label := "HEAD"
	if compact {
		label = ""
	}
	refPill(c, style, lucide.Check(), label).Tooltip(a.tf("headBadgeTip", head))
}

// detailsPlaceholderKey names the sentence the details pane shows when no
// commit is selected. "Select a commit" is the right one whenever the page has
// rows or is about to; "details appear once there is a commit" is only for a
// repository that has no commits at all. An empty file history and a failed
// load also leave nothing selected, but claiming the repository is empty there
// would contradict the list on the left, which says which case it is.
func (a *app) detailsPlaceholderKey() string {
	if a.fileFilter == "" && a.historyErr == "" && len(a.commits) == 0 {
		return "detailsNoCommits"
	}
	return "selectCommit"
}

func (a *app) commitFileList(c *ui.Context) {
	tk := theme.TokensOf(c)
	a.reconcileHistoricalFileSelection()
	switch {
	case a.commitFilesLoading:
		listEmptyState(c, lucide.LoaderCircle(), tk.TextTertiary, a.t("diffLoading"), "")
		return
	case a.commitFilesErr != "":
		ui.Column(c).Gap(6).Children(func() {
			display.Text(c, a.commitFilesErr, display.TextProps{Type: display.TextDanger, FontSize: scaledFont(11), Rows: 3})
			if workbenchButton(c, a.t("retry"), buttonNormal, false).Clicked() {
				a.loadCommitDetail(true)
			}
		})
		return
	case len(a.commitFiles) == 0:
		// Nothing changed here at all, which is different from a change that
		// has no text (a submodule pointer): say which one it is.
		listEmptyState(c, lucide.FileDiff(), tk.TextTertiary, a.t("noCommitFiles"), "")
		return
	}
	// A row's menu can replace commitFiles while this frame is still drawing.
	// Keep the row count, heights and indexes on the same immutable snapshot.
	if a.selectedStash != "" && a.stashTrackedCount < len(a.commitFiles) {
		for i, files := range [][]gitrepo.ChangedFile{a.commitFiles[:a.stashTrackedCount], a.commitFiles[a.stashTrackedCount:]} {
			if len(files) == 0 {
				continue
			}
			group := c.Key(i)
			title := a.t("changes")
			if i == 1 {
				title = a.t("fileGroupUntracked")
			}
			display.Text(group, fmt.Sprintf("%s (%d)", title, len(files)), display.TextProps{Strong: true, Type: display.TextSecondary, FontSize: scaledFont(11)}).Padding(5, 4)
			a.commitFileRows(group, files)
		}
		return
	}
	display.Text(c, fmt.Sprintf("%s (%d)", a.t("changes"), len(a.commitFiles)), display.TextProps{Strong: true, Type: display.TextSecondary, FontSize: scaledFont(11)}).Padding(5, 4)
	a.commitFileRows(c, a.commitFiles)
}

func (a *app) commitFileRows(c *ui.Context, files []gitrepo.ChangedFile) {
	tk := theme.TokensOf(c)
	heights := make([]float32, len(files))
	reveal, remembered := -1, -1
	for i, file := range files {
		heights[i] = fileRowHeight() + fileRowGap
		if a.detailFileRemembered != nil && a.detailFileRememberedContext == a.currentDetailContext() && a.detailFileRemembered.Path == file.Path {
			remembered = i
		}
		if a.detailFileFocus != nil && a.detailFileFocusContext == a.currentDetailContext() && a.detailFileFocus.Path == file.Path {
			reveal = i
		}
	}
	origin := a.historicalFileOrigin()
	compact := a.page == "history" && a.historyFilesWidth() < scaledFont(130)
	ui.Column(c).FillWidth().Role(ui.RoleList).Multiselectable(true).Children(func() {
		virtualPatchRows(c, a.commitFilesViewport, &a.commitFilesScroll, heights, reveal, func(rowContext *ui.Context, i int) {
			file := files[i]
			ui.Column(rowContext).FillWidth().Height(heights[i]).Shrink(0).Children(func() {
				primary := a.detailTabShown() == detailTabFile && a.detailTabs().Active.File.Path == file.Path
				selected := a.historicalFileSelected(file.Path)
				// The original position and full raw path survive virtualization.
				row := fileRowShell(rowContext, tk, file.Path, selected).
					PositionInSet(i+1, len(files)).Tooltip(file.Path)
				if compact {
					row.Padding(0, 3).Gap(4)
				}
				a.wireHistoryFileKeys(row, detailFileKey{Path: file.Path})
				wasFocused := ui.Local(row, "file-was-focused", func() bool { return false })
				if row.Focused() {
					key := detailFileKey{Path: file.Path}
					a.detailFileRemembered = &key
					a.detailFileRememberedContext = a.currentDetailContext()
					if !*wasFocused {
						row.ScrollIntoView()
					}
				}
				*wasFocused = row.Focused()
				row.Children(func() {
					fileRowPath(rowContext, file.Path, primary)
				})
				a.fileDetailMenu(rowContext, row, file.Path)
				menuOpen := *ui.Local(row, "contextMenu", func() bool { return false })
				wasOpen := ui.Local(row, "historical-file-menu-open", func() bool { return false })
				if menuOpen && !*wasOpen {
					a.selectHistoricalFile(origin, file, 0, true)
				}
				*wasOpen = menuOpen
				if row.Clicked() {
					a.selectHistoricalFile(origin, file, row.ClickModifiers(), false)
				}
			})
		}, remembered)
	})
}

func changedFileCode(status string) string {
	if status == "" {
		return "?"
	}
	return status[:1]
}

func (a *app) commitDiffView(c *ui.Context) {
	switch {
	case a.commitDiffByFile && (a.detailTabs().Active.Summary || a.commitFileTooLarge):
		title, hint := "largeCommitDiff", "largeCommitChooseFile"
		if a.commitFileTooLarge {
			title, hint = "largeCommitFileDiff", "largeCommitExportFile"
		} else if a.detailsChangesHidden {
			hint = "largeCommitShowFiles"
		}
		ui.Column(c).FillWidth().Padding(8, 12).Gap(4).Children(func() {
			display.Text(c, a.t(title), display.TextProps{FontSize: scaledFont(12)}).FillWidth()
			display.Text(c, a.t(hint), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).FillWidth()
		})
		return
	case a.commitDiffLoading:
		a.diffPlaceholder(c, lucide.LoaderCircle(), a.t("diffLoading"))
		return
	case a.commitDiffErr != "":
		if a.commitFilesErr != "" {
			// The file list failed first, and its column already shows the
			// error with the one retry that reloads both; repeating the same
			// text here read as two separate failures.
			a.diffPlaceholder(c, lucide.AlertTriangle(), a.t("diffBlockedByFiles"))
			return
		}
		tk := theme.TokensOf(c)
		ui.Column(c).Fill().Center().Gap(8).Padding(24).Children(func() {
			ui.Icon(c, lucide.AlertTriangle()).FontSize(scaledFont(22)).TextColor(tk.TextTertiary)
			display.Text(c, a.commitDiffErr, display.TextProps{Type: display.TextDanger, FontSize: scaledFont(12), Rows: 4}).
				MaxWidth(520).TextAlign(ui.Center)
			// A failed patch used to be a dead end: tapping the same commit
			// again returned because its files were already loaded.
			if workbenchButton(c, a.t("retry"), buttonNormal, false).Clicked() {
				if a.commitDiffByFile {
					a.loadLargeCommitTab()
					a.invalidate()
				} else {
					a.loadCommitDetail(true)
				}
			}
		})
		return
	case len(a.commitDiff.Lines) == 0:
		a.diffPlaceholder(c, lucide.FileDiff(), a.t("diffEmpty"))
		return
	}
	ui.Row(c.Key("commit-patch-layout")).FillWidth().Shrink(0).Justify(ui.End).Padding(0, 4, 4, 4).Children(func() {
		a.patchLayoutButton(c, nil)
	})
	// File-history actions in headers or rows can reset the live patch mid-frame.
	diff := a.commitDiff
	tabs := a.detailTabs()
	sections := a.visibleCommitSections()
	widths := a.historyPatchWidths(c, &tabs.patchWidths, sections, diff.Lines)
	findLine := a.diffFind.revealLine(detailFileKey{})
	sectionRows := make([]patchRows, len(sections))
	for i, section := range sections {
		sectionRows[i] = a.commitSectionPatchRows(diff.Lines, section)
	}
	if tabs.PatchScroll == nil {
		tabs.PatchScroll = make(map[detailTabKey]*ui.ScrollState)
	}
	patchScroll := tabs.PatchScroll[tabs.Active]
	if patchScroll == nil {
		patchScroll = &ui.ScrollState{}
		tabs.PatchScroll[tabs.Active] = patchScroll
	}
	layout.ScrollArea(c.Key(patchScroll), layout.ScrollAreaProps{Direction: layout.Horizontal, State: patchScroll}, func() {
		// The whole stacked diff is one selectable container, so a drag across
		// rows selects the code as text and Cmd+C copies it. Rows stay plain
		// for that reason: text inside a button is not selectable.
		ui.Column(c).Width(widths.canvas).AlignSelf(ui.Start).Selectable().Children(func() {
			sectionHeights := make([]float32, len(sections))
			revealSection, pressedSection := -1, -1
			for at, sec := range sections {
				sectionHeights[at] = diffFileHeaderH
				if a.commitCollapsed[sec.Path] {
					continue
				}
				rows := sectionRows[at]
				if previewFor(sec.Path, sec.Binary) != textPreview {
					sectionHeights[at] += previewSummaryHeight
					continue
				}
				for i := range rows.len() {
					row := rows.at(i)
					if a.commitDragHeld && rowContainsLine(row, a.commitPressIndex) {
						pressedSection = at
					}
					sectionHeights[at] += rows.height(a, i)
					if rowContainsLine(row, findLine) || a.commitReveal && rowContainsLine(row, a.commitCursor) {
						revealSection = at
					}
				}
			}
			scroll := &a.commitDiffScroll
			if a.detailTabs().Active.Summary {
				scroll = &a.commitInfoScroll
			}
			virtualPatchRows(c, a.commitDiffViewport, scroll, sectionHeights, revealSection, func(section *ui.Context, at int) {
				sec := sections[at]
				sectionContext := section.Key(sec.Start)
				a.commitDiffFileHeader(sectionContext, sec, widths.canvas)
				if a.commitCollapsed[sec.Path] {
					return
				}
				if kind := previewFor(sec.Path, sec.Binary); kind != textPreview {
					a.previewSummary(sectionContext, detailFileKey{Path: sec.Path}, kind)
					return
				}
				rows := sectionRows[at]
				heights := make([]float32, rows.len())
				reveal, pressed := -1, -1
				for i := range heights {
					row := rows.at(i)
					heights[i] = rows.height(a, i)
					if a.commitDragHeld && rowContainsLine(row, a.commitPressIndex) {
						pressed = i
					}
					if rowContainsLine(row, findLine) || a.commitReveal && rowContainsLine(row, a.commitCursor) {
						reveal = i
					}
				}
				a.paintPatchRows(sectionContext, rows, widths, a.commitDiffViewport, scroll, heights, reveal, pressed, func(cell *ui.Context, index int, cellWidth float32, side patchSide) {
					line := diff.Lines[index]
					if line.Kind == "hunk" {
						a.commitHunkLine(cell, line, cellWidth, index)
						return
					}
					a.commitDiffLineForSide(cell, line, sec.Path, cellWidth, index, side)
				})
			}, pressedSection)
		})
	}).FillWidth().Height(a.commitDiffHeight()).Shrink(0)
}

// commitDiffFileHeader is the band above one file's patch. It names the file,
// says how many hunks follow and collapses the section on a click. The header
// also points the details at the file, so clicking it highlights the file row
// the section belongs to.
func (a *app) commitDiffFileHeader(c *ui.Context, sec commitDiffSection, width float32) {
	tk := theme.TokensOf(c)
	collapsed := a.commitCollapsed[sec.Path]
	selected := a.commitPath == sec.Path
	row := ui.ButtonBase(c).Label(sec.Path).Row().Width(width).Height(diffFileHeaderH).Shrink(0).
		Padding(0, 8).Gap(6).AlignItems(ui.Center).Background(tk.BgContainer).
		BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary)
	if selected {
		row.Background(selectionBg(tk, tk.BgContainer))
	} else if row.Hovered() {
		row.Background(tk.FillTertiary.Over(tk.BgContainer))
	}
	row.Children(func() {
		icon := lucide.ChevronRight()
		if !collapsed {
			icon = lucide.ChevronDown()
		}
		ui.Icon(c, icon).FontSize(scaledFont(12)).TextColor(tk.TextTertiary).Shrink(0)
		ui.Icon(c, lucide.FileDiff()).FontSize(scaledFont(12)).TextColor(tk.TextTertiary).Shrink(0)
		// A closed section says how much it hides; the count joins the path in
		// one paragraph because the header scrolls sideways with the patch and a
		// right-aligned note would sit past the longest line.
		label := sec.Path
		if collapsed {
			label = fmt.Sprintf("%s · %s", sec.Path, a.hunksLabel(countHunks(a.commitDiff, sec)))
		}
		display.Text(c, label, display.TextProps{Code: true, FontSize: scaledFont(10), Strong: selected}).
			Font(a.patchFontFamily()).SingleLine().Grow(1).Basis(0).MinWidth(0)
	})
	if row.Clicked() {
		a.commitPath = sec.Path
		a.toggleCommitSection(sec.Path)
	}
	// The same file actions the changed-files list offers, on the header the
	// reader is looking at.
	a.fileDetailMenu(c, row, sec.Path)
}

// countHunks is how many hunk headers a section holds, as the collapsed header
// reports it. m is the combined diff the section indexes into.
func countHunks(diff gitrepo.Diff, sec commitDiffSection) int {
	hunks := 0
	for i := sec.Start; i < sec.Start+sec.Lines && i < len(diff.Lines); i++ {
		if diff.Lines[i].Kind == "hunk" {
			hunks++
		}
	}
	return hunks
}

// diffLineText is the line's content as the file has it, without the
// diff's +/- marker, which only says what the row is.
func diffLineText(line gitrepo.DiffLine) string {
	switch {
	case (line.Kind == "add" || line.Kind == "del") && len(line.Text) > 0:
		return line.Text[1:]
	case line.Kind == "context":
		return strings.TrimPrefix(line.Text, " ")
	}
	return line.Text
}

// commitDiffLine draws one committed-diff row. The commit's content is
// read-only, so the row's menu offers the copy and file actions Sublime
// Merge has there rather than staging.
//
// The row is a plain container, not a button: MyGo only lets text be selected
// in a scope that no clickable element owns, so the code has to sit outside
// any button for a drag to select it. The line numbers are the button and the
// handle for whole-line selection, which is also how editors behave.
func (a *app) commitDiffLine(c *ui.Context, line gitrepo.DiffLine, path string, width float32, index int) {
	a.commitDiffLineForSide(c, line, path, width, index, patchBoth)
}

func (a *app) commitDiffLineForSide(c *ui.Context, line gitrepo.DiffLine, path string, width float32, index int, side patchSide) {
	tk := theme.TokensOf(c)
	bg := tk.BgContainer
	switch line.Kind {
	case "add":
		bg = tk.Success.Alpha(0.075).Over(tk.BgContainer)
	case "del":
		bg = tk.Error.Alpha(0.065).Over(tk.BgContainer)
	case "meta":
		bg = tk.FillQuaternary.Over(tk.BgContainer)
	}
	selectable := commitSelectable(line.Kind)
	if selectable && a.commitSel[index] {
		alpha := float32(0.16)
		if index == a.commitCursor {
			alpha = 0.24
		}
		bg = tk.Blue.Alpha(alpha).Over(bg)
	}
	row := ui.Row(c).Width(width).Height(a.patchRowHeight()).Shrink(0).Padding(0, 12, 0, 0).Background(bg)
	if row.Hovered() {
		row.Background(tk.FillTertiary.Alpha(0.35).Over(bg))
	}
	gutter := gutterInk(tk, bg)
	var numbers ui.Element
	row.Children(func() {
		numbers = a.patchGutter(c, line, side, gutter)
		a.diffFind.lineText(c, diffFindLine{index: index}, diffFindText(line), a.patchFontSize(), side == patchNew && line.Kind == "context").TextColor(lineTextColor(tk, line.Kind)).Role(diffCodeRole(line.Kind))
	})
	items := []overlay.MenuItem{{Key: "copy-line", Label: a.t("copyLine"), Icon: lucide.Copy()}}
	if selectable {
		a.trackCommitPress(c, index, side, numbers)
		if numbers.Clicked() {
			if a.commitDragMoved {
				a.commitDragMoved = false
				a.commitDragging = false
				a.commitReleasePress()
			} else {
				a.commitReleasePress()
				a.clickCommitLine(index, numbers.ClickModifiers())
			}
		}
		if targets := a.commitLineTargets(index); len(targets) > 1 {
			items[0].Label = a.copyLinesLabel(len(targets))
			items[0].Key = "copy-lines"
		}
		if index == a.commitCursor && a.commitReveal {
			row.ScrollIntoView()
			a.commitReveal = false
		}
	}
	// The menu describes the section the row is in, not the file the details
	// pane happens to highlight: a row can be right-clicked without the
	// reader first choosing its file.
	if path != "" {
		items = append(items,
			overlay.MenuItem{Key: "copy-path", Label: a.t("copyPath"), Icon: lucide.Copy()},
			overlay.MenuItem{Divider: true},
			overlay.MenuItem{Key: "history", Label: a.t("fileHistory"), Icon: lucide.History()},
			// A file the commit deleted has no version to blame at that
			// revision, so the item stays visible but cannot run.
			overlay.MenuItem{Key: "blame", Label: a.blameMenuLabel(path), Icon: lucide.FileSearch(), Disabled: a.commitBlameDisabled(path)},
		)
	}
	if len(a.commitLineIndices()) > 1 {
		items = append(items,
			overlay.MenuItem{Divider: true},
			overlay.MenuItem{Key: "select-all", Label: a.t("selectAllLines"), Icon: lucide.ListChecks()},
		)
		if len(a.commitSel) > 0 {
			items = append(items,
				overlay.MenuItem{Key: "invert", Label: a.t("invertLines"), Icon: lucide.FlipHorizontal()},
			)
		}
	}
	switch overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: items}) {
	case "copy-line":
		a.copyText(diffLineText(line))
	case "copy-lines":
		a.copyText(diffLinesText(a.commitDiff, a.commitLineTargets(index)))
	case "copy-path":
		a.copyText(path)
	case "select-all":
		a.selectCommitLines(a.commitLineIndices())
	case "invert":
		a.invertCommitLines()
	case "history":
		a.showFileHistory(path)
	case "blame":
		a.openBlameAt(path, a.selectedSHA)
	}
	// Right-clicking outside the selection selects that line, as in the
	// workspace diff, so "copy line" copies the row under the pointer and the
	// menu labels follow on the next frame.
	if selectable && !a.commitSel[index] &&
		*ui.Local(row, "contextMenu", func() bool { return false }) {
		a.selectCommitLines([]int{index})
	}
}

// clickCommitLine applies Finder-like selection rules to a committed diff:
// a plain click selects one row, Shift extends from the anchor, Cmd toggles.
// Hunk and meta rows keep their own menus and take no selection.
func (a *app) clickCommitLine(index int, mods ui.Modifiers) {
	if index < 0 || index >= len(a.commitDiff.Lines) {
		return
	}
	switch a.commitDiff.Lines[index].Kind {
	case "add", "del", "context":
	default:
		return
	}
	if a.commitSel == nil {
		a.commitSel = make(map[int]bool)
	}
	switch {
	case mods&ui.Shift != 0 && a.commitAnchor >= 0 && a.commitAnchor < len(a.commitDiff.Lines):
		a.commitSel = patchSelectionRange(a.commitSelectionOrder(), a.commitAnchor, index)
	case mods&ui.Cmd != 0 || mods&ui.Super != 0:
		if a.commitSel[index] {
			delete(a.commitSel, index)
		} else {
			a.commitSel[index] = true
		}
		a.commitAnchor = index
		if len(a.commitSel) == 0 {
			a.commitSel = nil
		}
	default:
		a.commitSel = map[int]bool{index: true}
		a.commitAnchor = index
	}
	a.commitCursor = index
	a.invalidate()
}

// commitLineTargets returns the rows a copy on index applies to: the whole
// selection when index is part of it, otherwise just index.
func (a *app) commitLineTargets(index int) []int {
	if !a.commitSel[index] {
		return []int{index}
	}
	return sortedSelection(a.commitSel)
}

func (a *app) clearCommitSelection() {
	a.commitSel = nil
	a.commitAnchor = -1
	a.commitCursor = -1
	a.commitReveal = false
	a.commitPressIndex = -1
	a.commitDragHeld = false
	a.commitDragging = false
	a.commitDragMoved = false
}

// commitDiffShown reports whether the commit's patch is on screen at all: the
// details pane is open, Settings has not hidden the Diff column, and the
// patch's own tab is the one in front (the Summary tab covers it). Line
// selection, Cmd+C and the line arrows only act while the rows they describe
// can be seen; hiding the pane hands the keys back to the commit list, and
// showing it again restores the selection.
func (a *app) commitDiffShown() bool {
	return !a.historyDetailsHidden && !a.detailsDiffHidden && !a.worktreeSelected &&
		!a.commitDiffLoading && a.commitDiffErr == "" && len(a.commitDiff.Lines) > 0
}

// commitNoBlamePaths is the file list's contribution to the blame check: a
// commit that deleted a file leaves no version at that revision to read.
func commitNoBlamePaths(files []gitrepo.ChangedFile) map[string]bool {
	noBlame := make(map[string]bool, len(files))
	for _, file := range files {
		if file.Status == "D" {
			noBlame[file.Path] = true
		}
	}
	return noBlame
}

// addSubmoduleNoBlame adds the gitlinks the patch header marked. A submodule
// pointer is recorded as a mode, not as lines, so git blame cannot read it
// either. A path that ends the commit as a regular file can be blamed even
// when the other side of the change deleted a gitlink: the deleted section is
// only the old shape of the same path.
func addSubmoduleNoBlame(noBlame map[string]bool, sections []commitDiffSection) {
	regular := make(map[string]bool, len(sections))
	for _, sec := range sections {
		if sec.NewFile && !sec.Submodule {
			regular[sec.Path] = true
		}
	}
	for _, sec := range sections {
		if sec.Submodule && !regular[sec.Path] {
			noBlame[sec.Path] = true
		}
	}
}

// commitBlameDisabled reports whether the selected commit's version of path
// can be blamed. A deleted file and a submodule pointer both have no text at
// that revision; the map is built once per load so a menu does not walk the
// file list for every row of a large commit.
func (a *app) commitBlameDisabled(path string) bool {
	if a.commitNoBlameSHA != a.selectedSHA {
		return false
	}
	return a.commitNoBlame[path]
}

// blameMenuLabel names the blame action, saying why it is disabled for a file
// the commit cannot blame: a dimmed item with no reason reads as a bug.
func (a *app) blameMenuLabel(path string) string {
	if a.commitBlameDisabled(path) {
		return a.t("blameUnavailable")
	}
	return a.t("blame")
}

// commitSelectable reports whether a committed diff row takes part in the
// line selection. Hunk and meta rows keep their own menus.
func commitSelectable(kind string) bool {
	return kind == "add" || kind == "del" || kind == "context"
}

// commitLineIndices lists the selectable rows in order, the set the
// keyboard walks and a drag extends over.
func (a *app) commitLineIndices() []int {
	var out []int
	for _, sec := range a.visibleCommitSections() {
		if a.commitCollapsed[sec.Path] {
			continue
		}
		for _, index := range sec.Rows {
			if index >= 0 && index < len(a.commitDiff.Lines) && commitSelectable(a.commitDiff.Lines[index].Kind) {
				out = append(out, index)
			}
		}
	}
	return out
}

func (a *app) commitSelectionOrder() []int {
	if !a.splitDiff {
		return a.commitLineIndices()
	}
	var order []int
	for _, section := range a.visibleCommitSections() {
		rows := a.commitSectionPatchRows(a.commitDiff.Lines, section)
		order = rows.appendSelectionOrder(order, commitSelectable)
	}
	return order
}

// moveCommitLineSelection walks the commit diff's selectable rows once a
// line selection exists, keeping the anchor when extend is set, as
// Shift+arrows do in an editor.
func (a *app) moveCommitLineSelection(delta int, extend bool) bool {
	if !a.commitDiffShown() || len(a.commitSel) == 0 {
		return false
	}
	indices := a.commitSelectionOrder()
	if len(indices) == 0 {
		return false
	}
	current := a.commitCursor
	if current < 0 {
		current = a.commitAnchor
	}
	position := 0
	for i, index := range indices {
		if index == current {
			position = i
			break
		}
	}
	if current >= 0 {
		position = clampIndex(position+delta, len(indices))
	}
	target := indices[position]
	if extend {
		a.commitSel = patchSelectionRange(indices, a.commitAnchor, target)
	} else {
		a.commitSel = map[int]bool{target: true}
		a.commitAnchor = target
	}
	a.commitCursor = target
	a.commitReveal = true
	a.invalidate()
	return true
}

// trackCommitPress runs the mouse half of the commit diff's selection: a
// press arms a row, and crossing other rows while it stays down extends the
// range from where the press started.
func (a *app) trackCommitPress(c *ui.Context, index int, side patchSide, body ui.Element) {
	btn, pressed := body.PressedButton()
	if pressed && btn == 0 && body.PressedInside() {
		a.commitPressIndex = index
		a.commitPressSide = side
		a.commitDragHeld = true
		a.commitDragging = false
		a.commitDragMoved = false
		// Ask for a frame on every move from here, as the workspace drag
		// does, so the crossed rows rebuild during the drag.
		body.PointerPosition()
		return
	}
	if a.commitPressIndex < 0 {
		return
	}
	_, py, over := body.PointerPosition()
	if index == a.commitPressIndex {
		if side != a.commitPressSide {
			return
		}
		if !pressed {
			a.commitReleasePress()
			return
		}
		a.commitDragHeld = true
		if a.commitDragging {
			a.autoScrollCommitDrag(c, body, py)
		}
		return
	}
	if !a.commitDragHeld || !over {
		return
	}
	if !a.commitDragging {
		a.beginCommitDrag()
	}
	a.dragToCommitLine(index)
}

func (a *app) commitReleasePress() {
	a.commitPressIndex = -1
	a.commitDragHeld = false
	a.commitDragging = false
}

// beginCommitDrag starts a drag range at the pressed row. A drag always
// ranges from the row it was pressed on; a second drag that kept the
// previous anchor would extend from the wrong end.
func (a *app) beginCommitDrag() {
	a.commitDragging = true
	a.commitAnchor = a.commitPressIndex
	a.commitSel = map[int]bool{a.commitPressIndex: true}
}

// autoScrollCommitDrag scrolls the details pane while a commit diff drag
// holds near its top or bottom edge, so rows outside the pane can be
// reached. It asks for the next frame itself, like the workspace drag:
// without pointer moves nothing else would rebuild.
func (a *app) autoScrollCommitDrag(c *ui.Context, body ui.Element, py float32) {
	vp := a.commitDiffViewport
	scroll := &a.commitDiffScroll
	if a.detailTabs().Active.Summary {
		scroll = &a.commitInfoScroll
	}
	if vp.H <= 0 || scroll.MaxY <= 0 {
		return
	}
	pointerY := body.Bounds().Y + py
	step := autoScrollStep(pointerY, vp.Y, vp.Y+vp.H)
	if step == 0 {
		return
	}
	next := scroll.Y + step
	if next < 0 {
		next = 0
	}
	if next > scroll.MaxY {
		next = scroll.MaxY
	}
	if next == scroll.Y {
		return
	}
	scroll.Y = next
	c.AnimationFrame()
}

// dragToCommitLine grows the selection to index without dropping the anchor.
func (a *app) dragToCommitLine(index int) {
	if index < 0 || index >= len(a.commitDiff.Lines) {
		return
	}
	if !commitSelectable(a.commitDiff.Lines[index].Kind) {
		return
	}
	if a.commitAnchor < 0 {
		a.commitAnchor = index
	}
	next := patchSelectionRange(a.commitSelectionOrder(), a.commitAnchor, index)
	same := len(next) == len(a.commitSel)
	if same {
		for i := range next {
			if !a.commitSel[i] {
				same = false
				break
			}
		}
	}
	if same && a.commitCursor == index {
		return
	}
	a.commitSel = next
	a.commitCursor = index
	a.commitDragMoved = true
	a.invalidate()
}

// commitHunkLine draws a hunk header of a committed diff. The commit is
// read-only, so its menu copies the hunk rather than staging it.
func (a *app) commitHunkLine(c *ui.Context, line gitrepo.DiffLine, width float32, index int) {
	tk := theme.TokensOf(c)
	row := ui.ButtonBase(c).Label(line.Text).Row().Width(width).Height(24).Shrink(0).
		Padding(0, 12).AlignItems(ui.Center).Justify(ui.Start).
		Background(tk.Info.Alpha(0.06).Over(tk.BgContainer))
	row.Children(func() {
		a.diffFind.lineText(c, diffFindLine{index: index}, line.Text, scaledFont(10.5)).
			TextColor(tk.Info)
	})
	menu := overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: []overlay.MenuItem{
		{Key: "copy", Label: a.t("copyHunk"), Icon: lucide.Copy()},
	}})
	if menu == "copy" {
		a.copyText(diffHunkText(a.commitDiff, line.Hunk))
	}
}
