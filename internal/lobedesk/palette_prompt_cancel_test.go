package lobedesk

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// Previously exposed palette prompts must revoke a captured action on cancel,
// just like the new tag prompt. Each fixture is an owned disposable repository.
func TestPalettePromptCancelledCallbacks(t *testing.T) {
	for _, command := range []string{"branch.create", "stash.saveTracked"} {
		for _, dismiss := range []string{"escape", "backdrop", "button"} {
			t.Run(command+"/"+dismiss, func(t *testing.T) {
				a, _, root := worktreeApp(t)
				ctx, cancel := context.WithCancel(context.Background())
				a.ctx = ctx
				callbacks := make(chan func(), 64)
				a.runOnMain = func(fn func()) { callbacks <- fn }
				t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
				if command == "stash.saveTracked" {
					if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("tracked change\n"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				var err error
				a.status, err = a.repo.Status(ctx)
				if err != nil {
					t.Fatal(err)
				}
				refs := pullFixtureGit(t, root, "for-each-ref")
				tt := paletteStashTester(a, "summary", 900, 640)
				tt.Key(ui.Cmd, ui.KeyP)
				tt.Frame()
				tt.Type(command)
				tt.Frame()
				tt.Key(0, ui.KeyEnter)
				tt.Frame()
				tt.Frame()
				retained := a.prompt.action
				if retained == nil {
					t.Fatal("missing captured prompt action")
				}
				switch dismiss {
				case "escape":
					tt.Key(0, ui.KeyEscape)
				case "backdrop":
					tt.ClickAt(2, 2)
				case "button":
					b, ok := tt.AccessByRole(ui.RoleButton, a.t("cancel"))
					if !ok {
						t.Fatal("missing Cancel")
					}
					tt.ClickAt(b.Bounds.X+b.Bounds.W/2, b.Bounds.Y+b.Bounds.H/2)
				}
				tt.Frame()
				tt.Frame()
				retained("cancelled-callback")
				started := a.busy
				for a.busy || a.loading {
					runContinuation(t, callbacks)
				}
				if started || pullFixtureGit(t, root, "for-each-ref") != refs {
					t.Fatal("cancelled palette prompt retained a live write callback")
				}
			})
		}
	}
}
