package lobedesk

import (
	"context"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func createBranchEntry(t *testing.T, a *app) appShortcut {
	t.Helper()
	a.commandPalette.query = ""
	for _, s := range a.commandPaletteEntries() {
		if s.ID == "branch.create" {
			return s
		}
	}
	t.Fatal("missing create-and-switch palette command")
	return appShortcut{}
}
func createBranchApp(t *testing.T) *app {
	a := browsingApp(t)
	a.status.Entries = nil
	a.commitText = "title\n\nbody"
	a.amend = true
	return a
}

func TestPaletteCreateBranchHandoff(t *testing.T) {
	for _, opener := range []string{"message", "history"} {
		for _, mod := range []ui.Modifiers{ui.Cmd, ui.Cmd | ui.Shift} {
			for _, activation := range []string{"enter", "click"} {
				for _, dismiss := range []string{"escape", "backdrop", "button"} {
					t.Run(opener+"/"+shortcutModName(mod)+"/"+activation+"/"+dismiss, func(t *testing.T) {
						a := createBranchApp(t)
						entry := createBranchEntry(t, a)
						tt := paletteStashTester(a, opener, 900, 640)
						if opener == "history" {
							tt.Key(ui.Cmd, ui.KeyF)
							tt.Type("needle")
							tt.Frame()
						} else {
							label := a.t("commitMessage")
							f, ok := tt.AccessByRole(ui.RoleTextField, label)
							if !ok {
								t.Fatal("missing opener")
							}
							tt.ClickAt(f.Bounds.X+10, f.Bounds.Y+10)
						}
						tt.Key(ui.Cmd, ui.KeyA)
						tt.Frame()
						tt.Key(mod, ui.KeyP)
						tt.Frame()
						tt.Type(entry.ID)
						tt.Frame()
						row, ok := tt.AccessByRole(ui.RoleOption, a.t(entry.Label))
						if !ok || row.Disabled {
							t.Fatal("missing enabled command")
						}
						if activation == "enter" {
							tt.Key(0, ui.KeyEnter)
						} else {
							tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
						}
						tt.Frame()
						tt.Frame()
						if !a.prompt.open || a.commandPalette.open || a.busy || !tt.HasText(a.tf("createBranchStart", shortSHA(a.status.Head))) || !tt.HasText(a.t("createBranchSwitch")) {
							t.Fatal("handoff lost source or checkout intent")
						}
						dialogs := 0
						for _, n := range tt.AccessAll() {
							if n.Role == ui.RoleDialog {
								dialogs++
							}
						}
						if dialogs != 1 {
							t.Fatal("stacked dialogs")
						}
						tt.Type("   ")
						tt.Frame()
						tt.Key(0, ui.KeyEnter)
						tt.Frame()
						if !a.prompt.open || a.busy {
							t.Fatal("whitespace name executed")
						}
						tt.Key(ui.Cmd, ui.KeyA)
						tt.Type("feature/owned")
						tt.Frame()
						if a.prompt.value != "feature/owned" {
							t.Fatal("name input not focused")
						}
						entry.Run(a)
						if a.prompt.value != "feature/owned" {
							t.Fatal("repeat replaced name")
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
						tt.Type("X")
						tt.Frame()
						if a.prompt.open || a.busy || !a.amend {
							t.Fatal("cancel changed modal/write/amend")
						}
						switch opener {
						case "message":
							if a.commitText != "X" {
								t.Fatal("lost commit message selection")
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

func TestPaletteCreateBranchGuards(t *testing.T) {
	entry := createBranchEntry(t, createBranchApp(t))
	for _, reason := range []string{"nil", "head", "conflict", "operation", "busy", "loading", "reload", "chooser", "ref", "dialog"} {
		t.Run(reason, func(t *testing.T) {
			a := createBranchApp(t)
			a.openCommandPalette()
			a.commandPalette.query = entry.ID
			switch reason {
			case "nil":
				a.commandPalette.open = false
				a.repo = nil
				a.openCommandPalette()
				a.commandPalette.query = entry.ID
			case "head":
				a.status.Head = ""
			case "conflict":
				a.status.Entries = []gitrepo.Entry{{Path: "conflict", Conflict: true}}
			case "operation":
				a.operation.Kind = "merge"
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "reload":
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
				t.Fatal("disabled command lacks reason")
			}
			wasBusy := a.busy
			tt.Key(0, ui.KeyEnter)
			entry.Run(a)
			if a.prompt.open || a.busy != wasBusy || !a.commandPalette.open {
				t.Fatal("guarded entry executed")
			}
		})
	}
	for _, change := range []string{"root", "epoch", "head", "branch", "conflict", "operation", "reopen"} {
		t.Run("accept/"+change, func(t *testing.T) {
			a := createBranchApp(t)
			entry.Run(a)
			if !a.prompt.open {
				t.Fatal("missing prompt")
			}
			retained := a.prompt.action
			switch change {
			case "root":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			case "epoch":
				a.loadSeq++
			case "head":
				a.status.Head = strings.Repeat("b", 40)
			case "branch":
				a.status.Branch = "other"
			case "conflict":
				a.status.Entries = []gitrepo.Entry{{Path: "conflict", Conflict: true}}
			case "operation":
				a.operation.Kind = "merge"
			case "reopen":
				a.prompt = promptState{}
				a.openCommandPalette()
			}
			a.prompt = promptState{}
			noticesBefore := len(a.notices[toastMain])
			retained("feature/stale")
			retained("feature/again")
			if a.busy || !a.progressSince.IsZero() {
				t.Fatal("stale prompt started a write")
			}
			added := a.notices[toastMain][noticesBefore:]
			if change == "root" {
				if len(added) != 0 {
					t.Fatal("stale prompt notified a different repository")
				}
			} else if len(added) != 1 || added[0].kind != "warning" {
				t.Fatal("same-repository refusal must explain once")
			}
		})
	}
}

func TestPaletteCreateBranchOwned(t *testing.T) {
	for _, scenario := range []string{"enter", "click", "detached", "pinned", "invalid", "duplicate", "external dirty"} {
		t.Run(scenario, func(t *testing.T) {
			a, _, root := worktreeApp(t)
			ctx, cancel := context.WithCancel(context.Background())
			a.ctx = ctx
			callbacks := make(chan func(), 64)
			a.runOnMain = func(fn func()) { callbacks <- fn }
			t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
			pullFixtureGit(t, root, "tag", "preserved-tag")
			if scenario == "detached" {
				pullFixtureGit(t, root, "checkout", "--detach", "HEAD")
			}
			var err error
			a.status, err = a.repo.Status(ctx)
			if err != nil {
				t.Fatal(err)
			}
			a.commitText = "draft\n\nbody"
			a.amend = true
			entry := createBranchEntry(t, a)
			read := func(path string) string {
				t.Helper()
				b, e := os.ReadFile(filepath.Join(root, path))
				if e != nil {
					t.Fatal(e)
				}
				return string(b)
			}
			sha := a.status.Head
			index := read(".git/index")
			work := read("base.txt")
			refs := pullFixtureGit(t, root, "for-each-ref")
			headName := pullFixtureGit(t, root, "rev-parse", "--abbrev-ref", "HEAD")
			tt := paletteStashTester(a, "summary", 900, 640)
			open := func() {
				tt.Key(ui.Cmd, ui.KeyP)
				tt.Frame()
				tt.Type(entry.ID)
				tt.Frame()
				tt.Key(0, ui.KeyEnter)
				tt.Frame()
				tt.Frame()
				if !a.prompt.open || !tt.HasText(a.tf("createBranchStart", shortSHA(sha))) {
					t.Fatal("missing captured source")
				}
			}
			open()
			tt.Type("cancelled")
			tt.Frame()
			tt.Key(0, ui.KeyEscape)
			tt.Frame()
			tt.Frame()
			if pullFixtureGit(t, root, "for-each-ref") != refs || pullFixtureGit(t, root, "rev-parse", "--abbrev-ref", "HEAD") != headName || read(".git/index") != index || read("base.txt") != work {
				t.Fatal("cancel changed owned repository")
			}
			open()
			name := "feature/owned"
			if scenario == "invalid" {
				name = "bad..name"
			}
			if scenario == "duplicate" {
				name = "main"
			}
			tt.Type(name)
			tt.Frame()
			retained := a.prompt.action
			if scenario == "pinned" {
				pullFixtureGit(t, root, "commit", "--allow-empty", "-m", "external HEAD advance")
				refs = pullFixtureGit(t, root, "for-each-ref")
			}
			if scenario == "external dirty" {
				if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("external dirty\n"), 0600); err != nil {
					t.Fatal(err)
				}
				work = read("base.txt")
			}
			refsBeforeCreate := pullFixtureGit(t, root, "for-each-ref", "--format=%(objectname) %(refname)")
			if scenario == "click" {
				b, ok := tt.AccessByRole(ui.RoleButton, a.t("createBranchSwitch"))
				if !ok {
					t.Fatal("missing explicit action")
				}
				tt.ClickAt(b.Bounds.X+b.Bounds.W/2, b.Bounds.Y+b.Bounds.H/2)
			} else {
				tt.Key(0, ui.KeyEnter)
			}
			tt.Frame()
			if !a.busy || a.prompt.open {
				t.Fatal("accept did not start create")
			}
			retained("feature/duplicate")
			for a.busy || a.loading {
				runContinuation(t, callbacks)
			}
			retained("feature/duplicate-after")
			if a.busy || a.commitText != "draft\n\nbody" || !a.amend || read(".git/index") != index || read("base.txt") != work {
				t.Fatal("creation changed local content/draft")
			}
			if scenario == "invalid" || scenario == "duplicate" {
				if pullFixtureGit(t, root, "for-each-ref") != refs || pullFixtureGit(t, root, "rev-parse", "HEAD") != sha || pullFixtureGit(t, root, "rev-parse", "--abbrev-ref", "HEAD") != headName {
					t.Fatal("failed creation changed refs/HEAD")
				}
				found := false
				for _, n := range a.notices[toastMain] {
					found = found || n.kind == "error" && n.title == a.t("createBranchFailed")
				}
				if !found {
					t.Fatal("missing existing error feedback")
				}
				return
			}
			if pullFixtureGit(t, root, "rev-parse", "HEAD") != sha || pullFixtureGit(t, root, "symbolic-ref", "--short", "HEAD") != name || pullFixtureGit(t, root, "rev-parse", "refs/heads/"+name) != sha {
				t.Fatal("did not switch to exact captured start")
			}
			currentRefs := pullFixtureGit(t, root, "for-each-ref", "--format=%(objectname) %(refname)")
			var preservedRefs []string
			createdRef := sha + " refs/heads/" + name
			created := 0
			for _, ref := range strings.Split(currentRefs, "\n") {
				if ref == createdRef {
					created++
				} else {
					preservedRefs = append(preservedRefs, ref)
				}
			}
			if created != 1 || strings.Join(preservedRefs, "\n") != refsBeforeCreate {
				t.Fatal("creation changed existing refs or created extra refs")
			}
		})
	}
}

func TestPaletteCreateBranchNarrowAndTab(t *testing.T) {
	for _, loc := range []locale{localeEnglish, localeChinese} {
		for _, mode := range []string{"light", "dark"} {
			t.Run(fmt.Sprint(loc, "/", mode), func(t *testing.T) {
				a := createBranchApp(t)
				a.locale = loc
				a.themeMode = mode
				entry := createBranchEntry(t, a)
				tt := paletteStashTester(a, "summary", 360, 360)
				capture := func(part string) {
					if dir := os.Getenv("SUPERGIT_BRANCH_CREATE_CAPTURE_DIR"); dir != "" {
						f, e := os.Create(filepath.Join(dir, fmt.Sprintf("branch-%s-360x360-%s-%s.png", part, loc, mode)))
						if e != nil {
							t.Fatal(e)
						}
						e = png.Encode(f, tt.Image())
						ce := f.Close()
						if e != nil {
							t.Fatal(e)
						}
						if ce != nil {
							t.Fatal(ce)
						}
					}
				}
				tt.Key(ui.Cmd, ui.KeyP)
				tt.Frame()
				tt.Type(entry.ID)
				tt.Frame()
				capture("entry")
				tt.Key(0, ui.KeyEnter)
				tt.Frame()
				tt.Frame()
				tt.Type("feature/owned")
				tt.Frame()
				for _, n := range tt.AccessAll() {
					if n.Role == ui.RoleTextField || n.Role == ui.RoleButton || n.Role == ui.RoleDialog {
						if n.Bounds.X < 16 || n.Bounds.X+n.Bounds.W > 344 {
							t.Fatalf("outside narrow gutters: %+v", n)
						}
					}
				}
				accept, ok := tt.AccessByRole(ui.RoleButton, a.t("createBranchSwitch"))
				if !ok || accept.Disabled {
					t.Fatal("nonempty branch name did not enable acceptance")
				}
				// Render the completed disabled-to-ready color transition. The
				// headless clock does not advance when Frame is called.
				tt.Advance(150 * time.Millisecond)
				capture("prompt")
				tt.Key(0, ui.KeyTab)
				tt.Frame()
				tt.Frame()
				if !tt.Focused(a.t("cancel")) {
					t.Fatal("Tab did not leave name input")
				}
			})
		}
	}
}
