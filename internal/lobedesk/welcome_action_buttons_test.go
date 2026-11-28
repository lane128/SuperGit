package lobedesk

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestWelcomeAndMenuActionCopyWithoutEllipsis(t *testing.T) {
	cases := []struct {
		name    string
		locale  locale
		choose  string
		newRepo string
		clone   string
	}{
		{
			name:    "english",
			locale:  localeEnglish,
			choose:  "Open Repository",
			newRepo: "New Repository",
			clone:   "Clone Repository",
		},
		{
			name:    "chinese",
			locale:  localeChinese,
			choose:  "打开仓库",
			newRepo: "创建仓库",
			clone:   "克隆仓库",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &app{locale: tc.locale, page: "history"}
			if got := a.t("chooseFolder"); got != tc.choose || strings.Contains(got, "…") {
				t.Fatalf("chooseFolder = %q, want %q without ellipsis", got, tc.choose)
			}
			if got := a.t("newRepo"); got != tc.newRepo || strings.Contains(got, "…") {
				t.Fatalf("newRepo = %q, want %q without ellipsis", got, tc.newRepo)
			}
			if got := a.t("cloneRepo"); got != tc.clone || strings.Contains(got, "…") {
				t.Fatalf("cloneRepo = %q, want %q without ellipsis", got, tc.clone)
			}

			// Verify Welcome page exposes the exact labels without ellipsis.
			welcome := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				a.emptyPage(c)
			}, 1280, 800)
			for _, label := range []string{tc.choose, tc.newRepo, tc.clone} {
				btn, ok := welcome.AccessByRole(ui.RoleButton, label)
				if !ok || btn.Offscreen || btn.Disabled {
					t.Fatalf("Welcome missing interactive button %q: %+v", label, btn)
				}
			}

			// Verify sidebar repository context menu exposes exact newRepo/cloneRepo labels without ellipsis.
			repoApp, _, _ := worktreeApp(t)
			repoApp.locale = tc.locale
			sidebar := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				ui.Column(c).Fill().Children(func() {
					repoApp.sidebarHeader(c)
				})
			}, 980, 640)
			title, ok := sidebar.AccessByRole(ui.RoleButton, repoApp.repoName())
			if !ok {
				t.Fatalf("missing repository menu button %q", repoApp.repoName())
			}
			sidebar.ClickAt(title.Bounds.X+title.Bounds.W/2, title.Bounds.Y+title.Bounds.H/2)
			for _, label := range []string{tc.newRepo, tc.clone} {
				item, ok := sidebar.AccessByRole(ui.RoleMenuItem, label)
				if !ok || item.Offscreen || item.Disabled {
					t.Fatalf("sidebar repoMenu missing enabled item %q: %+v", label, item)
				}
			}
		})
	}
}

func TestWelcomeActionButtonsEqualWidthAndAlignment(t *testing.T) {
	prevScale := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(prevScale) })

	locales := []struct {
		name   string
		locale locale
		labels [3]string
	}{
		{name: "en", locale: localeEnglish, labels: [3]string{"Open Repository", "New Repository", "Clone Repository"}},
		{name: "zh", locale: localeChinese, labels: [3]string{"打开仓库", "创建仓库", "克隆仓库"}},
	}
	sizes := []struct {
		w, h int
	}{
		{1280, 800},
		{980, 640},
	}
	scales := []struct {
		name  string
		scale float32
	}{
		{name: "small", scale: fontScaleSmall},
		{name: "standard", scale: fontScaleStandard},
		{name: "large", scale: fontScaleLarge},
	}

	for _, loc := range locales {
		for _, sz := range sizes {
			for _, sc := range scales {
				name := fmt.Sprintf("%s_%dx%d_%s", loc.name, sz.w, sz.h, sc.name)
				t.Run(name, func(t *testing.T) {
					setUIFontScale(sc.scale)
					a := &app{locale: loc.locale, page: "history"}
					tt := ui.NewTester(func(c *ui.Context) {
						theme.Apply(c)
						a.emptyPage(c)
					}, sz.w, sz.h)

					var bounds [3]ui.Rect
					for i, label := range loc.labels {
						btn, ok := tt.AccessByRole(ui.RoleButton, label)
						if !ok || btn.Offscreen || btn.Disabled {
							t.Fatalf("missing interactive button %q: %+v", label, btn)
						}
						if btn.Bounds.X < 0 || btn.Bounds.Y < 0 ||
							btn.Bounds.X+btn.Bounds.W > float32(sz.w) || btn.Bounds.Y+btn.Bounds.H > float32(sz.h) {
							t.Fatalf("button %q overflows viewport (%dx%d): %+v", label, sz.w, sz.h, btn.Bounds)
						}
						bounds[i] = btn.Bounds
					}

					wantW := scaledFont(220)
					for i, b := range bounds {
						if math.Abs(float64(b.W-wantW)) > 0.5 {
							t.Fatalf("button[%d] %q width = %.2f, want %.2f", i, loc.labels[i], b.W, wantW)
						}
						if math.Abs(float64(b.W-bounds[0].W)) > 0.5 {
							t.Fatalf("button[%d] width %.2f != button[0] width %.2f", i, b.W, bounds[0].W)
						}
						if math.Abs(float64(b.X-bounds[0].X)) > 0.5 {
							t.Fatalf("button[%d] X %.2f != button[0] X %.2f", i, b.X, bounds[0].X)
						}
					}

					for i := 0; i < 2; i++ {
						gap := bounds[i+1].Y - (bounds[i].Y + bounds[i].H)
						if math.Abs(float64(gap-8)) > 0.5 {
							t.Fatalf("gap between button[%d] and button[%d] = %.2f, want 8", i, i+1, gap)
						}
					}
				})
			}
		}
	}

	t.Run("error_recent_large_min_window", func(t *testing.T) {
		setUIFontScale(fontScaleLarge)
		a := &app{
			locale:  localeEnglish,
			page:    "history",
			loadErr: "/tmp/not-a-repo is not a Git repository.",
			recentRepositories: []string{
				"/tmp/recent-alpha",
				"/tmp/recent-beta",
				"/tmp/recent-gamma",
			},
		}
		tabs := newRepositoryTabs(t.Context(), a)
		t.Cleanup(tabs.shutdown)
		tt := ui.NewTester(tabs.view, 980, 640)
		tt.SetTitleBar(ui.TitleBar{Height: 38, Left: 72})
		tt.Frame()

		labels := [3]string{"Open Repository", "New Repository", "Clone Repository"}
		wantW := scaledFont(220)
		var btnBounds [3]ui.Rect
		for i, label := range labels {
			btn, ok := tt.AccessByRole(ui.RoleButton, label)
			if !ok || btn.Offscreen || btn.Disabled {
				t.Fatalf("button %q unavailable in error+recent+large state: %+v", label, btn)
			}
			if math.Abs(float64(btn.Bounds.W-wantW)) > 0.5 {
				t.Fatalf("button %q width = %.2f, want %.2f", label, btn.Bounds.W, wantW)
			}
			btnBounds[i] = btn.Bounds
			if i > 0 && math.Abs(float64(btn.Bounds.X-btnBounds[0].X)) > 0.5 {
				t.Fatalf("button %q X %.2f != openBounds X %.2f", label, btn.Bounds.X, btnBounds[0].X)
			}
		}

		if center := btnBounds[0].X + btnBounds[0].W/2; math.Abs(float64(center-980.0/2)) > 0.5 {
			t.Fatalf("Welcome actions centered at %.2f, want window center %.2f", center, 980.0/2)
		}
		settings, ok := tt.AccessByRole(ui.RoleButton, a.t("settings"))
		if !ok || settings.Offscreen || settings.Disabled {
			t.Fatalf("Welcome missing interactive Settings button: %+v", settings)
		}

		guidance, ok := tt.AccessByRole(ui.RoleText, a.loadErr)
		if !ok || guidance.Offscreen {
			t.Fatalf("error guidance missing or offscreen: %+v", guidance)
		}
		gap := btnBounds[0].Y - (guidance.Bounds.Y + guidance.Bounds.H)
		if gap < 0 || gap > 20 {
			t.Fatalf("error-to-button gap = %.2f, want in [0, 20] (guidance=%+v, open=%+v)", gap, guidance.Bounds, btnBounds[0])
		}

		recent, ok := tt.AccessByRole(ui.RoleButton, "recent-alpha")
		if !ok || recent.Offscreen {
			t.Fatalf("recent repository button missing or offscreen in error+recent+large state: %+v", recent)
		}
		lastBtnBottom := btnBounds[2].Y + btnBounds[2].H
		if recent.Bounds.Y < lastBtnBottom {
			t.Fatalf("recent repository button overlaps action buttons (lastBtnBottom=%.2f, recent=%+v)", lastBtnBottom, recent.Bounds)
		}
	})
}

func TestWelcomeActionButtonsRenderIcons(t *testing.T) {
	prevScale := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(prevScale) })
	setUIFontScale(fontScaleStandard)

	t.Run("helper_intrinsic_width_includes_icon_and_gap", func(t *testing.T) {
		entries := []struct {
			icon  *ui.SVG
			label string
			role  buttonRole
		}{
			{icon: lucide.FolderOpen(), label: "Open Repository", role: buttonPrimary},
			{icon: lucide.FolderPlus(), label: "New Repository", role: buttonNormal},
			{icon: lucide.Download(), label: "Clone Repository", role: buttonNormal},
		}
		for _, e := range entries {
			withIcon := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				ui.Column(c).Fill().AlignItems(ui.Start).Children(func() {
					workbenchIconLabelButton(c, e.icon, e.label, e.role, false)
				})
			}, 400, 100)
			textOnly := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				ui.Column(c).Fill().AlignItems(ui.Start).Children(func() {
					workbenchButton(c, e.label, e.role, false)
				})
			}, 400, 100)

			ib, ok := withIcon.AccessByRole(ui.RoleButton, e.label)
			if !ok {
				t.Fatalf("icon-label button %q not found", e.label)
			}
			tb, ok := textOnly.AccessByRole(ui.RoleButton, e.label)
			if !ok {
				t.Fatalf("text-only button %q not found", e.label)
			}
			minDelta := scaledFont(14) + 6 - 0.5
			if ib.Bounds.W < tb.Bounds.W+minDelta {
				t.Fatalf("%q icon button width %.2f should exceed text-only %.2f by at least %.2f", e.label, ib.Bounds.W, tb.Bounds.W, minDelta)
			}
		}
	})

	t.Run("empty_page_paints_icons_on_all_three_buttons", func(t *testing.T) {
		a := &app{locale: localeEnglish, page: "history"}
		welcome := ui.NewTester(func(c *ui.Context) {
			theme.Apply(c)
			a.emptyPage(c)
		}, 1280, 800)
		welcomeImg := welcome.Image()

		textOnlyBaseline := ui.NewTester(func(c *ui.Context) {
			theme.Apply(c)
			ui.Column(c).Fill().Center().Gap(8).Children(func() {
				ui.Column(c).Width(scaledFont(220)).Gap(8).Children(func() {
					workbenchButton(c, "Open Repository", buttonPrimary, false).FillWidth()
					workbenchButton(c, "New Repository", buttonNormal, false).FillWidth()
					workbenchButton(c, "Clone Repository", buttonNormal, false).FillWidth()
				})
			})
		}, 1280, 800)
		baseImg := textOnlyBaseline.Image()

		for _, label := range []string{"Open Repository", "New Repository", "Clone Repository"} {
			wb, ok := welcome.AccessByRole(ui.RoleButton, label)
			if !ok {
				t.Fatalf("Welcome button %q missing", label)
			}
			bb, ok := textOnlyBaseline.AccessByRole(ui.RoleButton, label)
			if !ok {
				t.Fatalf("baseline button %q missing", label)
			}
			// Compare interior pixels of the rendered Welcome button against the
			// same-width text-only button; an icon shifts and adds glyphs so many
			// pixels differ.
			w, h := int(math.Min(float64(wb.Bounds.W), float64(bb.Bounds.W))), int(math.Min(float64(wb.Bounds.H), float64(bb.Bounds.H)))
			diffPixels := 0
			for dy := 4; dy < h-4; dy++ {
				for dx := 4; dx < w-4; dx++ {
					p1 := welcomeImg.RGBAAt(int(wb.Bounds.X)+dx, int(wb.Bounds.Y)+dy)
					p2 := baseImg.RGBAAt(int(bb.Bounds.X)+dx, int(bb.Bounds.Y)+dy)
					if p1 != p2 {
						diffPixels++
					}
				}
			}
			if diffPixels < 20 {
				t.Fatalf("Welcome button %q appears to lack a rendered icon (diff pixels vs text-only = %d)", label, diffPixels)
			}
		}
	})
}
