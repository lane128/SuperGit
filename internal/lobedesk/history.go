package lobedesk

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
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
// share, and neither side shrinks below its minimum. The details minimum is
// the files column's 180 plus the patch's 300, so the patch keeps its own
// minimum at the smallest window instead of falling back to an even split.
const (
	historySplitDefault   float32 = 0.45
	historySplitMinList   float32 = 260
	historySplitMinDetail float32 = 487
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
	a.historyLoadingMore = true
	a.invalidate()
	go func() {
		page, err := repo.HistoryPage(ctx, tips, offset, 200)
		a.hopToMain(func() {
			if a.historyPageStale(seq, repo, reload) || a.fileFilter != "" {
				return
			}
			a.historyLoadingMore = false
			if err != nil {
				// Partial results remain browsable. HasMore stays true so an
				// interrupted read cannot be mistaken for the traversal's end.
				a.historyPageErr = err.Error()
				a.invalidate()
				return
			}
			a.commits = append(a.commits, page.Commits...)
			a.historyTips = page.Tips
			a.historyMore = page.HasMore
			if !a.historyMore {
				previous := a.selectedSHA
				a.reconcileSelection()
				a.refreshSelectedCommit = gitrepo.Commit{}
				if a.selectedSHA != previous && a.selectedSHA != "" {
					a.loadCommitDetail(true)
				}
			}
			if sha := a.pendingJumpSHA; sha != "" {
				a.pendingJumpSHA = ""
				a.jumpToCommit(sha)
			}
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
	if a.historyRef.Kind != "" {
		found := false
		for _, commit := range a.historySource() {
			if commit.SHA == sha {
				found = true
				break
			}
		}
		if !found {
			a.historyRef = historyRef{}
			// The pinned tip belongs to the ref that just gave way; a later
			// reload would otherwise keep paging that ref's walk behind a header
			// that says "all refs".
			a.historyPinned = ""
			a.pendingJumpSHA = sha
			a.reloadHistoryFromTip("")
			a.invalidate()
			return
		}
	}
	a.jumpToCommit(sha)
}

// jumpToCommit selects a commit in the history list, loading older pages
// until it appears. A parent link can point past the loaded window, so the
// jump keeps paging instead of stopping at 200 rows.
func (a *app) jumpToCommit(sha string) {
	if a.repo == nil || sha == "" {
		return
	}
	// A jump is about one commit, so a search that hides it would make the
	// link look broken. Clearing the query also makes paging report the real
	// end of history.
	a.historyQuery = ""
	find := func(commits []gitrepo.Commit) bool {
		for _, commit := range commits {
			if commit.SHA == sha {
				return true
			}
		}
		return false
	}
	if a.fileFilter != "" && !find(a.fileCommits) {
		// A parent of a file's commit may be older than the file itself, so
		// the jump continues in the full history.
		a.resetFileHistory()
		a.fileFilter = ""
		a.fileCommits = nil
		a.fileHistoryErr = ""
		a.resetHistoryTraversal()
		a.loadMoreHistory()
	}
	if find(a.historySource()) {
		a.pendingJumpSHA = ""
		a.revealSelection = true
		a.selectCommit(sha)
		a.invalidate()
		return
	}
	if a.historyLoadingMore {
		a.pendingJumpSHA = sha
		return
	}
	if !a.historyMore {
		a.pendingJumpSHA = ""
		a.queue("warning", a.t("commitNotLoaded"), shortSHA(sha))
		a.invalidate()
		return
	}
	a.pendingJumpSHA = sha
	a.loadMoreHistory()
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

func (a *app) historyPage(c *ui.Context) {
	source := a.historySource()
	subtitle := a.commitsLabel(len(source))
	commits := filterCommits(source, a.historyQuery)
	// The list, the selection and the details pane all index into these rows;
	// cache them for the frame instead of walking and filtering three times.
	a.historyRowsCache, a.historyRowsFrame, a.historyRowsSet = commits, a.frameSeq, true
	if strings.TrimSpace(a.historyQuery) != "" {
		subtitle = a.commitsLabel(len(commits))
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
	if !a.worktreeSelected && (a.selectedSHA == "" || a.hasSelectedIn(source) || !more) {
		if sha := historySelection(commits, a.selectedSHA); sha != "" && sha != a.selectedSHA {
			a.selectCommit(sha)
		}
	}
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
			}).MaxWidth(220).Tooltip(a.fileFilter)
			clear := action.ActionIcon(c, lucide.X(), action.ActionIconProps{
				Size: theme.Small, Title: a.t("showAllHistory"),
			})
			if clear.Clicked() {
				a.clearFileFilter()
			}
		}
		a.historyDetailsToggle(c)
		ui.Box(c).Width(292).Shrink(0).Children(func() {
			field.SearchBar(c, &a.historyQuery, field.SearchBarProps{
				Placeholder: a.t("searchHistory"),
				ShortKey:    "mod+f",
				Size:        theme.Small,
				Variant:     field.InputFilled,
			})
		})
	})
	row := ui.Row(c).Grow(1).Basis(0).MinHeight(0).AlignItems(ui.Stretch)
	// The row has no width on the page's first frame; the panes then take the
	// stored share of whatever room the row gives them, and from the second
	// frame the drag can turn pointer movement into a share of the real width.
	a.historySplit.rowWidth = row.Bounds().W
	row.Children(func() {
		a.commitList(c)
		// Hiding the details gives the list the whole row: no divider, no
		// details, and the header's toggle brings the pane back.
		if a.historyDetailsHidden {
			return
		}
		a.splitHandle(c, &a.historySplit, historySplitDefault,
			historySplitMinList, historySplitMinDetail, a.t("resizeHistoryPanes"))
		a.historyDetailsPane(c)
	})
}

// historyShare is how much of the pane row the commit list takes. The minimum
// widths win over the stored share when the window is too narrow for both,
// and the details get the rest.
func (a *app) historyShare() float32 {
	return a.historySplit.fraction(historySplitDefault, historySplitMinList, historySplitMinDetail)
}

// historyDetailsToggle collapses or restores the details pane. The control
// lives in the page header because that is the one strip that exists in both
// states; a close button on the pane itself would vanish with the pane.
func (a *app) historyDetailsToggle(c *ui.Context) {
	icon, title := lucide.PanelRightClose(), a.t("hideCommitDetails")
	if a.historyDetailsHidden {
		icon, title = lucide.PanelRightOpen(), a.t("showCommitDetails")
	}
	button := action.ActionIcon(c, icon, action.ActionIconProps{Size: theme.Small, Title: title})
	if button.Clicked() {
		a.setHistoryDetailsHidden(!a.historyDetailsHidden)
		a.persistSettings()
		a.invalidate()
	}
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
	case len(source) == 0:
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
		ring := ui.Transparent
		if list.FocusWithin() {
			ring = tk.Blue.Alpha(0.35)
		}
		list.Border(1, ring)
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
				graph = newCommitGraph(commits, a.lanePalette(tk))
				if link, ok := a.worktreeGraphConnection(commits, graph); ok {
					graph.Worktree = &link
					graph.Width += worktreeGraphGutter
				}
				if width := a.commitListWidth(); width > 0 {
					// The graph keeps a quarter of the list: at 980 the column
					// is under 310 wide, and eight lanes at their natural 32
					// points would take almost all of it. Lanes pack closer
					// before they push the subject out.
					graph.FitWidth(width * 0.26)
				}
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
				// The working tree's row stays at the top of every list state:
				// it is where a new commit comes from, and a search that hides
				// every commit, or a repository with no commits yet, must not
				// take the commit box away with it. It sits in the same list
				// role as the commits, so it is read as the list's first row.
				ui.Column(c).Fill().Role(ui.RoleList).Children(func() {
					// The pinned row is inset like the rows the scroll area
					// holds, so its tint lines up with theirs.
					ui.Column(c).FillWidth().PaddingX(6).Children(func() {
						a.historyWorktreeRow(c, 0, len(commits)+1)
					})
					if state == historyListEmpty {
						a.diffPlaceholder(c, lucide.History(), a.t("noHistory"), a.t("noHistoryHint"))
					} else if state == historyListNoMatch {
						a.diffPlaceholder(c, lucide.SearchSlash(), a.t("searchNoMatch"))
					} else {
						scroll := layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.historyScroll}, func() {
							// No gap between rows: the lane lines are drawn row by row,
							// and a gap would cut them into dashes.
							ui.Column(c).FillWidth().Padding(6).Gap(0).Children(func() {
								heights := make([]float32, len(commits))
								reveal := -1
								for i := range commits {
									heights[i] = a.historyRowHeight()
									if a.revealSelection && commits[i].SHA == a.selectedSHA {
										reveal = i
									}
								}
								virtualPatchRows(c, a.historyViewport, &a.historyScroll, heights, reveal, func(rowContext *ui.Context, i int) {
									a.commitRow(rowContext.Key(commits[i].SHA), commits[i], i, 1, len(commits)+1, graph)
								})
							})
						}).Grow(1).Basis(0).MinHeight(0)
						a.historyViewport = scroll.Bounds()
					}
				})
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

func (a *app) showFileHistory(path string) {
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
	a.fileHistoryTips = nil
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

// commitSubjectNeedsTooltip estimates whether the subject has to clip: the
// graph, the capsule and the row's padding take their share first. A tooltip
// on every row would cover the next commit, which is why the row itself has
// none.
func commitSubjectNeedsTooltip(width float32, subject string) bool {
	if width <= 0 {
		return false
	}
	available := width - 180
	if available < 0 {
		available = 0
	}
	return float32(len([]rune(subject)))*6.5 > available
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

// lead is how many rows sit above the commits in the same list -- the working
// tree's row -- so the place a reader hears counts that row too.
func (a *app) commitRow(c *ui.Context, commit gitrepo.Commit, index, lead, total int, graph *commitGraph) {
	tk := theme.TokensOf(c)
	selected := a.selectedSHA == commit.SHA
	rowHeight := a.historyRowHeight()
	listWidth := a.commitListWidth()
	// No tooltip on the row itself: the subject is already on the row, and a
	// bubble repeating it only covers the next commit under the pointer. The
	// ref capsules keep theirs -- those names can be elided.
	// The list chooses one commit at a time and the details follow it, so the
	// row reports itself as a selectable option rather than a plain button:
	// only an option can say which row the arrows are on through
	// aria-selected, as the branch list does.
	row := ui.ButtonBase(c).Label(commit.Subject).Row().FillWidth().Height(rowHeight).Padding(0, 8).
		Gap(4).Radius(tk.BorderRadius).AlignItems(ui.Center).
		Role(ui.RoleOption).AccessSelected(selected).PositionInSet(index+lead+1, total)
	// A merge commit's shape says it merged two histories, and a screen reader
	// hears only the title; the description carries that one state the row
	// cannot show as text.
	if len(commit.Parents) > 1 {
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
		if a.showGraph {
			width := graph.Width
			if width <= 0 {
				width = 24
			}
			ui.Box(c).Width(width).Height(rowHeight).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
				if index >= len(graph.Spans) || index >= len(graph.Rows) {
					return
				}
				center := r.Y + r.H/2
				x0 := int(r.X + graph.worktreeGutter())
				if graph.Worktree != nil {
					sourceY := a.worktreeGraphBounds.Y + a.worktreeGraphBounds.H/2
					points := graph.Worktree.rowPoints(graph, index, r, sourceY)
					line, _ := a.graphStyle()
					strokeWorktreeDash(p, points, line, graphInk(tk, graph.LaneColor(graph.Worktree.Color)))
				}
				gap := graph.LaneGap()
				rowTop, rowBottom := r.Y, r.Y+r.H
				graphRow := graph.Rows[index]
				line, node := a.graphStyle()
				rowStroke := line
				if selected {
					rowStroke += 1
				}
				for _, span := range graph.Spans[index] {
					col := graphInk(tk, graph.LaneColor(span.Color))
					sourceX, parentX, viaX := laneXGap(x0, span.SourceLane, gap), laneXGap(x0, span.ParentLane, gap), laneXGap(x0, span.ViaLane, gap)
					// Row heights are uniform, so the two nodes sit a whole
					// number of rows above and below this row's own centre.
					sourceY := center - float32(index-span.Start)*r.H
					parentY := center + float32(span.End-index)*r.H
					points := graphEdgePoints(sourceX, sourceY, viaX, parentX, parentY, r.H, rowTop, rowBottom)
					if len(points) > 0 {
						p.StrokePolyline(points, rowStroke, col)
					}
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
				switch shape {
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
		}
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(scaledFont(2)).Children(func() {
			titleGap := float32(6)
			if rowRefsNarrow(listWidth) {
				titleGap = 4
			}
			ui.Row(c).FillWidth().Gap(titleGap).AlignItems(ui.Center).Children(func() {
				titleText := display.Text(c, commit.Subject, display.TextProps{FontSize: scaledFont(12), Strong: selected, Rows: 1}).Grow(1).MinWidth(0).
					AccessHidden(true)
				if commitSubjectNeedsTooltip(listWidth, commit.Subject) {
					// The title's own bubble, not the row's: a short subject
					// still should not cover the next commit.
					titleText.Tooltip(commit.Subject)
				}
				head := headBranch(commit.Refs)
				compactHead := rowRefsNarrow(listWidth)
				if !compactHead {
					a.headTag(c, commit.Refs, rowBg, false)
				}
				refs := commitRefs(commit.Refs)
				shown, folded := rowRefsLimit(refs, a.historyRef, head, refSlotsForWidth(listWidth))
				for _, ref := range shown {
					badge := a.refBadge(ref)
					if compactHead && ref == head {
						// A narrow row folds the HEAD chip into the branch's
						// own capsule: the check says which branch HEAD is on,
						// and the word "HEAD" would have cost the subject a
						// word. The tooltip keeps the full sentence.
						pill := refPill(c, refCapsuleColors(tk.Blue, rowBg), lucide.Check(), elideMiddle(badge.Label, refBadgeRunes)).
							MaxWidth(rowCapsuleWidthForList(listWidth)).Tooltip(a.tf("headBadgeTip", head))
						a.refBadgeMenu(c, pill, ref)
						continue
					}
					// The capsule wears the color of the line the ref points at,
					// so a label can be traced to its lane; a tag shares the
					// branch's line on that row. HEAD keeps its blue chip beside
					// the title.
					kind, name, filterable := a.parseCommitRef(ref)
					color := tk.TextSecondary
					if filterable {
						color = refBadgeColor(a.lanePalette(tk), graph, index, kind, name)
					} else if family, ok := refColor(tk, badge.Color); ok {
						color = family
					}
					pill := refPill(c, refCapsuleColors(color, rowBg), badge.Icon,
						elideMiddle(badge.Label, refBadgeRunes)).MaxWidth(rowCapsuleWidthForList(listWidth)).Tooltip(badge.Label)
					a.refBadgeMenu(c, pill, ref)
				}
				if len(folded) > 0 {
					labels := make([]string, 0, len(folded))
					for _, ref := range folded {
						labels = append(labels, a.refBadge(ref).Label)
					}
					pill := refPill(c, neutralCapsule(tk, rowBg), nil, "+"+itoa(len(folded))).
						Tooltip(strings.Join(labels, "\n"))
					a.foldedRefsMenu(c, pill, folded)
				}
			})
			if !a.hideAuthor || !a.hideTime {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
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
					if !a.hideTime {
						display.Text(c, formatDateTime(commit.Date), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11.5)}).
							Shrink(0)
					}
				})
			}
		})
	})
	_ = index
	a.commitMenu(c, row, commit)
	// Opening a row's context menu selects the row, so the details on the
	// right describe the commit the menu acts on, as Sublime Merge does. The
	// context menu owns the right click, so this reads the menu's open state
	// rather than target.RightClicked().
	if *ui.Local(row, "contextMenu", func() bool { return false }) && a.selectedSHA != commit.SHA {
		a.selectCommit(commit.SHA)
	}
	// The arrows move the selection while the row the pointer chose keeps the
	// focus; activating that row on Enter or Space would snap the selection
	// back to it. History has no action of its own for either key, so it
	// claims them and leaves the arrow selection alone.
	row.OnShortcut(0, ui.KeyEnter, func() {})
	row.OnShortcut(0, ui.KeySpace, func() {})
	if row.Clicked() {
		a.selectCommit(commit.SHA)
	}
}

// worktreeRow is the list's first row: the working tree itself. Sublime Merge
// keeps it above the commits so the changes a new commit would take are one
// row from the history they would extend. Dirty worktrees use a temporary
// dashed lane; real commits and their graph rows remain unchanged.
func (a *app) historyWorktreeRow(c *ui.Context, index, total int) {
	tk := theme.TokensOf(c)
	selected := a.worktreeSelected
	rowHeight := a.historyRowHeight()
	title := a.t("noChangesShort")
	if n := len(a.status.Entries); n > 0 {
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
	a.revealIfSelected(row, selected)
	row.Children(func() {
		if a.showGraph {
			width := float32(24)
			if a.historyGraph != nil && a.historyGraph.Width > width {
				width = a.historyGraph.Width
			}
			lane := ui.Box(c).Width(width).Height(rowHeight).Shrink(0)
			link, dirty := a.worktreeGraphConnection(nil, nil)
			if a.historyGraph != nil && a.historyGraph.Worktree != nil {
				link = *a.historyGraph.Worktree
			}
			if dirty {
				if link.HasHead {
					lane.Tooltip(a.t("worktreeBasedOnHead"))
					row.Description(a.t("worktreeBasedOnHead"))
				}
				lane.Draw(func(p *ui.Painter, r ui.Rect) { a.drawWorktreeNode(p, r, tk, link) })
			}
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
	heights := make([]float32, len(a.commitFiles))
	reveal, remembered := -1, -1
	for i, file := range a.commitFiles {
		heights[i] = fileRowHeight() + fileRowGap
		if a.detailFileRemembered != nil && a.detailFileRememberedContext == a.currentDetailContext() && a.detailFileRemembered.Path == file.Path {
			remembered = i
		}
		if a.detailFileFocus != nil && a.detailFileFocusContext == a.currentDetailContext() && a.detailFileFocus.Path == file.Path {
			reveal = i
		}
	}
	ui.Column(c).FillWidth().Role(ui.RoleList).Children(func() {
		virtualPatchRows(c, a.commitFilesViewport, &a.commitFilesScroll, heights, reveal, func(rowContext *ui.Context, i int) {
			file := a.commitFiles[i]
			ui.Column(rowContext).FillWidth().Height(heights[i]).Shrink(0).Children(func() {
				selected := a.detailTabShown() == detailTabFile && a.detailTabs().Active.File.Path == file.Path
				// The original position and full raw path survive virtualization.
				row := fileRowShell(rowContext, tk, file.Path, selected).
					PositionInSet(i+1, len(a.commitFiles)).Tooltip(file.Path)
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
					code := changedFileCode(file.Status)
					a.statusTag(rowContext, code, display.TagProps{
						Variant: display.TagFilled,
						Color:   display.TagColor(changedFileColor(file.Status)),
						Size:    theme.Small,
						Round:   true,
					})
					fileRowPath(rowContext, row, file.Path, file.OriginalPath, selected)
				})
				a.fileDetailMenu(rowContext, row, file.Path)
				if row.Clicked() {
					a.selectCommitFile(file.Path)
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

func changedFileColor(status string) string {
	if status == "" {
		return "gray"
	}
	switch status[:1] {
	case "A":
		return "success"
	case "D":
		return "error"
	case "R", "C":
		return "geekblue"
	case "M", "T":
		return "warning"
	default:
		return "gray"
	}
}

func (a *app) commitDiffView(c *ui.Context) {
	switch {
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
				a.loadCommitDetail(true)
			}
		})
		return
	case len(a.commitDiff.Lines) == 0:
		a.diffPlaceholder(c, lucide.FileDiff(), a.t("diffEmpty"))
		return
	}
	width := diffWidth(a.commitDiff)
	layout.ScrollArea(c, layout.ScrollAreaProps{Direction: layout.Horizontal}, func() {
		// The whole stacked diff is one selectable container, so a drag across
		// rows selects the code as text and Cmd+C copies it. Rows stay plain
		// for that reason: text inside a button is not selectable.
		ui.Column(c).Width(width).AlignSelf(ui.Start).Selectable().Children(func() {
			sections := a.visibleCommitSections()
			sectionHeights := make([]float32, len(sections))
			revealSection, pressedSection := -1, -1
			for at, sec := range sections {
				sectionHeights[at] = diffFileHeaderH
				if a.commitCollapsed[sec.Path] {
					continue
				}
				for _, index := range sec.Rows {
					if a.commitDragHeld && a.commitPressIndex == index {
						pressedSection = at
					}
					height := diffRowH
					if a.commitDiff.Lines[index].Kind == "hunk" {
						height = diffHunkH
					}
					sectionHeights[at] += height
					if a.commitReveal && a.commitCursor == index {
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
				a.commitDiffFileHeader(sectionContext, sec, width)
				if a.commitCollapsed[sec.Path] {
					return
				}
				heights := make([]float32, len(sec.Rows))
				reveal, pressed := -1, -1
				for row, index := range sec.Rows {
					if a.commitDragHeld && a.commitPressIndex == index {
						pressed = row
					}
					heights[row] = diffRowH
					if a.commitDiff.Lines[index].Kind == "hunk" {
						heights[row] = diffHunkH
					}
					if a.commitReveal && a.commitCursor == index {
						reveal = row
					}
				}
				virtualPatchRows(sectionContext, a.commitDiffViewport, scroll, heights, reveal, func(row *ui.Context, at int) {
					index := sec.Rows[at]
					line := a.commitDiff.Lines[index]
					if line.Kind == "hunk" {
						a.commitHunkLine(row, line, width)
						return
					}
					a.commitDiffLine(row, line, sec.Path, width, index)
				}, pressed)
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
			SingleLine().Grow(1).Basis(0).MinWidth(0)
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
	row := ui.Row(c).Width(width).Height(diffRowH).Shrink(0).Padding(0, 12, 0, 0).Background(bg)
	if row.Hovered() {
		row.Background(tk.FillTertiary.Alpha(0.35).Over(bg))
	}
	gutter := gutterInk(tk, bg)
	var numbers ui.Element
	row.Children(func() {
		numbers = ui.ButtonBase(c).Label(a.lineGutterLabel(line)).Row().Width(92).Height(diffRowH).Shrink(0).
			Padding(0, 8).Gap(8).AlignItems(ui.Center)
		numbers.Children(func() {
			ui.Text(c, lineNumber(line.Old)).Font("Menlo").FontSize(scaledFont(10)).Width(34).Shrink(0).
				TextAlign(ui.End).TextColor(gutter)
			ui.Text(c, lineNumber(line.New)).Font("Menlo").FontSize(scaledFont(10)).Width(34).Shrink(0).
				TextAlign(ui.End).TextColor(gutter)
		})
		ui.Text(c, strings.ReplaceAll(line.Text, "\t", "    ")).Font("Menlo").FontSize(scaledFont(diffCodeSize)).
			SingleLine().TextColor(lineTextColor(tk, line.Kind)).Role(diffCodeRole(line.Kind))
	})
	items := []overlay.MenuItem{{Key: "copy-line", Label: a.t("copyLine"), Icon: lucide.Copy()}}
	if selectable {
		a.trackCommitPress(c, index, numbers)
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
		lo, hi := a.commitAnchor, index
		if lo > hi {
			lo, hi = hi, lo
		}
		next := make(map[int]bool, hi-lo+1)
		for _, i := range a.commitLineIndices() {
			if i >= lo && i <= hi {
				next[i] = true
			}
		}
		a.commitSel = next
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

// moveCommitLineSelection walks the commit diff's selectable rows once a
// line selection exists, keeping the anchor when extend is set, as
// Shift+arrows do in an editor.
func (a *app) moveCommitLineSelection(delta int, extend bool) bool {
	if !a.commitDiffShown() || len(a.commitSel) == 0 {
		return false
	}
	indices := a.commitLineIndices()
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
		lo, hi := a.commitAnchor, target
		if lo < 0 || lo >= len(a.commitDiff.Lines) {
			lo = target
		}
		if lo > hi {
			lo, hi = hi, lo
		}
		next := make(map[int]bool, hi-lo+1)
		for _, index := range indices {
			if index >= lo && index <= hi {
				next[index] = true
			}
		}
		a.commitSel = next
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
func (a *app) trackCommitPress(c *ui.Context, index int, body ui.Element) {
	btn, pressed := body.PressedButton()
	if pressed && btn == 0 && body.PressedInside() {
		a.commitPressIndex = index
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
	lo, hi := a.commitAnchor, index
	if lo > hi {
		lo, hi = hi, lo
	}
	next := make(map[int]bool, hi-lo+1)
	for _, i := range a.commitLineIndices() {
		if i >= lo && i <= hi {
			next[i] = true
		}
	}
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
func (a *app) commitHunkLine(c *ui.Context, line gitrepo.DiffLine, width float32) {
	tk := theme.TokensOf(c)
	row := ui.ButtonBase(c).Label(line.Text).Row().Width(width).Height(24).Shrink(0).
		Padding(0, 12).AlignItems(ui.Center).
		Background(tk.Info.Alpha(0.06).Over(tk.BgContainer))
	row.Children(func() {
		ui.Text(c, line.Text).Font("Menlo").FontSize(scaledFont(10.5)).SingleLine().
			TextColor(tk.Info)
	})
	menu := overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: []overlay.MenuItem{
		{Key: "copy", Label: a.t("copyHunk"), Icon: lucide.Copy()},
	}})
	if menu == "copy" {
		a.copyText(diffHunkText(a.commitDiff, line.Hunk))
	}
}
