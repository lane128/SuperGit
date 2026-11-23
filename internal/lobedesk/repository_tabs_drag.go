package lobedesk

import (
	"slices"

	"github.com/egoist/mygo/ui"
)

type repositoryTabDrag struct{ session *app }

func (tabs *repositoryTabs) tabDropTarget(row ui.Element, target *app, color ui.Color) *app {
	if over, ok := ui.DragOver[repositoryTabDrag](row); ok && over.session != target {
		before := slices.Index(tabs.sessions, over.session) > slices.Index(tabs.sessions, target)
		row.Draw(func(p *ui.Painter, bounds ui.Rect) {
			x := bounds.X
			if !before {
				x += bounds.W - 2
			}
			p.Fill(ui.Rect{X: x, Y: bounds.Y + 4, W: 2, H: bounds.H - 8}, color, 1)
		})
	}
	if dropped, ok := ui.Drop[repositoryTabDrag](row); ok && dropped.session != target {
		return dropped.session
	}
	return nil
}

func (tabs *repositoryTabs) moveTab(source, target *app) bool {
	from, to := slices.Index(tabs.sessions, source), slices.Index(tabs.sessions, target)
	if from < 0 || to < 0 || from == to || !tabs.canSwitch() {
		return false
	}
	if source.repo != nil && slices.ContainsFunc(tabs.sessions[min(from, to):max(from, to)+1], func(other *app) bool {
		return other != source && other.repo != nil
	}) {
		tabs.sessionState.orderChanged = true
	}
	if from < to {
		copy(tabs.sessions[from:to], tabs.sessions[from+1:to+1])
	} else {
		copy(tabs.sessions[to+1:from+1], tabs.sessions[to:from])
	}
	tabs.sessions[to] = source
	return true
}
