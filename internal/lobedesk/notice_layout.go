package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
)

// The main window reserves the measured branch-action footer beneath toasts.
// The settings window has its own viewport and no repository-action footer.
func (a *app) toastHost(c *ui.Context, owner toastOwner) {
	inset := float32(0)
	if owner == toastMain {
		inset = a.mainToastBottomInset
	}
	feedback.ToastHost(c, feedback.ToastHostProps{}).Padding(16, 16, 16+inset, 16)
}
