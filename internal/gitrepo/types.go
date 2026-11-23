package gitrepo

// Entry preserves repository-relative filenames verbatim, including newlines.
type Entry struct {
	Path         string
	OriginalPath string
	Index        byte
	Worktree     byte
	Submodule    string
	Conflict     bool
}

func (e Entry) Staged() bool {
	return e.Index != '.' && e.Index != ' ' && e.Index != '?' && e.Index != 0
}
func (e Entry) Unstaged() bool {
	return e.Worktree != '.' && e.Worktree != ' ' && e.Worktree != 0 || e.Conflict
}
func (e Entry) IsSubmodule() bool { return len(e.Submodule) > 0 && e.Submodule[0] == 'S' }

// SubmoduleHasLocalEdits reports the porcelain v2 tracked and untracked
// dirtiness flags, independently of whether the child's HEAD moved.
func (e Entry) SubmoduleHasLocalEdits() bool {
	return e.IsSubmodule() && len(e.Submodule) == 4 && (e.Submodule[2] == 'M' || e.Submodule[3] == 'U')
}

// WorkspaceUnstaged reports changes this repository can record. A child
// checkout's edits are reported in that checkout, while its gitlink can
// still change here independently in the index or worktree.
func (e Entry) WorkspaceUnstaged() bool {
	return e.Unstaged() && (e.Conflict || e.Worktree != 'M' || !e.SubmoduleHasLocalEdits() || e.Submodule[1] == 'C')
}

func (e Entry) WorkspaceChanged() bool { return e.Staged() || e.WorkspaceUnstaged() }

type Status struct {
	Branch, Head, Upstream string
	UpstreamGone           bool
	Ahead, Behind          int
	Entries                []Entry
}
type DiffLine struct {
	Kind, Text string
	Old, New   int
	Hunk       int
}
type Hunk struct{ Header, Patch string }
type Diff struct {
	Binary        bool
	Text          string
	Lines         []DiffLine
	Hunks         []Hunk
	CanStageHunks bool
}
type Commit struct {
	SHA                                      string
	Parents                                  []string
	Subject, Author, AuthorEmail, Date, Refs string
	// Day is Date's calendar day in local time ("2006-01-02"), parsed once when
	// the log is read. The history search matches it, so a reader can filter to
	// a day without the search re-parsing every timestamp on every frame.
	Day string
}
type ChangedFile struct{ Path, OriginalPath, Status string }
type Branch struct {
	Name, SHA, Upstream string
	Ahead, Behind       int
	UpstreamGone        bool
	Current             bool
	// Commit is the branch's tip commit as the ref listing reads it. The
	// branch page's details pane shows its subject, author and time without a
	// second git process per selection.
	Commit Commit
}
type Worktree struct {
	Path, HEAD, Branch string
	Bare, Locked       bool
}
type Submodule struct {
	Name, Path, Recorded, Index, Current, Branch string
	Initialized, Dirty                           bool
	ChangedFiles                                 int
	Error                                        string
}
