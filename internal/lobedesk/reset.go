package lobedesk

import (
	"context"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

type resetState struct {
	root    string
	head    string
	branch  string
	open    bool
	sha     string
	subject string
}

func (a *app) resetTargetCurrent(target resetState) bool {
	if a.repo == nil || (target.root != "" && a.repo.Root != target.root) {
		return false
	}
	if a.status.Head != target.head || a.status.Branch != target.branch {
		a.queue("warning", a.t("resetTargetChanged"), "")
		a.invalidate()
		return false
	}
	return true
}

func (a *app) discardStaleReset() bool {
	if a.reset.root != "" && !a.resetTargetCurrent(a.reset) {
		a.reset = resetState{}
		return true
	}
	return false
}

func (a *app) openResetDialog(commit gitrepo.Commit) {
	a.discardStaleReset()
	if a.reset.open {
		a.queue("warning", a.t("confirmationPending"), "")
		a.invalidate()
		return
	}
	if a.rejectPendingDialog(toastMain) {
		return
	}
	root := ""
	if a.repo != nil {
		root = a.repo.Root
	}
	a.reset = resetState{root: root, head: a.status.Head, branch: a.status.Branch, open: true, sha: commit.SHA, subject: commit.Subject}
	a.invalidate()
}

// resetDialog offers the three reset modes in one place instead of three
// nearly identical confirm dialogs. Soft and mixed only change references and
// the index, so they run from the dialog; hard discards work and asks again.
func (a *app) resetDialog(c *ui.Context) {
	a.resetDialogFor(c, toastMain)
}

func (a *app) resetDialogFor(c *ui.Context, owner toastOwner) {
	if owner != toastMain {
		return
	}
	a.discardStaleReset()
	if !a.reset.open {
		return
	}
	tk := theme.TokensOf(c)
	chosen := ""
	ui.Modal(c, &a.reset.open, func() {
		ui.Column(c).Width(480).Padding(18).Gap(10).
			Background(tk.BgElevated).Radius(tk.BorderRadiusLG).Border(1, tk.Border).
			Children(func() {
				display.Text(c, a.t("resetTitle"), display.TextProps{Strong: true, FontSize: scaledFont(14)})
				display.Text(c, a.tf("resetMessage", shortSHA(a.reset.sha), a.reset.subject),
					display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12), Rows: 3})
				// A reset cannot start while another operation or a read holds
				// the repository. Disabling the modes says so up front; a click
				// that slipped through before the frame lands keeps the dialog
				// open instead of closing it for a reset that never ran.
				blocked := a.busy || a.loading || a.repo == nil
				ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
					for _, option := range []struct{ key, label string }{
						{"soft", a.t("resetSoft")},
						{"mixed", a.t("resetMixed")},
						{"hard", a.t("resetHard")},
					} {
						role := buttonNormal
						if option.key == "hard" {
							role = buttonDanger
						}
						button := workbenchButton(c, option.label, role, blocked)
						if !blocked && button.Clicked() {
							chosen = option.key
						}
					}
				})
				display.Text(c, a.t("resetModesHelp"), display.TextProps{
					Type: display.TextSecondary, FontSize: scaledFont(10), Rows: 3,
				})
				ui.Row(c).FillWidth().Justify(ui.End).Children(func() {
					// Cancel takes the opening focus: the first focusable is
					// Soft reset, so Enter on a fresh dialog would reset the
					// repository before the reader chose a mode.
					if workbenchButton(c, a.t("cancel"), buttonNormal, false).
						AutoFocus().Clicked() {
						a.reset.open = false
					}
				})
			})
	})
	switch chosen {
	case "soft", "mixed":
		// performReset keeps the dialog when the operation cannot start, so
		// the reader still has the mode and the commit to choose again.
		a.performReset(chosen, a.reset.sha)
	case "hard":
		target := a.reset
		a.reset = resetState{}
		a.askConfirm(a.t("resetHardConfirmTitle"),
			a.tf("resetHardConfirmMessage", shortSHA(target.sha), target.subject), a.t("resetHardAction"), true, func() {
				a.performResetTarget("hard", target)
			})
	}
	if !a.reset.open && chosen == "" {
		// Cancelled by the button or by closing the modal.
		a.reset = resetState{}
	}
}

// performReset clears the dialog only once the reset actually starts: a busy
// or repository-less app would otherwise close it for an operation that never
// ran.
func (a *app) performReset(mode, sha string) bool {
	if a.discardStaleReset() {
		return false
	}
	target := a.reset
	target.sha = sha
	return a.performResetTarget(mode, target)
}

func (a *app) performResetTarget(mode string, target resetState) bool {
	if !a.resetTargetCurrent(target) {
		return false
	}
	expected := gitrepo.Status{Head: target.head, Branch: target.branch}
	started := a.performDoneAs(busyReset, a.t("resetFailed"), a.t("resetSuccess"), nil, func(ctx context.Context, repo *gitrepo.Repository) error {
		return repo.ResetToChecked(ctx, target.sha, mode, expected)
	})
	if started {
		a.reset = resetState{}
	}
	return started
}
