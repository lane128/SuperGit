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
	// MyGo tooltips register their own Escape overlay. A lingering background
	// tip can consume the modal's first Escape, so only build summary tips
	// while the main window has no modal. Keep accessible descriptions below.
	showTips := !a.hasPendingDialog(toastMain)
	// The whole block is one selectable scope. MyGo only lets a paragraph be
	// selected when a Selectable box sits between it and any control the pane
	// lives under, and the scroll area around this column is not a scope of its
	// own.
	scope := ui.Column(c).FillWidth().Padding(14).Gap(8).Selectable()
	scope.Children(func() {
		subject := display.Text(c, commit.Subject, display.TextProps{Heading: 5, Strong: true, FontSize: scaledFont(tk.FontSize)})
		a.commitMessageLinkMenu(c, subject, commit.Subject, scope)
		// The refs sit with the subject and before the message body: they say
		// which branches and tags point at this commit, which is what the
		// reader checks first. Below a long body they would need a scroll to
		// find, and a body can be arbitrarily long.
		head := headBranch(commit.Refs)
		// commitRefs keeps the branch name after "HEAD -> ", so a commit whose
		// only ref is HEAD's branch still arrives with one; a non-empty head
		// never needs its own condition here.
		if refs := commitRefOrder(commitRefs(commit.Refs), a.historyRef, head); len(refs) > 0 {
			a.commitSummaryRefs(c.Key("summary-refs:"+commit.SHA), scope, tk, commit.Refs, head, refs, showTips)
		}
		if a.commitMessageErr != "" {
			display.Text(c, a.commitMessageErr, display.TextProps{Type: display.TextDanger, FontSize: scaledFont(12), Rows: 4})
			if workbenchButton(c, a.t("retryCommitMessage"), buttonNormal, a.commitDiffLoading).Clicked() {
				a.loadCommitDetail(true)
			}
		}
		if body := commitBody(a.commitMessage); body != "" {
			message := display.Text(c, body, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12)})
			a.commitMessageLinkMenu(c, message, body, scope)
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
			author.Description(commit.Author)
			if showTips {
				author.Tooltip(commit.Author)
			}
			a.commitAuthorMenu(c, author, commit.Author, commit.AuthorEmail)
			dateText := formatDateTime(commit.Date)
			date := display.Text(c, dateText, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).
				Shrink(0).SingleLine()
			a.commitDateMenu(c, date, dateText)
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
					signer.Description(sig.Signer)
					if showTips {
						signer.Tooltip(sig.Signer)
					}
				}
			})
		}
		if len(commit.Parents) > 0 {
			ui.Row(c).FillWidth().Gap(8).Wrap().AlignItems(ui.Center).Children(func() {
				display.Text(c, a.t("parent"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(10)}).Width(54)
				for _, parent := range commit.Parents {
					sha := parent
					button := workbenchButton(c, shortSHA(sha), buttonNormal, false).Shrink(0)
					if button.Clicked() {
						a.jumpWithHistory(sha)
					}
				}
			})
		}
	})
}

// Keep the first row compact even for release commits with many tags. The
// disclosure leaves every original pill and its menu available on demand.
func (a *app) commitSummaryRefs(c *ui.Context, scope ui.Element, tk *theme.Tokens, rawRefs, head string, refs []string, showTips bool) {
	const gap, inset = 4, 14
	group := ui.Column(c).FillWidth().Gap(gap)
	width := scope.Bounds().W
	measured := ui.Local(group, "width", func() float32 { return -1 })
	if *measured != width {
		*measured = width
		c.After(0)
	}
	if width <= 0 {
		width, _ = c.Size()
	}
	width = max(0, width-2*inset)
	headerWidth := float32(0)
	if head != "" {
		headerWidth = historyRefPillWidth(c, "HEAD", true, gap)
	}
	pillLimit := min(width, refCapsuleWidth(refCapsuleDetailWidth))
	type inlineRef struct {
		label string
		width float32
	}
	var inline []inlineRef
	used := headerWidth
	for _, ref := range refs {
		badge := a.refBadge(ref)
		label := measuredRefPillLabel(c, badge.Label, badge.Icon != nil, pillLimit, refCapsuleDetailRunes)
		cost := historyRefPillWidth(c, label, badge.Icon != nil, gap)
		if used+cost-gap > width {
			break
		}
		inline = append(inline, inlineRef{label, cost})
		used += cost
	}
	// Keep room for the disclosure in both states, so toggling it never
	// shifts the badges that already fit on the first line.
	disclosureWidth := scaledFont(54)
	if len(inline) < len(refs) {
		for len(inline) > 0 && used+disclosureWidth > width {
			used -= inline[len(inline)-1].width
			inline = inline[:len(inline)-1]
		}
		// Give the highest-priority ref the space beside the disclosure
		// when a readable shortened label can fit there.
		if len(inline) == 0 {
			badge := a.refBadge(refs[0])
			limit := min(pillLimit, width-headerWidth-disclosureWidth-gap)
			label := measuredRefPillLabel(c, badge.Label, badge.Icon != nil, limit, refCapsuleDetailRunes)
			if cost := historyRefPillWidth(c, label, badge.Icon != nil, 0); cost <= limit {
				inline = append(inline, inlineRef{label, cost + gap})
			}
		}
	}
	shown := len(inline)
	folded := len(refs) - shown
	expanded := ui.Local(group, "expanded", func() bool { return false })
	previousShown := ui.Local(group, "shown", func() int { return shown })
	group.Children(func() {
		ui.Row(c).Gap(gap).Wrap().Children(func() {
			a.headTag(c, rawRefs, tk.BgContainer, false)
			var firstRevealed, lastFocusable ui.Element
			foundRevealed := false
			for i, ref := range refs[:shown] {
				pill, focusable := a.commitSummaryRef(c.Key("ref:"+ref), tk, ref, inline[i].label, pillLimit, showTips)
				if focusable {
					lastFocusable = pill
					if !foundRevealed && i >= *previousShown {
						firstRevealed, foundRevealed = pill, true
					}
				}
			}
			buttonContext := c.Key("disclosure")
			if folded == 0 {
				// Check the old focus before hiding this stable control.
				button := ui.ButtonBase(buttonContext)
				if button.Focused() {
					if foundRevealed {
						firstRevealed.Focus()
					} else {
						lastFocusable.Focus()
					}
				}
				button.Hidden(true)
			} else {
				text, label := "+"+itoa(folded), a.tf("moreRefs", folded)
				if *expanded {
					text, label = "−"+itoa(folded), a.t("showFewerRefs")
				}
				button := workbenchButton(buttonContext, text, buttonNormal, false).
					Width(disclosureWidth).Height(refCapsuleHeight).PaddingX(0).Shrink(0).
					Role(ui.RoleDisclosure).Label(label).Expanded(*expanded)
				if showTips {
					button.Tooltip(label)
				}
				if button.Clicked() {
					*expanded = !*expanded
					c.After(0)
				}
			}
			*previousShown = shown
			// Keep both sets in the same parent so a ref that moves between
			// them during resize keeps its focus and context menu identity.
			if *expanded {
				for _, ref := range refs[shown:] {
					a.commitSummaryRef(c.Key("ref:"+ref), tk, ref, "", pillLimit, showTips)
				}
			}
		})
	})
}

func (a *app) commitSummaryRef(c *ui.Context, tk *theme.Tokens, ref, label string, limit float32, showTips bool) (ui.Element, bool) {
	badge := a.refBadge(ref)
	if label == "" {
		label = measuredRefPillLabel(c, badge.Label, badge.Icon != nil, limit, refCapsuleDetailRunes)
	}
	// Use the same lane color and ref actions as the commit list.
	kind, name, filterable := a.parseCommitRef(ref)
	color := tk.TextSecondary
	if filterable {
		color = refBadgeColor(a.lanePalette(tk), a.historyGraph, a.selectedCommitIndex(), kind, name)
	} else if family, ok := refColor(tk, badge.Color); ok {
		color = family
	}
	pill := refPill(c, refCapsuleColors(color, tk.BgContainer), badge.Icon, label).
		MaxWidth(limit).Description(badge.Label)
	if filterable {
		pill.Focusable().Label(badge.Label).FocusRing(true)
	}
	if showTips {
		pill.Tooltip(badge.Label)
	}
	a.refBadgeMenu(c, pill, ref)
	return pill, filterable
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
