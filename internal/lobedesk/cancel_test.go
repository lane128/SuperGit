package lobedesk

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Pressing cancel means the operation stopped rather than failed. The git
// error under a killed process is usually "signal: killed", so the context's
// own error is what decides; a real failure or a success is not a cancel.
func TestOperationCancelledOnlyWhenSomeonePressedCancel(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if !operationCancelled(cancelled, errors.New("signal: killed")) {
		t.Fatal("a cancelled context with a git error was not treated as cancelled")
	}

	expired, cancelExpired := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancelExpired()
	<-expired.Done()
	if operationCancelled(expired, errors.New("signal: killed")) {
		t.Fatal("the three-minute timeout was treated as a user cancel")
	}

	if operationCancelled(context.Background(), errors.New("exit status 128")) {
		t.Fatal("a plain failure was treated as cancelled")
	}
	if operationCancelled(context.Background(), nil) {
		t.Fatal("a success was treated as cancelled")
	}
}

// Nothing is running most of the time, and the title bar's cancel button only
// exists while something is, but the action must not panic if it is reached
// after the operation already finished.
func TestCancelOperationIsSafeWithNothingRunning(t *testing.T) {
	a := &app{}
	a.cancelOperation()
}

// beginOperation is the one place that sets busy and hands out the cancel, so
// the title bar's cancel button always reaches the command that is running.
// It also carries the operation's name, and finishOperation clears all of it
// whatever the outcome.
func TestBeginOperationRegistersAndFinishClears(t *testing.T) {
	a := &app{ctx: context.Background()}
	ctx, _ := a.beginOperation(busyPush)
	if !a.busy || a.opCancel == nil {
		t.Fatalf("after begin: busy=%v cancel=%v; want both set", a.busy, a.opCancel)
	}
	if a.busyKey != string(busyPush) {
		t.Fatalf("after begin: busyKey=%q; want %q", a.busyKey, busyPush)
	}
	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("the operation context has no deadline")
	}
	a.finishOperation()
	if a.busy || a.opCancel != nil || a.busyKey != "" {
		t.Fatalf("after finish: busy=%v key=%q; want both cleared", a.busy, a.busyKey)
	}
	a.cancelOperation()
}
