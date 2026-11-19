package lobedesk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestCommitFileUndoUIPreflightExplainsUnsupportedConversion(t *testing.T) {
	a, callbacks, tt := commitUndoUIFixture(t)
	root := a.repo.Root
	selectionGit(t, root, "config", "core.autocrlf", "true")
	item := openUndoUIMenu(t, a, tt)
	tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
	waitSelectionOperation(t, a, callbacks)
	tt.Frame()
	if a.confirm.open {
		t.Fatal("unsupported target acquired an actionable confirmation")
	}
	notices := a.notices[toastMain]
	if len(notices) != 1 || notices[0].kind != "error" || notices[0].description != a.t("undoFileUnsupported") {
		t.Fatal("unsupported conversion lacks localized guidance")
	}
	got, err := os.ReadFile(filepath.Join(root, "same.go"))
	if err != nil || string(got) != "source change\n" || selectionGit(t, root, "status", "--porcelain=v1", "-z") != "" {
		t.Fatal("preflight failure changed source")
	}
}
func TestCommitFileUndoUIEscapeCancelsWithoutWrite(t *testing.T) {
	a, callbacks, tt := commitUndoUIFixture(t)
	requestUndoUI(t, a, callbacks, tt)
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.confirm.open || a.busy || selectionGit(t, a.repo.Root, "status", "--porcelain=v1", "-z") != "" {
		t.Fatal("Escape did not cancel without writing")
	}
}
func TestCommitFileUndoUILongScopeKeepsFooterReachable(t *testing.T) {
	a, callbacks, initial := commitUndoUIFixture(t)
	requestUndoUI(t, a, callbacks, initial)
	longPath := strings.Repeat("长目录 folder/", 100) + "same.go"
	a.confirm.message = a.tf("undoFileMessage", longPath, a.selectedSHA, a.commits[0].Parents[0], "main", a.status.Head)
	beforeScale := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(beforeScale) })
	setUIFontScale(fontScaleLarge)
	tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); ui.Column(c).Fill(); a.confirmDialog(c) }, 480, 360)
	before, ok := tt.AccessByRole(ui.RoleButton, a.t("cancel"))
	if !ok || before.Bounds.Y+before.Bounds.H > 360 {
		t.Fatal("long scope moved Cancel off screen")
	}
	found := false
	for _, item := range tt.AccessAll() {
		if item.Name == a.confirm.message {
			found = true
		}
	}
	if !found {
		t.Fatal("long scope is inaccessible")
	}
	if a.confirm.patchScroll.MaxY <= 0 {
		t.Fatal("long scope did not offer scrolling")
	}
	tt.Scroll(240, 150, 0, a.confirm.patchScroll.MaxY+10000)
	tt.Frame()
	if a.confirm.patchScroll.Y != a.confirm.patchScroll.MaxY {
		t.Fatal("long scope end is unreachable")
	}
	after, ok := tt.AccessByRole(ui.RoleButton, a.t("cancel"))
	if !ok || after.Bounds != before.Bounds {
		t.Fatal("footer moved while scrolling")
	}
	clickPatchScopeButton(t, tt, a.t("cancel"))
	if a.confirm.open || a.busy {
		t.Fatal("long scope cancellation did not close")
	}
}
