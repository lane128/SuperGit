package desktop

import (
	"context"
	"supergit/internal/gitrepo"
)

type workspaceFileKey struct {
	path   string
	staged bool
}
type workspaceHeaderKey struct{ staged bool }
type workspaceRow struct {
	key    workspaceFileKey
	entry  gitrepo.Entry
	index  int
	header bool
	count  int
}
type workspacePresentation struct {
	repository, revision uint64
	rows                 []workspaceRow
}

func (a *app) workspaceModel() *workspacePresentation {
	if m := a.workspaceView; m != nil && m.repository == a.guard.repository && m.revision == a.workspaceRevision {
		return m
	}
	m := &workspacePresentation{repository: a.guard.repository, revision: a.workspaceRevision}
	for _, staged := range []bool{false, true} {
		header := len(m.rows)
		m.rows = append(m.rows, workspaceRow{header: true, key: workspaceFileKey{staged: staged}})
		for i, e := range a.data.status.Entries {
			if staged && e.Staged() || !staged && e.Unstaged() {
				m.rows = append(m.rows, workspaceRow{key: workspaceFileKey{e.Path, staged}, entry: e, index: i})
				m.rows[header].count++
			}
		}
	}
	a.workspaceView = m
	return m
}

// Reconcile only when the model or accepted target changes, so a view build
// cannot replace a cursor delivered by the list with the previous target.
type workspaceCursor struct {
	model *workspacePresentation
	key   workspaceFileKey
	row   int
}

func (s *workspaceCursor) sync(model *workspacePresentation, path string, staged bool) {
	key := workspaceFileKey{path, staged}
	if s.model == model && s.key == key {
		return
	}
	s.model, s.key, s.row = model, key, -1
	if model != nil && path != "" {
		for i, item := range model.rows {
			if !item.header && item.key == key {
				s.row = i
				break
			}
		}
	}
}
func (a *app) workspaceSelection(model *workspacePresentation, row int) (workspaceRow, bool) {
	if a.repo == nil || a.page != "workspace" || a.guard.writing || a.opening || model == nil || row < 0 || row >= len(model.rows) || model != a.workspaceModel() {
		return workspaceRow{}, false
	}
	item := model.rows[row]
	if item.header || item.index < 0 || item.index >= len(a.data.status.Entries) {
		return workspaceRow{}, false
	}
	current := a.data.status.Entries[item.index]
	if current != item.entry || !(item.key.staged && current.Staged() || !item.key.staged && current.Unstaged()) {
		return workspaceRow{}, false
	}
	return item, true
}
func (a *app) chooseWorkspaceFile(model *workspacePresentation, row int, activate bool) {
	item, ok := a.workspaceSelection(model, row)
	if !ok {
		return
	}
	a.loadDiff(item.key.path, item.key.staged)
	if item.entry.Conflict && activate {
		a.page = "conflicts"
		a.loadConflict(item.key.path)
	}
}
func (a *app) newWorkspaceRead() (context.Context, context.CancelFunc) {
	if a.diffCancel != nil {
		a.diffCancel()
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.diffCancel = cancel
	return ctx, cancel
}
