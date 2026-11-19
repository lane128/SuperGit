package lobedesk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestNewRepositoryAsyncFailureRetainsKeyboardCorrection(t *testing.T) {
	a, tt, done, old := newRepoKeyboardTester(t, true)
	failedPath := filepath.Join(t.TempDir(), "失败 init")
	openNewRepoKeyboardEntrance(t, a, tt, true)
	tt.Type(failedPath)
	if a.newRepoForm.path != failedPath {
		t.Fatal("initial typing missed destination")
	}
	if !t.Run("owned Git failure", func(t *testing.T) {
		t.Setenv("GIT_DEFAULT_HASH", "invalid-format-for-fixture")
		tt.Key(0, ui.KeyEnter)
		tt.Frame()
		if !a.busy {
			t.Fatal("Enter did not start Git init")
		}
		input, ok := tt.AccessByRole(ui.RoleTextField, a.t("cloneDestination"))
		if !ok || !input.Disabled {
			t.Fatal("busy form did not render disabled input")
		}
		runContinuation(t, done)
		if a.busy || a.loading || !a.newRepoForm.open || a.newRepoForm.path != failedPath || a.newRepoForm.err == "" || a.repo.Root != old || a.commitText != "retained draft" {
			t.Fatal("asynchronous failure lost original context")
		}
	}) {
		return
	}
	tt.Frame()
	input, ok := tt.AccessByRole(ui.RoleTextField, a.t("cloneDestination"))
	if !ok || input.Disabled {
		t.Fatal("failed operation did not re-enable input")
	}
	if _, err := os.Lstat(failedPath); err == nil {
		if !strings.Contains(a.newRepoForm.err, failedPath) {
			t.Fatal("residual destination was not explained")
		}
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	newPath := filepath.Join(t.TempDir(), "重试 repo")
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type(newPath)
	if a.newRepoForm.path != newPath {
		t.Fatal("async failure lost focus or selection replacement")
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if !a.busy || a.repo.Root != old || a.commitText != "retained draft" {
		t.Fatal("corrected keyboard retry failed or switched early")
	}
	runContinuation(t, done)
	for a.loading {
		runContinuation(t, done)
	}
	canonical, err := filepath.EvalSymlinks(newPath)
	if err != nil {
		t.Fatal(err)
	}
	if a.busy || a.loading || a.repo == nil || a.repo.Root != canonical || a.newRepoForm.open || a.commitText != "" || a.loadErr != "" || a.status.Head != "" || len(a.commits) != 0 {
		t.Fatal("keyboard retry did not open new unborn repository")
	}
}
