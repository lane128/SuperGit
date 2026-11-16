package lobedesk

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIgnoredFileRecoveryUsesCurrentLanguage(t *testing.T) {
	for _, overwrite := range []bool{false, true} {
		t.Run(fmt.Sprintf("overwrite=%t", overwrite), func(t *testing.T) {
			a, _, root := worktreeApp(t)
			run := func(dir string, args ...string) string {
				t.Helper()
				out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
				if err != nil {
					t.Fatalf("fixture Git: %v: %s", err, out)
				}
				return string(out)
			}
			name := "ignored space.bin"
			if err := os.WriteFile(filepath.Join(root, ".git", "info", "exclude"), []byte("/"+name+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(t.TempDir(), "linked")
			if overwrite {
				target = root
				run(root, "switch", "-c", "incoming")
				if err := os.WriteFile(filepath.Join(root, name), []byte("incoming tracked bytes\n"), 0600); err != nil {
					t.Fatal(err)
				}
				run(root, "add", "-f", "--", name)
				run(root, "commit", "-m", "Incoming ignored collision")
				run(root, "switch", "main")
			} else if err := a.repo.AddWorktree(context.Background(), target, "review", "main"); err != nil {
				t.Fatal(err)
			}
			local := []byte("keep binary bytes\x00\xff\n")
			path := filepath.Join(target, name)
			if err := os.WriteFile(path, local, 0600); err != nil {
				t.Fatal(err)
			}
			beforeHead := run(target, "rev-parse", "HEAD")
			beforeIndex := run(target, "ls-files", "--stage")
			beforeTrees := run(root, "worktree", "list", "--porcelain")
			var err error
			zh := fmt.Sprintf("存在被忽略文件 %q；为保留内容，此操作已停止，请先移出或备份", name)
			en := fmt.Sprintf("Ignored file %q is present. To preserve its contents, this operation stopped. Move it out or back it up before retrying.", name)
			if overwrite {
				err = a.repo.Integrate(context.Background(), "cherry-pick", "incoming")
				zh = fmt.Sprintf("目标会覆盖被忽略文件 %q，请先移出或备份", name)
				en = fmt.Sprintf("The target would overwrite ignored file %q. Move it out or back it up before retrying.", name)
			} else {
				err = a.repo.RemoveWorktree(context.Background(), target)
			}
			if err == nil || err.Error() != zh {
				t.Fatalf("missing original ignored-file refusal: %v", err)
			}
			for _, tc := range []struct {
				language locale
				want     string
			}{{localeChinese, zh}, {localeEnglish, en}} {
				a.locale = tc.language
				if got := a.operationErrorDescription(err); got != tc.want {
					t.Errorf("%s recovery=%q; want %q", tc.language, got, tc.want)
				}
			}
			got, readErr := os.ReadFile(path)
			if readErr != nil || !bytes.Equal(got, local) || run(target, "rev-parse", "HEAD") != beforeHead || run(target, "ls-files", "--stage") != beforeIndex || run(root, "worktree", "list", "--porcelain") != beforeTrees {
				t.Fatal("ignored-file guard changed content, HEAD, Index or worktree identity")
			}
		})
	}
}
