package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"testing"
)

func TestCommitMessageFocusExpiresWhileEditorIsUnmounted(t *testing.T) {
	for _, kind := range []string{"busy", "loading", "chooser", "reference popup"} {
		t.Run(kind, func(t *testing.T) {
			a, _ := overviewFixture(t)
			a.page = "changes"
			a.focusCommitMessage()
			setBlocked := func(value bool) {
				switch kind {
				case "busy":
					a.busy = value
				case "loading":
					a.loading = value
				case "chooser":
					a.repositoryChooserOpen = value
				case "reference popup":
					a.historyRefOpen = value
				}
			}
			setBlocked(true)
			mounted := false
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				a.registerShortcuts(c)
				ui.Column(c).Width(550).Children(func() {
					if mounted {
						a.commitMessageFields(c)
					}
				})
			}, 600, 400)
			setBlocked(false)
			mounted = true
			tt.Frame()
			if tt.Focused(a.t("commitSummary")) {
				t.Fatal("expired request stole focus when editor mounted later")
			}
		})
	}
}
