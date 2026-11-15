package lobedesk

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// A file row is a button so it keeps Tab, Shift+F10 and the accessibility
// role, but a focused button takes Enter and Space for itself, which would
// only reselect the row. The row's shortcuts claim both keys first and run
// the page's actions on the current selection.
func TestFileRowShortcutsTakeEnterAndSpace(t *testing.T) {
	root := t.TempDir()
	sentinel := filepath.Join(t.TempDir(), "opened")
	bin := t.TempDir()
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$SG_OPEN_SENTINEL\"\n"
	if err := os.WriteFile(filepath.Join(bin, "open"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SG_OPEN_SENTINEL", sentinel)

	a := &app{locale: localeChinese, page: "history", worktreeSelected: true}
	a.repo = &gitrepo.Repository{Root: root}
	a.ctx = context.Background()
	rows := []gitrepo.Entry{
		{Path: "dir/a.go", Worktree: 'M'},
		{Path: "dir/b.go", Worktree: 'M'},
	}
	a.status = gitrepo.Status{Entries: rows}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		// The page's arrow keys move the selection while the row keeps the
		// focus, as the real view does.
		c.OnWindowShortcut(0, ui.KeyDown, func() { a.selectFile("dir/b.go", false) })
		ui.Column(c).Width(600).Height(400).Children(func() {
			for _, entry := range rows {
				a.fileRow(c, entry, false)
			}
		})
	}, 600, 400)
	r, ok := tt.Find("a.go")
	if !ok {
		t.Fatal("the file row is not laid out")
	}
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	if a.selectedPath != "dir/a.go" {
		t.Fatalf("clicking the file row selected %q", a.selectedPath)
	}
	// The selection moves off the focused row, so a button activation would
	// pull it back; the row shortcut must run the page's action on the new
	// selection instead.
	tt.Key(0, ui.KeyDown)
	if a.selectedPath != "dir/b.go" {
		t.Fatalf("the arrow key selected %q", a.selectedPath)
	}

	tt.Key(0, ui.KeyEnter)
	deadline := time.Now().Add(2 * time.Second)
	for {
		data, err := os.ReadFile(sentinel)
		if err == nil {
			if !strings.Contains(string(data), filepath.Join("dir", "b.go")) {
				t.Errorf("Enter opened %q; want the selected file dir/b.go", strings.TrimSpace(string(data)))
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("Enter did not open the selected file")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if a.selectedPath != "dir/b.go" {
		t.Errorf("Enter activated the focused row button and selected %q; want the page's selection", a.selectedPath)
	}

	tt.Key(0, ui.KeySpace)
	if !a.busy {
		t.Fatal("Space did not start staging the selected file")
	}
	if a.selectedPath != "dir/b.go" {
		t.Errorf("Space activated the focused row button and selected %q; want the page's selection", a.selectedPath)
	}
}
