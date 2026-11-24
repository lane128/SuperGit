package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestHeadCapsuleRendersWhiteInkOnSelectedLightAndDarkRows(t *testing.T) {
	for _, palette := range []string{themePaletteSuperGit, themePaletteCatppuccin} {
		for _, dark := range []bool{false, true} {
			cfg := themeConfig(palette)
			tk := cfg.Tokens(dark)
			rowBg := tk.PrimaryBg
			style := headCapsuleColors(tk, rowBg)
			if style.Ink != ui.RGB(255, 255, 255) {
				t.Fatalf("palette=%s dark=%v HEAD ink=%v; want white", palette, dark, style.Ink)
			}
			if got := contrastRatio(style.Ink, style.Surface); got < 4.5 {
				t.Fatalf("palette=%s dark=%v HEAD contrast=%.2f; want >= 4.5", palette, dark, got)
			}

			// Render the pill onto a selected row and verify the glyphs paint bright white,
			// not dark tk.Text over solid blue.
			tt := ui.NewTester(func(c *ui.Context) {
				c.SetTheme(cfg.Theme(dark))
				ui.Row(c).Width(200).Height(44).Padding(8).Background(rowBg).AlignItems(ui.Center).Children(func() {
					refPill(c, headCapsuleColors(theme.TokensOf(c), rowBg), nil, "main")
				})
			}, 200, 44)
			tt.Frame()
			node, ok := tt.AccessByRole(ui.RoleText, "main")
			if !ok {
				t.Fatalf("palette=%s dark=%v missing main pill text", palette, dark)
			}
			img := tt.Image()
			maxLum := 0.0
			minLum := 1.0
			x0, x1 := int(node.Bounds.X), int(node.Bounds.X+node.Bounds.W)
			y0, y1 := int(node.Bounds.Y), int(node.Bounds.Y+node.Bounds.H)
			for y := y0; y <= y1; y++ {
				for x := x0; x <= x1; x++ {
					if !img.Rect.Bounds().Empty() && x >= img.Rect.Min.X && x < img.Rect.Max.X && y >= img.Rect.Min.Y && y < img.Rect.Max.Y {
						r, g, b, _ := img.At(x, y).RGBA()
						lum := surfaceLuminance(ui.RGB(uint8(r>>8), uint8(g>>8), uint8(b>>8)))
						if lum > maxLum {
							maxLum = lum
						}
						if lum < minLum {
							minLum = lum
						}
					}
				}
			}
			if maxLum < 0.7 {
				t.Fatalf("palette=%s dark=%v main pill text max luminance=%.3f; want bright white ink (>= 0.70)", palette, dark, maxLum)
			}
		}
	}
}

func TestAddRemoteModalIsSingleFrameAndDistinguishesDisabledPrimaryButton(t *testing.T) {
	for _, dark := range []bool{false, true} {
		mode := "light"
		if dark {
			mode = "dark"
		}
		a := &app{
			locale:    localeEnglish,
			themeMode: mode,
			palette:   themePaletteSuperGit,
			repo:      &gitrepo.Repository{Root: "/fixture"},
			addRemoteForm: addRemoteFormState{
				root: "/fixture",
				open: true,
			},
		}
		tt := ui.NewTester(func(c *ui.Context) {
			a.applyTheme(c)
			a.addRemoteDialog(c)
		}, 900, 640)
		tt.Frame()

		dialog, ok := tt.AccessByRole(ui.RoleDialog, "")
		if !ok {
			for _, n := range tt.AccessAll() {
				if n.Role == ui.RoleDialog {
					dialog, ok = n, true
					break
				}
			}
		}
		if !ok {
			t.Fatalf("dark=%v missing dialog role", dark)
		}
		// Inner sheet is 420px wide; a double-frame outer modal with 20px padding
		// on each side would inflate RoleDialog to ~460px.
		if dialog.Bounds.W > 424 {
			t.Fatalf("dark=%v modal width=%.1f; expected single-frame 420px card without outer modal padding", dark, dialog.Bounds.W)
		}

		tk := themeConfig(themePaletteSuperGit).Tokens(dark)
		primaryBg, _ := workbenchButtonColors(tk, buttonPrimary, false, false, true, false)
		normalBg, _ := workbenchButtonColors(tk, buttonNormal, false, false, false, false)
		if primaryBg == normalBg {
			t.Fatalf("dark=%v disabled primary button fill matches normal cancel button (%v)", dark, primaryBg)
		}
	}
}

func TestCollapsibleSectionsUseFullBleedHeadersWithoutChildIndent(t *testing.T) {
	for _, dark := range []bool{false, true} {
		mode := "light"
		if dark {
			mode = "dark"
		}
		a := &app{
			page:      "history",
			locale:    localeEnglish,
			themeMode: mode,
			palette:   themePaletteSuperGit,
			repo:      &gitrepo.Repository{Root: "/fixture"},
			branches:  []gitrepo.Branch{{Name: "main", Current: true}, {Name: "feat/fold-band"}},
			status: gitrepo.Status{
				Entries: []gitrepo.Entry{
					{Path: "internal/lobedesk/ref_capsule.go", Index: ' ', Worktree: 'M'},
				},
			},
		}

		// 1. Files collapsible section: full-bleed band + flush child row (no extra left indent).
		title := a.t("fileGroupModified")
		tt := ui.NewTester(func(c *ui.Context) {
			a.applyTheme(c)
			tk := theme.TokensOf(c)
			ui.Column(c).Width(280).Height(260).Background(tk.BgContainer).Children(func() {
				a.fileSection(c, filesSectionUnstaged, title, a.unstagedEntries(), false)
			})
		}, 280, 260)
		tt.Frame()

		header, ok := tt.AccessByRole(ui.RoleButton, title)
		if !ok || !header.Expanded {
			t.Fatalf("dark=%v missing expanded file section header: %+v", dark, header)
		}
		if header.Bounds.X > 1 || header.Bounds.W < 278 {
			t.Fatalf("dark=%v file section header not full-bleed: %+v", dark, header.Bounds)
		}
		tk := themeConfig(themePaletteSuperGit).Tokens(dark)
		bg, _, _ := fileSectionHeaderColors(tk, false)
		if bg == tk.BgContainer {
			t.Fatalf("dark=%v file section header background equals container background", dark)
		}

		row, ok := tt.AccessByRole(ui.RoleOption, "ref_capsule.go")
		if !ok {
			for _, n := range tt.AccessAll() {
				if n.Role == ui.RoleOption {
					row, ok = n, true
					break
				}
			}
		}
		if !ok {
			t.Fatalf("dark=%v missing child file row", dark)
		}
		if row.Bounds.X > header.Bounds.X+8 {
			t.Fatalf("dark=%v child file row X=%.1f still indented beneath header X=%.1f", dark, row.Bounds.X, header.Bounds.X)
		}

		tt.ClickAt(header.Bounds.X+header.Bounds.W/2, header.Bounds.Y+header.Bounds.H/2)
		tt.Frame()
		collapsedHeader, ok := tt.AccessByRole(ui.RoleButton, title)
		if !ok || collapsedHeader.Expanded {
			t.Fatalf("dark=%v clicking file section header did not collapse it: %+v", dark, collapsedHeader)
		}

		a.toggleSidebarGroup("branches")
		// 2. Sidebar collapsible section: full-bleed band + flush branch row (no 22px indent).
		var sidebarW float32
		sb := ui.NewTester(func(c *ui.Context) {
			a.applyTheme(c)
			ui.Row(c).Width(240).Height(420).Children(func() { sidebarW = a.sidebar(c) })
		}, 240, 420)
		sb.Frame()

		sbHeader, ok := sb.AccessByRole(ui.RoleDisclosure, a.t("branches"))
		if !ok || !sbHeader.Expanded {
			t.Fatalf("dark=%v missing expanded sidebar branches header: %+v", dark, sbHeader)
		}
		if sbHeader.Bounds.X > 1 || sbHeader.Bounds.W < sidebarW-1 {
			t.Fatalf("dark=%v sidebar header not full-bleed (sidebarW=%.1f): %+v", dark, sidebarW, sbHeader.Bounds)
		}
		branchRow, ok := sb.AccessByRole(ui.RoleButton, "feat/fold-band")
		if !ok {
			t.Fatalf("dark=%v missing expanded sidebar branch row", dark)
		}
		if branchRow.Bounds.X > sbHeader.Bounds.X+8 {
			t.Fatalf("dark=%v sidebar branch row X=%.1f still indented beneath header X=%.1f", dark, branchRow.Bounds.X, sbHeader.Bounds.X)
		}
		sb.ClickAt(sbHeader.Bounds.X+sbHeader.Bounds.W/2, sbHeader.Bounds.Y+sbHeader.Bounds.H/2)
		sb.Frame()
		if folded, ok := sb.AccessByRole(ui.RoleDisclosure, a.t("branches")); !ok || folded.Expanded {
			t.Fatalf("dark=%v clicking sidebar header did not collapse it: %+v", dark, folded)
		}
	}
}
