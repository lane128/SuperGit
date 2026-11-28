package main

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"supergit/internal/delivery"
)

func TestRunTaskFlowPersistsAcrossInvocations(t *testing.T) {
	state := t.TempDir()
	startFlowTask(t, state, "task-a")
	if out := runFlowCommand(t, state, 0, "advance", "-id", "task-a", "-to", "analysis"); !strings.Contains(out, "analysis") {
		t.Fatalf("advance output = %q", out)
	}
	runFlowCommand(t, state, 0, "block", "-id", "task-a", "-reason", "waiting on evidence")
	blocked := readFlowTask(t, state, "task-a")
	if blocked.Status != delivery.StatusBlocked || blocked.BlockedReason != "waiting on evidence" {
		t.Fatalf("blocked task = %s, %q", blocked.Status, blocked.BlockedReason)
	}
	runFlowCommand(t, state, 0, "resume", "-id", "task-a")
	runFlowCommand(t, state, 2, "check", "-id", "task-a", "-name", "analysis", "-phase", "analysis")
	runFlowCommand(t, state, 0, "check", "-id", "task-a", "-name", "analysis", "-candidate", "new", "-passed", "-evidence", "analysis reviewed")
	before := runFlowCommand(t, state, 0, "status", "-id", "task-a", "-json")
	runFlowCommand(t, state, 1, "advance", "-id", "task-a", "-to", "design", "-candidate", "wrong")
	if got := runFlowCommand(t, state, 0, "status", "-id", "task-a", "-json"); got != before {
		t.Fatal("failed advance changed the persisted task")
	}
	runFlowCommand(t, state, 0, "advance", "-id", "task-a", "-to", "design", "-candidate", "new")
	for _, step := range []struct{ check, to string }{
		{"design", "design_review"}, {"design_review", "implementation"}, {"tests", "code_review"},
		{"code_review", "acceptance"}, {"acceptance", "pull_request"},
	} {
		runFlowCommand(t, state, 0, "check", "-id", "task-a", "-name", step.check, "-candidate", "new", "-passed")
		runFlowCommand(t, state, 0, "advance", "-id", "task-a", "-to", step.to)
	}
	runFlowCommand(t, state, 0, "check", "-id", "task-a", "-name", "remote_checks", "-candidate", "new", "-passed")
	runFlowCommand(t, state, 1, "advance", "-id", "task-a", "-to", "merge")
	startFlowTask(t, state, "task-b")
	runFlowCommand(t, state, 2, "queue", "-id", "task-b", "-acquire", "-release")
	runFlowCommand(t, state, 0, "queue", "-id", "task-b", "-acquire")
	runFlowCommand(t, state, 1, "advance", "-id", "task-a", "-to", "merge")
	runFlowCommand(t, state, 0, "queue", "-id", "task-b", "-release")
	runFlowCommand(t, state, 0, "queue", "-id", "task-a", "-acquire")
	runFlowCommand(t, state, 0, "advance", "-id", "task-a", "-to", "merge")
	task := readFlowTask(t, state, "task-a")
	if task.Phase != delivery.PhaseMerge || task.Candidate != "new" || task.Checks["remote_checks"].Phase != delivery.PhasePullRequest {
		t.Fatalf("persisted phase = %s, candidate = %q, check phase = %s", task.Phase, task.Candidate, task.Checks["remote_checks"].Phase)
	}
}

func TestRunStatusJSONIncludesPhaseAndAuthorization(t *testing.T) {
	state := t.TempDir()
	runFlowCommand(t, state, 2, "start", "-id", "task-z", "-request", "fixture", "-by", "maintainer", "-version-policy", "patch", "-release=false")
	startFlowTask(t, state, "task-z")
	startFlowTask(t, state, "task-a")
	task := readFlowTask(t, state, "task-z")
	if task.Phase != delivery.PhaseIntake || !task.Authorization.Release || task.Authorization.RevokedAt != nil || task.BaseBranch != "main" {
		t.Fatalf("phase = %s, authorization = %+v, branch = %q", task.Phase, task.Authorization, task.BaseBranch)
	}
	if !slices.Equal(task.Authorization.AcceptanceItems, []string{"first", "second"}) {
		t.Fatalf("acceptance items = %q", task.Authorization.AcceptanceItems)
	}
	out := runFlowCommand(t, state, 0, "status", "-json")
	var summaries []struct {
		ID, Phase, Status string
	}
	if err := json.Unmarshal([]byte(out), &summaries); err != nil {
		t.Fatalf("decode status summaries: %v", err)
	}
	if len(summaries) != 2 || summaries[0].ID != "task-z" || summaries[1].ID != "task-a" || summaries[0].Phase != "intake" || summaries[0].Status != "active" {
		t.Fatalf("summaries = %+v", summaries)
	}
}

func TestRunRevokeThenAuthorizeRevision(t *testing.T) {
	state := t.TempDir()
	startFlowTask(t, state, "task-a")
	runFlowCommand(t, state, 0, "revoke", "-id", "task-a", "-by", "maintainer")
	if task := readFlowTask(t, state, "task-a"); task.Authorization.RevokedAt == nil {
		t.Fatal("revocation was not persisted")
	}
	runFlowCommand(t, state, 0, "authorize", "-id", "task-a", "-by", "reviewer", "-reason", "approval restored", "-release", "-version-policy", "exact", "-version", "9.9.9")
	auth := readFlowTask(t, state, "task-a").Authorization
	if auth.RevokedAt != nil || !auth.Release || auth.VersionPolicy != "exact" || auth.Version != "9.9.9" || len(auth.Revisions) != 1 {
		t.Fatalf("updated authorization = %+v", auth)
	}
	if rev := auth.Revisions[0]; rev.By != "reviewer" || rev.Reason != "approval restored" || rev.At.IsZero() {
		t.Fatalf("authorization revision = %+v", rev)
	}
}

func startFlowTask(t *testing.T, state, id string) {
	t.Helper()
	out := runFlowCommand(t, state, 0, "start", "-id", id, "-request", "fixture", "-repo", "local/repo", "-by", "maintainer", "-version-policy", "patch", "-channel", "darwin-arm64", "-acceptance", " first, , second ")
	if !strings.Contains(out, id) || !strings.Contains(out, "intake") {
		t.Fatalf("start output = %q", out)
	}
}

func readFlowTask(t *testing.T, state, id string) delivery.Task {
	t.Helper()
	out := runFlowCommand(t, state, 0, "status", "-id", id, "-json")
	var task delivery.Task
	if err := json.Unmarshal([]byte(out), &task); err != nil {
		t.Fatalf("decode task status: %v", err)
	}
	return task
}

func runFlowCommand(t *testing.T, state string, want int, args ...string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(append(args, "-state", state), &stdout, &stderr)
	if code != want || (want != 0 && (stdout.Len() != 0 || stderr.Len() == 0)) || (want == 0 && stderr.Len() != 0) {
		t.Fatalf("run %q: exit = %d, want %d, stdout = %q, stderr = %q", args, code, want, stdout.String(), stderr.String())
	}
	return stdout.String()
}
