package lobedesk

import (
	"fmt"
	"testing"
	"time"

	"supergit/internal/gitrepo"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestBlameEscapeWithHoveredBackgroundRef(t *testing.T) {
	a, done, _ := worktreeApp(t)
	commit := gitrepo.Commit{SHA: "head", Subject: "Background", Refs: "long-background-branch"}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Children(func() {
			ui.Box(c).Height(576).Shrink(0)
			a.commitInfoContent(c, commit, theme.TokensOf(c))
		})
		a.blameDialog(c)
	}, 980, 640)
	// The reference sits below the dialog's lower edge. Leave the real
	// pointer there while opening Blame through a keyboard/menu action.
	ref, ok := tt.AccessByRole(ui.RoleText, "long-background-branch")
	if !ok {
		t.Fatal("background reference missing")
	}
	if ref.Bounds.Y+ref.Bounds.H/2 <= 620 {
		t.Fatalf("reference must sit below the modal panel: %+v", ref.Bounds)
	}
	tt.Move(ref.Bounds.X+ref.Bounds.W/2, ref.Bounds.Y+ref.Bounds.H/2)
	tt.Advance(time.Second)
	if _, ok := tt.AccessByRole(ui.RoleTooltip, ""); !ok {
		t.Fatal("background reference tooltip did not appear before opening Blame")
	}
	a.openBlameAt("base.txt", "HEAD")
	runContinuation(t, done)
	tt.Frame()
	tt.Advance(time.Second)
	if _, ok := tt.AccessByRole(ui.RoleTooltip, ""); ok {
		t.Error("background tooltip remains visible behind Blame")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.blameOpen {
		t.Fatal("first Escape was consumed by a background reference tooltip")
	}
}

func TestBlameViewportFitsMinimumWindow(t *testing.T) {
	before := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(before) })
	for _, language := range []locale{localeEnglish, localeChinese} {
		for _, scale := range []float32{1, 1.15} {
			t.Run(fmt.Sprintf("%s/%.2f", language, scale), func(t *testing.T) {
				setUIFontScale(scale)
				a, done, _ := worktreeApp(t)
				a.locale = language
				a.openBlameAt("base.txt", "HEAD")
				runContinuation(t, done)
				tt := ui.NewTester(func(c *ui.Context) {
					theme.Apply(c)
					a.blameDialog(c)
				}, 980, 640)
				viewport, ok := tt.AccessByRole(ui.RoleScroll, "")
				if !ok {
					t.Fatal("Blame source viewport missing")
				}
				r := viewport.Bounds
				if r.X < 20 || r.Y < 20 || r.X+r.W > 960 || r.Y+r.H > 620 {
					t.Fatalf("Blame source viewport extends into window margin: %+v", r)
				}
				for _, control := range []struct {
					role  ui.Role
					label string
				}{
					{ui.RoleToggleButton, a.t("blameIgnoreWhitespace")},
					{ui.RoleButton, a.t("close")},
				} {
					item, found := tt.AccessByRole(control.role, control.label)
					if !found || item.Offscreen || item.Bounds.X < 20 || item.Bounds.Y < 20 || item.Bounds.X+item.Bounds.W > 960 || item.Bounds.Y+item.Bounds.H > 620 {
						t.Errorf("Blame control %q outside window: %+v", control.label, item)
					}
				}
			})
		}
	}
}

func TestBlameWhitespaceToggleExposesPointerAndKeyboardState(t *testing.T) {
	a, done, _ := worktreeApp(t)
	a.openBlameAt("base.txt", "HEAD")
	runContinuation(t, done)
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.blameDialog(c)
	}, 980, 640)
	label := a.t("blameIgnoreWhitespace")
	toggle, ok := tt.AccessByRole(ui.RoleToggleButton, label)
	if !ok || toggle.Checked {
		t.Fatal("whitespace option must expose an initially unchecked toggle")
	}
	tt.ClickAt(toggle.Bounds.X+toggle.Bounds.W/2, toggle.Bounds.Y+toggle.Bounds.H/2)
	runContinuation(t, done)
	tt.Frame()
	toggle, ok = tt.AccessByRole(ui.RoleToggleButton, label)
	if !ok || !toggle.Checked || !a.blameIgnoreWS {
		t.Fatal("pointer activation did not report the selected whitespace mode")
	}
	tt.Key(0, ui.KeySpace)
	runContinuation(t, done)
	tt.Frame()
	toggle, ok = tt.AccessByRole(ui.RoleToggleButton, label)
	if !ok || toggle.Checked || a.blameIgnoreWS {
		t.Fatal("Space did not clear the focused whitespace toggle")
	}
	if a.blamePath != "base.txt" || a.blameRev != "HEAD" || len(a.blame) != 1 {
		t.Fatal("changing whitespace mode lost the original blame target or content")
	}
}
