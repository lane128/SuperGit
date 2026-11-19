package lobedesk

import (
	"runtime"
	"testing"

	"github.com/egoist/mygo/ui"
)

func fileTabBinding(t *testing.T, id string) appShortcut {
	t.Helper()
	for _, s := range shortcutTable() {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("missing file tab shortcut %s", id)
	return appShortcut{}
}

func TestFileTabShortcutsCycleVersionedTabsFromActualEditor(t *testing.T) {
	pairs := [][2]string{{"detail.previousFileTab", "detail.nextFileTab"}}
	if runtime.GOOS == "darwin" {
		pairs = append(pairs, [2]string{"detail.previousFileTabBracket", "detail.nextFileTabBracket"}, [2]string{"detail.previousFileTabCommandPage", "detail.nextFileTabCommandPage"})
	}
	for _, page := range []string{"changes", "history"} {
		for _, pair := range pairs {
			t.Run(page+"/"+pair[0], func(t *testing.T) {
				previous, next := fileTabBinding(t, pair[0]), fileTabBinding(t, pair[1])
				a, _ := overviewFixture(t)
				a.locale = localeEnglish
				a.page = page
				draft := "title preserved\n\nbody preserved\n\n"
				a.commitText = draft
				a.amend = true
				first, second := detailFileKey{Path: "same.go"}, detailFileKey{Path: "same.go", Staged: true}
				tabs := a.detailTabs()
				tabs.Open(first)
				tabs.Open(second)
				tabs.Activate(detailTabKey{Summary: true})
				tt := pageTester(a, func(c *ui.Context) { a.registerShortcuts(c); a.contentCard(c) })
				body, ok := tt.AccessByRole(ui.RoleTextField, a.t("commitDescription"))
				if !ok {
					t.Fatal("missing actual body editor")
				}
				tt.ClickAt(body.Bounds.X+body.Bounds.W/2, body.Bounds.Y+body.Bounds.H/2)
				check := func(want detailTabKey) {
					t.Helper()
					tt.Frame()
					if a.detailTabs().Active != want {
						t.Fatalf("active=%+v want=%+v", a.detailTabs().Active, want)
					}
					label := a.detailSummaryLabel()
					if !want.Summary {
						label = want.File.Path + " · " + a.t("unstaged")
						if want.File.Staged {
							label = want.File.Path + " · " + a.t("staged")
						}
					}
					node, ok := tt.AccessByRole(ui.RoleTab, label)
					if !ok || !node.Focused {
						t.Fatalf("activated tab not focused: %q", label)
					}
					if a.commitText != draft || !a.amend {
						t.Fatal("tab shortcut edited draft/amend")
					}
					if !want.Summary && (a.selectedPath != want.File.Path || a.selectedStaged != want.File.Staged) {
						t.Fatal("tab switched wrong file version")
					}
				}
				for _, want := range []detailTabKey{{File: first}, {File: second}, {Summary: true}} {
					tt.Key(next.Mod, next.Key)
					check(want)
				}
				// Start the reverse direction from the actual Summary field too.
				summary, _ := tt.AccessByRole(ui.RoleTextField, a.t("commitSummary"))
				tt.ClickAt(summary.Bounds.X+summary.Bounds.W/2, summary.Bounds.Y+summary.Bounds.H/2)
				for _, want := range []detailTabKey{{File: second}, {File: first}, {Summary: true}} {
					tt.Key(previous.Mod, previous.Key)
					check(want)
				}
				tt.Key(0, ui.KeyTab)
				tt.Frame()
				tt.Frame()
				node, _ := tt.AccessByRole(ui.RoleTab, a.detailSummaryLabel())
				if node.Focused {
					t.Fatal("tab request stole focus after normal Tab")
				}
			})
		}
	}
}

func TestFileTabShortcutsFromActualHistorySearch(t *testing.T) {
	for _, s := range fileTabShortcutBindings() {
		t.Run(s.ID, func(t *testing.T) {
			a, _ := overviewFixture(t)
			a.page = "history"
			a.locale = localeEnglish
			file := detailFileKey{Path: "same.go"}
			a.detailTabs().Open(file)
			a.detailTabs().Activate(detailTabKey{Summary: true})
			tt := pageTester(a, func(c *ui.Context) { a.registerShortcuts(c); a.historyPage(c) })
			tt.Key(ui.Cmd, ui.KeyF)
			tt.Frame()
			tt.Key(s.Mod, s.Key)
			tt.Frame()
			if a.detailTabs().Active != (detailTabKey{File: file}) {
				t.Fatal("shortcut from history search did not activate the file tab")
			}
			if a.historyQuery != "" {
				t.Fatal("shortcut edited the search query")
			}
		})
	}
}

func TestFileTabShortcutsPreserveHistoricalContextAndScroll(t *testing.T) {
	a, _ := overviewFixture(t)
	a.page = "history"
	a.worktreeSelected = false
	a.selectedSHA = "historical commit"
	a.commitText = "draft\n\nbody\n\n"
	a.amend = true
	file := detailFileKey{Path: "same.go"}
	a.detailTabs().Open(file)
	a.detailTabs().Activate(detailTabKey{Summary: true})
	a.commitInfoScroll = ui.ScrollState{Y: 42}
	a.detailTabs().Scroll = map[detailTabKey]detailScrollPosition{{File: file}: {Y: 87}}
	before := a.currentDetailContext()
	// Keep content unmounted so its layout cannot clamp the saved scroll.
	tt := pageTester(a, func(c *ui.Context) { a.registerShortcuts(c); ui.Column(c).Fill() })
	next := fileTabBinding(t, "detail.nextFileTab")
	previous := fileTabBinding(t, "detail.previousFileTab")
	tt.Key(next.Mod, next.Key)
	if a.commitPath != file.Path || a.commitDiffScroll.Y != 87 {
		t.Fatal("historical file or scroll was not restored")
	}
	a.commitDiffScroll.Y = 123
	tt.Key(next.Mod, next.Key)
	if !a.detailTabs().Active.Summary || a.commitInfoScroll.Y != 42 {
		t.Fatal("single-file cycle did not restore summary scroll")
	}
	tt.Key(previous.Mod, previous.Key)
	if a.commitDiffScroll.Y != 123 || a.currentDetailContext() != before || a.worktreeSelected || a.commitText != "draft\n\nbody\n\n" || !a.amend {
		t.Fatal("cycling lost historical context, per-tab scroll or draft")
	}
}

func TestFileTabFocusDoesNotMoveToSummaryWhenHistoricalDiffHides(t *testing.T) {
	for _, normalize := range []bool{false, true} {
		t.Run(map[bool]string{false: "before normalization", true: "after normalization"}[normalize], func(t *testing.T) {
			a, _ := overviewFixture(t)
			a.page = "history"
			a.worktreeSelected = false
			a.selectedSHA = "history"
			a.detailTabs().Open(detailFileKey{Path: "same.go"})
			a.requestDetailTabFocus()
			a.detailsDiffHidden = true
			if normalize {
				a.normalizeDetailVisibility()
			}
			tt := pageTester(a, func(c *ui.Context) { a.registerShortcuts(c); ui.Column(c).Fill() })
			tt.Frame()
			if a.detailFocusPending {
				t.Fatal("hidden file focus survived or moved to summary")
			}
		})
	}
}

func TestFileTabFocusExpiresWhileTabsAreUnmounted(t *testing.T) {
	for _, reason := range []string{"busy", "loading", "chooser", "dialog", "epoch", "commit", "page"} {
		t.Run(reason, func(t *testing.T) {
			a, _ := overviewFixture(t)
			a.page = "history"
			a.detailTabs().Open(detailFileKey{Path: "same.go"})
			a.requestDetailTabFocus()
			switch reason {
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "chooser":
				a.repositoryChooserOpen = true
			case "dialog":
				a.confirm.open = true
			case "epoch":
				a.loadSeq++
			case "commit":
				a.worktreeSelected = false
				a.selectedSHA = "another commit"
			case "page":
				a.page = "stash"
			}
			tt := pageTester(a, func(c *ui.Context) { a.registerShortcuts(c); ui.Column(c).Fill() })
			tt.Frame()
			if a.detailFocusPending {
				t.Fatal("stale focus survived while tab strip was unmounted")
			}
			a.busy, a.loading, a.repositoryChooserOpen, a.confirm.open = false, false, false, false
			a.page = "history"
			a.worktreeSelected = true
			a.selectedSHA = ""
			tt.Frame()
			if a.detailFocusPending {
				t.Fatal("cleared focus request returned after unblocking")
			}
		})
	}
}

func TestFileTabTextKeyLeavesUnavailableAndEditingKeysAlone(t *testing.T) {
	a, _ := overviewFixture(t)
	a.page = "history"
	a.detailTabs().Open(detailFileKey{Path: "same.go"})
	for _, key := range []ui.TextKey{{Key: ui.KeyPageUp}, {Key: ui.KeyPageDown}, {Mods: ui.Shift, Key: ui.KeyPageDown}, {Mods: ui.Cmd, Key: ui.KeyA}} {
		if a.fileTabTextKey(key) {
			t.Fatalf("ordinary editing key was consumed: %+v", key)
		}
	}
	a.busy = true
	for _, s := range fileTabShortcutBindings() {
		if a.fileTabTextKey(ui.TextKey{Mods: s.Mod, Key: s.Key}) {
			t.Fatal("unavailable file-tab key was consumed")
		}
	}
}

func TestFileTabShortcutGuardsAndEmptyTabs(t *testing.T) {
	for _, id := range []string{"detail.previousFileTab", "detail.nextFileTab"} {
		for _, kind := range []string{"nil", "busy", "loading", "history refresh", "chooser", "reference popup", "new", "clone", "confirmation", "other page", "hidden details", "hidden historical diff", "no files"} {
			t.Run(id+"/"+kind, func(t *testing.T) {
				s := fileTabBinding(t, id)
				a, _ := overviewFixture(t)
				a.page = "history"
				a.detailTabs().Open(detailFileKey{Path: "same.go"})
				a.detailTabs().Activate(detailTabKey{Summary: true})
				switch kind {
				case "nil":
					a.repo = nil
				case "busy":
					a.busy = true
				case "loading":
					a.loading = true
				case "history refresh":
					a.historyReloading = true
				case "chooser":
					a.repositoryChooserOpen = true
				case "reference popup":
					a.historyRefOpen = true
				case "new":
					a.newRepoForm.open = true
				case "clone":
					a.cloneForm.open = true
				case "confirmation":
					a.confirm.open = true
				case "other page":
					a.page = "stash"
				case "hidden details":
					a.historyDetailsHidden = true
				case "hidden historical diff":
					a.worktreeSelected = false
					a.selectedSHA = "history"
					a.detailTabs().Open(detailFileKey{Path: "same.go"})
					a.detailTabs().Activate(detailTabKey{Summary: true})
					a.detailsDiffHidden = true
				case "no files":
					a.detailTabs().OpenFiles = nil
				}
				before := a.currentDetailContext()
				tt := ui.NewTester(func(c *ui.Context) { a.registerShortcuts(c); ui.Column(c).Fill() }, 400, 200)
				if s.Available == nil || s.Available(a) {
					t.Fatal("guarded shortcut available")
				}
				tt.Key(s.Mod, s.Key)
				s.Run(a)
				if a.currentDetailContext() != before || a.detailFocusPending {
					t.Fatal("guarded navigation changed context or queued focus")
				}
				if a.repo != nil && !a.detailTabs().Active.Summary {
					t.Fatal("guarded navigation changed active tab")
				}
			})
		}
	}
}
