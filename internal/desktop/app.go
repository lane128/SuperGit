package desktop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"supergit/internal/appicon"
	"supergit/internal/gitrepo"
)

type snapshot struct {
	status         gitrepo.Status
	history        []gitrepo.Commit
	historyTips    []string
	historyMore    bool
	historyErr     error
	branches       []gitrepo.Branch
	tags           []gitrepo.Tag
	tagsErr        error
	worktrees      []gitrepo.Worktree
	modules        []gitrepo.Submodule
	remotes        []gitrepo.Remote
	remoteBranches []gitrepo.Branch
	stashes        []gitrepo.Stash
	integration    gitrepo.Operation
	warnings       []string
	changeStamp    string
}
type app struct {
	window                                               *mygo.Window
	repo                                                 *gitrepo.Repository
	ctx                                                  context.Context
	cancel                                               context.CancelFunc
	guard                                                jobGuard
	data                                                 snapshot
	historyRevision                                      uint64
	historyView                                          historyCache
	page                                                 string
	loading, opening, diffLoading, commitLoading         bool
	operation, message, notice                           string
	logs                                                 []string
	drafts                                               map[string]string
	parents                                              []string
	path                                                 string
	staged                                               bool
	diff, commitDiff                                     gitrepo.Diff
	diffLoaded                                           bool
	diffErr                                              error
	selectedLines                                        map[int]bool
	splitDiff                                            bool
	historyDiffList                                      ui.ListState
	tagInspection                                        *tagInspection
	tagQuery                                             string
	tagList                                              ui.ListState
	tagListReset                                         bool
	selectedSHA                                          string
	commitText                                           commitMessageState
	commitTextCancel                                     context.CancelFunc
	commitFiles                                          []gitrepo.ChangedFile
	commitFilesLoaded, commitDiffLoaded                  bool
	commitFilesErr, commitDiffErr                        error
	commitPath                                           string
	commitFileRevision                                   uint64
	commitFileView                                       *commitFilePresentation
	commitFileCursor                                     commitFileCursor
	commitFileList                                       ui.ListState
	focusCommitFiles, historyDiffReset                   bool
	workspaceDiffReset, historyListReset                 bool
	historyIndex                                         int
	historyList, diffList                                ui.ListState
	sidebarWidth, fileWidth, historyWidth, detailsHeight float32
	dark                                                 bool
	fontSize                                             float32
	form                                                 actionDraft
	confirmation                                         *actionTarget
	returnPage                                           string
	writeCancel                                          context.CancelFunc
	cancellable                                          bool
	prefs                                                preferences
	prefsPath                                            string
	historyLimit                                         int
	historyPageLoading                                   bool
	historyPageCancel                                    context.CancelFunc
	historyQuery                                         string
	focusHistorySearch, focusHistoryList                 bool
	historyCursor                                        historyCursor
	commitCancel                                         context.CancelFunc
	workspaceRevision                                    uint64
	workspaceView                                        *workspacePresentation
	workspaceCursor                                      workspaceCursor
	workspaceList                                        ui.ListState
	focusWorkspaceList                                   bool
	workspaceSelectionReset                              bool
	diffCancel                                           context.CancelFunc
	stashSHA                                             string
	stashDiff                                            gitrepo.Diff
	stashLoading, stashLoaded, stashListReset            bool
	stashErr                                             error
	stashCancel                                          context.CancelFunc
	stashList                                            ui.ListState
	conflict                                             *gitrepo.Conflict
	conflictText, conflictError                          string
	conflictLoading                                      bool
	watch                                                repositoryWatch
	focused, closed                                      bool
	watchCancel                                          context.CancelFunc
	// dockIcon remembers the drawing the Dock shows, so the view can ask for
	// one every frame and the system is only touched when the appearance
	// changes.
	dockIcon *appicon.Switcher
}

var desktopConfigDir = os.UserConfigDir

func Run(initialPath string) error {
	a := &app{page: "workspace", sidebarWidth: 176, fileWidth: 325, historyWidth: 530, detailsHeight: 340, fontSize: 13, drafts: make(map[string]string), historyIndex: -1, dockIcon: appicon.New(func(png []byte) error { return mygo.App.Dock.SetIcon(png) })}
	a.historyLimit = 300
	if config, err := desktopConfigDir(); err == nil {
		a.prefsPath = filepath.Join(config, "SuperGit", "preferences.json")
		p, err := loadPreferences(a.prefsPath)
		a.prefs, a.dark, a.fontSize = p, p.Dark, p.FontSize
		if err != nil {
			a.record("读取偏好失败", err)
		}
	}
	a.ctx, a.cancel = context.WithCancel(context.Background())
	defer func() {
		a.cancel()
		if a.watchCancel != nil {
			a.watchCancel()
		}
	}()
	mygo.App.WhenReady(func() {
		a.window = mygo.NewWindow(mygo.WindowOptions{Title: "SuperGit", Width: 1440, Height: 920, MinWidth: 1040, MinHeight: 680, Content: ui.View(a.view)})
		a.startWatching()
		if initialPath != "" {
			a.open(initialPath, false)
		}
	})
	return mygo.App.Run()
}
func (a *app) invalidate() {
	if a.window != nil {
		a.window.Invalidate()
	}
}
func (a *app) record(label string, err error) {
	text := label
	if err != nil {
		text += "：" + err.Error()
	}
	a.notice = text
	a.logs = append([]string{time.Now().Format("15:04:05") + "  " + text}, a.logs...)
	if len(a.logs) > 40 {
		a.logs = a.logs[:40]
	}
}
func (a *app) choose() {
	if a.guard.writing || a.opening {
		return
	}
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.window, Title: "打开 Git 工作区", Directory: true})
	if err != nil {
		a.record("打开目录失败", err)
	} else if len(paths) > 0 {
		a.parents = nil
		a.open(paths[0], false)
	}
}
func (a *app) open(path string, enter bool) {
	if a.guard.writing {
		return
	}
	a.clearLineSelection()
	a.cancelHistoryPage()
	a.cancelCommitMessageRead()
	a.cancelStashPreview()
	a.guard.Switch()
	a.watch = repositoryWatch{}
	a.cancel()
	a.ctx, a.cancel = context.WithCancel(context.Background())
	ctx := a.ctx
	t := a.guard.Begin("open")
	a.opening = true
	a.loading = false
	a.diffLoading = false
	a.commitLoading = false
	go func() {
		r, err := gitrepo.Open(ctx, path)
		mygo.RunOnMain(func() {
			if !a.guard.Accept(t) {
				return
			}
			a.opening = false
			if err != nil {
				a.record("打开失败", err)
				a.invalidate()
				return
			}
			if a.repo != nil {
				a.drafts[a.repo.Root] = a.message
				if enter {
					a.parents = append(a.parents, a.repo.Root)
				}
			}
			a.repo = r
			a.confirmation = nil
			a.form = actionDraft{}
			a.conflict = nil
			a.conflictText, a.conflictError = "", ""
			a.stashSHA, a.stashDiff = "", gitrepo.Diff{}
			a.stashLoaded, a.stashErr = false, nil
			a.stashListReset = true
			a.historyLimit, a.historyQuery = 300, ""
			a.focusHistorySearch, a.focusHistoryList = false, false
			a.historyListReset = true
			a.historyCursor = historyCursor{}
			a.commitCancel = nil
			a.workspaceView = nil
			a.workspaceCursor = workspaceCursor{}
			a.focusWorkspaceList = false
			a.workspaceSelectionReset = true
			a.diffCancel = nil
			a.prefs.remember(r.Root)
			a.persist()
			a.message = a.drafts[r.Root]
			a.data = snapshot{}
			a.historyRevision++
			a.historyView = historyCache{}
			a.path = ""
			a.selectedSHA = ""
			a.commitText = commitMessageState{}
			a.commitFilesLoaded, a.commitDiffLoaded = false, false
			a.commitFilesErr, a.commitDiffErr = nil, nil
			a.commitPath = ""
			a.replaceCommitFiles(nil)
			a.focusCommitFiles = false
			a.historyDiffReset = true
			a.diff = gitrepo.Diff{}
			a.diffLoaded, a.diffErr = false, nil
			a.commitDiff = gitrepo.Diff{}
			a.historyIndex = -1
			a.tagInspection = nil
			a.tagQuery = ""
			a.tagListReset = true
			a.page = "workspace"
			a.record("已打开 "+r.Root, nil)
			a.refresh()
			a.invalidate()
		})
	}()
}
func (a *app) refresh() { a.refreshSnapshot(false) }
func (a *app) refreshSnapshot(automatic bool) {
	if a.repo == nil || a.loading || a.opening || a.guard.writing {
		return
	}
	a.cancelHistoryPage()
	if !automatic {
		a.clearLineSelection()
	}
	r, ctx := a.repo, a.ctx
	limit := a.historyLimit
	if limit < 1 {
		limit = 300
	}
	t := a.guard.Begin("snapshot")
	a.loading = true
	go func() {
		s, err := readSnapshot(ctx, r, limit)
		mygo.RunOnMain(func() {
			if !a.guard.Accept(t) {
				return
			}
			a.loading = false
			if err != nil {
				a.record("刷新失败", err)
			} else {
				s = a.installSnapshot(s)
				a.workspaceRevision++
				if s.changeStamp != "" {
					a.watch.stamp = s.changeStamp
				}
				if len(s.warnings) > 0 {
					a.record("部分数据读取失败", fmt.Errorf("%s", strings.Join(s.warnings, "；")))
				}
				a.historyRevision++
				a.historyIndex = -1
				for i, commit := range s.history {
					if commit.SHA == a.selectedSHA {
						a.historyIndex = i
						break
					}
				}
				a.reconcileFileSelection()
				if a.path != "" {
					a.reloadDiff(a.path, a.staged, automatic)
				}
				if a.selectedSHA == "" && len(s.history) > 0 {
					a.selectCommit(0)
				}
			}
			a.invalidate()
		})
	}()
}
func (a *app) loadDiff(path string, staged bool) { a.reloadDiff(path, staged, false) }
func (a *app) reloadDiff(path string, staged, preserve bool) {
	req, ctx, ok := a.beginWorkspaceDiff(path, staged, preserve)
	if !ok {
		return
	}
	r, cancel := a.repo, a.diffCancel
	go func() {
		defer cancel()
		d, err := r.Diff(ctx, req.path, req.staged)
		mygo.RunOnMain(func() {
			if a.finishWorkspaceDiff(req, d, err) {
				a.invalidate()
			}
		})
	}()
}
func (a *app) mutate(label string, fn func(context.Context, *gitrepo.Repository) error, committing bool) {
	a.mutateTask(label, fn, committing, false)
}
func (a *app) mutateTask(label string, fn func(context.Context, *gitrepo.Repository) error, committing, cancellable bool) {
	if a.repo == nil || a.opening || a.loading || !a.guard.StartWrite() {
		return
	}
	a.cancelHistoryPage()
	r := a.repo
	a.operation = label
	a.guard.Begin("watch")
	a.guard.Begin("snapshot")
	a.guard.Begin("diff")
	a.diffLoading = false
	a.guard.Begin("commit")
	a.cancelStashPreview()
	a.guard.Begin("conflict")
	a.commitLoading, a.stashLoading, a.conflictLoading = false, false, false
	ctx, cancel := context.WithCancel(context.Background())
	a.writeCancel, a.cancellable = cancel, cancellable
	go func() {
		err := fn(ctx, r)
		cancel()
		mygo.RunOnMain(func() {
			a.guard.EndWrite()
			a.writeCancel, a.cancellable = nil, false
			a.operation = ""
			a.record(label, err)
			if err == nil && committing {
				a.message = ""
				a.drafts[r.Root] = ""
			}
			a.refresh()
			a.invalidate()
		})
	}()
}
func (a *app) commit() {
	if a.canCommit() {
		msg := a.message
		a.mutate("提交", func(ctx context.Context, r *gitrepo.Repository) error { return r.Commit(ctx, msg) }, true)
	}
}
func (a *app) canCommit() bool {
	if a.repo == nil || a.page != "workspace" || strings.TrimSpace(a.message) == "" || a.guard.writing || a.loading || a.opening {
		return false
	}
	staged := false
	for _, e := range a.data.status.Entries {
		if e.Conflict {
			return false
		}
		staged = staged || e.Staged()
	}
	return staged
}
func (a *app) view(c *ui.Context) {
	c.SetTheme(appTheme(a.dark))
	a.syncDockIcon(c.Theme().Dark)
	t := c.Theme()
	a.keyboard(c)
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).FillWidth().Height(58).Shrink(0).Padding(10, 16).Gap(12).AlignItems(ui.Center).Border(1, t.Border).Children(func() {
			ui.Text(c, "SuperGit").FontSize(19).Bold()
			if a.repo != nil {
				ui.Text(c, filepath.Base(a.repo.Root)).Bold().Tooltip(a.repo.Root)
				branch := a.data.status.Branch
				if branch == "" {
					branch = "(detached)"
				}
				ui.Text(c, branch).TextColor(t.TextMuted)
				if a.data.status.Upstream != "" {
					ui.Textf(c, "↑%d  ↓%d", a.data.status.Ahead, a.data.status.Behind).TextColor(t.TextMuted)
				}
			}
			ui.Box(c).Grow(1)
			if len(a.parents) > 0 {
				if ui.Button(c, "返回父仓库").Disabled(a.guard.writing || a.opening).Clicked() {
					path := a.parents[len(a.parents)-1]
					a.parents = a.parents[:len(a.parents)-1]
					a.open(path, false)
				}
			}
			if ui.Button(c, "克隆／新建…").Disabled(a.busy()).Clicked() {
				a.page = "repositories"
			}
			if ui.Button(c, "打开仓库…").Disabled(a.guard.writing || a.opening).KeyShortcuts(ui.Cmd, ui.KeyO).Tooltip("打开仓库 (⌘O)").Clicked() {
				a.choose()
			}
			if ui.Button(c, "刷新 ⌘R").Disabled(a.repo == nil || a.loading || a.opening || a.guard.writing).KeyShortcuts(ui.Cmd, ui.KeyR).Clicked() {
				a.refresh()
			}
		})
		if a.repo == nil {
			ui.Column(c).Grow(1).FillWidth().Children(func() {
				if a.page == "action" {
					a.actionView(c)
				} else {
					a.repositoryEntry(c)
				}
			})
		} else {
			ui.Split(c, &a.sidebarWidth, func() { a.navigation(c) }, func() {
				ui.Column(c).Fill().Clip().Children(func() {
					a.operationBar(c)
					// The optional operation bar must not change the page subtree's
					// identity, focus or editor state when it appears/disappears.
					c.Key("page-" + a.page)
					switch a.page {
					case "action":
						a.actionView(c)
					case "repositories":
						a.repositoryEntry(c)
					case "remotes":
						a.remotes(c)
					case "stash":
						a.stashes(c)
					case "conflicts":
						a.conflicts(c)
					case "workspace":
						a.workspace(c)
					case "history":
						a.history(c)
					case "modules":
						a.modules(c)
					case "branches":
						a.branches(c)
					case "worktrees":
						a.worktrees(c)
					case "logs":
						a.logView(c)
					}
				})
			}).Grow(1).FillWidth().SplitReset(176)
		}
		ui.Row(c).FillWidth().Height(32).Shrink(0).Padding(5, 16).Gap(12).Background(t.Surface).Children(func() {
			state := "就绪"
			if a.opening {
				state = "正在打开仓库…"
			} else if a.guard.writing {
				state = a.operation + "…"
			} else if a.loading {
				state = "读取仓库…"
			}
			if a.guard.writing && a.cancellable && a.writeCancel != nil {
				if ui.Button(c, "取消任务").FontSize(11).Clicked() {
					a.writeCancel()
					a.cancellable = false
					a.notice = "已请求取消，正在等待 Git 退出并刷新实际状态"
				}
			}
			ui.Text(c, state).FontSize(12).Bold()
			ui.Text(c, oneLine(a.notice)).FontSize(12).Grow(1).TextColor(t.TextMuted).Tooltip(a.notice)
			if ui.Button(c, "操作记录").FontSize(11).Clicked() {
				a.page = "logs"
			}
		})
	})
}
func (a *app) navigation(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(10).Gap(14).Background(t.Surface).Children(func() {
		ui.Sidebar(c, &a.page, func() {
			for index, item := range pageShortcuts {
				label := item.Label
				if item.Page == "modules" {
					label = fmt.Sprintf("子模块  %d", len(a.data.modules))
				} else if item.Page == "stash" {
					label = fmt.Sprintf("Stash  %d", len(a.data.stashes))
				}
				ui.SidebarItem(c, item.Page, nil, label).Tooltip(fmt.Sprintf("%s (⌘%d)", item.Label, index+1)).KeyShortcuts(ui.Cmd, item.Key)
			}
		}).Grow(1).FillWidth()
		ui.Column(c).Gap(8).FillWidth().Children(func() {
			label := "切换深色"
			if a.dark {
				label = "切换浅色"
			}
			if ui.Button(c, label).FillWidth().Clicked() {
				a.dark = !a.dark
				a.persist()
			}
			ui.Row(c).Gap(6).Children(func() {
				ui.Text(c, "代码字号").FontSize(12).Grow(1)
				if ui.Button(c, "−").Disabled(a.fontSize <= 11).Clicked() {
					a.fontSize--
					a.persist()
				}
				if ui.Button(c, "+").Disabled(a.fontSize >= 19).Clicked() {
					a.fontSize++
					a.persist()
				}
			})
			ui.Text(c, "MyGo 原生预览版").FontSize(11).TextColor(t.TextMuted)
		})
	})
}
func oneLine(s string) string {
	return strings.NewReplacer("\n", " ↵ ", "\r", "", "\t", "  ").Replace(s)
}
func short(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	if s == "" {
		return "—"
	}
	return s
}
