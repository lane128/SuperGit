package lobedesk

import (
	"context"
	"strings"

	"supergit/internal/gitrepo"
)

// List the actual destination in the background. A repository refresh, a new
// palette, or another dialog must not receive this operation's picker.
func (a *app) paletteAdvertisedRefs(title string, remote gitrepo.Remote, tags bool, choose func(gitrepo.Branch)) {
	if a.repo == nil || a.loading || a.busy || a.tabSnapshotUnavailable() {
		return
	}
	repo, epoch, generation := a.repo, a.loadSeq, a.commandPaletteGeneration
	previous := a.commandPalette
	ctx, cancel := a.beginOperation("")
	go func() {
		defer cancel()
		refs, err := repo.AdvertisedRefs(ctx, remote, tags)
		cancelled := ctx.Err() == context.Canceled
		a.hopToMain(func() {
			a.finishOperation()
			a.invalidate()
			if cancelled || a.repo != repo || a.activeRepositorySession() != a || a.loadSeq != epoch || a.commandPaletteGeneration != generation || a.localWatch.closed || a.hasPendingDialog(toastMain) {
				return
			}
			if err != nil {
				a.queue("error", a.paletteText("Could not read remote references", "读取远端引用失败"), a.operationErrorDescription(err))
				return
			}
			if len(refs) == 0 {
				hint := a.t("noRemoteBranches")
				if tags {
					hint = a.t("noTags")
				}
				a.queue("info", remote.Name+" · "+hint, "")
				return
			}
			hint, badge := a.t("branchName"), "R"
			if tags {
				hint, badge = a.t("tagName"), "T"
			}
			// Preserve Back navigation and the original command availability guard
			// across the asynchronous read. The previous selection released its modal.
			a.commandPalette = previous
			choosePaletteObject(a, title+" · "+remote.Name, hint, refs, func(ref gitrepo.Branch) paletteChoice {
				return paletteChoice{id: ref.Name, label: ref.Name, prefix: badge, sha: ref.SHA, detail: shortSHA(ref.SHA)}
			}, choose)
		})
	}()
}

func (a *app) paletteDeleteRemoteTag() {
	title := a.paletteText("Delete Remote Tag", "删除远端标签")
	a.choosePaletteRemote(title, func(remote gitrepo.Remote) {
		a.paletteAdvertisedRefs(title, remote, true, func(tag gitrepo.Branch) {
			target := remote.Name + "/" + tag.Name + " (" + shortSHA(tag.SHA) + ")"
			a.askConfirm(title, a.paletteText("Delete remote tag "+target+"?", "删除远端标签 "+target+"？"), a.t("delete"), true, func() {
				a.perform(a.paletteText("Could not delete remote tag", "删除远端标签失败"), a.paletteText("Remote tag deleted", "已删除远端标签"), func(ctx context.Context, repo *gitrepo.Repository) error {
					return repo.DeleteRemoteTagChecked(ctx, remote, tag.Name, tag.SHA)
				})
			})
		})
	})
}

func (a *app) palettePull() {
	title := a.t("pull")
	start := gitrepo.Status{Head: a.status.Head, Branch: a.status.Branch}
	a.choosePalette(title, a.paletteText("Pull options", "拉取选项"), []paletteChoice{
		{id: "ff-only", label: "pull --ff-only", detail: a.paletteText("Fast-forward only", "仅允许快进")},
		{id: "merge", label: "pull --no-rebase", detail: a.paletteText("Merge the remote branch", "合并远端分支")},
		{id: "rebase", label: "pull --rebase", detail: a.paletteText("Replay local commits on the remote branch", "在远端分支上重放本地提交")},
	}, func(mode paletteChoice) {
		a.choosePaletteRemote(mode.label, func(remote gitrepo.Remote) {
			a.paletteAdvertisedRefs(mode.label, remote, false, func(branch gitrepo.Branch) {
				target := remote.Name + "/" + branch.Name + " (" + shortSHA(branch.SHA) + ") → " + start.Branch
				a.askConfirm(mode.label, target, title, false, func() {
					a.performAs(busyPull, a.t("pullFailed"), a.t("pullSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
						return repo.PullBranchChecked(ctx, remote, branch, start, mode.id)
					})
				})
			})
		})
	})
}

func (a *app) palettePush() {
	force := paletteChoice{id: "force", label: "push --force-with-lease", detail: a.paletteText("Require the expected remote tip", "仅覆盖已确认的远端提交")}
	if a.remoteBranchesErr != "" {
		force.reason = a.t("quickCheckoutReadError")
	}
	a.choosePalette(a.t("pushButton"), a.paletteText("Push options", "推送选项"), []paletteChoice{
		{id: "push", label: "push", detail: a.paletteText("Publish a branch", "推送分支")},
		{id: "track", label: "push --set-upstream", detail: a.paletteText("Also set the branch upstream", "同时设置分支上游")},
		force,
	}, func(mode paletteChoice) {
		a.choosePaletteBranch(mode.label, a.branches, "L", nil, func(branch gitrepo.Branch) {
			a.choosePaletteRemote(mode.label+" · "+branch.Name, func(remote gitrepo.Remote) {
				name := branch.Name
				if upstream, ok := strings.CutPrefix(branch.Upstream, remote.Name+"/"); ok {
					name = upstream
				}
				a.inputPalette(mode.label+" · "+remote.Name, a.paletteText("Destination branch", "目标分支名"), name, func(destination string) {
					a.paletteConfirmPush(mode.id, remote, branch, destination)
				})
			})
		})
	})
}

func (a *app) paletteConfirmPush(mode string, remote gitrepo.Remote, source gitrepo.Branch, destination string) {
	if mode == "force" {
		if a.remoteBranchesErr != "" {
			return
		}
		lease := ""
		for _, ref := range a.remoteBranches {
			if ref.Name == remote.Name+"/"+destination {
				lease = ref.SHA
				break
			}
		}
		expected := a.t("forcePushRemoteAbsent")
		if lease != "" {
			expected = shortSHA(lease)
		}
		a.askConfirm(a.t("forcePushTitle"), a.tf("forcePushMessage", source.Name, remote.Name, destination, shortSHA(source.SHA), expected), a.t("forcePush"), true, func() {
			a.performAs(busyForcePush, a.t("forcePushFailed"), a.t("forcePushSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
				return repo.PushForceChecked(ctx, remote, source, destination, lease)
			})
		})
		return
	}
	title := a.t("pushDialogTitle")
	if mode == "track" {
		title = "push --set-upstream"
	}
	a.askConfirm(title, a.tf("pushMessage", source.Name, remote.Name, destination, shortSHA(source.SHA)), a.t("pushButton"), false, func() {
		a.performAs(busyPush, a.t("pushFailed"), a.t("pushSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.PushChecked(ctx, remote, source, destination, mode == "track")
		})
	})
}
