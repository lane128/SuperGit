package lobedesk

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestBlankRepositoryTabJourney(t *testing.T) {
	isolateSettings(t)
	a, _, aRoot := worktreeApp(t)
	_, _, bRoot := worktreeApp(t)
	const subject = "QA history subject with enough words to test the narrow list"
	const code = "QA B dedicated patch line"
	if err := os.WriteFile(filepath.Join(bRoot, "base.txt"), []byte("base\n"+code+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	splitCommitGit(t, bRoot, "add", "--", "base.txt")
	splitCommitGit(t, bRoot, "commit", "-m", subject)
	fingerprintA, fingerprintB := repositoryOpenFingerprint(t, aRoot), repositoryOpenFingerprint(t, bRoot)
	for _, tag := range []string{"v1", "v2", "v3"} {
		splitCommitGit(t, bRoot, "tag", tag)
	}
	fingerprintB = repositoryOpenFingerprint(t, bRoot)
	a.locale, a.page, a.showGraph = localeEnglish, "history", true
	// Opening a repository selects the working tree's row, whose docked commit
	// box is the editor the journey types into.
	a.worktreeSelected = true
	callbacks := make(chan func(), 128)
	a.runOnMain = func(fn func()) { callbacks <- fn }
	var selected string
	calls := 0
	a.openDirectoryDialog = func(options mygo.OpenDialogOptions) ([]string, error) {
		calls++
		if !options.Directory {
			t.Fatal("open must request directories")
		}
		if selected == "" {
			return nil, nil
		}
		return []string{selected}, nil
	}
	tabs := newRepositoryTabs(a.ctx, a)
	t.Cleanup(tabs.shutdown)
	sessionPath := filepath.Join(t.TempDir(), "repository-session.json")
	tabs.sessionState.path = sessionPath
	if err := writeRepositorySession(sessionPath, repositorySession{Roots: []string{aRoot}, Active: aRoot}); err != nil {
		t.Fatal(err)
	}
	tabs.active.reload()
	view := ui.NewTester(tabs.view, 1320, 840)
	view.SetTitleBar(ui.TitleBar{Height: 38, Left: 72})
	wait := func() {
		t.Helper()
		for tabs.opening != nil || tabs.active.loading || tabs.active.commitFilesLoading || tabs.active.commitDiffLoading {
			runContinuation(t, callbacks)
			view.Frame()
		}
		view.Frame()
		view.Frame()
	}
	click := func(role ui.Role, label string) {
		t.Helper()
		clickRepositoryTabsControl(t, view, role, label)
		wait()
	}
	add := func() {
		t.Helper()
		label := a.t("newRepositoryTab")
		button, ok := view.AccessByRole(ui.RoleButton, label)
		if !ok || button.Offscreen || button.Disabled || button.Bounds.Y >= 38 {
			t.Fatalf("new-tab control missing from the title strip: %+v", button)
		}
		// Sample the text region to the right of the icon, away from the
		// button edges. The accessible label alone doesn't paint any text.
		bounds, img := button.Bounds, view.Image()
		painted := 0
		for y := int(bounds.Y + 7); y < int(bounds.Y+bounds.H-7); y++ {
			for x := int(bounds.X + 32); x < int(bounds.X+bounds.W-6); x++ {
				pixel := img.RGBAAt(x, y)
				if pixel.R < 160 && pixel.G < 160 && pixel.B < 175 {
					painted++
				}
			}
		}
		if painted < 12 {
			t.Fatalf("new-tab label should be painted before hovering: text pixels=%d button=%+v", painted, bounds)
		}
		view.ClickAt(bounds.X+bounds.W-12, bounds.Y+bounds.H/2)
		wait()
	}
	wait()
	const aSummary, aBody = "Alpha draft summary", "Alpha draft body"
	focusCommitField(t, view, a.t("commitSummary"))
	view.Type(aSummary)
	focusCommitField(t, view, a.t("commitDescription"))
	view.Type(aBody)
	if a.commitText != aSummary+"\n\n"+aBody {
		t.Fatalf("setup failed to type Alpha draft: %q", a.commitText)
	}

	add()
	blank := tabs.active
	if calls != 0 || blank == a || blank.repo != nil || len(tabs.sessions) != 2 || tabs.sessions[0] != a || tabs.sessions[1] != blank {
		t.Fatalf("plus did not create an active blank next to Alpha: chooser calls=%d sessions=%d", calls, len(tabs.sessions))
	}
	if alpha, ok := view.AccessByRole(ui.RoleTab, filepath.Base(aRoot)); !ok || alpha.Offscreen || alpha.Disabled {
		t.Fatalf("Alpha inaccessible while blank active: %+v", alpha)
	}
	const openWelcomeRepository = "Open Repository…"
	click(ui.RoleButton, openWelcomeRepository)
	if calls != 1 || tabs.active != blank || blank.repo != nil || len(tabs.sessions) != 2 || blank.loading || blank.repositoryChooserOpen || a.commitText != aSummary+"\n\n"+aBody {
		t.Fatal("cancel disturbed the blank tab or original draft")
	}
	tabs.saveSession()
	if saved := readRepositorySession(sessionPath); !slices.Equal(saved.Roots, []string{aRoot}) || saved.Active != aRoot {
		t.Fatalf("active blank disturbed saved workspace: %+v", saved)
	}

	selected = t.TempDir()
	click(ui.RoleButton, blank.t("chooseFolder"))
	if tabs.active != blank || blank.repo != nil || len(tabs.sessions) != 2 || blank.loadErr == "" {
		t.Fatal("invalid choice did not leave a blank tab with recoverable error")
	}
	selected = bRoot
	click(ui.RoleButton, blank.t("chooseFolder"))
	b := tabs.active
	if b != blank || b.repo == nil || b.repo.Root != bRoot || len(tabs.sessions) != 2 || tabs.sessions[1] != blank || b.loadErr != "" || b.commitText != "" {
		t.Fatal("valid choice did not fill the blank in place")
	}
	const bSummary = "Bravo draft summary"
	// The commit editor lives in the working tree's row, which a repository
	// does not open on; there is no separate Changes page to click.
	selectWorktreeRow(t, view, b)
	focusCommitField(t, view, b.t("commitSummary"))
	view.Type(bSummary)
	click(ui.RoleTab, filepath.Base(aRoot))
	if tabs.active != a || a.commitText != aSummary+"\n\n"+aBody {
		t.Fatal("returning to Alpha lost its draft")
	}
	click(ui.RoleTab, filepath.Base(bRoot))
	if tabs.active != b || b.commitText != bSummary {
		t.Fatal("returning to Bravo lost its draft")
	}

	click(ui.RoleOption, subject)
	click(ui.RoleOption, "base.txt")
	head := b.selectedSHA
	if head == "" || head != b.status.Head {
		t.Fatal("failed to select Bravo history commit")
	}
	for _, size := range [][2]int{{1320, 840}, {980, 640}, {1320, 840}} {
		view.SetSize(size[0], size[1])
		for range 5 {
			view.Frame()
		}
		for _, item := range []struct {
			role ui.Role
			name string
		}{{ui.RoleOption, subject}, {ui.RoleOption, "base.txt"}, {ui.RoleTab, "base.txt"}, {ui.RoleInsertion, "+" + code}} {
			node, ok := view.AccessByRole(item.role, item.name)
			if !ok || node.Offscreen || node.Bounds.W <= 0 || node.Bounds.H <= 0 {
				t.Errorf("%v: unavailable %q: %+v", size, item.name, node)
			}
		}
		if b.selectedSHA != head || b.commitPath != "base.txt" || b.detailTabs().Active.File.Path != "base.txt" || b.commitDiffViewport.W < historyDiffPaneMin-0.5 {
			t.Errorf("%v: selection or patch viewport regressed: sha=%s path=%s tab=%+v patch=%+v", size, b.selectedSHA, b.commitPath, b.detailTabs().Active, b.commitDiffViewport)
		}
	}

	// The keyboard entry creates exactly one blank. Opening an alias of A
	// reuses A, while B and the blank stay independently selectable.
	view.Key(ui.Cmd, ui.KeyT)
	wait()
	aliasTab := tabs.active
	if aliasTab.repo != nil || len(tabs.sessions) != 3 {
		t.Fatal("New Tab shortcut did not leave one additional Welcome")
	}
	tabs.saveSession()
	if saved := readRepositorySession(sessionPath); saved.Active != bRoot {
		t.Fatalf("Welcome lost the last selected checkout: %+v", saved)
	}
	selected = filepath.Join(aRoot, "nested")
	if err := os.Mkdir(selected, 0755); err != nil {
		t.Fatal(err)
	}
	click(ui.RoleButton, aliasTab.t("chooseFolder"))
	if tabs.active != a || aliasTab.repo != nil || len(tabs.sessions) != 3 || a.commitText != aSummary+"\n\n"+aBody {
		t.Fatal("opening an alias from Welcome duplicated or reset the existing checkout")
	}
	click(ui.RoleButton, a.t("close")+" "+a.t("welcomeTab"))
	click(ui.RoleTab, filepath.Base(bRoot))
	if len(tabs.sessions) != 2 || tabs.active != b || b.selectedSHA != head || b.commitText != bSummary {
		t.Fatal("closing Welcome changed a loaded tab")
	}
	view.Key(ui.Cmd, ui.KeyW)
	view.Frame()
	click(ui.RoleButton, b.t("cancel"))
	if len(tabs.sessions) != 2 || tabs.active != b || b.selectedSHA != head || b.commitText != bSummary {
		t.Fatal("cancelling Close Tab changed selection or draft")
	}

	tabs.saveSession()
	if saved := readRepositorySession(sessionPath); !slices.Equal(saved.Roots, []string{aRoot, bRoot}) || saved.Active != bRoot {
		t.Fatalf("filled tabs did not save in order: %+v", saved)
	}
	restored, restoreCallbacks := startRepositorySession(t, a, sessionPath, "")
	if saved := restored.repositorySession(); !slices.Equal(saved.Roots, []string{aRoot, bRoot}) || saved.Active != bRoot || restored.active.commitText != bSummary {
		t.Fatalf("saved workbench did not reopen on Bravo with its draft: %+v", saved)
	}
	restored.activate(restored.sessions[0], false)
	for restored.active.loading {
		runContinuation(t, restoreCallbacks)
	}
	if restored.active.commitText != aSummary+"\n\n"+aBody {
		t.Fatal("Alpha did not restore its own title and body")
	}
	if repositoryOpenFingerprint(t, aRoot) != fingerprintA || repositoryOpenFingerprint(t, bRoot) != fingerprintB {
		t.Fatal("journey mutated a fixture checkout")
	}
}
