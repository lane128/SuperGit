package lobedesk

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestActiveDetailTabAndCloseStayVisibleAfterNarrowing(t *testing.T) {
	a := &app{page: "history", worktreeSelected: true, locale: localeEnglish}
	for i := 0; i < 8; i++ {
		a.detailTabs().Open(detailFileKey{Path: fmt.Sprintf("long-file-name-%02d.go", i)})
	}
	a.detailReveal = true
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.historyDiffPane(c) }, 1600, 600)
	tt.Frame()
	tt.Frame()
	tt.Advance(time.Second)
	active := a.detailTabs().Active
	label := active.File.Path + " · " + a.t("unstaged")
	before, ok := tt.AccessByRole(ui.RoleTab, label)
	if !ok || before.Bounds.X < 0 || before.Bounds.X+before.Bounds.W > 1600 {
		t.Fatal("fixture active tab is not initially visible")
	}
	a.detailReveal = false
	tt.SetSize(320, 600)
	tt.Frame()
	tt.Frame()
	tt.Frame()
	tt.Advance(time.Second)
	for _, item := range []struct {
		Role  ui.Role
		Label string
	}{{ui.RoleTab, label}, {ui.RoleButton, a.t("close") + " " + active.File.Path}} {
		node, ok := tt.AccessByRole(item.Role, item.Label)
		if !ok || node.Bounds.X < 0 || node.Bounds.X+node.Bounds.W > 320 {
			t.Fatalf("%s outside resized pane: %+v", item.Label, node.Bounds)
		}
	}
	if a.detailTabs().Active != active || len(a.detailTabs().OpenFiles) != 8 {
		t.Fatal("resize changed tab identities")
	}
}
