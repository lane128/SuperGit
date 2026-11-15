package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// A capsule's label can be elided in a narrow row, and the first thing a
// reader wants from its menu is the full name. Copying leads, matching the
// other ref menus; opening the ref's history follows.
func TestRefCapsuleMenuPutsCopyFirst(t *testing.T) {
	a := &app{locale: localeChinese, selectedSHA: "bbbb"}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(800).Height(300).Children(func() {
			a.commitRow(c, gitrepo.Commit{
				SHA: "bbbb", Subject: "Add the thing", Refs: "main, origin/main",
			}, 0, 0, 1, &commitGraph{})
		})
	}, 800, 300)
	pill, ok := tt.Find("main")
	if !ok {
		t.Fatal("the main branch capsule is not laid out")
	}
	tt.RightClickAt(pill.X+pill.W/2, pill.Y+pill.H/2)
	copyItem, ok := tt.Find(a.t("copyName"))
	if !ok {
		t.Fatalf("the capsule menu has no %q item", a.t("copyName"))
	}
	historyItem, ok := tt.Find(a.t("viewHistory"))
	if !ok {
		t.Fatalf("the capsule menu has no %q item", a.t("viewHistory"))
	}
	if copyItem.Y >= historyItem.Y {
		t.Errorf("copy sits at y=%v, below history at y=%v; want copy first", copyItem.Y, historyItem.Y)
	}
}
