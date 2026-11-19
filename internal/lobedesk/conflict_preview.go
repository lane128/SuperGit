package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"supergit/internal/gitrepo"
)

func (a *app) conflictPreviewEntry(path string) (gitrepo.Entry, bool) {
	for _, entry := range a.status.Entries {
		if entry.Path == path && entry.Conflict {
			return entry, true
		}
	}
	return gitrepo.Entry{}, false
}

func (a *app) conflictPreviewGuidance(c *ui.Context, entry gitrepo.Entry, overview ...bool) {
	ui.Column(c).FillWidth().Shrink(0).Padding(10, 12).Gap(6).Children(func() {
		display.Text(c, a.t("conflictPreviewHint"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)})
		if workbenchButton(c, a.t("conflictEditorOpen"), buttonNormal, a.repo == nil || a.busy || a.loading).Clicked() {
			if entry.Path != a.selectedPath || (len(overview) > 0 && overview[0]) {
				a.openDetailFile(detailFileKey{Path: entry.Path})
			}
			a.openConflictEditor()
		}
		if entry.Index == 'D' || entry.Worktree == 'D' {
			display.Text(c, a.t("conflictDeletionHint"), display.TextProps{Type: display.TextWarning, FontSize: scaledFont(11)})
		}
	})
}
