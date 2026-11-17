package lobedesk

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestHistoryDensityKeepsTextAndRefsInsideRows(t *testing.T) {
	before := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(before) })
	for _, scale := range []float32{fontScaleSmall, fontScaleStandard, fontScaleLarge} {
		setUIFontScale(scale)
		for _, language := range []locale{localeChinese, localeEnglish} {
			for _, columns := range []struct{ author, time bool }{{true, true}, {true, false}, {false, true}, {false, false}} {
				t.Run(fmt.Sprintf("%s/%v/author=%v/time=%v", language, scale, columns.author, columns.time), func(t *testing.T) {
					commit := gitrepo.Commit{SHA: "head", Subject: "修复历史布局 with a long subject", Author: "作者 Long author name", Date: "2026-11-17T09:00:00Z", Refs: "HEAD -> main, tag: v1, tag: v2"}
					a := &app{locale: language, page: "history", selectedSHA: commit.SHA, showGraph: true, hideAuthor: !columns.author, hideTime: !columns.time}
					a.historySplit.rowWidth = 760
					tt := ui.NewTester(func(c *ui.Context) {
						theme.Apply(c)
						ui.Column(c).Width(310).Children(func() {
							a.historyWorktreeRow(c, 0, 2)
							a.commitRow(c, commit, 0, 1, 2, &commitGraph{})
						})
					}, 310, 200)
					tt.Frame()
					row, ok := tt.AccessByRole(ui.RoleOption, commit.Subject)
					if !ok {
						t.Fatal("commit row missing")
					}
					want := scaledFont(44)
					if scale == fontScaleSmall {
						want = 41.6
					}
					if !columns.author && !columns.time {
						want = scaledFont(34)
					}
					if !nearlyEqual(row.Bounds.H, want) {
						t.Fatalf("row height=%v want=%v", row.Bounds.H, want)
					}
					check := func(label string, bounds ui.Rect) {
						t.Helper()
						r, ok := tt.Find(label)
						// Find returns the row before its identically named title.
						// Its painted text is checked by the edge-ink assertion below.
						if ok && r == bounds {
							return
						}
						if !ok || r.Y < bounds.Y+scaledFont(4)-0.5 || r.Y+r.H > bounds.Y+bounds.H-scaledFont(4)+0.5 {
							t.Fatalf("%q exceeds row text area: text=%+v row=%+v", label, r, bounds)
						}
					}
					check(commit.Subject, row.Bounds)
					check("main", row.Bounds)
					if columns.author {
						check(commit.Author, row.Bounds)
					} else if _, ok := tt.Find(commit.Author); ok {
						t.Fatal("hidden author still rendered")
					}
					if columns.time {
						check(formatDateTime(commit.Date), row.Bounds)
					} else if _, ok := tt.Find(formatDateTime(commit.Date)); ok {
						t.Fatal("hidden time still rendered")
					}
					work, ok := tt.AccessByRole(ui.RoleOption, a.t("noChangesShort"))
					if !ok || !nearlyEqual(work.Bounds.H, row.Bounds.H) || !nearlyEqual(work.Bounds.Y+work.Bounds.H, row.Bounds.Y) {
						t.Fatal("worktree and commit row geometry diverged")
					}
					check(a.t("noChangesShort"), work.Bounds)
					for _, bounds := range []ui.Rect{work.Bounds, row.Bounds} {
						pixels := tt.Image()
						ground := pixels.RGBAAt(int(bounds.X+bounds.W/2), int(bounds.Y+bounds.H-1))
						for y := int(bounds.Y) + 1; y < int(bounds.Y+bounds.H)-1; y++ {
							if float32(y) >= bounds.Y+scaledFont(4)-1 && float32(y) < bounds.Y+bounds.H-scaledFont(4)+1 {
								continue
							}
							for x := int(bounds.X) + 32; x < int(bounds.X+bounds.W)-32; x++ {
								if pixels.RGBAAt(x, y) != ground {
									t.Fatalf("row text/reference ink enters vertical padding at %d,%d: row=%+v", x, y, bounds)
								}
							}
						}
					}
					if columns.author || columns.time {
						check(a.t("commitChanges"), work.Bounds)
					} else if _, ok := tt.Find(a.t("commitChanges")); ok {
						t.Fatal("compact worktree row still contains a second line")
					}
				})
			}
		}
	}
}
