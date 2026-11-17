package lobedesk

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func noticeLifetimeTester(a *app) *ui.Tester {
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.toastHost(c, toastMain)
		a.flushNotice(c, toastMain)
		a.noticeDetailsDialog(c, toastMain)
	}, 980, 640)
	for range 5 {
		tt.Advance(50 * time.Millisecond)
		tt.Frame()
	}
	return tt
}

func TestActionableErrorRemainsAvailableUntilHandled(t *testing.T) {
	for _, recovery := range []bool{false, true} {
		t.Run(map[bool]string{false: "details", true: "recovery"}[recovery], func(t *testing.T) {
			a := &app{locale: localeEnglish}
			called := false
			raw := strings.Repeat("original Git failure\n", 20)
			n := &toastNotice{kind: "error", title: "Operation failed", description: raw}
			label := "View details"
			if recovery {
				label = "Retry"
				n.description, n.action, n.onAction = "Try the operation again", label, func() { called = true }
			}
			a.pushNotice(toastMain, n)
			tt := noticeLifetimeTester(a)
			tt.Advance(30 * time.Second)
			tt.Frame()
			tt.Advance(time.Second)
			tt.Frame()
			button, ok := tt.Find(label)
			if !ok {
				t.Fatal("error action expired before the user handled it")
			}
			tt.ClickAt(button.X+button.W/2, button.Y+button.H/2)
			tt.Frame()
			if recovery && !called {
				t.Error("persistent recovery action did not execute")
			}
			if !recovery && !tt.HasText(raw) {
				t.Error("persistent details action lost original output")
			}
		})
	}
}

func TestOrdinaryAndUndoNoticesKeepTheirExpiry(t *testing.T) {
	for _, undo := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "undo"}[undo], func(t *testing.T) {
			a := &app{locale: localeEnglish}
			n := &toastNotice{kind: "success", title: "Saved changes"}
			if undo {
				n.action, n.onAction, n.timeout = "Undo", func() {}, 8*time.Second
			}
			a.pushNotice(toastMain, n)
			tt := noticeLifetimeTester(a)
			tt.Advance(6 * time.Second)
			tt.Frame()
			tt.Advance(time.Second)
			tt.Frame()
			if got := tt.HasText(n.title); got != undo {
				t.Errorf("after seven seconds visible=%v; undo=%v", got, undo)
			}
			tt.Advance(3 * time.Second)
			tt.Frame()
			tt.Advance(time.Second)
			tt.Frame()
			if tt.HasText(n.title) {
				t.Error("ordinary/undo notice did not expire")
			}
		})
	}
}
