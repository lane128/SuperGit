package gitrepo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestProgressWriterParsesMonotonicFetchAndPush(t *testing.T) {
	t.Run("fetch", func(t *testing.T) {
		var sink = limitedBuffer{limit: 4096}
		var updates []OperationProgress
		w := newProgressStderrWriter(&sink, "fetch", func(p OperationProgress) {
			updates = append(updates, p)
		})
		stream := strings.Join([]string{
			"remote: Enumerating objects: 182, done.\n",
			"remote: Counting objects:  50% (91/182)\rremote: Counting objects: 100% (182/182), done.\n",
			"remote: Compressing objects: 100% (121/121), done.\n",
			"Receiving objects:  50% (91/182), 1.00 MiB | 2.00 MiB/s\rReceiving objects: 100% (182/182), 2.10 MiB, done.\n",
			"Resolving deltas: 100% (45/45), done.\n",
		}, "")
		if _, err := w.Write([]byte(stream)); err != nil {
			t.Fatalf("write failed: %v", err)
		}
		w.Flush()
		if len(updates) < 5 {
			t.Fatalf("expected parsed fetch progress updates, got %d: %+v", len(updates), updates)
		}
		if !updates[0].Indeterminate || updates[0].Phase != "Enumerating objects" {
			t.Fatalf("first update = %+v; want indeterminate Enumerating objects", updates[0])
		}
		var prev float32
		for _, u := range updates[1:] {
			if u.Indeterminate {
				t.Fatalf("unexpected indeterminate update after percent phase: %+v", u)
			}
			if u.Fraction < prev {
				t.Fatalf("fraction moved backwards: %v -> %v (%+v)", prev, u.Fraction, u)
			}
			prev = u.Fraction
		}
		if prev < 0.95 {
			t.Fatalf("final fetch fraction = %v; want >= 0.95", prev)
		}
	})

	t.Run("push", func(t *testing.T) {
		var sink = limitedBuffer{limit: 4096}
		var updates []OperationProgress
		w := newProgressStderrWriter(&sink, "push", func(p OperationProgress) {
			updates = append(updates, p)
		})
		stream := "Enumerating objects: 20, done.\n" +
			"Counting objects: 100% (20/20), done.\n" +
			"Compressing objects:  50% (8/16)\rCompressing objects: 100% (16/16), done.\n" +
			"Writing objects:  50% (10/20)\rWriting objects: 100% (20/20), done.\n" +
			"remote: Resolving deltas: 100% (5/5), done.\n"
		_, _ = w.Write([]byte(stream))
		w.Flush()
		if len(updates) < 5 {
			t.Fatalf("expected push updates, got %+v", updates)
		}
		last := updates[len(updates)-1]
		if last.Fraction < 0.99 {
			t.Fatalf("final push fraction = %v; want ~1.0", last.Fraction)
		}
	})
}

func TestProgressWriterCollapsesCarriageReturnsAndKeepsErrors(t *testing.T) {
	sink := limitedBuffer{limit: 1024}
	count := 0
	w := newProgressStderrWriter(&sink, "fetch", func(OperationProgress) { count++ })
	for i := 0; i < 400; i++ {
		_, _ = fmt.Fprintf(w, "Receiving objects: %3d%% (%d/400), 12.34 MiB | 5.00 MiB/s\r", i*100/400, i)
	}
	_, _ = w.Write([]byte("fatal: repository not found\n"))
	w.Flush()
	if sink.truncated {
		t.Fatalf("collapsed progress frames unexpectedly truncated stderr buffer: len=%d", len(sink.String()))
	}
	if !strings.Contains(sink.String(), "fatal: repository not found") {
		t.Fatalf("stderr lost fatal error message: %q", sink.String())
	}
	if count < 50 {
		t.Fatalf("expected progress callbacks during transfer, got %d", count)
	}
}

func TestWithScaledProgressMapsRange(t *testing.T) {
	var got []float32
	ctx := WithProgress(context.Background(), func(p OperationProgress) {
		got = append(got, p.Fraction)
	})
	scaled := WithScaledProgress(ctx, 0.5, 1.0)
	fn := progressFromContext(scaled)
	fn(OperationProgress{Active: true, Fraction: 0.0})
	fn(OperationProgress{Active: true, Fraction: 0.5})
	fn(OperationProgress{Active: true, Fraction: 1.0})
	want := []float32{0.5, 0.75, 1.0}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("scaled[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestRealFetchAndPushStreamProgress(t *testing.T) {
	ctx := context.Background()
	bare := filepath.Join(t.TempDir(), "remote.git")
	if _, err := run(ctx, "", "", "init", "--bare", bare); err != nil {
		t.Fatalf("init bare: %v", err)
	}

	work1 := filepath.Join(t.TempDir(), "work1")
	repo1, err := Init(ctx, work1)
	if err != nil {
		t.Fatalf("init work1: %v", err)
	}
	if _, err := run(ctx, repo1.Root, "", "config", "user.name", "Test User"); err != nil {
		t.Fatal(err)
	}
	if _, err := run(ctx, repo1.Root, "", "config", "user.email", "test@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo1.Root, "data.txt"), []byte(strings.Repeat("supergit progress test\n", 200)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := repo1.Stage(ctx, "data.txt"); err != nil {
		t.Fatal(err)
	}
	if err := repo1.Commit(ctx, "initial commit"); err != nil {
		t.Fatal(err)
	}
	if err := repo1.AddRemote(ctx, "origin", bare); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var pushUpdates []OperationProgress
	pushCtx := WithProgress(ctx, func(p OperationProgress) {
		mu.Lock()
		pushUpdates = append(pushUpdates, p)
		mu.Unlock()
	})
	if err := repo1.Push(pushCtx, "origin", "main", "main", true); err != nil {
		t.Fatalf("push with progress failed: %v", err)
	}
	if len(pushUpdates) == 0 {
		t.Fatal("expected git push --progress to emit at least one progress update")
	}

	cloneDir := filepath.Join(t.TempDir(), "work2")
	var cloneUpdates []OperationProgress
	cloneCtx := WithProgress(ctx, func(p OperationProgress) {
		mu.Lock()
		cloneUpdates = append(cloneUpdates, p)
		mu.Unlock()
	})
	// Use file:// so Git exercises pack transfer rather than local hardlink cloning.
	repo2, err := Clone(cloneCtx, "file://"+bare, cloneDir)
	if err != nil {
		t.Fatalf("clone with progress failed: %v", err)
	}
	if len(cloneUpdates) == 0 {
		t.Fatal("expected git clone --progress to emit at least one progress update")
	}

	if err := os.WriteFile(filepath.Join(repo1.Root, "more.txt"), []byte("second commit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := repo1.Stage(ctx, "more.txt"); err != nil {
		t.Fatal(err)
	}
	if err := repo1.Commit(ctx, "second commit"); err != nil {
		t.Fatal(err)
	}
	if err := repo1.Push(ctx, "origin", "main", "main", false); err != nil {
		t.Fatal(err)
	}

	var fetchUpdates []OperationProgress
	fetchCtx := WithProgress(ctx, func(p OperationProgress) {
		mu.Lock()
		fetchUpdates = append(fetchUpdates, p)
		mu.Unlock()
	})
	if err := repo2.Fetch(fetchCtx, "origin"); err != nil {
		t.Fatalf("fetch with progress failed: %v", err)
	}
	if len(fetchUpdates) == 0 {
		t.Fatal("expected git fetch --progress to emit at least one progress update")
	}
}
