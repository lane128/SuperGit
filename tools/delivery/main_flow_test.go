package main

import (
	"bytes"
	"encoding/json"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
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
	} {
		runFlowCommand(t, state, 0, "check", "-id", "task-a", "-name", step.check, "-candidate", "new", "-passed")
		runFlowCommand(t, state, 0, "advance", "-id", "task-a", "-to", step.to)
	}
	runFlowCommand(t, state, 0, "check", "-id", "task-a", "-name", "code_review", "-candidate", "revised", "-passed")
	before = runFlowCommand(t, state, 0, "status", "-id", "task-a", "-json")
	if out := runFlowCommand(t, state, 1, "advance", "-id", "task-a", "-to", "acceptance", "-candidate", "revised"); !strings.Contains(out, `"tests"`) {
		t.Fatalf("review gate output = %q, want tests gate error", out)
	}
	if got := runFlowCommand(t, state, 0, "status", "-id", "task-a", "-json"); got != before {
		t.Fatal("failed review advance changed the persisted task")
	}
	runFlowCommand(t, state, 0, "check", "-id", "task-a", "-name", "tests", "-candidate", "revised", "-passed", "-evidence", "tests rerun after review fixes")
	runFlowCommand(t, state, 0, "advance", "-id", "task-a", "-to", "acceptance", "-candidate", "revised")
	runFlowCommand(t, state, 0, "check", "-id", "task-a", "-name", "acceptance", "-candidate", "revised", "-passed")
	runFlowCommand(t, state, 0, "advance", "-id", "task-a", "-to", "pull_request")
	runFlowCommand(t, state, 0, "check", "-id", "task-a", "-name", "remote_checks", "-candidate", "revised", "-passed")
	runFlowCommand(t, state, 1, "advance", "-id", "task-a", "-to", "merge")
	startFlowTask(t, state, "task-b")
	runFlowCommand(t, state, 2, "queue", "-id", "task-b", "-acquire", "-release")
	runFlowCommand(t, state, 0, "queue", "-id", "task-b", "-acquire")
	runFlowCommand(t, state, 1, "advance", "-id", "task-a", "-to", "merge")
	runFlowCommand(t, state, 0, "queue", "-id", "task-b", "-release")
	runFlowCommand(t, state, 0, "queue", "-id", "task-a", "-acquire")
	runFlowCommand(t, state, 0, "advance", "-id", "task-a", "-to", "merge")
	task := readFlowTask(t, state, "task-a")
	if task.Phase != delivery.PhaseMerge || task.Candidate != "revised" || task.Checks["remote_checks"].Phase != delivery.PhasePullRequest {
		t.Fatalf("persisted phase = %s, candidate = %q, check phase = %s", task.Phase, task.Candidate, task.Checks["remote_checks"].Phase)
	}
}

func TestRunStatusJSONIncludesPhaseAndAuthorization(t *testing.T) {
	state := t.TempDir()
	runFlowCommand(t, state, 2, "start", "-id", "task-z", "-request", "fixture", "-by", "maintainer", "-version-policy", "patch", "-release=false")
	startFlowTask(t, state, "task-z")
	startFlowTask(t, state, "task-a")
	detail := runFlowCommand(t, state, 0, "status", "-id", "task-z", "-json")
	var status taskStatusResponse
	if err := json.Unmarshal([]byte(detail), &status); err != nil {
		t.Fatalf("decode task status: %v", err)
	}
	if status.Phase != delivery.PhaseIntake || !status.Authorization.Release || status.Authorization.Revoked || status.BaseBranch != "main" {
		t.Fatalf("phase = %s, authorization = %+v, branch = %q", status.Phase, status.Authorization, status.BaseBranch)
	}
	items := readFlowTask(t, state, "task-z").Authorization.AcceptanceItems
	if !slices.Equal(items, []string{"first", "second"}) {
		t.Fatalf("acceptance items = %q", items)
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

func TestRunRejectsRepositoryStateRoot(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		inRepo     bool
		useEnv     bool
	}{
		{name: "current repository", path: ".", inRepo: true},
		{name: "repository from outside", path: "repo"},
		{name: "missing repository subdirectory", path: "repo/missing/state"},
		{name: "symlink into repository", path: "alias/missing/state"},
		{name: "environment default", path: "alias", useEnv: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outside := t.TempDir()
			repo := filepath.Join(outside, "repo")
			git := exec.CommandContext(t.Context(), "git", "init", "--quiet", repo)
			git.Env = []string{"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=" + os.DevNull}
			if out, err := git.CombinedOutput(); err != nil {
				t.Fatalf("git init: %v\n%s", err, out)
			}
			if err := os.Symlink(repo, filepath.Join(outside, "alias")); err != nil {
				t.Fatal(err)
			}
			t.Chdir(outside)
			state := filepath.Join(outside, tc.path)
			if tc.inRepo {
				t.Chdir(repo)
				state = tc.path
			}
			args := []string{"start", "-id", "task-a", "-request", "fixture", "-by", "maintainer", "-version-policy", "patch"}
			if tc.useEnv {
				t.Setenv("SUPERGIT_DELIVERY_STATE", state)
			} else {
				args = append(args, "-state", state)
			}
			var stdout, stderr bytes.Buffer
			if code := run(args, &stdout, &stderr); code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "repository") {
				t.Errorf("exit = %d, want 2, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
			}
			entries, err := os.ReadDir(repo)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || entries[0].Name() != ".git" {
				t.Errorf("checkout changed after rejecting -state: %v", entries)
			}
			startFlowTask(t, filepath.Join(outside, "safe-state"), "task-safe")
		})
	}
}

func TestRunRejectsRepositoryTaskDirectory(t *testing.T) {
	for _, tc := range []struct {
		name    string
		symlink bool
	}{
		{name: "nested checkout"},
		{name: "symlinked checkout", symlink: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			state := filepath.Join(root, "state")
			tasks := filepath.Join(state, "tasks")
			checkout := tasks
			if tc.symlink {
				checkout = filepath.Join(root, "checkout")
				if err := os.Mkdir(state, 0700); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.MkdirAll(filepath.Join(checkout, ".git"), 0700); err != nil {
				t.Fatal(err)
			}
			if tc.symlink {
				if err := os.Symlink(checkout, tasks); err != nil {
					t.Fatal(err)
				}
			}

			args := []string{"start", "-id", "task-a", "-request", "fixture", "-by", "maintainer", "-version-policy", "patch", "-state", state}
			var stdout, stderr bytes.Buffer
			if code := run(args, &stdout, &stderr); code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "repository") {
				t.Errorf("exit = %d, want 2, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
			}
			entries, err := os.ReadDir(state)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || entries[0].Name() != "tasks" {
				t.Errorf("state directory gained ledger files: %v", entries)
			}
			entries, err = os.ReadDir(checkout)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || entries[0].Name() != ".git" {
				t.Errorf("task checkout changed: %v", entries)
			}
		})
	}
}

func TestRunStatusJSONHasOnlyPublicFields(t *testing.T) {
	state := t.TempDir()
	startFlowTask(t, state, "task-a")
	runFlowCommand(t, state, 0, "check", "-id", "task-a", "-name", "intake", "-evidence", "old")
	runFlowCommand(t, state, 0, "check", "-id", "task-a", "-name", "intake", "-passed", "-evidence", "latest")
	runFlowCommand(t, state, 0, "revoke", "-id", "task-a", "-by", "maintainer")
	out := runFlowCommand(t, state, 0, "status", "-id", "task-a", "-json")
	status := assertStatusJSONFields(t, []byte(out), []string{
		"authorization", "base_branch", "blocked_reason", "candidate", "checks", "history_count", "id", "phase",
		"pr", "repository", "request", "status", "tag", "tier", "version",
	})
	auth := assertStatusJSONFields(t, status["authorization"], []string{"authorized_by", "release", "revoked"})
	if string(auth["authorized_by"]) != `"maintainer"` || string(auth["release"]) != "true" || string(auth["revoked"]) != "true" {
		t.Errorf("authorization values = %s", status["authorization"])
	}
	checks := assertStatusJSONFields(t, status["checks"], []string{"intake"})
	check := assertStatusJSONFields(t, checks["intake"], []string{"candidate", "evidence", "name", "passed", "phase", "recorded_at"})
	if string(check["evidence"]) != `"latest"` || string(check["passed"]) != "true" || string(status["history_count"]) != "4" {
		t.Errorf("latest check = %s, history count = %s", checks["intake"], status["history_count"])
	}
}

func assertStatusJSONFields(t *testing.T, data []byte, want []string) map[string]json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("decode status fields: %v", err)
	}
	if got := slices.Sorted(maps.Keys(fields)); !slices.Equal(got, want) {
		t.Fatalf("fields = %v, want %v", got, want)
	}
	return fields
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
	task, err := delivery.NewStore(state).Load(id)
	if err != nil {
		t.Fatal(err)
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
	if want != 0 {
		return stderr.String()
	}
	return stdout.String()
}
