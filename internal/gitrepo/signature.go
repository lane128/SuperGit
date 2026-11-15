package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Signature is Git's verdict on a commit's signature. Status is the letter
// `%G?` prints: G a good signature, B a bad one, U a good signature whose key
// has no known trust, X an expired signature, Y an expired key, R a revoked
// key, E one Git could not check, and N no signature at all. Signer is `%GS`,
// the name on the key (or its id) when Git can name one.
type Signature struct {
	Status string
	Signer string
}

// Signed reports whether there is a signature to talk about. An unreadable or
// empty answer is treated like an unsigned commit: the details pane then shows
// nothing rather than a badge saying it does not know.
func (s Signature) Signed() bool {
	switch s.Status {
	case "", "N":
		return false
	}
	return true
}

// CommitSignature has Git verify one commit's signature. Git does the checking
// itself (GPG or SSH, whichever signed the commit), so the answer matches what
// `git log --show-signature` would print. A machine without the key or the
// verifier still gets an answer: Git reports "could not check" (E) instead of
// failing, so a failed call means the commit itself could not be read.
func (r *Repository) CommitSignature(ctx context.Context, sha string) (Signature, error) {
	if !oidRE.MatchString(sha) {
		return Signature{}, errors.New("无效的提交 SHA")
	}
	out, err := run(ctx, r.Root, "", "show", "--no-patch", "--no-show-signature", "--encoding=UTF-8", "--format=format:%G?%x00%GS%x00", sha, "--")
	if err != nil {
		return Signature{}, err
	}
	return parseSignature(out)
}

// parseSignature reads the two NUL-terminated fields `%G?` and `%GS` print.
func parseSignature(out string) (Signature, error) {
	if !strings.HasSuffix(out, "\x00") {
		return Signature{}, fmt.Errorf("无效的签名记录")
	}
	fields := strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
	if len(fields) != 2 {
		return Signature{}, fmt.Errorf("无效的签名记录")
	}
	return Signature{Status: strings.TrimSpace(fields[0]), Signer: strings.TrimSpace(fields[1])}, nil
}
