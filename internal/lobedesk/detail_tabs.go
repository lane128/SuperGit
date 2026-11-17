package lobedesk

// Paths are repository-relative, unmodified Git paths. Index and worktree
// patches for the same path are separate reading targets.
type detailFileKey struct {
	Path   string
	Staged bool
}

type detailContextKey struct {
	Root, CommitSHA string
	Worktree        bool
}

type detailTabKey struct {
	Summary bool
	File    detailFileKey
}

type detailTabsState struct {
	OpenFiles   []detailFileKey
	Active      detailTabKey
	Scroll      map[detailTabKey]detailScrollPosition
	Collapsed   map[detailFileKey]bool
	FilesScroll detailScrollPosition
}

type detailScrollPosition struct{ X, Y float32 }

func (s *detailTabsState) Open(file detailFileKey) {
	if file.Path == "" {
		return
	}
	for _, open := range s.OpenFiles {
		if open == file {
			s.Active = detailTabKey{File: file}
			return
		}
	}
	s.OpenFiles = append(s.OpenFiles, file)
	s.Active = detailTabKey{File: file}
}

func (s *detailTabsState) Activate(tab detailTabKey) bool {
	if tab.Summary {
		s.Active = detailTabKey{Summary: true}
		return true
	}
	for _, file := range s.OpenFiles {
		if tab.File == file {
			s.Active = detailTabKey{File: file}
			return true
		}
	}
	return false
}

func (s *detailTabsState) Close(file detailFileKey) {
	for i, open := range s.OpenFiles {
		if open != file {
			continue
		}
		s.OpenFiles = append(s.OpenFiles[:i], s.OpenFiles[i+1:]...)
		if !s.Active.Summary && s.Active.File == file {
			s.Active = detailTabKey{Summary: true}
			if len(s.OpenFiles) > 0 {
				s.Active = detailTabKey{File: s.OpenFiles[min(i, len(s.OpenFiles)-1)]}
			}
		}
		return
	}
}

// Keep the working tree and the most recently viewed historical commit. A new
// repository drops both; a different SHA only drops the historical context.
type detailContextsState struct {
	Root              string
	HistorySHA        string
	Worktree, History detailTabsState
}

func (a *app) currentDetailContext() detailContextKey {
	key := detailContextKey{Worktree: a.worktreeSelected}
	if a.repo != nil {
		key.Root = a.repo.Root
	}
	if !key.Worktree {
		key.CommitSHA = a.selectedSHA
	}
	return key
}

func (a *app) detailTabs() *detailTabsState {
	key := a.currentDetailContext()
	s := &a.detailContexts
	if s.Root != key.Root {
		*s = detailContextsState{Root: key.Root}
	}
	tabs := &s.Worktree
	if !key.Worktree {
		if s.HistorySHA != key.CommitSHA {
			s.HistorySHA = key.CommitSHA
			s.History = detailTabsState{}
		}
		tabs = &s.History
	}
	if tabs.Active.File.Path == "" {
		tabs.Active = detailTabKey{Summary: true}
	}
	return tabs
}
