package lobedesk

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestCommitLongSubjectCanBeReadAndSelectedAtEnd(t *testing.T) {
	previous := uiFontScaleValue()
	setUIFontScale(fontScaleLarge)
	t.Cleanup(func() { setUIFontScale(previous) })
	subject := strings.Repeat("Long subject words ", 40) + "SUBJECT_END"
	commit := gitrepo.Commit{SHA: strings.Repeat("a", 40), Subject: subject, Author: "Fixture author", Date: "2026-11-18 12:34:56"}
	a := &app{}
	var scroll ui.ScrollState
	tt := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		ui.Column(c).Width(292).Height(260).OnClick(func() {}).Children(func() {
			layout.ScrollArea(c, layout.ScrollAreaProps{State: &scroll}, func() { a.commitInfoContent(c, commit, theme.TokensOf(c)) }).Fill()
		})
	}, 292, 260)
	title, ok := tt.Find(subject)
	if !ok || title.H < 120 {
		t.Fatalf("long subject still clipped to a short block: %+v", title)
	}
	tt.Scroll(140, 120, 0, 100000)
	tt.Frame()
	if scroll.MaxY <= 0 || scroll.Y < scroll.MaxY-1 {
		t.Fatalf("subject end unreachable: %+v", scroll)
	}
	title, ok = tt.Find(subject)
	if !ok {
		t.Fatal("title missing after scrolling")
	}
	y := title.Y + title.H - 5
	if y <= 0 || y >= 260 {
		t.Fatalf("last subject line outside viewport: y=%v", y)
	}
	tt.Press(title.X+1, y)
	tt.Move(title.X+title.W-1, y)
	tt.Release(title.X+title.W-1, y)
	tt.Key(ui.Cmd, ui.KeyC)
	if copied := tt.Clipboard(); !strings.HasSuffix(copied, "SUBJECT_END") {
		t.Fatalf("visible final title line could not be selected: %q", copied)
	}
}
