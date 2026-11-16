package lobedesk

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// Placeholder sections retain their complete reading range without creating
// every distant header and Retry control while a large overview is pending.
func TestLargeWorktreeOverviewClipsPlaceholderSections(t *testing.T) {
	defer setUIFontScale(uiFontScaleValue())
	for _, config := range []struct {
		language locale
		scale    float32
	}{{localeEnglish, fontScaleStandard}, {localeEnglish, fontScaleLarge}, {localeChinese, fontScaleStandard}, {localeChinese, fontScaleLarge}} {
		setUIFontScale(config.scale)
		for _, kind := range []string{"loading", "error", "empty"} {
			for _, width := range []float32{220, 500} {
				t.Run(fmt.Sprintf("%s/scale=%.2f/%s/width=%.0f", config.language, config.scale, kind, width), func(t *testing.T) {
					a := &app{page: "history", worktreeSelected: true, locale: config.language}
					for i := range 1000 {
						entry := worktreeOverviewEntry{Key: detailFileKey{Path: fmt.Sprintf("files/%04d.txt", i)}}
						switch kind {
						case "loading":
							entry.Loading = true
						case "error":
							entry.Err = strings.Repeat("Cannot read this patch: 路径暂时不可访问。 ", 12)
						}
						a.overviewEntries = append(a.overviewEntries, entry)
					}
					var ordinaryHeight float32
					ordinary := ui.NewTester(func(c *ui.Context) {
						a.applyTheme(c)
						ui.Column(c).Width(width).Children(func() {
							block := a.worktreeOverviewFile(c, theme.TokensOf(c), a.overviewEntries[0], "", 0)
							ordinaryHeight = block.Bounds().H
						})
					}, int(width), 240)
					for range 4 {
						ordinary.Frame()
					}
					tt := ui.NewTester(func(c *ui.Context) {
						a.applyTheme(c)
						ui.Column(c).Width(width).Height(240).Children(func() {
							scroll := layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.commitInfoScroll}, func() { a.worktreeOverview(c) }).Grow(1).Basis(0)
							a.commitDiffViewport = scroll.Bounds()
						})
					}, int(width), 240)
					for range 4 {
						tt.Frame()
					}
					label := func(index int) string { return fmt.Sprintf("files/%04d.txt · %s", index, a.t("unstaged")) }
					if _, ok := tt.AccessByRole(ui.RoleButton, label(900)); ok {
						t.Fatal("distant placeholder header built before it was visible")
					}
					first, ok := tt.AccessByRole(ui.RoleButton, label(0))
					if !ok {
						t.Fatal("first placeholder missing")
					}
					second, ok := tt.AccessByRole(ui.RoleButton, label(1))
					if !ok {
						t.Fatal("second placeholder missing")
					}
					height := second.Bounds.Y - first.Bounds.Y
					if math.Abs(float64(height-ordinaryHeight)) > 0.5 {
						t.Fatalf("placeholder height %.2f differs from ordinary layout %.2f", height, ordinaryHeight)
					}
					wantMax := first.Bounds.Y + 1000*ordinaryHeight - 240
					if math.Abs(float64(a.commitInfoScroll.MaxY-wantMax)) > 2 {
						t.Fatalf("scroll extent %.2f differs from ordinary layout %.2f", a.commitInfoScroll.MaxY, wantMax)
					}
					if height <= 35 || a.commitInfoScroll.MaxY < 990*height {
						t.Fatalf("placeholder body or extent lost: height=%.2f max=%.2f", height, a.commitInfoScroll.MaxY)
					}
					a.commitInfoScroll.Y = 900 * height
					for range 4 {
						tt.Frame()
					}
					if _, ok := tt.AccessByRole(ui.RoleButton, label(900)); !ok {
						t.Fatal("scrolling did not materialize distant placeholder")
					}
					if kind == "error" {
						if _, ok := tt.AccessByRole(ui.RoleButton, a.t("retry")); !ok {
							t.Fatal("visible error lost its Retry action")
						}
					}
					a.commitInfoScroll.Y = 0
					for range 4 {
						tt.Frame()
					}
					if _, ok := tt.AccessByRole(ui.RoleButton, label(0)); !ok {
						t.Fatal("returning to the top lost first placeholder")
					}
					// Change the live column width without replacing the UI tree.
					width = 720 - width
					for range 4 {
						ordinary.Frame()
						tt.Frame()
					}
					first, ok = tt.AccessByRole(ui.RoleButton, label(0))
					second, secondOK := tt.AccessByRole(ui.RoleButton, label(1))
					if !ok || !secondOK {
						t.Fatal("width change lost visible placeholders")
					}
					if math.Abs(float64(second.Bounds.Y-first.Bounds.Y-ordinaryHeight)) > 0.5 {
						t.Fatal("width change retained stale placeholder height")
					}
					wantMax = first.Bounds.Y + 1000*ordinaryHeight - 240
					if math.Abs(float64(a.commitInfoScroll.MaxY-wantMax)) > 2 {
						t.Fatalf("width change left stale extent %.2f, want %.2f", a.commitInfoScroll.MaxY, wantMax)
					}
				})
			}
		}
	}
}
