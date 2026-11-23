package lobedesk

import (
	"context"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

type comparisonRevision struct{ root, sha, name string }

func (r comparisonRevision) label() string {
	if r.name == "" {
		return shortSHA(r.sha)
	}
	return r.name + " · " + shortSHA(r.sha)
}

func (a *app) comparisonRevision(sha, name string) comparisonRevision {
	if a.repo == nil {
		return comparisonRevision{}
	}
	return comparisonRevision{root: a.repo.Root, sha: sha, name: name}
}

type revisionComparison struct {
	base comparisonRevision
	view *comparisonView
}

type comparisonView struct {
	preview                     filePreviewSlot
	target                      comparisonRevision
	find                        *diffFindState
	pane                        ui.Handle
	open, loading               bool
	files                       []gitrepo.ChangedFile
	selected                    int
	patch                       gitrepo.Diff
	patchWidth                  float32
	err                         string
	cancel                      context.CancelFunc
	fileScroll, patchScroll     ui.ScrollState
	fileViewport, patchViewport ui.Rect
}

func (a *app) discardStaleComparison() {
	base := a.comparison.base
	if base.root != "" && (a.repo == nil || a.repo.Root != base.root) {
		a.closeComparison()
		a.comparison = revisionComparison{}
	}
}

func (a *app) closeComparison() {
	if v := a.comparison.view; v != nil {
		if v.cancel != nil {
			v.cancel()
		}
		v.preview.clear()
		v.open = false
		a.comparison.view = nil
	}
}

const compareBaseKey = "set-comparison-base"
const compareTargetKey = "compare-with-base"

func (a *app) comparisonMenuItems(r comparisonRevision) []overlay.MenuItem {
	a.discardStaleComparison()
	disabled := r.sha == "" || a.exclusiveBusy() || a.loading || a.tabSnapshotUnavailable()
	items := []overlay.MenuItem{{Key: compareBaseKey, Label: a.t("setComparisonBase"), Icon: lucide.GitCommitHorizontal(), Disabled: disabled}}
	if base := a.comparison.base; base.root != "" && base.root == r.root {
		items = append(items, overlay.MenuItem{Key: compareTargetKey, Label: a.t("compareWithBase"), Icon: lucide.GitCompareArrows(), Extra: shortSHA(base.sha), Disabled: disabled})
	}
	return items
}

func (a *app) comparisonMenuAction(key string, r comparisonRevision) {
	a.discardStaleComparison()
	if a.repo == nil || a.repo.Root != r.root || r.sha == "" || a.localWatch.closed || a.activeRepositorySession() != a || a.exclusiveBusy() || a.loading || a.tabSnapshotUnavailable() || a.hasPendingDialog(toastMain) {
		return
	}
	switch key {
	case compareBaseKey:
		a.comparison.base = r
		a.queue("success", a.t("comparisonBaseSet"), r.label())
	case compareTargetKey:
		if a.comparison.base.root != r.root || a.comparison.base.sha == "" {
			return
		}
		a.comparison.view = &comparisonView{target: r, open: true, selected: -1}
		a.loadComparison(-1)
	}
	a.invalidate()
}

func (a *app) loadComparison(index int) {
	a.discardStaleComparison()
	v := a.comparison.view
	if v == nil || !v.open || index < -1 || index >= len(v.files) {
		return
	}
	if v.cancel != nil {
		v.cancel()
	}
	ctx, cancel := context.WithCancel(a.ctx)
	v.cancel = cancel
	v.selected, v.loading, v.err = index, true, ""
	v.patch, v.patchScroll, v.find = gitrepo.Diff{}, ui.ScrollState{}, nil
	repo, base, target := a.repo, a.comparison.base.sha, v.target.sha
	var file gitrepo.ChangedFile
	if index >= 0 {
		file = v.files[index]
	}
	go func() {
		var files []gitrepo.ChangedFile
		var patch gitrepo.Diff
		var err error
		if index < 0 {
			files, err = repo.CompareRevisionFiles(ctx, base, target)
		} else {
			patch, err = repo.CompareRevisionFile(ctx, base, target, file)
		}
		a.hopToMain(func() {
			defer cancel()
			if ctx.Err() != nil || a.comparison.view != v || !v.open || a.repo == nil || a.repo.Root != v.target.root || a.localWatch.closed {
				return
			}
			v.loading = false
			if err != nil {
				v.err = err.Error()
			} else if index < 0 {
				v.files = files
				if len(files) > 0 {
					a.loadComparison(0)
				}
			} else {
				v.patch, v.patchWidth = patch, a.diffWidth(patch)
			}
			a.invalidate()
		})
	}()
	a.invalidate()
}

func (a *app) comparisonDialog(c *ui.Context) {
	a.discardStaleComparison()
	v := a.comparison.view
	if v == nil {
		return
	}
	width, height := c.Size()
	panelWidth := min(1100, max(0, width-32))
	tk := theme.TokensOf(c)
	workbenchModal(c.Key(v), &v.open, func() {
		ui.Column(c).Width(panelWidth).Height(min(720, max(0, height-32))).Padding(16).Gap(10).
			Background(tk.BgElevated).Radius(12).Border(1, tk.Border).Children(func() {
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				display.Text(c, a.t("compareRevisions"), display.TextProps{Strong: true, FontSize: scaledFont(15)}).Grow(1)
				if workbenchButton(c, a.t("close"), buttonNormal, false).Clicked() {
					v.open = false
				}
			})
			for _, endpoint := range []struct {
				label string
				rev   comparisonRevision
			}{{a.t("comparisonBase"), a.comparison.base}, {a.t("comparisonTarget"), v.target}} {
				ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
					display.Text(c, endpoint.label, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12)}).Width(scaledFont(48))
					ui.Text(c, endpoint.rev.label()).SingleLine().FontSize(scaledFont(12)).Grow(1).Basis(0).MinWidth(0).Tooltip(endpoint.rev.sha)
				})
			}
			if len(v.files) == 0 {
				if v.loading || v.err != "" {
					a.comparisonReadStatus(c, v)
				} else {
					a.diffPlaceholder(c, lucide.GitCompareArrows(), a.t("comparisonIdentical"))
				}
				return
			}
			ui.Row(c).FillWidth().Grow(1).Basis(0).MinHeight(0).Gap(10).AlignItems(ui.Stretch).Children(func() {
				ui.Column(c).Width(min(340, panelWidth*0.34)).MinWidth(0).Gap(4).Children(func() {
					display.Text(c, a.tf("fileGroupCount", len(v.files)), display.TextProps{Strong: true, FontSize: scaledFont(12)})
					a.comparisonFileList(c, v)
				})
				ui.Box(c).Width(1).Background(tk.BorderSecondary)
				pane := ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(4).TabStop(true).Label(v.files[v.selected].Path).Bind(&v.pane)
				closeFind := func() { v.find = nil; v.pane.Focus(); a.invalidate() }
				if !v.loading && v.err == "" {
					pane.OnShortcut(ui.Cmd, ui.KeyF, func() {
						if v.find == nil {
							v.find = &diffFindState{}
						}
						v.find.focus = true
						a.invalidate()
					})
				}
				if v.find != nil {
					pane.OnShortcut(0, ui.KeyEscape, closeFind)
				}
				pane.Children(func() {
					file := v.files[v.selected]
					header := file.Path
					if file.OriginalPath != "" {
						header = file.OriginalPath + " → " + file.Path
					}
					display.Text(c, header, display.TextProps{Strong: true, FontSize: scaledFont(12)}).SingleLine().FillWidth().Tooltip(header)
					if !v.loading && a.filePreview(c, &v.preview, previewVersions(file, a.comparison.base.sha, v.target.sha), v.patch.Binary, 0) {
						return
					}
					if v.loading || v.err != "" {
						a.comparisonReadStatus(c, v)
					} else {
						a.drawDiffFindBar(c, v.find, func() []diffFindSection {
							if len(v.patch.Lines) == 0 {
								return nil
							}
							return []diffFindSection{{lines: v.patch.Lines}}
						}, closeFind)
						a.comparisonPatch(c, v)
					}
				})
			})
		})
	}).Padding(0)
	if !v.open {
		a.closeComparison()
	}
}

func (a *app) comparisonReadStatus(c *ui.Context, v *comparisonView) {
	if v.loading {
		a.diffPlaceholder(c, lucide.LoaderCircle(), a.t("diffLoading"))
		return
	}
	ui.Column(c).Fill().Center().Padding(12).Gap(8).Children(func() {
		display.Text(c, a.tf("diffFailed", v.err), display.TextProps{Type: display.TextDanger, Rows: 4, FontSize: scaledFont(12)}).FillWidth().TextAlign(ui.Center)
		if workbenchButton(c, a.t("retry"), buttonNormal, false).Clicked() {
			a.loadComparison(v.selected)
		}
	})
}

func (a *app) comparisonFileList(c *ui.Context, v *comparisonView) {
	tk := theme.TokensOf(c)
	scroll := layout.ScrollArea(c, layout.ScrollAreaProps{State: &v.fileScroll}, func() {
		ui.Column(c).FillWidth().Role(ui.RoleList).Children(func() {
			heights := make([]float32, len(v.files))
			for i := range heights {
				heights[i] = fileRowHeight()
			}
			virtualPatchRows(c, v.fileViewport, &v.fileScroll, heights, v.selected, func(c *ui.Context, i int) {
				file := v.files[i]
				selected := v.selected == i
				row := fileRowShell(c, tk, file.Path, selected).PositionInSet(i+1, len(v.files)).Tooltip(file.Path)
				row.Children(func() {
					fileRowPath(c, file.Path, selected)
				})
				if row.Clicked() && !selected {
					a.loadComparison(i)
				}
			})
		})
	}).Fill()
	v.fileViewport = scroll.Bounds()
}

func (a *app) comparisonPatch(c *ui.Context, v *comparisonView) {
	tk := theme.TokensOf(c)
	width := v.find.fitMatch(c, detailFileKey{}, v.patch.Lines, patchWidths{canvas: v.patchWidth}, a.patchFontSize()).canvas
	scroll := layout.ScrollArea(c, layout.ScrollAreaProps{Direction: layout.Both, State: &v.patchScroll}, func() {
		ui.Column(c).Width(max(width, v.patchViewport.W)).AlignSelf(ui.Start).Selectable().Children(func() {
			heights := make([]float32, len(v.patch.Lines))
			for i := range heights {
				heights[i] = a.patchRowHeight()
			}
			virtualPatchRows(c, v.patchViewport, &v.patchScroll, heights, v.find.revealLine(detailFileKey{}), func(c *ui.Context, i int) {
				line := v.patch.Lines[i]
				v.find.lineText(c, diffFindLine{index: i}, diffFindText(line), a.patchFontSize()).
					Height(heights[i]).Shrink(0).TextColor(lineTextColor(tk, line.Kind)).Role(diffCodeRole(line.Kind))
			})
		})
	}).Fill()
	v.patchViewport = scroll.Bounds()
}
