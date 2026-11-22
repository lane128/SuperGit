package lobedesk

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

func summaryMetadataView(a *app, width int) *ui.Tester {
	a.detailsDiffHidden = true
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Children(func() { a.detailOverview(c) })
	}, width, 360)
}

func TestHistorySummaryMetadataColumns(t *testing.T) {
	commit := gitrepo.Commit{
		SHA: "1234567890abcdef1234567890abcdef12345678", Subject: "Release candidate",
		Author: "Avery", Date: time.Date(2026, 11, 19, 12, 34, 56, 0, time.Local).Format(time.RFC3339),
		Parents: []string{"abcdef0123456789abcdef0123456789abcdef01"},
	}
	a := &app{locale: localeEnglish, selectedSHA: commit.SHA, commits: []gitrepo.Commit{commit}}
	view := summaryMetadataView(a, 720)
	for _, width := range []int{720, 292} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			view.SetSize(width, 360)
			view.Frame()
			var firstKey, firstValue ui.Rect
			var previousBottom float32
			for _, row := range []struct{ key, value string }{
				{"Hash", commit.SHA}, {"Author", "Avery"}, {"Date", "2026-11-19 12:34:56"}, {"Parent", "abcdef01"},
			} {
				key, foundKey := view.Find(row.key)
				value, foundValue := view.Find(row.value)
				if row.key == "Parent" {
					// The link itself starts at the value column (no button padding).
					parent, ok := view.AccessByRole(ui.RoleLink, row.value)
					value, foundValue = parent.Bounds, ok && !parent.Offscreen
				}
				if !foundValue {
					t.Errorf("missing %s value %q; rendered %q", row.key, row.value, view.Texts())
					continue
				}
				if value.Y < previousBottom-1 {
					t.Errorf("%s value starts at y=%.1f; previous value ends at y=%.1f", row.key, value.Y, previousBottom)
				}
				previousBottom = value.Y + value.H
				if !foundKey {
					t.Errorf("missing %s label; rendered %q", row.key, view.Texts())
					continue
				}
				if key.W <= 0 || key.H <= 0 || value.W <= 0 || value.H <= 0 || key.X < 0 || key.X+key.W > value.X+1 || value.X+value.W > float32(width)+1 {
					t.Errorf("%s label and value must fit side by side: label=%+v value=%+v", row.key, key, value)
				}
				if min(key.Y+key.H, value.Y+value.H) <= max(key.Y, value.Y) {
					t.Errorf("%s label and value must share a row: label=%+v value=%+v", row.key, key, value)
				}
				if firstKey.W == 0 {
					firstKey, firstValue = key, value
				} else if math.Abs(float64(key.X-firstKey.X)) > 1 || math.Abs(float64(value.X-firstValue.X)) > 1 {
					t.Errorf("%s starts at (%.1f, %.1f); metadata columns start at (%.1f, %.1f)", row.key, key.X, value.X, firstKey.X, firstValue.X)
				}
			}
		})
	}
}

func TestHistorySummaryMetadataAuthorIdentity(t *testing.T) {
	for _, tc := range []struct{ name, email, want string }{
		{"with email", "avery@example.invalid", "Avery<avery@example.invalid>"},
		{"without email", "", "Avery"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			commit := gitrepo.Commit{SHA: "12345678", Subject: "Release candidate", Author: "Avery", AuthorEmail: tc.email}
			a := &app{locale: localeEnglish, selectedSHA: commit.SHA, commits: []gitrepo.Commit{commit}}
			view := summaryMetadataView(a, 720)
			if bounds, ok := view.Find(tc.want); !ok || bounds.W <= 0 || bounds.H <= 0 {
				t.Errorf("expected visible author %q; rendered %q", tc.want, view.Texts())
			}
			if tc.email == "" && (view.HasText("<") || view.HasText(">")) {
				t.Errorf("empty email left angle brackets in the Summary: %q", view.Texts())
			}
		})
	}
}

func TestHistorySummaryMetadataCopyPlacement(t *testing.T) {
	commit := gitrepo.Commit{SHA: "1234567890abcdef1234567890abcdef12345678", Subject: "Release candidate"}
	a := &app{locale: localeEnglish, selectedSHA: commit.SHA, commits: []gitrepo.Commit{commit}}
	view := summaryMetadataView(a, 540)
	var wideOffset float32
	for _, width := range []int{540, 720, 292} {
		view.SetSize(width, 360)
		view.Frame()
		hash, found := view.Find(commit.SHA)
		button, ok := view.AccessByRole(ui.RoleButton, "Copy SHA")
		if !found || hash.W <= 0 || !ok || button.Offscreen || button.Disabled {
			t.Fatalf("at %d DIP hash or copy button unavailable: hash=%+v button=%+v", width, hash, button)
		}
		b := button.Bounds
		gap := b.X - (hash.X + hash.W)
		if gap < -1 || gap > b.W || b.X+b.W > float32(width)+1 || min(b.Y+b.H, hash.Y+hash.H) <= max(b.Y, hash.Y) {
			t.Errorf("at %d DIP copy button should fit just after the SHA on the same row: hash=%+v button=%+v", width, hash, b)
		}
		// Find reports the text element's box, which can stretch far beyond its ink.
		// Once the full SHA fits, extra pane width must not move its copy button.
		offset := b.X - hash.X
		switch width {
		case 540:
			wideOffset = offset
		case 720:
			if math.Abs(float64(offset-wideOffset)) > 1 {
				t.Errorf("widening the pane moved Copy SHA from %.1f to %.1f DIP from the hash start", wideOffset, offset)
			}
		}
	}
}

func TestHistorySummaryMetadataCopiesCurrentSHA(t *testing.T) {
	a, copied := copySHAPaletteFixture(t)
	next := gitrepo.Commit{SHA: "0123456789abcdef0123456789abcdef01234568", Subject: "Next commit"}
	a.commits = append(a.commits, next)
	view := summaryMetadataView(a, 292)
	for i, commit := range a.commits {
		a.selectedSHA = commit.SHA
		view.Frame()
		button, ok := view.AccessByRole(ui.RoleButton, "Copy SHA")
		if !ok || button.Offscreen || button.Disabled {
			t.Fatalf("Copy SHA unavailable for %s: %+v", commit.SHA, button)
		}
		b := button.Bounds
		view.ClickAt(b.X+b.W/2, b.Y+b.H/2)
		if len(*copied) != i+1 || (*copied)[i] != commit.SHA {
			t.Fatalf("after selecting %s, clipboard writes = %q", commit.SHA, *copied)
		}
	}
}
