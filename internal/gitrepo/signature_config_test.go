package gitrepo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A user who sets log.showSignature=true has git print "Good ... signature"
// before every commit. The reads here parse git's stdout, so that text used to
// become the SHA (or fail outright): history said "invalid starting SHA",
// file history returned a SHA with the signature sentence in it, and commit
// details failed on "Good". The reads pin the setting off instead.
func TestSignatureDisplayConfigDoesNotCorruptReads(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen is not available for a signed commit")
	}
	dir := fixture(t)
	key := filepath.Join(t.TempDir(), "signing_key")
	if out, err := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", key).CombinedOutput(); err != nil {
		t.Skipf("ssh-keygen cannot make a key: %v\n%s", err, out)
	}
	pub, err := os.ReadFile(key + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	signers := filepath.Join(t.TempDir(), "allowed_signers")
	if err := os.WriteFile(signers, []byte("test@example.invalid "+string(pub)), 0644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "config", "gpg.format", "ssh")
	git(t, dir, "config", "user.signingkey", key)
	git(t, dir, "config", "gpg.ssh.allowedSignersFile", signers)
	write(t, dir, "base.txt", "original\n")
	git(t, dir, "add", "--", "base.txt")
	git(t, dir, "commit", "-S", "-m", "signed commit")
	git(t, dir, "config", "log.showSignature", "true")

	// The setting really does put the verification line in front of the SHA;
	// without it the test would pass even if the reads were unprotected.
	raw := git(t, dir, "log", "-1", "--format=%H")
	if !strings.Contains(raw, "Good") {
		t.Fatalf("the fixture does not print a signature; got %q", raw)
	}
	sha := strings.Fields(strings.SplitN(raw, "\n", 2)[1])[0]
	oid := regexp.MustCompile(`^[0-9a-f]{40}$`)
	if !oid.MatchString(sha) {
		t.Fatalf("the commit SHA in the fixture is not an OID: %q", sha)
	}

	r := openTest(t, dir)
	commits, err := r.History(context.Background(), 10)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(commits) != 1 || commits[0].SHA != sha {
		t.Fatalf("History = %+v; want the signed commit %s", commits, sha)
	}
	fileCommits, err := r.FileHistory(context.Background(), "base.txt", 10)
	if err != nil {
		t.Fatalf("FileHistory: %v", err)
	}
	if len(fileCommits) != 1 || !oid.MatchString(fileCommits[0].SHA) || fileCommits[0].SHA != sha {
		t.Fatalf("FileHistory = %+v; want the signed commit %s", fileCommits, sha)
	}
	if _, err := r.CommitFiles(context.Background(), sha); err != nil {
		t.Fatalf("CommitFiles: %v", err)
	}
	if _, err := r.ShowAll(context.Background(), sha); err != nil {
		t.Fatalf("ShowAll: %v", err)
	}
	if _, err := r.Diff(context.Background(), "base.txt", false); err != nil {
		t.Fatalf("Diff: %v", err)
	}
}
