package lobedesk

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func paletteTester(a *app, editors bool, width, height int) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.registerShortcuts(c)
		ui.Column(c).Fill().Children(func() {
			if editors {
				a.commitMessageFields(c)
			}
			a.commandPaletteDialog(c)
		})
	}, width, height)
}

func TestCommandPaletteRenderedBounds(t *testing.T) {
	for _, size := range [][2]int{{360, 360}, {900, 640}} {
		for _, language := range []locale{localeEnglish, localeChinese} {
			for _, mode := range []string{"light", "dark"} {
				name := fmt.Sprintf("%dx%d-%s-%s", size[0], size[1], language, mode)
				t.Run(name, func(t *testing.T) {
					a := &app{locale: language, themeMode: mode}
					a.openCommandPalette()
					tt := ui.NewTester(func(c *ui.Context) {
						a.applyTheme(c)
						ui.Column(c).Fill().Background(theme.TokensOf(c).BgLayout)
						a.commandPaletteDialog(c)
					}, size[0], size[1])
					for _, node := range tt.AccessAll() {
						if node.Role != ui.RoleTextField && node.Role != ui.RoleOption && node.Role != ui.RoleDialog {
							continue
						}
						if node.Bounds.X < 16 || node.Bounds.X+node.Bounds.W > float32(size[0]-16) || (node.Role != ui.RoleOption && (node.Bounds.Y < 16 || node.Bounds.Y+node.Bounds.H > float32(size[1]-16))) {
							t.Errorf("palette control outside window gutters: %+v", node)
						}
					}
					if dir := os.Getenv("SUPERGIT_PALETTE_CAPTURE_DIR"); dir != "" {
						f, err := os.Create(filepath.Join(dir, name+".png"))
						if err != nil {
							t.Fatal(err)
						}
						err = png.Encode(f, tt.Image())
						closeErr := f.Close()
						if err != nil {
							t.Fatal(err)
						}
						if closeErr != nil {
							t.Fatal(closeErr)
						}
					}
				})
			}
		}
	}
}

func TestCommandPaletteAliasesCancelToActualBody(t *testing.T) {
	for _, mod := range []ui.Modifiers{ui.Cmd, ui.Cmd | ui.Shift} {
		t.Run(shortcutModName(mod), func(t *testing.T) {
			a, _ := overviewFixture(t)
			a.locale, a.page = localeEnglish, "changes"
			a.commitText, a.amend = "summary\n\nbody", true
			tt := paletteTester(a, true, 600, 500)
			body, ok := tt.AccessByRole(ui.RoleTextField, a.t("commitDescription"))
			if !ok {
				t.Fatal("actual Body missing")
			}
			tt.ClickAt(body.Bounds.X+body.Bounds.W/2, body.Bounds.Y+body.Bounds.H/2)
			tt.Key(mod, ui.KeyP)
			tt.Frame()
			if !a.commandPalette.open || !tt.Focused(a.t("commandPaletteSearch")) {
				t.Fatal("alias did not focus palette search")
			}
			tt.Type("page.")
			tt.Frame()
			tt.Key(ui.Cmd, ui.Key4)
			tt.Key(ui.Cmd, ui.KeyEnter)
			tt.Key(ui.Cmd|ui.Shift, ui.KeyA)
			tt.Frame()
			if a.page != "changes" || a.busy || a.commitText != "summary\n\nbody" || !a.amend {
				t.Fatal("palette leaked keys or changed draft")
			}
			tt.Key(0, ui.KeyEscape)
			tt.Frame()
			tt.Frame()
			if a.commandPalette.open || !tt.Focused(a.t("commitDescription")) {
				t.Fatalf("cancel did not restore actual Body: open=%v, access=%+v", a.commandPalette.open, tt.AccessAll())
			}
			tt.Key(0, ui.KeyEnd)
			tt.Type("X")
			if a.commitFields.Body != "bodyX" {
				t.Fatalf("typing missed restored Body: %#v", a.commitFields)
			}
			tt.Key(mod, ui.KeyP)
			tt.Frame()
			if a.commandPalette.query != "" {
				t.Fatal("reopen retained old query")
			}
		})
	}
}

func shortcutModName(mod ui.Modifiers) string {
	if mod&ui.Shift != 0 {
		return "shift"
	}
	return "plain"
}

func TestCommandPaletteCancelFromSummaryAndHistorySearch(t *testing.T) {
	for _, source := range []string{"summary", "search"} {
		for _, cancel := range []string{"escape", "backdrop"} {
			t.Run(source+"/"+cancel, func(t *testing.T) {
				a, _ := overviewFixture(t)
				a.locale, a.page = localeEnglish, "changes"
				a.commitText = "draft\n\nbody"
				if source == "search" {
					a.page = "history"
					a.historyQuery = "needle"
				}
				tt := ui.NewTester(func(c *ui.Context) {
					theme.Apply(c)
					a.registerShortcuts(c)
					ui.Column(c).Fill().Children(func() {
						if source == "search" {
							a.historyPage(c)
						} else {
							a.commitMessageFields(c)
						}
						a.commandPaletteDialog(c)
					})
				}, 900, 640)
				label := a.t("commitSummary")
				if source == "search" {
					tt.Key(ui.Cmd, ui.KeyF)
				} else {
					node, ok := tt.AccessByRole(ui.RoleTextField, label)
					if !ok {
						t.Fatalf("missing actual %s", label)
					}
					tt.ClickAt(node.Bounds.X+node.Bounds.W/2, node.Bounds.Y+node.Bounds.H/2)
				}
				tt.Key(ui.Cmd, ui.KeyP)
				tt.Frame()
				tt.Type("page.")
				tt.Frame()
				if cancel == "escape" {
					tt.Key(0, ui.KeyEscape)
				} else {
					tt.ClickAt(2, 2)
				}
				tt.Frame()
				tt.Frame()
				if a.commandPalette.open || source == "summary" && !tt.Focused(label) {
					t.Fatal("cancel did not restore original actual input")
				}
				tt.Key(0, ui.KeyEnd)
				tt.Type("X")
				if source == "search" {
					if a.historyQuery != "needleX" || a.commitText != "draft\n\nbody" {
						t.Fatal("search typing or draft changed")
					}
				} else if a.commitFields.Summary != "draftX" || a.commitFields.Body != "body" {
					t.Fatal("summary typing missed original field")
				}
			})
		}
	}
}

func TestCommandPaletteDelayedEscapeClosesOnce(t *testing.T) {
	a, _ := overviewFixture(t)
	a.locale, a.page = localeEnglish, "changes"
	tt := paletteTester(a, true, 600, 500)
	tt.Key(ui.Cmd, ui.KeyP)
	tt.Frame()
	tt.Type("page.")
	tt.Frame()
	tt.SetFocused(true)
	tt.Key(0, ui.KeyTab)
	tt.Advance(2 * time.Second)
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	tt.Frame()
	if a.commandPalette.open {
		t.Fatal("one Escape after focused-row delay did not close palette")
	}
}

func TestCommandPaletteSearchAndExactAction(t *testing.T) {
	a, _ := overviewFixture(t)
	a.locale, a.page = localeEnglish, "changes"
	a.commitText = "draft\n\nbody"
	tt := paletteTester(a, true, 600, 500)
	tt.Key(ui.Cmd, ui.KeyP)
	tt.Frame()
	tt.Type("PAGE.")
	tt.Frame()
	if got := len(a.commandPaletteEntries()); got != 4 {
		t.Fatalf("page filter got %d entries", got)
	}
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Frame()
	if a.commandPalette.open || a.page != "branches" || a.commitText != "draft\n\nbody" {
		t.Fatal("Enter did not execute exact highlighted page after closing")
	}
	// The normal dispatcher works again after closure.
	tt.Key(ui.Cmd, ui.Key4)
	tt.Frame()
	if a.page != "worktrees" {
		t.Fatal("background dispatch remained blocked after close")
	}
}

func TestCommandPaletteCatalogAndUnavailableRows(t *testing.T) {
	a := &app{locale: localeEnglish}
	a.openCommandPalette()
	tt := paletteTester(a, false, 360, 360)
	seen := map[string]bool{}
	for _, s := range a.commandPaletteEntries() {
		if seen[s.ID] || s.Field || s.Local || strings.HasPrefix(s.ID, "list.") || s.ID == "page.activate" {
			t.Fatalf("invalid catalog entry %+v", s)
		}
		seen[s.ID] = true
	}
	if len(seen) != 18 || !seen["repo.reopenClosedTab"] || !seen["repo.clone"] || !seen["tag.create"] || !seen["history.chooseRef"] || !seen["branch.create"] || !seen["stash.saveTracked"] || !seen["repo.fetch"] || !seen["repo.quickSwitch"] || seen["repo.quickSwitchAlias"] || !seen["branch.quickCheckout"] || seen["repo.reloadF5"] || seen["repo.pullF7"] || seen["repo.commit"] {
		t.Fatalf("wrong initial whitelist: %v", seen)
	}
	if !tt.HasText(a.t("commandPaletteNeedRepo")) {
		t.Fatal("unavailable actions lack visible reason")
	}
	tt.Type("page.")
	tt.Frame()
	if a.commandPalette.cursor != -1 {
		t.Fatal("disabled-only results have a selectable cursor")
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if !a.commandPalette.open || a.page != "" {
		t.Fatal("disabled Enter executed or closed")
	}
	a.commandPalette.query = "definitely-no-such-command"
	tt.Frame()
	if !tt.HasText(a.t("commandPaletteNoResults")) {
		t.Fatal("empty results lack feedback")
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if !a.commandPalette.open {
		t.Fatal("empty Enter closed palette")
	}
}

func TestCommandPaletteStaleCallbacksAndGuards(t *testing.T) {
	for _, change := range []string{"busy", "loading", "history", "chooser", "ref", "dialog", "root", "epoch", "reopen"} {
		t.Run(change, func(t *testing.T) {
			a, _ := overviewFixture(t)
			a.page = "changes"
			a.openCommandPalette()
			generation := a.commandPalette.generation
			switch change {
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "history":
				a.historyReloading = true
			case "chooser":
				a.repositoryChooserOpen = true
			case "ref":
				a.historyRefOpen = true
			case "dialog":
				a.cloneForm.open = true
			case "root":
				a.repo = nil
			case "epoch":
				a.loadSeq++
			case "reopen":
				a.commandPalette.open = false
				a.openCommandPalette()
			}
			a.runPaletteCommand("page.branches", generation)
			if a.page != "changes" {
				t.Fatal("stale callback executed")
			}
			if change != "reopen" {
				a.commandPalette.open = false
				a.openCommandPalette()
				if change != "root" && change != "epoch" && a.commandPalette.open {
					t.Fatal("guard permitted opening")
				}
			}
		})
	}
}

func TestCommandPaletteCommitMessageActionOwnsFocus(t *testing.T) {
	a, _ := overviewFixture(t)
	a.locale, a.page = localeEnglish, "branches"
	a.commitText = "draft\n\nbody"
	tt := paletteTester(a, true, 600, 500)
	tt.Key(ui.Cmd, ui.KeyP)
	tt.Frame()
	tt.Type("workspace.focusCommitMessage")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Frame()
	if a.commandPalette.open || a.page != "history" || !a.worktreeSelected || !tt.Focused(a.t("commitSummary")) || a.commitText != "draft\n\nbody" {
		t.Fatal("target action did not own Summary focus")
	}
	tt.Key(0, ui.KeyTab)
	tt.Frame()
	tt.Frame()
	if !tt.Focused(a.t("commitDescription")) {
		t.Fatal("palette stole focus after target action")
	}
}

func TestCommandPaletteModalOwnershipAndRetainedWindowAction(t *testing.T) {
	a, _ := overviewFixture(t)
	a.page = "changes"
	a.openCommandPalette()
	if !a.hasPendingDialog(toastMain) || a.hasPendingDialog(toastSettings) {
		t.Fatal("palette owns wrong modal window")
	}
	a.dispatchWindowShortcut(func(a *app) { a.openPage("branches") })
	if a.page != "changes" {
		t.Fatal("retained background callback leaked through palette")
	}
	old := a.commandPalette.generation
	a.openCommandPalette()
	if old != a.commandPalette.generation {
		t.Fatal("reopening stacked/reset palette")
	}
}
