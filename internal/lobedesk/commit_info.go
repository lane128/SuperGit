package lobedesk

import (
	"fmt"
	"github.com/egoist/mygo/ui"
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
		if stash, ok := a.selectedStashObject(); ok {
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				display.Tag(c, stash.Ref, display.TagProps{Size: theme.Small, Color: display.TagWarning}).Shrink(0)
				if a.commitFilesLoaded {
					display.Text(c, fmt.Sprintf("%s (%d)", a.t("changes"), a.stashTrackedCount), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)})
					display.Tag(c, fmt.Sprintf("−%d", a.stashDeleted), display.TagProps{Size: theme.Small, Color: display.TagError})
					display.Tag(c, fmt.Sprintf("+%d", a.stashAdded), display.TagProps{Size: theme.Small, Color: display.TagSuccess})
				}
				ui.Row(c).Grow(1)
				a.stashActions(c, stash)
			})
			display.Text(c, a.t("stashTree")+"  "+stash.Tree, display.TextProps{Code: true, FontSize: scaledFont(10), Type: display.TextSecondary}).Font(a.patchFontFamily()).SingleLine().Tooltip(stash.Tree)
		}
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
		ui.Column(c).FillWidth().Gap(2).Children(func() {
			a.commitSummaryMetadata(c, commit, showTips)
		})
	})
}

func (a *app) commitSummaryMetadata(c *ui.Context, commit gitrepo.Commit, showTips bool) {
	// Use the tallest single-line control as a floor, not a fixed height:
	// wrapped identities and multiple parents must still be able to grow.
	rowHeight := max(float32(24), scaledFont(24))
	field := func(label string, values func()) {
		ui.Row(c).FillWidth().MinHeight(rowHeight).Gap(8).AlignItems(ui.Center).Children(func() {
			display.Text(c, label, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(10)}).Width(scaledFont(54)).Shrink(0)
			ui.Row(c).Grow(1).Basis(0).MinWidth(0).Gap(4).AlignItems(ui.Center).Children(values)
		})
	}
	field(a.t("summaryHash"), func() {
		// Natural width keeps Copy next to the hash; only the text shrinks.
		hash := display.Text(c, commit.SHA, display.TextProps{Code: true, FontSize: scaledFont(10), Type: display.TextSecondary}).
			Font(a.patchFontFamily()).SingleLine().Shrink(1).MinWidth(0).Description(commit.SHA)
		if showTips {
			hash.Tooltip(commit.SHA)
		}
		if compactActionIcon(c, lucide.Copy(), a.t("copySHA"), buttonNormal, false).Shrink(0).Clicked() {
			a.copyText(commit.SHA)
		}
	})
	field(a.t("author"), func() {
		identity := commit.Author
		if commit.AuthorEmail != "" {
			identity += "<" + commit.AuthorEmail + ">"
		}
		author := display.Text(c, identity, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).
			MinWidth(0).Shrink(1).Description(identity)
		if showTips {
			author.Tooltip(identity)
		}
		a.commitAuthorMenu(c, author, commit.Author, commit.AuthorEmail)
	})
	field(a.t("summaryDate"), func() {
		dateText := formatDateTime(commit.Date)
		date := display.Text(c, dateText, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).MinWidth(0).Shrink(1)
		a.commitDateMenu(c, date, dateText)
	})
	// Git verified the signature (or failed to). Unsigned commits get no
	// row at all: a badge saying "unsigned" on every ordinary commit would
	// be noise, and the reader asked for signed history by signing.
	if sig := a.commitSig; sig.Signed() {
		ui.Row(c).MinHeight(rowHeight).Gap(6).AlignItems(ui.Center).Children(func() {
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
		field(a.t("parent"), func() {
			tk := theme.TokensOf(c)
			ui.Row(c).FillWidth().Gap(12).Wrap().AlignItems(ui.Center).Children(func() {
				for i, parent := range commit.Parents {
					sha := parent
					if a.selectedStash != "" && i > 0 {
						// The index and untracked snapshots are not History tips.
						display.Text(c, shortSHA(sha), display.TextProps{Code: true, FontSize: scaledFont(11)}).Font(a.patchFontFamily()).Tooltip(sha)
						continue
					}
					// Empty URL: this is history navigation, not an external link.
					link := ui.Link(c.Key(sha), shortSHA(sha), "").Font(a.patchFontFamily()).FontSize(scaledFont(11)).
						SingleLine().Shrink(0).Underline().TextColor(tk.Link).FocusRing(true).FocusRingColor(tk.Link).Description(sha)
					if link.Hovered() {
						link.TextColor(tk.LinkHover)
					}
					if showTips {
						link.Tooltip(sha)
					}
					if link.Clicked() {
						if a.selectedStash != "" {
							a.showCommit(sha)
						} else {
							a.jumpWithHistory(sha)
						}
					}
				}
			})
		})
	}
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
		w, _ := c.MeasureText(0, ui.Span{Text: "HEAD →", Size: scaledFont(11)})
		headerWidth = w + gap
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
		ui.Row(c).Gap(gap).Wrap().AlignItems(ui.Center).Children(func() {
			if head != "" {
				display.Text(c, "HEAD →", display.TextProps{Type: display.TextSecondary, Strong: true, FontSize: scaledFont(11)}).
					SingleLine().Shrink(0).Tooltip(a.tf("headBadgeTip", head))
			}
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
	chip := summaryRefChip(c, tk, color, badge.Icon, label).
		MaxWidth(limit).Description(badge.Label)
	if filterable {
		chip.Focusable().Label(badge.Label).FocusRing(true)
	}
	if showTips {
		chip.Tooltip(badge.Label)
	}
	a.refBadgeMenu(c, chip, ref, a.selectedSHA)
	return chip, filterable
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
