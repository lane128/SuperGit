package lobedesk

import (
	"context"
	"errors"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

// Keep the popover within the workbench even at the largest font size.
const syncCommitPageSize = 6

type syncCommitPage struct {
	repo                                *gitrepo.Repository
	epoch                               uint64
	head, branch, upstream, upstreamTip string
	incoming                            bool
	offset                              int
	cancel                              context.CancelFunc
	commits                             []gitrepo.Commit
	more                                bool
	err                                 string
}

func (a *app) dismissSyncCommits() {
	if p := a.syncCommits; p != nil && p.cancel != nil {
		p.cancel()
	}
	a.syncCommits = nil
}

func (a *app) syncCommitsCurrent(p *syncCommitPage) bool {
	return a.syncCommits == p && a.syncOpen && a.activeRepositorySession() == a && !a.localWatch.closed &&
		!a.busy && !a.loading && a.loadSeq == p.epoch && a.repo == p.repo && a.status.Head == p.head &&
		a.status.Branch == p.branch && a.status.Upstream == p.upstream
}

func (a *app) readSyncCommits(incoming bool, upstreamTip string, offset int) {
	a.dismissSyncCommits()
	if a.ctx == nil || a.repo == nil {
		return
	}
	status := a.status
	ctx, cancel := context.WithCancel(a.ctx)
	p := &syncCommitPage{repo: a.repo, epoch: a.loadSeq, head: status.Head, branch: status.Branch, upstream: status.Upstream,
		incoming: incoming, upstreamTip: upstreamTip, offset: offset, cancel: cancel}
	a.syncCommits = p
	go func() {
		defer cancel()
		var err error
		if upstreamTip == "" {
			upstreamTip, err = p.repo.TrackingTip(ctx, status)
		}
		var commits []gitrepo.Commit
		var more bool
		if err == nil {
			include, exclude := status.Head, upstreamTip
			if incoming {
				include, exclude = exclude, include
			}
			commits, more, err = p.repo.TrackingCommits(ctx, include, exclude, offset, syncCommitPageSize)
		}
		a.hopToMain(func() {
			if !a.syncCommitsCurrent(p) {
				return
			}
			p.cancel = nil
			p.upstreamTip, p.commits, p.more = upstreamTip, commits, more
			if errors.Is(err, gitrepo.ErrTrackingChanged) {
				p.err = a.t("syncTrackingChanged")
			} else if err != nil {
				p.err = err.Error()
			}
			a.invalidate()
		})
	}()
	a.invalidate()
}

func syncListItem(c *ui.Context, menu ui.MenuParts, label string, disabled bool) ui.Element {
	tk := theme.TokensOf(c)
	item := menu.Item().Label(label).Disabled(disabled).Padding(5, 10).Radius(tk.BorderRadiusSM).MinHeight(28).MinWidth(0)
	if disabled {
		item.TextColor(tk.TextDisabled)
	} else if item.Highlighted() || item.Pressed() {
		item.Background(tk.FillTertiary)
	}
	return item
}

func (a *app) syncCommitList(c *ui.Context, menu ui.MenuParts, p *syncCommitPage) {
	tk := theme.TokensOf(c)
	action := func(label string, disabled bool) bool {
		item := syncListItem(c, menu, label, disabled).KeepOpen()
		item.Children(func() { ui.Text(c, label).FontSize(scaledFont(12)).SingleLine() })
		return item.Clicked() && !disabled
	}
	if action(a.t("syncBack"), false) {
		a.dismissSyncCommits()
		return
	}
	title := a.t("syncOutgoing")
	if p.incoming {
		title = a.t("syncIncoming")
	}
	ui.Text(c, title).Padding(6, 10).FontSize(scaledFont(12)).FontWeight(600)
	switch {
	case p.cancel != nil:
		ui.Text(c, a.t("syncCommitsLoading")).Padding(6, 10).TextColor(tk.TextSecondary)
	case p.err != "":
		ui.Text(c, a.t("commitsReadFailed")).Padding(4, 10).TextColor(tk.Error)
		ui.Text(c, p.err).Padding(4, 10).FontSize(scaledFont(11)).SingleLine().Tooltip(p.err)
		if action(a.t("retry"), false) {
			a.readSyncCommits(p.incoming, p.upstreamTip, p.offset)
		}
	case len(p.commits) == 0:
		ui.Text(c, a.t("syncNoCommits")).Padding(6, 10).TextColor(tk.TextSecondary)
	default:
		for _, commit := range p.commits {
			label := shortSHA(commit.SHA) + " · " + commit.Subject
			item := syncListItem(c, menu, label, false).AlignSelf(ui.Stretch).Gap(6).Description(commit.SHA).Tooltip(label)
			item.Children(func() {
				ui.Text(c, shortSHA(commit.SHA)).Shrink(0).FontSize(scaledFont(11)).TextColor(tk.TextSecondary)
				ui.Text(c, commit.Subject).MinWidth(0).Shrink(1).SingleLine().FontSize(scaledFont(13))
			})
			if item.Clicked() && a.syncCommitsCurrent(p) {
				a.openPage("history")
				if !a.historyReloading && historyLists(a.historySource(), commit.SHA) {
					a.showCommit(commit.SHA)
				} else {
					// A hidden ref, file filter or partial traversal may omit the
					// target. Open the chosen SHA using History's revision scope.
					a.applyHistoryRef(historyRef{Kind: historyRefCommit, Name: commit.SHA})
				}
			}
		}
		if p.offset > 0 || p.more {
			menu.Group(func(row ui.Element) {
				row.Gap(4)
				if action(a.t("syncPreviousPage"), p.offset == 0) {
					a.readSyncCommits(p.incoming, p.upstreamTip, p.offset-syncCommitPageSize)
				}
				if action(a.t("syncNextPage"), !p.more) {
					a.readSyncCommits(p.incoming, p.upstreamTip, p.offset+syncCommitPageSize)
				}
			})
		}
	}
}
