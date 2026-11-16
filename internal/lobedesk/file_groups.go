package lobedesk

import "supergit/internal/gitrepo"

// Stable keys preserve saved folds; this order is shared by the Files column,
// keyboard traversal, and the overview read queue. Empty groups remain present.
type workingFileGroup struct {
	Key, Title string
	Entries    []gitrepo.Entry
	Staged     bool
}

func (a *app) workingFileGroups() []workingFileGroup {
	return []workingFileGroup{
		{Key: filesSectionUnstaged, Title: a.t("fileGroupModified"), Entries: a.unstagedEntries()},
		{Key: filesSectionUntracked, Title: a.t("fileGroupUntracked"), Entries: a.untrackedEntries()},
		{Key: filesSectionStaged, Title: a.t("fileGroupStaged"), Entries: a.stagedEntries(), Staged: true},
	}
}
