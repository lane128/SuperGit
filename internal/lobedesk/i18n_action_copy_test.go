package lobedesk

import (
	"strings"
	"testing"
)

// One action reaches the reader under several names: the menu item that opens
// its dialog, the dialog title, the confirm button, the busy label in the title
// bar and the failure toast. Each name is its own i18n key, so a reworded menu
// could say 丢弃 while the toast says 放弃, and nothing failed until a reader
// noticed. These groups pin one Chinese verb per action, and one English stem
// for its busy and failure copies. Menu/title pairs are checked separately by
// TestActionLabelsCarryTheirEllipsisInI18n; generic confirm buttons (保存、删除、
// 创建) are shared by several dialogs and stay out of the groups.
//
// Adding a busy label or a failure text means adding it here, beside the rest of
// its action. The coverage check at the bottom of the test refuses an unlisted
// one, and actionCopyExempt records the failures that have no action copies.
type actionCopies struct {
	action string
	zh     string // verb every Chinese copy contains
	en     string // word stem the English busy*/Failed copies start with; "" skips it
	keys   []string
}

var actionCopyGroups = []actionCopies{
	{"save repository alias", "保存", "Sav", []string{"saveRepositoryAlias", "saveRepositoryAliasFailed"}},
	{"jump to message commit", "跳转", "Jump", []string{"jumpMessageCommit", "jumpMessageCommitFailed"}},
	{"open message link", "打开", "Open", []string{"openMessageLinkFailed"}},
	{"check file undo", "检查", "Check", []string{"busyCheckUndoFile"}},
	{"undo file changes", "撤销", "Undo", []string{"undoFile", "undoFileTitle", "undoFileAction", "undoFileFailed", "busyUndoFile"}},
	{"export file patch", "导出", "Export", []string{
		"exportFilePatch", "exportFilePatchTitle", "exportFilePatchFailed", "busyExportFilePatch",
	}},
	{"create repository", "创建", "Creat", []string{
		"newRepo", "newRepoTitle", "busyCreateRepo", "createdOpenFailed",
	}},
	{"clone", "克隆", "Clon", []string{
		"cloneRepo", "cloneRepoTitle", "cloneAction", "busyClone", "clonedOpenFailed",
	}},
	{"commit", "提交", "Comm", []string{
		"commit", "commitChanges", "commitTo", "commitFailed", "busyCommit",
	}},
	{"stage", "暂存", "Stag", []string{
		"stageAll", "stageTracked", "busyStageTracked", "stageFile", "stageHunk", "stageLine", "stageLines",
		"stageFailed", "stageHunkFailed", "stageLineFailed", "stageLinesFailed", "busyStageAll",
	}},
	{"unstage", "取消暂存", "Unstag", []string{
		"unstageAll", "unstageFile", "unstageHunk", "unstageLine", "unstageLines",
		"unstageFailed", "unstageHunkFailed", "unstageLineFailed", "unstageLinesFailed", "busyUnstageAll",
	}},
	{"discard", "放弃", "Discard", []string{
		"discard", "discardChanges", "discardTitle", "discardLine", "discardLines",
		"discardLinesTitle", "discardHunk", "discardHunkTitle", "discardFailed",
		"discardLinesFailed", "undoDiscardFailed", "busyDiscard",
	}},
	{"delete file", "删除", "Delet", []string{
		"deleteFile", "deleteFailed", "busyDeleteFile",
	}},
	{"save stash", "保存", "Sav", []string{
		"saveStash", "saveStashAction", "saveStashTitle", "saveStashUntracked",
		"saveStashFailed", "busySaveStash",
	}},
	{"stash file", "存入", "", []string{
		"stashFile", "stashFileAction",
	}},
	{"drop stash", "删除", "Drop", []string{
		"dropStash", "dropStashAction", "dropStashTitle", "dropStashFailed", "busyDropStash",
	}},
	{"apply stash", "应用", "Appl", []string{
		"applyStash", "applyStashFailed", "busyApplyStash",
	}},
	{"pop stash", "弹出", "Pop", []string{
		"popStash", "popStashFailed", "busyPopStash",
	}},
	{"push", "推送", "Push", []string{
		"pushButton", "pushBranchAction", "pushDialogTitle", "pushFailed", "pushUpstreamFailed", "busyPush",
	}},
	{"force push", "强制推送", "Force", []string{
		"forcePush", "forcePushAction", "forcePushTitle", "forcePushFailed", "busyForcePush",
	}},
	{"pull", "拉取", "Pull", []string{
		"pull", "pullFailed", "busyPull",
	}},
	{"add remote", "添加", "Add", []string{"addRemoteEntry", "addRemote", "addRemoteFailed", "remoteAdded"}},
	{"fetch", "获取", "Fetch", []string{
		"fetch", "fetchPrune", "fetchFailed", "fetchPruneFailed", "fetchingNow",
	}},
	{"merge", "合并", "Merg", []string{
		"mergeIntoCurrent", "mergeIntoCurrentAction", "mergeCommitInto", "mergeFailed", "busyMerge",
	}},
	{"rebase", "变基", "Rebas", []string{
		"rebaseOnto", "rebaseOntoAction", "rebaseBranchOnto", "rebaseFailed", "busyRebase",
	}},
	{"cherry-pick", "拣选", "Cherr", []string{
		"cherryPick", "cherryPickTitle", "cherryPickAction", "cherryPickFailed", "busyCherryPick",
	}},
	{"revert", "回退", "Rever", []string{
		"revert", "revertTitle", "revertAction", "revertFailed", "busyRevert",
	}},
	{"reset", "重置", "Reset", []string{
		"resetAction", "resetTitle", "resetBranchToCommit", "resetToUpstream",
		"resetFailed", "busyReset",
	}},
	{"checkout", "检出", "Chec", []string{
		"checkout", "checkoutCommit", "checkoutCommitTitle", "checkoutRemoteBranch",
		"checkoutRemoteBranchTitle", "checkoutCommitFailed", "checkoutRemoteBranchFailed",
		"busyCheckout",
	}},
	{"switch branch", "切换", "Switch", []string{
		"switch", "switchToBranch", "switchFailed", "busySwitch",
	}},
	{"create worktree", "创建", "Creat", []string{
		"newWorktree", "newWorktreeTitle", "createWorktreeFailed", "busyCreateWorktree",
	}},
	{"remove worktree", "移除", "Remov", []string{
		"removeWorktree", "removeWorktreeTitle", "removeWorktreeFailed", "busyRemoveWorktree",
	}},
	{"update submodule", "更新", "Updat", []string{
		"updateSubmodule", "updateSubmoduleFailed", "busySubmodule",
	}},
	{"create tag", "创建", "Creat", []string{
		"createTag", "createTagTitle", "createAnnotatedTag", "createAnnotatedTagTitle",
		"createTagFailed", "createAnnotatedTagFailed",
	}},
	{"delete tag", "删除", "Delet", []string{
		"deleteTag", "deleteTagAction", "deleteTagTitle", "deleteTagFailed",
	}},
	{"push tag", "推送", "Push", []string{
		"pushTag", "pushTagTitle", "pushTagFailed", "busyPushTag",
	}},
	{"create branch", "创建", "Creat", []string{
		"newBranchTitle", "newBranchAction", "newBranchNoCheckout",
		"newBranchNoCheckoutAction", "createBranchFromTag", "createBranchFailed",
	}},
	{"rename branch", "重命名", "Renam", []string{
		"renameBranch", "renameBranchTitle", "renameBranchFailed",
	}},
	{"delete branch", "删除", "Delet", []string{
		"deleteBranch", "deleteBranchTitle", "deleteBranchFailed",
	}},
	{"force delete branch", "强制删除", "Force", []string{
		"forceDeleteBranch", "forceDeleteBranchAction", "forceDeleteTitle",
		"forceDelete", "forceDeleteFailed",
	}},
	{"delete remote branch", "删除", "Delet", []string{
		"deleteRemoteBranch", "deleteRemoteBranchTitle", "deleteRemoteBranchFailed",
		"busyDeleteRemoteBranch",
	}},
	{"undo commit", "撤销", "Undo", []string{
		"undoCommit", "undoCommitTitle", "undo", "undoFailed", "busyUndoCommit",
	}},
	{"continue", "继续", "Contin", []string{
		"continueOperation", "continueFailed", "busyContinue",
	}},
	{"abort", "中止", "Abort", []string{
		"abortOperation", "abortTitle", "abortFailed", "busyAbort",
	}},
	{"skip", "跳过", "Skip", []string{
		"skipOperation", "skipTitle", "skipFailed", "busySkip",
	}},
	{"reset settings", "恢复", "", []string{
		"resetSettings", "resetSettingsAction", "resetSettingsTitle", "resetSettingsDone",
	}},
	{"install update", "更新", "update", []string{
		"updateRestartAndInstall", "updateApplyFailed",
	}},
}

// Failures that have no action copies to agree with. Keep the reason next to
// the key: a failure that gains a busy label later should leave this list.
var actionCopyExempt = map[string]string{
	"commitIdentityFailed":        "a background identity read failure, not a commit action failure",
	"chooseParentDirFailed":       "the native folder picker has no busy label or menu item",
	"diffFailed":                  "a read failure, not a user action",
	"historyLoadFailed":           "a read failure, not a user action",
	"localWatchFailed":            "background local-change detection has no user action or busy label",
	"openFailed":                  "a read failure, not a user action",
	"reloadFailed":                "a read failure, not a user action",
	"resetUpstreamFailed":         "a read failure, not a user action",
	"switchRepoFailed":            "a read failure, not a user action",
	"stashesReadFailed":           "a read failure, not a user action",
	"commitsReadFailed":           "a read failure, not a user action",
	"settingsSaveFailed":          "settings save on close has no menu, title or button copy",
	"copyFailed":                  "the system clipboard has no busy label",
	"resolveFailed":               "conflict resolution has no one verb: resolve vs take a side",
	"takeSideFailed":              "conflict resolution has no one verb: resolve vs take a side",
	"updateOpenReleasePageFailed": "external browser URL launch has no busy label",
}

func TestEveryCopyOfAnActionUsesOneVerb(t *testing.T) {
	zh, en := messages[localeChinese], messages[localeEnglish]
	seen := map[string]string{}
	for _, group := range actionCopyGroups {
		for _, key := range group.keys {
			if previous, ok := seen[key]; ok {
				t.Errorf("%q is grouped under both %q and %q", key, previous, group.action)
			}
			seen[key] = group.action
			chinese, ok := zh[key]
			if !ok {
				t.Errorf("%s: zh is missing %q", group.action, key)
				continue
			}
			english, ok := en[key]
			if !ok {
				t.Errorf("%s: en is missing %q", group.action, key)
				continue
			}
			if !strings.Contains(chinese, group.zh) {
				t.Errorf("%s: zh %q = %q does not use %q", group.action, key, chinese, group.zh)
			}
			if group.en == "" {
				continue
			}
			if strings.HasPrefix(key, "busy") || strings.HasSuffix(key, "Failed") {
				if !startsEnglishWord(english, group.en) {
					t.Errorf("%s: en %q = %q does not name the action with %q", group.action, key, english, group.en)
				}
			}
		}
	}
	for key := range zh {
		if strings.HasPrefix(key, "busy") && key != "busyNow" {
			if _, ok := seen[key]; !ok {
				t.Errorf("busy label %q has no action group; add it beside its failure text", key)
			}
		}
		if strings.HasSuffix(key, "Failed") {
			if _, ok := seen[key]; !ok && actionCopyExempt[key] == "" {
				t.Errorf("failure %q has no action group; add it or record why in actionCopyExempt", key)
			}
		}
	}
	for key := range actionCopyExempt {
		if _, ok := zh[key]; !ok {
			t.Errorf("actionCopyExempt names %q, which is not a zh key", key)
		}
	}
}

// startsEnglishWord reports whether stem begins a word in text, so "Push"
// matches "Pushing…" and "Push failed" but not "Pushed away".
func startsEnglishWord(text, stem string) bool {
	text, stem = strings.ToLower(text), strings.ToLower(stem)
	for i := 0; i+len(stem) <= len(text); i++ {
		if !strings.HasPrefix(text[i:], stem) {
			continue
		}
		if i == 0 || !isASCIILetter(text[i-1]) {
			return true
		}
	}
	return false
}

func isASCIILetter(b byte) bool {
	return b >= 'a' && b <= 'z'
}
