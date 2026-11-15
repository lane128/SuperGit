package gitrepo

import (
	"context"
	"strings"
	"testing"
)

// Git answers with two NUL-terminated fields, the %G? letter and the signer.
// An unsigned commit fills the letter with N and leaves the signer empty.
func TestParseSignatureReadsBothFields(t *testing.T) {
	sig, err := parseSignature("G\x00Ada Lovelace\x00")
	if err != nil {
		t.Fatalf("parseSignature: %v", err)
	}
	if sig.Status != "G" || sig.Signer != "Ada Lovelace" || !sig.Signed() {
		t.Errorf("signature = %#v; want a good signature by Ada Lovelace", sig)
	}

	unsigned, err := parseSignature("N\x00\x00")
	if err != nil {
		t.Fatalf("parseSignature(unsigned): %v", err)
	}
	if unsigned.Signer != "" || unsigned.Signed() {
		t.Errorf("unsigned = %#v; want no signer and nothing to show", unsigned)
	}

	if _, err := parseSignature("N\x00"); err == nil {
		t.Error("parseSignature accepted a record with one field")
	}
	if _, err := parseSignature("N\x00\x00\n"); err == nil {
		t.Error("parseSignature accepted a record without the trailing terminator")
	}
}

// The call asks Git itself, so an ordinary unsigned commit answers N rather
// than failing: the details pane shows nothing for it.
func TestCommitSignatureReportsAnUnsignedCommit(t *testing.T) {
	dir := fixture(t)
	git(t, dir, "commit", "--allow-empty", "-m", "unsigned")
	sha := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	r := openTest(t, dir)
	sig, err := r.CommitSignature(context.Background(), sha)
	if err != nil {
		t.Fatalf("CommitSignature: %v", err)
	}
	if sig.Status != "N" || sig.Signed() {
		t.Errorf("signature = %#v; want status N and nothing to show", sig)
	}
	if _, err := r.CommitSignature(context.Background(), "not-a-sha"); err == nil {
		t.Error("CommitSignature accepted a value that is not a SHA")
	}
}
