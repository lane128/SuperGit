package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestHistoricalFilesRejectStaleSelectionCallbacks(t *testing.T) {
	for _, change := range []string{"repository", "root", "reload", "detail reload", "commit round trip", "worktree", "page", "busy", "loading", "history reload", "chooser", "reference", "dialog", "hidden details", "hidden files", "file loading", "file error", "removed file", "changed rename"} {
		t.Run(change, func(t *testing.T) {
			a, tt := historicalSelectionFixture(t)
			clickHistoricalFile(t, tt, "a.txt", 0)
			origin, file := a.historicalFileOrigin(), a.commitFiles[2]
			switch change {
			case "repository":
				a.repo = &gitrepo.Repository{Root: a.repo.Root}
			case "root":
				a.repo.Root = t.TempDir()
			case "reload":
				a.loadSeq++
			case "detail reload":
				a.commitSeq++
			case "commit round trip":
				files := a.commitFiles
				a.selectCommit("another-commit")
				a.selectCommit("selected-commit")
				a.commitFiles, a.commitFilesLoaded, a.commitFilesLoading = files, true, false
			case "worktree":
				a.selectWorktree()
			case "page":
				a.page = "changes"
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "history reload":
				a.historyReloading = true
			case "chooser":
				a.repositoryChooserOpen = true
			case "reference":
				a.historyRefOpen = true
			case "dialog":
				a.confirm.open = true
			case "hidden details":
				a.historyDetailsHidden = true
			case "hidden files":
				a.detailsChangesHidden = true
			case "file loading":
				a.commitFilesLoading = true
			case "file error":
				a.commitFilesErr = "owned read failure"
			case "removed file":
				a.commitFiles = a.commitFiles[:2]
			case "changed rename":
				a.commitFiles[2].OriginalPath = "different-source.txt"
			}
			path, tab := a.commitPath, a.detailTabs().Active
			beforeA, beforeC := a.historicalFileSelected("a.txt"), a.historicalFileSelected("c.txt")
			a.selectHistoricalFile(origin, file, ui.Cmd, false)
			if a.commitPath != path || a.detailTabs().Active != tab || a.historicalFileSelected("a.txt") != beforeA || a.historicalFileSelected("c.txt") != beforeC {
				t.Fatal("stale selection changed membership or preview")
			}
		})
	}
}

func TestHistoricalFilesEmptyRefreshAndRawRenamePaths(t *testing.T) {
	a, tt := historicalSelectionFixture(t)
	const renamed = "dir/作者 new.txt"
	files := []gitrepo.ChangedFile{{Path: "a.txt", Status: "A"}, {Path: renamed, OriginalPath: "old name.txt", Status: "R"}}
	a.commitFiles = files
	tt.Frame()
	clickHistoricalFile(t, tt, "a.txt", 0)
	clickHistoricalFile(t, tt, renamed, ui.Cmd)
	assertHistoricalFiles(t, tt, "a.txt", renamed)
	if list, ok := tt.AccessByRole(ui.RoleList, ""); !ok || !list.Multiselectable {
		t.Fatal("historical list is not accessible as multiselectable")
	}
	a.commitFiles = nil
	tt.Frame()
	if a.commitPath != "" || !a.detailTabs().Active.Summary {
		t.Fatal("empty refresh retained old preview")
	}
	a.commitFiles = files
	tt.Frame()
	assertHistoricalFiles(t, tt)
}

func TestHistoricalFilesContextCopyStaysSingleFile(t *testing.T) {
	a, tt := historicalSelectionFixture(t)
	previous := writeClipboard
	t.Cleanup(func() { writeClipboard = previous })
	var copied []string
	writeClipboard = func(value string) error { copied = append(copied, value); return nil }
	clickHistoricalFile(t, tt, "a.txt", 0)
	clickHistoricalFile(t, tt, "c.txt", ui.Cmd)
	row, ok := tt.AccessByRole(ui.RoleOption, "c.txt")
	if !ok {
		t.Fatal("file row missing")
	}
	tt.RightClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	item, ok := tt.AccessByRole(ui.RoleMenuItem, a.t("copyPath"))
	if !ok {
		t.Fatal("single-file copy action missing")
	}
	tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
	if len(copied) != 1 || copied[0] != "c.txt" {
		t.Fatalf("single-file menu copied collection: %q", copied)
	}
	assertHistoricalFiles(t, tt, "a.txt", "c.txt")
}

func TestHistoricalFilesRefreshDoesNotLeaveSummary(t *testing.T) {
	a, tt := historicalSelectionFixture(t)
	clickHistoricalFile(t, tt, "a.txt", 0)
	clickHistoricalFile(t, tt, "c.txt", ui.Cmd)
	a.activateDetailTab(detailTabKey{Summary: true})
	a.commitFiles = a.commitFiles[:2]
	tt.Frame()
	assertHistoricalFiles(t, tt, "a.txt")
	if !a.detailTabs().Active.Summary {
		t.Fatal("refresh reopened a file while the reader was on Summary")
	}
}
