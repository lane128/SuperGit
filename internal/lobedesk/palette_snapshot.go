package lobedesk

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func (a *app) snapshotRevision() string {
	if a.page == "history" && !a.worktreeSelected {
		if a.selectedStash != "" {
			return a.selectedStash
		}
		if a.selectedSHA != "" {
			return a.selectedSHA
		}
	}
	return a.status.Head
}
func paletteSnapshotUnavailable(a *app) string {
	if a.snapshotRevision() == "" {
		return a.t("commandPaletteNeedHead")
	}
	return ""
}
func (a *app) startFileHistoryTips() []string {
	if a.fileHistoryRevision == "" {
		return nil
	}
	return []string{a.fileHistoryRevision}
}

// A read owns one picker generation. Closing the bar, replacing the view or
// switching repositories prevents its result from installing a new surface.
func readHistoryPalette[T any](a *app, title string, read func(context.Context, *gitrepo.Repository) (T, error), done func(T)) {
	if a.repo == nil || a.ctx == nil {
		return
	}
	loading := a.paletteText("Loading…", "正在读取…")
	a.choosePalette(title, loading, []paletteChoice{{id: "loading", label: loading, reason: loading}}, func(paletteChoice) {})
	gen, epoch, repo, ctx := a.commandPalette.generation, a.loadSeq, a.repo, a.ctx
	go func() {
		value, err := read(ctx, repo)
		a.hopToMain(func() {
			if a.repo != repo || a.loadSeq != epoch || ctx.Err() != nil || !a.commandPalette.open || a.commandPalette.generation != gen {
				return
			}
			query := a.commandPalette.query
			a.finishPaletteStep(gen, func() {
				// Replace the loading placeholder at the same depth. Back
				// returns to the original picker, and actionID remains owned.
				a.commandPalette.step = a.commandPalette.step.parent
				if err != nil {
					a.queue("error", title, err.Error())
					return
				}
				done(value)
				if a.commandPalette.open {
					a.commandPalette.query = query
				}
			})
			if a.commandPalette.open && a.commandPalette.generation == gen {
				// Availability changed while Git was reading.
				a.commandPalette = commandPaletteState{}
			}
			a.invalidate()
		})
	}()
}

func (a *app) openSnapshotFilePicker(mode string) {
	rev := a.snapshotRevision()
	if rev == "" {
		return
	}
	if mode == "tree" {
		a.openSnapshotDirectory(rev, rev, "")
		return
	}
	title := a.t("fileHistory")
	if mode == "blame" {
		title = a.t("blame")
	}
	title += " · " + shortSHA(rev)
	worktree := a.worktreeSelected && mode == "blame"
	readHistoryPalette(a, title, func(ctx context.Context, r *gitrepo.Repository) ([]gitrepo.SnapshotFile, error) {
		return r.SnapshotFiles(ctx, rev)
	}, func(files []gitrepo.SnapshotFile) {
		rows := make([]paletteChoice, 0, len(files))
		unavailableBlame := make(map[string]bool)
		if worktree {
			for _, entry := range a.status.Entries {
				if !canBlame(entry) {
					unavailableBlame[entry.Path] = true
				}
			}
		}
		for _, f := range files {
			row := paletteChoice{id: f.Path, label: f.Path, sha: f.OID}
			if f.Kind == "commit" {
				row.detail = a.paletteText("Submodule", "子模块")
				if mode == "blame" {
					row.reason = a.t("blameUnavailableFile")
				}
			}
			if unavailableBlame[f.Path] {
				row.reason = a.t("blameUnavailableFile")
			}
			rows = append(rows, row)
		}
		if len(rows) == 0 {
			empty := a.paletteText("This tree is empty", "此文件树为空")
			rows = append(rows, paletteChoice{id: "empty", label: empty, reason: empty})
		}
		a.choosePalette(title, a.paletteText("Search file paths", "搜索文件路径"), rows, func(row paletteChoice) {
			switch mode {
			case "blame":
				if worktree {
					a.openBlame(row.id)
				} else {
					a.openBlameAt(row.id, rev)
				}
			case "history":
				a.showFileHistoryAt(row.id, rev)
			}
		})
	})
}

func (a *app) openSnapshotDirectory(rev, oid, prefix string) {
	title := a.paletteText("View Tree", "查看文件树") + " · " + shortSHA(rev)
	if prefix != "" {
		title += " / " + prefix
	}
	readHistoryPalette(a, title, func(ctx context.Context, r *gitrepo.Repository) ([]gitrepo.SnapshotFile, error) {
		return r.SnapshotDirectory(ctx, oid)
	}, func(entries []gitrepo.SnapshotFile) {
		rows := make([]paletteChoice, 0, len(entries))
		objects := make(map[string]gitrepo.SnapshotFile, len(entries))
		for _, entry := range entries {
			row := paletteChoice{id: entry.Path, label: entry.Path, sha: entry.OID}
			switch entry.Kind {
			case "tree":
				row.label += "/"
			case "commit":
				row.detail = a.paletteText("Submodule", "子模块")
			}
			rows = append(rows, row)
			if prefix != "" {
				entry.Path = prefix + "/" + entry.Path
			}
			objects[row.id] = entry
		}
		if len(rows) == 0 {
			empty := a.paletteText("This directory is empty", "此目录为空")
			rows = append(rows, paletteChoice{id: "empty", label: empty, reason: empty})
		}
		a.choosePalette(title, a.paletteText("Search this directory", "搜索当前目录"), rows, func(row paletteChoice) {
			entry := objects[row.id]
			if entry.Kind == "tree" {
				a.openSnapshotDirectory(rev, entry.OID, entry.Path)
				return
			}
			a.openSnapshotFile(rev, entry)
		})
	})
}

type snapshotFileView struct {
	picture               *previewPicture
	parent                *paletteStep
	actionID              string
	epoch                 uint64
	open                  bool
	root, rev, path, text string
	scroll                ui.ScrollState
}

func (a *app) openSnapshotFile(rev string, f gitrepo.SnapshotFile) {
	if a.repo == nil {
		return
	}
	parent, id, epoch := a.commandPalette.step, a.commandPalette.actionID, a.loadSeq
	show := func(body string) {
		a.snapshotView = snapshotFileView{open: true, root: a.repo.Root, rev: rev, path: f.Path, text: body, parent: parent, actionID: id, epoch: epoch}
		a.invalidate()
	}
	if f.Kind == "commit" {
		show(a.paletteText("Submodule", "子模块") + " · " + f.OID)
		return
	}
	switch previewFor(f.Path, false) {
	case imagePreview:
		readHistoryPalette(a, f.Path, func(ctx context.Context, r *gitrepo.Repository) (previewPicture, error) {
			return readPreviewPicture(ctx, func() ([]byte, error) { return r.BlobPreview(ctx, f.OID) }), nil
		}, func(pic previewPicture) { show(""); a.snapshotView.picture = &pic })
		return
	case binaryPreview:
		show(fmt.Sprintf(a.paletteText("Binary file (%d bytes)", "二进制文件（%d 字节）"), f.Size))
		return
	}
	if f.Size > 1<<20 {
		show(fmt.Sprintf(a.paletteText("File is too large to preview (%d bytes).", "文件过大，无法预览（%d 字节）。"), f.Size))
		return
	}
	readHistoryPalette(a, f.Path, func(ctx context.Context, r *gitrepo.Repository) (string, error) { return r.SnapshotBlob(ctx, f.OID) }, func(body string) {
		if !utf8.ValidString(body) || strings.ContainsRune(body, '\x00') {
			body = fmt.Sprintf(a.paletteText("Binary file (%d bytes)", "二进制文件（%d 字节）"), f.Size)
		}
		show(body)
	})
}
func (a *app) discardStaleSnapshotFile() {
	if a.snapshotView.open && (a.repo == nil || a.repo.Root != a.snapshotView.root) {
		a.snapshotView = snapshotFileView{}
	}
}
func (a *app) snapshotFileDialog(c *ui.Context) {
	a.discardStaleSnapshotFile()
	v := &a.snapshotView
	if !v.open {
		return
	}
	width, height := c.Size()
	tk := theme.TokensOf(c)
	workbenchModal(c, &v.open, func() {
		ui.Column(c).Width(min(900, max(0, width-40))).Height(min(620, max(0, height-40))).Padding(16).Gap(10).Background(tk.BgElevated).Radius(12).Border(1, tk.Border).Children(func() {
			ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
				display.Text(c, v.path, display.TextProps{Strong: true, FontSize: scaledFont(14)}).SingleLine().Grow(1).Basis(0).MinWidth(0).Tooltip(v.path)
				display.Text(c, shortSHA(v.rev), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12)})
				if v.parent != nil && v.epoch == a.loadSeq && workbenchButton(c, a.paletteText("Back", "返回"), buttonNormal, false).Clicked() {
					v.open = false
					a.commandPalette.actionID = v.actionID
					a.showPaletteStep(v.parent)
				}
				if workbenchButton(c, a.t("close"), buttonNormal, false).Clicked() {
					v.open = false
				}
			})
			layout.ScrollArea(c, layout.ScrollAreaProps{State: &v.scroll}, func() {
				if v.picture != nil {
					ui.Column(c).FillWidth().Gap(6).Children(func() { a.drawPicture(c, *v.picture) })
					return
				}
				ui.Text(c, v.text).Selectable().Font(a.patchFontFamily()).FontSize(a.patchFontSize()).FillWidth().MinWidth(0)
			}).Fill()
		})
	})
	if !v.open {
		*v = snapshotFileView{}
	}
}
