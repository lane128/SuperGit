package lobedesk

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"testing"
)

type undoResultReporter interface {
	commitFileUndoNotice(commitFileUndoTarget, error)
}

func TestCommitFileUndoResultUsesReturnedErrorDespiteLateCancellation(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		kind string
	}{
		{"successful-replacement", nil, "success"},
		{"replacement-denied", &os.LinkError{Op: "rename", Old: ".owned-temp", New: "selected.txt", Err: syscall.EACCES}, "error"},
		{"actual-cancellation", fmt.Errorf("interrupted read: %w", context.Canceled), "info"},
		{"deadline", fmt.Errorf("read timed out: %w", context.DeadlineExceeded), "error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			a := &app{ctx: ctx, locale: localeEnglish}
			reporter, ok := any(a).(undoResultReporter)
			if !ok {
				t.Fatal("undo feedback cannot classify the actual returned error independently of a late context cancellation")
			}
			target := commitFileUndoTarget{Patch: commitFilePatchTarget{Root: "/owned-feedback-fixture"}}
			reporter.commitFileUndoNotice(target, tc.err)
			notices := a.notices[toastMain]
			if len(notices) != 1 || notices[0].kind != tc.kind || notices[0].root != target.Patch.Root {
				t.Fatalf("wrong result after late cancellation: %+v", notices)
			}
			if tc.name == "replacement-denied" && notices[0].description != tc.err.Error() {
				t.Fatal("replacement failure reason was lost")
			}
		})
	}
}
