package desktop

import (
	"strings"
	"supergit/internal/gitrepo"
)

type tagInspection struct {
	name, objectSHA string
	commit          gitrepo.Commit
}

func filterTags(tags []gitrepo.Tag, query string) []gitrepo.Tag {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return tags
	}
	var matches []gitrepo.Tag
	for _, tag := range tags {
		if strings.Contains(strings.ToLower(tag.Name+" "+tag.ObjectSHA+" "+tag.TargetSHA), query) {
			matches = append(matches, tag)
		}
	}
	return matches
}

func (a *app) tagSearchChanged() { a.tagListReset = true }

// setSelectedCommit changes metadata identity only. Callers start the existing
// independent message and file reads after accepting the selection.
func (a *app) setSelectedCommit(commit gitrepo.Commit) {
	a.tagInspection = nil
	a.selectedSHA = commit.SHA
	a.historyIndex = -1
	for i := range a.data.history {
		if a.data.history[i].SHA == commit.SHA {
			a.historyIndex = i
			break
		}
	}
}

func (a *app) selectedCommit() *gitrepo.Commit {
	for i := range a.data.history {
		if a.data.history[i].SHA == a.selectedSHA {
			return &a.data.history[i]
		}
	}
	if pin := a.tagInspection; pin != nil && pin.commit.SHA == a.selectedSHA {
		return &pin.commit
	}
	return nil
}

func (a *app) selectTagCommit(name, objectSHA string) (gitrepo.Commit, bool) {
	if a.closed || a.repo == nil || a.page != "branches" || a.guard.writing || a.opening || a.data.tagsErr != nil {
		return gitrepo.Commit{}, false
	}
	for _, tag := range a.data.tags {
		if tag.Name != name || tag.ObjectSHA != objectSHA || tag.Commit.SHA == "" {
			continue
		}
		commit := tag.Commit
		commit.Parents = append([]string(nil), commit.Parents...)
		a.setSelectedCommit(commit)
		a.tagInspection = &tagInspection{name: tag.Name, objectSHA: tag.ObjectSHA, commit: commit}
		a.historyQuery = ""
		a.historySearchChanged()
		a.focusHistoryList = true
		a.page = "history"
		return commit, true
	}
	return gitrepo.Commit{}, false
}

func (a *app) showTagCommit(name, objectSHA string) {
	if commit, ok := a.selectTagCommit(name, objectSHA); ok {
		a.loadCommitMessage(commit.SHA)
		a.loadCommitFiles()
	}
}
