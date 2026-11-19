package lobedesk

import (
	"context"
	"errors"
	"testing"
	"time"

	"supergit/internal/gitrepo"
)

// Delaying the real UI continuation until worker cleanup reproduces the event
// loop ordering: deferred context cleanup must not reclassify a Git failure.
func TestOperationCompletionKeepsResultBeforeContextCleanup(t *testing.T) {
	for _, outcome := range []string{"failure", "cancel", "success"} {
		t.Run(outcome, func(t *testing.T) {
			a, _, _ := worktreeApp(t)
			ctx, cancel := context.WithCancel(context.Background())
			a.ctx = ctx
			callbacks := make(chan func(), 64)
			a.runOnMain = func(fn func()) { callbacks <- fn }
			t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
			started := make(chan context.Context, 1)
			release := make(chan struct{})
			done := false
			a.performDone("operation failed", "operation succeeded", func(error) { done = true }, func(ctx context.Context, _ *gitrepo.Repository) error {
				started <- ctx
				<-release
				if outcome == "success" {
					return nil
				}
				return errors.New("owned operation failed")
			})
			var operationCtx context.Context
			select {
			case operationCtx = <-started:
			case <-time.After(10 * time.Second):
				t.Fatal("operation never started")
			}
			if outcome == "cancel" {
				a.cancelOperation()
			}
			close(release)
			select {
			case <-operationCtx.Done():
			case <-time.After(10 * time.Second):
				t.Fatal("worker did not clean up its context")
			}
			for a.busy || a.loading {
				runContinuation(t, callbacks)
			}
			if !done || len(a.notices[toastMain]) != 1 {
				t.Fatal("operation did not complete exactly once")
			}
			n := a.notices[toastMain][0]
			switch outcome {
			case "failure":
				if n.kind != "error" || n.title != "operation failed" || n.description != "owned operation failed" {
					t.Fatalf("cleanup changed failure into %+v", *n)
				}
			case "cancel":
				if n.kind != "info" || n.title != a.t("operationCancelled") {
					t.Fatalf("user cancellation became %+v", *n)
				}
			case "success":
				if n.kind != "success" || n.title != "operation succeeded" {
					t.Fatalf("cleanup changed success into %+v", *n)
				}
			}
		})
	}
}
