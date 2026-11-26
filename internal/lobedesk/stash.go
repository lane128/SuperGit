package lobedesk

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

// The stash list is a column of the History list now, so the page's split
// constants went with it. The saved StashSplit preference stays readable for
// settings files written by older builds.

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
	if a.selectedStash == "" {
		return
	}
	a.selectedStash = ""
	a.commitSeq++
	a.commitFiles, a.commitFilesLoaded = nil, false
	a.commitFilesLoading, a.commitDiffLoading = false, false
	a.commitFilesErr, a.commitDiffErr = "", ""
	a.commitPath = ""
	a.stashTrackedCount = 0
	a.resetCommitDiff()
	a.clearCommitSelection()
}

func (a *app) stashSaveMenu(c *ui.Context) {
	tk := theme.TokensOf(c)
	open := &a.stashSaveOpen
	blocked := a.workspaceStashBlocked()
	if blocked {
		*open = false
	}
	chosen := overlay.DropdownMenu(c, open, overlay.DropdownMenuProps{Items: []overlay.MenuItem{
		{Key: "tracked", Label: a.t("saveStash"), Icon: lucide.Archive(), Disabled: blocked},
		{Key: "untracked", Label: a.t("saveStashUntracked"), Icon: lucide.Archive(), Disabled: blocked},
	}}, func(trigger ui.Element) {
		styleWorkbenchButton(c, trigger, buttonNormal, blocked, *open).
			Width(24).Height(24).PaddingX(0).Radius(0, 6, 6, 0).BorderWidth(1, 1, 1, 0)
		label := a.t("saveStashAction")
		trigger.Label(label).Tooltip(a.t("workspaceStashOptions")).Shrink(0)
		trigger.Children(func() {
			ui.Icon(c, lucide.ChevronDown()).FontSize(scaledFont(12)).TextColor(tk.TextSecondary).Shrink(0)
		})
	})
	if blocked {
		return
	}
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

func stashCommit(s gitrepo.Stash) gitrepo.Commit {
	return gitrepo.Commit{SHA: s.SHA, Subject: stashTitle(s), Author: s.Author, AuthorEmail: s.AuthorEmail, Date: s.Date, Day: s.Day, Parents: strings.Fields(s.ParentSHAs)}
}

// The same row can be rendered independently of a full History graph.
func (a *app) stashRow(c *ui.Context, stash gitrepo.Stash, pos, size int) {
	a.historyEntryRow(c, stashCommit(stash), &stash, 0, pos, size, nil)
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
	if sha == "" {
		return
	}
	a.pendingJumpSHA = ""
	a.pendingJumpSHA = ""
	if a.selectedStash == sha && (a.commitFilesLoaded || a.commitFilesLoading) {
		return
	}
	a.saveDetailScroll()
	a.messageCommitRequest = nil
	a.messageMenuGeneration++
	a.clearHistoryNavigation()
	a.worktreeSelected = false
	a.setSelectedCommit("")
	a.selectedStash = sha
	a.commitPath, a.commitMessage, a.commitMessageErr = "", "", ""
	a.commitSig = gitrepo.Signature{}
	a.commitFiles, a.commitFilesLoaded = nil, false
	a.commitCollapsed = nil
	a.resetCommitDiff()
	a.clearCommitSelection()
	a.commitInfoScroll, a.commitDiffScroll, a.commitFilesScroll = ui.ScrollState{}, ui.ScrollState{}, ui.ScrollState{}
	a.detailTabs()
	a.loadStashDiff()
	a.invalidate()
}

func (a *app) loadStashDiff() {
	if a.repo == nil || a.selectedStash == "" {
		return
	}
	a.commitSeq++
	seq, repo, sha := a.commitSeq, a.repo, a.selectedStash
	ignoreWhitespace := a.ignoreDiffWhitespace
	a.commitFilesLoading, a.commitDiffLoading = true, true
	a.commitFilesErr, a.commitDiffErr = "", ""
	go func() {
		snapshot, err := repo.StashChanges(a.ctx, sha, ignoreWhitespace)
		var added, deleted int
		for _, patch := range snapshot.TrackedDiff {
			for _, line := range patch.Diff.Lines {
				switch line.Kind {
				case "add":
					added++
				case "del":
					deleted++
				}
			}
		}
		diff, sections := buildCommitDiff(append(snapshot.TrackedDiff, snapshot.UntrackedDiff...))
		a.hopToMain(func() {
			if seq != a.commitSeq || a.repo != repo || a.selectedStash != sha {
				return
			}
			a.commitFilesLoading, a.commitDiffLoading = false, false
			if err != nil {
				a.commitFilesErr, a.commitDiffErr = err.Error(), err.Error()
				a.commitFilesLoaded = false
			} else {
				a.stashTrackedCount = len(snapshot.Tracked)
				a.commitFiles = append(snapshot.Tracked, snapshot.Untracked...)
				a.commitFilesLoaded = true
				a.commitDiff, a.commitSections = diff, sections
				a.stashAdded, a.stashDeleted = added, deleted
			}
			a.invalidate()
		})
	}()
}

func (a *app) stashMenuItems() []overlay.MenuItem {
	return []overlay.MenuItem{
		{Key: "apply", Label: a.t("applyStash"), Icon: lucide.ArchiveRestore(), Disabled: a.busy || a.loading},
		{Key: "pop", Label: a.t("popStash"), Icon: lucide.ArchiveX(), Disabled: a.busy || a.loading},
		{Divider: true},
		{Key: "copy", Label: a.t("copyStashMessage"), Icon: lucide.Copy()},
		{Divider: true},
		{Key: "drop", Label: a.t("dropStashAction"), Icon: lucide.Trash2(), Danger: true, Disabled: a.busy || a.loading},
	}
}

func (a *app) stashMenu(c *ui.Context, row ui.Element, stash gitrepo.Stash) {
	a.runStashAction(overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: a.stashMenuItems()}), stash)
}

func (a *app) stashActions(c *ui.Context, stash gitrepo.Stash) {
	c = c.Key(stash.SHA)
	anchor := ui.Row(c).Shrink(0)
	open := ui.Local(anchor, "stash-actions", func() bool { return false })
	anchor.Children(func() {
		chosen := overlay.DropdownMenu(c, open, overlay.DropdownMenuProps{Items: a.stashMenuItems()}, func(trigger ui.Element) {
			styleWorkbenchButton(c, trigger, buttonNormal, false, *open)
			trigger.Label(a.t("stashActions")).Tooltip(a.t("stashActions"))
			trigger.Children(func() { ui.Text(c, "•••") })
		})
		a.runStashAction(chosen, stash)
	})
}

func (a *app) runStashAction(chosen string, stash gitrepo.Stash) {
	switch chosen {
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
