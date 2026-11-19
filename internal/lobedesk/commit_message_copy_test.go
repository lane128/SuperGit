package lobedesk

import (
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestMessageMenuCopiesCapturedPartialSelection(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		for _, failure := range []bool{false, true} {
			name := string(language)
			if failure {
				name += "/clipboard failure"
			}
			t.Run(name, func(t *testing.T) {
				const body = "作者 hello https://example.invalid/docs after"
				commit := gitrepo.Commit{Subject: "Selected", SHA: strings.Repeat("a", 40)}
				a := &app{locale: language, selectedSHA: commit.SHA, commitMessage: "Selected\n\n" + body}
				a.commitFields.Summary = "keep draft"
				attempts := []string{}
				previous := writeClipboard
				writeClipboard = func(value string) error {
					attempts = append(attempts, value)
					if failure {
						return errors.New("owned selection clipboard failure")
					}
					return nil
				}
				t.Cleanup(func() { writeClipboard = previous })
				tt := ui.NewTester(func(c *ui.Context) {
					theme.Apply(c)
					ui.Column(c).Fill().Children(func() { a.commitInfoContent(c, commit, theme.TokensOf(c)) })
				}, 800, 400)
				r, ok := tt.Find(body)
				if !ok {
					t.Fatal("message missing")
				}
				y := r.Y + r.H/2
				tt.Press(r.X+1, y)
				tt.Move(r.X+68, y)
				tt.Release(r.X+68, y)
				tt.Key(ui.Cmd, ui.KeyC)
				baseline := tt.Clipboard()
				if !strings.HasPrefix(baseline, "作者") || baseline == body {
					t.Fatalf("partial Unicode fixture invalid: %q", baseline)
				}
				tt.RightClickAt(r.X+10, y)
				label := "Copy Selected Text"
				if language == localeChinese {
					label = "复制所选文本"
				}
				item, ok := tt.AccessByRole(ui.RoleMenuItem, label)
				if !ok || item.Disabled {
					t.Fatal("message target menu lacks enabled selection copy")
				}
				a.commitMessage = "Selected\n\nRefreshed https://example.invalid/new"
				a.selectedSHA = strings.Repeat("b", 40)
				tt.Frame()
				tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
				if len(attempts) != 1 || attempts[0] != baseline {
					t.Fatalf("copied new or whole message: %q want %q", attempts, baseline)
				}
				notices := a.notices[toastMain]
				wantKind := "success"
				if failure {
					wantKind = "error"
				}
				if len(notices) != 1 || notices[0].kind != wantKind {
					t.Fatal("clipboard feedback did not match outcome")
				}
				if failure && !strings.Contains(notices[0].title, "owned selection clipboard failure") {
					t.Fatal("clipboard error missing")
				}
				if a.selectedSHA != strings.Repeat("b", 40) || a.commitFields.Summary != "keep draft" || a.pendingJumpSHA != "" || len(tt.OpenedURLs()) != 0 || a.busy || a.loading {
					t.Fatal("selection copy mutated navigation or repository state")
				}
			})
		}
	}
}

func TestMessageMenuCopySelectionEmptyOutsideAndCancel(t *testing.T) {
	for _, scenario := range []string{"empty", "outside selection", "cancel"} {
		t.Run(scenario, func(t *testing.T) {
			const body = "Before https://example.invalid/docs after"
			a := &app{locale: localeEnglish, commitMessage: "Selected\n\n" + body}
			calls := 0
			previous := writeClipboard
			writeClipboard = func(string) error { calls++; return nil }
			t.Cleanup(func() { writeClipboard = previous })
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				ui.Column(c).Fill().Children(func() { a.commitInfoContent(c, gitrepo.Commit{Subject: "Selected"}, theme.TokensOf(c)) })
			}, 800, 400)
			r, ok := tt.Find(body)
			if !ok {
				t.Fatal("message missing")
			}
			y := r.Y + r.H/2
			if scenario != "empty" {
				tt.Press(r.X+1, y)
				tt.Move(r.X+30, y)
				tt.Release(r.X+30, y)
			}
			x := r.X + 10
			if scenario == "outside selection" {
				x = r.X + r.W - 5
			}
			tt.RightClickAt(x, y)
			item, ok := tt.AccessByRole(ui.RoleMenuItem, "Copy Selected Text")
			if !ok {
				t.Fatal("copy item absent")
			}
			if scenario == "cancel" {
				if item.Disabled {
					t.Fatal("selected text copy disabled")
				}
				tt.Key(0, ui.KeyEscape)
				if _, ok := tt.AccessByRole(ui.RoleMenuItem, "Copy Selected Text"); ok {
					t.Fatal("Escape left selection menu open")
				}
			} else {
				if !item.Disabled {
					t.Fatal("empty selection copy enabled")
				}
				tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
			}
			if calls != 0 || len(a.notices[toastMain]) != 0 || len(tt.OpenedURLs()) != 0 {
				t.Fatal("disabled or cancelled selection copy dispatched an action")
			}
		})
	}
}

func TestMessageMenuCopiesAcrossSubjectAndBody(t *testing.T) {
	const body = "Unicode 作者 https://example.invalid/docs after"
	a := &app{locale: localeEnglish, commitMessage: "Selected subject\n\n" + body}
	copied := ""
	previous := writeClipboard
	writeClipboard = func(value string) error { copied = value; return nil }
	t.Cleanup(func() { writeClipboard = previous })
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Children(func() { a.commitInfoContent(c, gitrepo.Commit{Subject: "Selected subject"}, theme.TokensOf(c)) })
	}, 800, 400)
	first, ok := tt.Find("Selected subject")
	if !ok {
		t.Fatal("subject missing")
	}
	last, ok := tt.Find(body)
	if !ok {
		t.Fatal("body missing")
	}
	tt.Press(first.X+1, first.Y+first.H/2)
	tt.Move(last.X+last.W-1, last.Y+last.H/2)
	tt.Release(last.X+last.W-1, last.Y+last.H/2)
	tt.Key(ui.Cmd, ui.KeyC)
	baseline := tt.Clipboard()
	if !strings.HasPrefix(baseline, "Selected subject\nUnicode 作者") || !strings.HasSuffix(baseline, "after") {
		t.Fatalf("cross-paragraph fixture invalid: %q", baseline)
	}
	tt.RightClickAt(last.X+10, last.Y+last.H/2)
	item, ok := tt.AccessByRole(ui.RoleMenuItem, "Copy Selected Text")
	if !ok || item.Disabled {
		t.Fatal("cross-paragraph copy missing")
	}
	tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
	if copied != baseline {
		t.Fatalf("context copy differs from ordinary copy: %q want %q", copied, baseline)
	}
}
