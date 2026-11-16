package lobedesk

import (
	"context"

	"supergit/internal/gitrepo"
)

// Every batch reports one explicit scope. A protected entry blocks the batch
// instead of being silently skipped while the UI claims 'all' succeeded.
func (a *app) batchFileTargets(kind string) ([]fileActionTarget, string) {
	if a.repo == nil || a.loading {
		return nil, a.t("loadingRepo")
	}
	if a.busy {
		return nil, a.t("fileActionBusy")
	}
	targets := make([]fileActionTarget, 0, len(a.status.Entries))
	for _, entry := range a.status.Entries {
		if entry.Conflict || kind == "discard" && entry.IsSubmodule() {
			return nil, a.t("fileActionsProtected")
		}
		include := false
		switch kind {
		case "stage":
			include = entry.Unstaged()
		case "unstage":
			include = entry.Staged()
		case "discard":
			include = entry.Unstaged() && !isUntracked(entry)
		}
		if include {
			if !fileStageSupported(entry, kind == "unstage") {
				return nil, a.t("fileActionsProtected")
			}
			targets = append(targets, fileActionTarget{Root: a.repo.Root, Generation: a.loadSeq, Entry: entry, Staged: kind == "unstage"})
		}
	}
	if len(targets) == 0 {
		return nil, a.t("noChangesShort")
	}
	return targets, ""
}

func (a *app) runBatchStage(staged bool) {
	kind, label, busy := "stage", a.t("stageAll"), busyStageAll
	if staged {
		kind, label, busy = "unstage", a.t("unstageAll"), busyUnstageAll
	}
	targets, reason := a.batchFileTargets(kind)
	if reason != "" {
		return
	}
	paths := make([]string, 0, len(targets))
	for _, target := range targets {
		if staged && target.Entry.OriginalPath != "" {
			paths = append(paths, target.Entry.OriginalPath)
		}
		paths = append(paths, target.Entry.Path)
	}
	a.performAs(busy, label, "", func(ctx context.Context, repo *gitrepo.Repository) error {
		if err := validateFileActionTargets(ctx, repo, targets, true); err != nil {
			return err
		}
		if staged {
			return repo.UnstagePaths(ctx, paths)
		}
		return repo.StagePaths(ctx, paths)
	})
}

func (a *app) confirmDiscardAll() {
	targets, reason := a.batchFileTargets("discard")
	if reason == "" {
		a.prepareDiscardTargets(targets, true)
	}
}
