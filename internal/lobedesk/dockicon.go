package lobedesk

// syncDockIcon makes the Dock follow the window's appearance. The view builds
// every frame; the Switcher keeps the last choice and only touches the system
// when the appearance changes.
func (a *app) syncDockIcon(dark bool) {
	_ = a.dockIcon.Apply(dark)
}
