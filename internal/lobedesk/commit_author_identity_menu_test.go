package lobedesk

import (
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func setFixtureAuthorEmail(t *testing.T, commit *gitrepo.Commit, email string) {
	t.Helper()
	field := reflect.ValueOf(commit).Elem().FieldByName("AuthorEmail")
	if !field.IsValid() || !field.CanSet() || field.Kind() != reflect.String {
		t.Fatal("commit author email is unavailable")
	}
	field.SetString(email)
}
func TestCommitAuthorIdentityMenuCopiesCapturedAuthor(t *testing.T) {
	for _, tc := range []struct{ label, want string }{{"Copy Author", "作者 Fixture <author@example.invalid>"}, {"Copy Author Email", "author@example.invalid"}} {
		t.Run(tc.label, func(t *testing.T) {
			a := &app{locale: localeEnglish}
			commit := gitrepo.Commit{SHA: strings.Repeat("a", 40), Subject: "Selected", Author: "作者 Fixture"}
			// A reserved-domain address provides local test data only.
			email := "author@example.invalid"
			setFixtureAuthorEmail(t, &commit, email)
			copied := ""
			previous := writeClipboard
			writeClipboard = func(value string) error { copied = value; return nil }
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
			item, ok := tt.AccessByRole(ui.RoleMenuItem, tc.label)
			if !ok {
				t.Fatalf("missing %s action", tc.label)
			}
			commit.Author = "Different author"
			setFixtureAuthorEmail(t, &commit, "different@example.invalid")
			tt.Frame()
			tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
			want := tc.want
			if copied != want {
				t.Fatalf("copied wrong author identity: %q, want %q", copied, want)
			}
		})
	}
}
