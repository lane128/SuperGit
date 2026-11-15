package desktop

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"path/filepath"
	"strings"
	"supergit/internal/gitrepo"
)

func (a *app) busy() bool { return a.guard.writing || a.opening || a.loading }
func (a *app) startForm(d actionDraft) {
	if a.busy() {
		return
	}
	a.returnPage = a.page
	a.form = d
	a.confirmation = nil
	a.page = "action"
}
func (a *app) ask(d actionDraft) {
	if a.busy() {
		return
	}
	a.returnPage = a.page
	target := a.capture(d)
	a.confirmation = &target
	a.page = "action"
}
func actionField(c *ui.Context, label string, value *string) {
	ui.Column(c.Key(label)).FillWidth().Gap(5).Children(func() {
		ui.Text(c, label).Bold().FontSize(12)
		ui.TextInput(c, value).Label(label).FillWidth()
	})
}
func (a *app) actionView(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().Padding(24).Gap(16).Children(func() {
			if a.confirmation != nil {
				target := *a.confirmation
				d := target.Draft
				ui.Text(c, "确认"+string(d.Kind)).FontSize(22).Bold()
				if target.Root != "" {
					ui.Text(c, "仓库："+oneLine(target.Root)).Selectable().TextColor(t.TextMuted)
				}
				ui.Text(c, d.description()).Selectable().FillWidth().Padding(16).Border(1, t.Border)
				if d.RemoteCaptured {
					address := d.Remote.URL
					if d.Kind == actionPush {
						address = d.Remote.PushURL
					}
					ui.Text(c, "已确认地址："+displayURL(address)).Selectable()
				}
				ui.Row(c).Gap(12).Children(func() {
					if ui.Button(c, "返回编辑").Disabled(a.busy() || !d.editable()).Clicked() {
						a.form = d
						a.confirmation = nil
					}
					if ui.Button(c, "取消").Disabled(a.busy()).Tooltip("取消 (Esc)").KeyShortcuts(0, ui.KeyEscape).Clicked() {
						a.cancelAction()
					}
					if ui.PrimaryButton(c, "确认执行").Disabled(a.busy() || !target.valid(a.repo, &a.guard)).Clicked() {
						a.executeTarget(target)
					}
				})
				if !target.valid(a.repo, &a.guard) && !a.guard.writing {
					ui.Text(c, "仓库已变化，请取消并重新选择目标。").TextColor(t.Danger)
				}
				return
			}
			d := &a.form
			ui.Text(c, string(d.Kind)).FontSize(22).Bold()
			switch d.Kind {
			case actionInit:
				actionField(c, "新目录路径", &d.First)
				a.destinationPicker(c, &d.First)
			case actionClone:
				actionField(c, "仓库地址", &d.First)
				actionField(c, "新目录路径", &d.Second)
				a.destinationPicker(c, &d.Second)
			case actionCreateTag:
				actionField(c, "标签名称", &d.First)
				ui.Text(c, "目标提交 SHA："+d.Second).Selectable().FillWidth()
				ui.Text(c, "创建本地轻量标签；已有同名引用时停止。").FontSize(12).TextColor(t.TextMuted)
			case actionCreateBranch:
				actionField(c, "分支名称", &d.First)
				actionField(c, "起始提交／引用", &d.Second)
				ui.Checkbox(c, &d.Flag, "创建后切换")
			case actionRename:
				actionField(c, "原分支", &d.First)
				actionField(c, "新分支名称", &d.Second)
			case actionTrack:
				actionField(c, "本地分支名称", &d.First)
				actionField(c, "远端分支（例如 origin/main）", &d.Second)
			case actionAddRemote:
				actionField(c, "远端名称", &d.First)
				actionField(c, "远端地址", &d.Second)
			case actionPush:
				actionField(c, "远端名称", &d.First)
				actionField(c, "本地源分支", &d.Second)
				actionField(c, "远端目标分支", &d.Third)
				ui.Checkbox(c, &d.Flag, "设置此分支的 upstream")
			case actionSaveStash:
				actionField(c, "Stash 说明", &d.First)
				ui.Checkbox(c, &d.Flag, "包含未跟踪文件")
			case actionAddWorktree:
				actionField(c, "新目录路径", &d.First)
				a.destinationPicker(c, &d.First)
				actionField(c, "新分支名称", &d.Second)
				actionField(c, "起始提交／引用", &d.Third)
			default:
				actionField(c, "目标", &d.First)
			}
			ui.Text(c, "先检查目标，再确认执行。仓库认证使用系统 Git 凭据助手或 SSH agent。").FontSize(12).TextColor(t.TextMuted)
			ui.Row(c).Gap(12).Children(func() {
				if ui.Button(c, "取消").Disabled(a.busy()).Tooltip("取消 (Esc)").KeyShortcuts(0, ui.KeyEscape).Clicked() {
					a.cancelAction()
				}
				if ui.PrimaryButton(c, "检查操作").Disabled(a.busy() || strings.TrimSpace(d.First) == "").Clicked() {
					target := a.capture(*d)
					a.confirmation = &target
				}
			})
		})
	})
}
func (a *app) destinationPicker(c *ui.Context, path *string) {
	if ui.Button(c, "选择父目录…").Clicked() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.window, Title: "选择新仓库／工作区的父目录", Directory: true})
		if err != nil {
			a.record("选择目录失败", err)
		} else if len(paths) > 0 {
			*path = filepath.Join(paths[0], "new-repository")
		}
	}
}
func (a *app) executeTarget(target actionTarget) {
	if a.busy() || !target.valid(a.repo, &a.guard) {
		return
	}
	d := target.Draft
	if d.Kind == actionInit || d.Kind == actionClone {
		a.createRepository(d)
		return
	}
	a.confirmation = nil
	a.page = a.returnPage
	a.mutateTask(string(d.Kind), d.run, false, d.network())
}
func (a *app) createRepository(d actionDraft) {
	if a.busy() || !a.guard.StartWrite() {
		return
	}
	a.operation = string(d.Kind)
	ctx, cancel := context.WithCancel(context.Background())
	a.writeCancel, a.cancellable = cancel, d.network()
	go func() {
		var r *gitrepo.Repository
		var err error
		if d.Kind == actionClone {
			r, err = gitrepo.Clone(ctx, d.First, d.Second)
		} else {
			r, err = gitrepo.Init(ctx, d.First)
		}
		cancel()
		mygo.RunOnMain(func() {
			a.guard.EndWrite()
			a.writeCancel = nil
			a.cancellable = false
			a.operation = ""
			a.record(string(d.Kind), err)
			if err == nil {
				a.confirmation = nil
				a.parents = nil
				a.open(r.Root, false)
			}
			a.invalidate()
		})
	}()
}
func (a *app) operationBar(c *ui.Context) {
	op := a.data.integration
	if op.Kind == "" {
		return
	}
	ui.Row(c).FillWidth().Padding(10, 16).Gap(12).Background(c.Theme().Accent.Alpha(.10).Over(c.Theme().Background)).AlignItems(ui.Center).Children(func() {
		ui.Text(c, fmt.Sprintf("进行中：%s · %s", op.Kind, short(op.Head))).Bold().Grow(1)
		if ui.Button(c, "查看冲突").Clicked() {
			a.page = "conflicts"
		}
		if ui.Button(c, "继续操作").Disabled(a.busy() || op.Kind == "am").Clicked() {
			a.ask(actionDraft{Kind: actionContinue, First: op.Kind, Expected: op})
		}
		if ui.Button(c, "中止操作").Disabled(a.busy() || op.Kind == "am").Clicked() {
			a.ask(actionDraft{Kind: actionAbort, First: op.Kind, Expected: op})
		}
	})
}
