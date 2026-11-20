package lobedesk

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"supergit/internal/gitrepo"
)

func init() { registerPaletteActions(refPaletteActions()...) }

func refPaletteActions() []paletteAction {
	actions := []paletteAction{
		{ID: "branch.createFromTag", English: "Create Branch from Tag…", Chinese: "从标签创建分支…", Run: (*app).paletteBranchFromTag},
		{ID: "branch.createFromRemote", English: "Create Branch from Remote Branch…", Chinese: "从远端分支创建分支…", Run: (*app).paletteBranchFromRemote},
		{ID: "branch.rename", English: "Rename Branch…", Chinese: "重命名分支…", Run: (*app).paletteRenameBranch},
		{ID: "branch.delete", English: "Delete Branch…", Chinese: "删除分支…", Run: (*app).paletteDeleteBranch},
		{ID: "branch.deleteRemote", English: "Delete Remote Branch…", Chinese: "删除远端分支…", Run: (*app).paletteDeleteRemoteBranch},
		{ID: "branch.rebase", English: "Rebase Branch…", Chinese: "变基分支…", Run: (*app).paletteRebaseBranch},
		{ID: "repo.pull", English: "Pull…", Chinese: "拉取…", Run: (*app).palettePull},
		{ID: "repo.push", English: "Push…", Chinese: "推送…", Run: (*app).palettePush},
		{ID: "remote.add", English: "Add Remote…", Chinese: "添加远端…", Run: (*app).paletteAddRemote},
		{ID: "remote.delete", English: "Delete Remote…", Chinese: "删除远端…", Run: (*app).paletteDeleteRemote},
		{ID: "remote.rename", English: "Rename Remote…", Chinese: "重命名远端…", Run: (*app).paletteRenameRemote},
		{ID: "tag.delete", English: "Delete Tag…", Chinese: "删除标签…", Run: (*app).paletteDeleteTag},
		{ID: "tag.deleteRemote", English: "Delete Remote Tag…", Chinese: "删除远端标签…", Run: (*app).paletteDeleteRemoteTag},
		{ID: "tag.push", English: "Push Tag…", Chinese: "推送标签…", Run: (*app).palettePushTag},
		{ID: "branch.setUpstream", English: "Set Branch Upstream", Chinese: "设置分支上游", Run: func(a *app) { a.paletteUpstream(true) }},
		{ID: "branch.unsetUpstream", English: "Unset Branch Upstream", Chinese: "取消分支上游", Run: func(a *app) { a.paletteUpstream(false) }},
	}
	for i, action := range actions {
		id := action.ID
		actions[i].Unavailable = func(a *app) string { return a.refPaletteUnavailable(id) }
	}
	return actions
}

func (a *app) refPaletteUnavailable(id string) string {
	switch id {
	case "remote.add":
		return ""
	case "remote.delete", "remote.rename", "tag.deleteRemote":
		if len(a.remotes) == 0 {
			return a.t("noRemote")
		}
	case "repo.pull", "repo.push":
		if len(a.remotes) == 0 {
			return a.t("noRemote")
		}
		if id == "repo.push" {
			return a.localRefListReason()
		}
		if a.status.Branch == "" || a.status.Head == "" {
			return a.t("commandPaletteNeedHead")
		}
		return a.paletteRefCheckoutReason()
	case "tag.delete", "tag.push", "branch.createFromTag":
		if id == "tag.push" && len(a.remotes) == 0 {
			return a.t("noRemote")
		}
		if a.tagsErr != "" {
			return a.t("quickCheckoutReadError")
		}
		if len(a.tags) == 0 {
			return a.t("noTags")
		}
		if id == "branch.createFromTag" {
			return a.paletteRefCheckoutReason()
		}
	case "branch.createFromRemote", "branch.deleteRemote", "branch.setUpstream":
		if a.remoteBranchesErr != "" {
			return a.t("quickCheckoutReadError")
		}
		if len(a.remoteBranches) == 0 {
			return a.t("noRemoteBranches")
		}
		switch id {
		case "branch.createFromRemote":
			return a.paletteRefCheckoutReason()
		case "branch.setUpstream":
			return a.localRefListReason()
		}
	case "branch.rebase":
		if a.remoteBranchesErr != "" || a.branchesErr != "" || a.loadErr != "" {
			return a.t("quickCheckoutReadError")
		}
		if a.status.Branch == "" || a.status.Head == "" {
			return a.t("commandPaletteNeedHead")
		}
		return a.paletteRefCheckoutReason()
	default:
		return a.localRefListReason()
	}
	return ""
}

// A tag or remote source already supplies a commit, including in a freshly
// initialized repository that has fetched refs but has no local HEAD yet.
func (a *app) paletteRefCheckoutReason() string {
	switch {
	case a.loadErr != "":
		return a.t("quickCheckoutReadError")
	case a.operation.Kind != "":
		return a.t("quickCheckoutOperation")
	case len(a.status.Entries) != 0:
		return a.t("quickCheckoutDirty")
	default:
		return ""
	}
}

func (a *app) localRefListReason() string {
	if a.branchesErr != "" {
		return a.t("quickCheckoutReadError")
	}
	if len(a.branches) == 0 {
		return a.t("noBranches")
	}
	return ""
}

// Keep the selected Git object alongside its rendered row. Later steps must
// carry its SHA or remote fingerprint, even if a model slice is replaced.
func choosePaletteObject[T any](a *app, title, hint string, objects []T, row func(T) paletteChoice, choose func(T)) {
	options := make([]paletteChoice, 0, len(objects))
	targets := make(map[string]T, len(objects))
	for _, object := range objects {
		item := row(object)
		options = append(options, item)
		targets[item.id] = object
	}
	a.choosePalette(title, hint, options, func(item paletteChoice) {
		if target, ok := targets[item.id]; ok && item.reason == "" {
			choose(target)
		}
	})
}

func (a *app) choosePaletteBranch(title string, branches []gitrepo.Branch, badge string, reason func(gitrepo.Branch) string, choose func(gitrepo.Branch)) {
	choosePaletteObject(a, title, a.t("branchName"), branches, func(b gitrepo.Branch) paletteChoice {
		item := paletteChoice{id: b.Name, label: b.Name, prefix: badge, sha: b.SHA, detail: shortSHA(b.SHA)}
		if reason != nil {
			item.reason = reason(b)
		}
		return item
	}, choose)
}

func (a *app) choosePaletteTag(title string, commitOnly bool, choose func(gitrepo.Tag)) {
	choosePaletteObject(a, title, a.t("tagName"), a.tags, func(t gitrepo.Tag) paletteChoice {
		item := paletteChoice{id: t.Name, label: t.Name, prefix: "T", sha: t.ObjectSHA, detail: shortSHA(t.ObjectSHA)}
		if commitOnly && t.TargetType != "commit" {
			item.reason = a.paletteText("Tag points to a "+t.TargetType, "标签指向 "+t.TargetType+" 对象")
		}
		return item
	}, choose)
}

func (a *app) choosePaletteRemote(title string, choose func(gitrepo.Remote)) {
	choosePaletteObject(a, title, a.t("remoteName"), a.remotes, func(r gitrepo.Remote) paletteChoice {
		return paletteChoice{id: r.Name, label: r.Name, sha: r.Fingerprint}
	}, choose)
}

func (a *app) paletteBranchFromTag() {
	title := a.t("createBranchFromTag")
	a.choosePaletteTag(title, true, func(tag gitrepo.Tag) {
		a.inputPalette(title+" · "+tag.Name, a.t("branchName"), "", func(name string) {
			a.perform(a.t("createBranchFailed"), a.t("branchCreated"), func(ctx context.Context, repo *gitrepo.Repository) error {
				return repo.CreateBranchFromTagChecked(ctx, name, tag)
			})
		})
	})
}

func (a *app) paletteBranchFromRemote() {
	title := a.paletteText("Create Branch from Remote Branch", "从远端分支创建分支")
	a.choosePaletteBranch(title, a.remoteBranches, "R", nil, func(b gitrepo.Branch) {
		_, suggested := a.remoteBranchParts(b.Name)
		a.inputPalette(title+" · "+b.Name, a.t("branchName"), suggested, func(name string) {
			a.performAs(busyCheckout, a.t("checkoutRemoteBranchFailed"), a.t("remoteBranchCheckedOut"), func(ctx context.Context, repo *gitrepo.Repository) error {
				return repo.TrackBranchChecked(ctx, name, b.Name, b.SHA)
			})
		})
	})
}

func (a *app) paletteRenameBranch() {
	title := a.t("renameBranchTitle")
	a.choosePaletteBranch(title, a.branches, "L", nil, func(b gitrepo.Branch) {
		a.inputPalette(title+" · "+b.Name, a.t("branchName"), b.Name, func(name string) {
			if name == b.Name {
				return
			}
			a.perform(a.t("renameBranchFailed"), a.t("branchRenamed"), func(ctx context.Context, repo *gitrepo.Repository) error {
				return repo.RenameBranchChecked(ctx, b.Name, name, b.SHA)
			})
		})
	})
}

func (a *app) paletteDeleteBranch() {
	title := a.t("deleteBranchTitle")
	a.choosePaletteBranch(title, a.branches, "L", func(b gitrepo.Branch) string {
		if b.Current || b.Name == a.status.Branch {
			return a.t("current")
		}
		for _, wt := range a.worktrees {
			if wt.Branch == b.Name {
				return a.t("quickCheckoutOccupied")
			}
		}
		return ""
	}, func(b gitrepo.Branch) {
		a.choosePalette(title+" · "+b.Name, a.t("delete"), []paletteChoice{
			{id: "safe", label: "git branch -d", detail: a.paletteText("Delete a merged branch", "删除已合并的分支")},
			{id: "force", label: "git branch -D", detail: a.paletteText("Also allow an unmerged branch", "允许删除未合并的分支")},
		}, func(item paletteChoice) {
			if item.id == "force" {
				a.confirmForceDeleteBranch(b)
			} else {
				a.confirmDeleteBranch(b)
			}
		})
	})
}

func (a *app) paletteDeleteRemoteBranch() {
	a.choosePaletteBranch(a.t("deleteRemoteBranchTitle"), a.remoteBranches, "R", nil, func(b gitrepo.Branch) {
		remote, name := a.remoteBranchParts(b.Name)
		a.confirmDeleteRemoteBranch(remote, name, b.SHA)
	})
}

func (a *app) paletteRebaseBranch() {
	var options []paletteChoice
	add := func(branches []gitrepo.Branch, kind, prefix string) {
		for _, b := range branches {
			if b.SHA == "" || kind == "L" && (b.Current || b.Name == a.status.Branch) {
				continue
			}
			options = append(options, paletteChoice{id: prefix + b.Name, label: b.Name, prefix: kind, sha: b.SHA, detail: shortSHA(b.SHA)})
		}
	}
	add(a.branches, "L", "refs/heads/")
	add(a.remoteBranches, "R", "refs/remotes/")
	a.choosePalette(a.t("rebaseOnto"), a.t("branchName"), options, func(item paletteChoice) {
		a.confirmIntegrate("rebase", item.id, item.sha)
	})
}

func (a *app) paletteAddRemote() {
	title := a.t("addRemote")
	a.inputPalette(title, a.t("remoteName"), "", func(name string) {
		a.inputPalette(title+" · "+name, a.t("remoteAddress"), "", func(address string) {
			invalid := errors.New(a.t("invalidRemoteAddress"))
			a.perform(a.t("addRemoteFailed"), a.t("remoteAdded"), func(ctx context.Context, repo *gitrepo.Repository) error {
				err := repo.AddRemote(ctx, strings.TrimSpace(name), strings.TrimSpace(address))
				var parseErr *url.Error
				if errors.As(err, &parseErr) {
					return invalid
				}
				return err
			})
		})
	})
}

func (a *app) paletteRenameRemote() {
	title := a.paletteText("Rename Remote", "重命名远端")
	a.choosePaletteRemote(title, func(remote gitrepo.Remote) {
		a.inputPalette(title+" · "+remote.Name, a.t("remoteName"), remote.Name, func(name string) {
			if name == remote.Name {
				return
			}
			a.perform(a.paletteText("Could not rename remote", "重命名远端失败"), a.paletteText("Remote renamed", "已重命名远端"), func(ctx context.Context, repo *gitrepo.Repository) error {
				return repo.RenameRemoteChecked(ctx, remote, name)
			})
		})
	})
}

func (a *app) paletteDeleteRemote() {
	title := a.paletteText("Delete Remote", "删除远端")
	a.choosePaletteRemote(title, func(remote gitrepo.Remote) {
		message := a.paletteText("Remove "+remote.Name+" and its remote-tracking branches from this repository?", "从此仓库移除 "+remote.Name+" 及其远端跟踪分支？")
		a.askConfirm(title, message, a.t("delete"), true, func() {
			a.perform(a.paletteText("Could not remove remote", "删除远端失败"), a.paletteText("Remote removed", "已删除远端"), func(ctx context.Context, repo *gitrepo.Repository) error {
				return repo.RemoveRemoteChecked(ctx, remote)
			})
		})
	})
}

func (a *app) paletteDeleteTag() {
	a.choosePaletteTag(a.t("deleteTagTitle"), false, a.confirmDeleteTag)
}

func (a *app) palettePushTag() {
	title := a.t("pushTagTitle")
	a.choosePaletteTag(title, false, func(tag gitrepo.Tag) {
		a.choosePaletteRemote(title+" · "+tag.Name, func(remote gitrepo.Remote) {
			a.askConfirm(title, a.tf("pushTagMessage", tag.Name, shortSHA(tag.ObjectSHA), remote.Name), a.t("pushButton"), false, func() {
				a.performAs(busyPushTag, a.t("pushTagFailed"), a.t("tagPushed"), func(ctx context.Context, repo *gitrepo.Repository) error {
					return repo.PushTagChecked(ctx, remote, tag)
				})
			})
		})
	})
}

func (a *app) paletteUpstream(set bool) {
	title := a.paletteText("Set Branch Upstream", "设置分支上游")
	if !set {
		title = a.paletteText("Unset Branch Upstream", "取消分支上游")
	}
	a.choosePaletteBranch(title, a.branches, "L", func(b gitrepo.Branch) string {
		if !set && b.Upstream == "" {
			return a.t("noUpstream")
		}
		return ""
	}, func(b gitrepo.Branch) {
		change := func(target *gitrepo.Branch) {
			a.perform(a.paletteText("Could not change upstream", "修改上游失败"), a.paletteText("Upstream updated", "已更新上游"), func(ctx context.Context, repo *gitrepo.Repository) error {
				return repo.ChangeUpstreamChecked(ctx, b, target)
			})
		}
		if set {
			a.choosePaletteBranch(title+" · "+b.Name, a.remoteBranches, "R", nil, func(target gitrepo.Branch) { change(&target) })
		} else {
			a.askConfirm(title, b.Name+" → "+b.Upstream, title, false, func() { change(nil) })
		}
	})
}
