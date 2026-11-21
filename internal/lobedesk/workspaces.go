package lobedesk

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func (a *app) worktreesPage(c *ui.Context) {
	a.syncWorkspaceSelection()
	tk := theme.TokensOf(c)
	a.pageHeader(c, a.t("worktreesAndSubmodules"),
		a.workspacesSubtitle(len(a.worktrees), len(a.submodules), a.worktreesErr != "", a.submodulesErr != ""), func() {
			button := workbenchButton(c, a.t("newWorktree"), buttonNormal, a.busy || a.loading)
			if button.Clicked() {
				a.openWorktreeForm()
			}
		})
	list := ui.Column(c).Grow(1).Basis(0).MinHeight(0).Focusable()
	ring := ui.Transparent
	if list.FocusWithin() {
		ring = tk.Blue.Alpha(0.35)
	}
	list.Border(1, ring)
	list.Children(func() {
		ui.Row(c).Height(34).Shrink(0).Padding(0, 16).Gap(8).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
			display.Text(c, a.t("worktreesHelp"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)})
		})
		layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.workspacesScroll}, func() {
			ui.Column(c).FillWidth().Padding(10).Gap(2).Children(func() {
				a.worktreeSection(c)
				a.submoduleSection(c)
			})
		}).Grow(1).Basis(0).MinHeight(0)
	})
}

func (a *app) worktreeSection(c *ui.Context) {
	failed := a.worktreesErr != ""
	ui.Row(c).FillWidth().Height(32).Margin(0, 12, 0, 2).PaddingX(4).Gap(6).AlignItems(ui.Center).Children(func() {
		display.Text(c, a.t("worktrees"), display.TextProps{Type: display.TextSecondary, Strong: true, FontSize: scaledFont(11)})
		display.Text(c, a.workspaceGroupCount(len(a.worktrees), failed), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)})
	})
	if failed {
		// The error belongs to this section: the submodule list below may have
		// read fine, and one shared message could not say which failed.
		a.failedWithNamedRetry(c, a.worktreesErr, 12, a.t("retryWorktrees"))
	}
	if len(a.worktrees) == 0 {
		// An empty list after a failed read is not "there are no worktrees";
		// the section's error already says what happened.
		if !failed {
			sectionEmptyState(c, a.t("noWorktrees"), a.t("noWorktreesHint"), 11)
		}
		return
	}
	ui.Column(c).FillWidth().Gap(2).Role(ui.RoleList).Children(func() {
		for i, worktree := range a.worktrees {
			a.worktreeRow(c, worktree, i+1, len(a.worktrees))
		}
	})
}

func (a *app) worktreeRow(c *ui.Context, worktree gitrepo.Worktree, pos, size int) {
	tk := theme.TokensOf(c)
	current := a.repo != nil && filepath.Clean(worktree.Path) == filepath.Clean(a.repo.Root)
	key := "worktree:" + worktree.Path
	selected := a.worktreeSel == key
	// The row is an option of the list, as the branch and remote rows are: Tab
	// reaches it, the menu key opens its menu, and a screen reader hears which
	// checkout the page has chosen. The name is the folder it shows first; the
	// description keeps the full path that the label replaces.
	row := ui.ButtonBase(c).Label(filepath.Base(worktree.Path)).Description(worktree.Path).
		Row().FillWidth().MinHeight(46).Padding(5, 8).Gap(10).AlignItems(ui.Center).
		Radius(tk.BorderRadius).Border(1, ui.Transparent).
		Role(ui.RoleOption).AccessSelected(selected).PositionInSet(pos, size)
	if current {
		row.Description(a.t("current") + " · " + worktree.Path)
	}
	row.OnShortcut(0, ui.KeyEnter, func() { a.activateWorktreeSelection() })
	// Space activates the focused button, which would only reselect the row the
	// pointer chose; mirror Enter so both keys run the page's action.
	row.OnShortcut(0, ui.KeySpace, func() { a.activateWorktreeSelection() })
	// As in the branch list, selection comes first so the row the keyboard is
	// on always reads as selected; the current checkout keeps the neutral tint
	// only while the selection is elsewhere. Blue stays reserved for the
	// selection, so "selected" and "current" never look alike.
	switch {
	case selected:
		row.Background(selectionBg(tk, tk.BgContainer))
	case current:
		row.Background(tk.FillSecondary)
	case row.Hovered():
		row.Background(tk.FillTertiary)
	}
	row.Children(func() {
		ui.Box(c).Size(30, 30).Shrink(0).Radius(9).Center().Background(tk.FillTertiary).Children(func() {
			ui.Icon(c, lucide.FolderTree()).FontSize(scaledFont(15)).TextColor(tk.TextSecondary)
		})
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(2).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				display.Text(c, filepath.Base(worktree.Path), display.TextProps{FontSize: scaledFont(12), Strong: current}).SingleLine().AccessHidden(true)
				if current {
					a.statusTag(c, a.t("current"), display.TagProps{
						Variant: display.TagFilled, Color: display.TagSuccess, Size: theme.Small, Round: true,
					})
				}
				if worktree.Bare {
					display.Tag(c, a.t("bareRepo"), display.TagProps{Variant: display.TagBorderless, Size: theme.Small})
				}
				if worktree.Locked {
					a.statusTag(c, a.t("lockedWorktree"), display.TagProps{
						Variant: display.TagFilled, Color: display.TagWarning, Size: theme.Small, Round: true,
					})
				}
			})
			display.Text(c, worktree.Path, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(10)}).SingleLine().Tooltip(worktree.Path)
		})
		branch := worktree.Branch
		if branch == "" {
			branch = a.t("detached")
		}
		display.Text(c, branch, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(10)}).Shrink(0)
	})
	row.OnClick(func() { a.worktreeSel = key; a.invalidate() })
	if row.DoubleClicked() && !current {
		a.openRepository(worktree.Path, true)
	}
	a.revealWorkspaceRow(row, selected)
	a.worktreeMenu(c, row, worktree, current)
	// Opening the row's menu also selects the row, matching the other lists.
	if *ui.Local(row, "contextMenu", func() bool { return false }) && a.worktreeSel != key {
		a.worktreeSel = key
		a.invalidate()
	}
}

func (a *app) worktreeMenu(c *ui.Context, row ui.Element, worktree gitrepo.Worktree, current bool) {
	removable := !current && !worktree.Bare && !worktree.Locked && !a.busy && !a.loading
	items := []overlay.MenuItem{}
	if !current {
		items = append(items, overlay.MenuItem{
			Key: "switch", Label: a.t("switchWorktree"), Icon: lucide.GitBranch(), Disabled: a.busy || a.loading,
		}, overlay.MenuItem{Divider: true})
	}
	items = append(items,
		overlay.MenuItem{Key: "reveal", Label: a.t("revealInFinder"), Icon: lucide.FolderOpen()},
		overlay.MenuItem{Key: "copy", Label: a.t("copyAbsolutePath"), Icon: lucide.Copy()},
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{Key: "remove", Label: a.t("removeWorktree"), Icon: lucide.Trash2(), Danger: true, Disabled: !removable},
	)
	switch overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: items}) {
	case "switch":
		a.openRepository(worktree.Path, true)
	case "reveal":
		a.openPath(worktree.Path, "reveal")
	case "copy":
		a.copyText(worktree.Path)
	case "remove":
		a.confirmRemoveWorktree(worktree)
	}
}

func (a *app) confirmRemoveWorktree(worktree gitrepo.Worktree) {
	message := a.tf("removeWorktreeMessage", worktree.Path)
	if worktree.Branch != "" {
		message = a.tf("removeWorktreeBranchMessage", worktree.Path, worktree.Branch)
	}
	a.askConfirm(a.t("removeWorktreeTitle"), message, a.t("remove"), true, func() {
		a.removeWorktree(worktree)
	})
}

// removeWorktree removes the checkout and, when it still has a branch, offers
// to delete that branch from the success toast. Git keeps a worktree's branch
// after the checkout goes away, so the branch would otherwise stay behind.
func (a *app) removeWorktree(worktree gitrepo.Worktree) {
	if a.repo == nil || a.busy || a.loading || a.tabSnapshotUnavailable() {
		return
	}
	repo := a.repo
	ctx, cancel := a.beginOperation(busyRemoveWorktree)
	go func() {
		defer cancel()
		err := repo.RemoveWorktreeChecked(ctx, worktree)
		a.hopToMain(func() {
			a.finishOperation()
			switch {
			case operationCancelled(ctx, err):
				a.queue("info", a.t("operationCancelled"), "")
			case err != nil:
				a.queue("error", a.t("removeWorktreeFailed"), a.operationErrorDescription(err))
			case worktree.Branch != "":
				branch := gitrepo.Branch{Name: worktree.Branch, SHA: worktree.HEAD}
				a.pushNotice(toastMain, &toastNotice{
					kind:        "success",
					title:       a.t("worktreeRemoved"),
					description: a.tf("worktreeBranchKept", branch.Name),
					action:      a.t("delete"),
					onAction:    func() { a.deleteBranchAfterWorktree(repo, branch) },
					timeout:     10 * time.Second,
				})
			default:
				a.queue("success", a.t("worktreeRemoved"), "")
			}
			a.reload()
			a.invalidate()
		})
	}()
}

// deleteBranchAfterWorktree deletes the branch the removed checkout left
// behind. The toast lives for ten seconds, long enough to open another
// repository in between; the branch name means something only in the
// repository it came from, so the action refuses once that repository is no
// longer the open one.
func (a *app) deleteBranchAfterWorktree(repo *gitrepo.Repository, branch gitrepo.Branch) {
	if repo == nil || a.repo == nil || a.repo.Root != repo.Root {
		return
	}
	a.perform(a.t("deleteBranchFailed"), a.t("branchDeleted"), func(ctx context.Context, repo *gitrepo.Repository) error {
		return repo.DeleteBranchChecked(ctx, branch.Name, branch.SHA, false)
	})
}

func (a *app) submoduleSection(c *ui.Context) {
	failed := a.submodulesErr != ""
	ui.Row(c).FillWidth().Height(32).Margin(0, 12, 0, 2).PaddingX(4).Gap(6).AlignItems(ui.Center).Children(func() {
		display.Text(c, a.t("submodules"), display.TextProps{Type: display.TextSecondary, Strong: true, FontSize: scaledFont(11)})
		display.Text(c, a.workspaceGroupCount(len(a.submodules), failed), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)})
	})
	if failed {
		a.failedWithNamedRetry(c, a.submodulesErr, 12, a.t("retrySubmodules"))
	}
	if len(a.submodules) == 0 {
		if !failed {
			sectionEmptyState(c, a.t("noSubmodules"), a.t("noSubmodulesHint"), 11)
		}
		return
	}
	ui.Column(c).FillWidth().Gap(2).Role(ui.RoleList).Children(func() {
		for i, module := range a.submodules {
			a.submoduleRow(c, module, i+1, len(a.submodules))
		}
	})
}

func (a *app) submoduleRow(c *ui.Context, module gitrepo.Submodule, pos, size int) {
	tk := theme.TokensOf(c)
	key := "submodule:" + module.Path
	selected := a.worktreeSel == key
	// As with the worktree rows: focusable, named, and selected for readers.
	row := ui.ButtonBase(c).Label(module.Name).Description(module.Path).
		Row().FillWidth().MinHeight(46).Padding(5, 8).Gap(10).AlignItems(ui.Center).
		Radius(tk.BorderRadius).Border(1, ui.Transparent).
		Role(ui.RoleOption).AccessSelected(selected).PositionInSet(pos, size)
	row.OnShortcut(0, ui.KeyEnter, func() { a.activateWorktreeSelection() })
	row.OnShortcut(0, ui.KeySpace, func() { a.activateWorktreeSelection() })
	switch {
	case selected:
		row.Background(selectionBg(tk, tk.BgContainer))
	case row.Hovered():
		row.Background(tk.FillTertiary)
	}
	row.Children(func() {
		ui.Box(c).Size(30, 30).Shrink(0).Radius(9).Center().Background(tk.FillTertiary).Children(func() {
			ui.Icon(c, lucide.Boxes()).FontSize(scaledFont(15)).TextColor(tk.TextSecondary)
		})
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(2).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				display.Text(c, module.Name, display.TextProps{FontSize: scaledFont(12)}).SingleLine().AccessHidden(true)
				switch {
				case module.Error != "":
					a.statusTag(c, a.t("submoduleError"), display.TagProps{
						Variant: display.TagFilled, Color: display.TagError, Size: theme.Small, Round: true,
					})
				case !module.Initialized:
					display.Tag(c, a.t("notInitialized"), display.TagProps{Variant: display.TagBorderless, Size: theme.Small})
				case module.Dirty:
					a.statusTag(c, a.t("submoduleDirty"), display.TagProps{
						Variant: display.TagFilled, Color: display.TagWarning, Size: theme.Small, Round: true,
					})
				default:
					a.statusTag(c, a.t("initialized"), display.TagProps{
						Variant: display.TagFilled, Color: display.TagSuccess, Size: theme.Small, Round: true,
					})
				}
			})
			detail := module.Path
			if module.Branch != "" {
				detail += " · " + module.Branch
			}
			if module.Error != "" {
				detail = module.Error
			}
			display.Text(c, detail, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(10)}).SingleLine().Tooltip(detail)
		})
	})
	row.OnClick(func() { a.worktreeSel = key; a.invalidate() })
	if row.DoubleClicked() && module.Initialized {
		a.openRepository(filepath.Join(a.repo.Root, module.Path), true)
	}
	a.revealWorkspaceRow(row, selected)
	a.submoduleMenu(c, row, module)
	// Opening the row's menu also selects the row, matching the other lists.
	if *ui.Local(row, "contextMenu", func() bool { return false }) && a.worktreeSel != key {
		a.worktreeSel = key
		a.invalidate()
	}
}

// workspaceKeys lists the worktree and submodule rows in render order so the
// arrow keys walk the same order the page shows.
func (a *app) workspaceKeys() []string {
	keys := make([]string, 0, len(a.worktrees)+len(a.submodules))
	for _, worktree := range a.worktrees {
		keys = append(keys, "worktree:"+worktree.Path)
	}
	for _, module := range a.submodules {
		keys = append(keys, "submodule:"+module.Path)
	}
	return keys
}

// syncWorkspaceSelection drops a selection the list no longer shows: a
// worktree removed or a repository switched elsewhere leaves a key that
// highlights no row, and Enter on it does nothing.
func (a *app) syncWorkspaceSelection() {
	if a.worktreeSel == "" {
		return
	}
	for _, key := range a.workspaceKeys() {
		if key == a.worktreeSel {
			return
		}
	}
	a.worktreeSel = ""
	a.revealSelection = false
}

func (a *app) moveWorkspaceSelection(delta int) {
	keys := a.workspaceKeys()
	if len(keys) == 0 {
		return
	}
	index := -1
	for i, key := range keys {
		if key == a.worktreeSel {
			index = i
			break
		}
	}
	if index < 0 {
		index = listStartIndex(delta, len(keys))
	} else {
		index = clampIndex(index+delta, len(keys))
	}
	if keys[index] == a.worktreeSel {
		return
	}
	a.worktreeSel = keys[index]
	a.revealSelection = true
	a.invalidate()
}

// activateWorktreeSelection switches to the selected worktree or opens the
// selected submodule repository, as a double click does.
func (a *app) activateWorktreeSelection() {
	if a.page != "worktrees" || a.worktreeSel == "" || a.repo == nil || a.busy || a.loading {
		return
	}
	if a.confirm.open || a.prompt.open || a.blameOpen || a.worktreeForm.open || a.reset.open {
		return
	}
	kind, target, ok := strings.Cut(a.worktreeSel, ":")
	if !ok {
		return
	}
	// The list may have changed since the row was chosen; a stale key would
	// otherwise switch to a worktree this page no longer shows.
	visible := false
	for _, key := range a.workspaceKeys() {
		if key == a.worktreeSel {
			visible = true
			break
		}
	}
	if !visible {
		return
	}
	switch kind {
	case "worktree":
		if filepath.Clean(target) == filepath.Clean(a.repo.Root) {
			return
		}
		a.openRepository(target, true)
	case "submodule":
		for _, module := range a.submodules {
			if module.Path != target || !module.Initialized {
				continue
			}
			a.openRepository(filepath.Join(a.repo.Root, module.Path), true)
			return
		}
	}
}

// revealWorkspaceRow scrolls the keyboard selection into view once per move;
// mouse selection never raises the flag, so the list does not jump under the
// cursor.
func (a *app) revealWorkspaceRow(row ui.Element, selected bool) {
	if !selected || !a.revealSelection {
		return
	}
	row.ScrollIntoView()
	a.revealSelection = false
}

func (a *app) submoduleMenu(c *ui.Context, row ui.Element, module gitrepo.Submodule) {
	updateLabel := a.t("updateSubmodule")
	if !module.Initialized {
		updateLabel = a.t("initializeSubmodule")
	}
	items := []overlay.MenuItem{}
	if module.Initialized {
		items = append(items,
			overlay.MenuItem{Key: "switch", Label: a.t("openSubmodule"), Icon: lucide.GitBranch(), Disabled: a.busy || a.loading},
			overlay.MenuItem{Divider: true},
		)
	}
	items = append(items,
		overlay.MenuItem{Key: "update", Label: updateLabel, Icon: lucide.RefreshCw(), Disabled: a.busy || a.loading},
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{Key: "reveal", Label: a.t("revealInFinder"), Icon: lucide.FolderOpen()},
		overlay.MenuItem{Key: "copy", Label: a.t("copyAbsolutePath"), Icon: lucide.Copy()},
	)
	switch overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: items}) {
	case "switch":
		a.openRepository(filepath.Join(a.repo.Root, module.Path), true)
	case "update":
		a.performAs(busySubmodule, a.t("updateSubmoduleFailed"), a.t("submoduleUpdated"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.UpdateSubmoduleChecked(ctx, module)
		})
	case "reveal":
		a.openPath(filepath.Join(a.repo.Root, module.Path), "reveal")
	case "copy":
		a.copyText(filepath.Join(a.repo.Root, module.Path))
	}
}

type worktreeFormState struct {
	root     string
	open     bool
	deferred bool // failed creation waits for another modal to finish
	path     string
	branch   string
	start    string
}

// openWorktreeForm asks for a destination, a new branch and the start point;
// the start defaults to the current branch so the new worktree shares HEAD.
func (a *app) discardStaleWorktreeForm() bool {
	if a.worktreeForm.root != "" && (a.repo == nil || a.repo.Root != a.worktreeForm.root) {
		a.worktreeForm = worktreeFormState{}
		return true
	}
	return false
}

func (a *app) openWorktreeForm() {
	a.discardStaleWorktreeForm()
	if a.repo == nil {
		return
	}
	if a.worktreeForm.open {
		a.queue("warning", a.t("promptPending"), "")
		a.invalidate()
		return
	}
	if a.rejectPendingDialog(toastMain) {
		return
	}
	if a.worktreeForm.deferred {
		a.worktreeFormSeq++
		a.worktreeForm.open = true
		a.worktreeForm.deferred = false
		a.invalidate()
		return
	}
	start := a.status.Branch
	if start == "" {
		start = a.status.Head
	}
	if start == "" {
		a.queue("warning", a.t("worktreeNeedsCommit"), "")
		a.invalidate()
		return
	}
	path := filepath.Join(filepath.Dir(a.repo.Root), filepath.Base(a.repo.Root)+"-worktree")
	a.worktreeFormSeq++
	a.worktreeForm = worktreeFormState{root: a.repo.Root, open: true, path: path, start: start}
	a.invalidate()
}

// chooseWorktreeParent picks the folder that will hold the new checkout. The
// worktree directory itself must not exist yet, so the picker chooses a parent
// and keeps the name already typed in the form.
func (a *app) chooseWorktreeParent() {
	if a.discardStaleWorktreeForm() || a.window == nil || a.repo == nil || !a.worktreeForm.open {
		return
	}
	root, seq := a.repo.Root, a.worktreeFormSeq
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:    a.window,
		Title:     a.t("chooseParentDir"),
		Directory: true,
	})
	if a.repo == nil || a.repo.Root != root || a.worktreeFormSeq != seq || !a.worktreeForm.open {
		return
	}
	if err != nil {
		a.queue("error", a.t("chooseParentDirFailed"), err.Error())
		a.invalidate()
		return
	}
	if len(paths) == 0 {
		return
	}
	a.applyWorktreeParent(root, seq, paths[0])
}

func (a *app) applyWorktreeParent(root string, seq uint64, parent string) {
	if a.repo == nil || a.repo.Root != root || a.worktreeFormSeq != seq || !a.worktreeForm.open {
		return
	}
	a.worktreeForm.path = worktreePathFromParent(parent, a.worktreeForm.path, a.repo.Root)
	a.invalidate()
}

// worktreePathFromParent keeps the typed worktree name and puts it under the
// chosen parent, falling back to the default name when the field is empty.
func worktreePathFromParent(parent, current, repoRoot string) string {
	name := filepath.Base(filepath.Clean(current))
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = filepath.Base(filepath.Clean(repoRoot)) + "-worktree"
	}
	return filepath.Join(parent, name)
}

func (a *app) worktreeFormDialog(c *ui.Context) {
	a.worktreeFormDialogFor(c, toastMain)
}

func (a *app) worktreeFormDialogFor(c *ui.Context, owner toastOwner) {
	if owner != toastMain {
		return
	}
	a.discardStaleWorktreeForm()
	if !a.worktreeForm.open {
		return
	}
	tk := theme.TokensOf(c)
	confirmed := false
	workbenchModal(c, &a.worktreeForm.open, func() {
		ui.Column(c).Width(480).Padding(18).Gap(10).
			Background(tk.BgElevated).Radius(12).Border(1, tk.Border).
			Children(func() {
				display.Text(c, a.t("newWorktreeTitle"), display.TextProps{Strong: true, FontSize: scaledFont(14)})
				var pathInput ui.Element
				ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
					pathInput = field.Input(c, &a.worktreeForm.path, field.InputProps{
						Variant: field.InputOutlined, Placeholder: a.t("worktreePath"), Size: theme.Middle,
					}).Grow(1).Basis(0)
					if workbenchButton(c, a.t("chooseParentDir"), buttonNormal, false).Clicked() {
						a.chooseWorktreeParent()
					}
				})
				// The path and the start point come prefilled; the branch is the
				// only required field left to type, so the form opens on it.
				// AutoFocus once: Focus every frame would take Tab back from the
				// start field and the buttons.
				branchInput := field.Input(c, &a.worktreeForm.branch, field.InputProps{
					Variant: field.InputOutlined, Placeholder: a.t("worktreeBranch"), Size: theme.Middle,
				}).AutoFocus()
				startInput := field.Input(c, &a.worktreeForm.start, field.InputProps{
					Variant: field.InputOutlined, Placeholder: a.t("worktreeStart"), Size: theme.Middle,
				})
				display.Text(c, a.t("worktreeFormHint"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11), Rows: 2})
				ready := strings.TrimSpace(a.worktreeForm.path) != "" &&
					strings.TrimSpace(a.worktreeForm.branch) != "" &&
					strings.TrimSpace(a.worktreeForm.start) != ""
				// Enter in any field creates the worktree once the form is
				// complete, as Enter submits the single-field prompts.
				if ready && (pathInput.Submitted() || branchInput.Submitted() || startInput.Submitted()) {
					confirmed = true
					a.worktreeForm.open = false
				}
				ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Justify(ui.End).Children(func() {
					if workbenchButton(c, a.t("cancel"), buttonNormal, false).Clicked() {
						a.worktreeForm.open = false
					}
					button := workbenchButton(c, a.t("create"), buttonPrimary, !ready)
					if ready && button.Clicked() {
						confirmed = true
						a.worktreeForm.open = false
					}
				})
			})
	})
	if confirmed {
		a.submitWorktreeForm()
		return
	}
	if !a.worktreeForm.open {
		a.worktreeForm = worktreeFormState{}
	}
}

// submitWorktreeForm starts the creation and keeps the typed values until it
// succeeds. Closing the form and clearing it first meant one git error (a
// taken branch, a missing parent) threw away the whole form; the reader had
// to retype it to try again.
func (a *app) submitWorktreeForm() {
	if a.discardStaleWorktreeForm() || a.repo == nil {
		return
	}
	root, seq := a.repo.Root, a.worktreeFormSeq
	form := a.worktreeForm
	form.root = root
	form.open = false
	path := strings.TrimSpace(form.path)
	branch := strings.TrimSpace(form.branch)
	start := strings.TrimSpace(form.start)
	a.worktreeForm = worktreeFormState{}
	restore := func(err error) {
		if err == nil || a.repo == nil || a.repo.Root != root || a.worktreeFormSeq != seq {
			return
		}
		form.deferred = a.hasPendingDialog(toastMain)
		form.open = !form.deferred
		a.worktreeForm = form
	}
	started := a.performDoneAs(busyCreateWorktree, a.t("createWorktreeFailed"), a.t("worktreeCreated"), restore,
		func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.AddWorktree(ctx, path, branch, start)
		})
	if !started && a.repo != nil {
		// Another operation holds the repository, so the form has to come
		// back rather than vanish with the reader's input.
		restore(context.Canceled)
		a.invalidate()
	}
}
