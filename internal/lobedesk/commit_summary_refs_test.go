package lobedesk

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestHistorySummaryKeepsLongCurrentBranchReadable(t *testing.T) {
	oldScale := uiFontScaleValue()
	setUIFontScale(fontScaleLarge)
	t.Cleanup(func() { setUIFontScale(oldScale) })

	head := "release/" + strings.Repeat("W", 43) + "-production"
	commit := gitrepo.Commit{
		SHA: strings.Repeat("ab", 20), Subject: "Release candidate",
		Refs: "HEAD -> " + head + ", tag: v2.3.4, tag: v2.3.5",
	}
	a := &app{locale: localeEnglish, themeMode: "light"}
	view := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.commitInfoContent(c, commit, theme.TokensOf(c))
	}, 292, 260)
	for range 4 {
		view.Frame()
	}
	branch, inline := view.Access(head)
	more, disclosed := view.AccessByRole(ui.RoleDisclosure, "2 more")
	if !inline || branch.Offscreen || !disclosed || more.Offscreen || branch.Bounds.Y != more.Bounds.Y {
		t.Fatalf("expected the current branch and two-tag disclosure on the same line, got branch=%+v, disclosure=%+v, texts=%v", branch, more, view.Texts())
	}
	for _, text := range view.Texts() {
		if strings.HasPrefix(text, "relea") && strings.Contains(text, "…") && strings.HasSuffix(text, "duction") {
			return
		}
	}
	t.Fatalf("the shortened branch lost its distinguishing suffix: %v", view.Texts())
}

func TestHistorySummaryMovesFocusWhenRefDisclosureFitsAway(t *testing.T) {
	commit := gitrepo.Commit{
		SHA: strings.Repeat("42", 20), Subject: "Release candidate",
		Refs: "tag: release-candidate-01, tag: release-candidate-02, tag: release-candidate-03",
	}
	a := &app{locale: localeEnglish, themeMode: "light"}
	view := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		a.commitInfoContent(c, commit, theme.TokensOf(c))
	}, 300, 360)
	for range 3 {
		view.Frame()
	}
	view.Key(0, ui.KeyTab)
	if !view.Focused("release-candidate-01") {
		t.Fatal("expected the first tag in the narrow Summary")
	}
	view.Key(0, ui.KeyTab)
	if !view.Focused("2 more") {
		t.Fatal("expected to focus the disclosure before resizing")
	}
	view.SetSize(640, 360)
	for range 4 {
		view.Frame()
	}
	if _, ok := view.AccessByRole(ui.RoleDisclosure, "2 more"); ok {
		t.Fatal("all tags fit but the disclosure is still exposed")
	}
	if !view.Focused("release-candidate-02") {
		t.Fatal("focus did not move from the disappearing disclosure to the first newly visible tag")
	}
}

func TestHistorySummaryKeepsCommitReadableWithManyRefsOnResize(t *testing.T) {
	const height = 640
	const body = "Regression: keep the release notes visible."
	const parent = "a5a5a5a5"
	refs := make([]string, 24)
	for i := range refs {
		refs[i] = fmt.Sprintf("tag: release-candidate-%02d", i+1)
	}
	commit := gitrepo.Commit{
		SHA: strings.Repeat("31", 20), Parents: []string{parent}, Subject: "Release candidate",
		Author: "Avery Contributor", Refs: strings.Join(refs, ", "), Date: "2026-11-19T12:34:56Z",
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	a := &app{
		ctx: ctx, runOnMain: func(func()) {}, locale: localeEnglish, page: "history", showGraph: true, themeMode: "light",
		repo: &gitrepo.Repository{Root: t.TempDir()}, commits: []gitrepo.Commit{commit, {SHA: parent, Subject: "Previous release"}},
		selectedSHA: commit.SHA, commitFilesLoaded: true, commitMessage: commit.Subject + "\n\n" + body,
	}
	view := ui.NewTester(a.root, 1600, height)
	for _, width := range []int{1600, 1320, 980, 1600} {
		view.SetSize(width, height)
		for range 4 {
			view.Frame()
		}
		bodyBounds, found := view.Find(body)
		if !found || bodyBounds.W <= 0 || bodyBounds.H <= 0 || bodyBounds.Y+bodyBounds.H > height {
			t.Errorf("at %d × %d the short message is outside the initial Summary viewport: %+v", width, height, bodyBounds)
		}
		for _, label := range []string{a.t("copySHA"), parent} {
			node, ok := view.AccessByRole(ui.RoleButton, label)
			if !ok || node.Offscreen || node.Disabled || node.Bounds.Y+node.Bounds.H > height {
				t.Errorf("at %d × %d Summary action %q is outside the initial viewport: %+v", width, height, label, node)
			}
		}
		if a.selectedSHA != commit.SHA || a.commitInfoScroll.Y != 0 || a.detailTabShown() != detailTabSummary {
			t.Fatalf("at width %d resize changed the selected commit, Summary tab, or scroll position", width)
		}
		if width == 1600 {
			if ref, ok := view.Find("release-candidate-02"); !ok || ref.W <= 0 || ref.H <= 0 {
				t.Errorf("at width %d the second tag has room to display inline: %+v", width, ref)
			}
		}
		if width != 980 {
			continue
		}
		more, ok := view.AccessByRole(ui.RoleDisclosure, "23 more")
		if !ok || more.Expanded || more.Offscreen || more.Disabled {
			t.Fatalf("narrow Summary lacks an enabled disclosure for 23 tags: %+v", more)
		}
		view.ClickAt(more.Bounds.X+more.Bounds.W/2, more.Bounds.Y+more.Bounds.H/2)
		view.Frame()
		expanded, ok := view.AccessByRole(ui.RoleDisclosure, "Show fewer refs")
		if !ok || !expanded.Expanded || expanded.Bounds != more.Bounds {
			t.Fatalf("expanded disclosure lost its position or state: %+v", expanded)
		}
		if ref, ok := view.Find("release-candidate-02"); !ok || ref.W <= 0 || ref.H <= 0 {
			t.Fatalf("expanding did not reveal the first folded tag: %+v", ref)
		}
		view.Key(0, ui.KeyTab)
		view.Frame()
		if !view.Focused("release-candidate-02") {
			t.Fatal("Tab from the disclosure did not focus the first expanded tag")
		}
		view.SetSize(1600, height)
		for range 4 {
			view.Frame()
		}
		if !view.Focused("release-candidate-02") {
			t.Fatal("widening moved the focused tag out of keyboard focus")
		}
		view.Key(ui.Shift, ui.KeyF10)
		view.Frame()
		if copy, ok := view.AccessByRole(ui.RoleMenuItem, a.t("copyTagName")); !ok || copy.Disabled {
			t.Fatalf("the focused tag lost its original context menu: %+v", copy)
		}
		view.Key(0, ui.KeyEscape)
		view.Frame()
		if !view.Focused("release-candidate-02") {
			t.Fatal("closing the tag context menu lost keyboard focus")
		}
		view.SetSize(980, height)
		for range 4 {
			view.Frame()
		}
		view.Key(ui.Shift, ui.KeyTab)
		view.Key(0, ui.KeyEnter)
		view.Frame()
		if collapsed, ok := view.AccessByRole(ui.RoleDisclosure, "23 more"); !ok || collapsed.Expanded {
			t.Fatalf("Enter did not collapse the focused disclosure: %+v", collapsed)
		}
	}
}
