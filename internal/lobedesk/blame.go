package lobedesk

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

// canBlame rejects workspace entries whose content is not in HEAD: an
// untracked file, a file newly added to the index (HEAD has no version of it
// to walk), a deleted file, or a submodule pointer has no history for git
// blame to show.
func canBlame(entry gitrepo.Entry) bool {
	if entry.IsSubmodule() {
		return false
	}
	if entry.Index == '?' && entry.Worktree == '?' {
		return false
	}
	if entry.Index == 'A' {
		return false
	}
	return entry.Worktree != 'D'
}

// workspaceBlameLabel names the blame action, saying why it is disabled for a
// file with nothing to read: a dimmed item with no reason reads as a broken
// menu. The commit view has its own wording for its own reasons.
func (a *app) workspaceBlameLabel(entry gitrepo.Entry) string {
	if !canBlame(entry) {
		return a.t("blameUnavailableFile")
	}
	return a.t("blame")
}

func (a *app) openBlame(path string) {
	a.openBlameAt(path, "")
}

func (a *app) clearBlame() {
	a.blameSeq++
	a.blameRoot = ""
	a.blameOpen = false
	a.blamePath = ""
	a.blameRev = ""
	a.blame = nil
	a.blameErr = ""
	a.blameLoading = false
	a.blameIgnoreWS = false
}

func (a *app) discardStaleBlame() bool {
	if a.blameRoot != "" && (a.repo == nil || a.repo.Root != a.blameRoot) {
		a.clearBlame()
		return true
	}
	return false
}

// openBlameAt blames rev's version of path; an empty rev reads the worktree.
func (a *app) openBlameAt(path, rev string) {
	a.discardStaleBlame()
	if a.repo == nil || path == "" {
		return
	}
	if a.blameOpen {
		a.queue("warning", a.t("confirmationPending"), "")
		a.invalidate()
		return
	}
	if a.rejectPendingDialog(toastMain) {
		return
	}
	a.blameRoot = a.repo.Root
	a.blameOpen = true
	a.blamePath = path
	a.blameRev = rev
	a.blame = nil
	a.blameErr = ""
	a.blameIgnoreWS = false
	a.loadBlame()
	a.invalidate()
}

func (a *app) loadBlame() {
	if a.discardStaleBlame() || !a.blameOpen || a.repo == nil || a.blamePath == "" {
		return
	}
	a.blameSeq++
	seq := a.blameSeq
	repo, path, rev := a.repo, a.blamePath, a.blameRev
	ignoreWS := a.blameIgnoreWS
	a.blameLoading = true
	a.blameErr = ""
	go func() {
		lines, err := repo.BlameAt(a.ctx, rev, path, ignoreWS)
		a.hopToMain(func() {
			if !a.blameOpen || a.repo == nil || a.repo.Root != repo.Root || seq != a.blameSeq || a.blamePath != path || a.blameRev != rev {
				return
			}
			a.blameLoading = false
			if err != nil {
				a.blameErr = err.Error()
			} else {
				a.blame = lines
			}
			a.invalidate()
		})
	}()
}

func (a *app) blameDialog(c *ui.Context) {
	a.blameDialogFor(c, toastMain)
}

func (a *app) blameDialogFor(c *ui.Context, owner toastOwner) {
	if owner != toastMain {
		return
	}
	a.discardStaleBlame()
	if !a.blameOpen {
		if a.blameRoot != "" {
			a.clearBlame()
		}
		return
	}
	tk := theme.TokensOf(c)
	width, height := c.Size()
	ui.Modal(c, &a.blameOpen, func() {
		ui.Column(c).Width(min(900, max(0, width-40))).Height(min(620, max(0, height-40))).Padding(16).Gap(10).
			Background(tk.BgElevated).Radius(tk.BorderRadiusLG).Border(1, tk.Border).Children(func() {
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, lucide.FileSearch()).FontSize(scaledFont(15)).TextColor(tk.TextSecondary)
				display.Text(c, a.t("blameTitle"), display.TextProps{Strong: true, FontSize: scaledFont(14)})
				display.Text(c, a.blamePath, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12)}).SingleLine().
					Grow(1).Basis(0).MinWidth(0)
				// A revision says which version of the file these lines
				// describe; the worktree blame has none.
				if a.blameRev != "" {
					display.Text(c, "@"+shortSHA(a.blameRev), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).Shrink(0)
				}
				display.Text(c, a.blameLinesLabel(len(a.blame)), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).Shrink(0)
				toggle := styleWorkbenchButton(c, ui.ButtonBase(c), buttonNormal, false, a.blameIgnoreWS).
					Role(ui.RoleToggleButton).Label(a.t("blameIgnoreWhitespace")).Checked(a.blameIgnoreWS).Shrink(0)
				toggle.Children(func() {
					ui.Text(c, a.t("blameIgnoreWhitespace")).SingleLine()
				})
				if toggle.Clicked() {
					a.blameIgnoreWS = !a.blameIgnoreWS
					a.loadBlame()
					a.invalidate()
				}
				close := workbenchIconButton(c, lucide.X(), a.t("close"), buttonNormal, false)
				if close.Clicked() {
					a.blameOpen = false
				}
			})
			switch {
			case a.blameLoading:
				a.diffPlaceholder(c, lucide.LoaderCircle(), a.t("diffLoading"))
			case a.blameErr != "":
				ui.Column(c).Fill().Center().Gap(8).Padding(24).Children(func() {
					display.Text(c, a.blameErr, display.TextProps{Type: display.TextDanger, FontSize: scaledFont(12), Rows: 4}).
						MaxWidth(600).TextAlign(ui.Center)
					if workbenchButton(c, a.t("retry"), buttonNormal, false).Clicked() {
						a.loadBlame()
					}
				})
			default:
				a.blameRows(c)
			}
		})
	}).Padding(0) // The inner panel owns padding; Modal still supplies dismissal and shadow.
	if !a.blameOpen {
		a.clearBlame()
	}
}

// blameRowName is the row's accessible name: the commit the line comes from
// and the line's number, so a screen reader announces where a row leads
// instead of reading its whole source line first. Uncommitted lines have no
// commit to name, so they say that instead of a row of zeros.
func (a *app) blameRowName(line gitrepo.BlameLine) string {
	ref := shortSHA(line.SHA)
	if strings.Trim(line.SHA, "0") == "" {
		ref = a.t("uncommitted")
	}
	return a.tf("blameRowName", ref, line.Line)
}

// blameRowDescription carries what the label cannot: the button's name
// replaces the row's text for assistive technology, so the author, the date
// and the blamed source line would otherwise disappear from the row. They
// stay available as its description.
func (a *app) blameRowDescription(line gitrepo.BlameLine) string {
	parts := make([]string, 0, 3)
	if author := strings.TrimSpace(line.Author); author != "" {
		parts = append(parts, author)
	}
	if date := formatDateTime(line.Date); date != "" {
		parts = append(parts, date)
	}
	if text := strings.TrimSpace(line.Text); text != "" {
		parts = append(parts, text)
	}
	return strings.Join(parts, " · ")
}

func (a *app) blameCommitAction(sha string) func() {
	root, seq := a.blameRoot, a.blameSeq
	return func() {
		if a.blameRoot == root && a.blameSeq == seq {
			a.openBlameCommit(sha)
		}
	}
}

// openBlameCommit leaves the dialog and shows the blamed commit in history.
func (a *app) openBlameCommit(sha string) {
	if a.discardStaleBlame() || !a.blameOpen || sha == "" || strings.Trim(sha, "0") == "" {
		return
	}
	a.clearBlame()
	a.openPage("history")
	a.showCommit(sha)
	a.invalidate()
}

func (a *app) blameRows(c *ui.Context) {
	tk := theme.TokensOf(c)
	width := float32(120)
	for _, line := range a.blame {
		if n := float32(len([]rune(line.Text))) * 7.0; n > width {
			width = n
		}
	}
	width += 380
	layout.ScrollArea(c, layout.ScrollAreaProps{Direction: layout.Both}, func() {
		ui.Column(c).Width(width).AlignSelf(ui.Start).Shrink(0).Children(func() {
			for _, line := range a.blame {
				uncommitted := strings.Trim(line.SHA, "0") == ""
				// The row is a button: it takes the keyboard focus and the
				// menu key, and its accessible name says which commit and line
				// it shows without reading the source line aloud. The name
				// replaces the text inside for assistive technology, so the
				// columns still draw as before.
				row := ui.ButtonBase(c).Label(a.blameRowName(line)).Description(a.blameRowDescription(line)).
					Row().Justify(ui.Start).
					Width(width).Height(20).Shrink(0).Padding(0, 6).Gap(8).AlignItems(ui.Center)
				// The row ground is the dialog's surface, tinted on hover. The
				// hover tint is translucent, so it is flattened here: its alpha
				// alone would leave the row's real color unknown to the ink
				// below, and the date and line number are tertiary text.
				ground := tk.BgElevated
				if row.Hovered() {
					ground = tk.FillTertiary.Over(tk.BgElevated)
					row.Background(ground)
				}
				gutter := gutterInk(tk, ground)
				if !uncommitted {
					sha := line.SHA
					jump := a.blameCommitAction(sha)
					row = row.OnClick(jump).Tooltip(a.t("jumpToCommit"))
					// A focused button takes Enter for itself, which would only
					// report a click nobody listens to; the row's shortcut runs
					// the same jump the pointer does.
					row.OnShortcut(0, ui.KeyEnter, jump)
				}
				row.Children(func() {
					sha := shortSHA(line.SHA)
					shaColor := tk.TextSecondary
					if uncommitted {
						sha = a.t("uncommitted")
						shaColor = tk.Warning
					}
					ui.Text(c, sha).Font("Menlo").FontSize(scaledFont(9.5)).Width(78).Shrink(0).
						SingleLine().TextColor(shaColor)
					author := line.Author
					if author == "" || uncommitted {
						author = "—"
					}
					authorText := ui.Text(c, author).FontSize(scaledFont(10)).Width(110).Shrink(0).SingleLine().
						TextColor(tk.TextSecondary)
					if !uncommitted && line.Subject != "" {
						authorText.Tooltip(line.Subject)
					}
					ui.Text(c, formatDateTime(line.Date)).Font("Menlo").FontSize(scaledFont(9.5)).Width(118).Shrink(0).
						TextColor(gutter)
					ui.Text(c, itoa(line.Line)).Font("Menlo").FontSize(scaledFont(9.5)).Width(38).Shrink(0).
						TextAlign(ui.End).TextColor(gutter)
					// The code text is not selectable: the whole row is the click
					// target that jumps to its commit, and MyGo lets no text
					// inside a clickable element be selected. Making the code
					// selectable means moving the jump to the gutter columns.
					ui.Text(c, strings.ReplaceAll(line.Text, "\t", "    ")).Font("Menlo").FontSize(scaledFont(11)).
						SingleLine().TextColor(tk.Text)
				})
				a.blameMenu(c, row, line)
			}
		})
	}).Grow(1).Basis(0).MinHeight(0)
}

// blameMenu offers the actions a blamed row supports: jump to its commit,
// copy the full SHA, and open the file's history with this line's changes.
func (a *app) blameMenu(c *ui.Context, row ui.Element, line gitrepo.BlameLine) {
	uncommitted := strings.Trim(line.SHA, "0") == ""
	items := []overlay.MenuItem{
		{Key: "view", Label: a.t("jumpToCommit"), Icon: lucide.GitCommit(), Disabled: uncommitted},
		{Key: "copy-sha", Label: a.t("copySHA"), Icon: lucide.Copy(), Disabled: uncommitted},
		{Divider: true},
		{Key: "history", Label: a.t("fileHistory"), Icon: lucide.History()},
	}
	chosen := overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: items})
	if chosen != "" && (a.discardStaleBlame() || !a.blameOpen) {
		return
	}
	switch chosen {
	case "view":
		a.openBlameCommit(line.SHA)
	case "copy-sha":
		a.copyText(line.SHA)
	case "history":
		path := a.blamePath
		a.clearBlame()
		a.showFileHistory(path)
	}
}
