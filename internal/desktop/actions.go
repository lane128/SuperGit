package desktop

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"supergit/internal/gitrepo"
)

type actionKind string

const (
	actionInit            actionKind = "初始化仓库"
	actionClone           actionKind = "克隆仓库"
	actionCreateBranch    actionKind = "创建分支"
	actionCreateTag       actionKind = "创建轻量标签"
	actionSwitch          actionKind = "切换分支"
	actionRename          actionKind = "重命名分支"
	actionDeleteBranch    actionKind = "删除分支"
	actionTrack           actionKind = "跟踪远端分支"
	actionAddRemote       actionKind = "添加远端"
	actionFetch           actionKind = "Fetch"
	actionPull            actionKind = "Pull（仅快进）"
	actionPush            actionKind = "Push"
	actionSaveStash       actionKind = "保存 Stash"
	actionApplyStash      actionKind = "应用 Stash"
	actionPopStash        actionKind = "应用并移除 Stash"
	actionDropStash       actionKind = "删除 Stash"
	actionMerge           actionKind = "Merge"
	actionRebase          actionKind = "Rebase"
	actionCherryPick      actionKind = "Cherry-pick"
	actionRevert          actionKind = "Revert"
	actionContinue        actionKind = "继续操作"
	actionAbort           actionKind = "中止操作"
	actionAddWorktree     actionKind = "新建 Worktree"
	actionRemoveWorktree  actionKind = "移除 Worktree"
	actionSubmodule       actionKind = "初始化／更新子模块"
	actionResolve         actionKind = "保存并标记解决"
	actionResolveDeletion actionKind = "接受冲突中的删除"
)

type actionDraft struct {
	Kind                                           actionKind
	First, Second, Third                           string
	Flag                                           bool
	Expected                                       gitrepo.Operation
	Conflict                                       gitrepo.Conflict
	Result                                         string
	Remote                                         gitrepo.Remote
	RemoteCaptured                                 bool
	ExpectedHead, ExpectedBranch, ExpectedUpstream string
}
type actionTarget struct {
	Draft      actionDraft
	Root       string
	Generation uint64
}

func (a *app) capture(d actionDraft) actionTarget {
	root := ""
	if a.repo != nil {
		root = a.repo.Root
	}
	d.ExpectedHead, d.ExpectedBranch, d.ExpectedUpstream = a.data.status.Head, a.data.status.Branch, a.data.status.Upstream
	if d.Kind == actionPush || d.Kind == actionFetch || d.Kind == actionPull {
		name := d.First
		if d.Kind == actionPull {
			name = strings.SplitN(a.data.status.Upstream, "/", 2)[0]
		}
		d.RemoteCaptured = false
		for _, remote := range a.data.remotes {
			if remote.Name == name {
				d.Remote = remote
				d.RemoteCaptured = true
				break
			}
		}
	}
	return actionTarget{Draft: d, Root: root, Generation: a.guard.repository}
}
func (t actionTarget) valid(r *gitrepo.Repository, g *jobGuard) bool {
	root := ""
	if r != nil {
		root = r.Root
	}
	return root == t.Root && g.repository == t.Generation && !g.writing
}
func (d actionDraft) editable() bool {
	switch d.Kind {
	case actionInit, actionClone, actionCreateBranch, actionCreateTag, actionRename, actionTrack, actionAddRemote, actionPush, actionSaveStash, actionAddWorktree:
		return true
	}
	return false
}
func (d actionDraft) network() bool {
	switch d.Kind {
	case actionClone, actionFetch, actionPull, actionPush, actionSubmodule:
		return true
	}
	return false
}
func (d actionDraft) run(ctx context.Context, r *gitrepo.Repository) error {
	if r == nil {
		return errors.New("请先打开仓库")
	}
	if d.ExpectedHead != "" {
		s, err := r.Status(ctx)
		if err != nil {
			return err
		}
		if s.Head != d.ExpectedHead || s.Branch != d.ExpectedBranch || s.Upstream != d.ExpectedUpstream {
			return errors.New("当前分支或上游已变化，请刷新并重新确认")
		}
	}
	if d.Kind == actionPush || d.Kind == actionFetch || d.Kind == actionPull {
		if !d.RemoteCaptured {
			return errors.New("请刷新并确认有效的远端地址")
		}
		remotes, err := r.Remotes(ctx)
		if err != nil {
			return err
		}
		matches := false
		for _, remote := range remotes {
			if remote == d.Remote {
				matches = true
				break
			}
		}
		if !matches {
			return errors.New("远端地址已变化，请刷新并重新确认目标")
		}
	}
	switch d.Kind {
	case actionCreateTag:
		return r.CreateTag(ctx, d.First, d.Second)
	case actionCreateBranch:
		return r.CreateBranch(ctx, d.First, d.Second, d.Flag)
	case actionSwitch:
		return r.SwitchBranch(ctx, d.First)
	case actionRename:
		return r.RenameBranch(ctx, d.First, d.Second)
	case actionDeleteBranch:
		return r.DeleteBranch(ctx, d.First)
	case actionTrack:
		return r.TrackBranch(ctx, d.First, d.Second)
	case actionAddRemote:
		return r.AddRemote(ctx, d.First, d.Second)
	case actionFetch:
		return r.Fetch(ctx, d.First)
	case actionPull:
		return r.PullFF(ctx)
	case actionPush:
		return r.Push(ctx, d.First, d.Second, d.Third, d.Flag)
	case actionSaveStash:
		return r.SaveStash(ctx, d.First, d.Flag)
	case actionApplyStash:
		return r.ApplyStash(ctx, d.First)
	case actionPopStash:
		return r.PopStash(ctx, d.Second, d.First)
	case actionDropStash:
		return r.DropStash(ctx, d.Second, d.First)
	case actionMerge:
		return r.Integrate(ctx, "merge", d.First)
	case actionRebase:
		return r.Integrate(ctx, "rebase", d.First)
	case actionCherryPick:
		return r.Integrate(ctx, "cherry-pick", d.First)
	case actionRevert:
		return r.Integrate(ctx, "revert", d.First)
	case actionContinue, actionAbort:
		op, err := r.Operation(ctx)
		if err != nil {
			return err
		}
		if op != d.Expected {
			return errors.New("进行中操作已变化，请刷新")
		}
		if d.Kind == actionContinue {
			return r.Continue(ctx, d.First)
		}
		return r.Abort(ctx, d.First)
	case actionAddWorktree:
		return r.AddWorktree(ctx, d.First, d.Second, d.Third)
	case actionRemoveWorktree:
		return r.RemoveWorktree(ctx, d.First)
	case actionSubmodule:
		return r.UpdateSubmodule(ctx, d.First)
	case actionResolve:
		return r.ResolveConflict(ctx, d.Conflict, d.Result)
	case actionResolveDeletion:
		return r.ResolveConflictDeletion(ctx, d.Conflict)
	}
	return fmt.Errorf("不支持的操作：%s", d.Kind)
}
func (d actionDraft) description() string {
	switch d.Kind {
	case actionCreateTag:
		return "本地轻量标签：refs/tags/" + d.First + "\n目标提交 SHA：" + d.Second + "\n不覆盖已有标签；保留 HEAD、暂存区和工作区，不推送到远端。"
	case actionResolveDeletion:
		return "删除冲突文件：" + d.First + "\n接受删除版本并暂存删除；当前文件内容将移除。"
	case actionResolve:
		return fmt.Sprintf("文件：%s\n结果：%d 字节\n写入编辑结果并暂存此文件；保存前核对冲突版本和外部改动。", d.First, len(d.Result))
	case actionInit:
		return "新目录：" + d.First + "\n创建 main 分支的空仓库。"
	case actionClone:
		return "来源：" + displayURL(d.First) + "\n新目录：" + d.Second + "\n使用系统 Git 凭据助手；子模块需稍后显式初始化。"
	case actionPush:
		return fmt.Sprintf("远端：%s\n源：refs/heads/%s\n目标：refs/heads/%s\n普通推送，检查子模块提交是否已发布。设置上游：%t", d.First, d.Second, d.Third, d.Flag)
	case actionSaveStash:
		return fmt.Sprintf("说明：%s\n包含未跟踪文件：%t\n保存后清理已保存内容；子模块内部改动需单独处理。", d.First, d.Flag)
	case actionApplyStash, actionPopStash, actionDropStash:
		return "Stash SHA：" + d.First + "\n列表位置：" + d.Second + "\n应用恢复暂存状态；冲突时保留 Stash。删除后可能无法找回。"
	case actionAbort:
		return "操作：" + d.First + "\n恢复操作前状态；本次冲突编辑可能被覆盖。"
	case actionContinue:
		return "完成已解决的 " + d.First + "；Git hooks 仍会执行。"
	case actionRemoveWorktree:
		return "移除目录：" + d.First + "\n仅允许干净、未锁定的其他工作区；保留分支。"
	case actionSubmodule:
		return "子模块：" + d.First + "\n初始化并 checkout 到暂存区 Index 指针，不递归更新，不跟随远端最新提交。"
	case actionPull:
		return "从当前分支上游拉取，仅允许快进；分叉时停止，不自动合并或变基。"
	case actionAddRemote:
		return "远端：" + d.First + "\n地址：" + displayURL(d.Second)
	default:
		return d.First + "\n" + d.Second + "\n" + d.Third
	}
}
