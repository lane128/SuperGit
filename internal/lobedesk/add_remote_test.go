package lobedesk

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestAddRemoteFromSidebar(t *testing.T) {
	a, done, root := worktreeApp(t)
	a.locale = localeEnglish
	before, err := os.ReadFile(filepath.Join(root, ".git", "config"))
	if err != nil {
		t.Fatal(err)
	}
	other, _, otherRoot := worktreeApp(t)
	address := filepath.Join(t.TempDir(), "local target.git")
	if out, err := exec.Command("git", "clone", "--bare", "--", otherRoot, address).CombinedOutput(); err != nil {
		t.Fatalf("create the local bare remote: %v\n%s", err, out)
	}
	wait := func() {
		for a.busy || a.loading {
			runContinuation(t, done)
		}
	}
	tt := ui.NewTester(a.root, 980, 640)
	tt.Frame()
	clickRepositoryTabsControl(t, tt, ui.RoleDisclosure, "Remotes")
	open := func() {
		clickRepositoryTabsControl(t, tt, ui.RoleButton, "Add Remote…")
		if !a.hasPendingDialog(toastMain) {
			t.Fatal("the main window did not claim the remote form")
		}
	}
	fill := func(label, value string) {
		clickRepositoryTabsControl(t, tt, ui.RoleTextField, label)
		tt.Type(value)
		tt.Frame()
	}
	open()
	fill("Remote name", "backup")
	fill("Git address", address)
	clickRepositoryTabsControl(t, tt, ui.RoleButton, "Cancel")
	if data, err := os.ReadFile(filepath.Join(root, ".git", "config")); err != nil || string(data) != string(before) {
		t.Fatalf("cancel changed repository configuration: %v", err)
	}
	open()
	fill("Remote name", "backup")
	fill("Git address", address)
	clickRepositoryTabsControl(t, tt, ui.RoleButton, "Add Remote")
	if !a.busy {
		t.Fatal("adding a remote did not start a Git operation")
	}
	wait()
	if len(a.remotes) != 1 || a.remotes[0].Name != "backup" || a.remotes[0].URL != address {
		t.Fatalf("the refreshed checkout does not show the new remote: %+v", a.remotes)
	}
	config, err := os.ReadFile(filepath.Join(root, ".git", "config"))
	if err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	heading, ok := tt.AccessByRole(ui.RoleDisclosure, "Remotes")
	if !ok {
		t.Fatal("missing Remotes section")
	}
	tt.RightClickAt(heading.Bounds.X+heading.Bounds.W/2, heading.Bounds.Y+heading.Bounds.H/2)
	clickRepositoryTabsControl(t, tt, ui.RoleMenuItem, "Add Remote…")
	fill("Remote name", "backup")
	alternative := filepath.Join(t.TempDir(), "different.git")
	fill("Git address", alternative)
	clickRepositoryTabsControl(t, tt, ui.RoleButton, "Add Remote")
	if !a.busy {
		t.Fatal("submitting the duplicate did not start a Git operation")
	}
	wait()
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleTextField, "Git address"); !ok || a.addRemoteForm.address != alternative {
		t.Fatal("a duplicate remote must keep the form and its address available for a retry")
	}
	dialog := dialogTester(a, a.addRemoteDialog)
	if !dialog.HasText("error: remote backup already exists.") {
		t.Fatal("the form must explain the Git rejection while both fields are still editable")
	}
	if data, err := os.ReadFile(filepath.Join(root, ".git", "config")); err != nil || string(data) != string(config) {
		t.Fatalf("a duplicate remote changed repository configuration: %v", err)
	}
	if len(a.remoteBranches) != 0 {
		t.Fatal("adding a remote fetched tracking references automatically")
	}
	clickRepositoryTabsControl(t, tt, ui.RoleTextField, "Remote name")
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("archive")
	clickRepositoryTabsControl(t, tt, ui.RoleTextField, "Git address")
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type(address)
	tt.Frame()
	clickRepositoryTabsControl(t, tt, ui.RoleButton, "Add Remote")
	wait()
	if len(a.remotes) != 2 || a.remotes[0].Name != "archive" || a.remotes[1].Name != "backup" {
		t.Fatalf("retry should add the corrected remote without replacing the original: %+v", a.remotes)
	}
	tt.Frame()
	clickRepositoryTabsControl(t, tt, ui.RoleButton, "Fetch")
	if !a.busy {
		t.Fatal("manual Fetch did not start after adding a remote")
	}
	wait()
	if len(a.remoteBranches) != 2 || a.remoteBranches[0].Name != "archive/main" || a.remoteBranches[1].Name != "backup/main" {
		t.Fatalf("manual Fetch did not show the remote branch: %+v", a.remoteBranches)
	}
	if remotes, err := other.repo.Remotes(context.Background()); err != nil || len(remotes) != 0 {
		t.Fatalf("another checkout was affected (%q): %v, %v", otherRoot, remotes, err)
	}
}
