package lobedesk

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestWelcomeRepositoryOpenReflectsRestoreReadiness(t *testing.T) {
	a := &app{locale: localeEnglish, loading: true}
	calls := 0
	a.openDirectoryDialog = func(mygo.OpenDialogOptions) ([]string, error) {
		calls++
		return nil, nil
	}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.emptyPage(c)
	}, 500, 480)
	tt.Frame()
	node, ok := tt.AccessByRole(ui.RoleButton, a.t("chooseFolder"))
	if !ok || !node.Disabled {
		t.Fatalf("Welcome Open must be disabled while repository restore is loading: %+v", node)
	}
	a.loading = false
	tt.Frame()
	clickRepositoryTabsControl(t, tt, ui.RoleButton, a.t("chooseFolder"))
	if calls != 1 {
		t.Fatalf("Welcome Open did not invoke the picker once after restore: %d", calls)
	}
}

func repositoryOpenFingerprint(t *testing.T, root string) string {
	t.Helper()
	result := ""
	for _, args := range [][]string{{"symbolic-ref", "HEAD"}, {"rev-parse", "HEAD"}, {"for-each-ref", "--format=%(refname) %(objectname)"}, {"ls-files", "--stage"}, {"show", ":base.txt"}} {
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("fixture fingerprint: %v: %s", err, out)
		}
		result += string(out) + "\x00"
	}
	data, err := os.ReadFile(filepath.Join(root, "base.txt"))
	if err != nil {
		t.Fatal(err)
	}
	return result + string(data)
}

func TestOpenRepositoryCancelFromCommitFieldsPreservesContext(t *testing.T) {
	for _, fieldKey := range []string{"commitMessage"} {
		t.Run(fieldKey, func(t *testing.T) {
			a, _, root := worktreeApp(t)
			a.window = &mygo.Window{}
			a.commitText = "retained title\n\nretained body"
			a.historyRef = historyRef{Kind: historyRefLocal, Name: "main"}
			a.selectedPath = "base.txt"
			a.selectedSHA = "old selection"
			before := repositoryOpenFingerprint(t, root)
			calls := 0
			a.openDirectoryDialog = func(opts mygo.OpenDialogOptions) ([]string, error) { calls++; return nil, nil }
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				a.registerShortcuts(c)
				a.commitMessageField(c)
			}, 500, 400)
			focusCommitField(t, tt, a.t(fieldKey))

			tt.Key(ui.Cmd, ui.KeyO)
			current, ok := tt.AccessByRole(ui.RoleTextField, a.t(fieldKey))
			if calls != 1 || !ok || !current.Focused {
				t.Fatal("actual commit field failed dispatch or lost focus after injected cancellation")
			}
			if a.repo.Root != root || a.commitText != "retained title\n\nretained body" || a.selectedPath != "base.txt" || a.selectedSHA != "old selection" || a.historyRef.Name != "main" || a.loading {
				t.Fatal("cancel changed existing repository context")
			}
			if repositoryOpenFingerprint(t, root) != before {
				t.Fatal("cancel changed Git state")
			}
		})
	}
}

func TestOpenRepositoryCancelFromHistorySearchPreservesQuery(t *testing.T) {
	a, _, root := worktreeApp(t)
	a.window = &mygo.Window{}
	a.historyQuery = "retained query"
	a.historyRef = historyRef{Kind: historyRefLocal, Name: "main"}
	a.selectedSHA = "old selection"
	calls := 0
	a.openDirectoryDialog = func(mygo.OpenDialogOptions) ([]string, error) { calls++; return nil, nil }
	takingText := false
	tt := ui.NewTester(func(c *ui.Context) {
		takingText = c.FocusTakesText()
		theme.Apply(c)
		a.registerShortcuts(c)
		ui.Column(c).Fill().Children(func() { a.historyPage(c) })
	}, 980, 640)
	tt.Key(ui.Cmd, ui.KeyF)
	if !takingText {
		t.Fatal("search shortcut did not focus text input")
	}
	tt.Key(ui.Cmd, ui.KeyO)
	if calls != 1 || a.historyQuery != "retained query" || a.repo.Root != root || a.historyRef.Name != "main" || a.selectedSHA != "old selection" || !takingText {
		t.Fatal("chooser cancellation changed history search context")
	}
}

func TestOpenRepositoryKeepsAcceptedFolderDuringExternalGitChange(t *testing.T) {
	isolateSettings(t)
	a, callbacks, root := worktreeApp(t)
	_, _, nextRoot := worktreeApp(t)
	tabs := newRepositoryTabs(a.ctx, a)
	t.Cleanup(tabs.shutdown)
	a.reload()
	for a.loading {
		runContinuation(t, callbacks)
	}
	if a.localWatch.stamp == "" || a.loadErr != "" {
		t.Fatalf("could not establish watcher baseline: %s", a.loadErr)
	}
	a.openDirectoryDialog = func(mygo.OpenDialogOptions) ([]string, error) {
		splitCommitGit(t, root, "commit", "--allow-empty", "-m", "External commit")
		a.localWatch.next = time.Time{}
		a.pollLocalChanges(time.Now())
		for a.localWatch.reading {
			runContinuation(t, callbacks)
		}
		return []string{nextRoot}, nil
	}
	tt := ui.NewTester(tabs.view, 980, 760)
	tt.SetTitleBar(ui.TitleBar{Height: 38, Left: 72})
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyO)
	for a.loading || tabs.opening != nil || tabs.active.loading {
		runContinuation(t, callbacks)
		tt.Frame()
	}
	if len(tabs.sessions) != 2 || tabs.active.repo == nil || tabs.active.repo.Root != nextRoot {
		t.Fatalf("accepted folder did not open in its own tab: tabs=%d active=%v", len(tabs.sessions), tabs.active.repo)
	}
}

func TestOpenRepositoryShortcutLoadsChosenDirectory(t *testing.T) {
	a, done, root := worktreeApp(t)
	_, _, nextRoot := worktreeApp(t)
	before := repositoryOpenFingerprint(t, root)
	nextBefore := repositoryOpenFingerprint(t, nextRoot)
	a.window = &mygo.Window{}
	calls := 0
	a.openDirectoryDialog = func(opts mygo.OpenDialogOptions) ([]string, error) {
		calls++
		if opts.Parent != a.window || !opts.Directory {
			t.Fatal("wrong chooser options")
		}
		return []string{nextRoot}, nil
	}
	tt := ui.NewTester(func(c *ui.Context) { a.registerShortcuts(c); ui.Column(c).Fill() }, 400, 300)
	tt.Key(ui.Cmd, ui.KeyO)
	if calls != 1 || !a.loading {
		t.Fatal("chosen path did not start existing repository load")
	}
	// The fake parent only identifies chooser ownership; completion runs headlessly.
	a.window = nil
	runContinuation(t, done)
	if a.repo == nil || a.repo.Root != nextRoot || a.loading || a.loadErr != "" || a.selectedSHA == "" || a.page != "history" {
		t.Fatalf("chosen repository was not loaded: %s", a.loadErr)
	}
	if repositoryOpenFingerprint(t, root) != before || repositoryOpenFingerprint(t, nextRoot) != nextBefore {
		t.Fatal("opening repository changed Git state")
	}
}
