package lobedesk

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

type promptState struct {
	open        bool
	title       string
	placeholder string
	value       string
	actionName  string
	action      func(string)
	// selectAll asks the field to select its prefilled value once, in the
	// first frame after the dialog opens.
	selectAll bool
}

func (a *app) askPrompt(title, placeholder, initial, actionName string, action func(string)) {
	a.prompt = promptState{
		open:        true,
		title:       title,
		placeholder: placeholder,
		value:       initial,
		actionName:  actionName,
		action:      action,
		selectAll:   initial != "",
	}
	a.invalidate()
}

func (a *app) promptDialog(c *ui.Context) {
	if !a.prompt.open {
		return
	}
	tk := theme.TokensOf(c)
	confirmed := false
	ui.Modal(c, &a.prompt.open, func() {
		ui.Column(c).Width(400).Padding(18).Gap(10).
			Background(tk.BgElevated).Radius(tk.BorderRadiusLG).Border(1, tk.Border).
			Children(func() {
				display.Text(c, a.prompt.title, display.TextProps{Strong: true, FontSize: scaledFont(14)})
				// AutoFocus, not Focus: Focus keeps the field on every
				// frame, which takes Tab back before the reader can reach
				// Cancel or the action.
				input := field.Input(c, &a.prompt.value, field.InputProps{
					Variant:     field.InputFilled,
					Placeholder: a.prompt.placeholder,
					Size:        theme.Middle,
				}).AutoFocus()
				// Select a prefilled value once, as renaming in Finder and
				// every browser does: typing replaces the old name instead of
				// appending to it. Doing it every frame would make each
				// keystroke replace the one before.
				if a.prompt.selectAll {
					input.SetTextSelection(0, len([]rune(a.prompt.value)))
					a.prompt.selectAll = false
				}
				if input.Submitted() && strings.TrimSpace(a.prompt.value) != "" {
					confirmed = true
					a.prompt.open = false
				}
				ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Justify(ui.End).Children(func() {
					if action.Button(c, a.t("cancel"), action.ButtonProps{Variant: action.Default, Size: theme.Small}).Clicked() {
						a.prompt.open = false
					}
					ready := strings.TrimSpace(a.prompt.value) != ""
					button := action.Button(c, a.prompt.actionName, action.ButtonProps{Variant: action.Primary, Size: theme.Small}).
						Disabled(!ready)
					if ready && button.Clicked() {
						confirmed = true
						a.prompt.open = false
					}
				})
			})
	})
	if confirmed {
		action := a.prompt.action
		value := strings.TrimSpace(a.prompt.value)
		a.prompt = promptState{}
		if action != nil {
			action(value)
		}
		return
	}
	if !a.prompt.open {
		a.prompt = promptState{}
	}
}

type confirmState struct {
	open       bool
	title      string
	message    string
	actionName string
	danger     bool
	action     func()
}

func (a *app) askConfirm(title, message, actionName string, danger bool, action func()) {
	a.confirm = confirmState{
		open:       true,
		title:      title,
		message:    message,
		actionName: actionName,
		danger:     danger,
		action:     action,
	}
	a.invalidate()
}

func (a *app) confirmDialog(c *ui.Context) {
	if !a.confirm.open {
		return
	}
	tk := theme.TokensOf(c)
	confirmed := false
	ui.Modal(c, &a.confirm.open, func() {
		ui.Column(c).Width(430).Padding(18).Gap(10).
			Background(tk.BgElevated).Radius(tk.BorderRadiusLG).Border(1, tk.Border).
			Children(func() {
				display.Text(c, a.confirm.title, display.TextProps{Strong: true, FontSize: scaledFont(14)})
				display.Text(c, a.confirm.message, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12), Rows: 5})
				ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Justify(ui.End).Margin(8, 0, 0, 0).
					Children(func() {
						if action.Button(c, a.t("cancel"), action.ButtonProps{Variant: action.Default, Size: theme.Small}).Clicked() {
							a.confirm.open = false
						}
						label := a.confirm.actionName
						button := action.Button(c, label, action.ButtonProps{Variant: action.Default, Size: theme.Small})
						if a.confirm.danger {
							button.TextColor(tk.Error)
						}
						if button.Clicked() {
							confirmed = true
							a.confirm.open = false
						}
					})
			})
	})
	if confirmed {
		action := a.confirm.action
		a.confirm = confirmState{}
		if action != nil {
			action()
		}
		return
	}
	if !a.confirm.open {
		a.confirm = confirmState{}
	}
}

// writeClipboard is MyGo's clipboard write. Tests replace it to read what a
// menu copied; the platform clipboard needs a running application.
var writeClipboard = func(value string) error {
	return mygo.Clipboard.Write(transfer.TextData(value))
}

func (a *app) copyText(value string) {
	if err := writeClipboard(value); err != nil {
		a.queue("error", a.tf("copyFailed", err.Error()), "")
		return
	}
	a.queue("success", a.t("copied"), "")
	a.invalidate()
}

func (a *app) absolutePath(path string) string {
	if a.repo == nil {
		return path
	}
	return filepath.Join(a.repo.Root, filepath.FromSlash(path))
}

// openFile hands a repository path to macOS: reveal selects it in Finder,
// default opens it with the app the system associates with it.
func (a *app) openFile(path, kind string) {
	a.openPath(a.absolutePath(path), kind)
}

// openPath reveals or opens a path that is already absolute, such as a
// worktree root that lives outside the repository.
func (a *app) openPath(full, kind string) {
	go func() {
		args := []string{full}
		switch kind {
		case "reveal":
			args = []string{"-R", full}
		case "terminal":
			// Terminal is macOS's own and always present; other terminals are
			// the user's choice through their default app handler.
			args = []string{"-a", "Terminal", full}
		}
		if err := exec.Command("open", args...).Run(); err != nil {
			mygo.RunOnMain(func() {
				a.queue("error", a.tf("openFailedHint", err.Error()), "")
				a.invalidate()
			})
		}
	}()
}

func (a *app) fileMenu(c *ui.Context, row ui.Element, entry gitrepo.Entry, staged bool) {
	if entry.Conflict {
		switch overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: a.conflictFileMenuItems()}) {
		case "resolve":
			a.resolveConflict(entry.Path)
		case "ours":
			a.takeConflictSide(entry.Path, "ours")
		case "theirs":
			a.takeConflictSide(entry.Path, "theirs")
		case "copy-path":
			a.copyText(entry.Path)
		case "reveal":
			a.openFile(entry.Path, "reveal")
		case "absolute":
			a.copyText(a.absolutePath(entry.Path))
		case "open-default":
			a.openFile(entry.Path, "open")
		case "history":
			a.showFileHistory(entry.Path)
		case "blame":
			a.openBlame(entry.Path)
		}
		return
	}
	switch overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: a.fileMenuItems(entry, staged)}) {
	case "stage":
		a.toggleStage(entry, false)
	case "unstage":
		a.toggleStage(entry, true)
	case "discard":
		a.confirmDiscard(entry)
	case "stash":
		a.promptStashFile(entry)
	case "copy-path":
		a.copyText(entry.Path)
	case "absolute":
		a.copyText(a.absolutePath(entry.Path))
	case "open-default":
		a.openFile(entry.Path, "open")
	case "reveal":
		a.openFile(entry.Path, "reveal")
	case "history":
		a.showFileHistory(entry.Path)
	case "blame":
		a.openBlame(entry.Path)
	}
}

// fileMenuItems builds the file row's menu. The first group stages or
// unstages; an unstaged file also offers the destructive discard beside it.
// Stash opens a dialog, so its label carries the ellipsis, and the copies sit
// in their own group above the open/reveal pair and the history pair.
func (a *app) fileMenuItems(entry gitrepo.Entry, staged bool) []overlay.MenuItem {
	items := []overlay.MenuItem{}
	if staged {
		items = append(items, overlay.MenuItem{Key: "unstage", Label: a.t("unstageFile"), Icon: lucide.Minus()})
	} else {
		items = append(items, overlay.MenuItem{Key: "stage", Label: a.t("stageFile"), Icon: lucide.Plus()})
		untracked := entry.Index == '?' && entry.Worktree == '?'
		label := a.t("discardChanges")
		icon := lucide.Undo2()
		if untracked {
			label = a.t("deleteFile")
			icon = lucide.Trash2()
		}
		items = append(items,
			overlay.MenuItem{Divider: true},
			overlay.MenuItem{Key: "discard", Label: label, Icon: icon, Danger: true},
		)
	}
	return append(items,
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{Key: "stash", Label: a.t("stashFileAction"), Icon: lucide.Archive()},
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{Key: "copy-path", Label: a.t("copyPath"), Icon: lucide.Copy()},
		overlay.MenuItem{Key: "absolute", Label: a.t("copyAbsolutePath"), Icon: lucide.Copy()},
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{Key: "open-default", Label: a.t("openDefaultApp"), Icon: lucide.ExternalLink()},
		overlay.MenuItem{Key: "reveal", Label: a.t("revealInFinder"), Icon: lucide.FolderOpen()},
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{Key: "history", Label: a.t("fileHistory"), Icon: lucide.History()},
		overlay.MenuItem{Key: "blame", Label: a.workspaceBlameLabel(entry), Icon: lucide.FileSearch(), Disabled: !canBlame(entry)},
	)
}

// conflictFileMenuItems is the menu while a file still has conflicts: the
// three ways to settle it come first, because the file cannot be staged until
// one of them is used.
func (a *app) conflictFileMenuItems() []overlay.MenuItem {
	return []overlay.MenuItem{
		{Key: "resolve", Label: a.t("markResolved"), Icon: lucide.CircleCheck()},
		{Key: "ours", Label: a.t("useOurs"), Icon: lucide.GitBranch()},
		{Key: "theirs", Label: a.t("useTheirs"), Icon: lucide.GitMerge()},
		{Divider: true},
		{Key: "copy-path", Label: a.t("copyPath"), Icon: lucide.Copy()},
		{Key: "absolute", Label: a.t("copyAbsolutePath"), Icon: lucide.Copy()},
		{Divider: true},
		{Key: "open-default", Label: a.t("openDefaultApp"), Icon: lucide.ExternalLink()},
		{Key: "reveal", Label: a.t("revealInFinder"), Icon: lucide.FolderOpen()},
		{Divider: true},
		{Key: "history", Label: a.t("fileHistory"), Icon: lucide.History()},
		{Key: "blame", Label: a.t("blame"), Icon: lucide.FileSearch()},
	}
}

// diffHeaderMenu gives the workspace diff pane's file header the same file
// actions as the list row, with the conflict actions first while the file is
// unresolved, so resolving does not require going back to the list.
func (a *app) diffHeaderMenu(c *ui.Context, header ui.Element) {
	path := a.selectedPath
	if path == "" {
		return
	}
	items := []overlay.MenuItem{}
	if entry, ok := a.selectedEntry(); ok && entry.Conflict {
		items = append(items,
			overlay.MenuItem{Key: "resolve", Label: a.t("markResolved"), Icon: lucide.CircleCheck()},
			overlay.MenuItem{Key: "ours", Label: a.t("useOurs"), Icon: lucide.GitBranch()},
			overlay.MenuItem{Key: "theirs", Label: a.t("useTheirs"), Icon: lucide.GitMerge()},
			overlay.MenuItem{Divider: true},
		)
	}
	if entry, ok := a.selectedEntry(); ok && !entry.Conflict {
		items = append(items,
			overlay.MenuItem{Key: "stash", Label: a.t("stashFileAction"), Icon: lucide.Archive()},
			overlay.MenuItem{Divider: true},
		)
	}
	items = append(items,
		overlay.MenuItem{Key: "copy-path", Label: a.t("copyPath"), Icon: lucide.Copy()},
		overlay.MenuItem{Key: "absolute", Label: a.t("copyAbsolutePath"), Icon: lucide.Copy()},
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{Key: "open-default", Label: a.t("openDefaultApp"), Icon: lucide.ExternalLink()},
		overlay.MenuItem{Key: "reveal", Label: a.t("revealInFinder"), Icon: lucide.FolderOpen()},
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{Key: "history", Label: a.t("fileHistory"), Icon: lucide.History()},
		overlay.MenuItem{Key: "blame", Label: a.t("blame"), Icon: lucide.FileSearch()},
	)
	switch overlay.ContextMenu(c, header, overlay.DropdownMenuProps{Items: items}) {
	case "resolve":
		a.resolveConflict(path)
	case "stash":
		if entry, ok := a.selectedEntry(); ok {
			a.promptStashFile(entry)
		}
	case "ours":
		a.takeConflictSide(path, "ours")
	case "theirs":
		a.takeConflictSide(path, "theirs")
	case "copy-path":
		a.copyText(path)
	case "absolute":
		a.copyText(a.absolutePath(path))
	case "open-default":
		a.openFile(path, "open")
	case "reveal":
		a.openFile(path, "reveal")
	case "history":
		a.showFileHistory(path)
	case "blame":
		a.openBlameAt(path, a.selectedSHA)
	}
}

func (a *app) confirmDiscard(entry gitrepo.Entry) {
	untracked := entry.Index == '?' && entry.Worktree == '?'
	if untracked {
		a.askConfirm(a.t("deleteTitle"), a.tf("deleteMessage", entry.Path), a.t("delete"),
			true, func() {
				a.performAs(busyDeleteFile, a.t("deleteFailed"), a.t("deleteSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
					return repo.Discard(ctx, entry.Path, true)
				})
			})
		return
	}
	a.askConfirm(a.t("discardTitle"), a.tf("discardMessage", entry.Path), a.t("discard"),
		true, func() {
			a.performAs(busyDiscard, a.t("discardFailed"), a.t("discardSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
				return repo.Discard(ctx, entry.Path, false)
			})
		})
}

// historyFilterFor is the search a filter entry writes into the box. The
// author and day come from the commit the menu was opened on; the week and
// the month are relative to the clock, not to the commit, because they answer
// "what happened lately" rather than "what did this commit belong to". ok is
// false when the entry has no value to filter on.
func historyFilterFor(key string, commit gitrepo.Commit, now time.Time) (query string, ok bool) {
	switch key {
	case "filter-author":
		return commit.Author, commit.Author != ""
	case "filter-day":
		return commit.Day, commit.Day != ""
	case "filter-recent":
		return recentWeekQuery(now), true
	case "filter-month":
		return thisMonthQuery(now), true
	}
	return "", false
}

func (a *app) commitMenu(c *ui.Context, row ui.Element, commit gitrepo.Commit) {
	action := overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: a.commitMenuItems(commit)})
	if name, ok := strings.CutPrefix(action, commitBranchCheckoutPrefix); ok {
		a.switchBranch(name)
		return
	}
	switch action {
	case "copy-sha":
		a.copyText(commit.SHA)
	case "copy-subject":
		a.copyText(commit.Subject)
	case "copy-message":
		a.copyText(a.commitMessage)
	case "filter-author", "filter-day", "filter-recent", "filter-month":
		// Filtering is client-side: the next frame's historyPage moves the
		// selection to the first match if this commit is hidden.
		if query, ok := historyFilterFor(action, commit, time.Now()); ok {
			a.historyQuery = query
		}
	case commitBranchPickerKey:
		a.openCheckoutBranchDialog(commit)
	case "checkout":
		a.askConfirm(a.t("checkoutCommitTitle"), a.tf("checkoutCommitMessage", shortSHA(commit.SHA)),
			a.t("checkout"), false, func() {
				a.performAs(busyCheckout, a.t("checkoutCommitFailed"), a.t("checkoutCommitSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
					return repo.CheckoutCommit(ctx, commit.SHA)
				})
			})
	case "merge":
		a.askConfirm(a.t("mergeIntoCurrent"), a.tf("mergeCommitMessage", shortSHA(commit.SHA)),
			a.t("mergeIntoCurrent"), false, func() { a.integrate("merge", commit.SHA) })
	case "rebase":
		a.askConfirm(a.t("rebaseOnto"), a.tf("rebaseCommitMessage", shortSHA(commit.SHA)),
			a.t("rebaseOnto"), false, func() { a.integrate("rebase", commit.SHA) })
	case "reset":
		a.openResetDialog(commit)
	case "tag":
		a.promptCreateTag(commit.SHA)
	case "annotated-tag":
		a.promptCreateAnnotatedTag(commit.SHA)
	case "new-branch":
		a.promptNewBranch(commit.SHA)
	case "cherry-pick":
		a.askConfirm(a.t("cherryPickTitle"), a.tf("cherryPickMessage", shortSHA(commit.SHA)),
			a.t("cherryPickAction"), false, func() {
				a.integrate("cherry-pick", commit.SHA)
			})
	case "revert":
		a.askConfirm(a.t("revertTitle"), a.tf("revertMessage", shortSHA(commit.SHA)),
			a.t("revertAction"), false, func() {
				a.integrate("revert", commit.SHA)
			})
	case "undo":
		a.askConfirm(a.t("undoCommitTitle"), a.t("undoCommitMessage"), a.t("undo"), false, func() {
			a.performAs(busyUndoCommit, a.t("undoFailed"), a.t("undoSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
				return repo.UndoCommit(ctx)
			})
		})
	}
}

// commitBranchCheckoutPrefix marks a commit-menu item that checks out a branch
// sitting on this commit. The branch name follows it, so one item per branch
// stays a plain key; the dropdown has no submenus to group them under.
const commitBranchCheckoutPrefix = "checkout-branch:"

// commitBranchPickerKey is the one-item stand-in for a commit that sits on
// several branches: it opens the picker dialog rather than growing the menu by
// one row per branch.
const commitBranchPickerKey = "checkout-branches"

// commitMenuItems builds the commit row's menu. Copies come first, then the
// actions that move the current branch, then the ones that create refs, then
// the ones that rewrite a commit. The branch checkouts open the second group:
// checking one out keeps HEAD attached, which is what a reader right-clicking
// a shared commit usually wants, so it comes before the detached-HEAD checkout.
// They take one row whether the commit sits on one branch or on twenty, so the
// menu's height does not depend on how many branches share the commit. Reset
// rewrites what the current branch points at, so it sits alone at the bottom in
// the danger style rather than between checkout and the creation actions.
func (a *app) commitMenuItems(commit gitrepo.Commit) []overlay.MenuItem {
	items := []overlay.MenuItem{
		{Key: "copy-sha", Label: a.t("copySHA"), Icon: lucide.Copy(), Extra: shortSHA(commit.SHA)},
		{Key: "copy-subject", Label: a.t("copySubject"), Icon: lucide.Copy()},
	}
	// The full message is loaded for the selected commit only; the menu shows
	// the item once it arrives, which is usually before the menu is read.
	if a.selectedSHA == commit.SHA && a.commitMessage != "" {
		items = append(items, overlay.MenuItem{Key: "copy-message", Label: a.t("copyMessage"), Icon: lucide.Copy()})
	}
	// The filters write into the search box rather than into hidden state, so
	// what is being filtered on stays visible and one click from being
	// cleared. They are read-only jumps like "view history" in the ref menus,
	// so they get their own group after the copies.
	items = append(items,
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{Key: "filter-author", Label: a.t("filterByAuthor"), Icon: lucide.User(), Disabled: a.busy || a.loading || commit.Author == ""},
		overlay.MenuItem{Key: "filter-day", Label: a.t("filterByDay"), Icon: lucide.Calendar(), Disabled: a.busy || a.loading || commit.Day == ""},
		// The week and the month are the same kind of entry for the reader:
		// a search they would otherwise have to spell out by hand.
		overlay.MenuItem{Key: "filter-recent", Label: a.t("filterRecentWeek"), Icon: lucide.CalendarRange(), Disabled: a.busy || a.loading},
		overlay.MenuItem{Key: "filter-month", Label: a.t("filterThisMonth"), Icon: lucide.CalendarDays(), Disabled: a.busy || a.loading},
		overlay.MenuItem{Divider: true},
	)
	switch branches := a.branchesAt(commit.SHA); len(branches) {
	case 0:
	case 1:
		// The checked-out branch stays listed, dimmed: it says where HEAD is,
		// where omitting it would leave the reader guessing.
		branch := branches[0]
		item := overlay.MenuItem{
			Key:      commitBranchCheckoutPrefix + branch.Name,
			Label:    a.tf("checkoutBranchAtCommit", branch.Name),
			Icon:     lucide.GitBranch(),
			Disabled: a.busy || a.loading || branch.Current,
		}
		if branch.Current {
			item.Extra = a.t("current")
		}
		items = append(items, item)
	default:
		// A commit can sit on a dozen branches, and a menu that lists every one
		// of them grows past a small window, where the rows after it -- tags,
		// cherry-pick, reset -- become unreachable: the menu neither scrolls nor
		// shrinks. So the list stays one row and the branches are picked in a
		// searchable dialog instead.
		items = append(items, overlay.MenuItem{
			Key:      commitBranchPickerKey,
			Label:    a.t("checkoutBranches"),
			Icon:     lucide.GitBranch(),
			Extra:    fmt.Sprint(len(branches)),
			Disabled: a.busy || a.loading,
		})
	}
	// Merge, rebase and reset act on the current branch, and a generic label
	// leaves the reader guessing which side is which. Naming both ends, as
	// Sublime Merge's menu does, says where the commit is coming from and what
	// it will land on.
	current := a.currentBranch()
	source := commitMenuSource(commit, current)
	mergeLabel, rebaseLabel, resetLabel := a.t("mergeIntoCurrentAction"), a.t("rebaseOntoAction"), a.t("resetAction")
	if current != "" && source != "" {
		mergeLabel = a.tf("mergeCommitInto", source, current)
		rebaseLabel = a.tf("rebaseBranchOnto", current, source)
		resetLabel = a.tf("resetBranchToCommit", current)
	}
	items = append(items,
		overlay.MenuItem{Key: "checkout", Label: a.t("checkoutCommit"), Icon: lucide.GitCommitHorizontal(), Disabled: a.busy || a.loading},
		overlay.MenuItem{Key: "merge", Label: mergeLabel, Icon: lucide.GitMerge(), Disabled: a.busy || a.loading},
		overlay.MenuItem{Key: "rebase", Label: rebaseLabel, Icon: lucide.GitCompareArrows(), Disabled: a.busy || a.loading},
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{Key: "tag", Label: a.t("createTag"), Icon: lucide.Tag(), Disabled: a.busy || a.loading},
		overlay.MenuItem{Key: "annotated-tag", Label: a.t("createAnnotatedTag"), Icon: lucide.Tag(), Disabled: a.busy || a.loading},
		overlay.MenuItem{Key: "new-branch", Label: a.t("createBranchHere"), Icon: lucide.GitBranch(), Disabled: a.busy || a.loading},
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{Key: "cherry-pick", Label: a.t("cherryPick"), Icon: lucide.GitCommitHorizontal(), Disabled: a.busy || a.loading},
		overlay.MenuItem{Key: "revert", Label: a.t("revert"), Icon: lucide.Undo2(), Disabled: a.busy || a.loading},
	)
	// Undo belongs to HEAD alone: undoing a commit further back would also
	// drop every commit above it, which is what reset does. It takes the place
	// the last-commit card's menu had before the working tree moved onto this
	// page.
	if headBranch(commit.Refs) != "" {
		items = append(items, overlay.MenuItem{
			Key: "undo", Label: a.t("undoCommit"), Icon: lucide.Undo2(), Disabled: a.busy || a.loading,
		})
	}
	return append(items,
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{Key: "reset", Label: resetLabel, Icon: lucide.RotateCcw(), Danger: true, Disabled: a.busy || a.loading},
	)
}

func (a *app) integrate(kind, sha string) {
	busy := busyCherryPick
	label := a.t("cherryPickFailed")
	success := a.t("cherryPickSuccess")
	switch kind {
	case "revert":
		busy = busyRevert
		label = a.t("revertFailed")
		success = a.t("revertSuccess")
	case "merge":
		busy = busyMerge
		label = a.t("mergeFailed")
		success = a.t("mergeSuccess")
	case "rebase":
		busy = busyRebase
		label = a.t("rebaseFailed")
		success = a.t("rebaseSuccess")
	}
	a.performAs(busy, label, success, func(ctx context.Context, repo *gitrepo.Repository) error {
		return repo.Integrate(ctx, kind, sha)
	})
}

// continueOperation finishes an interrupted operation through git's own
// continue command, which is the only path for a rebase.
func (a *app) continueOperation(kind string) {
	a.performAs(busyContinue, a.t("continueFailed"), a.t("continueSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
		return repo.Continue(ctx, kind)
	})
}

// confirmAbortOperation asks first because aborting discards the conflict
// resolutions made so far.
func (a *app) confirmAbortOperation(kind string) {
	a.askConfirm(a.t("abortTitle"), a.tf("abortMessage", kind), a.t("abortOperation"), true, func() {
		a.performAs(busyAbort, a.t("abortFailed"), a.t("abortSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.Abort(ctx, kind)
		})
	})
}

// confirmSkipOperation drops the commit the operation stopped on, so it asks
// first like abort.
func (a *app) confirmSkipOperation(kind string) {
	a.askConfirm(a.t("skipTitle"), a.tf("skipMessage", kind), a.t("skipOperation"), true, func() {
		a.performAs(busySkip, a.t("skipFailed"), a.t("skipSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.Skip(ctx, kind)
		})
	})
}

// branchMenuItems builds the branch row's menu. It is separate from the menu
// itself so the grouping rules (copies together, a divider before a jump to
// history, danger last) can be tested without opening a menu.
func (a *app) branchMenuItems(branch gitrepo.Branch) []overlay.MenuItem {
	items := []overlay.MenuItem{
		{Key: "switch", Label: a.t("switch"), Icon: lucide.GitBranch(), Disabled: branch.Current || a.busy || a.loading},
		{Key: "merge", Label: a.t("mergeIntoCurrentAction"), Icon: lucide.GitMerge(), Disabled: branch.Current || a.busy || a.loading},
		{Key: "rebase", Label: a.t("rebaseOntoAction"), Icon: lucide.GitCompareArrows(), Disabled: branch.Current || a.busy || a.loading},
	}
	// Pulling and upstream resets act on the checked-out branch, so they stay
	// on its menu instead of pretending they work for another branch.
	if branch.Current && branch.Upstream != "" && !branch.UpstreamGone {
		items = append(items,
			overlay.MenuItem{Key: "pull", Label: a.t("pull"), Icon: lucide.ArrowDown(), Disabled: a.busy || a.loading},
			overlay.MenuItem{Key: "reset-upstream", Label: a.t("resetToUpstream"), Icon: lucide.RotateCcw(), Disabled: a.busy || a.loading},
		)
	}
	if remote, _ := a.branchPushTarget(branch); remote != "" {
		items = append(items, overlay.MenuItem{
			Key: "push", Label: a.t("pushBranchAction"), Icon: lucide.Upload(), Disabled: a.busy || a.loading,
		})
	}
	items = append(items,
		overlay.MenuItem{Key: "copy", Label: a.t("copyName"), Icon: lucide.Copy()},
		overlay.MenuItem{Key: "copy-sha", Label: a.t("copySHA"), Icon: lucide.Copy(), Extra: shortSHA(branch.SHA)},
		// Opening the branch's history is a jump, not a copy, so the two
		// groups are separated the way the file menu separates them.
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{Key: "history", Label: a.t("viewHistory"), Icon: lucide.History()},
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{Key: "new", Label: a.t("newBranchAction"), Icon: lucide.Plus(), Disabled: a.busy || a.loading},
		overlay.MenuItem{Key: "new-no-checkout", Label: a.t("newBranchNoCheckoutAction"), Icon: lucide.Plus(), Disabled: a.busy || a.loading},
		overlay.MenuItem{Key: "rename", Label: a.t("renameBranch"), Icon: lucide.Pencil(), Disabled: a.busy || a.loading},
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{Key: "delete", Label: a.t("deleteBranch"), Icon: lucide.Trash2(), Danger: true, Disabled: branch.Current || a.busy || a.loading},
		overlay.MenuItem{Key: "force-delete", Label: a.t("forceDeleteBranchAction"), Icon: lucide.Trash2(), Danger: true, Disabled: branch.Current || a.busy || a.loading},
	)
	return items
}

func (a *app) branchMenu(c *ui.Context, row ui.Element, branch gitrepo.Branch) {
	items := a.branchMenuItems(branch)
	switch overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: items}) {
	case "switch":
		a.switchBranch(branch.Name)
	case "merge":
		a.askConfirm(a.t("mergeIntoCurrent"), a.tf("mergeCommitMessage", branch.Name),
			a.t("mergeIntoCurrent"), false, func() { a.integrate("merge", branch.Name) })
	case "rebase":
		a.askConfirm(a.t("rebaseOnto"), a.tf("rebaseCommitMessage", branch.Name),
			a.t("rebaseOnto"), false, func() { a.integrate("rebase", branch.Name) })
	case "pull":
		a.pullFF()
	case "reset-upstream":
		a.openResetToUpstream()
	case "push":
		a.confirmPushBranch(branch)
	case "copy":
		a.copyText(branch.Name)
	case "copy-sha":
		a.copyText(branch.SHA)
	case "history":
		a.showRefHistory(historyRefLocal, branch.Name)
	case "new":
		a.promptNewBranch(branch.SHA)
	case "new-no-checkout":
		a.promptNewBranchAt(branch.SHA, false)
	case "rename":
		a.promptRenameBranch(branch.Name)
	case "delete":
		a.confirmDeleteBranch(branch.Name)
	case "force-delete":
		a.confirmForceDeleteBranch(branch.Name)
	}
}

func (a *app) remoteBranchMenuItems(ref gitrepo.Branch) []overlay.MenuItem {
	remote, _ := a.remoteBranchParts(ref.Name)
	return []overlay.MenuItem{
		{Key: "checkout", Label: a.t("checkoutRemoteBranch"), Icon: lucide.GitBranch(), Disabled: a.busy || a.loading},
		{Key: "merge", Label: a.t("mergeIntoCurrentAction"), Icon: lucide.GitMerge(), Disabled: a.busy || a.loading},
		{Key: "copy", Label: a.t("copyName"), Icon: lucide.Copy()},
		{Key: "copy-sha", Label: a.t("copySHA"), Icon: lucide.Copy(), Extra: shortSHA(ref.SHA)},
		{Divider: true},
		{Key: "history", Label: a.t("viewHistory"), Icon: lucide.History()},
		{Divider: true},
		{Key: "delete", Label: a.t("deleteRemoteBranch"), Icon: lucide.Trash2(), Danger: true, Disabled: remote == "" || a.busy || a.loading},
	}
}

func (a *app) remoteBranchMenu(c *ui.Context, row ui.Element, ref gitrepo.Branch) {
	remote, name := a.remoteBranchParts(ref.Name)
	items := a.remoteBranchMenuItems(ref)
	switch overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: items}) {
	case "checkout":
		a.promptCheckoutRemoteBranch(ref.Name, name)
	case "merge":
		a.askConfirm(a.t("mergeIntoCurrent"), a.tf("mergeCommitMessage", ref.Name),
			a.t("mergeIntoCurrent"), false, func() { a.integrate("merge", ref.Name) })
	case "copy":
		a.copyText(ref.Name)
	case "copy-sha":
		a.copyText(ref.SHA)
	case "history":
		a.showRefHistory(historyRefRemote, ref.Name)
	case "delete":
		a.confirmDeleteRemoteBranch(remote, name)
	}
}

func (a *app) promptCheckoutRemoteBranch(remoteRef, name string) {
	// A local branch already tracking this ref is the checkout the reader
	// wants; offering to create another branch with the same name would fail
	// in git ("a branch named ... already exists").
	if tracker := a.trackingBranch(remoteRef); tracker != "" {
		a.switchBranch(tracker)
		return
	}
	a.askPrompt(a.t("checkoutRemoteBranchTitle"), a.t("branchName"), name, a.t("create"), func(name string) {
		a.performAs(busyCheckout, a.t("checkoutRemoteBranchFailed"), a.t("remoteBranchCheckedOut"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.TrackBranch(ctx, name, remoteRef)
		})
	})
}

func (a *app) confirmDeleteRemoteBranch(remote, name string) {
	a.askConfirm(a.t("deleteRemoteBranchTitle"), a.tf("deleteRemoteBranchMessage", remote, name), a.t("delete"), true, func() {
		a.performAs(busyDeleteRemoteBranch, a.t("deleteRemoteBranchFailed"), a.t("remoteBranchDeleted"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.DeleteRemoteBranch(ctx, remote, name)
		})
	})
}

func (a *app) tagMenuItems(tag gitrepo.Tag) []overlay.MenuItem {
	items := []overlay.MenuItem{
		{Key: "branch", Label: a.t("createBranchFromTag"), Icon: lucide.GitBranch(), Disabled: tag.TargetType != "commit" || a.busy || a.loading},
		{Key: "copy", Label: a.t("copyTagName"), Icon: lucide.Copy()},
		{Key: "copy-sha", Label: a.t("copySHA"), Icon: lucide.Copy(), Extra: shortSHA(tag.TargetSHA)},
		{Divider: true},
		{Key: "history", Label: a.t("viewHistory"), Icon: lucide.History()},
		{Divider: true},
	}
	if a.tagPushRemote() != "" {
		items = append(items, overlay.MenuItem{
			Key: "push", Label: a.t("pushTag"), Icon: lucide.Upload(), Disabled: a.busy || a.loading,
		})
	}
	items = append(items,
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{Key: "delete", Label: a.t("deleteTagAction"), Icon: lucide.Trash2(), Danger: true, Disabled: a.busy || a.loading})
	return items
}

func (a *app) tagMenu(c *ui.Context, row ui.Element, tag gitrepo.Tag) {
	items := a.tagMenuItems(tag)
	switch overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: items}) {
	case "branch":
		a.promptNewBranch(tag.TargetSHA)
	case "copy":
		a.copyText(tag.Name)
	case "copy-sha":
		a.copyText(tag.TargetSHA)
	case "history":
		a.showRefHistory(historyRefTag, tag.Name)
	case "push":
		a.confirmPushTag(tag.Name)
	case "delete":
		a.confirmDeleteTag(tag)
	}
}

func (a *app) tagPushRemote() string {
	for _, remote := range a.remotes {
		if remote.Name == "origin" {
			return "origin"
		}
	}
	if len(a.remotes) > 0 {
		return a.remotes[0].Name
	}
	return ""
}

func (a *app) confirmPushTag(name string) {
	remote := a.tagPushRemote()
	if remote == "" {
		a.queue("warning", a.t("noRemote"), "")
		a.invalidate()
		return
	}
	a.askConfirm(a.t("pushTagTitle"), a.tf("pushTagMessage", name, remote), a.t("pushButton"), false, func() {
		a.performAs(busyPushTag, a.t("pushTagFailed"), a.t("tagPushed"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.PushTag(ctx, remote, name)
		})
	})
}

func (a *app) confirmDeleteTag(tag gitrepo.Tag) {
	a.askConfirm(a.t("deleteTagTitle"), a.tf("deleteTagMessage", tag.Name), a.t("delete"), true, func() {
		a.perform(a.t("deleteTagFailed"), a.t("tagDeleted"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.DeleteTag(ctx, tag.Name)
		})
	})
}

func (a *app) promptCreateTag(sha string) {
	a.askPrompt(a.t("createTagTitle"), a.t("tagName"), "", a.t("create"), func(name string) {
		a.perform(a.t("createTagFailed"), a.t("tagCreated"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.CreateTag(ctx, name, sha)
		})
	})
}

func (a *app) promptCreateAnnotatedTag(sha string) {
	a.askPrompt(a.t("createAnnotatedTagTitle"), a.t("tagName"), "", a.t("next"), func(name string) {
		a.askPrompt(a.t("tagMessageTitle"), a.t("tagMessage"), "", a.t("create"), func(message string) {
			a.perform(a.t("createAnnotatedTagFailed"), a.t("annotatedTagCreated"), func(ctx context.Context, repo *gitrepo.Repository) error {
				return repo.CreateAnnotatedTag(ctx, name, sha, message)
			})
		})
	})
}

func (a *app) promptNewBranch(start string) {
	a.promptNewBranchAt(start, true)
}

func (a *app) promptNewBranchAt(start string, checkout bool) {
	a.askPrompt(a.t("newBranchTitle"), a.t("branchName"), "", a.t("create"), func(name string) {
		success := a.t("branchCreated")
		if !checkout {
			success = a.t("branchCreatedNoCheckout")
		}
		a.perform(a.t("createBranchFailed"), success, func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.CreateBranch(ctx, name, start, checkout)
		})
	})
}

func (a *app) promptRenameBranch(old string) {
	a.askPrompt(a.t("renameBranchTitle"), a.t("branchName"), old, a.t("rename"), func(name string) {
		if name == old {
			return
		}
		a.perform(a.t("renameBranchFailed"), a.t("branchRenamed"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.RenameBranch(ctx, old, name)
		})
	})
}

func (a *app) confirmDeleteBranch(name string) {
	a.askConfirm(a.t("deleteBranchTitle"), a.tf("deleteBranchMessage", name), a.t("delete"), true, func() {
		a.perform(a.t("deleteBranchFailed"), a.t("branchDeleted"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.DeleteBranch(ctx, name)
		})
	})
}

func (a *app) confirmForceDeleteBranch(name string) {
	a.askConfirm(a.t("forceDeleteTitle"), a.tf("forceDeleteMessage", name), a.t("forceDelete"), true, func() {
		a.perform(a.t("forceDeleteFailed"), a.t("branchForceDeleted"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.ForceDeleteBranch(ctx, name)
		})
	})
}

func (a *app) resolveConflict(path string) {
	a.perform(a.t("resolveFailed"), a.t("resolveSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
		return repo.Stage(ctx, path)
	})
}

func (a *app) takeConflictSide(path, side string) {
	success := a.t("takeOursSuccess")
	if side == "theirs" {
		success = a.t("takeTheirsSuccess")
	}
	a.perform(a.t("takeSideFailed"), success, func(ctx context.Context, repo *gitrepo.Repository) error {
		return repo.TakeConflictSide(ctx, path, side)
	})
}

func (a *app) fileDetailMenu(c *ui.Context, row ui.Element, path string) {
	switch overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: []overlay.MenuItem{
		{Key: "copy-path", Label: a.t("copyPath"), Icon: lucide.Copy()},
		{Key: "absolute", Label: a.t("copyAbsolutePath"), Icon: lucide.Copy()},
		{Divider: true},
		{Key: "reveal", Label: a.t("revealInFinder"), Icon: lucide.FolderOpen()},
		{Divider: true},
		{Key: "history", Label: a.t("fileHistory"), Icon: lucide.History()},
		// This menu serves the commit details only. Blame reads the selected
		// commit's version, and a file that commit deleted has none.
		{Key: "blame", Label: a.blameMenuLabel(path), Icon: lucide.FileSearch(), Disabled: a.commitBlameDisabled(path)},
	}}) {
	case "copy-path":
		a.copyText(path)
	case "absolute":
		a.copyText(a.absolutePath(path))
	case "reveal":
		a.openFile(path, "reveal")
	case "history":
		a.showFileHistory(path)
	case "blame":
		a.openBlameAt(path, a.selectedSHA)
	}
}
