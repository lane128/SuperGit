package lobedesk

import (
	"testing"

	"supergit/internal/gitrepo"
)

func TestConflictPreviewExplainsExternalResolutionAndDeletion(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		for _, code := range []string{"UU", "UD", "DU", "DD"} {
			for _, overview := range []bool{false, true} {
				name := string(language) + "/" + code + "/file"
				if overview {
					name = string(language) + "/" + code + "/overview"
				}
				t.Run(name, func(t *testing.T) {
					entry := gitrepo.Entry{Path: "conflict.bin", Conflict: true, Index: code[0], Worktree: code[1]}
					patch := gitrepo.Diff{Text: "unmerged preview", Lines: []gitrepo.DiffLine{{Kind: "meta", Text: "unmerged preview"}}}
					a := &app{locale: language, selectedPath: entry.Path, diff: patch, status: gitrepo.Status{Entries: []gitrepo.Entry{entry}}}
					page := a.diffBody
					if overview {
						a.overviewEntries = []worktreeOverviewEntry{{Key: detailFileKey{Path: entry.Path}, Diff: patch}}
						page = a.worktreeOverview
					}
					tt := pageTester(a, page)
					general := "Compare and edit text conflicts here. For unsupported files, edit and save with an external tool, then use Mark resolved to stage the contents. Finish the pending operation separately."
					deletion := "This conflict includes a deletion. Choosing a side with no file deletes and stages this path."
					if language == localeChinese {
						general = "文本冲突可在这里比较并编辑。不支持的文件请用外部工具保存，再选择「标记为已解决」暂存内容。进行中的操作仍需单独完成。"
						deletion = "此冲突包含删除。选择不存在文件的一侧，会删除并暂存此路径。"
					}
					if _, ok := tt.Find(general); !ok {
						t.Fatal("conflict preview hides the external resolution path for text and binary files")
					}
					_, hasDeletion := tt.Find(deletion)
					if hasDeletion != (code != "UU") {
						t.Fatalf("deletion consequence shown=%v for %s", hasDeletion, code)
					}
					if _, ok := tt.Find("unmerged preview"); !ok {
						t.Fatal("conflict guidance hid the original preview")
					}
				})
			}
		}
	}
}

func TestConflictOverviewGuidanceUsesMeasuredLayoutHeight(t *testing.T) {
	key := detailFileKey{Path: "conflict.txt"}
	a := &app{status: gitrepo.Status{Entries: []gitrepo.Entry{{Path: key.Path, Conflict: true}}}}
	entry := worktreeOverviewEntry{Key: key, Diff: gitrepo.Diff{Lines: []gitrepo.DiffLine{{Kind: "meta", Text: "preview"}}}}
	if _, exact := a.overviewSectionHeight(entry); exact {
		t.Fatal("wrapping conflict guidance was assigned a fixed patch-only height")
	}
	a.detailContexts.Worktree.Collapsed = map[detailFileKey]bool{key: true}
	if height, exact := a.overviewSectionHeight(entry); !exact || height != 35 {
		t.Fatal("collapsed conflict header lost its exact extent")
	}
}
