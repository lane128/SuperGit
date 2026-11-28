package delivery

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAdvanceTaskRejectsCandidateChangeAfterCodeReview(t *testing.T) {
	for _, step := range []struct {
		from, to Phase
		check    string
	}{
		{PhaseAcceptance, PhasePullRequest, "acceptance"},
		{PhasePullRequest, PhaseMerge, "remote_checks"},
	} {
		t.Run(string(step.from), func(t *testing.T) {
			store := NewStore(t.TempDir())
			task := newLedgerTask(t, "task-a", taskStartTime)
			task.Phase, task.Candidate = step.from, "reviewed"
			recordLedgerCheck(t, &task, Check{Name: step.check, Candidate: "unreviewed", Passed: true, RecordedAt: taskStartTime})
			if err := store.Create(task); err != nil {
				t.Fatal(err)
			}
			if err := store.AcquireReleaseWindow(task.ID, taskStartTime); err != nil {
				t.Fatal(err)
			}
			before := readWindowTaskFile(t, store)
			if _, err := store.AdvanceTask(task.ID, step.to, "unreviewed", taskStartTime.Add(time.Minute)); err == nil {
				t.Error("accepted a candidate change after code review")
			}
			if !bytes.Equal(before, readWindowTaskFile(t, store)) {
				t.Error("rejected candidate change modified the persisted task")
			}
		})
	}
}

func TestCodeReviewCandidateChangeRequiresNewChecks(t *testing.T) {
	task := newLedgerTask(t, "task-a", taskStartTime)
	task.Phase, task.Candidate = PhaseCodeReview, "original"
	at := taskStartTime.Add(time.Minute)
	for _, name := range []string{"code_review", "tests"} {
		recordLedgerCheck(t, &task, Check{Name: name, Candidate: "original", Passed: true, RecordedAt: taskStartTime})
	}
	if err := task.SetCandidate("revised", at); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"code_review", "tests"} {
		if err := task.Advance(PhaseAcceptance, at); err == nil || !strings.Contains(err.Error(), name) {
			t.Fatalf("advance before fresh %s = %v", name, err)
		}
		recordLedgerCheck(t, &task, Check{Name: name, Candidate: "revised", Passed: true, RecordedAt: at})
	}
	advanceLedgerTask(t, &task, PhaseAcceptance, at)
}

func TestAcquireReleaseWindowRejectsUnknownAndTerminalTasks(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.AcquireReleaseWindow("missing", taskStartTime); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("acquire unknown task = %v, want missing task error", err)
	}
	task := newLedgerTask(t, "cancelled", taskStartTime)
	task.Status = StatusCancelled
	if err := store.Create(task); err != nil {
		t.Fatal(err)
	}
	if err := store.AcquireReleaseWindow(task.ID, taskStartTime); err == nil {
		t.Error("cancelled task acquired a window")
	}
}

func TestReleaseWindowBlocksSecondTask(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	for _, id := range []string{"task-a", "task-b"} {
		if err := store.Create(newLedgerTask(t, id, taskStartTime)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.CurrentReleaseWindow(); !errors.Is(err, ErrNoReleaseWindow) {
		t.Fatalf("empty window error = %v", err)
	}
	if err := store.AcquireReleaseWindow("task-a", taskStartTime); err != nil {
		t.Fatal(err)
	}
	other := NewStore(root)
	if err := other.AcquireReleaseWindow("task-a", taskStartTime.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	window, err := other.CurrentReleaseWindow()
	if err != nil || window.TaskID != "task-a" || !window.Since.Equal(taskStartTime) {
		t.Fatalf("window = %+v, err = %v", window, err)
	}
	info, err := os.Stat(filepath.Join(root, "queue.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("queue mode = %o, want 600", info.Mode().Perm())
	}
	if err := other.AcquireReleaseWindow("task-b", taskStartTime); err == nil || !strings.Contains(err.Error(), "task-a") {
		t.Fatalf("second acquire error = %v", err)
	}
	if err := other.ReleaseReleaseWindow("task-b"); err == nil {
		t.Fatal("second task released the first task's window")
	}

	advanceWindowTaskToVerify(t, store, "task-a")
	completeWindowTask(t, store, "task-a")
	if err := other.AcquireReleaseWindow("task-b", taskStartTime); err != nil {
		t.Fatal(err)
	}
	if err := other.ReleaseReleaseWindow("task-b"); err != nil {
		t.Fatal(err)
	}
	if err := other.ReleaseReleaseWindow("task-b"); !errors.Is(err, ErrNoReleaseWindow) {
		t.Fatalf("repeated release error = %v", err)
	}
}

func TestReleaseWindowReleasedOnComplete(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.Create(newLedgerTask(t, "task-a", taskStartTime)); err != nil {
		t.Fatal(err)
	}
	if err := store.AcquireReleaseWindow("task-a", taskStartTime); err != nil {
		t.Fatal(err)
	}
	advanceWindowTaskToVerify(t, store, "task-a")
	if _, err := store.AdvanceTask("task-a", PhaseComplete, "", taskStartTime); err == nil {
		t.Fatal("completed without public verification")
	}
	window, err := store.CurrentReleaseWindow()
	if err != nil || window.TaskID != "task-a" {
		t.Fatalf("window after failed advance = %+v, err = %v", window, err)
	}
	completeWindowTask(t, store, "task-a")
	task, err := store.Load("task-a")
	if err != nil || task.Phase != PhaseComplete {
		t.Fatalf("persisted phase = %q, err = %v", task.Phase, err)
	}
	if _, err := NewStore(store.Root).CurrentReleaseWindow(); !errors.Is(err, ErrNoReleaseWindow) {
		t.Fatalf("completed window error = %v", err)
	}
	if err := store.AcquireReleaseWindow("task-a", taskStartTime); err == nil {
		t.Fatal("completed task acquired a new window")
	}
}

func TestStoreReleasePhaseWritesRequireWindow(t *testing.T) {
	for _, method := range []string{"Update", "Save"} {
		t.Run(method, func(t *testing.T) {
			store := NewStore(t.TempDir())
			createWindowTaskAtPullRequest(t, store)
			before := readWindowTaskFile(t, store)
			var err error
			if method == "Update" {
				_, err = store.Update("task-a", func(task *Task) error {
					return task.Advance(PhaseMerge, taskStartTime)
				})
			} else {
				task, loadErr := store.Load("task-a")
				if loadErr != nil {
					t.Fatal(loadErr)
				}
				advanceLedgerTask(t, &task, PhaseMerge, taskStartTime)
				err = store.Save(task)
			}
			if !errors.Is(err, ErrReleaseWindowRequired) {
				t.Errorf("%s error = %v, want ErrReleaseWindowRequired", method, err)
			}
			if !bytes.Equal(before, readWindowTaskFile(t, store)) {
				t.Error("rejected phase write changed the task file")
			}
		})
	}
}

func TestReleaseWindowCannotBeReleasedDuringDelivery(t *testing.T) {
	for _, scenario := range []string{"release while in merge", "later transition with another holder"} {
		t.Run(scenario, func(t *testing.T) {
			store := NewStore(t.TempDir())
			createWindowTaskAtPullRequest(t, store)
			if err := store.Create(newLedgerTask(t, "task-b", taskStartTime)); err != nil {
				t.Fatal(err)
			}
			if err := store.AcquireReleaseWindow("task-a", taskStartTime); err != nil {
				t.Fatal(err)
			}
			if _, err := store.AdvanceTask("task-a", PhaseMerge, "", taskStartTime); err != nil {
				t.Fatal(err)
			}
			if scenario == "release while in merge" {
				if err := store.ReleaseReleaseWindow("task-a"); err == nil {
					t.Error("released the window during merge")
				}
				if err := store.AcquireReleaseWindow("task-b", taskStartTime); err == nil {
					t.Error("task B acquired the window during task A's merge")
				}
				return
			}

			recordWindowCheck(t, store, "task-a", "merge")
			before := readWindowTaskFile(t, store)
			// Simulate a queue replaced outside the API after a task entered merge.
			if err := os.Remove(filepath.Join(store.Root, "queue.json")); err != nil {
				t.Fatal(err)
			}
			if err := store.AcquireReleaseWindow("task-b", taskStartTime); err != nil {
				t.Fatal(err)
			}
			if _, err := store.AdvanceTask("task-a", PhaseReleasePreflight, "", taskStartTime); err == nil {
				t.Error("advanced to release_preflight while task B held the window")
			}
			if !bytes.Equal(before, readWindowTaskFile(t, store)) {
				t.Error("rejected release transition changed the task file")
			}
		})
	}
}

func TestRecoverReleaseWindowBeforeMergeAfterRevocation(t *testing.T) {
	store := NewStore(t.TempDir())
	createWindowTaskAtPullRequest(t, store)
	if err := store.Create(newLedgerTask(t, "task-b", taskStartTime)); err != nil {
		t.Fatal(err)
	}
	if err := store.AcquireReleaseWindow("task-a", taskStartTime); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update("task-a", func(task *Task) error {
		return task.Revoke("maintainer", taskStartTime)
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.RecoverReleaseWindow("task-a", "maintainer", "remote main has not changed"); err != nil {
		t.Fatal(err)
	}
	if err := store.AcquireReleaseWindow("task-b", taskStartTime); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load("task-a")
	if err != nil || got.Phase != PhasePullRequest || got.Status != StatusBlocked || got.Authorization.RevokedAt == nil {
		t.Fatalf("window recovery changed revoked task: %+v, %v", got, err)
	}
	if _, err := store.AdvanceTask("task-a", PhaseMerge, "", taskStartTime); err == nil {
		t.Fatal("revoked task advanced after window recovery")
	}
}

func createWindowTaskAtPullRequest(t *testing.T, store *Store) {
	t.Helper()
	task := newLedgerTask(t, "task-a", taskStartTime)
	task.Phase, task.Candidate = PhasePullRequest, "candidate"
	recordLedgerCheck(t, &task, Check{Name: "remote_checks", Candidate: "candidate", Passed: true, RecordedAt: taskStartTime})
	if err := store.Create(task); err != nil {
		t.Fatal(err)
	}
}

func readWindowTaskFile(t *testing.T, store *Store) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(store.Root, "tasks", "task-a.json"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func advanceWindowTaskToVerify(t *testing.T, store *Store, id string) {
	t.Helper()
	if _, err := store.AdvanceTask(id, PhaseAnalysis, "", taskStartTime); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(id, func(task *Task) error {
		return task.SetCandidate("candidate", taskStartTime)
	}); err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		check string
		to    Phase
	}{
		{"analysis", PhaseDesign}, {"design", PhaseDesignReview}, {"design_review", PhaseImplementation},
		{"tests", PhaseCodeReview}, {"code_review", PhaseAcceptance}, {"acceptance", PhasePullRequest},
		{"remote_checks", PhaseMerge}, {"merge", PhaseReleasePreflight}, {"release_preflight", PhaseTag},
		{"package", PhasePublish}, {"publish", PhaseVerify},
	} {
		if step.to == PhaseAcceptance {
			recordWindowCheck(t, store, id, "tests")
		}
		if step.to == PhaseTag {
			recordWindowCheck(t, store, id, "release_selection")
		}
		recordWindowCheck(t, store, id, step.check)
		if _, err := store.AdvanceTask(id, step.to, "", taskStartTime); err != nil {
			t.Fatalf("advance to %s: %v", step.to, err)
		}
	}
}

func completeWindowTask(t *testing.T, store *Store, id string) {
	t.Helper()
	recordWindowCheck(t, store, id, "public_verify")
	if _, err := store.AdvanceTask(id, PhaseComplete, "", taskStartTime); err != nil {
		t.Fatal(err)
	}
}

func recordWindowCheck(t *testing.T, store *Store, id, name string) {
	t.Helper()
	if _, err := store.Update(id, func(task *Task) error {
		check := Check{Name: name, Candidate: "candidate", Passed: true, RecordedAt: taskStartTime}
		if name == "release_selection" || name == "release_preflight" {
			check.ReleaseCommit, check.ReleasePlanRevision = "merged-main", new(0)
		}
		return task.RecordCheck(check)
	}); err != nil {
		t.Fatal(err)
	}
}
