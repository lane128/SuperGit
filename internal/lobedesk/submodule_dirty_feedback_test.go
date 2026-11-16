package lobedesk

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Use an actual dirty submodule refusal: mapping a synthetic error would not
// catch the backend returning an unclassified message to the language layer.
func TestDirtySubmoduleRecoveryUsesCurrentLanguage(t *testing.T) {
	t.Setenv("GIT_ALLOW_PROTOCOL", "file")
	a, _, parent := worktreeApp(t)
	_, _, source := worktreeApp(t)
	run := func(dir string, args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("fixture Git: %v: %s", err, out)
		}
		return string(out)
	}
	name := "子 module"
	run(parent, "-c", "protocol.file.allow=always", "submodule", "add", "--name", "fixture-child", source, name)
	run(parent, "commit", "-am", "module")
	child := filepath.Join(parent, name)
	path := filepath.Join(child, "base.txt")
	local := []byte("keep dirty Unicode 字节\n")
	if err := os.WriteFile(path, local, 0600); err != nil {
		t.Fatal(err)
	}
	modules, err := a.repo.Submodules(context.Background())
	if err != nil || len(modules) != 1 || !modules[0].Dirty {
		t.Fatalf("fixture must contain one dirty submodule: %+v, %v", modules, err)
	}
	state := func() [6]string {
		return [6]string{
			run(parent, "rev-parse", "HEAD"), run(parent, "show-ref"), run(parent, "ls-files", "--stage", "-z"),
			run(child, "rev-parse", "HEAD"), run(child, "show-ref"), run(child, "ls-files", "--stage", "-z"),
		}
	}
	before := state()
	const zh = "子模块内部有改动，请先提交或 Stash；更新不会丢弃内容"
	const en = "The submodule has local changes. Commit or stash them inside the submodule before updating. Update will not discard its contents."
	for _, checked := range []bool{false, true} {
		if checked {
			err = a.repo.UpdateSubmoduleChecked(context.Background(), modules[0])
		} else {
			err = a.repo.UpdateSubmodule(context.Background(), name)
		}
		if err == nil || err.Error() != zh {
			t.Fatalf("checked=%t: missing original dirty refusal: %v", checked, err)
		}
		for _, tc := range []struct {
			language locale
			want     string
		}{{localeChinese, zh}, {localeEnglish, en}} {
			a.locale = tc.language
			if got := a.operationErrorDescription(err); got != tc.want {
				t.Errorf("checked=%t %s recovery=%q; want %q", checked, tc.language, got, tc.want)
			}
		}
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(got, local) || state() != before {
		t.Fatal("dirty submodule refusal changed working bytes, parent/child HEAD, refs or Index entries")
	}
}
