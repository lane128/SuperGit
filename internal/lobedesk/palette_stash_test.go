package lobedesk

import (
	"context"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func paletteStashEntry(t *testing.T, a *app) appShortcut {
	t.Helper()
	a.commandPalette.query = ""
	for _, s := range a.commandPaletteEntries() {
		if s.ID == "stash.saveTracked" {
			return s
		}
	}
	t.Fatal("missing tracked Stash palette command")
	return appShortcut{}
}

// Render generic prompts before command pickers, exactly as mainView does.
func paletteStashTester(a *app, opener string, width, height int) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		a.registerShortcuts(c)
		ui.Column(c).Fill().Children(func() {
			if opener == "history" {
				a.historyPage(c)
			} else {
				a.commitMessageFields(c)
			}
			a.promptDialog(c)
			a.commandPickerDialogs(c)
		})
	}, width, height)
}

func stashEntryApp(t *testing.T) *app {
	a := browsingApp(t)
	a.status.Entries = []gitrepo.Entry{{Path: "tracked.txt", Index: 'M', Worktree: 'M'}}
	a.commitText = "title\n\nbody"
	a.amend = true
	return a
}

func TestPaletteStashHandoffCancel(t *testing.T) {
	for _, opener := range []string{"summary", "body", "history"} {
		for _, mod := range []ui.Modifiers{ui.Cmd, ui.Cmd | ui.Shift} {
			for _, activate := range []string{"enter", "click"} {
				for _, dismiss := range []string{"escape", "backdrop", "button"} {
					t.Run(opener+"/"+shortcutModName(mod)+"/"+activate+"/"+dismiss, func(t *testing.T) {
						a := stashEntryApp(t)
						entry := paletteStashEntry(t, a)
						tt := paletteStashTester(a, opener, 900, 640)
						if opener == "history" {
							tt.Key(ui.Cmd, ui.KeyF)
							tt.Type("needle")
							tt.Frame()
						} else {
							label := a.t("commitSummary")
							if opener == "body" {
								label = a.t("commitDescription")
							}
							field, ok := tt.AccessByRole(ui.RoleTextField, label)
							if !ok {
								t.Fatal("missing opener")
							}
							tt.ClickAt(field.Bounds.X+10, field.Bounds.Y+10)
						}
						tt.Key(ui.Cmd, ui.KeyA)
						tt.Frame()
						tt.Key(mod, ui.KeyP)
						tt.Frame()
						tt.Type(entry.ID)
						tt.Frame()
						row, ok := tt.AccessByRole(ui.RoleOption, a.t(entry.Label))
						if !ok || row.Disabled {
							t.Fatal("missing enabled Stash entry")
						}
						if activate == "enter" {
							tt.Key(0, ui.KeyEnter)
						} else {
							tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
						}
						tt.Frame()
						tt.Frame()
						if a.commandPalette.open || !a.prompt.open || a.busy {
							t.Fatal("did not hand off to only prompt")
						}
						count := 0
						for _, node := range tt.AccessAll() {
							if node.Role == ui.RoleDialog {
								count++
							}
						}
						if count != 1 {
							t.Fatalf("dialogs=%d", count)
						}
						tt.Key(0, ui.KeyEnter)
						tt.Frame()
						if !a.prompt.open || a.busy {
							t.Fatal("empty message executed")
						}
						tt.Type("owned message")
						tt.Frame()
						if a.prompt.value != "owned message" {
							t.Fatal("prompt lost initial text focus", a.prompt.value)
						}
						title := a.prompt.title
						entry.Run(a)
						tt.Frame()
						if a.prompt.title != title || a.prompt.value != "owned message" {
							t.Fatal("repeat replaced prompt")
						}
						switch dismiss {
						case "escape":
							tt.Key(0, ui.KeyEscape)
						case "backdrop":
							tt.ClickAt(2, 2)
						case "button":
							b, ok := tt.AccessByRole(ui.RoleButton, a.t("cancel"))
							if !ok {
								t.Fatal("missing cancel")
							}
							tt.ClickAt(b.Bounds.X+b.Bounds.W/2, b.Bounds.Y+b.Bounds.H/2)
						}
						tt.Frame()
						tt.Frame()
						tt.Type("X")
						tt.Frame()
						if a.prompt.open || a.commandPalette.open || a.busy || !a.amend {
							t.Fatal("cancel changed modal/write/amend")
						}
						switch opener {
						case "summary":
							if a.commitText != "X\n\nbody" {
								t.Fatal("lost summary selection", a.commitText)
							}
						case "body":
							if a.commitText != "title\n\nX" {
								t.Fatal("lost Body selection", a.commitText)
							}
						case "history":
							if a.historyQuery != "X" || a.commitText != "title\n\nbody" {
								t.Fatal("lost history selection/draft")
							}
						}
					})
				}
			}
		}
	}
}

func TestPaletteStashGuards(t *testing.T) {
	entry := paletteStashEntry(t, stashEntryApp(t))
	for _, reason := range []string{"no repo", "no head", "clean", "untracked only", "conflict", "operation", "busy", "loading", "history reload", "chooser", "ref", "dialog"} {
		t.Run(reason, func(t *testing.T) {
			a := stashEntryApp(t)
			a.openCommandPalette()
			a.commandPalette.query = entry.ID
			switch reason {
			case "no repo":
				a.commandPalette.open = false
				a.repo = nil
				a.openCommandPalette()
				a.commandPalette.query = entry.ID
			case "no head":
				a.status.Head = ""
			case "clean":
				a.status.Entries = nil
			case "untracked only":
				a.status.Entries = []gitrepo.Entry{{Path: "untracked.txt", Index: '?', Worktree: '?'}}
			case "conflict":
				a.status.Entries[0].Conflict = true
			case "operation":
				a.operation.Kind = "merge"
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "history reload":
				a.historyReloading = true
			case "chooser":
				a.repositoryChooserOpen = true
			case "ref":
				a.historyRefOpen = true
			case "dialog":
				a.confirm.open = true
			}
			tt := paletteStashTester(a, "summary", 360, 360)
			row, ok := tt.AccessByRole(ui.RoleOption, a.t(entry.Label))
			why := a.commandPaletteUnavailable(entry.ID)
			if !ok || !row.Disabled || why == "" || !tt.HasText(why) {
				t.Fatalf("guard has no disabled reason: %s %q", reason, why)
			}
			wasBusy := a.busy
			tt.Key(0, ui.KeyEnter)
			entry.Run(a)
			tt.Frame()
			if a.prompt.open || a.busy != wasBusy || !a.commandPalette.open {
				t.Fatal("guarded entry executed")
			}
		})
	}
	for _, change := range []string{"root", "epoch", "reopen"} {
		t.Run("palette stale/"+change, func(t *testing.T) {
			a := stashEntryApp(t)
			a.openCommandPalette()
			generation := a.commandPalette.generation
			switch change {
			case "root":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			case "epoch":
				a.loadSeq++
			case "reopen":
				a.commandPalette.open = false
				a.openCommandPalette()
			}
			a.runPaletteCommand(entry.ID, generation)
			if a.prompt.open || a.busy {
				t.Fatal("stale palette executed")
			}
		})
	}
}

func TestPaletteStashOwnedSave(t *testing.T) {
	for _, activate := range []string{"enter", "click"} {
		t.Run(activate, func(t *testing.T) {
			a, _, root := worktreeApp(t)
			ctx, cancel := context.WithCancel(context.Background())
			a.ctx = ctx
			callbacks := make(chan func(), 64)
			a.runOnMain = func(fn func()) { callbacks <- fn }
			t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
			write := func(path, text string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(root, path), []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			read := func(path string) string {
				t.Helper()
				b, err := os.ReadFile(filepath.Join(root, path))
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}
			write("base.txt", "older stash\n")
			pullFixtureGit(t, root, "stash", "push", "-m", "prior stash")
			prior := pullFixtureGit(t, root, "stash", "list", "--format=%H")
			write(".gitignore", "ignored.txt\n")
			pullFixtureGit(t, root, "add", ".gitignore")
			pullFixtureGit(t, root, "commit", "-m", "ignore fixture")
			write("base.txt", "staged\n")
			pullFixtureGit(t, root, "add", "base.txt")
			write("base.txt", "working\n")
			write("untracked.txt", "untracked\n")
			write("ignored.txt", "ignored\n")
			var err error
			a.status, err = a.repo.Status(ctx)
			if err != nil {
				t.Fatal(err)
			}
			a.commitText = "draft\n\nbody"
			a.amend = true
			entry := paletteStashEntry(t, a)
			head := pullFixtureGit(t, root, "rev-parse", "HEAD")
			refs := pullFixtureGit(t, root, "for-each-ref", "refs/heads")
			index := read(".git/index")
			indexBlob := pullFixtureGit(t, root, "rev-parse", ":base.txt")
			workBlob := pullFixtureGit(t, root, "hash-object", "--", "base.txt")
			tt := paletteStashTester(a, "summary", 900, 640)
			tt.Key(ui.Cmd, ui.KeyP)
			tt.Frame()
			tt.Type(entry.ID)
			tt.Frame()
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			tt.Frame()
			if !a.prompt.open {
				t.Fatal("missing owned prompt")
			}
			before := []string{pullFixtureGit(t, root, "stash", "list", "--format=%H"), read(".git/index"), read("base.txt"), read("untracked.txt"), read("ignored.txt")}
			tt.Key(0, ui.KeyEscape)
			tt.Frame()
			tt.Frame()
			after := []string{pullFixtureGit(t, root, "stash", "list", "--format=%H"), read(".git/index"), read("base.txt"), read("untracked.txt"), read("ignored.txt")}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("cancel mutated owned content")
			}
			tt.Key(ui.Cmd, ui.KeyP)
			tt.Frame()
			tt.Type(entry.ID)
			tt.Frame()
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			tt.Frame()
			tt.Type("owned saved message")
			tt.Frame()
			retained := a.prompt.action
			if read(".git/index") != index || read("base.txt") != "working\n" {
				t.Fatal("prompt wrote before save")
			}
			if activate == "enter" {
				tt.Key(0, ui.KeyEnter)
			} else {
				b, ok := tt.AccessByRole(ui.RoleButton, a.t("save"))
				if !ok {
					t.Fatal("missing Save")
				}
				tt.ClickAt(b.Bounds.X+b.Bounds.W/2, b.Bounds.Y+b.Bounds.H/2)
			}
			tt.Frame()
			if !a.busy || a.prompt.open {
				t.Fatal("accept did not save once")
			}
			retained("duplicate")
			for a.busy || a.loading {
				runContinuation(t, callbacks)
			}
			retained("duplicate after completion")
			if a.busy {
				t.Fatal("retained prompt created twice")
			}
			stashes, err := a.repo.Stashes(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(stashes) != 2 || stashes[1].SHA != prior || pullFixtureGit(t, root, "rev-parse", stashes[0].SHA+"^2:base.txt") != indexBlob || pullFixtureGit(t, root, "rev-parse", stashes[0].SHA+":base.txt") != workBlob {
				t.Fatal("stash did not preserve exact index/work versions", stashes)
			}
			if pullFixtureGit(t, root, "rev-parse", "HEAD") != head || pullFixtureGit(t, root, "for-each-ref", "refs/heads") != refs || read("base.txt") != "base\n" || pullFixtureGit(t, root, "show", ":base.txt") != "base" || read("untracked.txt") != "untracked\n" || read("ignored.txt") != "ignored\n" || a.commitText != "draft\n\nbody" || !a.amend {
				t.Fatal("stash changed unrelated content/draft")
			}
		})
	}
}

func TestPaletteStashChangedSessionFeedback(t *testing.T) {
	a := stashEntryApp(t)
	entry := paletteStashEntry(t, a)
	tt := paletteStashTester(a, "summary", 900, 640)
	tt.Key(ui.Cmd, ui.KeyP)
	tt.Frame()
	tt.Type(entry.ID)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Frame()
	tt.Type("message")
	tt.Frame()
	a.loadSeq++
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.prompt.open || a.busy || len(a.notices[toastMain]) != 1 || a.notices[toastMain][0].title != a.t("stashTrackedChanged") {
		t.Fatal("changed session silently refused Save")
	}
}

func TestPaletteStashPromptStaleScope(t *testing.T) {
	for _, change := range []string{"root", "epoch", "leave and return"} {
		t.Run(change, func(t *testing.T) {
			a := stashEntryApp(t)
			entry := paletteStashEntry(t, a)
			entry.Run(a)
			if !a.prompt.open {
				t.Fatal("missing prompt")
			}
			retained := a.prompt.action
			root := a.repo.Root
			switch change {
			case "root":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			case "epoch":
				a.loadSeq++
			case "leave and return":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
				a.loadSeq++
				a.repo = &gitrepo.Repository{Root: root}
				a.loadSeq++
			}
			a.prompt = promptState{}
			retained("stale")
			if a.busy {
				t.Fatal("stale prompt started a write")
			}
		})
	}
}

func TestPaletteStashAcceptanceRechecksReadiness(t *testing.T) {
	for _, change := range []string{"busy", "loading", "conflict", "operation", "untracked only"} {
		t.Run(change, func(t *testing.T) {
			a := stashEntryApp(t)
			entry := paletteStashEntry(t, a)
			tt := paletteStashTester(a, "summary", 900, 640)
			tt.Key(ui.Cmd, ui.KeyP)
			tt.Frame()
			tt.Type(entry.ID)
			tt.Frame()
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			tt.Frame()
			tt.Type("   ")
			tt.Frame()
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			if !a.prompt.open || a.busy {
				t.Fatal("whitespace-only message executed")
			}
			tt.Key(ui.Cmd, ui.KeyA)
			tt.Type("message")
			tt.Frame()
			switch change {
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "conflict":
				a.status.Entries[0].Conflict = true
			case "operation":
				a.operation.Kind = "merge"
			case "untracked only":
				a.status.Entries = []gitrepo.Entry{{Path: "new.txt", Index: '?', Worktree: '?'}}
			}
			wasBusy := a.busy
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			if a.prompt.open || a.busy != wasBusy || !a.progressSince.IsZero() || len(a.notices[toastMain]) != 1 {
				t.Fatal("changed readiness executed or silently refused")
			}
			if a.notices[toastMain][0].title != a.commandPaletteUnavailable(entry.ID) {
				t.Fatal("refusal did not explain the current reason")
			}
		})
	}
}

func TestPaletteStashNarrowPromptAndTab(t *testing.T) {
	for _, loc := range []locale{localeEnglish, localeChinese} {
		for _, mode := range []string{"light", "dark"} {
			t.Run(fmt.Sprint(loc, "/", mode), func(t *testing.T) {
				a := stashEntryApp(t)
				a.locale = loc
				a.themeMode = mode
				entry := paletteStashEntry(t, a)
				tt := paletteStashTester(a, "summary", 360, 360)
				tt.Key(ui.Cmd, ui.KeyP)
				tt.Frame()
				tt.Type(entry.ID)
				tt.Frame()
				if dir := os.Getenv("SUPERGIT_STASH_CAPTURE_DIR"); dir != "" {
					f, err := os.Create(filepath.Join(dir, fmt.Sprintf("stash-entry-360x360-%s-%s.png", loc, mode)))
					if err != nil {
						t.Fatal(err)
					}
					err = png.Encode(f, tt.Image())
					closeErr := f.Close()
					if err != nil {
						t.Fatal(err)
					}
					if closeErr != nil {
						t.Fatal(closeErr)
					}
				}
				tt.Key(0, ui.KeyEnter)
				tt.Frame()
				tt.Frame()
				for _, node := range tt.AccessAll() {
					if node.Role == ui.RoleTextField || node.Role == ui.RoleButton || node.Role == ui.RoleDialog {
						if node.Bounds.X < 16 || node.Bounds.X+node.Bounds.W > 344 {
							t.Fatalf("prompt outside narrow gutters: %+v", node)
						}
					}
				}
				tt.Type("message")
				tt.Frame()
				tt.Key(0, ui.KeyTab)
				tt.Frame()
				tt.Frame()
				if !tt.Focused(a.t("cancel")) {
					t.Fatal("Tab did not leave input for Cancel")
				}
				if dir := os.Getenv("SUPERGIT_STASH_CAPTURE_DIR"); dir != "" {
					f, err := os.Create(filepath.Join(dir, fmt.Sprintf("stash-prompt-360x360-%s-%s.png", loc, mode)))
					if err != nil {
						t.Fatal(err)
					}
					err = png.Encode(f, tt.Image())
					e := f.Close()
					if err != nil {
						t.Fatal(err)
					}
					if e != nil {
						t.Fatal(e)
					}
				}
			})
		}
	}
}
