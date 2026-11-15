package lobedesk

import (
	"context"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

type resetState struct {
	open    bool
	sha     string
	subject string
}

func (a *app) openResetDialog(commit gitrepo.Commit) {
	a.reset = resetState{open: true, sha: commit.SHA, subject: commit.Subject}
	a.invalidate()
}

// resetDialog offers the three reset modes in one place instead of three
// nearly identical confirm dialogs. Soft and mixed only change references and
// the index, so they run from the dialog; hard discards work and asks again.
func (a *app) resetDialog(c *ui.Context) {
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
						button := action.Button(c, option.label, action.ButtonProps{Variant: action.Default, Size: theme.Small}).
							Disabled(blocked)
						if option.key == "hard" {
							button.TextColor(tk.Error)
						}
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
					if action.Button(c, a.t("cancel"), action.ButtonProps{Variant: action.Default, Size: theme.Small}).
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
		sha, subject := a.reset.sha, a.reset.subject
		a.reset = resetState{}
		a.askConfirm(a.t("resetHardConfirmTitle"),
			a.tf("resetHardConfirmMessage", shortSHA(sha), subject), a.t("resetHardAction"), true, func() {
				a.performReset("hard", sha)
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
	started := a.performDoneAs(busyReset, a.t("resetFailed"), a.t("resetSuccess"), nil, func(ctx context.Context, repo *gitrepo.Repository) error {
		return repo.ResetTo(ctx, sha, mode)
	})
	if started {
		a.reset = resetState{}
	}
	return started
}
