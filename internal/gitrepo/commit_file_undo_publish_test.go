package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type undoPublisher interface {
	ApplyPreparedCommitFileUndo(context.Context, *PreparedCommitFileUndo) error
}

func publishUndo(t *testing.T, r *Repository, ctx context.Context, p *PreparedCommitFileUndo) error {
	t.Helper()
	publisher, ok := any(r).(undoPublisher)
	if !ok {
		t.Fatal("guarded undo publication is unavailable")
	}
	return publisher.ApplyPreparedCommitFileUndo(ctx, p)
}
func TestPublishCommitFileUndoOnlyChangesSelectedWorktreeFile(t *testing.T) {
	path := "目录/:(glob) 文件.txt"
	r, _, source, status := undoCheckFixture(t, path)
	p, err := r.PrepareCommitFileUndo(context.Background(), source, path, status)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	head, index := git(t, r.Root, "rev-parse", "HEAD"), git(t, r.Root, "ls-files", "--stage", "-z")
	if err := publishUndo(t, r, context.Background(), p); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(r.Root, path))
	if err != nil || string(got) != "original selected\n" {
		t.Fatal("selected file was not reversed")
	}
	control, err := os.ReadFile(filepath.Join(r.Root, "other/control.txt"))
	if err != nil || string(control) != "unchanged control\n" {
		t.Fatal("unselected file changed")
	}
	if git(t, r.Root, "rev-parse", "HEAD") != head || git(t, r.Root, "ls-files", "--stage", "-z") != index {
		t.Fatal("publication changed HEAD or index")
	}
	current, err := r.Status(context.Background())
	if err != nil || len(current.Entries) != 1 || current.Entries[0].Path != path || current.Entries[0].Index != '.' || current.Entries[0].Worktree != 'M' {
		t.Fatalf("wrong resulting status: %+v %v", current, err)
	}
}
func TestPublishCommitFileUndoRejectsChangedTarget(t *testing.T) {
	for _, kind := range []string{"cancelled", "closed", "head", "branch", "index", "hidden-edit", "identity", "mode", "other-working"} {
		t.Run(kind, func(t *testing.T) {
			r, _, source, status := undoCheckFixture(t, "selected.txt")
			p, err := r.PrepareCommitFileUndo(context.Background(), source, "selected.txt", status)
			if err != nil {
				t.Fatal(err)
			}
			defer p.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "cancelled":
				cancel()
			case "closed":
				if err := p.Close(); err != nil {
					t.Fatal(err)
				}
			case "head":
				write(t, r.Root, "later.txt", "later\n")
				git(t, r.Root, "add", "--all")
				git(t, r.Root, "commit", "-m", "later")
			case "branch":
				git(t, r.Root, "checkout", "-b", "other")
			case "index":
				write(t, r.Root, "other/control.txt", "staged change\n")
				git(t, r.Root, "add", "--all")
			case "hidden-edit":
				git(t, r.Root, "update-index", "--assume-unchanged", "--", "selected.txt")
				write(t, r.Root, "selected.txt", "hidden edit\n")
			case "identity":
				replacement := filepath.Join(t.TempDir(), "replacement")
				if err := os.WriteFile(replacement, []byte("source selected\n"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(replacement, filepath.Join(r.Root, "selected.txt")); err != nil {
					t.Fatal(err)
				}
			case "mode":
				if err := os.Chmod(filepath.Join(r.Root, "selected.txt"), 0755); err != nil {
					t.Fatal(err)
				}
			case "other-working":
				write(t, r.Root, "other/control.txt", "external edit\n")
			}
			head, index, beforeStatus := git(t, r.Root, "rev-parse", "HEAD"), git(t, r.Root, "ls-files", "--stage", "-z"), git(t, r.Root, "status", "--porcelain=v1", "-z")
			before, err := os.ReadFile(filepath.Join(r.Root, "selected.txt"))
			if err != nil {
				t.Fatal(err)
			}
			if err := publishUndo(t, r, ctx, p); err == nil {
				t.Fatal("changed/cancelled target was overwritten")
			}
			after, err := os.ReadFile(filepath.Join(r.Root, "selected.txt"))
			if err != nil || string(after) != string(before) || git(t, r.Root, "rev-parse", "HEAD") != head || git(t, r.Root, "ls-files", "--stage", "-z") != index || git(t, r.Root, "status", "--porcelain=v1", "-z") != beforeStatus {
				t.Fatal("rejected operation changed original or leaked temporary files")
			}
		})
	}
}
