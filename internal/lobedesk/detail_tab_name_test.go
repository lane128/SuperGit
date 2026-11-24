package lobedesk

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestDetailNameElisionKeepsWholeGraphemes(t *testing.T) {
	// Each side of the ellipsis must end/start at one of these hand-written
	// cluster boundaries, including marks, joined emoji and regional flags.
	clusters := []string{"c", "a", "f", "é", "-", "👩‍💻", "-", "🇨🇳", "-", "校", "验", "-", "f", "i", "l", "e", ".", "g", "o"}
	name := strings.Join(clusters, "")
	prefixes, suffixes := map[string]bool{"": true}, map[string]bool{"": true}
	for i := range clusters {
		prefixes[strings.Join(clusters[:i+1], "")] = true
		suffixes[strings.Join(clusters[i:], "")] = true
	}
	view := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		measure := func(s string) float32 { w, _ := c.MeasureText(0, ui.Span{Text: s, Size: 12, Weight: 600}); return w }
		if got := elideDetailName(name, 1000, measure); got != name {
			t.Errorf("short name changed: %q", got)
		}
		for width := float32(0); width < measure(name); width += 3 {
			got := elideDetailName(name, width, measure)
			if measure(got) > width+.01 {
				t.Errorf("width %g overflows: %q", width, got)
			}
			if got == "" {
				continue
			}
			parts := strings.Split(got, "…")
			if len(parts) != 2 || !prefixes[parts[0]] || !suffixes[parts[1]] {
				t.Errorf("split grapheme at width %g: %q", width, got)
			}
			if width >= 70 && (!strings.HasPrefix(got, "c") || !strings.HasSuffix(got, ".go")) {
				t.Errorf("name/extension lost: %q", got)
			}
		}
	}, 300, 50)
	view.Frame()
}

func TestDetailTabSelectionKeepsWidths(t *testing.T) {
	a := &app{locale: localeEnglish, detailTabViewport: 230}
	files := []detailFileKey{{Path: "short.go"}, {Path: "café-👩‍💻-configuration-validation-long-name.go"}}
	for _, file := range files {
		a.detailTabs().Open(file)
	}
	view := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Row(c).Children(func() {
			for _, file := range files {
				a.detailTabButton(c, theme.TokensOf(c), detailTabKey{File: file})
			}
		})
	}, 600, 60)
	bounds := map[string]ui.Rect{}
	for _, file := range files {
		tab, _ := view.AccessByRole(ui.RoleTab, file.Path)
		bounds[file.Path] = tab.Bounds
	}
	a.detailTabs().Activate(detailTabKey{File: files[0]})
	view.Frame()
	for _, file := range files {
		tab, _ := view.AccessByRole(ui.RoleTab, file.Path)
		if tab.Bounds != bounds[file.Path] {
			t.Errorf("selection shifted %s: %v to %v", file.Path, bounds[file.Path], tab.Bounds)
		}
		if tab.Bounds.W+24 > 230 {
			t.Errorf("tab exceeds viewport: %v", tab.Bounds)
		}
	}
}
