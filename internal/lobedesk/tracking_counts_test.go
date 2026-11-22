package lobedesk

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// The sync menu keeps both pending rows in place; a zero side is a disabled
// row with a count of 0 rather than a missing or jumping row.
func TestSyncMenuPendingRowsKeepZeroSide(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		a := &app{locale: language, repo: &gitrepo.Repository{Root: t.TempDir()},
			status:  gitrepo.Status{Branch: "main", Upstream: "origin/main", Ahead: 2},
			remotes: []gitrepo.Remote{{Name: "origin"}}}
		a.syncOpen = true
		tt := ui.NewTester(func(c *ui.Context) {
			a.applyTheme(c)
			ui.Row(c).Justify(ui.End).FillWidth().Children(func() { a.syncMenu(c) })
		}, 700, 420)
		tt.Frame()
		push, okPush := tt.Find(a.t("syncToPush"))
		pull, okPull := tt.Find(a.t("syncToPull"))
		if !okPush || !okPull || push.Y >= pull.Y {
			t.Fatalf("%s: pending rows missing or out of order: %v", language, tt.Texts())
		}
		ahead, okAhead := tt.AccessByRole(ui.RoleMenuItem, a.tf("branchAhead", 2))
		behind, okBehind := tt.AccessByRole(ui.RoleMenuItem, a.tf("branchBehind", 0))
		if !okAhead || !okBehind || ahead.Disabled || !behind.Disabled {
			t.Fatalf("%s: zero side must stay as a disabled row", language)
		}
		if !tt.HasText("0") || !tt.HasText("2") {
			t.Fatalf("%s: count pills missing: %v", language, tt.Texts())
		}
	}
}

func TestBranchDetailsTrackingCounts(t *testing.T) {
	for _, tc := range []struct {
		ahead, behind int
		synced        bool
	}{{0, 0, true}, {2, 3, false}} {
		a := &app{locale: localeEnglish}
		branch := gitrepo.Branch{Name: "main", Upstream: "origin/main", Ahead: tc.ahead, Behind: tc.behind}
		tt := ui.NewTester(func(c *ui.Context) {
			a.applyTheme(c)
			ui.Column(c).Width(260).Children(func() {
				a.localBranchDetails(c, branchDetailSelection{branch: branch})
			})
		}, 300, 300)
		if tt.HasText("↑") || tt.HasText("origin/main · ↑2 ↓3") {
			t.Fatalf("legacy arrow text still drawn: %v", tt.Texts())
		}
		if got := tt.HasText(a.t("upToDate")); got != tc.synced {
			t.Fatalf("up-to-date label = %v, want %v", got, tc.synced)
		}
		if !tc.synced {
			up, ok1 := tt.Find("2")
			down, ok2 := tt.Find("3")
			if !ok1 || !ok2 || up.X >= down.X {
				t.Fatalf("counts missing or out of order: %v", tt.Texts())
			}
		}
	}
}

func TestTrackingCountsVisualModes(t *testing.T) {
	outDir := os.Getenv("SUPERGIT_TRACKING_SNAPSHOT_DIR")
	for _, mode := range []string{"light", "dark"} {
		t.Run(mode, func(t *testing.T) {
			a := &app{
				locale:    localeChinese,
				themeMode: mode,
				repo:      &gitrepo.Repository{Root: t.TempDir()},
				status:    gitrepo.Status{Branch: "main", Upstream: "origin/main", Ahead: 2, Behind: 3},
				remotes:   []gitrepo.Remote{{Name: "origin"}},
				branches: []gitrepo.Branch{
					{Name: "main", SHA: "bd69c18a", Current: true, Upstream: "origin/main", Ahead: 2, Behind: 3},
					{Name: "feat/polish", SHA: "2aca561b", Upstream: "origin/feat/polish", Ahead: 1, Behind: 0},
					{Name: "release/v1", SHA: "7f6663ac", Upstream: "origin/release/v1", Ahead: 0, Behind: 0},
				},
				branchSel: "local:feat/polish",
			}
			a.syncOpen = true
			tt := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				tk := theme.TokensOf(c)
				ui.Column(c).FillWidth().FillHeight().Background(tk.BgLayout).Padding(16).Gap(16).Children(func() {
					ui.Row(c).FillWidth().AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
						a.toolbarCommandControl(c, true)
						a.syncMenu(c)
					})
					ui.Row(c).FillWidth().Gap(16).Children(func() {
						ui.Column(c).Width(360).Background(tk.BgContainer).Padding(10).Radius(8).Gap(4).Children(func() {
							for i, branch := range a.branches {
								a.branchRow(c, branch, i+1, len(a.branches))
							}
						})
						ui.Column(c).Width(260).Background(tk.BgContainer).Padding(12).Radius(8).Gap(12).Children(func() {
							a.localBranchDetails(c, branchDetailSelection{branch: a.branches[0]})
							a.localBranchDetails(c, branchDetailSelection{branch: a.branches[2]})
						})
					})
				})
			}, 980, 420)
			tt.Frame()
			if !tt.HasText(a.t("syncToPush")) || !tt.HasText(a.t("syncToPull")) || !tt.HasText(a.t("upToDate")) {
				t.Fatalf("%s: expected tracking rows and up-to-date label, got %v", mode, tt.Texts())
			}
			if outDir != "" {
				file, err := os.Create(filepath.Join(outDir, "tracking-"+mode+".png"))
				if err != nil {
					t.Fatal(err)
				}
				encodeErr := png.Encode(file, tt.Image())
				if err := file.Close(); err != nil || encodeErr != nil {
					t.Fatalf("write snapshot: encode=%v close=%v", encodeErr, err)
				}
			}
		})
	}
}
