package lobedesk

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestOwnedRenamedCommitFileHistoryMenuUsesParsedNewPath(t *testing.T) {
	a, _, root := worktreeApp(t)
	a.locale = localeEnglish
	destination := "folder/renamed\t文件.txt"
	if err := os.Mkdir(filepath.Join(root, "folder"), 0700); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("mv", "--", "base.txt", destination)
	git("commit", "-m", "rename file")
	sha := git("rev-parse", "HEAD")
	files, err := a.repo.CommitFiles(context.Background(), sha)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Path != destination || files[0].OriginalPath != "base.txt" || !strings.HasPrefix(files[0].Status, "R") {
		t.Fatalf("incorrect owned rename parse: %+v", files)
	}
	// This assertion covers parsed row -> actual menu -> file-history scope;
	// separate owned backend tests cover worker completion and traversal.
	a.ctx = canceledContext()
	a.runOnMain = func(func()) {}
	a.selectedSHA, a.commitFilesLoaded, a.commitFiles = sha, true, files
	t.Cleanup(a.resetFileHistory)
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(500).Height(250).Children(func() { a.commitFileList(c) })
	}, 500, 250)
	row, ok := tt.AccessByRole(ui.RoleOption, destination)
	if !ok {
		t.Fatal("missing parsed rename row")
	}
	tt.RightClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	item, ok := tt.AccessByRole(ui.RoleMenuItem, a.t("fileHistory"))
	if !ok || item.Disabled {
		t.Fatal("rename row has no enabled file history action")
	}
	tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
	tt.Frame()
	if a.fileFilter != destination || a.selectedSHA != "" || len(a.commitFiles) != 0 {
		t.Fatal("rename menu did not enter parsed new-path scope")
	}
}
