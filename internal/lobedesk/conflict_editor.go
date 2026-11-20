package lobedesk

import (
	"context"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

// Each draft retains the exact snapshot it was based on. Refreshing the file
// list must not silently rebase typed text onto a different conflict.
type conflictEditorKey struct{ Root, Path string }
type conflictEditorState struct {
	Original        gitrepo.Conflict
	Text, Error     string
	Loading, Loaded bool
	Preview         bool
	Source          int
	Scroll          [3]ui.ScrollState
}

func (a *app) currentConflictEditor() *conflictEditorState {
	if a.repo == nil {
		return nil
	}
	return a.conflictEditors[conflictEditorKey{a.repo.Root, a.selectedPath}]
}

func (a *app) openConflictEditor() {
	if a.repo == nil || a.busy || a.loading {
		return
	}
	key := conflictEditorKey{a.repo.Root, a.selectedPath}
	if _, ok := a.conflictPreviewEntry(key.Path); !ok {
		return
	}
	if a.conflictEditors == nil {
		a.conflictEditors = make(map[conflictEditorKey]*conflictEditorState)
	}
	s := a.conflictEditors[key]
	if s != nil && (s.Loading || s.Loaded) {
		s.Preview = false
		return
	}
	s = &conflictEditorState{Loading: true}
	a.conflictEditors[key] = s
	a.readConflictEditor(key, s, false)
}

func (a *app) readConflictEditor(key conflictEditorKey, s *conflictEditorState, keepDraft bool) {
	s.Loading = true
	repo := a.repo
	go func() {
		original, err := repo.Conflict(a.ctx, key.Path)
		a.hopToMain(func() {
			// A stale read may populate its own draft, never another file or repository.
			if a.conflictEditors[key] != s {
				return
			}
			s.Loading = false
			if err != nil {
				s.Error = err.Error()
			} else {
				s.Original, s.Loaded, s.Error = original, true, ""
				if !keepDraft {
					s.Text = original.Working
				}
			}
			a.invalidate()
		})
	}()
}

func (a *app) saveConflictEditor(s *conflictEditorState) {
	if s == nil || !s.Loaded || s.Loading || a.repo == nil || s.Original.Root != a.repo.Root || s != a.currentConflictEditor() {
		return
	}
	original, text := s.Original, s.Text
	a.performDoneAs("", a.t("resolveFailed"), a.t("resolveSuccess"), func(err error) {
		if err != nil {
			s.Error = err.Error()
			return
		}
		// Keep the draft until the status read confirms resolution. Reopening a
		// later conflict must obtain a new snapshot, even for the same path.
		delete(a.conflictEditors, conflictEditorKey{original.Root, original.Path})
	}, func(ctx context.Context, repo *gitrepo.Repository) error {
		return repo.ResolveConflict(ctx, original, text)
	})
}

func (a *app) conflictEditorBody(c *ui.Context, s *conflictEditorState) {
	c = c.Key(s)
	root := ui.Column(c.Key("conflict-body")).Fill().Gap(8).Padding(10, 12)
	root.Children(func() {
		ui.Row(c).FillWidth().Gap(6).Shrink(0).Children(func() {
			if workbenchButton(c, a.t("conflictEditorPatch"), buttonNormal, false).Clicked() {
				s.Preview = true
			}
			if s.Loaded && workbenchButton(c, a.t("conflictEditorReload"), buttonNormal, a.busy || a.loading || s.Loading).Clicked() {
				a.readConflictEditor(conflictEditorKey{a.repo.Root, a.selectedPath}, s, true)
			}
		})
		if s.Loading {
			display.Text(c, a.t("diffLoading"), display.TextProps{})
			return
		}
		if s.Error != "" {
			display.Text(c, s.Error, display.TextProps{Type: display.TextDanger, Rows: 3})
			if s.Loaded {
				display.Text(c, a.t("conflictEditorRecoveryHint"), display.TextProps{Type: display.TextSecondary, Rows: 3, FontSize: scaledFont(11)})
			}
		}
		if !s.Loaded {
			display.Text(c, a.t("conflictEditorFallback"), display.TextProps{Type: display.TextSecondary, Rows: 3})
			if workbenchButton(c, a.t("retry"), buttonNormal, a.busy || a.loading).Clicked() {
				a.openConflictEditor()
			}
			return
		}
		labels := []string{a.conflictSideText("conflictOurs"), a.conflictSideText("conflictTheirs"), a.t("conflictEditorBase")}
		sources := []string{s.Original.Ours, s.Original.Theirs, s.Original.Base}
		sourceButton := func(c *ui.Context, i int, label string) {
			kind := buttonNormal
			if s.Source == i {
				kind = buttonPrimary
			}
			button := workbenchButton(c.Key(label), label, kind, false)
			if button.Clicked() {
				s.Source = i
			}
		}
		// Three labels side by side need a little over 330 points. A narrower
		// details pane stacks them so the choice stays on screen instead of
		// running past the window's edge.
		if root.Bounds().W >= 340 {
			ui.Row(c).FillWidth().Gap(6).Shrink(0).Children(func() {
				for i, label := range labels {
					sourceButton(c, i, label)
				}
			})
		} else {
			ui.Column(c).FillWidth().Gap(4).Shrink(0).Children(func() {
				for i, label := range labels {
					sourceButton(c, i, label)
				}
			})
		}
		wide := root.Bounds().W >= 560 && s.Source != 2
		sourceHeight := float32(170)
		// The source panes give up their height first: the result editor and
		// the save action must stay on screen even when an error message and a
		// stacked source choice take the room above them.
		if available := root.Bounds().H; available > 0 {
			sourceHeight = max(60, min(sourceHeight, available-410))
		}
		ui.Row(c.Key("sources")).FillWidth().Height(sourceHeight).Shrink(0).Gap(12).AlignItems(ui.Stretch).Children(func() {
			for i := range sources {
				if (wide && i == 2) || (!wide && i != s.Source) {
					continue
				}
				ui.Column(c.Key(i)).Grow(1).Basis(0).MinWidth(0).Gap(6).Children(func() {
					if wide {
						display.Text(c, labels[i], display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)})
					}
					layout.ScrollArea(c.Key("source"), layout.ScrollAreaProps{Direction: layout.Both, State: &s.Scroll[i]}, func() {
						if s.Original.OIDs[[]int{1, 2, 0}[i]] == "" {
							display.Text(c, a.t("conflictEditorAbsent"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)})
						} else {
							ui.Text(c, sources[i]).Font("Menlo").FontSize(a.patchFontSize()).Selectable().Shrink(0)
						}
					}).FillWidth().Grow(1).Basis(0).MinHeight(0)
				})
			}
		})
		display.Text(c, a.t("conflictEditorResult"), display.TextProps{FontSize: scaledFont(11)})
		tk := theme.TokensOf(c)
		ui.Column(c.Key("result-surface")).FillWidth().Grow(1).Basis(0).MinHeight(100).Padding(10).
			Background(tk.BgContainer).Border(1, tk.BorderSecondary).Radius(tk.BorderRadius).Children(func() {
			ui.TextAreaBase(c.Key("result"), &s.Text).Lines(5, 100).Fill().MinHeight(0).
				Label(a.t("conflictEditorResult")).Font("Menlo").FontSize(a.patchFontSize()).TextColor(tk.Text).Disabled(a.busy)
		})
		display.Text(c, a.t("conflictEditorSaveHint"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11), Rows: 3})
		if workbenchButton(c, a.t("conflictEditorSave"), buttonPrimary, a.busy || a.loading).Clicked() {
			a.saveConflictEditor(s)
		}
	})
}
