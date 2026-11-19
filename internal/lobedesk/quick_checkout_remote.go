package lobedesk

import (
	"context"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

type quickCheckoutCandidate struct {
	gitrepo.Branch
	kind string // local, remote, tracker, create
}

func (a *app) quickCheckoutCandidates() []quickCheckoutCandidate {
	var result []quickCheckoutCandidate
	add := func(branches []gitrepo.Branch, kind, query string) {
		for _, b := range filterBranchesAt(branches, query) {
			result = append(result, quickCheckoutCandidate{b, kind})
		}
	}
	if a.quickCheckout.step == "targets" {
		var trackers []gitrepo.Branch
		for _, b := range a.branches {
			if b.Upstream == a.quickCheckout.remoteName {
				trackers = append(trackers, b)
			}
		}
		add(trackers, "tracker", a.quickCheckout.targetQuery)
		result = append(result, quickCheckoutCandidate{gitrepo.Branch{Name: a.t("quickCheckoutCreate"), SHA: a.quickCheckout.remoteSHA}, "create"})
	} else {
		add(a.branches, "local", a.quickCheckout.query)
		// RemoteBranches already excludes symbolic refs. A literal branch such
		// as feature/HEAD is still a valid remote destination.
		add(a.remoteBranches, "remote", a.quickCheckout.query)
	}
	return result
}

func (a *app) quickCheckoutRemoteReason(name, sha string) string {
	if a.remoteBranchesErr != "" {
		return a.t("quickCheckoutReadError")
	}
	found := false
	for _, b := range a.remoteBranches {
		if b.Name == name && b.SHA == sha {
			found = true
			break
		}
	}
	if !found {
		return a.t("quickCheckoutTargetChanged")
	}
	// A remote source being tracked in another worktree does not occupy a new
	// local destination. Reuse only the repository-level readiness checks.
	return a.quickCheckoutUnavailable(gitrepo.Branch{Name: "refs/remotes/" + name, SHA: sha})
}

func (a *app) quickCheckoutCandidateReason(b quickCheckoutCandidate) string {
	switch b.kind {
	case "remote":
		return a.quickCheckoutRemoteReason(b.Name, b.SHA)
	case "create":
		return a.quickCheckoutRemoteReason(a.quickCheckout.remoteName, a.quickCheckout.remoteSHA)
	case "tracker":
		if reason := a.quickCheckoutRemoteReason(a.quickCheckout.remoteName, a.quickCheckout.remoteSHA); reason != "" {
			return reason
		}
	}
	return a.quickCheckoutUnavailable(b.Branch)
}

func (a *app) chooseQuickCheckout(candidate quickCheckoutCandidate, generation uint64) {
	a.discardStaleQuickCheckout()
	if !a.quickCheckout.open || a.quickCheckout.generation != generation {
		return
	}
	// Re-find the exact displayed destination; never substitute the first tracker.
	found := false
	for _, b := range a.quickCheckoutCandidates() {
		if b.kind == candidate.kind && b.Name == candidate.Name && b.SHA == candidate.SHA && b.Upstream == candidate.Upstream && a.quickCheckoutCandidateReason(b) == "" {
			found = true
			break
		}
	}
	if !found {
		return
	}
	switch candidate.kind {
	case "remote":
		a.quickCheckout.remoteName, a.quickCheckout.remoteSHA = candidate.Name, candidate.SHA
		a.quickCheckout.savedCursor = a.quickCheckout.cursor
		a.quickCheckout.step = "name"
		for _, b := range a.branches {
			if b.Upstream == candidate.Name {
				a.quickCheckout.step = "targets"
				break
			}
		}
		_, a.quickCheckout.localName = a.remoteBranchParts(candidate.Name)
		a.quickCheckout.nameFocus = true
		a.quickCheckout.searchFocus = true
		a.quickCheckout.cursorPlaced = false
	case "create":
		a.quickCheckout.targetCursor = a.quickCheckout.cursor
		a.quickCheckout.step = "name"
		a.quickCheckout.nameFocus = true
	case "tracker":
		name, sha, upstream := candidate.Name, candidate.SHA, candidate.Upstream
		remoteSHA := a.quickCheckout.remoteSHA
		a.quickCheckout = quickCheckoutState{}
		if a.quickCheckoutReady() {
			a.performAs(busySwitch, a.t("switch"), a.t("switchSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
				remoteTip, err := repo.ResolveRef(ctx, "refs/remotes/"+upstream)
				if err != nil || remoteTip != remoteSHA {
					return gitrepo.ErrSwitchTargetChanged
				}
				branches, err := repo.Branches(ctx)
				if err != nil {
					return err
				}
				for _, b := range branches {
					if b.Name == name && b.SHA == sha && b.Upstream == upstream {
						return repo.SwitchBranchChecked(ctx, name, sha)
					}
				}
				return gitrepo.ErrSwitchTargetChanged
			})
		}
	default:
		a.runQuickCheckout(candidate.Name, candidate.SHA, generation)
	}
	a.invalidate()
}

func (a *app) quickCheckoutBack() {
	s := &a.quickCheckout
	s.searchFocus = true
	if s.step == "name" {
		for _, b := range a.branches {
			if b.Upstream == s.remoteName {
				s.step = "targets"
				s.cursor = s.targetCursor
				s.cursorPlaced = true
				s.cursorQuery = "targets" + s.targetQuery
				s.reveal = true
				a.invalidate()
				return
			}
		}
	}
	s.step = ""
	s.cursor = s.savedCursor
	s.cursorPlaced = true
	s.cursorQuery = s.query
	s.reveal = true
	a.invalidate()
}

func (a *app) quickCheckoutNameReason() string {
	s := &a.quickCheckout
	if reason := a.quickCheckoutRemoteReason(s.remoteName, s.remoteSHA); reason != "" {
		return reason
	}
	name := strings.TrimSpace(s.localName)
	if name == "" {
		return a.t("quickCheckoutNameRequired")
	}
	if !quickCheckoutBranchNameValid(name) {
		return a.t("quickCheckoutNameInvalid")
	}
	for _, b := range a.branches {
		if b.Name == name {
			return a.t("quickCheckoutNameExists")
		}
	}
	return ""
}

// Catch ref-format errors before closing the editable form. Git's existing
// check-ref-format --branch remains authoritative at execution time.
func quickCheckoutBranchNameValid(name string) bool {
	if name == "" || name == "HEAD" || strings.HasPrefix(name, "-") || strings.HasSuffix(name, ".") || strings.Contains(name, "..") || strings.Contains(name, "@{") || strings.ContainsAny(name, " ~^:?*[\\") {
		return false
	}
	for _, r := range name {
		if r < 32 || r == 127 {
			return false
		}
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".lock") {
			return false
		}
	}
	return true
}

func (a *app) submitQuickCheckoutRemote(generation uint64) {
	a.discardStaleQuickCheckout()
	if !a.quickCheckout.open || a.quickCheckout.generation != generation || a.quickCheckout.step != "name" || a.quickCheckoutNameReason() != "" {
		return
	}
	s := a.quickCheckout
	a.quickCheckout = quickCheckoutState{}
	if a.quickCheckoutReady() {
		a.performAs(busyCheckout, a.t("checkoutRemoteBranchFailed"), a.t("remoteBranchCheckedOut"), func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.TrackBranchChecked(ctx, strings.TrimSpace(s.localName), s.remoteName, s.remoteSHA)
		})
	}
	a.invalidate()
}

func (a *app) quickCheckoutNameForm(c *ui.Context, generation uint64) {
	display.Text(c, a.quickCheckout.remoteName, display.TextProps{FontSize: scaledFont(12), Rows: 2})
	display.Text(c, a.t("branchName"), display.TextProps{FontSize: scaledFont(12)})
	input := field.Input(c, &a.quickCheckout.localName, field.InputProps{Variant: field.InputFilled, Size: theme.Middle}).Label(a.t("branchName")).FillWidth().AutoFocus()
	input.OnShortcut(0, ui.KeyEscape, func() { a.quickCheckoutBack() })
	if a.quickCheckout.nameFocus {
		input.SetTextSelection(0, len([]rune(a.quickCheckout.localName)))
		a.quickCheckout.nameFocus = false
	}
	reason := a.quickCheckoutNameReason()
	if reason != "" {
		display.Text(c, reason, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11), Rows: 2})
	}
	submit := reason == "" && input.Submitted()
	ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Children(func() {
		if workbenchButton(c, a.t("quickCheckoutBack"), buttonNormal, false).Clicked() {
			a.quickCheckoutBack()
		}
		if workbenchButton(c, a.t("quickCheckoutCreate"), buttonPrimary, reason != "").Clicked() && reason == "" {
			submit = true
		}
	})
	if submit {
		a.submitQuickCheckoutRemote(generation)
	}
}
