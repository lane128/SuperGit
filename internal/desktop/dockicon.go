package desktop

// syncDockIcon makes the Dock follow the app's current appearance. The view
// builds every frame; the Switcher keeps the last choice and only touches the
// system when the appearance changes. A nil Switcher is a no-op.
func (a *app) syncDockIcon(dark bool) {
	_ = a.dockIcon.Apply(dark)
}
