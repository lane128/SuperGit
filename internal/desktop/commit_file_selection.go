package desktop

import "supergit/internal/gitrepo"

type commitFilePresentation struct {
	repository, revision uint64
	sha                  string
	files                []gitrepo.ChangedFile
}
type commitFileCursor struct {
	model *commitFilePresentation
	path  string
	row   int
}

func (s *commitFileCursor) sync(model *commitFilePresentation, path string) {
	if s.model == model && s.path == path {
		return
	}
	s.model, s.path, s.row = model, path, -1
	if model != nil && path != "" {
		for i, file := range model.files {
			if file.Path == path {
				s.row = i
				break
			}
		}
	}
}
func (a *app) replaceCommitFiles(files []gitrepo.ChangedFile) {
	a.commitFiles = files
	a.commitFileRevision++
}
func (a *app) commitFileModel() *commitFilePresentation {
	if m := a.commitFileView; m != nil && m.repository == a.guard.repository && m.revision == a.commitFileRevision && m.sha == a.selectedSHA {
		return m
	}
	m := &commitFilePresentation{repository: a.guard.repository, revision: a.commitFileRevision, sha: a.selectedSHA, files: append([]gitrepo.ChangedFile(nil), a.commitFiles...)}
	a.commitFileView = m
	return m
}
func (a *app) commitFileSelection(model *commitFilePresentation, row int) (gitrepo.ChangedFile, bool) {
	if a.repo == nil || a.page != "history" || a.guard.writing || a.opening || model == nil || model.sha == "" || row < 0 || row >= len(model.files) || row >= len(a.commitFiles) || model != a.commitFileModel() {
		return gitrepo.ChangedFile{}, false
	}
	file := model.files[row]
	if file != a.commitFiles[row] {
		return gitrepo.ChangedFile{}, false
	}
	return file, true
}
