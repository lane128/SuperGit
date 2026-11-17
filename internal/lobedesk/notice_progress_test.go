package lobedesk

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestUndoNoticeProgressUsesPausedToastLifetime(t *testing.T) {
	a := &app{locale: localeEnglish}
	acted := false
	a.pushNotice(toastMain, &toastNotice{kind: "success", title: "Line discarded", action: a.t("undo"), onAction: func() { acted = true }, timeout: 8 * time.Second})
	tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.toastHost(c, toastMain); a.flushNotice(c, toastMain) }, 560, 480)
	for range 6 {
		tt.Advance(100 * time.Millisecond)
	}
	if tt.FramePending() {
		t.Fatal("Undo countdown keeps requesting immediate frames after entry animation settles")
	}
	progress, ok := tt.AccessByRole(ui.RoleProgress, "Time remaining")
	if !ok || progress.Now <= 0 || progress.Now > 1 {
		t.Fatal("timed Undo has no visible remaining-time progress")
	}
	before := progress.Now
	tt.Advance(time.Second)
	progress, _ = tt.AccessByRole(ui.RoleProgress, "Time remaining")
	if progress.Now >= before {
		t.Fatal("countdown did not decrease")
	}
	tt.Move(progress.Bounds.X+progress.Bounds.W/2, progress.Bounds.Y+progress.Bounds.H/2)
	tt.Frame()
	paused, _ := tt.AccessByRole(ui.RoleProgress, "Time remaining")
	tt.Advance(10 * time.Second)
	progress, ok = tt.AccessByRole(ui.RoleProgress, "Time remaining")
	if !ok || progress.Now != paused.Now {
		t.Fatal("hover did not pause progress with the toast lifetime")
	}
	undo, ok := tt.AccessByRole(ui.RoleButton, a.t("undo"))
	if !ok {
		t.Fatal("paused toast lost Undo")
	}
	tt.ClickAt(undo.Bounds.X+undo.Bounds.W/2, undo.Bounds.Y+undo.Bounds.H/2)
	if !acted {
		t.Fatal("live Undo action did not execute")
	}
}

func TestUndoNoticeResumesAndExpiresAfterPointerLeaves(t *testing.T) {
	a := &app{locale: localeEnglish}
	a.pushNotice(toastMain, &toastNotice{title: "Temporary operation", action: "Undo temporary operation", onAction: func() {}, timeout: 8 * time.Second})
	tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.toastHost(c, toastMain); a.flushNotice(c, toastMain) }, 560, 480)
	for range 6 {
		tt.Advance(100 * time.Millisecond)
	}
	progress, ok := tt.AccessByRole(ui.RoleProgress, "Time remaining")
	if !ok {
		t.Fatal("missing Undo progress")
	}
	tt.Move(progress.Bounds.X+progress.Bounds.W/2, progress.Bounds.Y+progress.Bounds.H/2)
	tt.Frame()
	tt.Advance(10 * time.Second)
	paused, ok := tt.AccessByRole(ui.RoleProgress, "Time remaining")
	if !ok {
		t.Fatal("hovered Undo expired")
	}
	tt.Move(0, 0)
	tt.Frame()
	tt.Advance(time.Second)
	resumed, ok := tt.AccessByRole(ui.RoleProgress, "Time remaining")
	if !ok || resumed.Now >= paused.Now {
		t.Fatal("countdown did not resume after pointer left")
	}
	tt.Advance(8 * time.Second)
	for range 6 {
		tt.Advance(100 * time.Millisecond)
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Undo temporary operation"); ok {
		t.Fatal("resumed Undo did not expire")
	}
}

func TestKeyboardFocusExpandsOlderUndoNotice(t *testing.T) {
	a := &app{locale: localeEnglish}
	acted := false
	a.pushNotice(toastMain, &toastNotice{title: "Older operation", action: "Undo older operation", onAction: func() { acted = true }, timeout: 8 * time.Second})
	a.pushNotice(toastMain, &toastNotice{kind: "error", title: "New persistent failure", timeout: -1})
	tt := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		a.toastHost(c, toastMain)
		a.flushNotice(c, toastMain)
	}, 560, 480)
	for range 8 {
		tt.Advance(100 * time.Millisecond)
	}
	focused := false
	for range 6 {
		tt.Key(0, ui.KeyTab)
		if undo, ok := tt.AccessByRole(ui.RoleButton, "Undo older operation"); ok && undo.Focused {
			focused = true
			break
		}
	}
	if !focused {
		t.Fatal("keyboard cannot reach older Undo")
	}
	for range 8 {
		tt.Advance(100 * time.Millisecond)
	}
	older, ok := tt.Find("Older operation")
	front, frontOK := tt.Find("New persistent failure")
	if !ok || !frontOK || older.Y+older.H >= front.Y {
		t.Fatalf("keyboard focus leaves older toast hidden behind front card: older=%+v front=%+v", older, front)
	}
	tt.Advance(10 * time.Second)
	tt.Key(0, ui.KeySpace)
	if !acted {
		t.Fatal("focused older Undo expired or failed to execute")
	}
}

func TestPersistentNoticeHasNoExpiryProgress(t *testing.T) {
	a := &app{locale: localeEnglish}
	a.pushNotice(toastMain, &toastNotice{kind: "error", title: "Needs attention", timeout: -1})
	tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.toastHost(c, toastMain); a.flushNotice(c, toastMain) }, 560, 480)
	for range 6 {
		tt.Advance(100 * time.Millisecond)
	}
	if _, ok := tt.AccessByRole(ui.RoleProgress, "Time remaining"); ok {
		t.Fatal("persistent error falsely shows expiry")
	}
	tt.Advance(30 * time.Second)
	if !tt.HasText("Needs attention") {
		t.Fatal("persistent notice expired")
	}
}
