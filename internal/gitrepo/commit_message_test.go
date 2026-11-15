package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommitMessagePreservesFullTextAndImmutableSHA(t *testing.T) {
	dir := fixture(t)
	message := "标题 first line\ncontinued first paragraph\n\n说明：保留正文\n  indented example\n\nFixes: local-test-42\nSigned-off-by: Test Developer <test@example.invalid>\n\n"
	git(t, dir, "commit", "--allow-empty", "--cleanup=verbatim", "-m", message)
	sha := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	r := openTest(t, dir)
	got, err := r.CommitMessage(context.Background(), sha)
	if err != nil || got != message {
		t.Fatalf("message changed: %q err=%v", got, err)
	}
	git(t, dir, "commit", "--allow-empty", "-m", "different current commit")
	got, err = r.CommitMessage(context.Background(), sha)
	if err != nil || got != message {
		t.Fatal("read followed HEAD instead of captured SHA", err)
	}
	// Merge messages are metadata only, even with user signature display enabled.
	git(t, dir, "checkout", "-b", "side", sha)
	git(t, dir, "commit", "--allow-empty", "-m", "side")
	git(t, dir, "checkout", "main")
	git(t, dir, "merge", "--no-ff", "side", "-m", "Merge title", "-m", "Merge body\nTrailer: value")
	git(t, dir, "config", "log.showSignature", "true")
	sha = strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	got, err = r.CommitMessage(context.Background(), sha)
	if err != nil || got != "Merge title\n\nMerge body\nTrailer: value\n" {
		t.Fatalf("merge message: %q err=%v", got, err)
	}
}

func TestCommitMessageEmptyAndSourceEncoding(t *testing.T) {
	dir := fixture(t)
	git(t, dir, "commit", "--allow-empty", "--allow-empty-message", "-m", "")
	r := openTest(t, dir)
	sha := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	got, err := r.CommitMessage(context.Background(), sha)
	if err != nil || got != "" {
		t.Fatalf("empty message: %q %v", got, err)
	}
	git(t, dir, "config", "i18n.commitEncoding", "ISO-8859-1")
	git(t, dir, "config", "i18n.logOutputEncoding", "ISO-8859-1")
	path := filepath.Join(t.TempDir(), "message.txt")
	if err := os.WriteFile(path, []byte("caf\xe9\n\nbody r\xe9sum\xe9\n"), 0600); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "commit", "--allow-empty", "--cleanup=verbatim", "-F", path)
	sha = strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	got, err = r.CommitMessage(context.Background(), sha)
	if err != nil || got != "café\n\nbody résumé\n" {
		t.Fatalf("UTF-8 display: %q %v", got, err)
	}
}

func TestCommitMessageRejectsInvalidMissingOversizedAndCancelled(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	for _, sha := range []string{"HEAD", "--all", "-n1", "", strings.Repeat("0", 40)} {
		if text, err := r.CommitMessage(ctx, sha); err == nil || text != "" {
			t.Fatalf("accepted invalid/missing SHA %q", sha)
		}
	}
	sha := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	tree := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD^{tree}"))
	raw := "tree " + tree + "\nauthor Test <test@example.invalid> 1700000000 +0000\ncommitter Test <test@example.invalid> 1700000000 +0000\n\ninvalid byte: \xff\n"
	invalid, err := run(ctx, dir, raw, "hash-object", "-t", "commit", "-w", "--stdin")
	if err != nil {
		t.Fatal(err)
	}
	if text, err := r.CommitMessage(ctx, strings.TrimSpace(invalid)); err == nil || text != "" || !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("invalid UTF-8 was rendered: %q %v", text, err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if text, err := r.CommitMessage(cancelled, sha); err == nil || text != "" {
		t.Fatal("cancelled read returned a message")
	}
	messageFile := filepath.Join(t.TempDir(), "oversized.txt")
	if err := os.WriteFile(messageFile, []byte("Small title\n\n"+strings.Repeat("x", 1<<20)), 0600); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "commit", "--allow-empty", "-F", messageFile)
	sha = strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	if text, err := r.CommitMessage(ctx, sha); err == nil || text != "" || !strings.Contains(err.Error(), "1 MiB") {
		t.Fatalf("oversized message was displayed/truncated: %d bytes %v", len(text), err)
	}
}
