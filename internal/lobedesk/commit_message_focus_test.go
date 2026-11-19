package lobedesk

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func commitFocusBinding(t *testing.T) appShortcut {
	t.Helper()
	for _, s := range shortcutTable() {
		if s.ID == "workspace.focusCommitMessage" {
			return s
		}
	}
	t.Fatal("missing commit message focus binding")
	return appShortcut{}
}

func TestCommitMessageShortcutFocusesActualEditorOnce(t *testing.T) {
	for _, page := range []string{"changes", "history", "branches", "stash", "worktrees"} {
		t.Run(page, func(t *testing.T) {
			a, _ := overviewFixture(t)
			a.locale = localeEnglish
			a.page = page
			draft := "中文 summary\n\nbody line\n\n"
			a.commitText = draft
			a.amend = true
			binding := commitFocusBinding(t)
			keys := "Ctrl+9"
			if runtime.GOOS == "darwin" {
				keys = "⌘9"
			}
			if binding.Key != ui.Key9 || binding.Mod != ui.Cmd || binding.Keys != keys || binding.SkipWhenTyping || binding.Field {
				t.Fatalf("incorrect binding: %+v", binding)
			}
			fingerprint := func() string {
				cmd := exec.Command("git", "-C", a.repo.Root, "ls-files", "--stage", "-z")
				cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
				b, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}
			index := fingerprint()
			var other string
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				a.registerShortcuts(c)
				ui.Column(c).Width(550).Children(func() {
					if a.page == "changes" {
						a.commitMessageFields(c)
					} else {
						field.Input(c, &other, field.InputProps{}).Label("other page")
					}
				})
			}, 600, 400)
			if page == "changes" {
				node, ok := tt.AccessByRole(ui.RoleTextField, a.t("commitDescription"))
				if !ok {
					t.Fatal("missing body")
				}
				tt.ClickAt(node.Bounds.X+node.Bounds.W/2, node.Bounds.Y+node.Bounds.H/2)
			}
			tt.Key(binding.Mod, binding.Key)
			tt.Frame()
			if a.page != "changes" || !tt.Focused(a.t("commitSummary")) || a.commitText != draft || !a.amend {
				t.Fatal("navigation did not preserve draft/amend and focus actual summary")
			}
			tt.Key(0, ui.KeyEnd)
			tt.Type(" marker")
			if a.commitFields.Summary != "中文 summary marker" || a.commitFields.Body != "body line\n\n" {
				t.Fatalf("typing missed summary: %#v", a.commitFields)
			}
			tt.Key(0, ui.KeyTab)
			tt.Frame()
			tt.Frame()
			if !tt.Focused(a.t("commitDescription")) {
				t.Fatal("one-shot request stole focus after Tab")
			}
			if fingerprint() != index {
				t.Fatal("focus navigation changed Index")
			}
			data, err := os.ReadFile(filepath.Join(a.repo.Root, "same.go"))
			if err != nil || string(data) != "working version\n" {
				t.Fatal("focus navigation changed working bytes")
			}
		})
	}
}

func TestCommitMessageShortcutGuardsAndStaleRequests(t *testing.T) {
	s := commitFocusBinding(t)
	for _, blocked := range []string{"nil", "busy", "loading", "history refresh", "chooser", "ref popup", "clone", "new", "confirm"} {
		t.Run(blocked, func(t *testing.T) {
			a, _ := overviewFixture(t)
			a.page = "stash"
			switch blocked {
			case "nil":
				a.repo = nil
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "history refresh":
				a.historyReloading = true
			case "chooser":
				a.repositoryChooserOpen = true
			case "ref popup":
				a.historyRefOpen = true
			case "clone":
				a.cloneForm.open = true
			case "new":
				a.newRepoForm.open = true
			case "confirm":
				a.confirm.open = true
			}
			tt := ui.NewTester(func(c *ui.Context) { a.registerShortcuts(c); ui.Column(c).Fill() }, 400, 200)
			if s.Available == nil || s.Available(a) {
				t.Fatal("blocked shortcut available")
			}
			tt.Key(s.Mod, s.Key)
			s.Run(a)
			if a.page != "stash" {
				t.Fatal("blocked request changed page")
			}
		})
	}
	for _, change := range []string{"root", "sequence", "page", "dialog"} {
		t.Run("stale "+change, func(t *testing.T) {
			a, _ := overviewFixture(t)
			a.page = "changes"
			s.Run(a)
			switch change {
			case "root":
				a.repo.Root = t.TempDir()
			case "sequence":
				a.loadSeq++
			case "page":
				a.page = "stash"
			case "dialog":
				a.newRepoForm.open = true
			}
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				a.registerShortcuts(c)
				ui.Column(c).Width(550).Children(func() { a.commitMessageFields(c) })
			}, 600, 400)
			if tt.Focused(a.t("commitSummary")) {
				t.Fatal("stale request focused editor")
			}
			a.page = "changes"
			a.newRepoForm.open = false
			tt.Frame()
			if tt.Focused(a.t("commitSummary")) {
				t.Fatal("stale request survived cancellation")
			}
		})
	}
}
