package desktop

import (
	"context"
	"fmt"
	"slices"

	"github.com/egoist/mygo"
	"supergit/internal/gitrepo"
)

const historyPageSize = 300

type historyPageRequest struct {
	ticket   ticket
	revision uint64
	offset   int
	tips     []string
}

func (a *app) canLoadHistory() bool {
	return a.repo != nil && a.page == "history" && !a.busy() && !a.historyPageLoading && a.data.historyErr == nil && a.data.historyMore && len(a.data.historyTips) > 0
}

func (a *app) cancelHistoryPage() {
	if a.historyPageCancel != nil {
		a.historyPageCancel()
		a.historyPageCancel = nil
	}
	a.historyPageLoading = false
	a.guard.Begin("history-page")
}

func (a *app) beginHistoryPage() (historyPageRequest, context.Context, bool) {
	if !a.canLoadHistory() {
		return historyPageRequest{}, nil, false
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.historyPageCancel = cancel
	a.historyPageLoading = true
	return historyPageRequest{
		ticket: a.guard.Begin("history-page"), revision: a.historyRevision,
		offset: len(a.data.history), tips: append([]string(nil), a.data.historyTips...),
	}, ctx, true
}

// finishHistoryPage only mutates data, never a live ListState. Rebuilding the
// immutable presentation reconciles native selection by its existing SHA key.
func (a *app) finishHistoryPage(req historyPageRequest, page gitrepo.HistoryPage, err error) bool {
	if !a.historyPageLoading || !a.guard.Accept(req.ticket) || a.busy() || req.revision != a.historyRevision || req.offset != len(a.data.history) || !slices.Equal(req.tips, a.data.historyTips) {
		return false
	}
	a.historyPageLoading = false
	if a.historyPageCancel != nil {
		a.historyPageCancel()
		a.historyPageCancel = nil
	}
	if err == nil {
		err = validateHistoryPage(a.data.history, req.tips, page)
	}
	if err != nil {
		a.record("继续读取历史失败", err)
		return true
	}
	commits := make([]gitrepo.Commit, 0, len(a.data.history)+len(page.Commits))
	commits = append(commits, a.data.history...)
	a.data.history = append(commits, page.Commits...)
	a.data.historyMore = page.HasMore
	a.historyLimit = max(a.historyLimit, len(a.data.history))
	a.historyRevision++
	return true
}

func validateHistoryPage(loaded []gitrepo.Commit, tips []string, page gitrepo.HistoryPage) error {
	if !slices.Equal(tips, page.Tips) || len(page.Commits) > historyPageSize || page.HasMore && len(page.Commits) == 0 {
		return fmt.Errorf("历史分页身份或长度不一致，请刷新后重试")
	}
	seen := make(map[string]bool, len(loaded)+len(page.Commits))
	for _, c := range loaded {
		seen[c.SHA] = true
	}
	for _, c := range page.Commits {
		if c.SHA == "" || seen[c.SHA] {
			return fmt.Errorf("历史分页包含重复或无效的提交，请刷新后重试")
		}
		seen[c.SHA] = true
	}
	return nil
}

func (a *app) loadMoreHistory() {
	req, ctx, ok := a.beginHistoryPage()
	if !ok {
		return
	}
	r, cancel := a.repo, a.historyPageCancel
	go func() {
		defer cancel()
		page, err := r.HistoryPage(ctx, req.tips, req.offset, historyPageSize)
		mygo.RunOnMain(func() {
			if a.finishHistoryPage(req, page, err) {
				a.invalidate()
			}
		})
	}()
}
