package lobedesk

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

// checkoutBranchState holds the commit whose branches the picker is offering.
// It exists because a commit can sit on many branches: the context menu caps
// its branch rows, and the ones past the cap are chosen here with a search box
// rather than in a menu that would grow past the window.
type checkoutBranchState struct {
	root    string
	open    bool
	sha     string
	subject string
	query   string
	// cursor is the highlighted row in the filtered list, or -1 while no
	// branch can be checked out. The search box keeps the keyboard, so Up/Down
	// move the highlight under the typing hands and Enter takes that row.
	cursor int
	// cursorQuery is the query the cursor was placed for. A new query puts the
	// highlight back on the first branch that can be checked out, so typing
	// narrows the list without the highlight staying on a hidden row.
	cursorQuery string
	// cursorPlaced tells the first frame from a query that happens to be
	// empty already, so opening always places the highlight.
	cursorPlaced bool
	// reveal asks the next frame to scroll the highlight into view. Scrolling
	// every frame would fight the reader's own wheel, so it is set only when
	// the highlight moves.
	reveal bool
}

func (a *app) discardStaleCheckoutBranch() bool {
	if a.checkoutBranch.root != "" && (a.repo == nil || a.repo.Root != a.checkoutBranch.root) {
		a.checkoutBranch = checkoutBranchState{}
		return true
	}
	return false
}

func (a *app) openCheckoutBranchDialog(commit gitrepo.Commit) {
	a.discardStaleCheckoutBranch()
	if a.checkoutBranch.open {
		a.queue("warning", a.t("promptPending"), "")
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
	a.checkoutBranch = checkoutBranchState{
		root:    root,
		open:    true,
		sha:     commit.SHA,
		subject: commit.Subject,
		reveal:  true,
		// The first frame places the cursor, once the query is known.
		cursor:      -1,
		cursorQuery: "",
	}
	a.invalidate()
}

// filterBranchesAt returns the branches whose name contains query, in the order
// Git listed them. The rows and Enter's first pick read the same result, so the
// key and the top row never disagree.
// stepCheckoutCursor moves the highlight to the next branch that can be
// checked out, wrapping at either end so a short list never traps the keys.
// The current branch is skipped: pressing Enter on it would do nothing.
func stepCheckoutCursor(matches []gitrepo.Branch, cursor, delta int) int {
	return stepPickerCursor(len(matches), cursor, delta, func(index int) bool {
		return !matches[index].Current
	})
}

// checkoutBranchAt is the branch Enter takes: the highlighted row when it can
// be checked out, and otherwise the first match that can, so Enter stays one
// key away even before the arrows have been touched.
func checkoutBranchAt(matches []gitrepo.Branch, cursor int) (string, bool) {
	index := pickerCursorAt(len(matches), cursor, func(index int) bool {
		return !matches[index].Current
	})
	if index < 0 {
		return "", false
	}
	return matches[index].Name, true
}

func filterBranchesAt(branches []gitrepo.Branch, query string) []gitrepo.Branch {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return branches
	}
	var result []gitrepo.Branch
	for _, branch := range branches {
		if strings.Contains(strings.ToLower(branch.Name), q) {
			result = append(result, branch)
		}
	}
	return result
}

func (a *app) checkoutBranchDialog(c *ui.Context) {
	a.checkoutBranchDialogFor(c, toastMain)
}

func (a *app) checkoutBranchDialogFor(c *ui.Context, owner toastOwner) {
	if owner != toastMain {
		return
	}
	a.discardStaleCheckoutBranch()
	if !a.checkoutBranch.open {
		return
	}
	tk := theme.TokensOf(c)
	branches := a.branchesAt(a.checkoutBranch.sha)
	matches := filterBranchesAt(branches, a.checkoutBranch.query)
	if placePicker(&a.checkoutBranch.cursor, &a.checkoutBranch.cursorQuery, &a.checkoutBranch.cursorPlaced,
		a.checkoutBranch.query, func() int { return stepCheckoutCursor(matches, -1, 1) }) {
		a.checkoutBranch.reveal = true
	}
	// A refresh can shorten the list under a highlight that was already
	// placed; put it back on a branch that still exists and scroll to it, or
	// Enter would act on a row the reader cannot see.
	if repairPicker(&a.checkoutBranch.cursor, len(matches), func() int { return stepCheckoutCursor(matches, -1, 1) }) {
		a.checkoutBranch.reveal = true
	}
	chosen := ""
	workbenchModal(c, &a.checkoutBranch.open, func() {
		ui.Column(c).Width(420).Padding(16).Gap(10).
			Background(tk.BgElevated).Radius(12).Border(1, tk.Border).
			Children(func() {
				display.Text(c, a.t("checkoutBranchesTitle"), display.TextProps{Strong: true, FontSize: scaledFont(14)})
				display.Text(c, a.tf("checkoutBranchesMessage", shortSHA(a.checkoutBranch.sha), a.checkoutBranch.subject),
					display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12), Rows: 2})
				// Enter takes the highlighted branch, or the first one that is
				// not already checked out when no row is marked, so a checkout
				// is one key away.
				search := a.pickerSearchBar(c, &a.checkoutBranch.query, a.t("filterBranches"), func() {
					if name, ok := checkoutBranchAt(matches, a.checkoutBranch.cursor); ok {
						chosen = name
					}
				}, func(delta int) {
					a.checkoutBranch.cursor = stepCheckoutCursor(matches, a.checkoutBranch.cursor, delta)
					a.checkoutBranch.reveal = true
				})
				// The box holds the keyboard for as long as the dialog is
				// open, so Tab and row clicks cannot strand the arrows and the
				// typing; asking each frame outlasts a focus that moves.
				search.Focus()
				var active ui.Element
				hasActive := false
				pickerRows(c, func() {
					for index, branch := range matches {
						highlighted := index == a.checkoutBranch.cursor && !branch.Current
						row := a.checkoutBranchRow(c, branch, index+1, len(matches), highlighted,
							func() { chosen = branch.Name })
						// The search box keeps the keyboard; the row the arrows
						// are on is its active descendant, so assistive
						// technology follows the highlight without the row
						// claiming to be the chosen one.
						if highlighted {
							active, hasActive = row, true
						}
					}
					switch {
					case len(matches) == 0:
						display.Text(c, a.t("searchNoBranches"), display.TextProps{
							Type: display.TextSecondary, FontSize: scaledFont(11),
						}).Padding(6, 8)
					default:
						// A filter that leaves only the checked-out branch
						// has no choice to offer; say so instead of leaving
						// the arrows and Enter silently dead.
						if _, ok := checkoutBranchAt(matches, -1); !ok {
							display.Text(c, a.t("checkoutNoOtherBranch"), display.TextProps{
								Type: display.TextSecondary, FontSize: scaledFont(11), Rows: 2,
							}).Padding(6, 8)
						}
					}
				})
				if hasActive {
					search.ActiveDescendant(active)
				}
			})
	})
	if chosen != "" {
		sha := a.checkoutBranch.sha
		a.checkoutBranch = checkoutBranchState{}
		a.switchBranchAt(chosen, sha)
		return
	}
	if !a.checkoutBranch.open {
		// Cancelled by Escape or by the modal's backdrop.
		a.checkoutBranch = checkoutBranchState{}
	}
}

// checkoutBranchRow draws one branch the picker offers. The checked-out branch
// stays in the list, dimmed and labelled, so the reader can see that it is not
// a choice without wondering why it is missing.
func (a *app) checkoutBranchRow(c *ui.Context, branch gitrepo.Branch, pos, size int, highlighted bool, choose func()) ui.Element {
	tk := theme.TokensOf(c)
	row := pickerRowBase(c, branch.Name, pos, size, false)
	switch {
	case branch.Current:
		// Not a choice: the commit is already checked out. Say why instead
		// of offering an option that does nothing when activated.
		row.AccessDisabled(true).Description(a.t("current"))
		row.Cursor(ui.CursorDefault)
	case highlighted:
		row.Background(selectionBg(tk, tk.BgElevated))
	case row.Hovered():
		row.Background(tk.FillTertiary.Over(tk.BgElevated))
	}
	// Reveal the highlight once when it moves, not every frame: a scroll that
	// re-centres on each draw would pull the list back from the reader's wheel.
	pickerReveal(row, highlighted, &a.checkoutBranch.reveal)
	row.Children(func() {
		ui.Icon(c, lucide.GitBranch()).FontSize(scaledFont(12)).TextColor(tk.TextTertiary)
		display.Text(c, branch.Name, display.TextProps{FontSize: scaledFont(12)}).SingleLine().Grow(1).MinWidth(0)
		if branch.Current {
			ui.Text(c, a.t("current")).FontSize(scaledFont(10)).TextColor(tk.TextSecondary)
		}
	})
	row.Tooltip(branch.Name)
	if !branch.Current && row.Clicked() {
		choose()
	}
	return row
}
