package lobedesk

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestErrorNoticeDoesNotBlockBranchFooterCopy(t *testing.T) {
	defer setUIFontScale(uiFontScaleValue())
	for _, width := range []float32{980, 1320} {
		for _, scale := range []float32{fontScaleStandard, fontScaleLarge} {
			t.Run(fmt.Sprintf("width%.0f-scale%.2f", width, scale), func(t *testing.T) {
				setUIFontScale(scale)
				a := branchDetailFixture()
				a.locale = localeEnglish
				a.branchSel = "local:main"
				a.queue("error", "Operation failed", strings.Repeat("Git output\n", 100))
				copied := ""
				previous := writeClipboard
				writeClipboard = func(text string) error { copied = text; return nil }
				defer func() { writeClipboard = previous }()
				tt := ui.NewTester(func(c *ui.Context) {
					a.applyTheme(c)
					a.mainToastBottomInset = 0
					ui.Column(c).Fill().Children(func() { a.branchesPage(c) })
					a.toastHost(c, toastMain)
					a.flushNotice(c, toastMain)
				}, int(width), 640)
				for i := 0; i < 6; i++ {
					tt.Advance(100 * time.Millisecond)
					tt.Frame()
				}
				button, ok := tt.Find(a.t("copyName"))
				if !ok {
					t.Fatal("fixture did not show branch footer copy control")
				}
				tt.ClickAt(button.X+button.W/2, button.Y+button.H/2)
				if copied != "main" {
					t.Fatalf("error toast blocked footer Copy name at %+v", button)
				}
			})
		}
	}
}
