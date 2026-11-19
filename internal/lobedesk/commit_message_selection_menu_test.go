package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"strings"
	"supergit/internal/gitrepo"
	"testing"
)

// Right-clicking inside an existing selection must retain it. Right-clicking
// outside it intentionally moves the caret in the pinned SDK.
func TestMessageSelectionSurvivesContextCancelAndOpen(t *testing.T) {
	const raw = "https://example.invalid/docs"
	const body = "Before https://example.invalid/docs after"
	a := &app{locale: localeEnglish, commitMessage: "Selected\n\n" + body}
	commit := gitrepo.Commit{Subject: "Selected"}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Children(func() { a.commitInfoContent(c, commit, theme.TokensOf(c)) })
	}, 640, 360)
	r, ok := tt.Find(body)
	if !ok {
		t.Fatal("body absent")
	}
	y := r.Y + r.H/2
	tt.Press(r.X+1, y)
	tt.Move(r.X+30, y)
	tt.Release(r.X+30, y)
	tt.Key(ui.Cmd, ui.KeyC)
	selected := tt.Clipboard()
	if selected == "" || selected == body || !strings.HasPrefix(body, selected) {
		t.Fatalf("fixture did not select a prefix: %q", selected)
	}
	tt.RightClickAt(r.X+10, y)
	tt.Key(0, ui.KeyEscape)
	tt.SetClipboard("sentinel")
	tt.Key(ui.Cmd, ui.KeyC)
	if got := tt.Clipboard(); got != selected {
		t.Fatalf("cancel lost partial selection: %q, want %q", got, selected)
	}
	tt.RightClickAt(r.X+10, y)
	item, ok := tt.AccessByRole(ui.RoleMenuItem, raw)
	if !ok {
		t.Fatal("URL action absent")
	}
	tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
	tt.SetClipboard("sentinel")
	tt.Key(ui.Cmd, ui.KeyC)
	if got := tt.Clipboard(); got != selected {
		t.Fatalf("open lost partial selection: %q, want %q", got, selected)
	}
	if got := tt.OpenedURLs(); len(got) != 1 || got[0] != raw {
		t.Fatalf("wrong URL open: %v", got)
	}
}
