package lobedesk

import (
	"os"
	"path/filepath"
	"testing"
)

// Saving the reset defaults used to queue its warning and then a success
// toast, and the notice slot kept only the last one: the reader was told the
// reset worked while the next launch would forget it.
func TestResetSettingsReportsAFailedWrite(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	if err := os.WriteFile(home, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	a := &app{locale: localeChinese}
	a.resetSettings()

	if len(a.notices[toastMain]) != 1 {
		t.Fatalf("the failed reset queued %d notices; want the write warning", len(a.notices[toastMain]))
	}
	n := a.notices[toastMain][0]
	if n.kind != "warning" {
		t.Errorf("the failed reset reported %q %q; want the write warning", n.kind, n.title)
	}
	if n.title != a.t("settingsSaveFailed") {
		t.Errorf("the failed reset says %q; want %q", n.title, a.t("settingsSaveFailed"))
	}
}

// A direct reset that does write reports success to its default main owner.
// The settings confirmation explicitly supplies the settings window owner.
func TestResetSettingsReportsTheSuccessfulWrite(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	a := &app{locale: localeChinese}
	if !a.saveSettings() {
		t.Fatal("saving into a writable home failed")
	}
	a.resetSettings()

	if len(a.notices[toastMain]) != 1 {
		t.Fatalf("the reset queued %d notices; want one success", len(a.notices[toastMain]))
	}
	n := a.notices[toastMain][0]
	if n.kind != "success" || n.title != a.t("resetSettingsDone") {
		t.Errorf("the reset reported %q %q; want the success %q", n.kind, n.title, a.t("resetSettingsDone"))
	}
	if len(a.notices[toastSettings]) != 0 {
		t.Errorf("the reset queued a notice for a settings window that is not open")
	}
}
