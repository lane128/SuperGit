package lobedesk

import (
	"slices"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

type paletteStage uint8

const (
	paletteCommands paletteStage = iota
	paletteFetchOptions
	paletteFetchRemote
	paletteMergeRef
	paletteWorkflow
)

type paletteChoice struct {
	id, label, detail, prefix, reason, sha string
}

func (a *app) setPaletteStage(stage paletteStage) {
	a.commandPaletteGeneration++
	a.commandPalette = commandPaletteState{open: true, root: a.paletteRoot(), epoch: a.loadSeq,
		stage: stage, generation: a.commandPaletteGeneration,
		cursor: -1, reveal: true, searchFocus: true}
	a.invalidate()
}

func (a *app) paletteInputLabels() (title, chip, placeholder string) {
	switch a.commandPalette.stage {
	case paletteWorkflow:
		step := a.commandPalette.step
		return step.title, step.title, step.placeholder
	case paletteFetchOptions:
		return a.t("paletteFetchOptions"), a.t("fetch"), a.t("paletteFetchFilter")
	case paletteFetchRemote:
		return a.t("fetch"), "", a.t("remoteName")
	case paletteMergeRef:
		return a.t("paletteMergeBranch"), a.t("paletteMergeBranch"), a.t("branchName")
	default:
		return a.t("commandPalette"), "", a.t("commandPaletteSearch")
	}
}

func mergePaletteCommand() appShortcut {
	return appShortcut{ID: "branch.merge", Label: "paletteMergeCommand",
		Available: func(a *app) bool { return a.commandPaletteUnavailable("branch.merge") == "" && !a.commandPalette.open },
		Run: func(a *app) {
			if a.commandPaletteUnavailable("branch.merge") == "" && !a.commandPalette.open {
				a.setPaletteStage(paletteMergeRef)
			}
		}}
}

func (a *app) paletteMergeUnavailable() string {
	switch {
	case a.status.Branch == "" || a.status.Head == "":
		return a.t("paletteMergeNeedBranch")
	case a.loadErr != "" || a.branchesErr != "" || a.remoteBranchesErr != "":
		return a.t("quickCheckoutReadError")
	case a.operation.Kind != "":
		return a.t("quickCheckoutOperation")
	case a.operationConflicts():
		return a.t("quickCheckoutConflict")
	}
	return ""
}

func (a *app) paletteCommandChoice(s appShortcut) paletteChoice {
	detail := s.Keys
	if meta, ok := fetchPaletteMetadata[s.ID]; ok {
		detail = a.t(meta.detail)
	} else if detail == "" {
		detail = paletteGitHints[s.ID]
	}
	return paletteChoice{id: s.ID, label: a.t(s.Label), detail: detail, reason: a.commandPaletteUnavailable(s.ID)}
}

func (a *app) paletteChoices() []paletteChoice {
	var rows []paletteChoice
	switch a.commandPalette.stage {
	case paletteWorkflow:
		return a.stepChoices()
	case paletteCommands:
		for _, s := range a.commandPaletteEntries() {
			rows = append(rows, a.paletteCommandChoice(s))
		}
		return rows
	case paletteFetchOptions:
		for _, s := range fetchPaletteCommands() {
			row := a.paletteCommandChoice(s)
			if s.ID == "repo.fetch" {
				s.Label = "fetch"
				row.label, row.detail = a.t(s.Label), a.t("paletteFetchDetail")
			}
			meta := fetchPaletteMetadata[s.ID]
			names := append([]string{row.label, row.id, row.detail, messages[localeEnglish][s.Label], messages[localeChinese][s.Label]}, meta.aliases...)
			if commandMatchScore(a.commandPalette.query, names...) >= 0 {
				rows = append(rows, row)
			}
		}
		return rows
	case paletteFetchRemote:
		for _, remote := range a.remotes {
			rows = append(rows, paletteChoice{id: remote.Name, label: remote.Name})
		}
	case paletteMergeRef:
		add := func(branches []gitrepo.Branch, fullPrefix, badge string) {
			for _, b := range branches {
				if b.SHA == "" || badge == "L" && (b.Current || b.Name == a.status.Branch) {
					continue
				}
				rows = append(rows, paletteChoice{id: fullPrefix + b.Name, label: b.Name, prefix: badge, sha: b.SHA})
			}
		}
		add(a.branches, "refs/heads/", "L")
		add(a.remoteBranches, "refs/remotes/", "R")
	}
	id := "repo.fetch"
	if a.commandPalette.stage == paletteMergeRef {
		id = "branch.merge"
	}
	if a.commandPalette.stage == paletteFetchRemote {
		id = "repo.fetchRemote"
	}
	reason := a.commandPaletteUnavailable(id)
	for i := range rows {
		rows[i].reason = reason
	}
	return slices.DeleteFunc(rows, func(row paletteChoice) bool {
		return commandMatchScore(a.commandPalette.query, row.label, row.id) < 0
	})
}

func (a *app) acceptPaletteChoice(row paletteChoice, generation uint64) {
	a.discardStaleCommandPalette()
	s := a.commandPalette
	if !s.open || s.generation != generation {
		return
	}
	if s.stage == paletteCommands {
		a.runPaletteCommand(row.id, generation)
		return
	}

	// Confirm the exact ref and SHA (or fetch option) shown by the last frame.
	index := slices.Index(a.paletteChoices(), row)
	if index < 0 || row.reason != "" {
		return
	}
	if s.stage == paletteFetchOptions {
		a.dispatchPaletteCommand(row.id)
		return
	}
	if s.stage == paletteWorkflow {
		if s.step.choose != nil {
			a.commandPalette.cursor = index
			a.finishPaletteStep(generation, func() { s.step.choose(row) })
		}
		return
	}
	if s.stage == paletteMergeRef {
		if a.commandPaletteUnavailable("branch.merge") == "" {
			a.commandPalette = commandPaletteState{}
			a.confirmIntegrate("merge", row.id, row.sha)
		}
		return
	}
	if s.stage != paletteFetchRemote {
		return
	}
	if a.commandPaletteUnavailable("repo.fetchRemote") != "" {
		return
	}
	i := slices.IndexFunc(a.remotes, func(r gitrepo.Remote) bool { return r.Name == row.id })
	if i < 0 {
		return
	}
	a.commandPalette = commandPaletteState{}
	a.fetchRemotes(a.remotes[i:i+1], gitrepo.FetchDefault, a.t("fetchFailed"), a.t("fetchSuccess"))
}

func commandPanel(c *ui.Context, backdrop, panel ui.Element, title string) ui.Element {
	tk := theme.TokensOf(c)
	width, _ := c.Size()
	backdrop.Background(ui.RGBA(0, 0, 0, .10))
	panel.Column().Label(title).Width(min(1000, max(0, width-32))).Padding(4).Gap(3).
		Radius(4).Border(1, tk.Border).Background(tk.BgElevated)
	tk.Elevate(panel, theme.BoxShadowSecondary)
	return panel
}

func (a *app) paletteInput(c *ui.Context, chip, label string, query *string, enter func(), step func(int)) ui.Element {
	tk := theme.TokensOf(c)
	var input ui.Element
	ui.Row(c).FillWidth().Height(scaledFont(30)).AlignItems(ui.Center).Gap(5).PaddingX(6).Radius(3).Background(tk.FillTertiary).Children(func() {
		if chip != "" {
			ui.Text(c, chip).FontSize(scaledFont(12)).FontWeight(600).Padding(1, 6).Radius(3).Background(tk.FillSecondary).MaxWidthPercent(52).SingleLine().Tooltip(chip).Shrink(0)
		}
		input = ui.TextInputBase(c, query).Grow(1).Basis(0).MinWidth(0).FontSize(scaledFont(13)).
			Placeholder(label).Label(label).PlaceholderColor(tk.TextPlaceholder).TextColor(tk.Text).FocusRing(false)
		if enter != nil && input.Submitted() {
			enter()
		}
		if step != nil {
			input.OnShortcut(0, ui.KeyDown, func() { step(1); a.invalidate() })
			input.OnShortcut(0, ui.KeyUp, func() { step(-1); a.invalidate() })
		}
	})
	return input
}

func drawPaletteChoice(c *ui.Context, row paletteChoice, query string) {
	tk := theme.TokensOf(c)
	if row.prefix != "" {
		color := tk.Link
		if row.prefix == "R" {
			if purple, ok := refColor(tk, "purple"); ok {
				color = deepInk(purple, tk.BgElevated, 4.5)
			}
		}
		ui.Text(c, row.prefix).Width(18).FontWeight(700).FontSize(scaledFont(12)).TextColor(color)
	}
	ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
		if spans := commandHighlightSpans(query, row.label); len(spans) > 0 {
			ui.RichText(c, spans...).FontSize(scaledFont(12)).SingleLine().Tooltip(row.label)
		} else {
			ui.Text(c, row.label).FontSize(scaledFont(12)).SingleLine().Tooltip(row.label)
		}
		if row.reason != "" {
			display.Text(c, row.reason, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).SingleLine()
		}
	})
	if row.detail != "" {
		display.Text(c, row.detail, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11), Rows: 2}).MaxWidthPercent(62).Tooltip(row.detail)
	}
}
