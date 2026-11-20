package lobedesk

import (
	"errors"
	"runtime"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestOpenRepositoryShortcutIsListed(t *testing.T) {
	for _, s := range shortcutTable() {
		if s.ID != "repo.open" {
			continue
		}
		want := "Ctrl+O"
		if runtime.GOOS == "darwin" {
			want = "⌘O"
		}
		if s.Mod != ui.Cmd || s.Key != ui.KeyO || s.Keys != want || s.Field || s.SkipWhenTyping || s.Run == nil {
			t.Fatalf("open repository shortcut has wrong scope or binding: %+v", s)
		}
		return
	}
	t.Fatal("missing open repository shortcut")
}

func TestOpenRepositoryShortcutDispatchesFromTextFields(t *testing.T) {
	for _, multiline := range []bool{false, true} {
		a := &app{window: &mygo.Window{}, locale: localeEnglish, page: "history"}
		value := "draft or search text"
		calls := 0
		a.openDirectoryDialog = func(opts mygo.OpenDialogOptions) ([]string, error) {
			calls++
			if opts.Parent != a.window || !opts.Directory || opts.Title != a.t("openRepo") {
				t.Fatal("wrong native chooser options")
			}
			return nil, nil
		}
		tt := ui.NewTester(func(c *ui.Context) {
			theme.Apply(c)
			a.registerShortcuts(c)
			if multiline {
				field.TextArea(c, &value, field.TextAreaProps{}).Label("editor").Focus()
			} else {
				field.Input(c, &value, field.InputProps{}).Label("editor").Focus()
			}
		}, 400, 200)
		tt.Key(ui.Cmd, ui.KeyO)
		if calls != 1 || value != "draft or search text" || a.repo != nil || a.page != "history" || a.loading {
			t.Fatal("open shortcut did not reach chooser or cancellation changed state")
		}
	}
}

func TestChooseRepositoryRespectsGuards(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*app)
	}{
		{"busy", func(a *app) { a.busy = true }},
		{"loading", func(a *app) { a.loading = true }},
		{"chooser", func(a *app) { a.repositoryChooserOpen = true }},
		{"confirmation", func(a *app) { a.confirm.open = true; a.confirm.owner = toastMain }},
		{"prompt", func(a *app) { a.prompt.open = true }},
		{"reset", func(a *app) { a.reset.open = true }},
		{"checkout", func(a *app) { a.checkoutBranch.open = true }},
		{"worktree", func(a *app) { a.worktreeForm.open = true }},
		{"blame", func(a *app) { a.blameOpen = true }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &app{window: &mygo.Window{}, locale: localeEnglish}
			calls := 0
			a.openDirectoryDialog = func(mygo.OpenDialogOptions) ([]string, error) { calls++; return nil, nil }
			tc.setup(a)
			// Both the mouse entry and the window shortcut use this guarded boundary.
			a.chooseRepository()
			tt := ui.NewTester(func(c *ui.Context) { a.registerShortcuts(c); ui.Column(c).Fill() }, 300, 200)
			tt.Key(ui.Cmd, ui.KeyO)
			if calls != 0 {
				t.Fatal("guarded request opened native chooser")
			}
		})
	}
}

func TestChooseRepositoryDoesNotReenter(t *testing.T) {
	a := &app{window: &mygo.Window{}, locale: localeEnglish}
	calls := 0
	a.openDirectoryDialog = func(mygo.OpenDialogOptions) ([]string, error) {
		calls++
		if calls == 1 {
			a.chooseRepository()
		}
		return nil, nil
	}
	a.chooseRepository()
	if calls != 1 || a.repositoryChooserOpen {
		t.Fatal("chooser reentered or remained locked after cancellation")
	}
	a.chooseRepository()
	if calls != 2 {
		t.Fatal("chooser did not unlock after cancellation")
	}
}

func TestChooseRepositoryRechecksStateAfterNativeDialog(t *testing.T) {
	for _, setup := range []func(*app){
		func(a *app) { a.busy = true },
		func(a *app) { a.loading = true },
		func(a *app) { a.prompt.open = true },
	} {
		a := &app{window: &mygo.Window{}, locale: localeEnglish}
		a.openDirectoryDialog = func(mygo.OpenDialogOptions) ([]string, error) { setup(a); return []string{t.TempDir()}, nil }
		a.chooseRepository()
		if a.loadSeq != 0 || a.repositoryChooserOpen {
			t.Fatal("native dialog result ignored current guard state or retained lock")
		}
	}
}

func TestChooseRepositoryErrorPreservesDraftAndUnlocks(t *testing.T) {
	a := &app{window: &mygo.Window{}, locale: localeEnglish, page: "history"}
	a.commitText = "draft title\n\ndraft body"
	a.openDirectoryDialog = func(mygo.OpenDialogOptions) ([]string, error) { return nil, errors.New("chooser failed") }
	a.chooseRepository()
	if a.repositoryChooserOpen || a.loading || a.loadSeq != 0 || a.commitText != "draft title\n\ndraft body" {
		t.Fatal("chooser failure changed draft or retained lock")
	}
	if len(a.notices[toastMain]) != 1 || a.notices[toastMain][0].description != "chooser failed" {
		t.Fatal("missing chooser error feedback")
	}
}
