package lobedesk

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

const (
	stashPaneWidth = 300
	// The list and the selected stash's diff keep at least these widths when
	// the divider between them is dragged.
	stashSplitMinList   = 240
	stashSplitMinDetail = 380
	// stashOpeningRoom is the width assumed on the first frame, before the
	// row has been laid out; without it the list would take the whole page
	// for a frame and the detail would vanish.
	stashOpeningRoom = 940
)

func (a *app) stashesLabel(n int, failed bool) string {
	if failed {
		// A failed read leaves no rows, which is not the same as an empty
		// list; the header has to say the count is unknown.
		return a.t("stashesReadFailed")
	}
	if a.locale == localeEnglish {
		if n == 1 {
			return "1 stash"
		}
		return fmt.Sprintf("%d stashes", n)
	}
	return fmt.Sprintf("%d 个 Stash", n)
}

// clearStashDetail drops the selected stash's patch when the list read failed:
// the row it described may not be in the new result, and a patch beside a
// failed list reads as if that stash were still current. A single patch's own
// failure keeps its selection and retry; only the list path calls this.
func (a *app) clearStashDetail() {
	a.selectedStash = ""
	a.stashDiff = gitrepo.Diff{}
	a.stashDiffErr = ""
	a.stashSeq++
}

func (a *app) stashPage(c *ui.Context) {
	tk := theme.TokensOf(c)
	a.pageHeader(c, a.t("stashes"), a.stashesLabel(len(a.stashes), a.stashesErr != ""), func() {
		a.stashSaveMenu(c)
	})
	row := ui.Row(c).Grow(1).Basis(0).MinHeight(0).AlignItems(ui.Stretch)
	// The first frame has no width yet; the opening width stands in so the
	// two panes do not collapse before the layout lands.
	room := row.Bounds().W
	if room <= 0 {
		room = stashOpeningRoom
	}
	a.stashSplit.rowWidth = room
	row.Children(func() {
		ui.Column(c).Grow(a.stashShare()).Basis(0).MinWidth(0).AlignSelf(ui.Stretch).Children(func() {
			// A ring inside the pane marks the focused list without touching
			// the divider on the right.
			list := ui.Column(c).Fill().Focusable()
			ring := ui.Transparent
			if list.FocusWithin() {
				ring = tk.Blue.Alpha(0.35)
			}
			list.Border(1, ring)
			list.Children(func() {
				if a.stashesErr != "" {
					// The failed read leaves no rows to work with, so the
					// error carries the way back: retry the list read.
					ui.Column(c).Padding(12).Gap(6).Children(func() {
						display.Text(c, a.stashesErr, display.TextProps{Type: display.TextDanger, FontSize: scaledFont(11), Rows: 3})
						if workbenchButton(c, a.t("retry"), buttonNormal, false).Clicked() {
							a.reload()
						}
					})
				}
				layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.stashesScroll}, func() {
					ui.Column(c).FillWidth().Padding(7).Gap(2).Children(func() {
						if len(a.stashes) == 0 {
							// A failed list read leaves this empty; the error
							// above says why, so do not suggest there is
							// nothing to stash.
							if a.stashesErr == "" {
								listEmptyState(c, lucide.Archive(), tk.TextTertiary, a.t("noStashes"), a.t("noStashesHint"))
							}
							return
						}
						ui.Column(c).FillWidth().Gap(2).Role(ui.RoleList).Children(func() {
							for i, stash := range a.stashes {
								a.stashRow(c, stash, i+1, len(a.stashes))
							}
						})
					})
				}).Grow(1).Basis(0).MinHeight(0)
			})
		})
		a.splitHandle(c, &a.stashSplit,
			shareForWidth(a.stashSplit.rowWidth, stashPaneWidth), stashSplitMinList, stashSplitMinDetail,
			a.t("resizeStashPanes"))
		a.stashDetail(c)
	})
}

// stashShare is how much of the page the stash list takes; the selected
// stash's diff keeps the rest.
func (a *app) stashShare() float32 {
	return a.stashSplit.fraction(
		shareForWidth(a.stashSplit.rowWidth, stashPaneWidth),
		stashSplitMinList, stashSplitMinDetail)
}

func (a *app) stashSaveMenu(c *ui.Context) {
	tk := theme.TokensOf(c)
	open := &a.stashSaveOpen
	chosen := overlay.DropdownMenu(c, open, overlay.DropdownMenuProps{Items: []overlay.MenuItem{
		{Key: "tracked", Label: a.t("saveStash"), Icon: lucide.Archive()},
		{Key: "untracked", Label: a.t("saveStashUntracked"), Icon: lucide.Archive()},
	}}, func(trigger ui.Element) {
		styleWorkbenchButton(c, trigger, buttonNormal, false, *open)
		trigger.Label(a.t("saveStashAction"))
		trigger.Children(func() {
			ui.Icon(c, lucide.Archive()).FontSize(scaledFont(14)).TextColor(tk.TextSecondary)
			ui.Text(c, a.t("saveStashAction")).SingleLine()
			ui.Icon(c, lucide.ChevronDown()).FontSize(scaledFont(14)).TextColor(tk.TextSecondary)
		})
	})
	switch chosen {
	case "tracked":
		a.promptSaveStash(false)
	case "untracked":
		a.promptSaveStash(true)
	}
}

func (a *app) promptSaveStash(untracked bool) {
	if len(a.status.Entries) == 0 {
		a.queue("warning", a.t("stashNothing"), "")
		a.invalidate()
		return
	}
	a.askPrompt(a.t("saveStashTitle"), a.t("stashMessagePlaceholder"), "", a.t("save"), func(message string) {
		a.performAs(busySaveStash, a.t("saveStashFailed"), a.t("saveStashSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.SaveStash(ctx, strings.TrimSpace(message), untracked)
		})
	})
}

// promptStashFile saves the changes of one file, which is the context menu's
// answer to "I only want this file out of the way". The file path seeds the
// stash message so the entry is recognisable in the stash list.
func (a *app) promptStashFile(entry gitrepo.Entry) {
	if entry.Conflict || a.busy || a.loading {
		return
	}
	untracked := entry.Index == '?' && entry.Worktree == '?'
	a.askPrompt(a.t("saveStashTitle"), a.t("stashMessagePlaceholder"), entry.Path, a.t("save"), func(message string) {
		a.performAs(busySaveStash, a.t("saveStashFailed"), a.t("saveStashSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.SaveStashPaths(ctx, strings.TrimSpace(message), untracked, []string{entry.Path})
		})
	})
}

// stashTitle is the text a stash is listed, confirmed and copied by: Git's
// own message, or the "stash@{n}" reference when the message is empty. The row
// already fell back to the reference; copying read the message alone and put
// an empty string on the clipboard with a "copied" toast over it.
func stashTitle(stash gitrepo.Stash) string {
	if stash.Message != "" {
		return stash.Message
	}
	return stash.Ref
}

func (a *app) stashRow(c *ui.Context, stash gitrepo.Stash, pos, size int) {
	tk := theme.TokensOf(c)
	selected := a.selectedStash == stash.SHA
	message := stashTitle(stash)
	// The same single selection as the branches and commits: an option, so a
	// screen reader hears which stash the details describe.
	row := ui.ButtonBase(c).Label(message).Row().FillWidth().MinHeight(46).Padding(5, 8).
		Gap(8).Radius(tk.BorderRadius).AlignItems(ui.Center).Tooltip(message).
		Role(ui.RoleOption).AccessSelected(selected).PositionInSet(pos, size)
	switch {
	case selected:
		row.Background(selectionBg(tk, tk.BgContainer))
	case row.Hovered():
		row.Background(tk.FillTertiary)
	}
	a.revealIfSelected(row, selected)
	row.Children(func() {
		ui.Box(c).Size(30, 30).Shrink(0).Radius(9).Center().Background(tk.FillTertiary).Children(func() {
			ui.Icon(c, lucide.Archive()).FontSize(scaledFont(15)).TextColor(tk.TextSecondary)
		})
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(2).Children(func() {
			text := display.Text(c, message, display.TextProps{FontSize: scaledFont(12), Strong: selected}).SingleLine().AccessHidden(true)
			if selected {
				text.TextColor(tk.Text)
			}
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				display.Text(c, stash.Ref, display.TextProps{Code: true, FontSize: scaledFont(10), Type: display.TextSecondary})
				// The same timestamp the history list prints: the app keeps one
				// date format everywhere.
				display.Text(c, formatDateTime(stash.Date), display.TextProps{FontSize: scaledFont(10), Type: display.TextSecondary}).
					Shrink(0)
			})
		})
	})
	// The row is a button, and a focused button takes Enter and Space for
	// itself. Neither has a stash action, and activating the row would pull the
	// selection back to the last-clicked stash after the arrows moved it; claim
	// both keys and do nothing, as the history and branch rows do.
	row.OnShortcut(0, ui.KeyEnter, func() {})
	row.OnShortcut(0, ui.KeySpace, func() {})
	a.stashMenu(c, row, stash)
	if row.Clicked() {
		a.selectStash(stash.SHA)
	}
	// Opening the row's menu also selects the stash, so the diff beside it and
	// the menu describe the same entry, as the history list does.
	if *ui.Local(row, "contextMenu", func() bool { return false }) {
		a.selectStash(stash.SHA)
	}
}

func (a *app) stashDetail(c *ui.Context) {
	tk := theme.TokensOf(c)
	ui.Column(c).Grow(1 - a.stashShare()).Basis(0).MinWidth(0).AlignSelf(ui.Stretch).Children(func() {
		// The header describes one stash and offers its actions, so it only
		// exists once a stash is chosen. Without one, a bare divider above the
		// pane's empty state reads as a rendering gap.
		if a.selectedStash != "" {
			a.stashDetailHeader(c, tk)
		}
		switch {
		case a.selectedStash == "":
			a.diffPlaceholder(c, lucide.Archive(), a.t("stashDetailHint"))
		case a.stashDiffLoading:
			a.diffPlaceholder(c, lucide.LoaderCircle(), a.t("diffLoading"))
		case a.stashDiffErr != "":
			ui.Column(c).Fill().Center().Gap(8).Padding(24).Children(func() {
				display.Text(c, a.stashDiffErr, display.TextProps{Type: display.TextDanger, FontSize: scaledFont(12), Rows: 4}).
					MaxWidth(520).TextAlign(ui.Center)
				if workbenchButton(c, a.t("retry"), buttonNormal, false).Clicked() {
					a.loadStashDiff()
				}
			})
		case a.stashDiff.Text == "":
			a.diffPlaceholder(c, lucide.Archive(), a.t("stashEmpty"))
		default:
			width := a.diffWidth(a.stashDiff)
			layout.ScrollArea(c, layout.ScrollAreaProps{Direction: layout.Both}, func() {
				// One selectable scope for the whole patch, as in the other diff
				// views: a drag across lines copies the text it covers.
				ui.Column(c).Width(width).AlignSelf(ui.Start).Shrink(0).Selectable().Children(func() {
					for _, line := range a.stashDiff.Lines {
						a.diffLineFor(c, line, width, false, -1)
					}
				})
			}).Grow(1).Basis(0).MinHeight(0)
		}
	})
}

// stashDetailHeader names the chosen stash and keeps its actions beside it,
// the way Sublime Merge does, instead of hiding them in the list's menu.
func (a *app) stashDetailHeader(c *ui.Context, tk *theme.Tokens) {
	ui.Row(c).Height(36).Shrink(0).Padding(0, 12).Gap(8).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
		// The title can be elided in a narrow pane; the tooltip is where the
		// whole message is read. It used to carry the stash's SHA, which the
		// header never shows and which told the reader nothing about the text
		// under the pointer.
		display.Text(c, a.selectedStashLabel(), display.TextProps{Strong: true, FontSize: scaledFont(12)}).
			SingleLine().Tooltip(a.selectedStashLabel())
		ui.Row(c).Grow(1).Basis(0)
		if len(a.stashDiff.Hunks) > 0 {
			display.Tag(c, a.hunksLabel(len(a.stashDiff.Hunks)), display.TagProps{
				Variant: display.TagBorderless, Size: theme.Small,
			})
		}
		if stash, ok := a.selectedStashObject(); ok {
			blocked := a.busy || a.loading
			if workbenchButton(c, a.t("applyStash"), buttonNormal, blocked).Clicked() {
				a.applyStash(stash)
			}
			if workbenchButton(c, a.t("popStash"), buttonNormal, blocked).Clicked() {
				a.popStash(stash)
			}
			if workbenchButton(c, a.t("dropStashAction"), buttonDanger, blocked).Clicked() {
				a.confirmDropStash(stash)
			}
		}
	})
}

func (a *app) selectedStashLabel() string {
	for _, stash := range a.stashes {
		if stash.SHA == a.selectedStash {
			return stashTitle(stash)
		}
	}
	return a.selectedStash
}

func (a *app) selectedStashObject() (gitrepo.Stash, bool) {
	for _, stash := range a.stashes {
		if stash.SHA == a.selectedStash {
			return stash, true
		}
	}
	return gitrepo.Stash{}, false
}

func (a *app) selectStash(sha string) {
	if sha == "" || (a.selectedStash == sha && a.stashDiff.Text != "") {
		return
	}
	a.selectedStash = sha
	a.stashDiff = gitrepo.Diff{}
	a.stashDiffErr = ""
	a.loadStashDiff()
	a.invalidate()
}

func (a *app) loadStashDiff() {
	if a.repo == nil || a.selectedStash == "" {
		return
	}
	a.stashSeq++
	seq := a.stashSeq
	repo, sha := a.repo, a.selectedStash
	a.stashDiffLoading = true
	a.stashDiffErr = ""
	go func() {
		diff, err := repo.StashDiff(a.ctx, sha)
		a.hopToMain(func() {
			if seq != a.stashSeq || a.repo == nil || a.repo.Root != repo.Root || a.selectedStash != sha {
				return
			}
			a.stashDiffLoading = false
			if err != nil {
				a.stashDiffErr = err.Error()
			} else {
				a.stashDiff = diff
			}
			a.invalidate()
		})
	}()
}

func (a *app) stashMenu(c *ui.Context, row ui.Element, stash gitrepo.Stash) {
	items := []overlay.MenuItem{
		{Key: "apply", Label: a.t("applyStash"), Icon: lucide.ArchiveRestore(), Disabled: a.busy || a.loading},
		{Key: "pop", Label: a.t("popStash"), Icon: lucide.ArchiveX(), Disabled: a.busy || a.loading},
		{Divider: true},
		{Key: "copy", Label: a.t("copyStashMessage"), Icon: lucide.Copy()},
		{Divider: true},
		{Key: "drop", Label: a.t("dropStashAction"), Icon: lucide.Trash2(), Danger: true, Disabled: a.busy || a.loading},
	}
	switch overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: items}) {
	case "apply":
		a.applyStash(stash)
	case "pop":
		a.popStash(stash)
	case "copy":
		a.copyText(stashTitle(stash))
	case "drop":
		a.confirmDropStash(stash)
	}
}

func (a *app) applyStash(stash gitrepo.Stash) {
	a.performAs(busyApplyStash, a.t("applyStashFailed"), a.t("applyStashSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
		return repo.ApplyStash(ctx, stash.SHA)
	})
}

func (a *app) popStash(stash gitrepo.Stash) {
	a.performAs(busyPopStash, a.t("popStashFailed"), a.t("popStashSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
		return repo.PopStash(ctx, stash.Ref, stash.SHA)
	})
}

func (a *app) confirmDropStash(stash gitrepo.Stash) {
	label := stashTitle(stash)
	// Two stashes can carry the same message; the reference and the short SHA
	// are what tell the reader which entry the dialog is about.
	a.askConfirm(a.t("dropStashTitle"), a.tf("dropStashMessage", label, stash.Ref, shortSHA(stash.SHA)), a.t("drop"), true, func() {
		a.performAs(busyDropStash, a.t("dropStashFailed"), a.t("dropStashSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.DropStash(ctx, stash.Ref, stash.SHA)
		})
	})
}
