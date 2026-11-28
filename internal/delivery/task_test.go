package delivery

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

var taskStartTime = time.Date(2026, time.November, 28, 10, 0, 0, 0, time.UTC)

func taskStartRequest() StartRequest {
	return StartRequest{
		Request:         "Add a keyboard shortcut",
		Repository:      "owner/repo",
		BaseBranch:      "main",
		AuthorizedBy:    "maintainer",
		Release:         true,
		VersionPolicy:   "patch",
		Channel:         "darwin-arm64",
		AcceptanceItems: []string{"Confirm shortcut works"},
	}
}

func newLedgerTask(t *testing.T, id string, now time.Time) Task {
	t.Helper()
	task, err := NewTask(id, taskStartRequest(), now)
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func recordLedgerCheck(t *testing.T, task *Task, c Check) {
	t.Helper()
	if err := task.RecordCheck(c); err != nil {
		t.Fatal(err)
	}
}

func advanceLedgerTask(t *testing.T, task *Task, phase Phase, now time.Time) {
	t.Helper()
	if err := task.Advance(phase, now); err != nil {
		t.Fatalf("advance to %q: %v", phase, err)
	}
}

func TestAdvanceInitialTaskAndRequestValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*StartRequest)
	}{
		{"missing authorizer", func(r *StartRequest) { r.AuthorizedBy = "" }},
		{"missing request", func(r *StartRequest) { r.Request = "" }},
		{"release disabled", func(r *StartRequest) { r.Release = false }},
		{"missing version policy", func(r *StartRequest) { r.VersionPolicy = "" }},
		{"unknown version policy", func(r *StartRequest) { r.VersionPolicy = "major" }},
		{"missing exact version", func(r *StartRequest) { r.VersionPolicy = "exact" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := taskStartRequest()
			tc.change(&req)
			if _, err := NewTask("valid-id", req, taskStartTime); err == nil {
				t.Fatal("NewTask accepted an invalid request")
			}
		})
	}
	if _, err := NewTask("../escape", taskStartRequest(), taskStartTime); err == nil {
		t.Fatal("NewTask accepted an invalid ID")
	}
	for _, policy := range []string{"patch", "minor", "exact"} {
		t.Run(policy, func(t *testing.T) {
			req := taskStartRequest()
			req.VersionPolicy = policy
			if policy == "exact" {
				req.Version = "1.2.3"
			}
			task, err := NewTask("valid-id", req, taskStartTime)
			if err != nil {
				t.Fatal(err)
			}
			if task.ID != "valid-id" || task.Request != req.Request || task.Repository != req.Repository || task.BaseBranch != req.BaseBranch || task.Phase != PhaseIntake || task.Status != StatusActive {
				t.Fatalf("initial task = %+v", task)
			}
			auth := task.Authorization
			req.AcceptanceItems[0] = "edited after creation"
			if auth.AuthorizedBy != req.AuthorizedBy || auth.Mode != "auto" || auth.Autonomy != "per_request" || !auth.Release || auth.VersionPolicy != policy || auth.Version != req.Version || auth.Channel != req.Channel || len(auth.AcceptanceItems) != 1 || auth.AcceptanceItems[0] != "Confirm shortcut works" || !auth.GrantedAt.Equal(taskStartTime) {
				t.Fatalf("initial authorization = %+v", auth)
			}
			if !task.CreatedAt.Equal(taskStartTime) || !task.UpdatedAt.Equal(taskStartTime) || len(task.History) != 1 || task.History[0].Phase != PhaseIntake || task.History[0].Action == "" || !task.History[0].At.Equal(taskStartTime) {
				t.Fatalf("initial audit history = %+v", task)
			}
			advanceLedgerTask(t, &task, PhaseAnalysis, taskStartTime.Add(time.Minute))
		})
	}
}

func TestAdvanceRejectsOutOfOrderPhase(t *testing.T) {
	task := newLedgerTask(t, "task", taskStartTime)
	later := taskStartTime.Add(time.Minute)
	for _, phase := range []Phase{PhaseIntake, PhaseDesign, PhaseMerge, "unknown"} {
		if err := task.Advance(phase, later); err == nil {
			t.Errorf("advance from intake to %q succeeded", phase)
		}
	}
	if task.Phase != PhaseIntake || len(task.History) != 1 || !task.UpdatedAt.Equal(taskStartTime) {
		t.Fatalf("rejected transitions mutated the task: %+v", task)
	}
	advanceLedgerTask(t, &task, PhaseAnalysis, later)
	if err := task.Advance(PhaseIntake, later); err == nil {
		t.Fatal("advanced backwards to intake")
	}
}

func TestAdvanceRequiresPassedCheckForCurrentCandidate(t *testing.T) {
	task := newLedgerTask(t, "task", taskStartTime)
	later := taskStartTime.Add(time.Minute)
	advanceLedgerTask(t, &task, PhaseAnalysis, later)
	if err := task.SetCandidate("candidate-a", later); err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		check string
		phase Phase
	}{
		{"analysis", PhaseDesign},
		{"design", PhaseDesignReview},
		{"design_review", PhaseImplementation},
		{"tests", PhaseCodeReview},
		{"code_review", PhaseAcceptance},
		{"acceptance", PhasePullRequest},
		{"remote_checks", PhaseMerge},
		{"merge", PhaseReleasePreflight},
		{"release_preflight", PhaseTag},
		{"package", PhasePublish},
		{"publish", PhaseVerify},
		{"public_verify", PhaseComplete},
	} {
		t.Run(step.check, func(t *testing.T) {
			if err := task.Advance(step.phase, later); err == nil {
				t.Fatalf("advance to %q succeeded without %q", step.phase, step.check)
			}
			check := Check{Name: step.check, Candidate: "candidate-a", RecordedAt: later}
			recordLedgerCheck(t, &task, check)
			if err := task.Advance(step.phase, later); err == nil {
				t.Fatalf("advance to %q succeeded with a failed check", step.phase)
			}
			check.Passed = true
			recordLedgerCheck(t, &task, check)
			advanceLedgerTask(t, &task, step.phase, later)
		})
	}
	if task.Phase != PhaseComplete || task.Advance(PhaseComplete, later) == nil {
		t.Fatalf("complete did not behave as the terminal phase: %q", task.Phase)
	}
}

func TestAdvanceRejectsCheckFromOlderCandidate(t *testing.T) {
	task := newLedgerTask(t, "task", taskStartTime)
	later := taskStartTime.Add(time.Minute)
	advanceLedgerTask(t, &task, PhaseAnalysis, later)
	recordLedgerCheck(t, &task, Check{Name: "analysis", Passed: true, RecordedAt: later})
	advanceLedgerTask(t, &task, PhaseDesign, later)
	recordLedgerCheck(t, &task, Check{Name: "design", Passed: true, RecordedAt: later})
	if err := task.Advance(PhaseDesignReview, later); err == nil {
		t.Fatal("advanced from design with an empty candidate on both task and check")
	}
	if err := task.SetCandidate("old", later); err != nil {
		t.Fatal(err)
	}
	recordLedgerCheck(t, &task, Check{Name: "design", Candidate: "old", Passed: true, RecordedAt: later})
	if err := task.SetCandidate("new", later); err != nil {
		t.Fatal(err)
	}
	if err := task.Advance(PhaseDesignReview, later); err == nil {
		t.Fatal("advanced with a check for the previous candidate")
	}
	recordLedgerCheck(t, &task, Check{Name: "design", Candidate: "new", Passed: true, RecordedAt: later})
	advanceLedgerTask(t, &task, PhaseDesignReview, later)
}

func TestAdvanceRejectsCheckRecordedBeforeItsPhase(t *testing.T) {
	task := newLedgerTask(t, "task", taskStartTime)
	task.Phase = PhaseImplementation
	task.Candidate = "candidate-a"
	later := taskStartTime.Add(time.Minute)
	check := Check{Name: "public_verify", Phase: PhaseVerify, Candidate: task.Candidate, Passed: true, RecordedAt: later}
	recordLedgerCheck(t, &task, check)

	// Isolate the verify gate from the intervening transitions.
	task.Phase = PhaseVerify
	if err := task.Advance(PhaseComplete, later); err == nil {
		t.Fatal("completed using a public verification recorded during implementation")
	}
	check.Phase = PhaseImplementation
	recordLedgerCheck(t, &task, check)
	advanceLedgerTask(t, &task, PhaseComplete, later)
}

func TestSkipDesignRejectsEvidenceRecordedBeforeAnalysis(t *testing.T) {
	task := newLedgerTask(t, "task", taskStartTime)
	later := taskStartTime.Add(time.Minute)
	skip := Check{Name: "design", Evidence: "skipped: backend only", RecordedAt: later}
	recordLedgerCheck(t, &task, skip)
	advanceLedgerTask(t, &task, PhaseAnalysis, later)
	recordLedgerCheck(t, &task, Check{Name: "analysis", Passed: true, RecordedAt: later})
	if err := task.Advance(PhaseImplementation, later); err == nil {
		t.Fatal("skipped design using evidence recorded during intake")
	}
	recordLedgerCheck(t, &task, skip)
	advanceLedgerTask(t, &task, PhaseImplementation, later)
}

func TestSkipDesignRecordsReason(t *testing.T) {
	task := newLedgerTask(t, "task", taskStartTime)
	later := taskStartTime.Add(time.Minute)
	advanceLedgerTask(t, &task, PhaseAnalysis, later)
	recordLedgerCheck(t, &task, Check{Name: "analysis", Passed: true, RecordedAt: later})
	if err := task.Advance(PhaseImplementation, later); err == nil {
		t.Fatal("skipped design without a design check")
	}
	recordLedgerCheck(t, &task, Check{Name: "design", Passed: true, Evidence: "skipped without a prefix", RecordedAt: later})
	if err := task.Advance(PhaseImplementation, later); err == nil {
		t.Fatal("skipped design without the skipped: prefix")
	}
	recordLedgerCheck(t, &task, Check{Name: "design", Evidence: "skipped: backend only", RecordedAt: later})
	advanceLedgerTask(t, &task, PhaseImplementation, later)
	event := task.History[len(task.History)-1]
	if event.Phase != PhaseImplementation || !strings.Contains(event.Detail, "backend only") || !event.At.Equal(later) || !task.UpdatedAt.Equal(later) {
		t.Fatalf("skip history = %+v, updated at %v", event, task.UpdatedAt)
	}
}

func TestRevokedAuthorizationBlocksMerge(t *testing.T) {
	task := newLedgerTask(t, "task", taskStartTime)
	later := taskStartTime.Add(time.Minute)
	task.Phase = PhasePullRequest
	if err := task.SetCandidate("candidate-a", later); err != nil {
		t.Fatal(err)
	}
	recordLedgerCheck(t, &task, Check{Name: "remote_checks", Candidate: "candidate-a", Passed: true, RecordedAt: later})
	if err := task.Revoke("maintainer", later); err != nil {
		t.Fatal(err)
	}
	if task.Status != StatusActive || task.Authorization.RevokedAt == nil || !task.Authorization.RevokedAt.Equal(later) || task.Advance(PhaseMerge, later) == nil {
		t.Fatalf("revoked authorization did not prevent merge: %+v", task)
	}
	if err := task.AuthorizeRevision(AuthorizationRevision{By: "maintainer", Reason: "pause release", VersionPolicy: "patch"}, later); err != nil {
		t.Fatal(err)
	}
	if err := task.Advance(PhaseMerge, later); err == nil {
		t.Fatal("advanced to merge with release disabled")
	}
	if err := task.AuthorizeRevision(AuthorizationRevision{By: "maintainer", Reason: "resume release", Release: true, VersionPolicy: "patch"}, later); err != nil {
		t.Fatal(err)
	}
	advanceLedgerTask(t, &task, PhaseMerge, later)
	for _, phase := range []Phase{PhaseMerge, PhaseReleasePreflight, PhaseTag, PhasePublish, PhaseVerify, PhaseComplete} {
		t.Run(string(phase), func(t *testing.T) {
			at := later.Add(time.Minute)
			lateTask := task
			lateTask.Phase = phase
			if err := lateTask.Revoke("maintainer", at); err != nil {
				t.Fatal(err)
			}
			event := lateTask.History[len(lateTask.History)-1]
			if lateTask.Status != StatusBlocked || lateTask.BlockedReason == "" || !lateTask.UpdatedAt.Equal(at) || !strings.Contains(event.Detail, "maintainer") {
				t.Fatalf("revocation failed to block and record the task: %+v", lateTask)
			}
			if err := lateTask.Resume(at); err == nil {
				t.Fatal("resumed a release phase with revoked authorization")
			}
		})
	}
}

func TestAuthorizeRevisionRejectsInvalidDetailsWithoutClearingRevocation(t *testing.T) {
	for _, release := range []bool{true, false} {
		mode := "grant release"
		if !release {
			mode = "disable release"
		}
		t.Run(mode, func(t *testing.T) {
			for _, tc := range []struct {
				name   string
				change func(*AuthorizationRevision)
			}{
				{"missing author", func(r *AuthorizationRevision) { r.By = "" }},
				{"missing reason", func(r *AuthorizationRevision) { r.Reason = "" }},
				{"invalid policy", func(r *AuthorizationRevision) { r.VersionPolicy = "major" }},
				{"missing exact version", func(r *AuthorizationRevision) { r.VersionPolicy = "exact" }},
			} {
				t.Run(tc.name, func(t *testing.T) {
					task := newLedgerTask(t, "task", taskStartTime)
					task.Phase = PhaseMerge
					at := taskStartTime.Add(time.Minute)
					if err := task.Revoke("maintainer", at); err != nil {
						t.Fatal(err)
					}
					before := task
					rev := AuthorizationRevision{By: "maintainer", Reason: "change release policy", Release: release, VersionPolicy: "patch"}
					tc.change(&rev)
					if err := task.AuthorizeRevision(rev, at.Add(time.Minute)); err == nil {
						t.Fatal("invalid revision was accepted")
					}
					if !reflect.DeepEqual(task, before) {
						t.Fatalf("invalid revision changed the revoked task: %+v", task)
					}
				})
			}
		})
	}
}

func TestAuthorizeRevisionClearsRevocation(t *testing.T) {
	task := newLedgerTask(t, "task", taskStartTime)
	task.Phase = PhaseMerge
	revokedAt := taskStartTime.Add(time.Minute)
	if err := task.Revoke("maintainer", revokedAt); err != nil {
		t.Fatal(err)
	}
	at := revokedAt.Add(time.Minute)
	rev := AuthorizationRevision{By: "maintainer", Reason: "pin next release", Release: true, VersionPolicy: "exact", Version: "1.2.3", At: at}
	if err := task.AuthorizeRevision(rev, at); err != nil {
		t.Fatal(err)
	}
	auth := task.Authorization
	if auth.RevokedAt != nil || !auth.Release || auth.VersionPolicy != "exact" || auth.Version != "1.2.3" || len(auth.Revisions) != 1 || auth.Revisions[0] != rev || !auth.GrantedAt.Equal(taskStartTime) {
		t.Fatalf("revised authorization = %+v", auth)
	}
	event := task.History[len(task.History)-1]
	if task.Status != StatusActive || task.BlockedReason != "" || len(task.History) != 3 || !task.UpdatedAt.Equal(at) || !strings.Contains(event.Detail, rev.Reason) {
		t.Fatalf("task did not resume with a recorded revision: %+v", task)
	}
}

func TestAdvanceAndOtherMutatorsRespectTaskStatus(t *testing.T) {
	task := newLedgerTask(t, "task", taskStartTime)
	for i, step := range []struct {
		phase  Phase
		change func(time.Time) error
	}{
		{PhaseIntake, func(at time.Time) error { return task.SetTier("T3", at) }},
		{PhaseIntake, func(at time.Time) error { return task.SetPR(42, at) }},
		{PhaseIntake, func(at time.Time) error { return task.Block("awaiting input", at) }},
		{PhaseIntake, func(at time.Time) error { return task.Resume(at) }},
		{PhaseIntake, func(at time.Time) error { return task.SetCandidate("candidate-a", at) }},
		{PhaseIntake, func(at time.Time) error {
			return task.RecordCheck(Check{Name: "intake", Candidate: "candidate-a", Passed: true, RecordedAt: at})
		}},
		{PhaseAnalysis, func(at time.Time) error { return task.Advance(PhaseAnalysis, at) }},
	} {
		at := taskStartTime.Add(time.Duration(i+1) * time.Minute)
		previousEvents := len(task.History)
		if err := step.change(at); err != nil {
			t.Fatal(err)
		}
		if len(task.History) != previousEvents+1 {
			t.Fatalf("mutation %d did not append one event", i)
		}
		event := task.History[len(task.History)-1]
		if event.Phase != step.phase || event.Action == "" || !event.At.Equal(at) || !task.UpdatedAt.Equal(at) {
			t.Fatalf("mutation %d history = %+v, updated at %v", i, event, task.UpdatedAt)
		}
		if task.Status == StatusBlocked && (task.BlockedReason != "awaiting input" || task.Advance(PhaseAnalysis, at) == nil) {
			t.Fatalf("blocked task advanced or lost its reason: %+v", task)
		}
	}
	if task.Tier != "T3" || task.PR != 42 || task.Status != StatusActive || task.BlockedReason != "" {
		t.Fatalf("task metadata after resume = %+v", task)
	}

	task.Status = StatusCancelled
	at := taskStartTime.Add(time.Hour)
	previousEvents, previousUpdate := len(task.History), task.UpdatedAt
	for _, change := range []func() error{
		func() error { return task.RecordCheck(Check{Name: "analysis", RecordedAt: at}) },
		func() error { return task.SetCandidate("another", at) },
		func() error { return task.SetTier("T2", at) },
		func() error { return task.SetPR(43, at) },
		func() error { return task.Advance(PhaseDesign, at) },
		func() error { return task.Block("reason", at) },
		func() error { return task.Resume(at) },
		func() error { return task.Revoke("maintainer", at) },
		func() error {
			return task.AuthorizeRevision(AuthorizationRevision{Release: true, VersionPolicy: "patch"}, at)
		},
	} {
		if err := change(); err == nil {
			t.Error("mutation of a cancelled task succeeded")
		}
	}
	if len(task.History) != previousEvents || !task.UpdatedAt.Equal(previousUpdate) || task.Status != StatusCancelled {
		t.Fatalf("rejected mutations changed the task: %+v", task)
	}
}
