package gitrepo

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The interface lets the first regression fail on the absent capability while
// still exercising the real repository implementation once it is available.
type commitFilePatchWriter interface {
	WriteCommitFilePatch(context.Context, string, string, io.Writer) error
}

func patchWriter(t *testing.T, r *Repository) commitFilePatchWriter {
	t.Helper()
	w, ok := any(r).(commitFilePatchWriter)
	if !ok {
		t.Fatal("repository cannot stream a historical file patch")
	}
	return w
}
func applyOwnedPatch(t *testing.T, source, base string, patch []byte) string {
	t.Helper()
	target := t.TempDir()
	if base == "" {
		git(t, target, "init", "-b", "main")
	} else {
		git(t, source, "clone", "--quiet", "--no-hardlinks", "--", source, target)
		git(t, target, "checkout", "--detach", base)
	}
	cmd := exec.Command("git", "-C", target, "apply", "--binary", "--index", "--")
	cmd.Stdin = bytes.NewReader(patch)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("apply exported patch: %v: %s", err, out)
	}
	return target
}
func exportOwnedPatch(t *testing.T, root, sha, path string) []byte {
	t.Helper()
	r := openTest(t, root)
	head := git(t, root, "rev-parse", "HEAD")
	index := git(t, root, "ls-files", "--stage", "-z")
	status := git(t, root, "status", "--porcelain=v1", "-z")
	var patch bytes.Buffer
	if err := patchWriter(t, r).WriteCommitFilePatch(context.Background(), sha, path, &patch); err != nil {
		t.Fatal(err)
	}
	if git(t, root, "rev-parse", "HEAD") != head || git(t, root, "ls-files", "--stage", "-z") != index || git(t, root, "status", "--porcelain=v1", "-z") != status {
		t.Fatal("patch export changed source Git state")
	}
	return patch.Bytes()
}
func TestCommitFilePatchRootUsesCommittedLiteralPath(t *testing.T) {
	root := fixture(t)
	path := ":(glob) 文件\t.txt"
	write(t, root, path, "committed\n")
	write(t, root, "control.txt", "not exported\n")
	git(t, root, "add", "--all")
	git(t, root, "commit", "-m", "root")
	sha := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
	write(t, root, path, "dirty work\n")
	patch := exportOwnedPatch(t, root, sha, path)
	target := applyOwnedPatch(t, root, "", patch)
	got, err := os.ReadFile(filepath.Join(target, path))
	if err != nil || string(got) != "committed\n" {
		t.Fatalf("wrong exported bytes: %q %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(target, "control.txt")); !os.IsNotExist(err) {
		t.Fatal("single-file patch included control file")
	}
	got, err = os.ReadFile(filepath.Join(root, path))
	if err != nil || string(got) != "dirty work\n" {
		t.Fatal("export changed source worktree")
	}
}
func TestCommitFilePatchRenameAppliesBothPathsOnly(t *testing.T) {
	root := fixture(t)
	old, newPath := "old name.txt", "folder/renamed 文件\t.txt"
	write(t, root, old, "unchanged rename content\n")
	write(t, root, "control.txt", "before\n")
	git(t, root, "add", "--all")
	git(t, root, "commit", "-m", "base")
	base := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
	if err := os.MkdirAll(filepath.Join(root, "folder"), 0700); err != nil {
		t.Fatal(err)
	}
	git(t, root, "mv", "--", old, newPath)
	write(t, root, "control.txt", "after\n")
	git(t, root, "add", "--all")
	git(t, root, "commit", "-m", "rename and unrelated change")
	sha := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
	patch := exportOwnedPatch(t, root, sha, newPath)
	target := applyOwnedPatch(t, root, base, patch)
	if _, err := os.Stat(filepath.Join(target, old)); !os.IsNotExist(err) {
		t.Fatal("rename source survived patch")
	}
	got, err := os.ReadFile(filepath.Join(target, newPath))
	if err != nil || string(got) != "unchanged rename content\n" {
		t.Fatalf("wrong rename target: %q %v", got, err)
	}
	got, err = os.ReadFile(filepath.Join(target, "control.txt"))
	if err != nil || string(got) != "before\n" {
		t.Fatal("patch included unrelated commit change")
	}
}
func TestCommitFilePatchRenameExcludesReusedSourcePath(t *testing.T) {
	root := fixture(t)
	write(t, root, "old.txt", "original renamed file\n")
	git(t, root, "add", "--all")
	git(t, root, "commit", "-m", "base")
	base := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
	git(t, root, "mv", "--", "old.txt", "new.txt")
	write(t, root, "old.txt/control.txt", "unrelated file inside a new directory at the old name\n")
	git(t, root, "add", "--all")
	git(t, root, "commit", "-m", "rename and reuse old name")
	sha := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
	patch := exportOwnedPatch(t, root, sha, "new.txt")
	target := applyOwnedPatch(t, root, base, patch)
	if _, err := os.Stat(filepath.Join(target, "old.txt")); !os.IsNotExist(err) {
		files, _ := openTest(t, root).CommitFiles(context.Background(), sha)
		t.Fatalf("selected rename exported unrelated replacement at the old path; files=%+v patch=%s", files, patch)
	}
	got, err := os.ReadFile(filepath.Join(target, "new.txt"))
	if err != nil || string(got) != "original renamed file\n" {
		t.Fatal("selected rename bytes missing")
	}
}
func TestCommitFilePatchTextAndDeletionIgnorePresentationConfig(t *testing.T) {
	for _, deleted := range []bool{false, true} {
		t.Run(map[bool]string{false: "modified", true: "deleted"}[deleted], func(t *testing.T) {
			root := fixture(t)
			seed(t, root)
			base := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
			if deleted {
				git(t, root, "rm", "--", "base.txt")
			} else {
				write(t, root, "base.txt", "updated\n")
				git(t, root, "add", "--", "base.txt")
			}
			git(t, root, "commit", "-m", "selected change")
			git(t, root, "config", "diff.noprefix", "true")
			git(t, root, "config", "diff.outputIndicatorNew", ">")
			git(t, root, "config", "diff.outputIndicatorOld", "<")
			git(t, root, "config", "diff.outputIndicatorContext", "_")
			sha := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
			patch := exportOwnedPatch(t, root, sha, "base.txt")
			target := applyOwnedPatch(t, root, base, patch)
			got, err := os.ReadFile(filepath.Join(target, "base.txt"))
			if deleted {
				if !os.IsNotExist(err) {
					t.Fatal("deleted file remains")
				}
			} else if err != nil || string(got) != "updated\n" {
				t.Fatal("modified file bytes differ")
			}
		})
	}
}
func TestCommitFilePatchMergeUsesFirstParent(t *testing.T) {
	root := fixture(t)
	write(t, root, "selected.txt", "base\n")
	write(t, root, "control.txt", "base\n")
	git(t, root, "add", "--all")
	git(t, root, "commit", "-m", "base")
	git(t, root, "checkout", "-b", "side")
	write(t, root, "selected.txt", "side\n")
	git(t, root, "commit", "-am", "side")
	git(t, root, "checkout", "main")
	write(t, root, "control.txt", "main\n")
	git(t, root, "commit", "-am", "main")
	base := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
	git(t, root, "merge", "--no-ff", "side", "-m", "merge")
	sha := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
	patch := exportOwnedPatch(t, root, sha, "selected.txt")
	target := applyOwnedPatch(t, root, base, patch)
	got, err := os.ReadFile(filepath.Join(target, "selected.txt"))
	if err != nil || string(got) != "side\n" {
		t.Fatalf("merge patch wrong parent: %q %v", got, err)
	}
}
func TestCommitFilePatchStreamsBinaryBeyondPreviewLimit(t *testing.T) {
	root := fixture(t)
	payload := make([]byte, 5<<20)
	_, _ = rand.New(rand.NewSource(7)).Read(payload)
	if err := os.WriteFile(filepath.Join(root, "large.bin"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "--", "large.bin")
	git(t, root, "commit", "-m", "binary root")
	sha := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
	patch := exportOwnedPatch(t, root, sha, "large.bin")
	if len(patch) <= outputLimit || !bytes.Contains(patch, []byte("GIT binary patch")) {
		t.Fatalf("binary patch missing or limited: %d", len(patch))
	}
	target := applyOwnedPatch(t, root, "", patch)
	got, err := os.ReadFile(filepath.Join(target, "large.bin"))
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatal("binary patch did not reproduce exact bytes")
	}
}

type failingPatchWriter struct{}

func (failingPatchWriter) Write([]byte) (int, error) { return 0, errors.New("owned writer failure") }
func TestCommitFilePatchRejectsInvalidTargetAndWriteFailure(t *testing.T) {
	root := fixture(t)
	seed(t, root)
	r := openTest(t, root)
	w := patchWriter(t, r)
	sha := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
	for _, tc := range []struct{ sha, path string }{{"not-a-sha", "base.txt"}, {sha, "../outside"}, {sha, "not-changed.txt"}} {
		var out bytes.Buffer
		if err := w.WriteCommitFilePatch(context.Background(), tc.sha, tc.path, &out); err == nil || out.Len() != 0 {
			t.Fatalf("invalid target exported: %#v", tc)
		}
	}
	if err := w.WriteCommitFilePatch(context.Background(), sha, "base.txt", failingPatchWriter{}); err == nil {
		t.Fatal("writer failure ignored")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out bytes.Buffer
	if err := w.WriteCommitFilePatch(ctx, sha, "base.txt", &out); err == nil || out.Len() != 0 {
		t.Fatal("cancelled export produced bytes")
	}
}
