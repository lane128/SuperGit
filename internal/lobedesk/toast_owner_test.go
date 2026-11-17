package lobedesk

import (
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

func TestShownRepositoryNoticeExitsAfterRootSwitch(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	a.queue("error", "old repository error", "old Git details")
	a.queueFor(toastSettings, "warning", "settings save failed", "")
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.toastHost(c, toastMain)
		a.flushNotice(c, toastMain)
	}, 600, 300)
	for i := 0; i < 5; i++ {
		tt.Advance(50 * time.Millisecond)
		tt.Frame()
	}
	if !tt.HasText("old repository error") {
		t.Fatal("fixture did not display repository notice")
	}
	// Refresh creates a new Repository pointer at the same root. It must not
	// hide the error a reader is still using.
	a.repo = &gitrepo.Repository{Root: "first"}
	tt.Frame()
	if !tt.HasText("old repository error") {
		t.Fatal("same-root refresh hid the active error")
	}
	a.repo = &gitrepo.Repository{Root: "second"}
	for i := 0; i < 5; i++ {
		tt.Advance(50 * time.Millisecond)
		tt.Frame()
	}
	if tt.HasText("old repository error") || tt.HasText("old Git details") {
		t.Fatal("already-flushed repository notice survived root switch")
	}
	if len(a.notices[toastSettings]) != 1 {
		t.Fatal("root switch consumed another window's notice")
	}
}

func TestQueuedRepositoryNoticeCannotMoveToAnotherRoot(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	a.queue("error", "queued old error", "")
	a.repo = &gitrepo.Repository{Root: "second"}
	a.queue("success", "current result", "")
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.toastHost(c, toastMain)
		a.flushNotice(c, toastMain)
	}, 600, 300)
	for i := 0; i < 5; i++ {
		tt.Advance(50 * time.Millisecond)
		tt.Frame()
	}
	if tt.HasText("queued old error") || !tt.HasText("current result") {
		t.Fatal("queued result was displayed in the wrong repository")
	}
}

func TestShownRepositoryNoticeCannotRunActionAfterRootSwitch(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	acted := false
	a.pushNotice(toastMain, &toastNotice{kind: "success", title: "old action", action: "Undo old action", onAction: func() { acted = true }})
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.toastHost(c, toastMain)
		a.flushNotice(c, toastMain)
	}, 600, 300)
	for i := 0; i < 5; i++ {
		tt.Advance(50 * time.Millisecond)
		tt.Frame()
	}
	button, ok := tt.Find("Undo old action")
	if !ok {
		t.Fatal("fixture did not display action")
	}
	// The input can arrive against the last frame before the next render
	// removes the old toast. Its callback must guard its own repository.
	a.repo = &gitrepo.Repository{Root: "second"}
	tt.ClickAt(button.X+button.W/2, button.Y+button.H/2)
	if acted {
		t.Fatal("old toast action ran against a new repository")
	}
}

// A notice belongs to the window that shows it. Drawing the other window does
// not take it out of its queue, so the main window no longer swallows a
// settings result (or the reverse) before its window can draw it.
func TestANoticeWaitsForItsOwnWindow(t *testing.T) {
	a := &app{locale: localeChinese}
	a.queueFor(toastSettings, "warning", "could not save", "")
	if got := len(a.notices[toastSettings]); got != 1 {
		t.Fatalf("the settings queue holds %d notices; want 1", got)
	}

	ui.NewTester(func(c *ui.Context) { a.flushNotice(c, toastMain) }, 300, 200).Frame()
	if got := len(a.notices[toastSettings]); got != 1 {
		t.Fatalf("the main window took a settings notice; the queue holds %d", got)
	}

	ui.NewTester(func(c *ui.Context) { a.flushNotice(c, toastSettings) }, 300, 200).Frame()
	if got := len(a.notices[toastSettings]); got != 0 {
		t.Fatalf("the settings window left its own notice queued (%d)", got)
	}
}

// Repository results belong to the main window even while Settings has the
// keyboard; the panel must not show a repository error over its page.
func TestRepositoryNoticeBelongsToTheMainWindow(t *testing.T) {
	a := &app{locale: localeChinese}
	a.queue("error", "push failed", "denied")
	if got := len(a.notices[toastMain]); got != 1 {
		t.Fatalf("the main queue holds %d notices; want 1", got)
	}
	if got := len(a.notices[toastSettings]); got != 0 {
		t.Fatalf("the settings queue holds %d notices; want none", got)
	}
}

// One window's pending notice is not replaced by the other window's: a fetch
// failure waiting for the main window and a save failure for Settings both
// survive.
func TestTheWindowsKeepTheirOwnPendingNotices(t *testing.T) {
	a := &app{locale: localeChinese}
	a.queue("error", "fetch failed", "timed out")
	a.queueFor(toastSettings, "warning", "could not save", "")

	ui.NewTester(func(c *ui.Context) { a.flushNotice(c, toastMain) }, 300, 200).Frame()
	if got := len(a.notices[toastMain]); got != 0 {
		t.Fatalf("the main queue holds %d notices after its window drew; want none", got)
	}
	if got := len(a.notices[toastSettings]); got != 1 {
		t.Fatalf("the settings queue holds %d notices; want the save warning still waiting", got)
	}
	if a.notices[toastSettings][0].title != "could not save" {
		t.Errorf("the waiting notice is %q; want the save warning", a.notices[toastSettings][0].title)
	}
}

// A settings notice whose panel closed before it drew is not lost: the main
// window adopts it, in front of nothing the main window already queued.
func TestOrphanedSettingsNoticeMovesToTheMainWindow(t *testing.T) {
	a := &app{locale: localeChinese}
	a.queueFor(toastSettings, "warning", "could not save", "")
	a.adoptOrphanedNotice()
	if got := len(a.notices[toastSettings]); got != 0 {
		t.Fatalf("the settings queue holds %d notices; want none", got)
	}
	if len(a.notices[toastMain]) != 1 || a.notices[toastMain][0].title != "could not save" {
		t.Fatalf("the main queue reads %+v; want the adopted notice", a.notices[toastMain])
	}
}

// While the panel is open, its notice stays with it: the main window must not
// adopt a settings notice out from under a panel that will draw it.
func TestAnOpenPanelKeepsItsNotice(t *testing.T) {
	a := &app{locale: localeChinese}
	a.settingsWin.Store(&mygo.Window{})
	a.queueFor(toastSettings, "warning", "could not save", "")
	a.adoptOrphanedNotice()
	if got := len(a.notices[toastSettings]); got != 1 {
		t.Fatalf("the settings queue holds %d notices; want it kept for the open panel", got)
	}
	if got := len(a.notices[toastMain]); got != 0 {
		t.Fatalf("the main queue holds %d notices; want none", got)
	}
}

// Flushing does more than take the notice from its queue: the window's toast
// host shows the title and the action the notice carries.
func TestFlushedNoticeIsShown(t *testing.T) {
	a := &app{locale: localeChinese}
	a.queue("error", "push failed", "denied")
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.toastHost(c, toastMain)
		a.flushNotice(c, toastMain)
	}, 600, 300)
	for i := 0; i < 5; i++ {
		tt.Advance(50 * time.Millisecond)
		tt.Frame()
	}
	if _, ok := tt.Find("push failed"); !ok {
		t.Fatal("the flushed notice is queued but nothing shows it")
	}
}

// Notices show in the order they were queued. A second result does not replace
// the first before the reader has seen it, and the one still waiting is not
// dropped along with it.
func TestNoticesShowInTheOrderTheyArrived(t *testing.T) {
	a := &app{locale: localeChinese}
	a.queue("success", "first result", "")
	a.queue("error", "second result", "denied")
	if got := len(a.notices[toastMain]); got != 2 {
		t.Fatalf("the queue holds %d notices; want both", got)
	}
	// One flush, as the window does on the frame the result arrives; a flush
	// every frame would show the second notice before this test can look.
	flushed := false
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.toastHost(c, toastMain)
		if !flushed {
			flushed = true
			a.flushNotice(c, toastMain)
		}
	}, 600, 300)
	for i := 0; i < 5; i++ {
		tt.Advance(50 * time.Millisecond)
		tt.Frame()
	}
	if _, ok := tt.Find("first result"); !ok {
		t.Fatal("the first notice is not shown")
	}
	if _, ok := tt.Find("second result"); ok {
		t.Error("the second notice jumped ahead of the first")
	}
	if len(a.notices[toastMain]) != 1 || a.notices[toastMain][0].title != "second result" {
		t.Fatalf("the queue reads %+v; want the second notice still waiting", a.notices[toastMain])
	}
}

// A notice carries more than a title: the detail says what failed and the
// action is how the reader repairs it, so both have to reach the toast.
func TestFlushedNoticeKeepsItsDetailAndAction(t *testing.T) {
	a := &app{locale: localeChinese}
	acted := false
	a.pushNotice(toastMain, &toastNotice{
		kind: "error", title: "push failed", description: "the remote refused",
		action: "Undo", onAction: func() { acted = true },
	})
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.toastHost(c, toastMain)
		a.flushNotice(c, toastMain)
	}, 600, 300)
	for i := 0; i < 5; i++ {
		tt.Advance(50 * time.Millisecond)
		tt.Frame()
	}
	if _, ok := tt.Find("the remote refused"); !ok {
		t.Error("the toast drops the notice's description")
	}
	action, ok := tt.Find("Undo")
	if !ok {
		t.Fatal("the toast drops the notice's action")
	}
	tt.ClickAt(action.X+action.W/2, action.Y+action.H/2)
	if !acted {
		t.Error("clicking the action does not run the notice's callback")
	}
}
