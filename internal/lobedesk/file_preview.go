package lobedesk

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"path"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
	"supergit/internal/gitrepo"
)

type previewKind uint8

const (
	textPreview previewKind = iota
	imagePreview
	binaryPreview
)

func previewFor(pathname string, binary bool) previewKind {
	switch strings.ToLower(path.Ext(pathname)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp":
		return imagePreview
	case ".zip", ".7z", ".rar", ".tar", ".gz", ".bz2", ".xz", ".tgz", ".zst", ".jar", ".apk", ".ipa", ".exe", ".dll", ".so", ".dylib", ".o", ".a", ".dmg", ".pkg", ".msi", ".wasm", ".class", ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".woff", ".woff2", ".ttf", ".otf", ".mp3", ".wav", ".flac", ".mp4", ".mov", ".mkv", ".avi", ".db", ".sqlite", ".psd", ".heic", ".heif", ".avif", ".tif", ".tiff", ".ico":
		return binaryPreview
	}
	if binary {
		return binaryPreview
	}
	return textPreview
}

type previewSide struct{ rev, path string }
type previewPair struct{ before, after previewSide }

func previewVersions(file gitrepo.ChangedFile, base, target string) previewPair {
	old := file.OriginalPath
	if old == "" {
		old = file.Path
	}
	pair := previewPair{previewSide{base, old}, previewSide{target, file.Path}}
	switch {
	case strings.HasPrefix(file.Status, "A") || strings.HasPrefix(file.Status, "?"):
		pair.before.rev = ""
	case strings.HasPrefix(file.Status, "D"):
		pair.after.rev = ""
	}
	return pair
}
func (p previewPair) kind(binary bool) previewKind {
	for _, s := range []previewSide{p.after, p.before} {
		if s.rev != "" && previewFor(s.path, false) == imagePreview {
			return imagePreview
		}
	}
	return previewFor(p.after.path, binary)
}

type previewPicture struct {
	bitmap        *ui.Bitmap
	width, height int
	format        string
	problem       string
}

// Rapid navigation can cancel a read while Go's decoder is still running.
var previewWorkers = make(chan struct{}, 2)

func readPreviewPicture(ctx context.Context, read func() ([]byte, error)) previewPicture {
	select {
	case previewWorkers <- struct{}{}:
		defer func() { <-previewWorkers }()
	case <-ctx.Done():
		return previewPicture{}
	}
	if ctx.Err() != nil {
		return previewPicture{}
	}
	data, err := read()
	if ctx.Err() != nil {
		return previewPicture{}
	}
	return decodePreview(data, err)
}

func decodePreview(data []byte, err error) previewPicture {
	switch {
	case errors.Is(err, gitrepo.ErrPreviewTooLarge):
		return previewPicture{problem: "large"}
	case err != nil:
		return previewPicture{problem: "read"}
	case bytes.HasPrefix(data, []byte("version https://git-lfs.github.com/spec/v1\n")):
		return previewPicture{problem: "lfs"}
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	switch format {
	case "png", "jpeg", "gif", "webp", "bmp":
	default:
		return previewPicture{problem: "decode"}
	}
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return previewPicture{problem: "decode"}
	}
	const maxPixels = 16 << 20
	if cfg.Width > maxPixels/cfg.Height {
		return previewPicture{problem: "large"}
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return previewPicture{problem: "decode"}
	}
	// Keep a bounded screen-sized copy; dimensions below describe the original.
	const maxSide = 1280
	longest := max(cfg.Width, cfg.Height)
	canvas := image.Rect(0, 0, cfg.Width, cfg.Height)
	if longest > maxSide || img.Bounds() != canvas {
		scale := min(longest, maxSide)
		dst := image.NewNRGBA(image.Rect(0, 0, max(1, cfg.Width*scale/longest), max(1, cfg.Height*scale/longest)))
		draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, canvas, draw.Src, nil)
		img = dst
	}
	return previewPicture{bitmap: ui.NewBitmap(img), width: cfg.Width, height: cfg.Height, format: format}
}

type previewKey struct {
	pair              previewPair
	root              string
	epoch, generation uint64
}
type filePreviewSlot struct {
	key           previewKey
	cancel        context.CancelFunc
	before, after previewPicture
	ready         bool
	scroll        ui.ScrollState
}

func (v *filePreviewSlot) clear() {
	if v.cancel != nil {
		v.cancel()
	}
	*v = filePreviewSlot{}
}

func (a *app) previewPictures(v *filePreviewSlot, key previewKey) {
	if v.key == key {
		return
	}
	v.clear()
	v.key = key
	if a.repo == nil || a.ctx == nil {
		return
	}
	repo := a.repo
	ctx, cancel := context.WithCancel(a.ctx)
	v.cancel = cancel
	go func() {
		read := func(side previewSide) previewPicture {
			if side.rev == "" || ctx.Err() != nil {
				return previewPicture{}
			}
			return readPreviewPicture(ctx, func() ([]byte, error) { return repo.ReadPreview(ctx, side.rev, side.path) })
		}
		before, after := read(key.pair.before), read(key.pair.after)
		a.hopToMain(func() {
			if ctx.Err() != nil || v.key != key {
				return
			}
			cancel()
			v.cancel = nil
			v.before, v.after, v.ready = before, after, true
			a.invalidate()
		})
	}()
}

func (a *app) drawPicture(c *ui.Context, pic previewPicture) {
	tk := theme.TokensOf(c)
	if pic.bitmap == nil {
		label := a.paletteText("Unable to decode this image.", "无法解码此图片。")
		switch pic.problem {
		case "read":
			label = a.paletteText("This version could not be read.", "无法读取此版本。")
		case "large":
			label = a.paletteText("Image exceeds the preview limit.", "图片超出预览大小限制。")
		case "lfs":
			label = a.paletteText("This version contains a Git LFS pointer.", "此版本仅包含 Git LFS 指针。")
		}
		display.Text(c, label, display.TextProps{Type: display.TextSecondary}).FillWidth()
		return
	}
	caption := fmt.Sprintf("%s · %d × %d", strings.ToUpper(pic.format), pic.width, pic.height)
	if pic.format == "gif" {
		caption += " · " + a.paletteText("First frame", "首帧")
	}
	ui.Column(c).FillWidth().Height(210).Padding(4).Background(tk.FillQuaternary).Children(func() {
		ui.Image(c, pic.bitmap).Fill().Fit(ui.Contain).Label(caption)
	})
	display.Text(c, caption, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).FillWidth()
}

func (a *app) filePreview(c *ui.Context, slot *filePreviewSlot, p previewPair, binary bool, epoch uint64) bool {
	kind := p.kind(binary)
	if kind != imagePreview && slot.key.root != "" {
		slot.clear()
	}
	switch kind {
	case textPreview:
		return false
	case binaryPreview:
		display.Text(c, a.paletteText("Binary file · No content preview", "二进制文件 · 无内容预览"), display.TextProps{Type: display.TextSecondary}).Padding(12).FillWidth()
		return true
	}
	if a.repo == nil {
		return false
	}
	key := previewKey{pair: p, root: a.repo.Root, epoch: epoch}
	if p.before.rev == gitrepo.PreviewIndex || p.after.rev == gitrepo.PreviewIndex || p.after.rev == gitrepo.PreviewWorktree {
		key.generation = a.loadSeq
	}
	a.previewPictures(slot, key)
	if !slot.ready {
		display.Text(c, a.paletteText("Loading image…", "正在读取图片…"), display.TextProps{Type: display.TextSecondary}).Padding(12)
		return true
	}
	layout.ScrollArea(c, layout.ScrollAreaProps{State: &slot.scroll}, func() {
		ui.Row(c).FillWidth().Padding(10).Gap(12).Wrap().AlignItems(ui.Start).Children(func() {
			side := func(source previewSide, title string, pic previewPicture) {
				if source.rev == "" {
					return
				}
				ui.Column(c).Basis(180).Grow(1).MinWidth(0).MaxWidthPercent(100).Gap(5).Children(func() {
					display.Text(c, title, display.TextProps{Strong: true, FontSize: scaledFont(12)}).Tooltip(source.path)
					a.drawPicture(c, pic)
				})
			}
			side(p.before, a.paletteText("Before", "改前"), slot.before)
			side(p.after, a.paletteText("After", "改后"), slot.after)
		})
	}).FillWidth().Grow(1).Basis(0).MinHeight(0)
	return true
}

func (a *app) worktreePreview(c *ui.Context, entry gitrepo.Entry) bool {
	file := gitrepo.ChangedFile{Path: entry.Path, Status: string(entry.Worktree)}
	base, target := gitrepo.PreviewIndex, gitrepo.PreviewWorktree
	if a.selectedStaged {
		base, target = a.status.Head, gitrepo.PreviewIndex
		file.Status = string(entry.Index)
	}
	if file.Status == "R" || file.Status == "C" {
		file.OriginalPath = entry.OriginalPath
	}
	return a.filePreview(c, &a.preview, previewVersions(file, base, target), a.diff.Binary, a.diffSeq)
}

func (a *app) historyFilePreview(c *ui.Context) bool {
	pathname := a.detailTabs().Active.File.Path
	for i, file := range a.commitFiles {
		if file.Path != pathname {
			continue
		}
		target := a.selectedSHA
		var parents []string
		if stash, ok := a.selectedStashObject(); ok {
			target = stash.SHA
			parents = strings.Fields(stash.ParentSHAs)
			if i >= a.stashTrackedCount && len(parents) > 2 {
				target = parents[2]
				parents = nil
			}
		} else if commit, ok := a.selectedCommit(); ok {
			parents = commit.Parents
		}
		var base string
		if len(parents) > 0 {
			base = parents[0]
		}
		var binary bool
		for _, section := range a.commitSections {
			if section.Path == pathname && section.Binary {
				binary = true
				break
			}
		}
		return a.filePreview(c, &a.preview, previewVersions(file, base, target), binary, 0)
	}
	return false
}

const previewSummaryHeight float32 = 30

func (a *app) previewSummary(c *ui.Context, key detailFileKey, kind previewKind) {
	ui.Row(c).FillWidth().Height(previewSummaryHeight).AlignItems(ui.Center).PaddingX(8).Children(func() {
		if kind == imagePreview {
			if workbenchButton(c, a.paletteText("View image", "查看图片"), buttonNormal, false).Clicked() {
				a.openDetailFile(key)
			}
		} else {
			display.Text(c, a.paletteText("Binary file · No content preview", "二进制文件 · 无内容预览"), display.TextProps{Type: display.TextSecondary})
		}
	})
}
