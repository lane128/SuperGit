package lobedesk

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestConflictEditorWorkbenchJourney(t *testing.T) {
	for _, size := range [][2]float32{{900, 650}, {900, 900}, {1280, 900}} {
		width, height := size[0], size[1]
		t.Run(fmt.Sprintf("%dx%d", int(width), int(height)), func(t *testing.T) {
			a, done, root := conflictEditorFixture(t)
			a.page = "changes"
			t.Cleanup(func() {
				a.resetHistoryTraversal()
				a.resetFileHistory()
				if a.overviewCancel != nil {
					a.overviewCancel()
				}
			})
			local := strings.TrimSpace(splitCommitGit(t, root, "rev-parse", "HEAD"))
			incoming := strings.TrimSpace(splitCommitGit(t, root, "rev-parse", "MERGE_HEAD"))
			a.reload()
			tt := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				ui.Column(c).Fill().Children(func() {
					a.titleBar(c)
					row := ui.Row(c).Grow(1).Basis(0).MinHeight(0).AlignItems(ui.Stretch)
					a.sidebarSplit.rowWidth = row.Bounds().W
					row.Children(func() {
						a.sidebar(c)
						a.splitHandle(c, &a.sidebarSplit, shareForWidth(a.sidebarSplit.rowWidth-splitterWidth, sidebarWidth), sidebarPaneMin, a.workbenchContentMin(), a.t("resizeSidebarPane"))
						a.contentCard(c)
					})
				})
			}, int(width), int(height))
			wait := func() {
				t.Helper()
				timer := time.NewTimer(10 * time.Second)
				defer timer.Stop()
				for {
					tt.Frame()
					pending := a.loading || a.busy || a.diffLoading || a.commitDiffLoading || a.commitFilesLoading
					if s := a.currentConflictEditor(); s != nil {
						pending = pending || s.Loading
					}
					for _, entry := range a.overviewEntries {
						pending = pending || entry.Loading
					}
					if !pending {
						break
					}
					select {
					case fn := <-done:
						fn()
					case <-timer.C:
						t.Fatal("workbench did not settle")
					}
				}
				for range 3 {
					tt.Frame()
				}
			}
			click := func(role ui.Role, label string) {
				t.Helper()
				node, ok := tt.AccessByRole(role, label)
				if !ok || node.Offscreen || node.Disabled || node.Bounds.W <= 0 || node.Bounds.H <= 0 || node.Bounds.X+node.Bounds.W > width+1 || node.Bounds.Y+node.Bounds.H > height+1 {
					t.Fatalf("unusable control %q: %+v", label, node)
				}
				tt.ClickAt(node.Bounds.X+node.Bounds.W/2, node.Bounds.Y+node.Bounds.H/2)
			}
			shot := func(name string) {
				t.Helper()
				dir := os.Getenv("SUPERGIT_EDITOR_SHOTS")
				if dir == "" {
					return
				}
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				f, err := os.Create(filepath.Join(dir, fmt.Sprintf("%dx%d-%s.png", int(width), int(height), name)))
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(f, tt.Image())
				closeErr := f.Close()
				if err != nil || closeErr != nil {
					t.Fatalf("image: %v %v", err, closeErr)
				}
			}
			wait()
			commit, ok := tt.AccessByRole(ui.RoleButton, a.t("commit"))
			if !ok || !commit.Disabled {
				t.Fatal("unresolved conflict did not block commit")
			}
			// Enter directly from the Changes overview, not a preconfigured editor state.
			click(ui.RoleButton, a.t("conflictEditorOpen"))
			wait()
			click(ui.RoleButton, a.t("conflictTheirs"))
			if _, ok := tt.Find("incoming content\n"); !ok {
				t.Fatal("incoming version absent")
			}
			if width == 1280 {
				if _, ok := tt.Find("current content\n"); !ok {
					t.Fatal("wide comparison hides current side")
				}
			}
			click(ui.RoleButton, a.t("conflictEditorBase"))
			if _, ok := tt.Find("base\n"); !ok {
				t.Fatal("base absent")
			}
			resultField, _ := tt.AccessByRole(ui.RoleTextField, a.t("conflictEditorResult"))
			if height == 900 && resultField.Bounds.H < 250 {
				t.Fatalf("result editor wastes available detail height: %+v", resultField.Bounds)
			}
			if height == 650 {
				a.currentConflictEditor().Error = strings.Repeat("Cannot stage: another Git operation holds the Index lock. ", 6)
				tt.Frame()
			}
			click(ui.RoleTextField, a.t("conflictEditorResult"))
			tt.Key(ui.Cmd, ui.KeyA)
			tt.Type("combined current and incoming\n")
			tt.Advance(150 * time.Millisecond)
			shot("editing")
			click(ui.RoleButton, a.t("conflictEditorSave"))
			wait()
			if got := splitCommitGit(t, root, "show", ":base.txt"); got != "combined current and incoming\n" {
				t.Fatalf("Index=%q", got)
			}
			click(ui.RoleTextField, a.t("commitSummary"))
			tt.Key(ui.Cmd, ui.KeyA)
			tt.Type("Combine conflict in app")
			click(ui.RoleTextField, a.t("commitDescription"))
			tt.Key(ui.Cmd, ui.KeyA)
			tt.Type("Preserve both intents.")
			tt.Advance(150 * time.Millisecond)
			shot("resolved")
			click(ui.RoleButton, a.t("commit"))
			wait()
			if got := strings.TrimSpace(splitCommitGit(t, root, "show", "-s", "--format=%P", "HEAD")); got != local+" "+incoming {
				t.Fatalf("parents=%q", got)
			}
			if got := splitCommitGit(t, root, "show", "HEAD:base.txt"); got != "combined current and incoming\n" {
				t.Fatalf("tree=%q", got)
			}
			if got := splitCommitGit(t, root, "status", "--porcelain=v1"); got != "" {
				t.Fatalf("remaining=%q", got)
			}
			if a.operation.Kind != "" {
				t.Fatal("merge still pending")
			}
		})
	}
}
