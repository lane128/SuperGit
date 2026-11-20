package lobedesk

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func commitPatchFixture(t *testing.T) (*app, chan func(), *ui.Tester) {
	t.Helper()
	a, callbacks := overviewFixture(t)
	a.page = "history"
	a.locale = localeEnglish
	a.worktreeSelected = false
	a.selectedSHA = strings.TrimSpace(selectionGit(t, a.repo.Root, "rev-parse", "HEAD"))
	var err error
	a.commitFiles, err = a.repo.CommitFiles(context.Background(), a.selectedSHA)
	if err != nil {
		t.Fatal(err)
	}
	a.commits = []gitrepo.Commit{{SHA: a.selectedSHA}}
	a.commitFilesLoaded = true
	a.commitText = "keep draft title\n\nkeep draft body"
	tt := pageTester(a, func(c *ui.Context) {
		ui.Column(c).Width(500).Height(250).Children(func() { a.commitFileList(c) })
		a.confirmDialog(c)
	})
	return a, callbacks, tt
}
func openPatchMenu(t *testing.T, a *app, tt *ui.Tester) ui.AccessInfo {
	t.Helper()
	row, ok := tt.AccessByRole(ui.RoleOption, a.commitFiles[0].Path)
	if !ok {
		t.Fatal("missing historical file row")
	}
	tt.RightClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	item, ok := tt.AccessByRole(ui.RoleMenuItem, "Export File Patch…")
	if !ok {
		t.Fatal("historical file menu lacks patch export")
	}
	return item
}
func activatePatchExport(t *testing.T, a *app, tt *ui.Tester) {
	t.Helper()
	item := openPatchMenu(t, a, tt)
	tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
	if a.confirm.open && a.confirm.patchScope {
		tt.Frame()
		clickPatchScopeButton(t, tt, a.t("choosePatchDestination"))
	}
}
func TestCommitFilePatchMenuSavesApplicablePatchWithoutReload(t *testing.T) {
	a, callbacks, tt := commitPatchFixture(t)
	root := a.repo.Root
	destination := filepath.Join(t.TempDir(), "selected.patch")
	if err := os.WriteFile(destination, []byte("previous patch"), 0600); err != nil {
		t.Fatal(err)
	}
	head := selectionGit(t, root, "rev-parse", "HEAD")
	index := selectionGit(t, root, "show", ":same.go")
	work, _ := os.ReadFile(filepath.Join(root, "same.go"))
	calls := 0
	a.saveFileDialog = func(opts mygo.SaveDialogOptions) (string, error) {
		calls++
		if !strings.Contains(opts.Message, "same.go") || !strings.Contains(opts.Message, shortSHA(a.selectedSHA)) || !strings.Contains(opts.Message, "Empty tree") {
			t.Errorf("save dialog lacks export identity: %+v", opts)
		}
		return destination, nil
	}
	activatePatchExport(t, a, tt)
	waitSelectionOperation(t, a, callbacks)
	if calls != 1 {
		t.Fatalf("save chooser calls: %d", calls)
	}
	patch, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	selectionGit(t, target, "init", "-b", "main")
	cmd := exec.Command("git", "-C", target, "apply", "--index", "--")
	cmd.Stdin = bytes.NewReader(patch)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("exported patch cannot apply: %v %s", err, out)
	}
	got, err := os.ReadFile(filepath.Join(target, "same.go"))
	if err != nil || string(got) != "original\n" {
		t.Fatal("menu exported worktree or index instead of selected commit")
	}
	gotWork, _ := os.ReadFile(filepath.Join(root, "same.go"))
	if selectionGit(t, root, "rev-parse", "HEAD") != head || selectionGit(t, root, "show", ":same.go") != index || !bytes.Equal(gotWork, work) || a.loadSeq != 0 || a.selectedSHA != strings.TrimSpace(head) || a.commitText != "keep draft title\n\nkeep draft body" {
		t.Fatal("export changed source state, draft, or selection")
	}
	if len(a.notices[toastMain]) != 1 || a.notices[toastMain][0].kind != "success" {
		t.Fatal("missing export success feedback")
	}
}
func TestCommitFilePatchChooserCancellationErrorAndStaleReturnDoNotWrite(t *testing.T) {
	cases := []string{"cancel", "error", "root", "generation", "commit", "metadata", "busy", "loading", "dialog"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			a, _, tt := commitPatchFixture(t)
			destination := filepath.Join(t.TempDir(), "existing.patch")
			if err := os.WriteFile(destination, []byte("keep existing"), 0600); err != nil {
				t.Fatal(err)
			}
			calls := 0
			a.saveFileDialog = func(mygo.SaveDialogOptions) (string, error) {
				calls++
				switch name {
				case "cancel":
					return "", nil
				case "error":
					return "", errors.New("owned chooser failure")
				case "root":
					a.repo = &gitrepo.Repository{Root: t.TempDir()}
				case "generation":
					a.loadSeq++
				case "commit":
					a.selectedSHA = strings.Repeat("a", 40)
				case "metadata":
					a.commitFiles[0].OriginalPath = "different.txt"
				case "busy":
					a.busy = true
				case "loading":
					a.loading = true
				case "dialog":
					a.prompt.open = true
				}
				return destination, nil
			}
			activatePatchExport(t, a, tt)
			got, err := os.ReadFile(destination)
			if err != nil || string(got) != "keep existing" || calls != 1 || a.repositoryChooserOpen || a.commitText != "keep draft title\n\nkeep draft body" {
				t.Fatalf("chooser %s changed destination/draft or retained lock", name)
			}
			if name == "error" && (len(a.notices[toastMain]) != 1 || a.notices[toastMain][0].kind != "error") {
				t.Fatal("missing chooser failure feedback")
			}
		})
	}
}
func TestCommitFilePatchGenerationFailurePreservesExistingDestination(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing-object", true: "cancelled"}[cancelled], func(t *testing.T) {
			a, callbacks, tt := commitPatchFixture(t)
			dir := t.TempDir()
			destination := filepath.Join(dir, "existing.patch")
			if err := os.WriteFile(destination, []byte("keep existing"), 0600); err != nil {
				t.Fatal(err)
			}
			if cancelled {
				ctx, cancel := context.WithCancel(a.ctx)
				cancel()
				a.ctx = ctx
			} else {
				a.selectedSHA = strings.Repeat("a", 40)
				a.commits = []gitrepo.Commit{{SHA: a.selectedSHA}}
			}
			tt.Frame()
			a.saveFileDialog = func(mygo.SaveDialogOptions) (string, error) { return destination, nil }
			activatePatchExport(t, a, tt)
			waitSelectionOperation(t, a, callbacks)
			got, err := os.ReadFile(destination)
			if err != nil || string(got) != "keep existing" {
				t.Fatal("failed generation replaced destination")
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 1 {
				t.Fatal("failed generation leaked adjacent temporary file")
			}
			if len(a.notices[toastMain]) != 1 {
				t.Fatal("missing generation failure/cancellation feedback")
			}
		})
	}
}
func TestCommitFilePatchOpenMenuRejectsChangedTarget(t *testing.T) {
	for _, change := range []string{"root", "generation", "commit", "metadata"} {
		t.Run(change, func(t *testing.T) {
			a, _, tt := commitPatchFixture(t)
			calls := 0
			a.saveFileDialog = func(mygo.SaveDialogOptions) (string, error) { calls++; return "", nil }
			openPatchMenu(t, a, tt)
			switch change {
			case "root":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			case "generation":
				a.loadSeq++
			case "commit":
				a.selectedSHA = strings.Repeat("a", 40)
				a.commits = []gitrepo.Commit{{SHA: a.selectedSHA}}
			case "metadata":
				a.commitFiles[0].Status = "D"
			}
			tt.Frame()
			item, ok := tt.AccessByRole(ui.RoleMenuItem, "Export File Patch…")
			if ok {
				if !item.Disabled {
					t.Fatal("old menu target remained enabled")
				}
				tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
			}
			if calls != 0 {
				t.Fatal("stale menu opened save dialog")
			}
		})
	}
}
func TestCommitFilePatchSaveFilesystemFailuresPreserveDestination(t *testing.T) {
	for _, missingParent := range []bool{true, false} {
		t.Run(map[bool]string{true: "create", false: "replace"}[missingParent], func(t *testing.T) {
			a, callbacks, tt := commitPatchFixture(t)
			dir := t.TempDir()
			destination := filepath.Join(dir, "destination")
			if missingParent {
				destination = filepath.Join(destination, "patch")
			} else {
				if err := os.Mkdir(destination, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(destination, "keep"), []byte("existing directory contents"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			a.saveFileDialog = func(mygo.SaveDialogOptions) (string, error) { return destination, nil }
			activatePatchExport(t, a, tt)
			waitSelectionOperation(t, a, callbacks)
			entries, err := os.ReadDir(dir)
			want := 0
			if !missingParent {
				want = 1
				got, err := os.ReadFile(filepath.Join(destination, "keep"))
				if err != nil || string(got) != "existing directory contents" {
					t.Fatal("replace error changed destination contents")
				}
			}
			if err != nil || len(entries) != want {
				t.Fatal("save failure leaked temp file or created destination")
			}
			if len(a.notices[toastMain]) != 1 || a.notices[toastMain][0].kind != "error" {
				t.Fatal("save filesystem failure lacks feedback")
			}
		})
	}
}
func TestCommitFilePatchMenuOffersSingleFileExport(t *testing.T) {
	a, _, tt := commitPatchFixture(t)
	item := openPatchMenu(t, a, tt)
	if item.Disabled {
		t.Fatal("valid historical file export disabled")
	}
}
