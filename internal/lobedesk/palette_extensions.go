package lobedesk

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/egoist/mygo"
	"supergit/internal/gitrepo"
)

func init() { registerPaletteActions(extensionPaletteActions()...) }

func extensionPaletteActions() []paletteAction {
	result := []paletteAction{
		{ID: "add_submodule", English: "Add Submodule…", Chinese: "添加子模块…", Unavailable: extensionIdle, Run: (*app).addPaletteSubmodule},
		{ID: "open_submodule", English: "Open Submodule…", Chinese: "打开子模块…", Unavailable: func(a *app) string {
			for _, m := range a.submodules {
				if m.Initialized && m.Error == "" {
					return ""
				}
			}
			return a.paletteText("No initialized submodules", "没有已初始化的子模块")
		}, Run: (*app).openPaletteSubmodule},
		{ID: "apply_patch", English: "Apply Patch…", Chinese: "应用补丁…", Unavailable: extensionIdle, Run: (*app).openPalettePatch},
		{ID: "lfs_edit_pattern", English: "LFS: Edit Tracked Pattern…", Chinese: "LFS：编辑跟踪模式…", Unavailable: extensionIdle, Run: (*app).editPaletteLFSPattern},
	}
	for _, unlock := range []bool{false, true} {
		id, en, zh := "lfs_lock_file", "LFS: Lock File…", "LFS：锁定文件…"
		if unlock {
			id, en, zh = "lfs_unlock_file", "LFS: Unlock File…", "LFS：解锁文件…"
		}
		result = append(result, paletteAction{ID: id, English: en, Chinese: zh, Unavailable: func(a *app) string {
			if !gitrepo.HasGitLFS() {
				return a.paletteText("Install Git LFS and reopen SuperGit", "请安装 Git LFS 后重新打开 SuperGit")
			}
			if len(a.remotes) == 0 {
				return a.t("noRemote")
			}
			return extensionIdle(a)
		}, Run: func(a *app) { a.choosePaletteLFSRemote(a.paletteText(en, zh), unlock) }})
	}
	return append(result, flowPaletteActions()...)
}

func extensionIdle(a *app) string {
	if a.operation.Kind != "" {
		return a.t("quickCheckoutOperation")
	}
	if a.loadErr != "" {
		return a.t("quickCheckoutReadError")
	}
	return ""
}

// Reads for a parameter picker may include a lock server round trip. Keep the
// app cancellable and deliver the result only to the session that requested it.
func readExtension[T any](a *app, title string, load func(context.Context, *gitrepo.Repository) (T, error), done func(T)) {
	if a.repo == nil || a.busy || a.loading {
		return
	}
	repo, epoch := a.repo, a.loadSeq
	owner, generation := a.commandPalette, a.commandPaletteGeneration
	ctx, cancel := a.beginOperation("")
	go func() {
		defer cancel()
		result, err := load(ctx, repo)
		cancelled := ctx.Err() == context.Canceled
		a.hopToMain(func() {
			a.finishOperation()
			defer a.invalidate()
			if cancelled || repo != a.repo || epoch != a.loadSeq || generation != a.commandPaletteGeneration || a.localWatch.closed || a.repositoryTabs != nil && a.repositoryTabs.active != a {
				return
			}
			if err != nil {
				a.queue("error", title, a.operationErrorDescription(err))
				return
			}
			if !a.repositoryChooserOpen && !a.repositoryOpenPending() && !a.hasPendingDialog(toastMain) {
				// The dispatching step releases ownership as soon as it returns.
				// Restore its command and Backspace parent for this continuation.
				a.commandPalette = owner
				done(result)
				if !a.commandPalette.open {
					a.commandPalette = commandPaletteState{}
				}
			}
		})
	}()
}

func (a *app) runPaletteExtension(title string, fn func(context.Context, *gitrepo.Repository) error) {
	a.perform(title, title+a.paletteText(" completed", "已完成"), fn)
}

func (a *app) addPaletteSubmodule() {
	title := a.paletteText("Add Submodule", "添加子模块")
	status := a.status
	a.inputPalette(title, a.paletteText("Repository URL", "仓库地址"), "", func(address string) {
		a.inputPalette(title, a.paletteText("Path in this repository", "仓库内的路径"), cloneSourceName(address), func(path string) {
			a.inputPalette(title, a.paletteText("Submodule name", "子模块名称"), path, func(name string) {
				a.runPaletteExtension(title, func(ctx context.Context, r *gitrepo.Repository) error {
					return r.AddPaletteSubmodule(ctx, address, name, path, status)
				})
			})
		})
	})
}

func (a *app) openPaletteSubmodule() {
	title := a.paletteText("Open Submodule", "打开子模块")
	var rows []paletteChoice
	for _, m := range a.submodules {
		if m.Initialized && m.Error == "" {
			rows = append(rows, paletteChoice{id: m.Path, label: m.Path, detail: m.Name, sha: m.Index})
		}
	}
	a.choosePalette(title, a.paletteText("Choose a submodule", "选择子模块"), rows, func(row paletteChoice) {
		for _, m := range a.submodules {
			if m.Path == row.id && m.Index == row.sha {
				readExtension(a, title, func(ctx context.Context, r *gitrepo.Repository) (string, error) {
					return r.OpenPaletteSubmodule(ctx, m)
				}, func(path string) { a.openRepository(path, true) })
				return
			}
		}
	})
}

func (a *app) openPalettePatch() {
	if a.repo == nil || a.repositoryChooserOpen || a.window == nil && a.openDirectoryDialog == nil {
		return
	}
	title := a.paletteText("Apply Patch", "应用补丁")
	repo, epoch, status := a.repo, a.loadSeq, a.status
	open := a.openDirectoryDialog
	if open == nil {
		open = mygo.Dialog.Open
	}
	a.repositoryChooserOpen = true
	paths, err := open(mygo.OpenDialogOptions{Parent: a.window, Title: title, DefaultPath: repo.Root, ButtonLabel: title})
	a.repositoryChooserOpen = false
	defer a.invalidate()
	if a.repo != repo || a.loadSeq != epoch || a.busy || a.loading || a.hasPendingDialog(toastMain) {
		return
	}
	if err != nil {
		a.queue("error", title, err.Error())
		return
	}
	if len(paths) != 1 || !filepath.IsAbs(paths[0]) {
		return
	}
	a.runPaletteExtension(title, func(ctx context.Context, r *gitrepo.Repository) error { return r.ApplyPatchFile(ctx, paths[0], status) })
}

func (a *app) editPaletteLFSPattern() {
	title := a.paletteText("LFS: Edit Tracked Pattern", "LFS：编辑跟踪模式")
	status := a.status
	readExtension(a, title, func(ctx context.Context, r *gitrepo.Repository) ([]gitrepo.LFSPattern, error) {
		return r.LFSPatterns(ctx)
	}, func(patterns []gitrepo.LFSPattern) {
		var rows []paletteChoice
		for i, p := range patterns {
			label := p.Pattern
			if label == "" {
				label = a.paletteText("Add pattern…", "添加模式…")
			}
			rows = append(rows, paletteChoice{id: strconv.Itoa(i), label: label, detail: p.File})
		}
		a.choosePalette(title, a.paletteText("Path or pattern", "路径或模式"), rows, func(row paletteChoice) {
			i, err := strconv.Atoi(row.id)
			if err != nil || i < 0 || i >= len(patterns) {
				return
			}
			p := patterns[i]
			update := func(next string) {
				a.runPaletteExtension(title, func(ctx context.Context, r *gitrepo.Repository) error { return r.EditLFSPattern(ctx, p, next, status) })
			}
			edit := func() {
				a.inputPalette(title, a.paletteText("New path or pattern", "新的路径或模式"), p.Pattern, update)
			}
			if p.Pattern == "" {
				edit()
				return
			}
			a.choosePalette(title, p.Pattern, []paletteChoice{
				{id: "edit", label: a.paletteText("Edit pattern…", "编辑模式…")}, {id: "delete", label: a.paletteText("Delete pattern", "删除模式")},
			}, func(choice paletteChoice) {
				if choice.id == "edit" {
					edit()
					return
				}
				a.askConfirm(title, p.File+"\n"+p.Pattern, a.t("delete"), true, func() { update("") })
			})
		})
	})
}

func (a *app) choosePaletteLFSRemote(title string, unlock bool) {
	remotes := slices.Clone(a.remotes)
	start := func(remote gitrepo.Remote) {
		status := a.status
		readExtension(a, title, func(ctx context.Context, r *gitrepo.Repository) (gitrepo.LFSLockList, error) {
			return r.LFSLockFiles(ctx, remote, unlock)
		}, func(list gitrepo.LFSLockList) {
			var rows []paletteChoice
			for i, f := range list.Files {
				rows = append(rows, paletteChoice{id: strconv.Itoa(i), label: f.Path, detail: remote.Name})
			}
			if len(rows) == 0 {
				a.queue("info", title, a.paletteText("No matching files", "没有符合条件的文件"))
				return
			}
			a.choosePalette(title, a.paletteText("Choose a file", "选择文件"), rows, func(row paletteChoice) {
				i, err := strconv.Atoi(row.id)
				if err != nil || i < 0 || i >= len(list.Files) {
					return
				}
				file := list.Files[i]
				a.askConfirm(title, fmt.Sprintf("%s · %s", remote.Name, file.Path), title, false, func() {
					a.runPaletteExtension(title, func(ctx context.Context, r *gitrepo.Repository) error {
						return r.ChangeLFSLock(ctx, remote, list, file, status)
					})
				})
			})
		})
	}
	if len(remotes) == 1 {
		start(remotes[0])
		return
	}
	var rows []paletteChoice
	for _, r := range remotes {
		rows = append(rows, paletteChoice{id: r.Name, label: r.Name})
	}
	a.choosePalette(title, a.t("remoteName"), rows, func(row paletteChoice) {
		for _, r := range remotes {
			if r.Name == row.id {
				start(r)
				return
			}
		}
	})
}
