package lobedesk

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestMessageWithoutTargetsKeepsSDKCopyAndSelectAll(t *testing.T) {
	const body = "Plain 作者 message without navigation targets"
	a := &app{locale: localeEnglish, commitMessage: "Selected subject\n\n" + body}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Children(func() { a.commitInfoContent(c, gitrepo.Commit{Subject: "Selected subject"}, theme.TokensOf(c)) })
	}, 800, 400)
	r, ok := tt.Find(body)
	if !ok {
		t.Fatal("plain message absent")
	}
	y := r.Y + r.H/2
	tt.Press(r.X+1, y)
	tt.Move(r.X+68, y)
	tt.Release(r.X+68, y)
	tt.Key(ui.Cmd, ui.KeyC)
	baseline := tt.Clipboard()
	if baseline == "" || baseline == body {
		t.Fatal("fixture did not select partial text")
	}
	tt.RightClickAt(r.X+10, y)
	for _, label := range []string{"Copy", "Select All"} {
		found := false
		for _, item := range tt.Menu() {
			if item == label {
				found = true
			}
		}
		if !found {
			t.Fatalf("plain message lost SDK %s: %v", label, tt.Menu())
		}
	}
	if err := tt.ChooseMenuItem("Copy"); err != nil {
		t.Fatal(err)
	}
	if got := tt.Clipboard(); got != baseline {
		t.Fatalf("SDK menu copied %q want %q", got, baseline)
	}
	tt.RightClickAt(r.X+10, y)
	if err := tt.ChooseMenuItem("Select All"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyC)
	if got := tt.Clipboard(); !strings.Contains(got, "Selected subject\n"+body) {
		t.Fatalf("SDK Select All lost the complete message: %q", got)
	}
	if len(tt.OpenedURLs()) != 0 || len(a.notices[toastMain]) != 0 {
		t.Fatal("plain SDK text commands triggered app target actions")
	}
}
