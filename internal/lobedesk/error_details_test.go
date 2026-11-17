package lobedesk

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestUnbrokenErrorDetailsFitLargeFontAtMinimumWindow(t *testing.T) {
	defer setUIFontScale(uiFontScaleValue())
	setUIFontScale(fontScaleLarge)
	a := &app{locale: localeEnglish}
	raw := strings.Repeat("Unbroken文件", 512)
	a.noticeDetails[toastMain] = noticeDetailsState{open: true, notice: &toastNotice{title: "Failure", description: raw}}
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.noticeDetailsDialog(c, toastMain) }, 980, 640)
	text, ok := tt.Find(raw)
	if !ok || text.W > 800 || text.X < 0 {
		t.Fatalf("long token is not constrained to detail viewport: %+v", text)
	}
	for _, label := range []string{"Copy details", "Close"} {
		button, ok := tt.Find(label)
		if !ok || button.X < 0 || button.Y < 0 || button.X+button.W > 980 || button.Y+button.H > 640 {
			t.Fatalf("large font clipped %s", label)
		}
	}
	copied := ""
	previous := writeClipboard
	writeClipboard = func(text string) error { copied = text; return nil }
	defer func() { writeClipboard = previous }()
	button, _ := tt.Find("Copy details")
	tt.ClickAt(button.X+button.W/2, button.Y+button.H/2)
	if copied != raw {
		t.Fatal("copy lost unbroken output")
	}
}

func TestErrorDetailsActionsUseSharedButtonMetrics(t *testing.T) {
	before := uiFontScaleValue()
	setUIFontScale(fontScaleStandard)
	t.Cleanup(func() { setUIFontScale(before) })
	a := &app{locale: localeEnglish}
	a.noticeDetails[toastMain] = noticeDetailsState{open: true, notice: &toastNotice{title: "Failure", description: "Original output"}}
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.noticeDetailsDialog(c, toastMain) }, 560, 480)
	for _, label := range []string{"Copy details", "Close"} {
		button, ok := tt.AccessByRole(ui.RoleButton, label)
		if !ok || button.Bounds.H != 29 {
			t.Errorf("%s height=%v; want shared 29 pt button", label, button.Bounds.H)
		}
	}
}

func TestClosingSettingsDiscardsItsErrorDetails(t *testing.T) {
	a := &app{}
	a.settingsWin.Store(&mygo.Window{})
	a.noticeDetails[toastSettings] = noticeDetailsState{open: true, notice: &toastNotice{title: "Old settings error", description: "old output"}}
	a.settingsWin.Store(nil)
	ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.noticeDetailsDialog(c, toastMain) }, 980, 640)
	if a.noticeDetails[toastSettings].open || a.noticeDetails[toastSettings].notice != nil {
		t.Fatal("closed settings window retained old error details")
	}
	a.settingsWin.Store(&mygo.Window{})
	reopened := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.noticeDetailsDialog(c, toastSettings) }, 560, 480)
	if reopened.HasText("old output") {
		t.Fatal("reopened window displays old details")
	}
}

func TestLongErrorTitleKeepsOriginalInDetails(t *testing.T) {
	a := &app{locale: localeEnglish}
	raw := strings.Repeat("复制失败 Error原文", 100)
	a.queue("error", raw, "")
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.toastHost(c, toastMain)
		a.flushNotice(c, toastMain)
		a.noticeDetailsDialog(c, toastMain)
	}, 980, 640)
	for i := 0; i < 5; i++ {
		tt.Advance(50 * time.Millisecond)
		tt.Frame()
	}
	if tt.HasText(raw) {
		t.Fatal("long title still expands the toast")
	}
	button, ok := tt.Find("View details")
	if !ok {
		t.Fatal("long title has no details action")
	}
	tt.ClickAt(button.X+button.W/2, button.Y+button.H/2)
	tt.Frame()
	if !tt.HasText(raw) {
		t.Fatal("full original title lost from details")
	}
	copied := ""
	previous := writeClipboard
	writeClipboard = func(text string) error { copied = text; return nil }
	defer func() { writeClipboard = previous }()
	copyButton, ok := tt.Find("Copy details")
	if !ok {
		t.Fatal("long-title details cannot be copied")
	}
	tt.ClickAt(copyButton.X+copyButton.W/2, copyButton.Y+copyButton.H/2)
	if copied != raw {
		t.Fatal("copy lost original long title")
	}
}

func TestErrorDetailsRemainInTheirWindowAndCloseWithEscape(t *testing.T) {
	a := &app{locale: localeEnglish}
	a.settingsWin.Store(&mygo.Window{})
	raw := strings.Repeat("details 文件\n", 100)
	a.noticeDetails[toastSettings] = noticeDetailsState{open: true, notice: &toastNotice{title: "Settings failure", description: raw}}
	main := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.noticeDetailsDialog(c, toastMain) }, 980, 640)
	if main.HasText(raw) {
		t.Fatal("settings details moved into main window")
	}
	settings := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.noticeDetailsDialog(c, toastSettings) }, 560, 480)
	title, ok := settings.Find("Settings failure")
	if !ok || title.X < 16 || title.X+title.W > 544 {
		t.Fatal("detail panel overflows settings minimum width")
	}
	button, ok := settings.Find("Copy details")
	if !ok || button.Y < 0 || button.Y+button.H > 480 || button.X < 0 || button.X+button.W > 560 {
		t.Fatal("detail controls are outside settings minimum window")
	}
	settings.Key(0, ui.KeyEscape)
	settings.Frame()
	if settings.HasText(raw) {
		t.Fatal("Escape did not dismiss detail view")
	}
}

func TestLongErrorNoticeStaysCompactWithDetailsAction(t *testing.T) {
	a := &app{locale: localeEnglish, repo: &gitrepo.Repository{Root: "first"}}
	raw := strings.Repeat("Git output 文件\n", 100) + "last original line"
	a.queue("error", "Operation failed", raw)
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.toastHost(c, toastMain)
		a.flushNotice(c, toastMain)
		a.noticeDetailsDialog(c, toastMain)
	}, 980, 640)
	for i := 0; i < 5; i++ {
		tt.Advance(50 * time.Millisecond)
		tt.Frame()
	}
	if tt.HasText(raw) {
		t.Fatal("long original output still fills the toast")
	}
	button, ok := tt.Find("View details")
	if !ok {
		t.Fatal("compact error has no way to recover full details")
	}
	tt.ClickAt(button.X+button.W/2, button.Y+button.H/2)
	tt.Frame()
	if !tt.HasText(raw) {
		t.Fatal("detail view lost original output")
	}
	copied := ""
	previous := writeClipboard
	writeClipboard = func(text string) error { copied = text; return nil }
	defer func() { writeClipboard = previous }()
	copyButton, ok := tt.Find("Copy details")
	if !ok {
		t.Fatal("details have no copy action")
	}
	tt.ClickAt(copyButton.X+copyButton.W/2, copyButton.Y+copyButton.H/2)
	if copied != raw {
		t.Fatal("copy did not preserve full original output")
	}
	a.repo = &gitrepo.Repository{Root: "second"}
	tt.Frame()
	if tt.HasText(raw) {
		t.Fatal("old error details survived repository switch")
	}
}
