package ui

import (
	"github.com/egoist/mygo/internal/platform"
	"testing"
)

func TestDeferredInputFocusTransitionPreservesBurstText(t *testing.T) {
	for _, mods := range []Modifiers{Cmd, Cmd | Shift} {
		open, focus, namePhase := false, false, false
		body, query, name := "body", "", ""
		tt := NewTester(func(c *Context) {
			Column(c).Fill().Children(func() {
				TextAreaBase(c, &body).Label("Body").Height(80)
			})
			c.OnWindowShortcut(mods, KeyP, func() { open, focus = true, true })
			var field Element
			c.Key("one-command-session")
			DialogBase(c, &open, func(backdrop, panel Element) {
				panel.Column().Width(260).Children(func() {
					if namePhase {
						field = TextInputBase(c.Key("name"), &name).Label("Name")
						field.OnTextKey(func(k TextKey) bool {
							if k.Key == KeyBackspace && k.Mods == 0 && k.Text() == "" {
								namePhase, focus = false, true
								return true
							}
							return false
						})
					} else {
						field = TextInputBase(c.Key("query"), &query).Label("Query")
						field.OnSubmit(func() { namePhase, focus = true, true })
					}
				})
			})
			if open && focus {
				field.Focus()
				focus = false
			}
		}, 400, 260)
		tt.SetFocused(true)
		if err := tt.Click("Body"); err != nil {
			t.Fatal(err)
		}
		tt.Key(0, KeyEnd)
		key := func(m Modifiers, k Key) {
			tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(k), Mods: platform.Modifiers(m)})
			tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.Key(k), Mods: platform.Modifiers(m)})
		}
		text := func(s string) { tt.rt.event(platform.SurfaceEvent{Kind: platform.TextInput, Text: s}) }
		// No display frames between opening, typing, entering parameters,
		// returning, cancelling and typing into the restored original editor.
		key(mods, KeyP)
		text("crbr")
		key(0, KeyEnter)
		key(0, KeyBackspace)
		key(0, KeyEscape)
		text("X")
		tt.Frame()
		if open || namePhase || query != "crbr" || body != "bodyX" || name != "" {
			t.Fatalf("mods=%v open=%v parameters=%v query=%q name=%q body=%q", mods, open, namePhase, query, name, body)
		}
	}
}

func TestOrdinaryBurstTextStillBatchesUntilDisplayFrame(t *testing.T) {
	value := ""
	frames := 0
	tt := NewTester(func(c *Context) { frames++; TextInputBase(c, &value).Label("Input") }, 300, 100)
	tt.SetFocused(true)
	if err := tt.Click("Input"); err != nil {
		t.Fatal(err)
	}
	before := frames
	tt.rt.event(platform.SurfaceEvent{Kind: platform.TextInput, Text: "a"})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.TextInput, Text: "b"})
	if frames != before {
		t.Fatal("ordinary typing forced synchronous frames")
	}
	tt.Frame()
	if value != "ab" {
		t.Fatalf("batched text=%q", value)
	}
}
