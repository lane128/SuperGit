package lobedesk

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestCommitAuthorIdentityMenuWithoutEmailKeepsOnlyNameEnabled(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		t.Run(string(language), func(t *testing.T) {
			previousScale := uiFontScaleValue()
			setUIFontScale(fontScaleLarge)
			t.Cleanup(func() { setUIFontScale(previousScale) })
			a := &app{locale: language}
			commit := gitrepo.Commit{Author: "作者 " + strings.Repeat("Long author ", 30), Subject: "Selected", Date: "2026-11-18T12:00:00+08:00"}
			copied := 0
			previous := writeClipboard
			writeClipboard = func(string) error { copied++; return nil }
			t.Cleanup(func() { writeClipboard = previous })
			tt := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				ui.Column(c).Fill().Children(func() { a.commitInfoContent(c, commit, theme.TokensOf(c)) })
			}, 480, 360)
			author, ok := tt.Find(commit.Author)
			if !ok || author.W <= 0 || author.X < 0 || author.X+author.W > 480 {
				t.Fatal("long author has no visible hit target")
			}
			tt.RightClickAt(author.X+author.W/2, author.Y+author.H/2)
			labels := []string{"Copy Author", "Copy Author Name", "Copy Author Email"}
			if language == localeChinese {
				labels = []string{"复制作者", "复制作者姓名", "复制作者邮箱"}
			}
			for index, label := range labels {
				item, ok := tt.AccessByRole(ui.RoleMenuItem, label)
				if !ok || item.Disabled != (index != 1) {
					t.Fatalf("wrong disabled state for %s: %+v", label, item)
				}
			}
			for _, index := range []int{0, 2} {
				tt.Key(0, ui.KeyEscape)
				tt.RightClickAt(author.X+author.W/2, author.Y+author.H/2)
				item, ok := tt.AccessByRole(ui.RoleMenuItem, labels[index])
				if !ok {
					t.Fatalf("disabled action disappeared: %s", labels[index])
				}
				tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
				if copied != 0 {
					t.Fatalf("disabled action copied placeholder text: %s", labels[index])
				}
			}
		})
	}
}
