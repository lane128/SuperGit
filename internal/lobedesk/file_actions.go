package lobedesk

import (
	"context"
	"errors"
	"slices"

	"supergit/internal/gitrepo"
)

// File actions carry raw path/version and a repository refresh generation.
// Neither the selected file cursor nor a later repository can supply identity.
type fileActionTarget struct {
	Root       string
	Generation uint64
	Entry      gitrepo.Entry
	Staged     bool
}

func (a *app) fileActionTarget(entry gitrepo.Entry, staged bool) (fileActionTarget, bool) {
	if a.repo == nil {
		return fileActionTarget{}, false
	}
	target := fileActionTarget{Root: a.repo.Root, Generation: a.loadSeq, Entry: entry, Staged: staged}
	return target, a.fileActionTargetCurrent(target)
}

// Whole-file gitlink Stage/Unstage changes only the parent's Index. Internal
// child edits alone have no pointer to stage; destructive child resets remain
// protected by the separate discard guards below.
func fileStageSupported(entry gitrepo.Entry, staged bool) bool {
	if entry.Conflict {
		return false
	}
	if !entry.IsSubmodule() || staged {
		return true
	}
	return entry.Worktree == 'D' || len(entry.Submodule) > 1 && entry.Submodule[1] == 'C'
}

func (a *app) fileActionTargetCurrent(target fileActionTarget) bool {
	if a.repo == nil || a.repo.Root != target.Root || a.loadSeq != target.Generation || a.busy || a.loading || !fileStageSupported(target.Entry, target.Staged) {
		return false
	}
	for _, entry := range a.status.Entries {
		if entry == target.Entry {
			return (target.Staged && entry.Staged()) || (!target.Staged && entry.WorkspaceUnstaged())
		}
	}
	return false
}

type fileValidationScope int

const (
	fileTargetsOnly fileValidationScope = iota
	fileTargetsBatch
	fileTargetsDiscardAll
)

func validateFileActionTargets(ctx context.Context, repo *gitrepo.Repository, targets []fileActionTarget, scope fileValidationScope) error {
	if len(targets) == 0 || repo.Root != targets[0].Root {
		return errors.New("文件操作目标已改变，请刷新后重试")
	}
	status, err := repo.Status(ctx)
	if err != nil {
		return err
	}
	entries := make(map[gitrepo.Entry]struct{}, len(status.Entries))
	for _, entry := range status.Entries {
		if scope != fileTargetsOnly && batchFileBlocked(entry, scope == fileTargetsDiscardAll) {
			return errors.New("存在冲突或受保护的子模块，请逐个处理")
		}
		entries[entry] = struct{}{}
	}
	for _, target := range targets {
		if _, ok := entries[target.Entry]; target.Root != repo.Root || !ok || !fileStageSupported(target.Entry, target.Staged) {
			return errors.New("文件操作目标已改变，请刷新后重试")
		}
	}
	return nil
}

func (a *app) prepareFileStage(target fileActionTarget) func(context.Context, *gitrepo.Repository) error {
	if !a.fileActionTargetCurrent(target) {
		return nil
	}
	return func(ctx context.Context, repo *gitrepo.Repository) error {
		if err := validateFileActionTargets(ctx, repo, []fileActionTarget{target}, fileTargetsOnly); err != nil {
			return err
		}
		entry := target.Entry
		if entry.OriginalPath != "" {
			return repo.StageRename(ctx, entry.Path, entry.OriginalPath, target.Staged)
		}
		if target.Staged {
			return repo.Unstage(ctx, entry.Path)
		}
		return repo.Stage(ctx, entry.Path)
	}
}

func (a *app) prepareFileDiscard(entry gitrepo.Entry) {
	target, ok := a.fileActionTarget(entry, false)
	if ok {
		a.prepareDiscardTargets([]fileActionTarget{target}, false)
	}
}

func (a *app) discardTargetsCurrent(targets []fileActionTarget, batch bool) bool {
	if len(targets) == 0 {
		return false
	}
	for _, target := range targets {
		if target.Entry.IsSubmodule() || !a.fileActionTargetCurrent(target) {
			return false
		}
	}
	if batch {
		current, reason := a.batchFileTargets("discard")
		return reason == "" && slices.Equal(current, targets)
	}
	return true
}

func (a *app) prepareDiscardTargets(targets []fileActionTarget, batch bool) {
	a.prepareScopedDiscardTargets(targets, batch, false)
}

// A selected subset keeps its captured targets; it is neither a single-file
// message nor the whole-repository validation used by Discard All.
func (a *app) prepareScopedDiscardTargets(targets []fileActionTarget, batch, selected bool) {
	if !a.discardTargetsCurrent(targets, batch) {
		return
	}
	targets = append([]fileActionTarget(nil), targets...)
	target := targets[0]
	repo, ctx := a.repo, a.ctx
	stale := a.t("patchTargetChanged")
	a.discardPrepareSeq++
	request := a.discardPrepareSeq
	requests := make([]gitrepo.DiscardTarget, len(targets))
	for i, target := range targets {
		requests[i] = gitrepo.DiscardTarget{Path: target.Entry.Path, Untracked: isUntracked(target.Entry)}
	}
	go func() {
		prepared, err := repo.PrepareDiscard(ctx, requests)
		a.hopToMain(func() {
			if request != a.discardPrepareSeq || !a.discardTargetsCurrent(targets, batch) || a.confirm.open || a.prompt.open || a.reset.open ||
				selected && !a.workspaceDiscardSurfaceAllowed() {
				return
			}
			if err != nil {
				a.queue("error", a.t("discardFailed"), err.Error())
				a.invalidate()
				return
			}
			title, message, label := a.t("discardTitle"), a.tf("discardMessage", target.Entry.Path), a.t("discard")
			busy, failure, success := busyDiscard, a.t("discardFailed"), a.t("discardSuccess")
			if requests[0].Untracked {
				title, message, label = a.t("deleteTitle"), a.tf("deleteMessage", target.Entry.Path), a.t("delete")
				busy, failure, success = busyDeleteFile, a.t("deleteFailed"), a.t("deleteSuccess")
			} else if batch {
				title, message, label = a.t("discardAll"), a.tf("discardAllMessage", len(targets)), a.t("discardAll")
			} else if selected && len(targets) > 1 {
				title, message, label = a.tf("discardSelectedFiles", len(targets)), a.tf("discardSelectedMessage", len(targets)), a.t("discard")
			}
			a.askConfirm(title, message, label, true, func() {
				if !a.discardTargetsCurrent(targets, batch) {
					a.queue("info", stale, "")
					a.invalidate()
					return
				}
				a.performAs(busy, failure, success, func(ctx context.Context, current *gitrepo.Repository) error {
					if current.Root != target.Root {
						return errors.New(stale)
					}
					if batch || selected {
						scope := fileTargetsOnly
						if batch {
							scope = fileTargetsDiscardAll
						}
						if err := validateFileActionTargets(ctx, current, targets, scope); err != nil {
							return err
						}
					}
					return current.DiscardPrepared(ctx, prepared)
				})
			})
		})
	}()
}
