package lobedesk

import (
	"context"
	"maps"
	"slices"
	"strings"

	"supergit/internal/gitrepo"
)

func init() { registerPaletteActions(historyViewPaletteActions()...) }

func historyViewPaletteActions() []paletteAction {
	return []paletteAction{
		{ID: "view.toggleSidebar", English: "View: Toggle Side Bar", Chinese: "视图：切换侧边栏", Global: true, Run: func(a *app) { a.sidebarHidden = !a.sidebarHidden; a.persistSettings(); a.invalidate() }},
		{ID: "file.blame", English: "Blame…", Chinese: "逐行追溯…", Unavailable: paletteSnapshotUnavailable, Run: func(a *app) { a.openSnapshotFilePicker("blame") }},
		{ID: "file.history", English: "File History…", Chinese: "文件历史…", Unavailable: paletteSnapshotUnavailable, Run: func(a *app) { a.openSnapshotFilePicker("history") }},
		{ID: "commit.viewTree", English: "View Tree", Chinese: "查看文件树", Unavailable: paletteSnapshotUnavailable, Run: func(a *app) { a.openSnapshotFilePicker("tree") }},
		{ID: "history.navigateCommit", English: "Navigate to Commit…", Chinese: "跳转到提交…", Unavailable: paletteSnapshotUnavailable, Run: (*app).chooseHistoryCommit},
		{ID: "history.navigateBranch", English: "Navigate to Branch…", Chinese: "跳转到分支…", Unavailable: func(a *app) string { return a.historyBranchesUnavailable("navigate") }, Run: func(a *app) { a.chooseHistoryBranch("navigate") }},
		{ID: "history.navigateTag", English: "Navigate to Tag…", Chinese: "跳转到标签…", Unavailable: func(a *app) string {
			if a.tagsErr != "" {
				return a.tagsErr
			}
			for _, tag := range a.tags {
				if tag.TargetType == "commit" {
					return ""
				}
			}
			return a.paletteText("No tags pointing to commits", "没有指向提交的标签")
		}, Run: (*app).chooseHistoryTag},
		{ID: "history.navigateStash", English: "Navigate to Stash…", Chinese: "跳转到 Stash…", Unavailable: func(a *app) string {
			if a.stashesErr != "" {
				return a.stashesErr
			}
			if len(a.stashes) == 0 {
				return a.t("noStashes")
			}
			return ""
		}, Run: (*app).chooseHistoryStash},
		{ID: "diff.ignoreWhitespace", English: "Ignore Whitespace in Diffs: Toggle", Chinese: "差异：切换忽略空白", Global: true, Run: func(a *app) { a.setIgnoreDiffWhitespace(!a.ignoreDiffWhitespace); a.persistSettings(); a.invalidate() }},
		{ID: "history.expandMerges", English: "Expand Merge Commits By Default: Toggle", Chinese: "切换默认展开合并提交", Global: true, Run: func(a *app) {
			a.expandMerges = !a.expandMerges
			a.historyRowsSet = false
			a.persistSettings()
			a.invalidate()
		}},
		{ID: "view.selectTheme", English: "Select Theme", Chinese: "选择主题", Global: true, Run: (*app).chooseHistoryTheme},
		{ID: "history.hideBranch", English: "Hide Branch…", Chinese: "隐藏分支…", Unavailable: func(a *app) string { return a.historyBranchesUnavailable("hide") }, Run: func(a *app) { a.chooseHistoryBranch("hide") }},
		{ID: "history.showBranch", English: "Show Hidden Branch…", Chinese: "显示已隐藏的分支…", Unavailable: func(a *app) string { return a.historyBranchesUnavailable("show") }, Run: func(a *app) { a.chooseHistoryBranch("show") }},
		{ID: "history.hideExcept", English: "Hide All Branches Except…", Chinese: "仅显示所选分支…", Unavailable: func(a *app) string { return a.historyBranchesUnavailable("except") }, Run: func(a *app) { a.chooseHistoryBranch("except") }},
		{ID: "history.showAllBranches", English: "Show All Hidden Branches", Chinese: "显示所有已隐藏的分支", Unavailable: func(a *app) string {
			if a.repo == nil || len(a.hiddenBranches[a.repo.Root]) == 0 {
				return a.paletteText("No hidden branches", "没有已隐藏的分支")
			}
			return ""
		}, Run: func(a *app) { a.setHiddenBranches(nil) }},
	}
}

func (a *app) setIgnoreDiffWhitespace(ignore bool) {
	if a.ignoreDiffWhitespace == ignore {
		return
	}
	a.ignoreDiffWhitespace = ignore
	a.diffCache = worktreeDiffCache{}
	a.invalidateWorktreeOverview()
	a.clearLineSelection()
	a.clearCommitSelection()
	a.diff = gitrepo.Diff{}
	a.loadDiff()
	if a.worktreeSelected {
		a.loadWorktreeOverview()
	} else if a.selectedStash != "" {
		a.loadStashDiff()
	} else {
		a.loadCommitDetail(true)
	}
}

func (a *app) chooseHistoryTheme() {
	choices := make([]paletteChoice, 0, 3+len(appThemes))
	for _, mode := range []struct{ key, label string }{{"system", "themeSystem"}, {"light", "themeLight"}, {"dark", "themeDark"}} {
		row := paletteChoice{id: "appearance." + mode.key, label: a.t(mode.label), detail: a.paletteText("Appearance", "外观")}
		if mode.key == a.themeMode {
			row.prefix = "✓"
		}
		choices = append(choices, row)
	}
	for _, th := range appThemes {
		row := paletteChoice{id: th.key, label: a.t(th.label), detail: a.t("palette")}
		if th.key == a.palette {
			row.prefix = "✓"
		}
		choices = append(choices, row)
	}
	title := a.paletteText("Select Theme", "选择主题")
	a.choosePalette(title, title, choices, func(row paletteChoice) {
		if mode, ok := strings.CutPrefix(row.id, "appearance."); ok {
			a.themeMode = mode
		} else {
			a.palette = row.id
		}
		a.persistSettings()
		a.invalidate()
	})
}

func (a *app) branchHidden(fullRef string) bool {
	if a.repo == nil {
		return false
	}
	_, found := slices.BinarySearch(a.hiddenBranches[a.repo.Root], fullRef)
	return found
}

func (a *app) visibleBranchCount(branches []gitrepo.Branch, prefix string) int {
	if a.repo == nil || len(a.hiddenBranches[a.repo.Root]) == 0 {
		return len(branches)
	}
	n := 0
	for _, b := range branches {
		if !a.branchHidden(prefix + b.Name) {
			n++
		}
	}
	return n
}

func (a *app) historyBranchesUnavailable(mode string) string {
	if a.branchesErr != "" {
		return a.branchesErr
	}
	if a.remoteBranchesErr != "" {
		return a.remoteBranchesErr
	}
	total := len(a.branches) + len(a.remoteBranches)
	if total == 0 {
		return a.paletteText("No branches", "没有分支")
	}
	if mode == "except" {
		return ""
	}
	visible := a.visibleBranchCount(a.branches, "refs/heads/") + a.visibleBranchCount(a.remoteBranches, "refs/remotes/")
	if mode == "show" && visible == total {
		return a.paletteText("No hidden branches", "没有已隐藏的分支")
	}
	if mode != "show" && visible == 0 {
		return a.paletteText("No visible branches", "没有可见分支")
	}
	return ""
}

func (a *app) historyBranchChoices(mode string) []paletteChoice {
	var rows []paletteChoice
	add := func(branches []gitrepo.Branch, prefix, badge string) {
		for _, b := range branches {
			id := prefix + b.Name
			if mode != "except" && a.branchHidden(id) != (mode == "show") {
				continue
			}
			rows = append(rows, paletteChoice{id: id, label: b.Name, sha: b.SHA, detail: b.Commit.Subject, prefix: badge})
		}
	}
	add(a.branches, "refs/heads/", "L")
	add(a.remoteBranches, "refs/remotes/", "R")
	return rows
}

func (a *app) chooseHistoryBranch(mode string) {
	choices := a.historyBranchChoices(mode)
	title := a.paletteText("Navigate to Branch", "跳转到分支")
	switch mode {
	case "hide":
		title = a.paletteText("Hide Branch", "隐藏分支")
	case "show":
		title = a.paletteText("Show Hidden Branch", "显示已隐藏的分支")
	case "except":
		title = a.paletteText("Hide All Branches Except", "仅显示所选分支")
	}
	a.choosePalette(title, a.t("branchName"), choices, func(row paletteChoice) {
		if mode == "navigate" {
			a.openPage("history")
			a.showCommit(row.sha)
			return
		}
		hidden := slices.Clone(a.hiddenBranches[a.repo.Root])
		switch mode {
		case "hide":
			hidden = append(hidden, row.id)
		case "show":
			hidden = slices.DeleteFunc(hidden, func(ref string) bool { return ref == row.id })
		case "except":
			hidden = nil
			for _, other := range choices {
				if other.id != row.id {
					hidden = append(hidden, other.id)
				}
			}
		}
		a.setHiddenBranches(hidden)
	})
}

func (a *app) syncHiddenBranches(roots map[string][]string) {
	previous := a.hiddenBranches
	a.hiddenBranches = roots
	if a.repo != nil && !slices.Equal(previous[a.repo.Root], roots[a.repo.Root]) && a.historyRef.Kind == "" {
		// Reload also invalidates the old load generation if a refresh was pending.
		a.reload()
	}
}

func (a *app) setHiddenBranches(hidden []string) {
	if a.repo == nil {
		return
	}
	roots := maps.Clone(a.hiddenBranches)
	if roots == nil {
		roots = make(map[string][]string)
	}
	if len(hidden) == 0 {
		delete(roots, a.repo.Root)
	} else {
		slices.Sort(hidden)
		roots[a.repo.Root] = slices.Compact(hidden)
	}
	a.hiddenBranches = roots
	a.openPage("history")
	a.applyHistoryRef(historyRef{})
	a.persistSettings()
}

func (a *app) chooseHistoryCommit() {
	// An explicit revision can reach a commit beyond the pages read so far.
	rows := []paletteChoice{{id: "revision", label: a.paletteText("Enter commit hash or revision…", "输入提交哈希或版本…")}}
	for _, c := range a.commits {
		rows = append(rows, paletteChoice{id: c.SHA, sha: c.SHA, label: c.Subject, detail: shortSHA(c.SHA) + " · " + c.Author})
	}
	title := a.paletteText("Navigate to Commit", "跳转到提交")
	a.choosePalette(title, a.paletteText("Search commits", "搜索提交"), rows, func(row paletteChoice) {
		if row.sha != "" {
			a.openPage("history")
			a.showCommit(row.sha)
			return
		}
		a.inputPalette(title, a.paletteText("Commit hash or revision", "提交哈希或版本"), "", func(ref string) {
			readHistoryPalette(a, title, func(ctx context.Context, r *gitrepo.Repository) (string, error) { return r.ResolveRef(ctx, ref) }, func(sha string) {
				a.openPage("history")
				if historyLists(a.historySource(), sha) {
					a.showCommit(sha)
					return
				}
				a.applyHistoryRef(historyRef{Kind: historyRefCommit, Name: sha})
				a.pendingJumpSHA = sha
			})
		})
	})
}

func (a *app) chooseHistoryTag() {
	var rows []paletteChoice
	for _, tag := range a.tags {
		if tag.TargetType == "commit" {
			rows = append(rows, paletteChoice{id: tag.Name, label: tag.Name, sha: tag.TargetSHA, detail: tag.Commit.Subject})
		}
	}
	a.choosePalette(a.paletteText("Navigate to Tag", "跳转到标签"), a.t("tagName"), rows, func(row paletteChoice) { a.openPage("history"); a.showCommit(row.sha) })
}
func (a *app) chooseHistoryStash() {
	var rows []paletteChoice
	for _, s := range a.stashes {
		rows = append(rows, paletteChoice{id: s.Ref, label: s.Ref, sha: s.SHA, detail: s.Message})
	}
	a.choosePalette(a.paletteText("Navigate to Stash", "跳转到 Stash"), a.t("stash"), rows, func(row paletteChoice) {
		a.openPage("stash")
		a.revealSelection = true
		a.selectStash(row.sha)
		a.invalidate()
	})
}
