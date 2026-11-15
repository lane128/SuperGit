package lobedesk

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

const (
	// branchPaneWidth is the details pane's opening width; the list takes the
	// rest of the row. A zero saved share keeps this default.
	branchPaneWidth = 320
	// The list and the selected ref's details keep at least these widths when
	// the divider between them is dragged.
	branchSplitMinList   = 280
	branchSplitMinDetail = 240
	// branchOpeningRoom stands in for the pane row's width on the page's first
	// frame, when layout has not measured it yet.
	branchOpeningRoom = 940
)

// branchDefaultShare is the list's share of the row when the reader has not
// dragged the divider: everything the details' opening width leaves.
func (a *app) branchDefaultShare() float32 {
	room := a.branchSplit.rowWidth
	if room <= 0 {
		room = branchOpeningRoom
	}
	return 1 - shareForWidth(room, branchPaneWidth)
}

// branchShare is how much of the page the branch list takes; the selected
// ref's details keep the rest.
func (a *app) branchShare() float32 {
	return a.branchSplit.fraction(
		a.branchDefaultShare(),
		branchSplitMinList, branchSplitMinDetail)
}

type branchDetailKind int

const (
	branchDetailLocal branchDetailKind = iota
	branchDetailRemote
	branchDetailTag
)

// branchDetailSelection is one selected row's data, read from the complete
// lists rather than the search-filtered slices: hiding the selected row with a
// search must not blank the pane beside it.
type branchDetailSelection struct {
	kind   branchDetailKind
	branch gitrepo.Branch
	tag    gitrepo.Tag
	remote string
	name   string
}

// selectedBranchDetail resolves a.branchSel into the row it names. The key
// prefix says which list to search, mirroring the row keys the page builds.
func (a *app) selectedBranchDetail() (branchDetailSelection, bool) {
	kind, name, ok := strings.Cut(a.branchSel, ":")
	if !ok || name == "" {
		return branchDetailSelection{}, false
	}
	switch kind {
	case "local":
		for _, branch := range a.branches {
			if branch.Name == name {
				return branchDetailSelection{kind: branchDetailLocal, branch: branch, name: branch.Name}, true
			}
		}
	case "remote":
		for _, ref := range a.remoteBranches {
			if ref.Name == name {
				remote, short := a.remoteBranchParts(ref.Name)
				return branchDetailSelection{kind: branchDetailRemote, branch: ref, remote: remote, name: short}, true
			}
		}
	case "tag":
		for _, tag := range a.tags {
			if tag.Name == name {
				return branchDetailSelection{kind: branchDetailTag, tag: tag, name: tag.Name}, true
			}
		}
	}
	return branchDetailSelection{}, false
}

// branchDetail is the right pane: a fixed header naming the selected ref, the
// scrollable metadata, and the actions in a footer that stays reachable when
// the metadata is long.
func (a *app) branchDetail(c *ui.Context) {
	ui.Column(c).Grow(1 - a.branchShare()).Basis(0).MinWidth(0).AlignSelf(ui.Stretch).Children(func() {
		sel, ok := a.selectedBranchDetail()
		if !ok {
			a.diffPlaceholder(c, lucide.GitBranch(), a.branchDetailEmptyText())
			return
		}
		a.branchDetailHeader(c, sel)
		layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.branchDetailScroll}, func() {
			ui.Column(c).FillWidth().Padding(14).Gap(12).Children(func() {
				switch sel.kind {
				case branchDetailLocal:
					a.localBranchDetails(c, sel)
				case branchDetailRemote:
					a.remoteBranchDetails(c, sel)
				case branchDetailTag:
					a.tagDetails(c, sel)
				}
			})
		}).Grow(1).Basis(0).MinHeight(0)
		a.branchDetailActions(c, sel)
	})
}

// branchDetailEmptyText names what the pane says when no row is selected. A
// repository without any refs has nothing to select, so it gets the page's own
// empty state instead of an instruction the list cannot follow; a search that
// matches nothing says so rather than pointing at the list again.
func (a *app) branchDetailEmptyText() string {
	switch {
	case strings.TrimSpace(a.branchQuery) != "":
		return a.t("searchBranchesNoMatch")
	case len(a.branches) == 0 && len(a.remoteBranches) == 0 && len(a.tags) == 0:
		return a.t("noBranchesHint")
	}
	return a.t("branchDetailHint")
}

// branchDetailNameRunes is how much of a ref name fits the header before the
// middle elision applies. The estimate follows the divider, so a pane dragged
// narrow elides sooner instead of letting SingleLine clip the tail. Without a
// text-measuring API this is approximate; the tooltip still carries the name.
func (a *app) branchDetailNameRunes() int {
	room := a.branchSplit.rowWidth
	if room <= 0 {
		room = branchOpeningRoom
	}
	width := room*(1-a.branchShare()) - splitterWidth
	n := int((width - 50) / 7.5)
	if n < 12 {
		n = 12
	}
	if n > 40 {
		n = 40
	}
	return n
}

func (a *app) branchDetailHeader(c *ui.Context, sel branchDetailSelection) {
	tk := theme.TokensOf(c)
	ground := tk.BgContainer
	icon := lucide.GitBranch()
	switch sel.kind {
	case branchDetailRemote:
		icon = lucide.Cloud()
	case branchDetailTag:
		icon = lucide.Tag()
	}
	ui.Column(c).FillWidth().Shrink(0).Padding(12, 14).Gap(6).
		BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icon).FontSize(scaledFont(14)).TextColor(tk.TextSecondary).Shrink(0)
			// A branch name can outgrow the narrowest pane; the ellipse keeps
			// both ends and the tooltip holds the whole name.
			display.Text(c, elideMiddle(sel.name, a.branchDetailNameRunes()), display.TextProps{Strong: true, FontSize: scaledFont(13)}).
				SingleLine().Grow(1).Basis(0).MinWidth(0).Tooltip(sel.name)
		})
		ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Wrap().Children(func() {
			switch sel.kind {
			case branchDetailLocal:
				refPill(c, neutralCapsule(tk, ground), nil, a.t("localBranches"))
				if sel.branch.Current {
					a.statusTag(c, a.t("current"), display.TagProps{
						Variant: display.TagFilled, Color: display.TagSuccess, Size: theme.Small, Round: true,
					})
				}
			case branchDetailRemote:
				if sel.remote != "" {
					refPill(c, neutralCapsule(tk, ground), nil, sel.remote)
				}
				refPill(c, neutralCapsule(tk, ground), nil, a.t("remoteBranches"))
			case branchDetailTag:
				if sel.tag.ObjectType == "tag" {
					refPill(c, refCapsuleColors(tk.Gold, ground), nil, a.t("annotatedTag"))
				} else {
					refPill(c, neutralCapsule(tk, ground), nil, a.t("lightweightTag"))
				}
			}
		})
	})
}

func (a *app) localBranchDetails(c *ui.Context, sel branchDetailSelection) {
	branch := sel.branch
	switch {
	case branch.Upstream == "":
		a.branchDetailField(c, a.t("upstream"), a.t("branchNoUpstream"))
	case branch.UpstreamGone:
		ui.Column(c).FillWidth().Gap(1).Children(func() {
			display.Text(c, a.t("upstream"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(10)}).SingleLine()
			name := elideMiddle(branch.Upstream, a.branchDetailNameRunes())
			display.Text(c, name+" · "+a.t("upstreamGone"),
				display.TextProps{Type: display.TextWarning, FontSize: scaledFont(11)}).SingleLine().FillWidth().Tooltip(branch.Upstream)
		})
	default:
		// The counts sit on the upstream line, as in the list, so the reader
		// sees whether a pull or a push is waiting without doing the sum.
		value := fmt.Sprintf("%s · ↑%d ↓%d", branch.Upstream, branch.Ahead, branch.Behind)
		a.branchDetailField(c, a.t("upstream"), value)
	}
	a.commitMetadata(c, branch.Commit, branch.SHA, a.t("latestCommit"))
	if sha := branch.Commit.SHA; sha != "" {
		a.branchDetailField(c, "SHA", shortSHA(sha))
	} else if branch.SHA != "" {
		a.branchDetailField(c, "SHA", shortSHA(branch.SHA))
	}
}

func (a *app) remoteBranchDetails(c *ui.Context, sel branchDetailSelection) {
	ref := sel.branch
	// The commit answers what the ref points at; the SHA closes the block, as
	// it does for a local branch, instead of leading the pane.
	a.commitMetadata(c, ref.Commit, ref.SHA, a.t("latestCommit"))
	sha := ref.Commit.SHA
	if sha == "" {
		sha = ref.SHA
	}
	if sha != "" {
		a.branchDetailField(c, "SHA", shortSHA(sha))
	}
	// An empty tracking value names nothing, so the field appears only when a
	// local branch actually tracks this ref.
	if tracker := a.trackingBranch(ref.Name); tracker != "" {
		a.branchDetailField(c, a.t("trackingBranch"), tracker)
	}
}

func (a *app) tagDetails(c *ui.Context, sel branchDetailSelection) {
	tag := sel.tag
	targetLabel := a.t("commit")
	if tag.TargetType != "" && tag.TargetType != "commit" {
		targetLabel = a.tagTargetLabel(tag.TargetType)
	}
	// The target is what the tag is for; an annotated tag also has its own
	// object, which is worth showing only when it differs from the target.
	a.branchDetailField(c, targetLabel, shortSHA(tag.TargetSHA))
	if tag.ObjectSHA != "" && tag.ObjectSHA != tag.TargetSHA {
		a.branchDetailField(c, a.t("tagObject"), shortSHA(tag.ObjectSHA))
	}
	// The annotation is the tag's own content, so it comes before the commit
	// it points at; a long release note pushes the commit details down rather
	// than being buried at the bottom of the pane.
	if tag.Message != "" {
		// A tag message can be several paragraphs; the pane scrolls, so show
		// all of it rather than cutting it at a fixed number of lines.
		a.branchDetailParagraph(c, a.t("tagMessage"), tag.Message, 0)
	}
	if tag.TargetType == "commit" {
		a.commitMetadata(c, tag.Commit, tag.TargetSHA, a.t("latestCommit"))
	}
	if tag.SymbolicRef != "" {
		display.Text(c, tag.SymbolicRef, display.TextProps{Code: true, FontSize: scaledFont(10), Type: display.TextSecondary}).
			SingleLine().FillWidth().Tooltip(tag.SymbolicRef)
	}
}

// commitMetadata prints the one commit a ref points at: its subject and the
// author and time on a line below. The caller names the block and prints the
// SHA, since the same pieces describe a branch tip and a tag's target.
func (a *app) commitMetadata(c *ui.Context, commit gitrepo.Commit, fallbackSHA, title string) {
	subject := commit.Subject
	if subject == "" && fallbackSHA == "" {
		return
	}
	a.branchDetailParagraph(c, title, subject, 4)
	var meta []string
	if commit.Author != "" {
		meta = append(meta, commit.Author)
	}
	if date := formatDateTime(commit.Date); date != "" {
		meta = append(meta, date)
	}
	if len(meta) > 0 {
		a.branchDetailField(c, a.t("author"), strings.Join(meta, " · "))
	}
}

// branchDetailField is one label/value pair. The value carries a tooltip so a
// long upstream, author or branch name stays readable in a narrow pane.
func (a *app) branchDetailField(c *ui.Context, label, value string) {
	if strings.TrimSpace(value) == "" {
		value = "—"
	}
	ui.Column(c).FillWidth().Gap(1).Children(func() {
		display.Text(c, label, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(10)}).SingleLine()
		display.Text(c, value, display.TextProps{FontSize: scaledFont(11)}).SingleLine().FillWidth().Tooltip(value)
	})
}

// branchDetailParagraph is a field whose value may wrap: a commit subject or
// an annotated tag's message.
func (a *app) branchDetailParagraph(c *ui.Context, label, value string, rows int) {
	ui.Column(c).FillWidth().Gap(2).Children(func() {
		display.Text(c, label, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(10)}).SingleLine()
		display.Text(c, value, display.TextProps{FontSize: scaledFont(11), Rows: rows}).
			FillWidth().Tooltip(value)
	})
}

// detailAction is one footer button. MyGo hides the tooltip of a disabled
// element, so a disabled action carries its explanation on a wrapper that
// still takes the pointer; without one it would be a dead control with no
// reason on screen.
func (a *app) detailAction(c *ui.Context, label, hint string, disabled, danger bool, onClick func()) {
	build := func() {
		button := action.Button(c, label, action.ButtonProps{Variant: action.Default, Size: theme.Small}).Disabled(disabled)
		if danger {
			button.TextColor(theme.TokensOf(c).Error)
		}
		if !disabled && button.Clicked() {
			onClick()
		}
	}
	if disabled && hint != "" {
		ui.Box(c).Shrink(0).Tooltip(hint).Children(build)
		return
	}
	build()
}

// branchDetailActions is the footer of buttons. They call the same functions as
// the row's context menu, so every confirmation and background path is shared
// with the rest of the page.
func (a *app) branchDetailActions(c *ui.Context, sel branchDetailSelection) {
	tk := theme.TokensOf(c)
	blocked := a.busy || a.loading
	ui.Row(c).FillWidth().Shrink(0).Padding(12, 14).Gap(6).Wrap().AlignItems(ui.Center).
		BorderWidth(1, 0, 0, 0).BorderColor(tk.BorderSecondary).Children(func() {
		switch sel.kind {
		case branchDetailLocal:
			branch := sel.branch
			currentHint := ""
			if branch.Current {
				currentHint = a.t("currentBranchLocked")
			}
			a.detailAction(c, a.t("switch"), currentHint, blocked || branch.Current, false, func() {
				a.switchBranch(branch.Name)
			})
			a.detailAction(c, a.t("mergeIntoCurrent"), currentHint, blocked || branch.Current, false, func() {
				a.askConfirm(a.t("mergeIntoCurrent"), a.tf("mergeCommitMessage", branch.Name),
					a.t("mergeIntoCurrent"), false, func() { a.integrate("merge", branch.Name) })
			})
			a.detailAction(c, a.t("rebaseOnto"), currentHint, blocked || branch.Current, false, func() {
				a.askConfirm(a.t("rebaseOnto"), a.tf("rebaseCommitMessage", branch.Name),
					a.t("rebaseOnto"), false, func() { a.integrate("rebase", branch.Name) })
			})
			remote, _ := a.branchPushTarget(branch)
			pushHint := ""
			if remote == "" {
				pushHint = a.t("noPushRemote")
			}
			a.detailAction(c, a.t("pushBranchAction"), pushHint, blocked || remote == "", false, func() {
				a.confirmPushBranch(branch)
			})
			a.detailAction(c, a.t("renameBranch"), "", blocked, false, func() {
				a.promptRenameBranch(branch.Name)
			})
			a.detailAction(c, a.t("deleteBranch"), currentHint, blocked || branch.Current, true, func() {
				a.confirmDeleteBranch(branch.Name)
			})
			a.detailAction(c, a.t("copyName"), "", false, false, func() {
				a.copyText(branch.Name)
			})
		case branchDetailRemote:
			ref := sel.branch
			// A local branch already tracking this ref is the checkout the
			// reader wants; asking for a name would only create a second branch
			// or fail in git.
			tracker := a.trackingBranch(ref.Name)
			label := a.t("checkoutRemoteBranch")
			hint := ""
			disabled := blocked
			if tracker != "" {
				label = a.tf("switchToBranch", tracker)
				if tracker == a.status.Branch {
					disabled = true
					hint = a.tf("remoteCheckedOutHere", tracker)
				}
			}
			a.detailAction(c, label, hint, disabled, false, func() {
				if tracker != "" {
					a.switchBranch(tracker)
					return
				}
				a.promptCheckoutRemoteBranch(ref.Name, sel.name)
			})
			a.detailAction(c, a.t("deleteRemoteBranch"), "", blocked || sel.remote == "", true, func() {
				a.confirmDeleteRemoteBranch(sel.remote, sel.name)
			})
			a.detailAction(c, a.t("copyName"), "", false, false, func() {
				a.copyText(ref.Name)
			})
		case branchDetailTag:
			tag := sel.tag
			canCheckout := tag.TargetType == "commit" && tag.TargetSHA != ""
			checkoutHint := ""
			if !canCheckout {
				checkoutHint = a.t("tagNotCommit")
			}
			a.detailAction(c, a.t("checkoutTagAction"), checkoutHint, blocked || !canCheckout, false, func() {
				a.checkoutTag(tag)
			})
			// The menu hides the push item without a remote, so the footer
			// does the same instead of showing a button that cannot act.
			if a.tagPushRemote() != "" {
				a.detailAction(c, a.t("pushTag"), "", blocked, false, func() {
					a.confirmPushTag(tag.Name)
				})
			}
			a.detailAction(c, a.t("deleteTagAction"), "", blocked, true, func() {
				a.confirmDeleteTag(tag)
			})
			a.detailAction(c, a.t("copyTagName"), "", false, false, func() {
				a.copyText(tag.Name)
			})
		}
	})
}

// checkoutTag puts HEAD on the commit a tag points at. A lightweight or
// annotated tag that points at a tree or blob has nothing to check out, so
// only a commit target reaches the confirmation.
func (a *app) checkoutTag(tag gitrepo.Tag) {
	if tag.TargetType != "commit" || tag.TargetSHA == "" {
		return
	}
	a.askConfirm(a.t("checkoutCommitTitle"), a.tf("checkoutCommitMessage", shortSHA(tag.TargetSHA)),
		a.t("checkout"), false, func() {
			a.performAs(busyCheckout, a.t("checkoutCommitFailed"), a.t("checkoutCommitSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
				return repo.CheckoutCommit(ctx, tag.TargetSHA)
			})
		})
}
