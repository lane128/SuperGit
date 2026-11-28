package delivery

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

type Phase string

const (
	PhaseIntake           Phase = "intake"
	PhaseAnalysis         Phase = "analysis"
	PhaseDesign           Phase = "design"
	PhaseDesignReview     Phase = "design_review"
	PhaseImplementation   Phase = "implementation"
	PhaseCodeReview       Phase = "code_review"
	PhaseAcceptance       Phase = "acceptance"
	PhasePullRequest      Phase = "pull_request"
	PhaseMerge            Phase = "merge"
	PhaseReleasePreflight Phase = "release_preflight"
	PhaseTag              Phase = "tag"
	PhasePublish          Phase = "publish"
	PhaseVerify           Phase = "verify"
	PhaseComplete         Phase = "complete"
)

var phaseGates = [...]struct {
	phase Phase
	check string
}{
	{PhaseIntake, "intake"},
	{PhaseAnalysis, "analysis"},
	{PhaseDesign, "design"},
	{PhaseDesignReview, "design_review"},
	{PhaseImplementation, "tests"},
	{PhaseCodeReview, "code_review"},
	{PhaseAcceptance, "acceptance"},
	{PhasePullRequest, "remote_checks"},
	{PhaseMerge, "merge"},
	{PhaseReleasePreflight, "release_preflight"},
	{PhaseTag, "package"},
	{PhasePublish, "publish"},
	{PhaseVerify, "public_verify"},
	{PhaseComplete, ""},
}

type Status string

const (
	StatusActive    Status = "active"
	StatusBlocked   Status = "blocked"
	StatusCancelled Status = "cancelled"
)

type Authorization struct {
	AuthorizedBy    string
	Mode            string
	Autonomy        string
	Release         bool
	VersionPolicy   string
	Version         string
	Channel         string
	AcceptanceItems []string
	PolicyVersion   string
	GrantedAt       time.Time
	RevokedAt       *time.Time
	Revisions       []AuthorizationRevision
}

type AuthorizationRevision struct {
	By            string
	Reason        string
	Release       bool
	VersionPolicy string
	Version       string
	At            time.Time
}

type Event struct {
	At     time.Time
	Phase  Phase
	Action string
	Detail string
}

type Check struct {
	Name       string
	Phase      Phase
	Candidate  string
	Passed     bool
	Evidence   string
	RecordedAt time.Time
}

type Task struct {
	ID            string
	Request       string
	Repository    string
	BaseBranch    string
	Tier          string
	Phase         Phase
	Status        Status
	Authorization Authorization
	Candidate     string
	Checks        map[string]Check
	PR            int
	Version       string
	Tag           string
	BlockedReason string
	ReviewRounds  int
	CreatedAt     time.Time
	UpdatedAt     time.Time
	History       []Event
}

type StartRequest struct {
	Request         string
	Repository      string
	BaseBranch      string
	AuthorizedBy    string
	VersionPolicy   string
	Version         string
	Channel         string
	AcceptanceItems []string
	Release         bool
}

var taskIDPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

func validateTaskID(id string) error {
	if !taskIDPattern.MatchString(id) {
		return fmt.Errorf("invalid task ID %q (expected lowercase letters, digits, or hyphens)", id)
	}
	return nil
}

func NewTask(id string, req StartRequest, now time.Time) (Task, error) {
	if err := validateTaskID(id); err != nil {
		return Task{}, err
	}
	if req.AuthorizedBy == "" || req.Request == "" {
		return Task{}, errors.New("authorizer and request are required")
	}
	if !req.Release {
		return Task{}, errors.New("release authorization is required")
	}
	if err := validateVersionPolicy(req.VersionPolicy, req.Version); err != nil {
		return Task{}, err
	}

	task := Task{
		ID:         id,
		Request:    req.Request,
		Repository: req.Repository,
		BaseBranch: req.BaseBranch,
		Phase:      PhaseIntake,
		Status:     StatusActive,
		Authorization: Authorization{
			AuthorizedBy:    req.AuthorizedBy,
			Mode:            "auto",
			Autonomy:        "per_request",
			Release:         req.Release,
			VersionPolicy:   req.VersionPolicy,
			Version:         req.Version,
			Channel:         req.Channel,
			AcceptanceItems: slices.Clone(req.AcceptanceItems),
			PolicyVersion:   "1",
			GrantedAt:       now,
		},
		Checks: map[string]Check{
			"intake": {Name: "intake", Phase: PhaseIntake, Passed: true, Evidence: "request and release authorization recorded", RecordedAt: now},
		},
		CreatedAt: now,
	}
	task.recordEvent("start", req.AuthorizedBy, now)
	return task, nil
}

func (t *Task) RecordCheck(c Check) error {
	if err := t.ensureMutable(); err != nil {
		return err
	}
	if t.Checks == nil {
		t.Checks = make(map[string]Check)
	}
	c.Phase = t.Phase
	t.Checks[c.Name] = c
	detail := fmt.Sprintf("%s: passed=%t candidate=%q", c.Name, c.Passed, c.Candidate)
	if c.Evidence != "" {
		detail += " evidence=" + c.Evidence
	}
	t.recordEvent("check", detail, c.RecordedAt)
	return nil
}

func (t *Task) SetCandidate(candidate string, now time.Time) error {
	if err := t.ensureMutable(); err != nil {
		return err
	}
	t.Candidate = candidate
	t.recordEvent("candidate", candidate, now)
	return nil
}

func (t *Task) SetTier(tier string, now time.Time) error {
	if err := t.ensureMutable(); err != nil {
		return err
	}
	t.Tier = tier
	t.recordEvent("tier", tier, now)
	return nil
}

func (t *Task) SetPR(number int, now time.Time) error {
	if err := t.ensureMutable(); err != nil {
		return err
	}
	t.PR = number
	t.recordEvent("pr", fmt.Sprint(number), now)
	return nil
}

func (t *Task) Advance(to Phase, now time.Time) error {
	if err := t.ensureMutable(); err != nil {
		return err
	}
	if t.Status == StatusBlocked {
		return fmt.Errorf("task %q is blocked: %s", t.ID, t.BlockedReason)
	}

	from := phaseIndex(t.Phase)
	skipDesign := t.Phase == PhaseAnalysis && to == PhaseImplementation
	if !skipDesign && (from < 0 || from+1 >= len(phaseGates) || phaseGates[from+1].phase != to) {
		return fmt.Errorf("cannot advance from %q to %q", t.Phase, to)
	}
	name := phaseGates[from].check
	check := t.Checks[name]
	if !check.Passed {
		return fmt.Errorf("phase %q requires a passing %q check", t.Phase, name)
	}
	if check.Phase != t.Phase {
		return fmt.Errorf("check %q must be recorded during phase %q", name, t.Phase)
	}
	if check.Candidate != t.Candidate || (check.Candidate == "" && t.Phase != PhaseIntake && t.Phase != PhaseAnalysis) {
		return fmt.Errorf("check %q must match the current candidate for phase %q", name, t.Phase)
	}

	detail := fmt.Sprintf("%s → %s", t.Phase, to)
	if skipDesign {
		design := t.Checks["design"]
		if design.Phase != PhaseAnalysis || !strings.HasPrefix(design.Evidence, "skipped:") {
			return errors.New("skipping design requires a design check recorded during analysis with evidence starting with skipped:")
		}
		detail += "; design " + design.Evidence
	}
	if isReleasePhase(to) && !t.hasReleaseAuthorization() {
		return fmt.Errorf("phase %q requires active release authorization", to)
	}

	t.Phase = to
	t.recordEvent("advance", detail, now)
	return nil
}

func (t *Task) Block(reason string, now time.Time) error {
	if err := t.ensureMutable(); err != nil {
		return err
	}
	t.Status = StatusBlocked
	t.BlockedReason = reason
	t.recordEvent("block", reason, now)
	return nil
}

func (t *Task) Resume(now time.Time) error {
	if err := t.ensureMutable(); err != nil {
		return err
	}
	if isReleasePhase(t.Phase) && !t.hasReleaseAuthorization() {
		return fmt.Errorf("phase %q requires active release authorization", t.Phase)
	}
	reason := t.BlockedReason
	t.Status = StatusActive
	t.BlockedReason = ""
	t.recordEvent("resume", reason, now)
	return nil
}

func (t *Task) Revoke(by string, now time.Time) error {
	if err := t.ensureMutable(); err != nil {
		return err
	}
	t.Authorization.RevokedAt = &now
	if isReleasePhase(t.Phase) {
		t.Status = StatusBlocked
		t.BlockedReason = "release authorization revoked"
	}
	t.recordEvent("revoke", by, now)
	return nil
}

func (t *Task) AuthorizeRevision(rev AuthorizationRevision, now time.Time) error {
	if err := t.ensureMutable(); err != nil {
		return err
	}
	if strings.TrimSpace(rev.By) == "" || strings.TrimSpace(rev.Reason) == "" {
		return errors.New("authorization revision requires an author and a reason")
	}
	if err := validateVersionPolicy(rev.VersionPolicy, rev.Version); err != nil {
		return err
	}
	if rev.At.IsZero() {
		rev.At = now
	}
	auth := &t.Authorization
	auth.Revisions = append(auth.Revisions, rev)
	auth.Release = rev.Release
	auth.VersionPolicy = rev.VersionPolicy
	auth.Version = rev.Version
	auth.RevokedAt = nil
	t.Status = StatusActive
	t.BlockedReason = ""
	t.recordEvent("authorize", fmt.Sprintf("%s: %s", rev.By, rev.Reason), now)
	return nil
}

func validateVersionPolicy(policy, version string) error {
	switch policy {
	case "patch", "minor":
	case "exact":
		if version == "" {
			return errors.New("exact version policy requires a version")
		}
	default:
		return fmt.Errorf("invalid version policy %q", policy)
	}
	return nil
}

func phaseIndex(phase Phase) int {
	for i, gate := range phaseGates {
		if gate.phase == phase {
			return i
		}
	}
	return -1
}

func isReleasePhase(phase Phase) bool {
	return phaseIndex(phase) >= phaseIndex(PhaseMerge)
}

func (t *Task) hasReleaseAuthorization() bool {
	return t.Authorization.Release && t.Authorization.RevokedAt == nil
}

func (t *Task) ensureMutable() error {
	if t.Status == StatusCancelled {
		return fmt.Errorf("task %q is cancelled", t.ID)
	}
	return nil
}

func (t *Task) recordEvent(action, detail string, now time.Time) {
	t.UpdatedAt = now
	t.History = append(t.History, Event{At: now, Phase: t.Phase, Action: action, Detail: detail})
}
