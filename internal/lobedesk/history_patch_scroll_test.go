package lobedesk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestHistoryPatchKeepsEachFilesHorizontalPosition(t *testing.T) {
	a, _, root := worktreeApp(t)
	a.locale, a.page = localeEnglish, "history"
	callbacks := make(chan func(), 128)
	a.runOnMain = func(fn func()) { callbacks <- fn }
	t.Cleanup(func() { a.resetHistoryTraversal(); a.resetFileHistory() })

	longText := strings.Repeat("abcdefghij\t中文", 400) + " END_OF_LINE"
	for path, text := range map[string]string{"long.txt": longText, "short.txt": "short review content"} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(text+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	splitCommitGit(t, root, "add", "--", "long.txt", "short.txt")
	const subject = "Review long and short files"
	splitCommitGit(t, root, "commit", "-m", subject)
	head := splitCommitGit(t, root, "rev-parse", "HEAD")
	index := splitCommitGit(t, root, "ls-files", "--stage")

	a.reload()
	view := ui.NewTester(a.root, 980, 760)
	wait := func() {
		t.Helper()
		for a.loading || a.commitFilesLoading || a.commitDiffLoading {
			runContinuation(t, callbacks)
			view.Frame()
		}
		view.Frame()
	}
	click := func(role ui.Role, name string) {
		t.Helper()
		clickRepositoryTabsControl(t, view, role, name)
		wait()
	}
	const shortCode = "+short review content"
	longCode := "+" + strings.ReplaceAll(longText, "\t", "    ")
	wait()
	click(ui.RoleOption, subject)
	click(ui.RoleOption, "long.txt")

	var textWidth float32
	ui.NewTester(func(c *ui.Context) {
		textWidth, _ = c.MeasureText(0, ui.Span{Text: longCode, Font: a.patchFontFamily(), Size: scaledFont(diffCodeSize)})
	}, 560, 100)
	for _, width := range []int{980, 1600} {
		view.SetSize(width, 760)
		view.Frame()
		longRow, ok := view.AccessByRole(ui.RoleInsertion, longCode)
		if !ok {
			t.Fatal("long code line not in the patch")
		}
		pane := a.commitDiffViewport
		x := pane.X + pane.W/2
		// Use the painted file header, outside the selectable text surface.
		fileHeader, ok := view.AccessByRole(ui.RoleButton, "long.txt")
		if !ok {
			t.Fatal("missing long-file patch header")
		}
		view.Scroll(x, fileHeader.Bounds.Y+fileHeader.Bounds.H/2, 1e6, 0)
		view.Frame()
		far, ok := view.AccessByRole(ui.RoleInsertion, longCode)
		tail := far.Bounds.X + textWidth
		if !ok || tail <= pane.X || tail > pane.X+pane.W {
			tabs := a.detailTabs()
			t.Errorf("width %d: tail %.1f outside %+v; row before=%+v after=%+v scroll=%+v", width, tail, pane, longRow.Bounds, far.Bounds, tabs.PatchScroll[tabs.Active])
		}

		click(ui.RoleOption, "short.txt")
		short, ok := view.AccessByRole(ui.RoleInsertion, shortCode)
		if !ok || short.Offscreen || short.Bounds.X < pane.X || short.Bounds.X+short.Bounds.W > pane.X+pane.W {
			t.Errorf("width %d: short file opens outside its viewport: %+v", width, short)
		}
		view.Scroll(x, short.Bounds.Y+short.Bounds.H/2, 1e6, 0)
		view.Frame()
		if moved, ok := view.AccessByRole(ui.RoleInsertion, shortCode); !ok || moved.Bounds.X != short.Bounds.X {
			t.Errorf("width %d: short patch scrolls sideways despite fitting", width)
		}
		click(ui.RoleOption, "long.txt")
		if again, ok := view.AccessByRole(ui.RoleInsertion, longCode); !ok || again.Bounds.X != far.Bounds.X {
			t.Errorf("width %d: long file lost its previous horizontal position", width)
		}
	}
	if a.selectedSHA != strings.TrimSpace(head) || splitCommitGit(t, root, "rev-parse", "HEAD") != head || splitCommitGit(t, root, "ls-files", "--stage") != index || splitCommitGit(t, root, "status", "--porcelain=v1") != "" {
		t.Fatal("reading or resizing History changed the selected commit or repository")
	}
}
