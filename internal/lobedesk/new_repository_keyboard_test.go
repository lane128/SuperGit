package lobedesk

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func newRepoKeyboardTester(t *testing.T, existing bool) (*app, *ui.Tester, chan func(), string) {
	t.Helper()
	a := &app{locale: localeEnglish, page: "history"}
	old := ""
	if existing {
		a, _, old = worktreeApp(t)
		a.locale = localeEnglish
	} else {
		t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
		t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.ctx = ctx
	done := make(chan func(), 128)
	a.runOnMain = func(fn func()) { done <- fn }
	t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
	a.commitText = "retained draft"
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Children(func() {
			if existing {
				a.sidebarHeader(c)
			} else {
				a.emptyPage(c)
			}
			a.newRepositoryDialog(c)
		})
	}, 980, 640)
	return a, tt, done, old
}

func openNewRepoKeyboardEntrance(t *testing.T, a *app, tt *ui.Tester, existing bool) {
	t.Helper()
	if existing {
		title, ok := tt.AccessByRole(ui.RoleButton, a.repoName())
		if !ok {
			t.Fatal("missing repository menu")
		}
		tt.ClickAt(title.Bounds.X+title.Bounds.W/2, title.Bounds.Y+title.Bounds.H/2)
	}
	role := ui.RoleButton
	if existing {
		role = ui.RoleMenuItem
	}
	entry, ok := tt.AccessByRole(role, a.t("newRepo"))
	if !ok {
		t.Fatal("missing New repository entrance")
	}
	tt.ClickAt(entry.Bounds.X+entry.Bounds.W/2, entry.Bounds.Y+entry.Bounds.H/2)
	if !a.newRepoForm.open {
		t.Fatal("entrance failed to open form")
	}
}

func TestNewRepositoryRealTypingCancelAndReopen(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty", true: "existing"}[existing], func(t *testing.T) {
			a, tt, _, old := newRepoKeyboardTester(t, existing)
			dest := filepath.Join(t.TempDir(), "未创建 repo")
			for _, cancel := range []string{"escape", "button"} {
				openNewRepoKeyboardEntrance(t, a, tt, existing)
				tt.Type(dest)
				if a.newRepoForm.path != dest {
					t.Fatal("opening did not focus destination for real typing")
				}
				if cancel == "escape" {
					tt.Key(0, ui.KeyEscape)
				} else {
					button, ok := tt.AccessByRole(ui.RoleButton, a.t("cancel"))
					if !ok {
						t.Fatal("missing Cancel")
					}
					tt.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
				}
				tt.Frame()
				if a.newRepoForm.open || a.busy || a.loading || a.commitText != "retained draft" || existing && a.repo.Root != old {
					t.Fatal("cancel changed repository or draft")
				}
				if _, err := os.Lstat(dest); !os.IsNotExist(err) {
					t.Fatalf("cancel created destination: %v", err)
				}
			}
		})
	}
}

func TestNewRepositoryRealTypingInvalidPaths(t *testing.T) {
	a, tt, _, old := newRepoKeyboardTester(t, true)
	openNewRepoKeyboardEntrance(t, a, tt, true)
	tt.Type("   ")
	create, ok := tt.AccessByRole(ui.RoleButton, a.t("create"))
	if !ok || !create.Disabled {
		t.Fatal("blank destination permits creation")
	}
	tt.Key(0, ui.KeyEnter)
	if a.busy || a.loading || a.repo.Root != old || a.commitText != "retained draft" {
		t.Fatal("blank Enter changed state")
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("relative/path")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.newRepoForm.err != a.t("cloneAbsolutePath") || a.newRepoForm.path != "relative/path" || a.busy || a.loading || a.repo.Root != old || a.commitText != "retained draft" {
		t.Fatal("relative destination did not retain correction context")
	}
}

func TestNewRepositoryShortWindowErrorControls(t *testing.T) {
	before := uiFontScaleValue()
	setUIFontScale(fontScaleLarge)
	t.Cleanup(func() { setUIFontScale(before) })
	for _, language := range []locale{localeEnglish, localeChinese} {
		for _, mode := range []string{"light", "dark"} {
			t.Run(string(language)+"/"+mode, func(t *testing.T) {
				a := &app{locale: language, themeMode: mode}
				a.showNewRepository()
				a.newRepoForm.err = strings.Repeat("Long error 路径信息 ", 40)
				tt := ui.NewTester(func(c *ui.Context) {
					a.applyTheme(c)
					if c.Theme().Dark != (mode == "dark") {
						t.Fatal("incorrect rendered theme")
					}
					ui.Column(c).Fill().Children(func() { a.newRepositoryDialog(c) })
				}, 360, 360)
				tt.Type("/temporary/未创建 repo")
				if a.newRepoForm.path != "/temporary/未创建 repo" {
					t.Fatal("narrow form lost typing focus")
				}
				for _, label := range []string{a.t("cancel"), a.t("create")} {
					button, ok := tt.AccessByRole(ui.RoleButton, label)
					if !ok || button.Disabled || button.Bounds.W <= 0 || button.Bounds.H <= 0 || button.Bounds.X < 0 || button.Bounds.Y < 0 || button.Bounds.X+button.Bounds.W > 360 || button.Bounds.Y+button.Bounds.H > 360 {
						t.Fatalf("inaccessible short-window button %q: %+v", label, button.Bounds)
					}
				}
				cancel, _ := tt.AccessByRole(ui.RoleButton, a.t("cancel"))
				tt.ClickAt(cancel.Bounds.X+cancel.Bounds.W/2, cancel.Bounds.Y+cancel.Bounds.H/2)
				tt.Frame()
				if a.newRepoForm.open {
					t.Fatal("short-window Cancel was not reachable")
				}
			})
		}
	}
}

func TestNewRepositoryRealTypingEnterCorrection(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty", true: "existing"}[existing], func(t *testing.T) {
			a, tt, done, old := newRepoKeyboardTester(t, existing)
			occupied := t.TempDir()
			sentinel := filepath.Join(occupied, "keep.txt")
			if err := os.WriteFile(sentinel, []byte("keep bytes"), 0600); err != nil {
				t.Fatal(err)
			}
			openNewRepoKeyboardEntrance(t, a, tt, existing)
			tt.Type(occupied)
			if a.newRepoForm.path != occupied {
				t.Fatal("destination did not receive typed path")
			}
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			if a.newRepoForm.err == "" || a.newRepoForm.path != occupied || a.busy || a.loading || a.commitText != "retained draft" || existing && a.repo.Root != old {
				t.Fatal("occupied destination did not preserve correction context")
			}
			data, err := os.ReadFile(sentinel)
			if err != nil || string(data) != "keep bytes" {
				t.Fatal("occupied destination modified")
			}
			dest := filepath.Join(t.TempDir(), "新建 repo")
			tt.Key(ui.Cmd, ui.KeyA)
			tt.Type(dest)
			if a.newRepoForm.path != dest {
				t.Fatal("error lost input focus or selection replacement")
			}
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			if !a.busy || a.commitText != "retained draft" || existing && a.repo.Root != old {
				t.Fatal("corrected Enter failed or switched early")
			}
			runContinuation(t, done)
			for a.loading {
				runContinuation(t, done)
			}
			canonical, err := filepath.EvalSymlinks(dest)
			if err != nil {
				t.Fatal(err)
			}
			if a.repo == nil || a.repo.Root != canonical || a.newRepoForm.open || a.commitText != "" || a.status.Head != "" || len(a.commits) != 0 || a.loadErr != "" {
				t.Fatal("corrected Enter did not open unborn repository")
			}
			head, err := exec.Command("git", "-C", dest, "symbolic-ref", "HEAD").Output()
			if err != nil || strings.TrimSpace(string(head)) != "refs/heads/main" {
				t.Fatalf("wrong unborn branch: %s %v", head, err)
			}
			index, err := exec.Command("git", "-C", dest, "ls-files", "--stage").Output()
			if err != nil || len(index) != 0 {
				t.Fatalf("nonempty new Index: %s %v", index, err)
			}
		})
	}
}
