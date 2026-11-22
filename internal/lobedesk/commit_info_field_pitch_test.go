package lobedesk

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestCommitInfoSingleLineFieldPitch(t *testing.T) {
	oldScale := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(oldScale) })
	commit := gitrepo.Commit{
		Subject: "Fix spacing", SHA: "0123456789abcdef0123456789abcdef01234567",
		Author: "Lin", AuthorEmail: "lin@example.com", Date: "2026-11-18T09:30:00Z",
		Parents: []string{"abc123456789abcdef0123456789abcdef012345"},
	}
	for _, tc := range []struct {
		language locale
		labels   [4]string
	}{
		{localeEnglish, [4]string{"Hash", "Author", "Date", "Parent"}},
		{localeChinese, [4]string{"哈希", "作者", "日期", "父提交"}},
	} {
		for _, scale := range []float32{fontScaleStandard, fontScaleLarge} {
			t.Run(fmt.Sprintf("%s/%.2f", tc.language, scale), func(t *testing.T) {
				setUIFontScale(scale)
				a := &app{locale: tc.language}
				summary := ui.NewTester(func(c *ui.Context) {
					a.applyTheme(c)
					ui.Column(c).Fill().Children(func() { a.commitInfoContent(c, commit, theme.TokensOf(c)) })
				}, 640, 360)

				var centers [4]float32
				for i, label := range tc.labels {
					r, ok := summary.Find(label)
					if !ok || r.W <= 0 || r.H <= 0 {
						t.Fatalf("missing field label %q: %+v", label, r)
					}
					centers[i] = r.Y + r.H/2
				}
				first, second, third := centers[1]-centers[0], centers[2]-centers[1], centers[3]-centers[2]
				t.Logf("%v center steps = [%.2f %.2f %.2f] pt", tc.labels, first, second, third)
				if scale == fontScaleStandard {
					for _, step := range []float32{first, second, third} {
						if step < 25.5 || step > 26.5 {
							t.Errorf("default field pitch = %.2f pt, want 26 pt", step)
						}
					}
				}
				shortest, longest := min(first, second, third), max(first, second, third)
				if shortest <= 0 || longest-shortest > 1 {
					t.Errorf("%v center steps = [%.2f %.2f %.2f] pt, want positive steps within 1 pt of each other", tc.labels, first, second, third)
				}

				// Unconstrained controls provide the natural hit-target sizes at this font scale.
				copyLabel, parentLabel := a.t("copySHA"), shortSHA(commit.Parents[0])
				natural := ui.NewTester(func(c *ui.Context) {
					a.applyTheme(c)
					ui.Row(c).AlignItems(ui.Center).Children(func() {
						compactActionIcon(c, lucide.Copy(), copyLabel, buttonNormal, false)
						ui.Link(c, parentLabel, "").Font(a.patchFontFamily()).FontSize(scaledFont(11)).SingleLine()
					})
				}, 640, 360)
				for label, role := range map[string]ui.Role{copyLabel: ui.RoleButton, parentLabel: ui.RoleLink} {
					want, ok := natural.AccessByRole(role, label)
					if !ok || want.Bounds.W <= 0 || want.Bounds.H <= 0 {
						t.Fatalf("missing reference button %q", label)
					}
					got, ok := summary.AccessByRole(role, label)
					if !ok || got.Offscreen || got.Bounds.W < want.Bounds.W || got.Bounds.H < want.Bounds.H {
						t.Errorf("button %q bounds = %+v (offscreen %t), want visible and at least %.2f × %.2f pt", label, got.Bounds, got.Offscreen, want.Bounds.W, want.Bounds.H)
					}
				}
			})
		}
	}
}
