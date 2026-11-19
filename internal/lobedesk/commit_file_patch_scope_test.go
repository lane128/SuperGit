package lobedesk

import (
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

type patchScopeRequester interface {
	requestCommitFilePatchForPlatform(commitFilePatchTarget, string)
}

func requestPatchScope(t *testing.T, a *app, target commitFilePatchTarget, platform string) {
	t.Helper()
	request, ok := any(a).(patchScopeRequester)
	if !ok {
		t.Fatal("file patch scope summary is unavailable")
	}
	request.requestCommitFilePatchForPlatform(target, platform)
}
func patchScopeTester(a *app) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) { a.applyTheme(c); ui.Column(c).Fill(); a.confirmDialog(c) }, 700, 500)
}
func clickPatchScopeButton(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	item, ok := tt.AccessByRole(ui.RoleButton, label)
	if !ok {
		t.Fatalf("missing scope button %q", label)
	}
	tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
	tt.Frame()
}
func TestCommitFilePatchPlatformScopeCancelAndContinue(t *testing.T) {
	for _, platform := range []string{"windows", "linux", "darwin"} {
		t.Run(platform, func(t *testing.T) {
			a, _, _ := commitPatchFixture(t)
			calls := 0
			a.saveFileDialog = func(mygo.SaveDialogOptions) (string, error) { calls++; return "", nil }
			target := a.commitFilePatchTarget(a.commitFiles[0].Path)
			requestPatchScope(t, a, target, platform)
			if platform == "darwin" {
				if calls != 1 || a.confirm.open {
					t.Fatal("macOS acquired an extra scope step")
				}
				return
			}
			if calls != 0 || !a.confirm.open {
				t.Fatal("native chooser opened before scope review")
			}
			tt := patchScopeTester(a)
			clickPatchScopeButton(t, tt, a.t("cancel"))
			if calls != 0 || a.confirm.open {
				t.Fatal("scope cancellation opened chooser or kept modal")
			}
			requestPatchScope(t, a, target, platform)
			tt.Frame()
			clickPatchScopeButton(t, tt, "Choose Save Location…")
			if calls != 1 || a.confirm.open {
				t.Fatal("scope continuation did not open exactly one chooser")
			}
		})
	}
}
func TestCommitFilePatchScopeRejectsChangedTarget(t *testing.T) {
	for _, change := range []string{"root", "generation", "commit"} {
		t.Run(change, func(t *testing.T) {
			a, _, _ := commitPatchFixture(t)
			calls := 0
			a.saveFileDialog = func(mygo.SaveDialogOptions) (string, error) { calls++; return "", nil }
			requestPatchScope(t, a, a.commitFilePatchTarget(a.commitFiles[0].Path), "windows")
			tt := patchScopeTester(a)
			switch change {
			case "root":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			case "generation":
				a.loadSeq++
			case "commit":
				a.selectedSHA = strings.Repeat("a", 40)
			}
			tt.Frame()
			if _, ok := tt.AccessByRole(ui.RoleButton, "Choose Save Location…"); ok {
				clickPatchScopeButton(t, tt, "Choose Save Location…")
			}
			if calls != 0 || a.busy {
				t.Fatal("stale scope request opened chooser or started writing")
			}
		})
	}
}
func TestCommitFilePatchScopeKeepsExistingConfirmation(t *testing.T) {
	a, _, _ := commitPatchFixture(t)
	target := a.commitFilePatchTarget(a.commitFiles[0].Path)
	a.askConfirm("Existing operation", "Existing scope", "Continue", false, func() {})
	requestPatchScope(t, a, target, "windows")
	if !a.confirm.open || a.confirm.patchScope || a.confirm.title != "Existing operation" || a.confirm.message != "Existing scope" {
		t.Fatal("patch scope changed an existing confirmation")
	}
}
func TestCommitFilePatchScopePreservesCompleteLongRename(t *testing.T) {
	a, _, _ := commitPatchFixture(t)
	path := strings.Repeat("长目录 folder/", 80) + "destination.go"
	old := strings.Repeat("old folder/", 80) + "source.go"
	a.commitFiles = []gitrepo.ChangedFile{{Path: path, OriginalPath: old, Status: "R100"}}
	parent := strings.Repeat("b", 40)
	a.commits[0].Parents = []string{parent}
	a.saveFileDialog = func(mygo.SaveDialogOptions) (string, error) {
		t.Fatal("scope review opened native chooser")
		return "", nil
	}
	requestPatchScope(t, a, a.commitFilePatchTarget(path), "linux")
	tt := patchScopeTester(a)
	if !strings.Contains(a.confirm.message, path) || !strings.Contains(a.confirm.message, old) || !strings.Contains(a.confirm.message, shortSHA(parent)) {
		t.Fatal("scope lost complete rename or first-parent baseline")
	}
	found := false
	for _, item := range tt.AccessAll() {
		if item.Name == a.confirm.message {
			found = true
		}
	}
	if !found {
		t.Fatal("scope text inaccessible")
	}
	before, ok := tt.AccessByRole(ui.RoleButton, "Choose Save Location…")
	if !ok {
		t.Fatal("missing fixed footer")
	}
	tt.Scroll(350, 240, 0, 100000)
	tt.Frame()
	if a.confirm.patchScroll.MaxY <= 0 || a.confirm.patchScroll.Y < a.confirm.patchScroll.MaxY-1 {
		t.Fatalf("long scope cannot scroll to its end: %+v", a.confirm.patchScroll)
	}
	after, ok := tt.AccessByRole(ui.RoleButton, "Choose Save Location…")
	if !ok || before.Bounds != after.Bounds {
		t.Fatal("scope footer moved with long text")
	}
}
