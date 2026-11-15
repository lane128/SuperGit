package desktop

import "context"

// The list may deliver its new cursor before the view rebuilds. Only reconcile
// when the presentation or externally selected SHA actually changes.
type historyCursor struct {
	model *historyPresentation
	sha   string
	row   int
}

func (s *historyCursor) sync(model *historyPresentation, sha string) {
	if s.model == model && s.sha == sha {
		return
	}
	s.model, s.sha, s.row = model, sha, -1
	if model != nil {
		for i, c := range model.Commits {
			if c.SHA == sha {
				s.row = i
				break
			}
		}
	}
}

func (a *app) historySelectionIndex(model *historyPresentation, row int) (int, bool) {
	if a.repo == nil || a.page != "history" || a.guard.writing || a.opening || model == nil || row < 0 || row >= len(model.Commits) || row >= len(model.Indices) {
		return -1, false
	}
	if model != a.historyModel() {
		return -1, false
	}
	index := model.Indices[row]
	if index < 0 || index >= len(a.data.history) || a.data.history[index].SHA != model.Commits[row].SHA {
		return -1, false
	}
	return index, true
}

// Arrow-key browsing supersedes previous commit/file reads. Cancel their Git
// subprocess context as well as rejecting late results through the read lane.
func (a *app) newCommitRead() (context.Context, context.CancelFunc) {
	if a.commitCancel != nil {
		a.commitCancel()
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.commitCancel = cancel
	return ctx, cancel
}
