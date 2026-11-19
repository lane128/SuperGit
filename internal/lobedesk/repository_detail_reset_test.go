package lobedesk

import (
	"context"
	"testing"

	"supergit/internal/gitrepo"
)

// A successful root switch must discard the previous repository's metadata
// before the new commit's asynchronous detail result can arrive.
func TestRepositorySwitchClearsCommitMetadataBeforeDetailsArrive(t *testing.T) {
	a, _, _ := worktreeApp(t)
	_, _, other := worktreeApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	a.ctx = ctx
	done := make(chan func(), 128)
	a.runOnMain = func(fn func()) { done <- fn }
	t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
	a.commitMessage = "Previous repository subject\n\nPRIVATE previous repository body"
	a.commitMessageErr = "previous message read error"
	a.commitSig = gitrepo.Signature{Signer: "previous repository signer"}
	a.commitFilesErr = "previous file read error"
	a.openRepository(other, true)
	// Only accept the repository snapshot. Leave the newly launched detail
	// continuations queued, exposing exactly the frame seen in native testing.
	runContinuation(t, done)
	if a.loading || a.repo.Root != other {
		t.Fatal("new repository snapshot was not accepted")
	}
	if a.commitMessage != "" || a.commitMessageErr != "" || a.commitSig.Signer != "" || a.commitFilesErr != "" {
		t.Fatalf("old metadata visible during new details read: body=%q messageErr=%q signer=%q filesErr=%q", a.commitMessage, a.commitMessageErr, a.commitSig.Signer, a.commitFilesErr)
	}
}
