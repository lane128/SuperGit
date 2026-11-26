package lobedesk

import (
	"errors"

	"supergit/internal/gitrepo"
)

// operationErrorDescription keeps original Git output after localized guidance.
// A failed stash restore is not atomic, even when tracked edits were refused.
func (a *app) operationErrorDescription(err error) string {
	var workspace *gitrepo.WorkspaceActionError
	if errors.As(err, &workspace) && a.locale == localeEnglish && workspace.English != "" {
		guidance := workspace.English
		if workspace.Cause != nil {
			guidance += "\n\n" + a.operationErrorDescription(workspace.Cause)
		}
		return guidance
	}
	if errors.Is(err, gitrepo.ErrDirtyWorkingTree) {
		guidance := a.t("dirtyWorkingTreeGuidance")
		if err != gitrepo.ErrDirtyWorkingTree {
			return guidance + "\n\n" + err.Error()
		}
		return guidance
	}
	if errors.Is(err, gitrepo.ErrUntrackedOverwrite) {
		guidance := a.t("untrackedOverwriteGuidance")
		if err != gitrepo.ErrUntrackedOverwrite {
			return guidance + "\n\n" + err.Error()
		}
		return guidance
	}
	var ignored *gitrepo.IgnoredFileError
	if errors.As(err, &ignored) {
		key := "ignoredFileGuidance"
		if ignored.WouldOverwrite {
			key = "ignoredFileOverwriteGuidance"
		}
		return a.tf(key, ignored.Path)
	}
	for _, target := range []struct {
		err error
		key string
	}{
		{gitrepo.ErrDirtySubmodule, "dirtySubmoduleGuidance"},
		{gitrepo.ErrSubmoduleTargetChanged, "submoduleTargetChanged"},
		{gitrepo.ErrForcePushTargetChanged, "forcePushTargetChanged"},
		{gitrepo.ErrPushTargetChanged, "pushTargetChanged"},
		{gitrepo.ErrPushUpstreamFailed, "pushUpstreamFailed"},
		{gitrepo.ErrRemoteDeleteTargetChanged, "remoteDeleteTargetChanged"},
		{gitrepo.ErrTagPushTargetChanged, "tagPushTargetChanged"},
		{gitrepo.ErrRemoteTargetChanged, "remoteTargetChanged"},
	} {
		if errors.Is(err, target.err) {
			guidance := a.t(target.key)
			if err != target.err {
				return guidance + "\n\n" + err.Error()
			}
			return guidance
		}
	}
	if errors.Is(err, gitrepo.ErrWorktreeTargetChanged) {
		return a.t("worktreeTargetChanged")
	}
	if errors.Is(err, gitrepo.ErrUndoTargetChanged) {
		guidance := a.t("undoTargetChanged")
		if err != gitrepo.ErrUndoTargetChanged {
			return guidance + "\n\n" + err.Error()
		}
		return guidance
	}
	if errors.Is(err, gitrepo.ErrIntegrationTargetChanged) {
		return a.t("integrationTargetChanged")
	}
	if errors.Is(err, gitrepo.ErrOperationTargetChanged) {
		return a.t("operationTargetChanged")
	}
	if errors.Is(err, gitrepo.ErrResetTargetChanged) {
		return a.t("resetTargetChanged")
	}
	if errors.Is(err, gitrepo.ErrRenameTargetChanged) {
		guidance := a.t("renameTargetChanged")
		if err != gitrepo.ErrRenameTargetChanged {
			return guidance + "\n\n" + err.Error()
		}
		return guidance
	}
	if errors.Is(err, gitrepo.ErrSwitchTargetChanged) {
		guidance := a.t("switchTargetChanged")
		if err != gitrepo.ErrSwitchTargetChanged {
			return guidance + "\n\n" + err.Error()
		}
		return guidance
	}
	if errors.Is(err, gitrepo.ErrBranchTargetChanged) {
		guidance := a.t("branchTargetChanged")
		if err != gitrepo.ErrBranchTargetChanged {
			return guidance + "\n\n" + err.Error()
		}
		return guidance
	}
	if errors.Is(err, gitrepo.ErrTagTargetChanged) {
		guidance := a.t("tagTargetChanged")
		if err != gitrepo.ErrTagTargetChanged {
			return guidance + "\n\n" + err.Error()
		}
		return guidance
	}
	var restore *gitrepo.StashApplyError
	if errors.As(err, &restore) {
		key := "stashOverlapGuidance"
		if restore.Conflict {
			key = "stashConflictGuidance"
		}
		return a.t(key) + "\n\n" + restore.Unwrap().Error()
	}
	return err.Error()
}
