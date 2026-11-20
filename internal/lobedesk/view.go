package lobedesk

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

const (
	titleBarHeight = 38
	// sidebarWidth holds an icon and a page name in both languages: "Worktrees"
	// is the longest, and a name that wrapped or clipped would defeat the point
	// of naming the pages at all.
	sidebarWidth = 184
)

func (a *app) root(c *ui.Context) {
	a.renderWorkbench(c, nil)
}

func (a *app) renderWorkbench(c *ui.Context, repositoryStrip func()) {
	a.frameSeq++
	a.mainToastBottomInset = 0
	a.applyTheme(c)
	a.registerShortcuts(c)
	tk := theme.TokensOf(c)
	// The Dock follows the window: in system mode this is the appearance
	// macOS resolved for the window, in light and dark mode the chosen one.
	a.syncDockIcon(tk.Dark)
	// Dialogs retain their initiating window independently of keyboard focus.
	// Focus still determines which window drains its notification queue.
	settingsFocused := a.settingsFocused()
	page := ui.Column(c).Fill().Background(tk.BgLayout)
	page.Children(func() {
		if repositoryStrip != nil {
			repositoryStrip()
			if a.repo != nil {
				a.workbenchToolbar(c, false)
			}
		} else {
			a.titleBar(c)
		}
		c := c.Key(a)
		content := ui.Row(c).Grow(1).Basis(0).MinHeight(0).AlignItems(ui.Stretch)
		a.sidebarSplit.rowWidth, _ = c.Size()
		content.Children(func() {
			if a.sidebarHidden {
				a.contentCardAtWidth(c, a.sidebarSplit.rowWidth)
				return
			}
			sidebar := a.sidebar(c)
			def := shareForWidth(a.sidebarSplit.rowWidth-splitterWidth, sidebarWidth)
			a.splitHandle(c, &a.sidebarSplit, def, sidebarPaneMin, a.workbenchContentMin(), a.t("resizeSidebarPane"))
			a.contentCardAtWidth(c, a.sidebarSplit.rowWidth-sidebar-splitterWidth)
		})
		// Dialogs come before the toast host so a toast, such as an undo
		// action, stays above an open dialog and can be clicked.
		a.confirmDialog(c)
		a.noticeDetailsDialog(c, toastMain)
		a.promptDialog(c)
		a.resetDialog(c)
		a.checkoutBranchDialog(c)
		a.commandPickerDialogs(c)
		a.childNavigationDialog(c)
		a.worktreeFormDialog(c)
		a.addRemoteDialog(c)
		a.cloneRepositoryDialog(c)
		a.newRepositoryDialog(c)
		a.blameDialog(c)
		a.snapshotFileDialog(c)
		a.toastHost(c, toastMain)
		if !settingsFocused {
			// A settings notice whose panel closed is not lost: the main
			// window takes it over rather than leaving it queued forever.
			a.adoptOrphanedNotice()
			a.flushNotice(c, toastMain)
		}
	})
}

// applyTheme gives a window the theme the settings choose. Each window sets
// its own context, so the settings window and the main window agree whether
// the theme is fixed or follows the system.
func (a *app) applyTheme(c *ui.Context) {
	switch a.themeMode {
	case "light":
		c.SetTheme(themeConfig(a.palette).Theme(false))
	case "dark":
		c.SetTheme(themeConfig(a.palette).Theme(true))
	default:
		themeConfig(a.palette).Apply(c)
	}
}

func (a *app) titleBar(c *ui.Context) {
	a.workbenchToolbar(c, true)
}

func (a *app) workbenchToolbar(c *ui.Context, nativeChrome bool) {
	tk := theme.TokensOf(c)
	height := float32(titleBarHeight)
	inset := float32(0)
	if nativeChrome {
		bar := c.TitleBar()
		if bar.Height > 0 {
			height = bar.Height
		}
		if bar.Left > 0 {
			inset = max(bar.Left+8, 80) - 12
		}
	}
	toolbar := ui.Row(c).FillWidth().Height(height).Shrink(0).AlignItems(ui.Center).Gap(8).Padding(0, 12, 0, 12)
	if nativeChrome {
		toolbar.DragWindow()
	}
	toolbar.Children(func() {
		showBranch := a.repo != nil && !a.restoredCheckout
		progress := a.progressText()
		// Equal space on each side keeps the branch centered as status changes.
		ui.Row(c).Grow(1).Basis(0).MinWidth(0).AlignItems(ui.Center).Children(func() {
			if inset > 0 {
				ui.Box(c).Width(inset).Shrink(0)
			}
			ui.Row(c).Grow(1).Basis(0).MinWidth(0).AlignItems(ui.Center).Gap(6).Children(func() {
				// A merge, rebase or cherry-pick can outlive the command that
				// started it: a cancelled rebase leaves the repository in the
				// middle of it, and the page the reader was on says nothing.
				// The title bar is on every page, so the state is never hidden.
				if showBranch && a.operation.Kind != "" {
					// Changes keeps continue, skip and abort beside the files.
					guidance := "operationWhere"
					if a.operation.Kind == "merge" {
						guidance = "operationWhereMerge"
					}
					op := workbenchButton(c, a.tf("operation", a.operation.Kind), buttonNormal, false).
						MinWidth(0).Shrink(1).TextColor(tk.Warning).Tooltip(a.t(guidance))
					if op.Clicked() {
						a.openPage("changes")
						a.selectWorktree()
						a.invalidate()
					}
				}
				// Background work can outlast the moment a toast would be read,
				// so the bar says what it is doing while it runs instead of
				// leaving a disabled button as the only sign. Fetching names
				// itself; every other long operation shares one general line.
				if a.loading && showBranch && !a.loadSince.IsZero() {
					if remaining := progressDelay - time.Since(a.loadSince); remaining > 0 {
						c.After(remaining)
					}
				}
				if progress != "" {
					icon := ui.Icon(c, lucide.LoaderCircle()).FontSize(scaledFont(12)).TextColor(tk.TextSecondary)
					icon.Rotate(icon.Loop("spin", 900*time.Millisecond, ui.Linear) * 360)
					display.Text(c, progress, display.TextProps{
						Type: display.TextSecondary, FontSize: scaledFont(11),
					}).SingleLine().MinWidth(0).MaxWidth(110).Shrink(1).Tooltip(progress)
					// A git command can hang on a network that stopped answering;
					// without this the only way out is the three-minute timeout.
					if a.busy || a.fetching {
						if cancel := workbenchIconButton(c, lucide.X(), a.t("cancel"), buttonNormal, false); cancel.Clicked() {
							a.cancelOperation()
						}
					}
				}
			})
		})
		a.toolbarCommandControl(c, showBranch)
		ui.Row(c).Grow(1).Basis(0).MinWidth(0).AlignItems(ui.Center).Justify(ui.End).Gap(6).Children(func() {
			if a.repo != nil {
				a.stashSaveMenu(c)
				a.syncMenu(c)
			}
		})
	})
}

// Keep one silhouette and two keyboard-accessible targets. The branch and its
// tracking status stay centered in the whole control, including the icon.
const toolbarCommandWidth float32 = 360

func (a *app) toolbarCommandControl(c *ui.Context, showBranch bool) {
	ui.Row(c).Width(toolbarCommandWidth).Shrink(0).AlignItems(ui.Center).Children(func() {
		label := a.t("commandPalette")
		keys := shortcutKeys("palette.open")
		button := styleWorkbenchButton(c, ui.ButtonBase(c), buttonNormal, !a.canOpenCommandPalette(), false).
			Label(label).Tooltip(label + " (" + keys + ")")
		iconWidth := scaledFont(32)
		if showBranch {
			button.Width(iconWidth).Shrink(0).PaddingX(0).Radius(6, 0, 0, 6)
		} else {
			button.FillWidth()
		}
		button.Children(func() {
			ui.Icon(c, lucide.SquareTerminal()).FontSize(scaledFont(14))
			if !showBranch {
				ui.Text(c, a.t("toolbarCommands")).SingleLine()
				ui.Text(c, keys).FontSize(scaledFont(11)).TextColor(theme.TokensOf(c).TextSecondary)
			}
		})
		if button.Clicked() {
			button.Focus()
			a.openCommandPalette()
		}
		if showBranch {
			a.branchSelector(c, iconWidth)
		}
	})
}

// progressDelay is how long an operation must run before the title bar says
// anything. Staging a file or toggling a checkbox finishes in milliseconds,
// and a spinner that appears and vanishes within one glance reads as a
// glitch rather than as progress.
const progressDelay = 200 * time.Millisecond

// progressVisible reports whether the progress line has waited out its delay.
// A zero start time means the operation was not timed (tests, or a hand-built
// app), which shows immediately.
func progressVisible(since, now time.Time) bool {
	return since.IsZero() || now.Sub(since) >= progressDelay
}

// progressText is what the title bar says about work in flight. Fetching names
// the fetch because it is the only operation a reader starts and then walks
// away from; a named operation says what it is doing, which is the difference
// between "Working…" and "Pushing…" when a wait stretches out. Everything else
// busy shares the general line. Repository reads use their own delay, after
// operations, and idle shows nothing.
func (a *app) progressText() string {
	if (a.busy || a.fetching) && !progressVisible(a.progressSince, time.Now()) {
		return ""
	}
	if a.fetching {
		return a.t("fetchingNow")
	}
	if a.busy {
		if a.busyKey != "" {
			return a.t(a.busyKey)
		}
		return a.t("busyNow")
	}
	if a.loading && a.repo != nil && !a.restoredCheckout && progressVisible(a.loadSince, time.Now()) {
		return a.t("loadingRepo")
	}
	return ""
}

// repoMenu gives the repository name in the sidebar a menu for actions on
// the checkout as a whole, rather than on one file or commit.
func (a *app) repoMenu(c *ui.Context, target ui.Element) {
	root := a.repo.Root
	items := []overlay.MenuItem{
		{Key: "copy", Label: a.t("copyRepoPath"), Icon: lucide.Copy()},
		{Divider: true},
		{Key: "reveal", Label: a.t("revealInFinder"), Icon: lucide.FolderOpen()},
		{Key: "terminal", Label: a.t("openInTerminal"), Icon: lucide.Terminal()},
		{Divider: true},
		{Key: "new", Label: a.t("newRepo"), Disabled: !a.canShowCloneRepository()},
		{Key: "clone", Label: a.t("cloneRepo"), Disabled: !a.canShowCloneRepository()},
	}
	hasRecent := false
	for _, path := range a.recentRepositories {
		if path == root {
			continue
		}
		if !hasRecent {
			items = append(items, overlay.MenuItem{Divider: true}, overlay.MenuItem{Label: a.t("sessionRepositories"), Disabled: true})
			hasRecent = true
		}
		items = append(items, overlay.MenuItem{Key: "recent:" + path, Label: path, Disabled: !a.canOpenRecentRepository()})
	}
	chosen := repositoryContextMenu(c, target, root, items)
	switch chosen {
	case "copy":
		a.copyText(root)
	case "reveal":
		a.openPath(root, "reveal")
	case "terminal":
		a.openPath(root, "terminal")
	case "new":
		a.showNewRepository()
	case "clone":
		a.showCloneRepository()
	default:
		for _, path := range a.recentRepositories {
			if chosen == "recent:"+path {
				a.openRecentRepository(path)
				break
			}
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// sidebar names every place in the app. An icon-only rail asks the reader to
// learn five glyphs, and two of them (the workspace's compare icon and the
// branch icon) read alike at 19px; the Git clients this is compared with name
// their places in a sidebar, and the extra width buys the first-glance
// understanding the rail could not.
//
// Below History come the repository's own locations: its
// branches, remote branches, tags, stashes, worktrees and submodules, each a
// folded list under a counted heading. Picking a ref locates its commit in
// the history list and selects it, the way a branch tree in a Git client
// jumps to the branch's tip, without changing which commits History lists.
func (a *app) sidebar(c *ui.Context) float32 {
	tk := a.sidebarTokens(c)
	width := a.workbenchSidebarWidth()
	ui.Column(c).Width(width).Shrink(0).AlignSelf(ui.Stretch).Padding(6, 6).
		Background(tk.BgContainer).
		Children(func() {
			a.sidebarHeader(c)
			if a.repo == nil && a.repositoryTabs != nil {
				ui.Box(c).Grow(1)
			} else {
				ui.Column(c).FillWidth().Gap(1).Margin(0, 2).Children(func() {
					a.sidebarItem(c, lucide.History(), "history", a.t("history"), shortcutKeys("page.history"))
				})
				layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.sidebarScroll}, func() {
					// Keep counts inside the viewport and clear of the 12-DIP overlay scrollbar.
					ui.Column(c).FillWidth().Gap(1).Padding(0, 12, 0, 4).Children(func() {
						a.sidebarRefGroups(c)
					})
				}).Grow(1).Basis(0).MinHeight(0)
			}
			a.sidebarItem(c, lucide.Settings(), "settings", a.t("settings"), shortcutKeys("window.settings"))
		})
	return width
}

func (a *app) sidebarRefGroups(c *ui.Context) {
	a.sidebarGroup(c, "branches", a.t("branches"), a.visibleBranchCount(a.branches, "refs/heads/"), func() {
		a.sidebarBranchItems(c)
	})
	remoteHeading := a.sidebarGroup(c, "remotes", a.t("remotes"), a.visibleBranchCount(a.remoteBranches, "refs/remotes/"), func() {
		if a.repo != nil {
			row := a.sidebarRefRow(c.Key("add-remote"), lucide.Plus(), a.t("addRemoteEntry"), false, false, a.openAddRemote)
			row.Disabled(!a.canAddRemote())
		}
		for _, remote := range a.remoteBranches {
			if a.branchHidden("refs/remotes/" + remote.Name) {
				continue
			}
			ref := historyRef{Kind: historyRefRemote, Name: remote.Name}
			a.sidebarRefRow(c, lucide.Cloud(), remote.Name,
				a.historyRef == ref, false, func() { a.locateRef(ref.Kind, ref.Name) })
		}
	})
	if a.repo != nil {
		items := []overlay.MenuItem{{Key: "add", Label: a.t("addRemoteEntry"), Icon: lucide.Plus(), Disabled: !a.canAddRemote()}}
		if overlay.ContextMenu(c, remoteHeading, overlay.DropdownMenuProps{Items: items}) == "add" {
			a.openAddRemote()
		}
	}
	a.sidebarGroup(c, "tags", a.t("tags"), len(a.tags), func() {
		for _, tag := range a.tags {
			ref := historyRef{Kind: historyRefTag, Name: tag.Name}
			a.sidebarRefRow(c, lucide.Tag(), tag.Name,
				a.historyRef == ref, false, func() { a.locateRef(ref.Kind, ref.Name) })
		}
	})
	a.sidebarGroup(c, "stashes", a.t("stashes"), len(a.stashes), func() {
		for _, stash := range a.stashes {
			label := stash.Message
			if label == "" {
				label = stash.Ref
			}
			a.sidebarRefRow(c, lucide.Archive(), label, a.selectedStash == stash.SHA, false, func() {
				a.openPage("stash")
				// The click names one stash, so its row scrolls into view and
				// takes the selection the details describe.
				a.revealSelection = true
				a.selectStash(stash.SHA)
				a.invalidate()
			})
		}
	})
	a.sidebarGroup(c, "worktrees", a.t("worktrees"), len(a.worktrees), func() {
		for _, worktree := range a.worktrees {
			key := "worktree:" + worktree.Path
			// The row is named by its folder, but a worktree is identified by
			// its path: two checkouts can share a folder name in different
			// parents, and the page already marks the one this window has open.
			current := a.repo != nil && filepath.Clean(worktree.Path) == filepath.Clean(a.repo.Root)
			row := a.sidebarRefRow(c, lucide.FolderTree(), filepath.Base(worktree.Path),
				a.worktreeSel == key, current, func() {
					a.openSidebarWorkspace(key)
				})
			row.Tooltip(worktree.Path)
			if current {
				row.Description(a.t("current") + " · " + worktree.Path)
			}
		}
	})
	a.sidebarGroup(c, "submodules", a.t("submodules"), len(a.submodules), func() {
		for _, module := range a.submodules {
			key := "submodule:" + module.Path
			state := ""
			if !module.Initialized {
				state = a.t("notInitialized")
			}
			row := a.sidebarRefRowWithState(c, lucide.Boxes(), module.Name, state, false, false, func() {
				a.openSidebarWorkspace(key)
			})
			description := module.Path + "\n" + a.t("submoduleMenuHint")
			if state != "" {
				description = state + " · " + description
			}
			row.Description(description).Tooltip(description)
			a.submoduleMenu(c, row, module)
		}
	})
}

// openSidebarWorkspace enters the history of a checkout listed by this repository.
func (a *app) openSidebarWorkspace(key string) {
	if a.repo == nil || a.busy || a.loading || a.confirm.open || a.prompt.open || a.blameOpen || a.worktreeForm.open || a.reset.open {
		return
	}
	var path string
	for _, worktree := range a.worktrees {
		if key == "worktree:"+worktree.Path {
			path = worktree.Path
			break
		}
	}
	for _, module := range a.submodules {
		if key == "submodule:"+module.Path && module.Initialized {
			path = filepath.Join(a.repo.Root, module.Path)
			break
		}
	}
	if path == "" {
		return
	}
	a.openPage("history")
	if filepath.Clean(path) != filepath.Clean(a.repo.Root) {
		a.openRepository(path, true)
	}
}

// sidebarGroup is a counted heading and, while it is open, the rows under it.
// The whole heading folds the list: a 28-high row with only a 20-wide chevron
// as the target is easy to miss, and the reader is aiming at the group, not at
// the arrow. The rows below are the way into the page; the heading no longer
// navigates, so clicking its text cannot surprise with a page switch.
func (a *app) sidebarGroup(c *ui.Context, key, title string, count int, items func()) ui.Element {
	tk := a.sidebarTokens(c)
	countInk := tertiaryTextInk(tk, tk.FillTertiary.Over(tk.BgContainer))
	open := a.sidebarGroupOpen(key)
	chevron := lucide.ChevronRight()
	if open {
		chevron = lucide.ChevronDown()
	}
	row := ui.ButtonBase(c).Label(a.t(title)).Row().FillWidth().Height(28).Padding(0, 2).Gap(2).
		AlignItems(ui.Center).Radius(tk.BorderRadius).
		Role(ui.RoleDisclosure).Expanded(open)
	if row.Hovered() {
		row.Background(tk.FillTertiary)
	}
	row.Children(func() {
		// The arrow is decoration now; the row itself is the control.
		ui.Icon(c, chevron).FontSize(scaledFont(12)).TextColor(tk.TextSecondary).Shrink(0)
		display.Text(c, a.t(title), display.TextProps{
			Type: display.TextSecondary, Strong: true, FontSize: scaledFont(11),
		}).TextColor(tk.TextSecondary).SingleLine().Grow(1).Basis(0).MinWidth(0).AccessHidden(true)
		if count > 0 {
			display.Text(c, fmt.Sprint(count), display.TextProps{FontSize: scaledFont(10)}).
				TextColor(countInk).Shrink(0)
		}
	})
	row.Tooltip(a.tf("foldGroup", a.t(title)))
	if row.Clicked() {
		a.toggleSidebarGroup(key)
	}
	if open {
		ui.Column(c).FillWidth().Gap(1).Padding(0, 0, 0, 22).Children(items)
	}
	return row
}

// sidebarRefRow is one ref in a group: its name, and the check that shows the
// ref selector's filter when it names this ref. Clicking locates the ref's
// commit in History instead of changing that filter. current marks the
// checked-out branch with its own glyph, so a long name that clips still says
// which branch is live.
func (a *app) sidebarRefRow(c *ui.Context, icon *ui.SVG, label string, selected, current bool, onClick func()) ui.Element {
	return a.sidebarRefRowWithState(c, icon, label, "", selected, current, onClick)
}

func (a *app) sidebarRefRowWithState(c *ui.Context, icon *ui.SVG, label, state string, selected, current bool, onClick func()) ui.Element {
	tk := a.sidebarTokens(c)
	height := float32(24)
	if state != "" {
		height = max(40, scaledFont(36))
	}
	row := ui.ButtonBase(c).Label(label).Row().FillWidth().Height(height).Padding(0, 6).Gap(6).
		AlignItems(ui.Center).Radius(tk.BorderRadius).AccessSelected(selected)
	// Selection has its own surface, distinct from the sidebar and hover.
	if selected {
		row.Background(a.sidebarSelectionBg(tk))
	} else if row.Hovered() {
		row.Background(tk.FillTertiary)
	}
	if len([]rune(label)) > 22 {
		// Only a clipped name needs the tooltip; a short one repeats itself
		// and covers the next row.
		row.Tooltip(label)
	}
	if current {
		row.Description(a.t("current"))
	}
	color := tk.TextSecondary
	if selected || current {
		color = tk.Text
	}
	row.Children(func() {
		ui.Icon(c, icon).FontSize(scaledFont(10)).TextColor(tk.TextTertiary)
		if state == "" {
			display.Text(c, label, display.TextProps{FontSize: scaledFont(11), Strong: selected || current}).
				TextColor(color).SingleLine().Grow(1).Basis(0).MinWidth(0).AccessHidden(true)
		} else {
			// State gets its own line so a narrow sidebar keeps the module name.
			ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
				display.Text(c, label, display.TextProps{FontSize: scaledFont(11), Strong: selected || current}).
					TextColor(color).SingleLine().FillWidth().MinWidth(0).AccessHidden(true)
				display.Text(c, state, display.TextProps{FontSize: scaledFont(10)}).
					TextColor(tk.TextSecondary).SingleLine().FillWidth().MinWidth(0).AccessHidden(true)
			})
		}
		if current {
			ui.Icon(c, lucide.Check()).FontSize(scaledFont(11)).TextColor(tk.Green).Shrink(0).AccessHidden(true)
		}
	})
	if row.Clicked() {
		onClick()
	}
	return row
}

// sidebarBranchItems lists the local branches, with the checked-out one marked:
// the list is how the sidebar answers "where am I" without another page.
func (a *app) sidebarBranchItems(c *ui.Context) {
	for _, branch := range a.branches {
		if a.branchHidden("refs/heads/" + branch.Name) {
			continue
		}
		ref := historyRef{Kind: historyRefLocal, Name: branch.Name}
		row := a.sidebarRefRow(c, lucide.GitBranch(), branch.Name, a.historyRef == ref, branch.Current, func() {
			a.locateRef(ref.Kind, ref.Name)
		})
		a.branchMenu(c, row, branch)
	}
}

func (a *app) sidebarGroupOpen(key string) bool {
	if open, ok := a.sidebarOpen[key]; ok {
		return open
	}
	return key == "branches"
}

func (a *app) toggleSidebarGroup(key string) {
	if a.sidebarOpen == nil {
		a.sidebarOpen = map[string]bool{}
	}
	// Read the default first: the first click on an untouched group folds the
	// open default rather than opening what is already open.
	a.sidebarOpen[key] = !a.sidebarGroupOpen(key)
	a.invalidate()
}

// sidebarHeader names the open repository and is its switcher. The Welcome
// window has no repository, so it gets no header: a workspace name above pages
// that cannot open reads as a checkout that is not there.
func (a *app) sidebarHeader(c *ui.Context) {
	if a.repo == nil {
		return
	}
	tk := a.sidebarTokens(c)
	name := a.repoName()
	ui.Row(c).FillWidth().Padding(2, 6, 2, 8).Gap(8).AlignItems(ui.Center).Children(func() {
		display.Avatar(c, name, display.AvatarProps{
			Size:     24,
			Shape:    display.AvatarSquare,
			Fallback: display.AvatarText,
			// Avatar computes its ink from RGB, so use an opaque surface.
			Background: tk.BgElevated,
		})
		target := ui.ButtonBase(c).Label(name).Description(a.repo.Root).Row().Grow(1).Basis(0).MinWidth(0).Height(26).Gap(4).AlignItems(ui.Center).Radius(tk.BorderRadiusSM)
		if target.Hovered() {
			target.Background(tk.FillTertiary)
		}
		target.Children(func() {
			display.Text(c, name, display.TextProps{FontSize: scaledFont(12), Strong: true}).TextColor(tk.Text).SingleLine().Grow(1).Basis(0).MinWidth(0)
			ui.Icon(c, lucide.ChevronDown()).FontSize(12).TextColor(tk.TextSecondary)
		})
		a.repoMenu(c, target)
	})
}

// sidebarItem is one page in the sidebar: its icon, its name, and the key that
// opens it. The name is the row's label, so a screen reader reads the same word
// the row shows.
func (a *app) sidebarItem(c *ui.Context, icon *ui.SVG, page, title, keys string) {
	tk := a.sidebarTokens(c)
	active := a.page == page
	disabled := a.repo == nil && page != "settings"
	row := ui.ButtonBase(c).Label(title).Row().FillWidth().Height(32).Padding(0, 8).Gap(8).
		AlignItems(ui.Center).Radius(tk.BorderRadius).Disabled(disabled)
	if !disabled {
		switch {
		case active:
			row.Background(a.sidebarSelectionBg(tk))
		case row.Hovered():
			row.Background(tk.FillTertiary)
		}
	}
	color := tk.TextSecondary
	switch {
	case disabled:
		color = tk.TextDisabled
	case active:
		color = tk.Text
	}
	// The key hint is tertiary text, and the checked-out page's tint is the
	// darkest surface it can land on; darken it per row so it stays readable.
	keyGround := tk.BgContainer
	if active {
		keyGround = a.sidebarSelectionBg(tk)
	} else if row.Hovered() {
		keyGround = tk.FillTertiary.Over(tk.BgContainer)
	}
	keyInk := tertiaryTextInk(tk, keyGround)
	row.Children(func() {
		ui.Icon(c, icon).FontSize(scaledFont(15)).TextColor(color)
		display.Text(c, title, display.TextProps{FontSize: scaledFont(12), Strong: active}).
			TextColor(color).SingleLine().Grow(1).Basis(0).MinWidth(0).AccessHidden(true)
		display.Text(c, keys, display.TextProps{FontSize: scaledFont(10)}).
			TextColor(keyInk).Shrink(0).AccessHidden(true)
	})
	switch page {
	case "history":
		row.KeyShortcuts(ui.Cmd, ui.Key1)
	case "branches":
		row.KeyShortcuts(ui.Cmd, ui.Key2)
	case "stash":
		row.KeyShortcuts(ui.Cmd, ui.Key3)
	case "worktrees":
		row.KeyShortcuts(ui.Cmd, ui.Key4)
	case "settings":
		row.KeyShortcuts(ui.Cmd, ui.KeyComma)
	}
	if row.Clicked() && !disabled {
		a.openPage(page)
	}
}

// openPage moves the window to a place. History is the window: its list holds
// the working tree's row and the stash entries, so "changes" and "stash" are
// selections in that list rather than pages of their own.
func (a *app) openPage(page string) {
	// The settings panel is a window of its own, opened from every entry
	// point that used to switch the main window to the settings page.
	if page == "settings" {
		a.openSettingsWindow()
		return
	}
	if a.repo == nil {
		return
	}
	wantWorktree, wantStash := false, false
	switch page {
	case "changes":
		page, wantWorktree = "history", true
	case "stash":
		page, wantStash = "history", true
	}
	if page == a.page && !wantWorktree && !wantStash {
		return
	}
	if page != a.page {
		if page == "history" {
			a.restoreHistorySelection()
		} else if a.page == "history" {
			a.saveDetailScroll()
		}
	}
	a.page = page
	if page != "history" {
		// The ref picker belongs to the history header. Leaving the page
		// unmounts it without closing it, and its open flag holds the arrows
		// of whichever page comes next: close it, and start it empty when the
		// reader comes back.
		a.historyRefOpen = false
		a.historyRefQuery = ""
	}
	if wantWorktree {
		a.revealSelection = true
		a.openChanges()
	}
	if wantStash && len(a.stashes) > 0 && a.stashesErr == "" {
		a.revealSelection = true
		if a.selectedStash == "" {
			a.selectStash(a.stashes[0].SHA)
		}
	}
	a.invalidate()
}

func (a *app) contentCardAtWidth(c *ui.Context, width float32) {
	ui.Column(c).Grow(1).Basis(0).MinWidth(0).
		Background(theme.TokensOf(c).BgContainer).
		Children(func() {
			if a.repo != nil && a.loadErr != "" {
				a.reloadFailedBanner(c)
			}
			if a.tabSnapshotUnavailable() {
				ui.Column(c).Grow(1).Basis(0).MinHeight(0).MinWidth(0).Disabled(true).Children(func() {
					a.workbenchPage(c, width)
				})
				return
			}
			a.workbenchPage(c, width)
		})
}

func (a *app) workbenchPage(c *ui.Context, width float32) {
	switch {
	case (a.repo == nil || a.restoredCheckout) && a.loading:
		a.loadingPage(c)
	case a.repo == nil:
		a.emptyPage(c)
	case a.page == "history":
		a.historyPageAtWidth(c, width)
	case a.page == "branches":
		a.branchesPage(c)
	case a.page == "worktrees":
		a.worktreesPage(c)
	default:
		// History is the main window: its first row is the working
		// tree, so the live changes and the commit graph share one
		// page the way Sublime Merge's window does.
		a.historyPageAtWidth(c, width)
	}
}

func (a *app) loadingPage(c *ui.Context) {
	tk := theme.TokensOf(c)
	ui.Column(c).Fill().Center().Gap(10).Children(func() {
		icon := ui.Icon(c, lucide.LoaderCircle()).FontSize(scaledFont(26)).TextColor(tk.TextTertiary)
		icon.Rotate(icon.Loop("spin", 900*time.Millisecond, ui.Linear) * 360)
		display.Text(c, a.t("loadingRepo"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(13)})
	})
}

func (a *app) emptyPage(c *ui.Context) {
	tk := theme.TokensOf(c)
	zone := ui.Column(c).Fill().Center().Gap(12).Padding(32)
	blocked := a.repositoryChooserBlocked()
	if !blocked {
		if zone.FileDragOver() {
			zone.Draw(func(p *ui.Painter, r ui.Rect) {
				const inset = 8
				p.Stroke(ui.Rect{X: r.X + inset, Y: r.Y + inset, W: r.W - 2*inset, H: r.H - 2*inset}, tk.Primary, 8, 2)
			})
		}
		if paths := zone.DroppedFiles(); len(paths) > 0 {
			if len(paths) == 1 && filepath.IsAbs(paths[0]) {
				a.openRepository(paths[0], true)
			} else {
				a.loadErr = a.t("dropSingleRepository")
			}
		}
	}
	zone.Children(func() {
		ui.Icon(c, lucide.FolderGit2()).FontSize(scaledFont(30)).TextColor(tk.TextTertiary)
		display.Text(c, a.t("openRepo"), display.TextProps{Heading: 5, Strong: true, FontSize: scaledFont(tk.FontSize)})
		if a.loadErr == "" {
			display.Text(c, a.t("openRepoHint"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(13), Rows: 3}).
				MaxWidth(420).TextAlign(ui.Center)
		} else {
			display.Text(c, a.t("openFailed"), display.TextProps{Type: display.TextDanger, FontSize: scaledFont(12)}).
				MaxWidth(520).TextAlign(ui.Center)
			display.Text(c, a.loadErr, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11), Rows: 3}).
				MaxWidth(520).TextAlign(ui.Center)
		}
		choose := workbenchButton(c, a.t("chooseFolder"), buttonPrimary, blocked)
		if choose.Clicked() {
			a.chooseRepository()
		}
		if workbenchButton(c, a.t("newRepo"), buttonNormal, !a.canShowCloneRepository()).Clicked() {
			a.showNewRepository()
		}
		if workbenchButton(c, a.t("cloneRepo"), buttonNormal, !a.canShowCloneRepository()).Clicked() {
			a.showCloneRepository()
		}
		a.welcomeRecentRepositories(c)
	})
}

// reloadFailedBanner keeps a failed refresh from looking like a successful
// one: the pages keep drawing the last read, so without this banner the old
// commits and status still look live and act on the old repository.
func (a *app) reloadFailedBanner(c *ui.Context) {
	tk := theme.TokensOf(c)
	ui.Row(c).FillWidth().Shrink(0).Padding(10, 14).Gap(10).AlignItems(ui.Center).
		Background(tk.FillSecondary).
		BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
		ui.Icon(c, lucide.AlertTriangle()).FontSize(scaledFont(14)).TextColor(tk.Red)
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(2).Children(func() {
			display.Text(c, a.t("reloadFailed"), display.TextProps{Type: display.TextDanger, Strong: true, FontSize: scaledFont(12)})
			display.Text(c, a.tf("reloadStaleHint", a.loadErr), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11), Rows: 2})
		})
		blocked := a.loading || a.busy
		label := a.t("retry")
		if a.loading {
			label = a.t("retrying")
		}
		button := workbenchButton(c, label, buttonNormal, blocked)
		if !blocked && button.Clicked() {
			a.reload()
		}
	})
}

// failedWithRetry shows a list's read failure with the way to try again. The
// pages that read their own lists used to leave recovery to the title bar's
// refresh, which the error never mentioned.
func (a *app) failedWithRetry(c *ui.Context, message string, padding float32) {
	a.failedRead(c, message, padding, a.t("retry"))
}

// failedWithNamedRetry names the button after the group it rereads. Two failed
// groups on one page would otherwise show two buttons called "Retry", with no
// way to tell which read each one repeats.
func (a *app) failedWithNamedRetry(c *ui.Context, message string, padding float32, label string) {
	a.failedRead(c, message, padding, label)
}

// failedRead draws a failed read's message and, when it owns one, its retry.
// A reload banner above the page already carries the one Retry for the whole
// page, so a list error under it shows only the reason: two buttons doing the
// same full reread read as two different actions.
func (a *app) failedRead(c *ui.Context, message string, padding float32, retryLabel string) {
	// A reload is already out: the button would either do nothing (busy) or
	// start a second full read (loading), and the reader would see no change.
	// Only a read can say "retrying"; a fetch or a stage is a different job.
	retrying := a.loading || a.busy
	ui.Column(c).Padding(padding).Gap(6).AlignSelf(ui.Start).Children(func() {
		display.Text(c, message, display.TextProps{Type: display.TextDanger, FontSize: scaledFont(11), Rows: 3})
		if a.loadErr != "" {
			return
		}
		label := retryLabel
		if a.loading {
			label = a.t("retrying")
		}
		button := workbenchButton(c, label, buttonNormal, retrying)
		if !retrying && button.Clicked() {
			a.reload()
		}
	})
}

func (a *app) pageHeader(c *ui.Context, title, subtitle string, right func()) {
	tk := theme.TokensOf(c)
	ui.Row(c).Height(46).Shrink(0).Padding(0, 14).Gap(10).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
		ui.Row(c).Grow(1).Basis(0).MinWidth(0).AlignItems(ui.Center).Gap(10).Children(func() {
			// Heading 5 otherwise resolves to the theme's base size, which is
			// outside scaledFont and would stay put while the rest of the page
			// follows the reader's text-size preset.
			display.Text(c, title, display.TextProps{Heading: 5, Strong: true, FontSize: scaledFont(tk.FontSize)}).Shrink(0)
			if subtitle != "" {
				display.Text(c, subtitle, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12)}).SingleLine().Grow(1).Basis(0).MinWidth(0)
			}
		})
		if right != nil {
			ui.Row(c).Shrink(0).AlignItems(ui.Center).Gap(6).Children(right)
		}
	})
}
