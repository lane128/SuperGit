package delivery

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestStoreRoundTripAndList(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	if tasks, err := store.List(); err != nil || len(tasks) != 0 {
		t.Fatalf("empty list = %v, %v", tasks, err)
	}
	later := taskStartTime.Add(time.Minute)
	task := newLedgerTask(t, "first-z", taskStartTime)
	advanceLedgerTask(t, &task, PhaseAnalysis, later)
	if err := task.SetCandidate("candidate-a", later); err != nil {
		t.Fatal(err)
	}
	recordLedgerCheck(t, &task, Check{Name: "analysis", Candidate: "candidate-a", Passed: true, Evidence: "scope confirmed", RecordedAt: later})
	if err := task.AuthorizeRevision(AuthorizationRevision{By: "maintainer", Release: true, VersionPolicy: "minor", At: later}, later); err != nil {
		t.Fatal(err)
	}

	for _, entry := range []Task{newLedgerTask(t, "a-later", later), task, newLedgerTask(t, "first-a", taskStartTime)} {
		if err := store.Create(entry); err != nil {
			t.Fatal(err)
		}
	}
	store = NewStore(root)
	got, err := store.Load(task.ID)
	if err != nil || !reflect.DeepEqual(got, task) {
		t.Fatalf("Load = %+v, %v; want %+v", got, err, task)
	}

	for path, want := range map[string]os.FileMode{
		filepath.Join(root, "tasks"):                  0700,
		filepath.Join(root, "tasks", task.ID+".json"): 0600,
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if mode := info.Mode().Perm(); mode != want {
			t.Errorf("%s has mode %04o, want %04o", path, mode, want)
		}
	}
	tasks, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(tasks))
	for _, entry := range tasks {
		ids = append(ids, entry.ID)
	}
	if want := []string{"first-a", "first-z", "a-later"}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("list order = %v, want %v", ids, want)
	}
}

func TestStoreRejectsDuplicateTaskAndBadID(t *testing.T) {
	store := NewStore(t.TempDir())
	task := newLedgerTask(t, "task-1", taskStartTime)
	if err := store.Create(task); err != nil {
		t.Fatal(err)
	}
	duplicate := task
	duplicate.Request = "different request"
	if err := store.Create(duplicate); !errors.Is(err, os.ErrExist) {
		t.Fatalf("duplicate Create error = %v, want os.ErrExist", err)
	}
	if got, err := store.Load(task.ID); err != nil || got.Request != task.Request {
		t.Fatalf("original task replaced by duplicate: %+v, %v", got, err)
	}
	for _, id := range []string{"", "../escape", "with/slash", "UPPER", "under_score", "trailing\n"} {
		bad := task
		bad.ID = id
		if err := store.Create(bad); err == nil {
			t.Errorf("Create accepted ID %q", id)
		}
		if err := store.Save(bad); err == nil {
			t.Errorf("Save accepted ID %q", id)
		}
		if _, err := store.Load(id); err == nil || errors.Is(err, os.ErrNotExist) {
			t.Errorf("Load(%q) returned %v, want invalid ID error", id, err)
		}
	}
	if _, err := store.Load("missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing Load error = %v, want os.ErrNotExist", err)
	}
}

func TestStoreUpdatePersistsOnlySuccessfulChanges(t *testing.T) {
	store := NewStore(t.TempDir())
	task := newLedgerTask(t, "task", taskStartTime)
	if err := store.Create(task); err != nil {
		t.Fatal(err)
	}
	later := taskStartTime.Add(time.Minute)
	refused := errors.New("stop the update")
	if _, err := store.Update(task.ID, func(current *Task) error {
		if err := current.SetTier("T3", later); err != nil {
			return err
		}
		return refused
	}); !errors.Is(err, refused) {
		t.Fatalf("failed Update error = %v", err)
	}
	if got, err := store.Load(task.ID); err != nil || !reflect.DeepEqual(got, task) {
		t.Fatalf("failed Update changed the stored task: %+v, %v", got, err)
	}
	got, err := store.Update(task.ID, func(current *Task) error { return current.SetTier("T3", later) })
	if err != nil || got.Tier != "T3" || !got.UpdatedAt.Equal(later) || len(got.History) != 2 {
		t.Fatalf("successful Update = %+v, %v", got, err)
	}
	if persisted, err := store.Load(task.ID); err != nil || !reflect.DeepEqual(persisted, got) {
		t.Fatalf("stored Update = %+v, %v; want %+v", persisted, err, got)
	}
	if _, err := store.Update(task.ID, func(current *Task) error {
		current.ID = "different"
		return nil
	}); err == nil {
		t.Fatal("Update accepted an ID change")
	}
	if _, err := store.Load("different"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Update created a task under a different ID: %v", err)
	}
}

func TestStoreReportsCorruptFile(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.Create(newLedgerTask(t, "task", taskStartTime)); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Root, "tasks", "task.json")
	if err := os.WriteFile(path, []byte(`{"ID":`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load("task"); err == nil || !strings.Contains(err.Error(), "task.json") {
		t.Fatalf("Load corrupt task error = %v", err)
	}
	if _, err := store.List(); err == nil || !strings.Contains(err.Error(), "task.json") {
		t.Fatalf("List corrupt task error = %v", err)
	}
}

func TestStoreSerializesConcurrentWriters(t *testing.T) {
	root := t.TempDir()
	task := newLedgerTask(t, "task", taskStartTime)
	const writers = 8
	var group sync.WaitGroup
	createResults := make(chan error, writers)
	for range writers {
		group.Go(func() { createResults <- NewStore(root).Create(task) })
	}
	group.Wait()
	close(createResults)
	created := 0
	for err := range createResults {
		if err == nil {
			created++
		} else if !errors.Is(err, os.ErrExist) {
			t.Fatal(err)
		}
	}
	if created != 1 {
		t.Fatalf("%d concurrent Create calls succeeded, want 1", created)
	}

	updateResults := make(chan error, writers)
	for range writers {
		group.Go(func() {
			_, err := NewStore(root).Update(task.ID, func(current *Task) error {
				return current.SetPR(current.PR+1, taskStartTime.Add(time.Minute))
			})
			updateResults <- err
		})
	}
	group.Wait()
	close(updateResults)
	for err := range updateResults {
		if err != nil {
			t.Fatal(err)
		}
	}
	got, err := NewStore(root).Load(task.ID)
	if err != nil || got.PR != writers || len(got.History) != writers+1 {
		t.Fatalf("concurrent updates lost data: PR = %d, history contains %d events, error = %v", got.PR, len(got.History), err)
	}
}
