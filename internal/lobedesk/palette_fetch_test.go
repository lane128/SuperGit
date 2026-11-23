package lobedesk

import (
	"context"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func paletteFetchEntry(t *testing.T, a *app) appShortcut {
	t.Helper()
	a.commandPalette.query = ""
	for _, entry := range a.commandPaletteEntries() {
		if entry.ID == "repo.fetch" {
			return entry
		}
	}
	t.Fatal("missing palette Fetch command")
	return appShortcut{}
}

func TestPaletteFetchCatalogAndGuards(t *testing.T) {
	a := headShortcutApp(t)
	entry := paletteFetchEntry(t, a)
	if entry.Keys != "" || entry.Key != 0 || entry.Run == nil || entry.Available == nil {
		t.Fatal("Fetch must be a command without a shortcut")
	}
	for _, s := range shortcutTable() {
		if s.ID == entry.ID || s.Key == ui.KeyF6 {
			t.Fatal("Fetch invented a key binding")
		}
	}
	for _, reason := range []string{"nil", "no remote", "busy", "loading", "history", "chooser", "ref", "dialog", "palette", "checkout", "switch"} {
		t.Run(reason, func(t *testing.T) {
			a := headShortcutApp(t)
			a.remotes = []gitrepo.Remote{{Name: "origin"}}
			switch reason {
			case "nil":
				a.repo = nil
			case "no remote":
				a.remotes = nil
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "history":
				a.historyReloading = true
			case "chooser":
				a.repositoryChooserOpen = true
			case "ref":
				a.historyRefOpen = true
			case "dialog":
				a.confirm.open = true
			case "palette":
				a.openCommandPalette()
			case "checkout":
				a.openQuickCheckout()
			case "switch":
				a.openQuickRepositorySwitch()
			}
			if entry.Available(a) {
				t.Fatal("guarded command available")
			}
			entry.Run(a)
			if a.fetching || !a.lastAutoFetch.IsZero() {
				t.Fatal("retained command started Fetch")
			}
		})
	}
	for _, reason := range []string{"nil", "no remote", "busy", "loading", "history", "chooser", "ref", "dialog"} {
		t.Run("row/"+reason, func(t *testing.T) {
			a := headShortcutApp(t)
			a.remotes = []gitrepo.Remote{{Name: "origin"}}
			a.openCommandPalette()
			a.commandPalette.query = "repo.fetch"
			switch reason {
			case "nil":
				a.commandPalette.open = false
				a.repo = nil
				a.openCommandPalette()
				a.commandPalette.query = "repo.fetch"
			case "no remote":
				a.remotes = nil
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "history":
				a.historyReloading = true
			case "chooser":
				a.repositoryChooserOpen = true
			case "ref":
				a.historyRefOpen = true
			case "dialog":
				a.confirm.open = true
			}
			tt := paletteTester(a, false, 360, 360)
			row, ok := tt.AccessByRole(ui.RoleOption, a.t(entry.Label))
			why := a.commandPaletteUnavailable(entry.ID)
			if !ok || !row.Disabled || why == "" || !tt.HasText(why) {
				t.Fatal("disabled row lacks actual reason")
			}
			if reason == "no remote" && why != a.t("noRemote") {
				t.Fatal("missing remote reason")
			}
			tt.Key(0, ui.KeyEnter)
			if !a.commandPalette.open || a.fetching || !a.lastAutoFetch.IsZero() {
				t.Fatal("disabled Enter executed or closed")
			}
		})
	}
	for _, change := range []string{"root", "epoch", "reopen", "no remote"} {
		t.Run("stale/"+change, func(t *testing.T) {
			a := headShortcutApp(t)
			a.remotes = []gitrepo.Remote{{Name: "origin"}}
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
			case "no remote":
				a.remotes = nil
			}
			a.runPaletteCommand(entry.ID, generation)
			if a.fetching || !a.lastAutoFetch.IsZero() {
				t.Fatal("stale command executed")
			}
			if change == "reopen" && (!a.commandPalette.open || a.commandPalette.generation == generation) {
				t.Fatal("old callback closed new session")
			}
		})
	}
}

func TestPaletteFetchOwnedInputs(t *testing.T) {
	for _, opener := range []string{"message", "history"} {
		for _, activation := range []string{"enter", "click"} {
			t.Run(opener+"/"+activation, func(t *testing.T) {
				a, _, root := worktreeApp(t)
				ctx, cancel := context.WithCancel(context.Background())
				a.ctx = ctx
				done := make(chan func(), 64)
				a.runOnMain = func(fn func()) { done <- fn }
				t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
				t.Setenv("GIT_ALLOW_PROTOCOL", "file")
				a.locale = localeEnglish
				a.page = "changes"
				a.commitText = "title\n\nbody"
				a.amend = true
				bare := filepath.Join(t.TempDir(), "owned remote.git")
				pullFixtureGit(t, root, "clone", "--bare", root, bare)
				for _, name := range []string{"origin", "backup"} {
					pullFixtureGit(t, root, "remote", "add", name, bare)
					pullFixtureGit(t, root, "config", "remote."+name+".prune", "false")
					pullFixtureGit(t, root, "fetch", name)
					pullFixtureGit(t, root, "update-ref", "refs/remotes/"+name+"/stale", "HEAD")
				}
				pullFixtureGit(t, bare, "config", "user.name", "Fetch Fixture")
				pullFixtureGit(t, bare, "config", "user.email", "fetch@example.invalid")
				old := pullFixtureGit(t, bare, "rev-parse", "refs/heads/main")
				tree := pullFixtureGit(t, bare, "rev-parse", old+"^{tree}")
				want := pullFixtureGit(t, bare, "commit-tree", tree, "-p", old, "-m", "remote advance")
				pullFixtureGit(t, bare, "update-ref", "refs/heads/main", want)
				write := func(path, text string) {
					t.Helper()
					if err := os.WriteFile(filepath.Join(root, path), []byte(text), 0600); err != nil {
						t.Fatal(err)
					}
				}
				write("base.txt", "staged\n")
				pullFixtureGit(t, root, "add", "--", "base.txt")
				write("base.txt", "unstaged\n")
				write("untracked.txt", "untracked\n")
				var err error
				a.status, err = a.repo.Status(ctx)
				if err != nil {
					t.Fatal(err)
				}
				a.remotes, err = a.repo.Remotes(ctx)
				if err != nil {
					t.Fatal(err)
				}
				read := func(path string) string {
					t.Helper()
					b, err := os.ReadFile(filepath.Join(root, path))
					if err != nil {
						t.Fatal(err)
					}
					return string(b)
				}
				fingerprint := func() []string {
					return []string{pullFixtureGit(t, root, "rev-parse", "HEAD"), pullFixtureGit(t, root, "symbolic-ref", "HEAD"), pullFixtureGit(t, root, "for-each-ref", "refs/heads"), pullFixtureGit(t, root, "ls-files", "--stage"), pullFixtureGit(t, root, "status", "--porcelain=v1"), read(".git/index"), read(".git/config"), read("base.txt"), read("untracked.txt")}
				}
				before := fingerprint()
				refsBefore := pullFixtureGit(t, root, "for-each-ref")
				fetchBefore := read(".git/FETCH_HEAD")
				remoteBefore := pullFixtureGit(t, bare, "for-each-ref")
				entry := paletteFetchEntry(t, a)
				var tt *ui.Tester
				if opener == "history" {
					a.page = "history"
					tt = pageTester(a, func(c *ui.Context) { a.registerShortcuts(c); a.historyPage(c); a.commandPickerDialogs(c) })
					tt.Key(ui.Cmd, ui.KeyF)
					tt.Type("needle")
					tt.Frame()
				} else {
					tt = paletteTester(a, true, 900, 640)
					label := a.t("commitMessage")
					field, ok := tt.AccessByRole(ui.RoleTextField, label)
					if !ok {
						t.Fatal("missing opener")
					}
					tt.ClickAt(field.Bounds.X+10, field.Bounds.Y+10)
				}
				tt.Key(ui.Cmd, ui.KeyA)
				tt.Frame()
				// Both dismissal paths must leave the owned repositories untouched.
				for _, dismiss := range []string{"escape", "backdrop"} {
					tt.Key(ui.Cmd, ui.KeyP)
					tt.Frame()
					tt.Type("repo.fetch")
					tt.Frame()
					if dismiss == "escape" {
						tt.Key(0, ui.KeyEscape)
					} else {
						tt.ClickAt(2, 2)
					}
					tt.Frame()
					tt.Frame()
					if !reflect.DeepEqual(fingerprint(), before) || pullFixtureGit(t, root, "for-each-ref") != refsBefore || read(".git/FETCH_HEAD") != fetchBefore || pullFixtureGit(t, bare, "for-each-ref") != remoteBefore {
						t.Fatal("opening/cancelling changed Git")
					}
				}
				tt.Key(ui.Cmd, ui.KeyP)
				tt.Frame()
				tt.Type("repo.fetch")
				tt.Frame()
				row, ok := tt.AccessByRole(ui.RoleOption, a.t(entry.Label))
				if !ok || row.Disabled {
					t.Fatal("missing enabled Fetch row")
				}
				generation := a.commandPalette.generation
				if activation == "enter" {
					tt.Key(0, ui.KeyEnter)
				} else {
					tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
				}
				tt.Frame()
				tt.Frame()
				if a.commandPalette.open || !a.busy || !a.fetching {
					t.Fatal("Fetch should start in one step")
				}
				started := a.lastAutoFetch
				a.runPaletteCommand(entry.ID, generation)
				entry.Run(a)
				if a.lastAutoFetch != started {
					t.Fatal("repeat restarted Fetch")
				}
				for a.busy || a.loading {
					runContinuation(t, done)
				}
				tt.Frame()
				tt.Frame()
				if !reflect.DeepEqual(fingerprint(), before) || pullFixtureGit(t, bare, "for-each-ref") != remoteBefore || a.commitText != "title\n\nbody" || !a.amend || a.fetching {
					t.Fatalf("Fetch invariant: local=%v remote=%v draft=%q amend=%v fetching=%v", reflect.DeepEqual(fingerprint(), before), pullFixtureGit(t, bare, "for-each-ref") == remoteBefore, a.commitText, a.amend, a.fetching)
				}
				for _, name := range []string{"origin", "backup"} {
					if pullFixtureGit(t, root, "rev-parse", "refs/remotes/"+name+"/main") != want || pullFixtureGit(t, root, "rev-parse", "refs/remotes/"+name+"/stale") != old {
						t.Fatal("Fetch missed remote or pruned stale ref")
					}
				}
				if !strings.Contains(read(".git/FETCH_HEAD"), want) || read(".git/FETCH_HEAD") == fetchBefore {
					t.Fatal("FETCH_HEAD not updated")
				}
				found := false
				for _, n := range a.notices[toastMain] {
					found = found || n.title == a.t("fetchSuccess")
				}
				if !found {
					t.Fatal("missing existing Fetch success feedback")
				}
				tt.Type("X")
				tt.Frame()
				switch opener {
				case "message":
					if a.commitText != "X" {
						t.Fatal("lost commit message selection", a.commitText)
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

func TestPaletteFetchFailureFeedback(t *testing.T) {
	a, done, root := worktreeApp(t)
	t.Setenv("GIT_ALLOW_PROTOCOL", "file")
	a.commitText = "draft\n\nbody"
	pullFixtureGit(t, root, "remote", "add", "origin", filepath.Join(t.TempDir(), "missing.git"))
	var err error
	a.remotes, err = a.repo.Remotes(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	entry := paletteFetchEntry(t, a)
	tt := paletteTester(a, true, 600, 500)
	tt.Key(ui.Cmd, ui.KeyP)
	tt.Frame()
	tt.Type(entry.ID)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	for a.busy || a.loading {
		runContinuation(t, done)
	}
	if a.fetching || a.commitText != "draft\n\nbody" {
		t.Fatal("failed Fetch retained busy or changed draft")
	}
	found := false
	for _, n := range a.notices[toastMain] {
		found = found || n.kind == "error" && n.title == a.t("fetchFailed") && n.description != ""
	}
	if !found {
		t.Fatalf("missing failure feedback: palette=%+v busy=%v started=%v notices=%+v", a.commandPalette, a.busy, a.lastAutoFetch, a.notices[toastMain])
	}
}

func TestPaletteFetchRenderedEntry(t *testing.T) {
	for _, loc := range []locale{localeEnglish, localeChinese} {
		for _, mode := range []string{"light", "dark"} {
			t.Run(fmt.Sprint(loc, "/", mode), func(t *testing.T) {
				a := headShortcutApp(t)
				a.locale = loc
				a.themeMode = mode
				a.remotes = []gitrepo.Remote{{Name: "origin"}}
				entry := paletteFetchEntry(t, a)
				a.openCommandPalette()
				a.commandPalette.query = "repo.fetch"
				tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.commandPaletteDialog(c) }, 360, 360)
				row, ok := tt.AccessByRole(ui.RoleOption, a.t(entry.Label))
				if !ok || row.Disabled || row.Bounds.X < 16 || row.Bounds.X+row.Bounds.W > 344 {
					t.Fatal("invalid narrow Fetch row")
				}
				if dir := os.Getenv("SUPERGIT_FETCH_CAPTURE_DIR"); dir != "" {
					f, err := os.Create(filepath.Join(dir, fmt.Sprintf("fetch-360x360-%s-%s.png", loc, mode)))
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
			})
		}
	}
}
