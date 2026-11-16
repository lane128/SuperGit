package lobedesk

import (
	"context"
	"fmt"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"testing"
)

func TestOverviewViewportReprioritizesPendingReads(t *testing.T) {
	a := &app{page: "history", worktreeSelected: true, locale: localeEnglish}
	keys := make([]detailFileKey, 12)
	for i := range keys {
		keys[i] = detailFileKey{Path: fmt.Sprintf("file-%02d.go", i)}
		a.overviewEntries = append(a.overviewEntries, worktreeOverviewEntry{Key: keys[i], Loading: true})
	}
	q := newOverviewReadQueue(keys)
	a.overviewQueue = q
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(500).Height(180).Children(func() {
			scroll := layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.commitInfoScroll}, func() { a.worktreeOverview(c) }).Grow(1).Basis(0)
			a.commitDiffViewport = scroll.Bounds()
		})
	}, 500, 180)
	for range 3 {
		tt.Frame()
	}
	header := func(key detailFileKey) ui.AccessInfo {
		t.Helper()
		node, ok := tt.AccessByRole(ui.RoleButton, key.Path+" · "+a.t("unstaged"))
		if !ok {
			t.Fatal("missing overview file header")
		}
		return node
	}
	first, second := header(keys[0]), header(keys[1])
	height := second.Bounds.Y - first.Bounds.Y
	if height <= 0 {
		t.Fatal("overview fixture lacks distinct blocks")
	}
	a.commitInfoScroll.Y = 400
	for range 3 {
		tt.Frame()
	}
	var want detailFileKey
	for index, key := range keys {
		// Ordinary section geometry determines the first visible key even when
		// headers outside the viewport and overscan no longer exist.
		y := first.Bounds.Y + float32(index)*height - a.commitInfoScroll.Y
		if y+height > a.commitDiffViewport.Y && y < a.commitDiffViewport.Y+a.commitDiffViewport.H {
			want = key
			break
		}
	}
	if want.Path == "" || want == keys[0] {
		t.Fatal("fixture did not scroll beyond first pending file")
	}
	visibleHeader := header(want)
	if visibleHeader.Bounds.Y+height <= a.commitDiffViewport.Y || visibleHeader.Bounds.Y >= a.commitDiffViewport.Y+a.commitDiffViewport.H {
		t.Fatal("first visible pending file header is outside the viewport")
	}
	got, ok := q.next(t.Context())
	if !ok || got != want {
		t.Fatalf("scroll prioritized %+v, want first visible %+v", got, want)
	}
}

func TestOverviewQueuePromotesVisiblePendingFiles(t *testing.T) {
	a, b, c, d := detailFileKey{Path: "a"}, detailFileKey{Path: "b"}, detailFileKey{Path: "c"}, detailFileKey{Path: "d", Staged: true}
	q := newOverviewReadQueue([]detailFileKey{a, b, c, d})
	next := func(want detailFileKey) {
		t.Helper()
		got, ok := q.next(t.Context())
		if !ok || got != want {
			t.Fatalf("read %+v, want %+v", got, want)
		}
	}
	next(a)
	q.prioritize([]detailFileKey{c, d})
	next(c)
	// Scrolling again must change the next pending read, without restarting c.
	q.prioritize([]detailFileKey{b})
	next(b)
	next(d)
	if !q.push(c) || q.push(c) {
		t.Fatal("retry enqueue was lost or duplicated")
	}
	next(c)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, ok := q.next(ctx); ok {
		t.Fatal("canceled empty queue returned a read")
	}
	q.push(a)
	if _, ok := q.next(ctx); ok {
		t.Fatal("canceled queue started another read")
	}
	next(a)
}
