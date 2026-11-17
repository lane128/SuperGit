package lobedesk

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestCommitParentsRemainReachableInNarrowSummary(t *testing.T) {
	before := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(before) })
	for _, language := range []locale{localeEnglish, localeChinese} {
		for _, scale := range []float32{1, 1.15} {
			t.Run(fmt.Sprintf("%s/%.2f", language, scale), func(t *testing.T) {
				setUIFontScale(scale)
				a := &app{locale: language}
				commit := gitrepo.Commit{SHA: "merge", Subject: "Merge branches", Parents: []string{
					"aaaaaaaa11111111", "bbbbbbbb22222222", "cccccccc33333333", "dddddddd44444444",
				}}
				tt := ui.NewTester(func(c *ui.Context) {
					theme.Apply(c)
					ui.Column(c).Fill().Children(func() { a.commitInfoContent(c, commit, theme.TokensOf(c)) })
				}, 292, 640)
				for _, sha := range commit.Parents {
					link, ok := tt.AccessByRole(ui.RoleButton, shortSHA(sha))
					if !ok || link.Offscreen || link.Bounds.X < 14 || link.Bounds.X+link.Bounds.W > 279 || link.Bounds.Y+link.Bounds.H > 640 {
						t.Errorf("parent %s is outside the summary content: %+v", sha, link)
					}
				}
			})
		}
	}
}
