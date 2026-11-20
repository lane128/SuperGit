package lobedesk

import (
	"context"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func historyRefPaletteEntry(t *testing.T, a *app) appShortcut {
	t.Helper()
	a.commandPalette.query = ""
	for _, entry := range a.commandPaletteEntries() {
		if entry.ID == "history.chooseRef" {
			return entry
		}
	}
	t.Fatal("missing history reference range palette command")
	return appShortcut{}
}

func historyRefPaletteApp(t *testing.T) *app {
	a := createBranchApp(t)
	a.branches = []gitrepo.Branch{{Name: "main", SHA: a.status.Head}, {Name: "feature", SHA: a.status.Head}}
	a.historyRef = historyRef{Kind: historyRefLocal, Name: "main"}
	a.historyQuery = "original query"
	return a
}

func TestPaletteHistoryRefHandoffCancel(t *testing.T) {
	for _, opener := range []string{"message", "history"} {
		for _, mod := range []ui.Modifiers{ui.Cmd, ui.Cmd | ui.Shift} {
			for _, activate := range []string{"enter", "click"} {
				for _, dismiss := range []string{"escape", "backdrop"} {
					t.Run(opener+"/"+shortcutModName(mod)+"/"+activate+"/"+dismiss, func(t *testing.T) {
						a := historyRefPaletteApp(t)
						a.commitText = "title\n\nbody"
						entry := historyRefPaletteEntry(t, a)
						tt := paletteStashTester(a, opener, 900, 640)
						if opener == "history" {
							tt.Key(ui.Cmd, ui.KeyF)
						} else {
							label := a.t("commitMessage")
							field, ok := tt.AccessByRole(ui.RoleTextField, label)
							if !ok {
								t.Fatal("missing original input")
							}
							tt.ClickAt(field.Bounds.X+10, field.Bounds.Y+10)
						}
						tt.Key(ui.Cmd, ui.KeyA)
						tt.Frame()
						originalRef, originalQuery := a.historyRef, a.historyQuery
						tt.Key(mod, ui.KeyP)
						tt.Frame()
						tt.Type(entry.ID)
						tt.Frame()
						row, ok := tt.AccessByRole(ui.RoleOption, a.t(entry.Label))
						if !ok || row.Disabled {
							t.Fatal("missing available entry")
						}
						if activate == "enter" {
							tt.Key(0, ui.KeyEnter)
						} else {
							tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
						}
						tt.Frame()
						tt.Frame()
						if a.commandPalette.open || !a.historyRefOpen || a.historyRef != originalRef || a.historyQuery != originalQuery {
							t.Fatal("first activation applied a ref or missed handoff")
						}
						dialogs := 0
						for _, node := range tt.AccessAll() {
							if node.Role == ui.RoleDialog {
								dialogs++
							}
						}
						if dialogs != 1 {
							t.Fatalf("expected one dialog, got %d", dialogs)
						}
						tt.Type("feature")
						tt.Frame()
						if a.historyRefQuery != "feature" {
							t.Fatal("typing missed reference search")
						}
						if dismiss == "escape" {
							tt.Key(0, ui.KeyEscape)
						} else {
							tt.ClickAt(2, 2)
						}
						tt.Frame()
						tt.Frame()
						if a.historyRefOpen || a.historyRef != originalRef || a.historyQuery != originalQuery || a.commitText != "title\n\nbody" || !a.amend {
							t.Fatal("cancel changed range/search/draft")
						}
						tt.Type("X")
						tt.Frame()
						switch opener {
						case "message":
							if a.commitText != "X" {
								t.Fatal("lost commit message selection")
							}
						case "history":
							if a.historyQuery != "X" || a.commitText != "title\n\nbody" {
								t.Fatal("lost history selection")
							}
						}
					})
				}
			}
		}
	}
}

func TestPaletteHistoryRefGuards(t *testing.T) {
	for _, blocked := range []string{"repo", "file", "busy", "loading", "reload", "dialog", "ref"} {
		t.Run(blocked, func(t *testing.T) {
			a := historyRefPaletteApp(t)
			entry := historyRefPaletteEntry(t, a)
			switch blocked {
			case "repo":
				a.repo = nil
			case "file":
				a.fileFilter = "base.txt"
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "reload":
				a.historyReloading = true
			case "dialog":
				a.confirm.open = true
			case "ref":
				a.historyRefOpen = true
			}
			if a.commandPaletteUnavailable(entry.ID) == "" {
				t.Fatal("blocked entry has no reason")
			}
			wasOpen := a.historyRefOpen
			entry.Run(a)
			if a.historyRefOpen != wasOpen {
				t.Fatal("blocked entry opened")
			}
		})
	}
	for _, change := range []string{"root", "epoch", "file"} {
		t.Run("stale/"+change, func(t *testing.T) {
			a := historyRefPaletteApp(t)
			entry := historyRefPaletteEntry(t, a)
			a.commitText = "title\n\nbody"
			tt := paletteStashTester(a, "summary", 900, 640)
			field, ok := tt.AccessByRole(ui.RoleTextField, a.t("commitMessage"))
			if !ok {
				t.Fatal("missing original input")
			}
			tt.ClickAt(field.Bounds.X+10, field.Bounds.Y+10)
			tt.Key(ui.Cmd, ui.KeyA)
			tt.Key(ui.Cmd, ui.KeyP)
			tt.Frame()
			tt.Type(entry.ID)
			tt.Frame()
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			tt.Frame()
			before := a.historyRef
			switch change {
			case "root":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			case "epoch":
				a.loadSeq++
			case "file":
				a.fileFilter = "base.txt"
			}
			tt.Frame()
			tt.Frame()
			if a.historyRefOpen || a.historyRef != before || a.commitText != "title\n\nbody" || a.historyQuery != "original query" {
				t.Fatal("stale picker survived, applied or changed background state")
			}
			tt.Type("X")
			tt.Frame()
			if a.commitText != "X" || a.historyRefQuery != "" {
				t.Fatal("stale picker kept focus or lost the surviving original input selection")
			}
		})
	}
}

func TestPaletteHistoryRefOwned(t *testing.T) {
	for _, accept := range []string{"enter", "click"} {
		t.Run(accept, func(t *testing.T) {
			a, _, root := worktreeApp(t)
			ctx, cancel := context.WithCancel(context.Background())
			a.ctx = ctx
			callbacks := make(chan func(), 64)
			a.runOnMain = func(fn func()) { callbacks <- fn }
			t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
			first := pullFixtureGit(t, root, "rev-parse", "HEAD")
			pullFixtureGit(t, root, "branch", "feature", first)
			pullFixtureGit(t, root, "commit", "--allow-empty", "-m", "second")
			write := func(path, value string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(root, path), []byte(value), 0600); err != nil {
					t.Fatal(err)
				}
			}
			read := func(path string) string {
				t.Helper()
				b, err := os.ReadFile(filepath.Join(root, path))
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}
			write("base.txt", "saved stash\n")
			pullFixtureGit(t, root, "stash", "push", "-m", "preserved stash")
			write("base.txt", "staged\n")
			pullFixtureGit(t, root, "add", "base.txt")
			write("base.txt", "work\n")
			write("untracked.txt", "untracked\n")
			var err error
			a.status, err = a.repo.Status(ctx)
			if err != nil {
				t.Fatal(err)
			}
			a.branches = []gitrepo.Branch{{Name: "main", SHA: a.status.Head}, {Name: "feature", SHA: first}}
			a.commitText = "title\n\nbody"
			a.amend = true
			a.historyQuery = "old query"
			snapshot := func() string {
				return strings.Join([]string{
					pullFixtureGit(t, root, "status", "--porcelain=v1", "--untracked-files=all"),
					pullFixtureGit(t, root, "for-each-ref", "--format=%(refname) %(objectname)"), read(".git/HEAD"), read(".git/index"), read("base.txt"), read("untracked.txt"),
				}, "\x00")
			}
			before := snapshot()
			entry := historyRefPaletteEntry(t, a)
			tt := paletteStashTester(a, "summary", 900, 640)
			open := func() {
				tt.Key(ui.Cmd, ui.KeyP)
				tt.Frame()
				tt.Type(entry.ID)
				tt.Frame()
				tt.Key(0, ui.KeyEnter)
				tt.Frame()
				tt.Frame()
			}
			open()
			tt.Type("feature")
			tt.Frame()
			tt.Key(0, ui.KeyEscape)
			tt.Frame()
			tt.Frame()
			if snapshot() != before || a.historyQuery != "old query" {
				t.Fatal("cancel changed owned repository/search")
			}
			open()
			tt.Type("feature")
			tt.Frame()
			if accept == "enter" {
				tt.Key(0, ui.KeyEnter)
			} else {
				row, ok := tt.AccessByRole(ui.RoleOption, "feature")
				if !ok {
					t.Fatal("missing feature candidate")
				}
				tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
			}
			tt.Frame()
			if a.historyRefOpen || a.historyRef != (historyRef{Kind: historyRefLocal, Name: "feature"}) || a.historyQuery != "" {
				t.Fatal("confirmation did not apply existing range semantics")
			}
			for a.historyReloading || a.historyLoadingMore {
				runContinuation(t, callbacks)
			}
			if a.historyErr != "" || len(a.commits) != 1 || a.commits[0].SHA != first {
				t.Fatal("did not load selected reference history")
			}
			if snapshot() != before || a.commitText != "title\n\nbody" || !a.amend {
				t.Fatal("read-only selection changed Git state/draft")
			}
		})
	}
}

func TestPaletteHistoryRefNarrowAndRetry(t *testing.T) {
	for _, loc := range []locale{localeEnglish, localeChinese} {
		for _, mode := range []string{"light", "dark"} {
			for _, failed := range []bool{false, true} {
				t.Run(fmt.Sprint(loc, "/", mode, "/", failed), func(t *testing.T) {
					a := historyRefPaletteApp(t)
					a.locale = loc
					a.themeMode = mode
					if failed {
						a.branchesErr = "fatal: cannot read reference list"
					}
					entry := historyRefPaletteEntry(t, a)
					tt := paletteStashTester(a, "summary", 360, 360)
					capture := func(part string) {
						if dir := os.Getenv("SUPERGIT_HISTORY_REF_CAPTURE_DIR"); dir != "" {
							if err := os.MkdirAll(dir, 0700); err != nil {
								t.Fatal(err)
							}
							f, err := os.Create(filepath.Join(dir, fmt.Sprintf("ref-%s-360x360-%s-%s-%t.png", part, loc, mode, failed)))
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
					}
					tt.Key(ui.Cmd, ui.KeyP)
					tt.Frame()
					tt.Type(entry.ID)
					tt.Frame()
					tt.Advance(150 * time.Millisecond)
					capture("entry")
					tt.Key(0, ui.KeyEnter)
					tt.Frame()
					tt.Frame()
					tt.Advance(150 * time.Millisecond)
					for _, node := range tt.AccessAll() {
						if node.Role == ui.RoleTextField && node.Name == a.t("searchRefs") || node.Role == ui.RoleDialog || node.Role == ui.RoleButton && node.Name == a.t("retry") {
							if node.Bounds.X < 16 || node.Bounds.X+node.Bounds.W > 344 || node.Bounds.Y < 16 || node.Bounds.Y+node.Bounds.H > 344 {
								t.Fatalf("outside narrow gutters: %+v", node)
							}
						}
					}
					capture("picker")
					if failed {
						tt.Key(0, ui.KeyTab)
						tt.Frame()
						tt.Frame()
						if !tt.Focused(a.t("retry")) {
							t.Fatal("Retry unreachable after handoff")
						}
					}
					tt.Key(0, ui.KeyEscape)
					tt.Frame()
					tt.Frame()
				})
			}
		}
	}
}

func TestPaletteHistoryRefDeletedCandidate(t *testing.T) {
	a := historyRefPaletteApp(t)
	entry := historyRefPaletteEntry(t, a)
	tt := paletteStashTester(a, "summary", 900, 640)
	entry.Run(a)
	tt.Frame()
	tt.Frame()
	tt.Type("feature")
	tt.Frame()
	original := a.historyRef
	a.branches = a.branches[:1]
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if !a.historyRefOpen || a.historyRef != original || a.historyReloading {
		t.Fatal("deleted filtered reference was applied")
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("main")
	tt.Frame()
	row, ok := tt.AccessByRole(ui.RoleOption, "main")
	if !ok || row.Disabled {
		t.Fatal("existing ref did not remain selectable")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	tt.Frame()
}

func TestPaletteHistoryRefModalOwnership(t *testing.T) {
	a := historyRefPaletteApp(t)
	entry := historyRefPaletteEntry(t, a)
	tt := paletteStashTester(a, "summary", 900, 640)
	entry.Run(a)
	tt.Frame()
	tt.Frame()
	if !a.hasPendingDialog(toastMain) {
		t.Fatal("reference modal is absent from shared dialog ownership")
	}
	calls := 0
	a.dispatchWindowShortcut(func(*app) { calls++ })
	if calls != 0 {
		t.Fatal("retained background callback crossed reference modal")
	}
	page := a.page
	tt.Key(ui.Cmd, ui.Key2)
	tt.Frame()
	if a.page != page || !a.historyRefOpen {
		t.Fatal("page shortcut crossed reference modal")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	tt.Frame()
	if a.hasPendingDialog(toastMain) {
		t.Fatal("reference modal ownership survived cancellation")
	}
}

func TestPaletteHistoryRefEmptySnapshot(t *testing.T) {
	a := historyRefPaletteApp(t)
	a.branches = nil
	a.status.Head = ""
	entry := historyRefPaletteEntry(t, a)
	tt := paletteStashTester(a, "summary", 900, 640)
	entry.Run(a)
	tt.Frame()
	tt.Frame()
	row, ok := tt.AccessByRole(ui.RoleOption, a.t("allRefs"))
	if !ok || row.Disabled {
		t.Fatal("empty snapshot has no all-refs choice")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	tt.Frame()
}

func TestPaletteHistoryRefNoMatch(t *testing.T) {
	a := historyRefPaletteApp(t)
	entry := historyRefPaletteEntry(t, a)
	tt := paletteStashTester(a, "summary", 900, 640)
	entry.Run(a)
	tt.Frame()
	tt.Frame()
	tt.Type(strings.Repeat("z", 32))
	tt.Frame()
	before := a.historyRef
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if !a.historyRefOpen || a.historyRef != before || a.historyRefCursor != -1 {
		t.Fatal("empty matches applied a ref")
	}
}
