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

func TestPaletteCreateTagGuards(t *testing.T) {
	for _, change := range []string{"repo", "page", "worktree", "empty", "filtered", "busy", "loading", "reload", "ref", "dialog"} {
		t.Run(change, func(t *testing.T) {
			a := createTagPaletteApp(t)
			entry := createTagPaletteEntry(t, a)
			switch change {
			case "repo":
				a.repo = nil
			case "page":
				a.page = "changes"
			case "worktree":
				a.worktreeSelected = true
			case "empty":
				a.selectedSHA = ""
			case "filtered":
				a.historyQuery = "no matching row"
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "reload":
				a.historyReloading = true
			case "ref":
				a.historyRefOpen = true
			case "dialog":
				a.confirm.open = true
			}
			if a.commandPaletteUnavailable(entry.ID) == "" {
				t.Fatal("invalid tag context has no reason")
			}
			entry.Run(a)
			if a.prompt.open {
				t.Fatal("invalid context opened tag prompt")
			}
		})
	}
	for _, allowed := range []string{"hidden details", "file history", "dirty"} {
		t.Run(allowed, func(t *testing.T) {
			a := createTagPaletteApp(t)
			entry := createTagPaletteEntry(t, a)
			switch allowed {
			case "hidden details":
				a.historyDetailsHidden = true
			case "file history":
				a.fileFilter = "base.txt"
				a.fileCommits = a.commits
			case "dirty":
				a.status.Entries = []gitrepo.Entry{{Path: "dirty", Index: 'M', Worktree: 'M'}}
			}
			if a.commandPaletteUnavailable(entry.ID) != "" {
				t.Fatal("valid selected row was blocked")
			}
		})
	}
	for _, change := range []string{"root", "epoch", "busy", "loading", "page", "worktree"} {
		t.Run("accept/"+change, func(t *testing.T) {
			a := createTagPaletteApp(t)
			entry := createTagPaletteEntry(t, a)
			entry.Run(a)
			retained := a.prompt.action
			if retained == nil {
				t.Fatal("missing tag prompt")
			}
			switch change {
			case "root":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			case "epoch":
				a.loadSeq++
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "page":
				a.page = "branches"
			case "worktree":
				a.worktreeSelected = true
			}
			busy := a.busy
			a.prompt = promptState{}
			notices := len(a.notices[toastMain])
			retained("v-stale")
			retained("v-again")
			if a.busy != busy || !a.progressSince.IsZero() {
				t.Fatal("stale callback started a write")
			}
			added := a.notices[toastMain][notices:]
			if change == "root" {
				if len(added) != 0 {
					t.Fatal("stale callback notified another repo")
				}
			} else if len(added) != 1 || added[0].kind != "warning" {
				t.Fatal("stale refusal must warn once")
			}
		})
	}
}

func TestPaletteCreateTagOwned(t *testing.T) {
	for _, scenario := range []string{"enter", "click", "selection", "invalid", "duplicate"} {
		t.Run(scenario, func(t *testing.T) {
			a, _, root := worktreeApp(t)
			ctx, cancel := context.WithCancel(context.Background())
			a.ctx = ctx
			callbacks := make(chan func(), 64)
			a.runOnMain = func(fn func()) { callbacks <- fn }
			t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
			first := pullFixtureGit(t, root, "rev-parse", "HEAD")
			pullFixtureGit(t, root, "commit", "--allow-empty", "-m", "second")
			head := pullFixtureGit(t, root, "rev-parse", "HEAD")
			pullFixtureGit(t, root, "tag", "preserved", head)
			write := func(path, value string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(root, path), []byte(value), 0600); err != nil {
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
			write("base.txt", "stashed\n")
			pullFixtureGit(t, root, "stash", "push", "-m", "preserved")
			write("base.txt", "staged\n")
			pullFixtureGit(t, root, "add", "base.txt")
			write("base.txt", "work\n")
			write("untracked.txt", "untouched\n")
			var err error
			a.status, err = a.repo.Status(ctx)
			if err != nil {
				t.Fatal(err)
			}
			a.commits = []gitrepo.Commit{{SHA: head, Subject: "head"}, {SHA: first, Subject: "selected old commit"}}
			a.selectedSHA = first
			a.worktreeSelected = false
			a.commitText = "title\n\nbody"
			a.amend = true
			content := func() string {
				return strings.Join([]string{pullFixtureGit(t, root, "status", "--porcelain=v1", "--untracked-files=all"), read(".git/HEAD"), read(".git/index"), read("base.txt"), read("untracked.txt")}, "\x00")
			}
			before, refs := content(), pullFixtureGit(t, root, "for-each-ref", "--format=%(objectname) %(refname)")
			entry := createTagPaletteEntry(t, a)
			tt := paletteStashTester(a, "summary", 900, 640)
			open := func() {
				tt.Key(ui.Cmd, ui.KeyP)
				tt.Frame()
				tt.Type(entry.ID)
				tt.Frame()
				tt.Key(0, ui.KeyEnter)
				tt.Frame()
				tt.Frame()
			}
			open()
			tt.Type("cancelled")
			tt.Frame()
			tt.Key(0, ui.KeyEscape)
			tt.Frame()
			tt.Frame()
			if content() != before || pullFixtureGit(t, root, "for-each-ref", "--format=%(objectname) %(refname)") != refs {
				t.Fatal("cancel mutated owned repository")
			}
			open()
			name := "版本/v1.0"
			if scenario == "invalid" {
				name = "bad..tag"
			}
			if scenario == "duplicate" {
				name = "preserved"
			}
			tt.Type(name)
			tt.Frame()
			retained := a.prompt.action
			if scenario == "selection" {
				a.selectedSHA = head
			}
			if scenario == "click" {
				b, ok := tt.AccessByRole(ui.RoleButton, a.t("create"))
				if !ok {
					t.Fatal("missing Create")
				}
				tt.ClickAt(b.Bounds.X+b.Bounds.W/2, b.Bounds.Y+b.Bounds.H/2)
			} else {
				tt.Key(0, ui.KeyEnter)
			}
			tt.Frame()
			if !a.busy || a.prompt.open {
				t.Fatal("confirmation did not start tag creation")
			}
			retained("duplicate-extra")
			for a.busy || a.loading {
				runContinuation(t, callbacks)
			}
			retained("duplicate-after")
			if a.busy || content() != before || a.commitText != "title\n\nbody" || !a.amend || pullFixtureGit(t, root, "rev-parse", "HEAD") != head {
				t.Fatal("tag creation changed content/HEAD/draft")
			}
			current := pullFixtureGit(t, root, "for-each-ref", "--format=%(objectname) %(refname)")
			if scenario == "invalid" || scenario == "duplicate" {
				if current != refs {
					t.Fatal("failed tag creation changed refs")
				}
				found := false
				for _, notice := range a.notices[toastMain] {
					if notice.kind == "error" && notice.title == a.t("createTagFailed") {
						found = true
					}
				}
				if !found {
					t.Fatal("missing tag error feedback")
				}
				return
			}
			if pullFixtureGit(t, root, "rev-parse", "refs/tags/"+name) != first || pullFixtureGit(t, root, "cat-file", "-t", "refs/tags/"+name) != "commit" {
				t.Fatal("tag is not lightweight at exact captured old commit")
			}
			var preserved []string
			created := 0
			for _, ref := range strings.Split(current, "\n") {
				if ref == first+" refs/tags/"+name {
					created++
				} else {
					preserved = append(preserved, ref)
				}
			}
			if created != 1 || strings.Join(preserved, "\n") != refs {
				t.Fatal("tag creation changed existing refs or created extra refs")
			}
		})
	}
}

func TestPaletteCreateTagCancelledCallback(t *testing.T) {
	for _, dismiss := range []string{"escape", "backdrop", "button"} {
		t.Run(dismiss, func(t *testing.T) {
			a, _, root := worktreeApp(t)
			ctx, cancel := context.WithCancel(context.Background())
			a.ctx = ctx
			callbacks := make(chan func(), 64)
			a.runOnMain = func(fn func()) { callbacks <- fn }
			t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
			sha := pullFixtureGit(t, root, "rev-parse", "HEAD")
			a.commits = []gitrepo.Commit{{SHA: sha, Subject: "selected"}}
			a.selectedSHA = sha
			a.worktreeSelected = false
			refs := pullFixtureGit(t, root, "for-each-ref")
			entry := createTagPaletteEntry(t, a)
			tt := paletteStashTester(a, "summary", 900, 640)
			tt.Key(ui.Cmd, ui.KeyP)
			tt.Frame()
			tt.Type(entry.ID)
			tt.Frame()
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			tt.Frame()
			retained := a.prompt.action
			if retained == nil {
				t.Fatal("missing captured action")
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
			retained("cancelled-again")
			started := a.busy
			for a.busy || a.loading {
				runContinuation(t, callbacks)
			}
			if started || pullFixtureGit(t, root, "for-each-ref") != refs {
				t.Fatal("cancelled prompt retained a live tag creation callback")
			}
		})
	}
}

func TestPaletteCreateTagNarrowAndTab(t *testing.T) {
	for _, loc := range []locale{localeEnglish, localeChinese} {
		for _, mode := range []string{"light", "dark"} {
			t.Run(fmt.Sprint(loc, "/", mode), func(t *testing.T) {
				a := createTagPaletteApp(t)
				a.locale = loc
				a.themeMode = mode
				a.commits[0].SHA = strings.Repeat("a", 40)
				a.selectedSHA = a.commits[0].SHA
				entry := createTagPaletteEntry(t, a)
				tt := paletteStashTester(a, "summary", 360, 360)
				capture := func(part string) {
					if dir := os.Getenv("SUPERGIT_CREATE_TAG_CAPTURE_DIR"); dir != "" {
						if err := os.MkdirAll(dir, 0700); err != nil {
							t.Fatal(err)
						}
						f, err := os.Create(filepath.Join(dir, fmt.Sprintf("tag-%s-360x360-%s-%s.png", part, loc, mode)))
						if err != nil {
							t.Fatal(err)
						}
						err = png.Encode(f, tt.Image())
						ce := f.Close()
						if err != nil {
							t.Fatal(err)
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
				tt.Advance(150 * time.Millisecond)
				capture("entry")
				tt.Key(0, ui.KeyEnter)
				tt.Frame()
				tt.Frame()
				tt.Type("v1.0")
				tt.Frame()
				tt.Advance(150 * time.Millisecond)
				for _, n := range tt.AccessAll() {
					if n.Role == ui.RoleTextField && n.Name == a.t("tagName") || n.Role == ui.RoleButton || n.Role == ui.RoleDialog {
						if n.Bounds.X < 16 || n.Bounds.X+n.Bounds.W > 344 {
							t.Fatalf("outside narrow gutters: %+v", n)
						}
					}
				}
				capture("prompt")
				tt.Key(0, ui.KeyTab)
				tt.Frame()
				tt.Frame()
				if !tt.Focused(a.t("cancel")) {
					t.Fatal("Tab did not leave tag name")
				}
				tt.Key(0, ui.KeyEscape)
				tt.Frame()
				tt.Frame()
			})
		}
	}
}
