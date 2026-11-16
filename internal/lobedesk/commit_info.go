package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

// The commit's Summary tab shows the commit itself: subject, message, refs,
// hash and metadata. The files and the patch live in their own columns beside
// it, as Sublime Merge lays them out.

// commitInfoContent is the Summary tab's text: subject, body, refs, the hash
// and the commit's metadata.
func (a *app) commitInfoContent(c *ui.Context, commit gitrepo.Commit, tk *theme.Tokens) {
	// The whole block is one selectable scope. MyGo only lets a paragraph be
	// selected when a Selectable box sits between it and any control the pane
	// lives under, and the scroll area around this column is not a scope of its
	// own.
	ui.Column(c).FillWidth().Padding(14).Gap(8).Selectable().Children(func() {
		display.Text(c, commit.Subject, display.TextProps{Heading: 5, Strong: true, Rows: 3, FontSize: scaledFont(tk.FontSize)})
		// The refs sit with the subject and before the message body: they say
		// which branches and tags point at this commit, which is what the
		// reader checks first. Below a long body they would need a scroll to
		// find, and a body can be arbitrarily long.
		head := headBranch(commit.Refs)
		// commitRefs keeps the branch name after "HEAD -> ", so a commit whose
		// only ref is HEAD's branch still arrives with one; a non-empty head
		// never needs its own condition here.
		if refs := commitRefOrder(commitRefs(commit.Refs), a.historyRef, head); len(refs) > 0 {
			ui.Row(c).Gap(4).Wrap().Children(func() {
				a.headTag(c, commit.Refs, tk.BgContainer, false)
				for _, ref := range refs {
					badge := a.refBadge(ref)
					// Same lane color as the list row, so the selected
					// commit's labels agree with the row they came from.
					kind, name, filterable := a.parseCommitRef(ref)
					color := tk.TextSecondary
					if filterable {
						color = refBadgeColor(a.lanePalette(tk), a.historyGraph, a.selectedCommitIndex(), kind, name)
					} else if family, ok := refColor(tk, badge.Color); ok {
						color = family
					}
					pill := refPill(c, refCapsuleColors(color, tk.BgContainer), badge.Icon,
						elideMiddle(badge.Label, refCapsuleDetailRunes)).
						MaxWidth(refCapsuleWidth(refCapsuleDetailWidth)).Tooltip(badge.Label)
					// The same menu the list row's badge has: the details
					// pane shows the same ref, so it should offer the same
					// actions where the reader is looking.
					a.refBadgeMenu(c, pill, ref)
				}
			})
		}
		if a.commitMessageErr != "" {
			display.Text(c, a.commitMessageErr, display.TextProps{Type: display.TextDanger, FontSize: scaledFont(12), Rows: 4})
			if action.Button(c, a.t("retryCommitMessage"), action.ButtonProps{Size: theme.Small}).Disabled(a.commitDiffLoading).Clicked() {
				a.loadCommitDetail(true)
			}
		}
		if body := commitBody(a.commitMessage); body != "" {
			display.Text(c, body, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12)})
		}
		ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
			// The hash gives way when the pane is too narrow for it and the
			// button together: without the shrink the 40-character SHA at the
			// large preset pushed the copy button past the pane's edge, where
			// it was clipped and unreachable.
			display.Text(c, commit.SHA, display.TextProps{Code: true, FontSize: scaledFont(10), Type: display.TextSecondary}).
				SingleLine().Grow(1).Basis(0).MinWidth(0)
			// The row menu already copies the SHA, but the reading eye is
			// here; a copy button beside the hash saves hunting for the row.
			copySHA := action.ActionIcon(c, lucide.Copy(), action.ActionIconProps{
				Size: theme.Small, Title: a.t("copySHA"),
			}).Shrink(0)
			if copySHA.Clicked() {
				a.copyText(commit.SHA)
			}
		})
		// Author and date share one row. Two full metadata rows cost about
		// 26px, which in a short pane is a line of the message body; the date
		// keeps its old value right-aligned, so a long author name elides
		// instead of pushing it off the edge.
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			display.Text(c, a.t("author"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(10)}).Width(54)
			author := display.Text(c, commit.Author, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).
				Grow(1).Basis(0).MinWidth(0).SingleLine()
			// The name gives way to the date when the pane is narrow, so the
			// tooltip is where the whole name is read. It stays on truncated
			// text only: a tooltip repeating a name that already fits is noise.
			author.Tooltip(commit.Author)
			display.Text(c, formatDateTime(commit.Date), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).
				Shrink(0).SingleLine()
		})
		// Git verified the signature (or failed to). Unsigned commits get no
		// row at all: a badge saying "unsigned" on every ordinary commit would
		// be noise, and the reader asked for signed history by signing.
		if sig := a.commitSig; sig.Signed() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				display.Tag(c, a.signatureLabel(sig.Status), display.TagProps{
					Variant: display.TagFilled,
					Color:   display.TagColor(signatureTagColor(sig.Status)),
					Size:    theme.Small,
					Round:   true,
					Icon:    signatureIcon(sig.Status),
				}).Shrink(0)
				if sig.Signer != "" {
					signer := display.Text(c, sig.Signer, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).
						Grow(1).Basis(0).MinWidth(0).SingleLine()
					signer.Tooltip(sig.Signer)
				}
			})
		}
		if len(commit.Parents) > 0 {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				display.Text(c, a.t("parent"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(10)}).Width(54)
				for _, parent := range commit.Parents {
					sha := parent
					button := action.Button(c, shortSHA(sha), action.ButtonProps{Variant: action.Text, Size: theme.Small})
					if button.Clicked() {
						a.jumpToCommit(sha)
					}
				}
			})
		}
	})
}

// signatureTagColor maps Git's %G? letter to the badge's palette: a verified
// signature reads as success, a wrong or unusable one as an error, a good
// signature whose key is not trusted as information, and anything Git could
// not check as neutral.
func signatureTagColor(status string) string {
	switch status {
	case "G":
		return "success"
	case "U":
		return "info"
	case "B", "R", "X", "Y":
		return "error"
	}
	return "gray"
}

func signatureIcon(status string) *ui.SVG {
	switch status {
	case "G", "U":
		return lucide.ShieldCheck()
	case "B", "R":
		return lucide.ShieldX()
	case "X", "Y":
		return lucide.ShieldAlert()
	}
	return lucide.ShieldQuestion()
}

func (a *app) signatureLabel(status string) string {
	switch status {
	case "G":
		return a.t("signatureVerified")
	case "U":
		return a.t("signatureUntrusted")
	case "B":
		return a.t("signatureBad")
	case "X":
		return a.t("signatureExpired")
	case "Y":
		return a.t("signatureKeyExpired")
	case "R":
		return a.t("signatureKeyRevoked")
	}
	return a.t("signatureUncheckable")
}
