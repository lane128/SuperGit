package lobedesk

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestCommitDateMenuReopensAfterFailureAndKeepsAuthorSeparate(t *testing.T) {
	a := &app{locale: localeEnglish}
	commit := gitrepo.Commit{Subject: "Selected", Author: "Original author", Date: "Original date 12:34:56"}
	var copied []string
	previous := writeClipboard
	writeClipboard = func(value string) error {
		copied = append(copied, value)
		if len(copied) == 1 {
			return errors.New("owned first date failure")
		}
		return nil
	}
	t.Cleanup(func() { writeClipboard = previous })
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Children(func() { a.commitInfoContent(c, commit, theme.TokensOf(c)) })
	}, 480, 360)
	open := func(text, label, absent string) ui.AccessInfo {
		t.Helper()
		r, ok := tt.Find(text)
		if !ok {
			t.Fatalf("target missing: %s", text)
		}
		tt.RightClickAt(r.X+r.W/2, r.Y+r.H/2)
		item, ok := tt.AccessByRole(ui.RoleMenuItem, label)
		if !ok || item.Disabled {
			t.Fatalf("action missing: %s", label)
		}
		if _, found := tt.AccessByRole(ui.RoleMenuItem, absent); found {
			t.Fatalf("other target menu remained open: %s", absent)
		}
		return item
	}
	click := func(item ui.AccessInfo) { tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2) }
	click(open(commit.Date, "Copy Date", "Copy Author Name"))
	commit.Author, commit.Date = "Refreshed author", "Refreshed date 23:45:06"
	tt.Frame()
	// Use the neighboring author menu between the failed copy and date retry.
	click(open(commit.Author, "Copy Author Name", "Copy Date"))
	click(open(commit.Date, "Copy Date", "Copy Author Name"))
	want := []string{"Original date 12:34:56", "Refreshed author", "Refreshed date 23:45:06"}
	if strings.Join(copied, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("reopened menus mixed targets: %q", copied)
	}
	notices := a.notices[toastMain]
	if len(notices) != 3 || notices[0].kind != "error" || notices[1].kind != "success" || notices[2].kind != "success" {
		t.Fatal("failed date copy did not recover with success feedback")
	}
}

func TestCommitDateMenuEmptyAndCancelDoNotCopy(t *testing.T) {
	for _, text := range []string{"", "2026-11-18 12:34:56"} {
		t.Run("value-"+text, func(t *testing.T) {
			a := &app{locale: localeEnglish}
			calls := 0
			previous := writeClipboard
			writeClipboard = func(string) error { calls++; return nil }
			t.Cleanup(func() { writeClipboard = previous })
			// A visible fixture anchor lets the helper's empty-value guard be clicked;
			// the real summary leaves an absent date blank.
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				date := display.Text(c, "Date fixture", display.TextProps{})
				a.commitDateMenu(c, date, text)
			}, 480, 360)
			anchor, ok := tt.Find("Date fixture")
			if !ok {
				t.Fatal("fixture missing")
			}
			tt.RightClickAt(anchor.X+anchor.W/2, anchor.Y+anchor.H/2)
			item, ok := tt.AccessByRole(ui.RoleMenuItem, "Copy Date")
			if !ok || item.Disabled != (text == "") {
				t.Fatal("wrong empty-date disabled state")
			}
			if text == "" {
				tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
			} else {
				tt.Key(0, ui.KeyEscape)
				if _, open := tt.AccessByRole(ui.RoleMenuItem, "Copy Date"); open {
					t.Fatal("Escape left the date menu open")
				}
			}
			if calls != 0 || len(a.notices[toastMain]) != 0 {
				t.Fatal("empty or cancelled date action copied or reported success")
			}
		})
	}
}

func TestCommitDateMenuCopiesCapturedDisplayValue(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		for _, failure := range []bool{false, true} {
			for _, legacy := range []bool{false, true} {
				name := string(language)
				if failure {
					name += "/clipboard-failure"
				}
				if legacy {
					name += "/legacy"
				}
				t.Run(name, func(t *testing.T) {
					oldScale := uiFontScaleValue()
					setUIFontScale(fontScaleLarge)
					t.Cleanup(func() { setUIFontScale(oldScale) })
					original := "2026-11-18T07:30:19+08:00"
					expected := time.Date(2026, 11, 17, 23, 30, 19, 0, time.UTC).Local().Format("2006-01-02 15:04:05")
					if legacy {
						original = "Legacy date 12:34:56"
						expected = original
					}
					commit := gitrepo.Commit{SHA: strings.Repeat("a", 40), Subject: "Selected", Author: strings.Repeat("作者 name ", 20), Date: original}
					a := &app{locale: language, selectedSHA: commit.SHA}
					a.commitFields.Summary = "keep draft"
					a.commitFields.Body = "keep body"
					copied := ""
					calls := 0
					previous := writeClipboard
					writeClipboard = func(value string) error {
						copied = value
						calls++
						if failure {
							return errors.New("owned date clipboard failure")
						}
						return nil
					}
					t.Cleanup(func() { writeClipboard = previous })
					tt := ui.NewTester(func(c *ui.Context) {
						a.applyTheme(c)
						ui.Column(c).Fill().Children(func() { a.commitInfoContent(c, commit, theme.TokensOf(c)) })
					}, 480, 360)
					date, ok := tt.Find(expected)
					if !ok || date.W <= 0 || date.X < 0 || date.X+date.W > 480 {
						t.Fatal("date has no visible hit target")
					}
					tt.RightClickAt(date.X+date.W/2, date.Y+date.H/2)
					label := "Copy Date"
					if language == localeChinese {
						label = "复制日期"
					}
					item, ok := tt.AccessByRole(ui.RoleMenuItem, label)
					if !ok || item.Disabled {
						t.Fatal("date context menu lacks enabled copy action")
					}
					commit.Date = "2026-12-01T01:02:03Z"
					tt.Frame()
					tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
					if copied != expected || calls != 1 {
						t.Fatalf("copied refreshed, raw or truncated date: %q (%d calls), want %q", copied, calls, expected)
					}
					notices := a.notices[toastMain]
					if len(notices) != 1 {
						t.Fatalf("wrong feedback count: %d", len(notices))
					}
					if failure {
						if notices[0].kind != "error" || !strings.Contains(notices[0].title, "owned date clipboard failure") {
							t.Fatal("clipboard failure reported as success")
						}
					} else if notices[0].kind != "success" {
						t.Fatal("copy success missing")
					}
					if a.selectedSHA != commit.SHA || a.commitFields.Summary != "keep draft" || a.commitFields.Body != "keep body" || a.busy || a.loading {
						t.Fatal("date copy changed repository or draft state")
					}
				})
			}
		}
	}
}
