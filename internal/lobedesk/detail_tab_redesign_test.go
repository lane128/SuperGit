package lobedesk

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

// Catch clipping of the stage/close slot and selection-dependent width changes
// at the smallest supported detail viewport, including large English text.
func TestDetailTabRedesignNarrowVersions(t *testing.T) {
	defer setUIFontScale(fontScaleStandard)
	for _, scale := range []float32{fontScaleStandard, fontScaleLarge} {
		setUIFontScale(scale)
		for _, locale := range []locale{localeEnglish, localeChinese} {
			t.Run(fmt.Sprintf("%g/%s", scale, locale), func(t *testing.T) {
				const path = "internal/lobedesk/commit_details_select_test.go"
				a := &app{page: "history", locale: locale, worktreeSelected: true, repo: &gitrepo.Repository{Root: t.TempDir()}, status: gitrepo.Status{Entries: []gitrepo.Entry{{Path: path, Index: 'M', Worktree: 'M'}}}, detailReveal: true}
				for _, staged := range []bool{false, true} {
					a.detailTabs().Open(detailFileKey{Path: path, Staged: staged})
				}
				width := float32(560)
				view := ui.NewTester(func(c *ui.Context) {
					theme.Apply(c)
					ui.Row(c).Width(width).Height(240).Children(func() { a.historyDiffPane(c) })
				}, 600, 240)
				view.Frame()
				width = 230
				view.Frame()
				view.Frame()
				view.Frame()
				for _, staged := range []bool{true, false} {
					key := detailTabKey{File: detailFileKey{Path: path, Staged: staged}}
					a.detailTabs().Activate(key)
					a.detailReveal = true
					view.Frame()
					view.Frame()
					suffix := "unstaged"
					if staged {
						suffix = "staged"
					}
					tab, ok := view.AccessByRole(ui.RoleTab, "commit_details_select_test.go · "+a.t(suffix))
					close, closed := view.AccessByRole(ui.RoleButton, a.t("close")+" "+path)
					// There are two same-path close controls: locate the selected tab's neighbor.
					for _, node := range view.AccessAll() {
						if node.Role == ui.RoleButton && node.Name == a.t("close")+" "+path && node.Bounds.X >= tab.Bounds.X && node.Bounds.X < tab.Bounds.X+230 {
							close = node
							closed = true
							break
						}
					}
					if !ok || !closed || tab.Bounds.X < -0.5 || close.Bounds.X+close.Bounds.W > 230.5 {
						t.Fatalf("clipped version: tab=%+v close=%+v", tab, close)
					}
					if math.Abs(float64(tab.Bounds.H-scaledFont(36))) > .5 || close.Bounds.H > tab.Bounds.H || close.Bounds.Y < tab.Bounds.Y {
						t.Errorf("row/close geometry: tab=%+v close=%+v", tab.Bounds, close.Bounds)
					}
					if !strings.Contains(tab.Description, "M") || !strings.Contains(tab.Description, path) {
						t.Errorf("missing status/path: %q", tab.Description)
					}
				}
			})
		}
	}
}

func TestDetailTabStatusHollowBadge(t *testing.T) {
	for _, palette := range []string{themePaletteSuperGit, themePaletteDefault, themePaletteCatppuccin, themePaletteCatppuccinFrappe, themePaletteCatppuccinMacchiato} {
		for _, dark := range []bool{false, true} {
			for _, code := range []byte{'A', 'M', 'D', '?', 'R', 'C', 'T', '!', 'S'} {
				t.Run(fmt.Sprintf("%s/%t/%c", palette, dark, code), func(t *testing.T) {
					path := "example.go"
					entry := gitrepo.Entry{Path: path, Worktree: code}
					if code == '!' {
						entry.Conflict = true
					}
					if code == 'S' {
						entry.Submodule = "S.M."
					}
					a := &app{locale: localeEnglish, worktreeSelected: true, status: gitrepo.Status{Entries: []gitrepo.Entry{entry}}}
					key := detailTabKey{File: detailFileKey{Path: path}}
					a.detailTabs().Open(key.File)
					config := themeConfig(palette)
					view := ui.NewTester(func(c *ui.Context) {
						c.SetTheme(config.Theme(dark))
						tk := theme.TokensOf(c)
						ui.Row(c).Height(36).Background(tk.BgLayout).Children(func() { a.detailTabButton(c, tk, key) })
					}, 300, 50)
					view.Move(290, 45)
					view.Frame()
					tab, ok := view.AccessByRole(ui.RoleTab, path+" · Unstaged")
					if !ok {
						t.Fatal("missing tab")
					}
					if !strings.Contains(tab.Description, string(code)) {
						t.Errorf("missing status meaning: %q", tab.Description)
					}
					img := view.Image()
					tk := config.Tokens(dark)
					// Badge at 10,9: interior must remain the selected tab surface, while
					// its left border is opaque high-contrast ink.
					inside := img.RGBAAt(int(tab.Bounds.X)+12, int(tab.Bounds.Y)+12)
					if inside.R != tk.BgContainer.R || inside.G != tk.BgContainer.G || inside.B != tk.BgContainer.B {
						t.Errorf("badge interior is not transparent: %v", inside)
					}
					edge := img.RGBAAt(int(tab.Bounds.X)+10, int(tab.Bounds.Y)+18)
					ink := ui.RGB(edge.R, edge.G, edge.B)
					minimum := 4.5
					if strings.ContainsRune("AMD", rune(code)) {
						minimum = 7
					}
					for _, bg := range []ui.Color{tk.BgContainer, tk.BgLayout, tk.FillTertiary.Over(tk.BgLayout), tk.Fill.Over(tk.BgLayout)} {
						if ratio := contrastRatio(ink, bg); ratio < minimum {
							t.Errorf("badge contrast %.2f < %.1f", ratio, minimum)
						}
					}
				})
			}
		}
	}
}
