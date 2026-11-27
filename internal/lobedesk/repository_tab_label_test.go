package lobedesk

import (
	"bytes"
	"context"
	"image"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestRepositoryTabsDistinguishMatchingParentPaths(t *testing.T) {
	root := t.TempDir()
	first := &app{repo: &gitrepo.Repository{Root: filepath.Join(root, "client-a", "apps", "widget")}, locale: localeEnglish}
	second := &app{repo: &gitrepo.Repository{Root: filepath.Join(root, "client-b", "apps", "widget")}}
	third := &app{repo: &gitrepo.Repository{Root: filepath.Join(root, "client-c", "tools", "widget")}}
	tabs := newRepositoryTabs(context.Background(), first)
	tabs.sessions = append(tabs.sessions, second, third)
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		tabs.strip(c, first)
	}, 980, 100)
	tt.Frame()
	for _, want := range []struct {
		name string
		app  *app
	}{
		{"widget · " + filepath.Join("client-a", "apps"), first},
		{"widget · " + filepath.Join("client-b", "apps"), second},
		{"widget · tools", third},
	} {
		node, ok := tt.AccessByRole(ui.RoleTab, want.name)
		if !ok || node.Description != want.app.repo.Root || node.Offscreen {
			t.Errorf("could not distinguish %q in the tab strip: %+v", want.name, node)
		}
	}
}

func TestRepositoryTabsShowDifferingMiddleOfLongCheckoutPaths(t *testing.T) {
	root := t.TempDir()
	first := &app{repo: &gitrepo.Repository{Root: filepath.Join(root, "abcdefgh-A-identical-trailer", "apps", "widget")}, locale: localeEnglish}
	second := &app{repo: &gitrepo.Repository{Root: filepath.Join(root, "abcdefgh-B-identical-trailer", "apps", "widget")}}
	tabs := newRepositoryTabs(context.Background(), first)
	tabs.sessions = append(tabs.sessions, second)
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		tabs.strip(c, first)
	}, 980, 100)
	tt.Frame()

	var captions []string
	for _, text := range tt.Texts() {
		if strings.HasPrefix(text, "widget · ") && len([]rune(text)) <= 29 {
			captions = append(captions, text)
		}
	}
	if len(captions) != 2 || !strings.Contains(captions[0], "-A-") || !strings.Contains(captions[1], "-B-") {
		t.Fatalf("visible tab captions should preserve the differing path text, got %q", captions)
	}
}

func TestRepositoryTabsKeepLongNamesDistinctAtLargeTextSize(t *testing.T) {
	oldScale := uiFontScaleValue()
	setUIFontScale(fontScaleLarge)
	t.Cleanup(func() { setUIFontScale(oldScale) })
	root := t.TempDir()
	prefix := "payments-platform-international-migration-"
	first := &app{repo: &gitrepo.Repository{Root: filepath.Join(root, prefix+"api")}, locale: localeEnglish}
	second := &app{repo: &gitrepo.Repository{Root: filepath.Join(root, prefix+"web")}}
	tabs := newRepositoryTabs(context.Background(), first)
	tabs.sessions = append(tabs.sessions, second)
	widths := map[string]float32{}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		for _, text := range []string{prefix + "api", prefix + "web"} {
			widths[text], _ = c.MeasureText(0, ui.Span{Text: text, Size: scaledFont(11.5), Weight: 700})
		}
		tabs.strip(c, first)
	}, 980, 100)
	tt.Frame()
	for _, name := range []string{prefix + "api", prefix + "web"} {
		if widths[name] <= 148 {
			t.Fatalf("fixture name %q fits without truncation", name)
		}
		if _, ok := tt.AccessByRole(ui.RoleTab, name); !ok {
			t.Errorf("accessible tab label lost the full name %q", name)
		}
	}
	for _, suffix := range []string{"api", "web"} {
		if !slices.ContainsFunc(tt.Texts(), func(text string) bool {
			return strings.Contains(text, "…") && strings.HasSuffix(text, suffix)
		}) {
			t.Errorf("caption did not preserve visible distinguishing suffix %q", suffix)
		}
	}
}

// A caption must be elided against the size it is drawn at. This name fits a
// 13 DIP caption but not a 14 DIP one, so measuring the wrong size would hide
// a name that fits instead of showing it whole.
func TestRepositoryTabCaptionsMeasureTheCurrentFontSize(t *testing.T) {
	setUIFontScale(fontScaleStandard)
	defer setUIFontScale(fontScaleStandard)
	budget := repositoryCaptionWidth(repositoryCardWidth(300, 1))
	var checked bool
	tt := ui.NewTester(func(c *ui.Context) {
		if checked {
			return
		}
		for length := 4; length <= 40 && !checked; length++ {
			text := strings.Repeat("m", length)
			current := ui.Span{Text: text, Size: scaledFont(13), Weight: 500}
			small, _ := c.MeasureText(0, current)
			current.Size = scaledFont(14)
			current.Weight = 400
			if large, _ := c.MeasureText(0, current); small > budget || large <= budget {
				continue
			}
			session := &app{locale: localeEnglish, repo: &gitrepo.Repository{Root: filepath.Join(t.TempDir(), text)}}
			tabs := newRepositoryTabs(context.Background(), session)
			name := tabs.checkoutTabNames(c, tabs.sessions, budget)[session]
			if name.name != text || name.caption != text {
				t.Fatalf("caption %q of %q was elided for a %v DIP budget", name.caption, text, budget)
			}
			checked = true
		}
	}, 300, 40)
	tt.Frame()
	if !checked {
		t.Fatal("no test name straddled the caption width budget")
	}
}

// Labels are drawn at weight 500, so a name that only overflows when bolded
// must keep its full caption; selection is the fixed line, not weight.
func TestRepositoryTabCaptionsUseNormalWeight(t *testing.T) {
	setUIFontScale(fontScaleStandard)
	defer setUIFontScale(fontScaleStandard)
	budget := repositoryCaptionWidth(repositoryCardWidth(300, 1))
	var checked bool
	tt := ui.NewTester(func(c *ui.Context) {
		if checked {
			return
		}
		for wide := 1; wide <= 20 && !checked; wide++ {
			for narrow := 0; narrow <= 30 && !checked; narrow++ {
				text := strings.Repeat("m", wide) + strings.Repeat("i", narrow)
				span := ui.Span{Text: text, Size: scaledFont(13), Weight: 500}
				normal, _ := c.MeasureText(0, span)
				span.Weight = 600
				if bold, _ := c.MeasureText(0, span); normal > budget || bold <= budget {
					continue
				}
				session := &app{locale: localeEnglish, repo: &gitrepo.Repository{Root: filepath.Join(t.TempDir(), text)}}
				tabs := newRepositoryTabs(context.Background(), session)
				if name := tabs.checkoutTabNames(c, tabs.sessions, budget)[session]; name.caption != text {
					t.Fatalf("weight-500 caption %q was elided to %q", text, name.caption)
				}
				checked = true
			}
		}
	}, 300, 40)
	tt.Frame()
	if !checked {
		t.Fatal("no test name straddled the weight-sensitive budget")
	}
}

func TestRepositoryTabsPaintDistinctCJKCheckoutPaths(t *testing.T) {
	const prefix = "国际化多语言产品管理平台任务队列"
	assertDistinctTabPaint(t,
		filepath.Join(prefix+"东区", "widget"), "widget · "+prefix+"东区",
		filepath.Join(prefix+"西区", "widget"), "widget · "+prefix+"西区")
}

func TestRepositoryTabsPaintDistinctVisuallyEquivalentCheckoutNames(t *testing.T) {
	t.Run("zero width space", func(t *testing.T) {
		const plain, invisible = "payments-production", "payments-​production"
		assertDistinctTabPaint(t, plain, plain, invisible, invisible)
	})
	t.Run("canonical equivalence", func(t *testing.T) {
		const composed, decomposed = "café", "café"
		assertDistinctTabPaint(t, composed, composed, decomposed, decomposed)
	})
}

func assertDistinctTabPaint(t *testing.T, firstPath, firstLabel, secondPath, secondLabel string) {
	t.Helper()
	oldScale := uiFontScaleValue()
	setUIFontScale(fontScaleStandard)
	t.Cleanup(func() { setUIFontScale(oldScale) })

	root := t.TempDir()
	first := &app{repo: &gitrepo.Repository{Root: filepath.Join(root, firstPath)}, locale: localeEnglish}
	second := &app{repo: &gitrepo.Repository{Root: filepath.Join(root, secondPath)}, locale: localeEnglish}
	other := &app{repo: &gitrepo.Repository{Root: filepath.Join(root, "notes")}, locale: localeEnglish}
	tabs := newRepositoryTabs(context.Background(), first)
	active := other
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		tabs.strip(c, active)
	}, 980, 100)

	for _, selected := range []bool{false, true} {
		var captions [][]byte
		var origin image.Point
		for _, candidate := range []struct {
			tab, peer *app
			label     string
		}{
			{first, second, firstLabel},
			{second, first, secondLabel},
		} {
			tabs.sessions = []*app{candidate.tab, candidate.peer, other}
			if selected {
				active = candidate.tab
			}
			tt.Frame()
			node, ok := tt.AccessByRole(ui.RoleTab, candidate.label)
			if !ok || node.Offscreen || node.Selected != selected || node.Description != candidate.tab.repo.Root {
				t.Fatalf("expected visible checkout tab for %q, got %+v", candidate.label, node)
			}
			// Compare the text at the same position, excluding the icon and close button.
			point := image.Pt(int(node.Bounds.X)+26, int(node.Bounds.Y)+4)
			if len(captions) == 0 {
				origin = point
			} else if point != origin {
				t.Fatalf("tab moved between paints: %v and %v", origin, point)
			}
			width := int(node.Bounds.W) - 26 - 8
			clip := image.Rectangle{Min: point, Max: point.Add(image.Pt(width, 19))}
			pixels := tt.Image().SubImage(clip).(*image.RGBA)
			var caption []byte
			for y := clip.Min.Y; y < clip.Max.Y; y++ {
				start := pixels.PixOffset(clip.Min.X, y)
				caption = append(caption, pixels.Pix[start:start+4*width]...)
			}
			captions = append(captions, caption)
		}
		if bytes.Equal(captions[0], captions[1]) {
			t.Errorf("%q and %q paint identical captions (selected=%t)", firstLabel, secondLabel, selected)
		}
	}
}

func TestRepositoryTabsKeepCaptionsDistinctAcrossElisionGroups(t *testing.T) {
	root := t.TempDir()
	var sessions []*app
	for _, parent := range []string{
		"abcdefgh-A-path-XXXXXXXXXXX", "abcdefgh-B-path-XXXXXXXXXXX",
		"abcdefgh-A-path-YYYYYYYYYYY", "abcdefgh-B-path-YYYYYYYYYYY",
	} {
		sessions = append(sessions, &app{repo: &gitrepo.Repository{Root: filepath.Join(root, parent, "widget")}, locale: localeEnglish})
	}
	tabs := newRepositoryTabs(context.Background(), sessions[0])
	tabs.sessions = sessions
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		tabs.strip(c, sessions[0])
	}, 980, 100)
	tt.Frame()

	captions := make(map[string]bool)
	for _, text := range tt.Texts() {
		if strings.Contains(text, "…") {
			if captions[text] {
				t.Errorf("repeated visible checkout caption %q", text)
			}
			captions[text] = true
		}
	}
	if len(captions) != 4 {
		t.Errorf("want four distinguishable captions, got %v", captions)
	}
}

func TestRepositoryTabsAvoidCollisionsWithLiteralNumberedNames(t *testing.T) {
	oldScale := uiFontScaleValue()
	setUIFontScale(fontScaleStandard)
	t.Cleanup(func() { setUIFontScale(oldScale) })

	root := t.TempDir()
	const literal = "1 · widget · …-A-pa…"
	var sessions []*app
	for _, path := range []string{
		filepath.Join("abcdefgh-A-path-XXXXXXXXXXX", "widget"),
		filepath.Join("abcdefgh-B-path-XXXXXXXXXXX", "widget"),
		filepath.Join("abcdefgh-A-path-YYYYYYYYYYY", "widget"),
		filepath.Join("abcdefgh-B-path-YYYYYYYYYYY", "widget"),
		literal,
	} {
		sessions = append(sessions, &app{repo: &gitrepo.Repository{Root: filepath.Join(root, path)}, locale: localeEnglish})
	}
	tabs := newRepositoryTabs(context.Background(), sessions[0])
	tabs.sessions = slices.Clone(sessions)
	var names map[*app]checkoutTabName
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		names = tabs.checkoutTabNames(c, tabs.sessions, 148)
	}, 980, 100)
	tt.Frame()

	seen := make(map[string]string)
	for _, session := range sessions {
		label := names[session]
		if owner, duplicate := seen[label.caption]; duplicate {
			t.Errorf("%q and %q share the visible caption %q", owner, session.repo.Root, label.caption)
		}
		seen[label.caption] = session.repo.Root
	}
	if got := names[sessions[4]].caption; got != literal {
		t.Errorf("the literal repository name changed to %q", got)
	}
	before := names
	slices.Reverse(tabs.sessions)
	tt.Frame()
	for _, session := range sessions {
		if names[session].caption != before[session].caption {
			t.Errorf("reordering tabs changed %q to %q", before[session].caption, names[session].caption)
		}
	}
}
