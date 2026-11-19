package lobedesk

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestHistoricalDiffBaseline(t *testing.T) {
	before := uiFontScaleValue()
	setUIFontScale(fontScaleLarge)
	t.Cleanup(func() { setUIFontScale(before) })
	sha, parent, second := strings.Repeat("a", 40), strings.Repeat("b", 40), strings.Repeat("c", 40)
	for _, language := range []locale{localeEnglish, localeChinese} {
		for _, mode := range []string{"light", "dark"} {
			for _, kind := range []string{"ordinary", "merge", "root", "unknown", "hidden"} {
				t.Run(string(language)+"/"+mode+"/"+kind, func(t *testing.T) {
					a := &app{locale: language, themeMode: mode, selectedSHA: sha, commitText: "draft\n\nbody", amend: true}
					commit := gitrepo.Commit{SHA: sha, Parents: []string{parent}}
					if kind == "merge" {
						commit.Parents = append(commit.Parents, second)
					}
					if kind == "root" {
						commit.Parents = nil
					}
					if kind != "unknown" {
						a.commits = []gitrepo.Commit{commit}
					}
					if kind == "hidden" {
						a.detailsDiffHidden = true
					}
					expected := fmt.Sprintf("First parent %s → Commit %s", shortSHA(parent), shortSHA(sha))
					if language == localeChinese {
						expected = fmt.Sprintf("第一父提交 %s → 当前提交 %s", shortSHA(parent), shortSHA(sha))
					}
					if kind == "root" {
						expected = "Root commit; no parent"
						if language == localeChinese {
							expected = "根提交，无父提交"
						}
					}
					tt := ui.NewTester(func(c *ui.Context) {
						a.applyTheme(c)
						ui.Column(c).Fill().Background(theme.TokensOf(c).BgContainer).Children(func() { a.historyDiffFile(c, theme.TokensOf(c)) })
					}, 300, 240)
					if kind == "unknown" || kind == "hidden" {
						if tt.HasText(expected) || tt.HasText(a.t("commitDiffRootBaseline")) {
							t.Fatal("baseline shown for unavailable context")
						}
						if kind == "hidden" && !tt.HasText(a.t("diffHiddenNote")) {
							t.Fatal("hidden diff lost its explanation")
						}
						return
					}
					bounds, ok := tt.Find(expected)
					if !ok || bounds.X < 0 || bounds.Y < 0 || bounds.X+bounds.W > 300 || bounds.Y+bounds.H > 240 {
						t.Fatalf("missing or clipped baseline: %+v", bounds)
					}
					if !tt.HasText(a.t("diffEmpty")) {
						t.Fatal("baseline displaced empty diff feedback")
					}
					if a.selectedSHA != sha || a.commitText != "draft\n\nbody" || !a.amend {
						t.Fatal("baseline changed selected target or draft")
					}
					if dir := os.Getenv("SUPERGIT_BASELINE_CAPTURE_DIR"); dir != "" {
						f, err := os.Create(filepath.Join(dir, fmt.Sprintf("baseline-%s-%s-%s.png", kind, language, mode)))
						if err != nil {
							t.Fatal(err)
						}
						err = png.Encode(f, tt.Image())
						closeErr := f.Close()
						if err != nil || closeErr != nil {
							t.Fatalf("capture: %v %v", err, closeErr)
						}
					}
				})
			}
		}
	}
}
