package lobedesk

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestCommitMessageLinkMenuLongURLStaysInViewport(t *testing.T) {
	raw := "https://example.invalid/" + strings.Repeat("long-path/", 25) + "?q=one#part"
	second := strings.Replace(raw, "q=one", "q=two", 1)
	a := &app{locale: localeEnglish}
	commit := gitrepo.Commit{Subject: raw + " " + second, Author: "Fixture"}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Children(func() { a.commitInfoContent(c, commit, theme.TokensOf(c)) })
	}, 480, 360)
	r, ok := tt.Find(commit.Subject)
	if !ok {
		t.Fatal("URL subject missing")
	}
	tt.RightClickAt(r.X+5, r.Y+5)
	var previousBottom float32
	for _, address := range []string{raw, second} {
		item, ok := tt.AccessByRole(ui.RoleMenuItem, address)
		if !ok || item.Bounds.X < 0 || item.Bounds.X+item.Bounds.W > 480 || item.Bounds.Y < previousBottom || item.Bounds.Y+item.Bounds.H > 360 {
			t.Fatalf("URL menu escaped viewport or overlapped another item: %+v", item)
		}
		if item.Bounds.H < 64 {
			t.Fatal("long URL was reduced to an indistinguishable single-line choice")
		}
		previousBottom = item.Bounds.Y + item.Bounds.H
	}
	item, _ := tt.AccessByRole(ui.RoleMenuItem, second)
	tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
	if got := tt.OpenedURLs(); len(got) != 1 || got[0] != second {
		t.Fatalf("wrong long URL chosen: %v", got)
	}
}

func TestCommitMessageWithoutHTTPURLsDoesNotOfferOpen(t *testing.T) {
	const body = "file:///tmp/test javascript:alert(1) mailto:fixture@example.invalid"
	a := &app{locale: localeEnglish, commitMessage: "Selected\n\n" + body}
	commit := gitrepo.Commit{Subject: "Selected"}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Children(func() { a.commitInfoContent(c, commit, theme.TokensOf(c)) })
	}, 800, 360)
	r, ok := tt.Find(body)
	if !ok {
		t.Fatal("body absent")
	}
	tt.RightClickAt(r.X+5, r.Y+r.H/2)
	for _, item := range tt.AccessAll() {
		if item.Role == ui.RoleMenuItem && (strings.HasPrefix(item.Name, "Open") || strings.Contains(item.Name, ":")) {
			t.Fatalf("unsupported URL got an open action: %s", item.Name)
		}
	}
	if len(tt.OpenedURLs()) != 0 {
		t.Fatal("right click opened unsupported content")
	}
}

func TestCommitMessageLinkMenuScrollsToLastURL(t *testing.T) {
	var urls []string
	for index := 0; index < 8; index++ {
		urls = append(urls, "https://example.invalid/"+strings.Repeat("long-path/", 20)+string(rune('a'+index)))
	}
	a := &app{locale: localeEnglish}
	commit := gitrepo.Commit{Subject: strings.Join(urls, " ")}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Children(func() { a.commitInfoContent(c, commit, theme.TokensOf(c)) })
	}, 480, 360)
	r, ok := tt.Find(commit.Subject)
	if !ok {
		t.Fatal("subject absent")
	}
	tt.RightClickAt(r.X+5, r.Y+5)
	tt.Scroll(220, 140, 0, 100000)
	tt.Frame()
	last := urls[len(urls)-1]
	item, ok := tt.AccessByRole(ui.RoleMenuItem, last)
	if !ok || item.Offscreen || item.Bounds.Y < 0 || item.Bounds.Y+item.Bounds.H > 360 {
		t.Fatalf("last wrapped URL unreachable: %+v", item)
	}
	tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
	if got := tt.OpenedURLs(); len(got) != 1 || got[0] != last {
		t.Fatalf("could not open last URL: %v", got)
	}
}

func TestCommitMessageHTTPURLs(t *testing.T) {
	for _, tc := range []struct {
		text string
		want []string
	}{
		{"See (https://example.invalid/a_(b)?q=one#part). And http://example.invalid/two", []string{"https://example.invalid/a_(b)?q=one#part", "http://example.invalid/two"}},
		{"https://example.invalid/a https://example.invalid/a", []string{"https://example.invalid/a"}},
		{"https://example.invalid/releases/v1! https://example.invalid/a, https://example.invalid/b; https://example.invalid/c.", []string{"https://example.invalid/releases/v1!", "https://example.invalid/a,", "https://example.invalid/b;", "https://example.invalid/c."}},
		{"https://example.invalid/O'Reilly https://example.invalid/search?q=it's", []string{"https://example.invalid/O'Reilly", "https://example.invalid/search?q=it's"}},
		{"https://example.invalid/a?q=hello! https://example.invalid/a#fragment,", []string{"https://example.invalid/a?q=hello!", "https://example.invalid/a#fragment,"}},
		{"file:///tmp/test javascript:alert(1) mailto:a@example.invalid https://", nil},
		{"nestedhttps://example.invalid/a http://[::1]:bad", nil},
		{"HTTPS://example.invalid/%E6%96%87?q=x%26y#p", []string{"HTTPS://example.invalid/%E6%96%87?q=x%26y#p"}},
		{"Unicode　https://example.invalid/文档。", []string{"https://example.invalid/文档。"}},
		{"'https://example.invalid/O'Reilly'", []string{"https://example.invalid/O'Reilly"}},
		{"See 'https://example.invalid/a'. And “https://example.invalid/b”.", []string{"https://example.invalid/a", "https://example.invalid/b"}},
		{"https://example.invalid/作者’s “https://example.invalid/quoted”", []string{"https://example.invalid/作者’s", "https://example.invalid/quoted"}},
	} {
		if got := commitMessageHTTPURLs(tc.text); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("URLs in %q: %q, want %q", tc.text, got, tc.want)
		}
	}
}

func TestCommitMessageLinkMenuPreservesSelectionAndCapturedURL(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		t.Run(string(language), func(t *testing.T) {
			const first = "https://example.invalid/one?q=a#part"
			const second = "http://example.invalid/two"
			body := "Read " + first + " and " + second
			commit := gitrepo.Commit{Subject: "Selected", Author: "Fixture", SHA: strings.Repeat("a", 40)}
			a := &app{locale: language, commitMessage: "Selected\n\n" + body, selectedSHA: commit.SHA}
			a.commitText = "keep draft"
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				ui.Column(c).Fill().Children(func() { a.commitInfoContent(c, commit, theme.TokensOf(c)) })
			}, 800, 400)
			r, ok := tt.Find(body)
			if !ok {
				t.Fatal("body missing")
			}
			y := r.Y + r.H/2
			tt.Press(r.X+1, y)
			tt.Move(r.X+r.W-1, y)
			tt.Release(r.X+r.W-1, y)
			tt.Key(ui.Cmd, ui.KeyC)
			if tt.Clipboard() != body || len(tt.OpenedURLs()) != 0 {
				t.Fatal("link menu broke ordinary drag-copy")
			}
			tt.RightClickAt(r.X+r.W/2, y)
			item, ok := tt.AccessByRole(ui.RoleMenuItem, second)
			if !ok {
				t.Fatal("message URL context action missing")
			}
			a.commitMessage = "Selected\n\nRefreshed https://example.invalid/new"
			tt.Frame()
			tt.FailOpenURL(errors.New("owned URL failure"))
			tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
			if got := tt.OpenedURLs(); len(got) != 1 || got[0] != second {
				t.Fatalf("opened wrong captured URL: %v", got)
			}
			notices := a.notices[toastMain]
			if len(notices) != 1 || notices[0].kind != "error" || !strings.Contains(notices[0].title, "owned URL failure") {
				t.Fatal("URL open failure missing")
			}
			if a.commitText != "keep draft" || a.selectedSHA != commit.SHA || a.busy || a.loading {
				t.Fatal("opening message URL changed repository or draft state")
			}
			tt.FailOpenURL(nil)
			refreshed, ok := tt.Find("Refreshed https://example.invalid/new")
			if !ok {
				t.Fatal("refreshed body missing")
			}
			tt.RightClickAt(refreshed.X+5, refreshed.Y+refreshed.H/2)
			retry, ok := tt.AccessByRole(ui.RoleMenuItem, "https://example.invalid/new")
			if !ok {
				t.Fatal("reopened menu retained old URLs")
			}
			tt.ClickAt(retry.Bounds.X+retry.Bounds.W/2, retry.Bounds.Y+retry.Bounds.H/2)
			if got := tt.OpenedURLs(); len(got) != 2 || got[1] != "https://example.invalid/new" {
				t.Fatalf("URL retry failed: %v", got)
			}
			if len(a.notices[toastMain]) != 1 {
				t.Fatal("successful open reported an error or claimed page load")
			}
		})
	}
}
