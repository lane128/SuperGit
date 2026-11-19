package lobedesk

import (
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestCommitAuthorMenuCopiesCapturedFullName(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		t.Run(string(language), func(t *testing.T) {
			original := "作者 Name ’ with spaces " + strings.Repeat("long author ", 12)
			commit := gitrepo.Commit{SHA: strings.Repeat("a", 40), Subject: "Selected commit", Author: original, Date: "2026-11-18 12:00:00"}
			a := &app{locale: language, selectedSHA: commit.SHA}
			a.commitFields.Summary = "keep draft"
			a.commitFields.Body = "keep body"
			copied := ""
			previous := writeClipboard
			writeClipboard = func(value string) error { copied = value; return nil }
			t.Cleanup(func() { writeClipboard = previous })
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				ui.Column(c).Fill().Children(func() { a.commitInfoContent(c, commit, theme.TokensOf(c)) })
			}, 480, 360)
			author, ok := tt.Find(original)
			if !ok {
				t.Fatal("author text missing")
			}
			tt.RightClickAt(author.X+author.W/2, author.Y+author.H/2)
			label := "Copy Author Name"
			if language == localeChinese {
				label = "复制作者姓名"
			}
			item, ok := tt.AccessByRole(ui.RoleMenuItem, label)
			if !ok {
				t.Fatal("author context menu lacks copy name")
			}
			// A refresh must not retarget the menu already opened on the original author.
			commit.Author = "Different author after refresh"
			tt.Frame()
			tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
			if copied != original {
				t.Fatalf("copied wrong or elided author: %q", copied)
			}
			if a.commitFields.Summary != "keep draft" || a.commitFields.Body != "keep body" || a.selectedSHA != commit.SHA || a.busy || a.loading {
				t.Fatal("copy changed draft/selection or started repository work")
			}
		})
	}
}
func TestCommitAuthorMenuReportsClipboardFailure(t *testing.T) {
	a := &app{locale: localeEnglish}
	commit := gitrepo.Commit{SHA: strings.Repeat("a", 40), Author: "Fixture author", Subject: "Fixture subject"}
	previous := writeClipboard
	writeClipboard = func(string) error { return errors.New("owned clipboard failure") }
	t.Cleanup(func() { writeClipboard = previous })
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Children(func() { a.commitInfoContent(c, commit, theme.TokensOf(c)) })
	}, 480, 360)
	author, ok := tt.Find(commit.Author)
	if !ok {
		t.Fatal("author missing")
	}
	tt.RightClickAt(author.X+author.W/2, author.Y+author.H/2)
	item, ok := tt.AccessByRole(ui.RoleMenuItem, "Copy Author Name")
	if !ok {
		t.Fatal("copy menu missing")
	}
	tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
	notices := a.notices[toastMain]
	if len(notices) != 1 || notices[0].kind != "error" || !strings.Contains(notices[0].title, "owned clipboard failure") {
		t.Fatal("clipboard failure was reported as success")
	}
}
