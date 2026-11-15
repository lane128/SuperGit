package lobedesk

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// The details pane lives in a scroll area, and a focusable scroll area stops
// MyGo from treating a paragraph as selectable unless a Selectable container
// sits between the two. A drag over the message must copy the words it covers.
func TestCommitMessageTextCanBeSelected(t *testing.T) {
	const body = "Body text to copy"
	commit := gitrepo.Commit{
		SHA:     "0123456789abcdef0123456789abcdef01234567",
		Subject: "Subject",
		Author:  "Ada",
		Date:    "2026-11-14 03:00:00",
	}
	a := &app{commitMessage: "Subject\n\n" + body}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		// A clickable ancestor stands in for the controls the details pane really
		// sits under; MyGo refuses to select text under one unless a Selectable
		// container sits below it.
		ui.Column(c).Width(600).Height(300).OnClick(func() {}).Children(func() {
			layout.ScrollArea(c, layout.ScrollAreaProps{}, func() {
				ui.Column(c).Width(560).Padding(10).Children(func() {
					a.commitInfoContent(c, commit, theme.TokensOf(c))
				})
			})
		})
	}, 640, 300)
	r, ok := tt.Find(body)
	if !ok {
		t.Fatalf("the message %q is not laid out", body)
	}
	mid := r.Y + r.H/2
	tt.Press(r.X+1, mid)
	tt.Move(r.X+r.W/2, mid)
	tt.Move(r.X+r.W-1, mid)
	tt.Release(r.X+r.W-1, mid)
	tt.Key(ui.Cmd, ui.KeyC)
	if got := tt.Clipboard(); got != body {
		t.Errorf("dragging the message copied %q; want %q", got, body)
	}
}

// The settings path sits at the bottom of a scrolling page, under the same
// clickable ancestors. Its row must be a selectable scope of its own.
func TestSettingsPathTextCanBeSelected(t *testing.T) {
	path, err := settingsPath()
	if err != nil {
		t.Skipf("no settings path: %v", err)
	}
	a := &app{locale: localeChinese}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		// The page is taller than a window: every card has to be laid out for
		// the row at the foot to have a selectable scope.
		ui.Column(c).Width(1000).Height(2000).OnClick(func() {}).Children(func() {
			a.settingsPage(c)
		})
	}, 1000, 2000)
	r, ok := tt.Find(path)
	if !ok {
		t.Fatalf("the settings path %q is not laid out", path)
	}
	mid := r.Y + r.H/2
	tt.Press(r.X+1, mid)
	tt.Move(r.X+r.W/2, mid)
	tt.Move(r.X+r.W-1, mid)
	tt.Release(r.X+r.W-1, mid)
	tt.Key(ui.Cmd, ui.KeyC)
	// The path is one line, so a narrow card shows a prefix of it and the drag
	// copies what is on screen. What matters here is that the row selects at
	// all: without its own selectable scope, the drag under the clickable page
	// copies nothing.
	if got := tt.Clipboard(); got == "" || !strings.HasPrefix(path, got) {
		t.Errorf("dragging the settings path copied %q; want it to select the path shown in the row (%q)", got, path)
	}
}
