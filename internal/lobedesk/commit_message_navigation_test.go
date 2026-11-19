package lobedesk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func messageNavigationFixture(t *testing.T) (*app, chan func(), gitrepo.Commit, string) {
	t.Helper()
	a, callbacks, target := quickCheckoutFixture(t)
	parent := a.status.Head
	pullFixtureGit(t, a.repo.Root, "commit", "--no-gpg-sign", "--allow-empty", "-m", "Source message")
	source := gitrepo.Commit{SHA: pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD"), Subject: "Source message", Parents: []string{parent}}
	a.status.Head = source.SHA
	if target == source.SHA || target == parent {
		t.Fatal("target must differ from source and its parent")
	}
	a.page, a.worktreeSelected, a.selectedSHA = "history", false, source.SHA
	a.commits = []gitrepo.Commit{source, {SHA: parent, Subject: "Parent"}, {SHA: target, Subject: "Message target"}}
	a.commitFilesLoaded = true
	a.commitMessage = source.Subject + "\n\nSee " + target
	a.commitFields.Summary = "keep draft"
	a.historyRef = historyRef{Kind: historyRefLocal, Name: "main"}
	a.historyPinned = source.SHA
	return a, callbacks, source, target
}

func messageNavigationTester(a *app, source gitrepo.Commit) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Children(func() { a.commitInfoContent(c, source, theme.TokensOf(c)) })
	}, 800, 400)
}

func chooseMessageCommit(t *testing.T, tt *ui.Tester, body, label string) {
	t.Helper()
	r, ok := tt.Find(body)
	if !ok {
		t.Fatalf("message absent: %s", body)
	}
	tt.RightClickAt(r.X+5, r.Y+r.H/2)
	item, ok := tt.AccessByRole(ui.RoleMenuItem, label)
	if !ok {
		t.Fatalf("message commit action missing: %s", label)
	}
	tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
}

func TestMessageCommitMenuNavigatesCapturedIndependentTarget(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		t.Run(string(language), func(t *testing.T) {
			a, callbacks, source, target := messageNavigationFixture(t)
			a.locale = language
			a.historyQuery = "Source"
			body := "See " + strings.ToUpper(target) + " and https://example.invalid/" + target
			a.commitMessage = source.Subject + "\n\n" + body
			if err := os.WriteFile(filepath.Join(a.repo.Root, "same.go"), []byte("owned staged bytes\n"), 0600); err != nil {
				t.Fatal(err)
			}
			pullFixtureGit(t, a.repo.Root, "add", "--", "same.go")
			for name, content := range map[string]string{"same.go": "owned worktree bytes\n", "message-untracked": "owned untracked bytes\n"} {
				if err := os.WriteFile(filepath.Join(a.repo.Root, name), []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			beforeStatus := pullFixtureGit(t, a.repo.Root, "status", "--porcelain=v1", "-z")
			beforeIndex, err := os.ReadFile(filepath.Join(a.repo.Root, ".git", "index"))
			if err != nil {
				t.Fatal(err)
			}
			tt := messageNavigationTester(a, source)
			label := "Jump to Commit · " + target
			if language == localeChinese {
				label = "跳转到提交 · " + target
			}
			chooseMessageCommit(t, tt, body, label)
			if a.selectedSHA != source.SHA || a.historyQuery != "Source" {
				t.Fatal("jump mutated view before object resolution")
			}
			historyCompletion(t, callbacks)()
			if a.selectedSHA != target || !a.revealSelection || a.historyQuery != "" || !a.canNavigateHistory(-1) {
				t.Fatalf("wrong navigation result: %s query=%q", a.selectedSHA, a.historyQuery)
			}
			if len(tt.OpenedURLs()) != 0 || a.commitText != "draft\n\nbody" || a.commitFields.Summary != "keep draft" || a.historyRef.Name != "main" || a.historyPinned != source.SHA {
				t.Fatal("message jump changed URL, draft or ref scope")
			}
			if got := pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD"); got != source.SHA {
				t.Fatal("message navigation checked out target")
			}
			if got := pullFixtureGit(t, a.repo.Root, "status", "--porcelain=v1", "-z"); got != beforeStatus {
				t.Fatal("message navigation changed Git status")
			}
			afterIndex, err := os.ReadFile(filepath.Join(a.repo.Root, ".git", "index"))
			if err != nil || string(afterIndex) != string(beforeIndex) {
				t.Fatal("message navigation changed index")
			}
			for name, content := range map[string]string{"same.go": "owned worktree bytes\n", "message-untracked": "owned untracked bytes\n"} {
				got, err := os.ReadFile(filepath.Join(a.repo.Root, name))
				if err != nil || string(got) != content {
					t.Fatalf("navigation changed %s", name)
				}
			}
			a.navigateHistory(-1)
			if a.selectedSHA != source.SHA || a.historyQuery != "Source" {
				t.Fatal("Back did not restore the source and query")
			}
		})
	}
}

func TestMessageCommitMenuRejectsChangedOwnership(t *testing.T) {
	for _, phase := range []string{"open menu", "resolved"} {
		for _, change := range []string{"repository", "reopen", "root", "epoch", "reload", "source", "hidden", "query", "page", "file", "worktree", "busy", "loading", "reloading", "chooser", "ref menu", "dialog", "new menu"} {
			t.Run(phase+"/"+change, func(t *testing.T) {
				a, callbacks, source, target := messageNavigationFixture(t)
				tt := messageNavigationTester(a, source)
				body := "See " + target
				r, _ := tt.Find(body)
				tt.RightClickAt(r.X+5, r.Y+r.H/2)
				label := "Jump to Commit · " + target
				item, ok := tt.AccessByRole(ui.RoleMenuItem, label)
				if !ok {
					t.Fatal("message commit action missing")
				}
				var done func()
				if phase == "resolved" {
					tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
					done = historyCompletion(t, callbacks)
				}
				switch change {
				case "repository":
					a.repo = &gitrepo.Repository{Root: t.TempDir()}
				case "reopen":
					a.repo = &gitrepo.Repository{Root: a.repo.Root}
				case "root":
					a.repo.Root = t.TempDir()
				case "epoch":
					a.loadSeq++
				case "reload":
					a.historyReload++
				case "source":
					a.selectedSHA = target
				case "hidden":
					a.historyQuery = "not present anywhere"
				case "query":
					a.historyQuery = "Source"
				case "page":
					a.page = "changes"
				case "file":
					a.fileFilter = "same.go"
				case "worktree":
					a.worktreeSelected = true
				case "busy":
					a.busy = true
				case "loading":
					a.loading = true
				case "reloading":
					a.historyReloading = true
				case "chooser":
					a.repositoryChooserOpen = true
				case "ref menu":
					a.historyRefOpen = true
				case "dialog":
					a.confirm.open = true
				case "new menu":
					tt.Key(0, ui.KeyEscape)
					tt.Frame()
					r, _ = tt.Find(body)
					tt.RightClickAt(r.X+5, r.Y+r.H/2)
				}
				selected, query, root := a.selectedSHA, a.historyQuery, a.repo.Root
				if phase == "open menu" {
					// A newly opened menu is a new user action, so test replacement only
					// against the outstanding async request below.
					if change == "new menu" {
						return
					}
					tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
				} else {
					done()
				}
				if a.selectedSHA != selected || a.historyQuery != query || a.repo.Root != root || a.pendingJumpSHA != "" || a.historyNavigation.pending != nil || a.messageCommitRequest != nil || len(a.historyNavigation.positions) != 0 || len(a.notices[toastMain]) != 0 || len(tt.OpenedURLs()) != 0 || a.commitFields.Summary != "keep draft" {
					t.Fatal("stale action changed the new view")
				}
				if phase == "open menu" {
					select {
					case <-callbacks:
						t.Fatal("stale menu launched repository resolution")
					default:
					}
				}
			})
		}
	}
}
