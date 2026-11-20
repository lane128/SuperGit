package lobedesk

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

type promptState struct {
	root        string
	open        bool
	title       string
	placeholder string
	description string
	value       string
	actionName  string
	action      func(string)
	allowEmpty  bool
	// selectAll asks the field to select its prefilled value once, in the
	// first frame after the dialog opens.
	selectAll bool
	// Palette handoffs retain the picker's overlay and original opener.
	paletteHandoff bool
	paletteFocus   bool
	paletteCancel  func()
}

func (a *app) discardStalePrompt() {
	if a.prompt.root != "" && (a.repo == nil || a.repo.Root != a.prompt.root) {
		if a.prompt.paletteCancel != nil {
			a.prompt.paletteCancel()
		}
		a.prompt = promptState{}
	}
}

func (a *app) askPrompt(title, placeholder, initial, actionName string, action func(string)) {
	a.askTextPrompt(title, placeholder, initial, actionName, false, action)
}

func (a *app) askTextPrompt(title, placeholder, initial, actionName string, allowEmpty bool, action func(string)) {
	a.discardStalePrompt()
	if a.prompt.open {
		a.queue("warning", a.t("promptPending"), "")
		a.invalidate()
		return
	}
	if a.rejectPendingDialog(toastMain) {
		return
	}
	root := ""
	if a.repo != nil {
		root = a.repo.Root
	}
	if action != nil && root != "" {
		run := action
		action = func(value string) {
			if a.repo != nil && a.repo.Root == root {
				run(value)
			}
		}
	}
	a.prompt = promptState{
		root:        root,
		open:        true,
		title:       title,
		placeholder: placeholder,
		value:       initial,
		actionName:  actionName,
		action:      action,
		allowEmpty:  allowEmpty,
		selectAll:   initial != "",
	}
	a.invalidate()
}

func (a *app) promptDialog(c *ui.Context) {
	a.promptDialogFor(c, toastMain)
}

// All current generic prompts originate in repository actions in the main
// window. Settings must not adopt an input when it takes keyboard focus.
func (a *app) promptDialogFor(c *ui.Context, owner toastOwner) {
	if owner != toastMain {
		return
	}
	a.discardStalePrompt()
	if !a.prompt.open || a.prompt.paletteHandoff {
		return
	}
	tk := theme.TokensOf(c)
	confirmed := false
	ui.Modal(c.Key("workbench-prompt"), &a.prompt.open, func() {
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
				ready := a.prompt.allowEmpty || strings.TrimSpace(a.prompt.value) != ""
				if input.Submitted() && ready {
					confirmed = true
					a.prompt.open = false
				}
				ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Justify(ui.End).Children(func() {
					if workbenchButton(c, a.t("cancel"), buttonNormal, false).Clicked() {
						a.prompt.open = false
					}
					button := workbenchButton(c, a.prompt.actionName, buttonPrimary, !ready)
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
	owner       toastOwner
	root        string // empty for settings, which are independent of repository
	open        bool
	title       string
	message     string
	actionName  string
	danger      bool
	action      func()
	patchScope  bool
	undoFile    bool
	patchScroll ui.ScrollState
}

func (a *app) askConfirm(title, message, actionName string, danger bool, action func()) {
	root := ""
	if a.repo != nil {
		root = a.repo.Root
	}
	a.askConfirmInWindow(toastMain, root, title, message, actionName, danger, action)
}

func (a *app) discardStaleConfirmation() {
	if a.confirm.root != "" && (a.repo == nil || a.repo.Root != a.confirm.root) {
		a.confirm = confirmState{}
	}
	if a.confirm.owner == toastSettings {
		if w := a.settingsWindowSlot().Load(); w == nil || w.IsDestroyed() {
			a.confirm = confirmState{}
		}
	}
}

func (a *app) askConfirmInWindow(owner toastOwner, root, title, message, actionName string, danger bool, action func()) {
	a.discardStaleConfirmation()
	if a.confirm.open {
		a.queueFor(owner, "warning", a.t("confirmationPending"), "")
		a.invalidate()
		return
	}
	if a.rejectPendingDialog(owner) {
		return
	}
	if action != nil && root != "" {
		run := action
		action = func() {
			if a.repo != nil && a.repo.Root == root {
				run()
			}
		}
	}
	a.confirm = confirmState{
		owner:      owner,
		root:       root,
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
	a.confirmDialogFor(c, toastMain)
}

func (a *app) confirmDialogFor(c *ui.Context, owner toastOwner) {
	a.discardStaleConfirmation()
	if a.confirm.owner != owner {
		return
	}
	if !a.confirm.open {
		return
	}
	tk := theme.TokensOf(c)
	// Repository workers refuse to start during a read or another write. Keep
	// the original confirmation available instead of consuming that request.
	// Independent settings confirmations need not wait for repository work.
	blocked := a.confirm.root != "" && (a.repo == nil || a.loading || a.busy)
	confirmed := false
	if a.confirm.undoFile {
		confirmed = a.commitFileUndoDialog(c, blocked)
	} else if a.confirm.patchScope {
		confirmed = a.commitFilePatchScopeDialog(c, blocked)
	} else {
		// Other overlays can come and go while the confirmation is open.
		ui.Modal(c.Key("workbench-confirmation"), &a.confirm.open, func() {
			ui.Column(c).Width(430).Padding(20).Gap(10).
				Background(tk.BgElevated).Radius(12).Border(1, tk.Border).
				Children(func() {
					display.Text(c, a.confirm.title, display.TextProps{Strong: true, FontSize: scaledFont(14)})
					display.Text(c, a.confirm.message, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12), Rows: 5})
					ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Justify(ui.End).Margin(8, 0, 0, 0).
						Children(func() {
							if workbenchButton(c, a.t("cancel"), buttonNormal, false).Clicked() {
								a.confirm.open = false
							}
							label := a.confirm.actionName
							role := buttonNormal
							if a.confirm.danger {
								role = buttonDanger
							}
							button := workbenchButton(c, label, role, blocked)
							if !blocked && button.Clicked() {
								confirmed = true
								a.confirm.open = false
							}
						})
				})
		})
	}
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

// openFile hands a repository path to the desktop's file manager or default
// application, keeping the original repository-relative path intact.
func (a *app) openFile(path, kind string) {
	a.openPath(a.absolutePath(path), kind)
}

// openPath reveals or opens a path that is already absolute, such as a
// worktree root that lives outside the repository.
func (a *app) openPath(full, kind string) {
	go func() {
		cmd, err := externalOpenCommand(runtime.GOOS, full, kind)
		if err == nil {
			err = runExternalOpen(cmd)
		}
		if err != nil {
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
	open := ui.Local(row, "contextMenu", func() bool { return false })
	targets := ui.Local(row, "file-stage-targets", func() []fileActionTarget { return nil })
	if !*open {
		*targets = a.workspaceStageTargets(entry, staged)
	}
	items := a.fileMenuItems(entry, staged)
	if len(*targets) > 1 {
		label := "stageSelectedFiles"
		if staged {
			label = "unstageSelectedFiles"
		}
		items[0].Label = a.tf(label, len(*targets))
		items[0].Disabled = !a.workspaceStageReady(*targets)
		for i := range items {
			switch items[i].Key {
			case "discard":
				items[i].Label = a.t("discardThisFile")
				if isUntracked(entry) {
					items[i].Label = a.t("deleteThisFile")
				}
			case "stash":
				items[i].Label = a.t("stashThisFile")
			}
		}
	}
	switch overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: items}) {
	case "stage", "unstage":
		a.runWorkspaceStage(*targets)
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
	_, ready := a.fileActionTarget(entry, staged)
	items := []overlay.MenuItem{}
	if staged {
		items = append(items, overlay.MenuItem{Key: "unstage", Label: a.t("unstageFile"), Icon: lucide.Minus(), Disabled: !ready})
	} else {
		items = append(items, overlay.MenuItem{Key: "stage", Label: a.t("stageFile"), Icon: lucide.Plus(), Disabled: !ready})
		untracked := entry.Index == '?' && entry.Worktree == '?'
		label := a.t("discardChanges")
		icon := lucide.Undo2()
		if untracked {
			label = a.t("deleteFile")
			icon = lucide.Trash2()
		}
		items = append(items,
			overlay.MenuItem{Divider: true},
			overlay.MenuItem{Key: "discard", Label: label, Icon: icon, Danger: true, Disabled: !ready || entry.IsSubmodule()},
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
		{Key: "ours", Label: a.conflictSideText("useOurs"), Icon: lucide.GitBranch()},
		{Key: "theirs", Label: a.conflictSideText("useTheirs"), Icon: lucide.GitMerge()},
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
			overlay.MenuItem{Key: "ours", Label: a.conflictSideText("useOurs"), Icon: lucide.GitBranch()},
			overlay.MenuItem{Key: "theirs", Label: a.conflictSideText("useTheirs"), Icon: lucide.GitMerge()},
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
	a.prepareFileDiscard(entry)
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
		a.switchBranchAt(name, commit.SHA)
		return
	}
	switch action {
	case compareBaseKey, compareTargetKey:
		a.comparisonMenuAction(action, a.comparisonRevision(commit.SHA, ""))
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
		a.confirmIntegrate("merge", shortSHA(commit.SHA), commit.SHA)
	case "rebase":
		a.confirmIntegrate("rebase", shortSHA(commit.SHA), commit.SHA)
	case "reset":
		a.openResetDialog(commit)
	case "tag":
		a.promptCreateTag(commit.SHA)
	case "annotated-tag":
		a.promptCreateAnnotatedTag(commit.SHA)
	case "new-branch":
		a.promptNewBranch(commit.SHA)
	case "cherry-pick":
		a.confirmIntegrate("cherry-pick", shortSHA(commit.SHA), commit.SHA)
	case "revert":
		a.confirmIntegrate("revert", shortSHA(commit.SHA), commit.SHA)
	case "undo":
		expected := gitrepo.Status{Head: commit.SHA, Branch: a.status.Branch}
		a.askConfirm(a.t("undoCommitTitle"), a.tf("undoCommitMessage", shortSHA(commit.SHA)), a.t("undo"), false, func() {
			a.performAs(busyUndoCommit, a.t("undoFailed"), a.t("undoSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
				return repo.UndoCommitChecked(ctx, expected)
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
	items = append(items, a.comparisonMenuItems(a.comparisonRevision(commit.SHA, ""))...)
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

// confirmIntegrate freezes both sides before opening the confirmation.
func (a *app) confirmIntegrate(kind, label, sha string) {
	var title, message, action string
	switch kind {
	case "merge":
		title, message, action = "mergeIntoCurrent", "mergeCommitMessage", "mergeIntoCurrent"
	case "rebase":
		title, message, action = "rebaseOnto", "rebaseCommitMessage", "rebaseOnto"
	case "cherry-pick":
		title, message, action = "cherryPickTitle", "cherryPickMessage", "cherryPickAction"
	case "revert":
		title, message, action = "revertTitle", "revertMessage", "revertAction"
	default:
		return
	}
	expected := gitrepo.Status{Head: a.status.Head, Branch: a.status.Branch}
	if label != shortSHA(sha) {
		label = fmt.Sprintf("%s (%s)", label, shortSHA(sha))
	}
	a.askConfirm(a.t(title), a.tf(message, label), a.t(action), false, func() {
		a.integrate(kind, sha, expected)
	})
}

func (a *app) integrate(kind, sha string, expected gitrepo.Status) {
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
		return repo.IntegrateChecked(ctx, kind, sha, expected)
	})
}

// continueOperation finishes an interrupted operation through git's own
// continue command, which is the only path for a rebase.
func (a *app) continueOperation(kind string) {
	target := a.operation
	if target.Kind != kind || kind == "" {
		return
	}
	a.performAs(busyContinue, a.t("continueFailed"), a.t("continueSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
		return repo.ContinueChecked(ctx, target)
	})
}

// confirmAbortOperation asks first because aborting discards the conflict
// resolutions made so far.
func (a *app) confirmAbortOperation(kind string) {
	target := a.operation
	if target.Kind != kind || kind == "" {
		return
	}
	a.askConfirm(a.t("abortTitle"), a.tf("abortMessage", kind), a.t("abortOperation"), true, func() {
		a.performAs(busyAbort, a.t("abortFailed"), a.t("abortSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.AbortChecked(ctx, target)
		})
	})
}

// confirmSkipOperation drops the commit the operation stopped on, so it asks
// first like abort.
func (a *app) confirmSkipOperation(kind string) {
	target := a.operation
	if target.Kind != kind || kind == "" {
		return
	}
	a.askConfirm(a.t("skipTitle"), a.tf("skipMessage", kind), a.t("skipOperation"), true, func() {
		a.performAs(busySkip, a.t("skipFailed"), a.t("skipSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.SkipChecked(ctx, target)
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
		a.confirmIntegrate("merge", branch.Name, branch.SHA)
	case "rebase":
		a.confirmIntegrate("rebase", branch.Name, branch.SHA)
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
		a.promptRenameBranch(branch)
	case "delete":
		a.confirmDeleteBranch(branch)
	case "force-delete":
		a.confirmForceDeleteBranch(branch)
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
		a.confirmIntegrate("merge", ref.Name, ref.SHA)
	case "copy":
		a.copyText(ref.Name)
	case "copy-sha":
		a.copyText(ref.SHA)
	case "history":
		a.showRefHistory(historyRefRemote, ref.Name)
	case "delete":
		a.confirmDeleteRemoteBranch(remote, name, ref.SHA)
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

func (a *app) confirmDeleteRemoteBranch(remoteName, name, sha string) {
	var remote gitrepo.Remote
	for _, candidate := range a.remotes {
		if candidate.Name == remoteName {
			remote = candidate
			break
		}
	}
	if remote.Name == "" {
		a.queue("warning", a.t("noRemote"), "")
		a.invalidate()
		return
	}
	a.askConfirm(a.t("deleteRemoteBranchTitle"), a.tf("deleteRemoteBranchMessage", remote.Name, name, shortSHA(sha)), a.t("delete"), true, func() {
		a.performAs(busyDeleteRemoteBranch, a.t("deleteRemoteBranchFailed"), a.t("remoteBranchDeleted"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.DeleteRemoteBranchChecked(ctx, remote, name, sha)
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
	if tag.TargetType == "commit" {
		items = append(items, a.comparisonMenuItems(a.comparisonRevision(tag.TargetSHA, tag.Name))...)
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
	switch action := overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: items}); action {
	case compareBaseKey, compareTargetKey:
		if tag.TargetType == "commit" {
			a.comparisonMenuAction(action, a.comparisonRevision(tag.TargetSHA, tag.Name))
		}
	case "branch":
		a.promptNewBranch(tag.TargetSHA)
	case "copy":
		a.copyText(tag.Name)
	case "copy-sha":
		a.copyText(tag.TargetSHA)
	case "history":
		a.showRefHistory(historyRefTag, tag.Name)
	case "push":
		a.confirmPushTag(tag)
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

func (a *app) confirmPushTag(tag gitrepo.Tag) {
	var remote gitrepo.Remote
	for _, candidate := range a.remotes {
		if candidate.Name == a.tagPushRemote() {
			remote = candidate
			break
		}
	}
	if remote.Name == "" {
		a.queue("warning", a.t("noRemote"), "")
		a.invalidate()
		return
	}
	a.askConfirm(a.t("pushTagTitle"), a.tf("pushTagMessage", tag.Name, shortSHA(tag.ObjectSHA), remote.Name), a.t("pushButton"), false, func() {
		a.performAs(busyPushTag, a.t("pushTagFailed"), a.t("tagPushed"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.PushTagChecked(ctx, remote, tag)
		})
	})
}

func (a *app) confirmDeleteTag(tag gitrepo.Tag) {
	a.askConfirm(a.t("deleteTagTitle"), a.tf("deleteTagMessage", tag.Name, shortSHA(tag.ObjectSHA)), a.t("delete"), true, func() {
		a.perform(a.t("deleteTagFailed"), a.t("tagDeleted"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.DeleteTagChecked(ctx, tag.Name, tag.ObjectSHA)
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

func (a *app) promptRenameBranch(branch gitrepo.Branch) {
	a.askPrompt(a.t("renameBranchTitle"), a.t("branchName"), branch.Name, a.t("rename"), func(name string) {
		if name == branch.Name {
			return
		}
		a.perform(a.t("renameBranchFailed"), a.t("branchRenamed"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.RenameBranchChecked(ctx, branch.Name, name, branch.SHA)
		})
	})
}

func (a *app) confirmDeleteBranch(branch gitrepo.Branch) {
	a.askConfirm(a.t("deleteBranchTitle"), a.tf("deleteBranchMessage", branch.Name, shortSHA(branch.SHA)), a.t("delete"), true, func() {
		a.perform(a.t("deleteBranchFailed"), a.t("branchDeleted"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.DeleteBranchChecked(ctx, branch.Name, branch.SHA, false)
		})
	})
}

func (a *app) confirmForceDeleteBranch(branch gitrepo.Branch) {
	a.askConfirm(a.t("forceDeleteTitle"), a.tf("forceDeleteMessage", branch.Name, shortSHA(branch.SHA)), a.t("forceDelete"), true, func() {
		a.perform(a.t("forceDeleteFailed"), a.t("branchForceDeleted"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.DeleteBranchChecked(ctx, branch.Name, branch.SHA, true)
		})
	})
}

func (a *app) resolveConflict(path string) {
	a.perform(a.t("resolveFailed"), a.t("resolveSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
		return repo.Stage(ctx, path)
	})
}

// During rebase, Git's ours is the base with already replayed commits;
// theirs is the commit currently being replayed, not an incoming branch.
func (a *app) conflictSideText(key string) string {
	if a.operation.Kind == "rebase" {
		key += "Rebase"
	}
	return a.t(key)
}

func (a *app) takeConflictSide(path, side string) {
	success := a.conflictSideText("takeOursSuccess")
	if side == "theirs" {
		success = a.conflictSideText("takeTheirsSuccess")
	}
	a.perform(a.t("takeSideFailed"), success, func(ctx context.Context, repo *gitrepo.Repository) error {
		return repo.TakeConflictSide(ctx, path, side)
	})
}

func (a *app) fileDetailMenu(c *ui.Context, row ui.Element, path string) {
	open := ui.Local(row, "contextMenu", func() bool { return false })
	target := ui.Local(row, "commit-file-patch-target", func() commitFilePatchTarget { return commitFilePatchTarget{} })
	undoTarget := ui.Local(row, "commit-file-undo-target", func() commitFileUndoTarget { return commitFileUndoTarget{} })
	if !*open {
		*target = a.commitFilePatchTarget(path)
		*undoTarget = a.commitFileUndoTarget(path)
	}
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
		{Divider: true},
		{Key: "export-patch", Label: a.t("exportFilePatch"), Icon: lucide.Download(), Disabled: !a.commitFilePatchReady(*target)},
		{Key: "undo-file", Label: a.t("undoFile"), Disabled: !a.commitFileUndoReady(*undoTarget)},
	}}) {
	case "undo-file":
		a.requestCommitFileUndo(*undoTarget)
	case "export-patch":
		a.requestCommitFilePatch(*target)
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
