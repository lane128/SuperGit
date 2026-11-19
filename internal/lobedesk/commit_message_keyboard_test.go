package lobedesk

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestMessageCommitMenuKeyboardScrollAndCancel(t *testing.T) {
	oldScale := uiFontScaleValue()
	setUIFontScale(fontScaleLarge)
	t.Cleanup(func() { setUIFontScale(oldScale) })
	a, callbacks, source, target := messageNavigationFixture(t)
	var urls []string
	for index := range 8 {
		urls = append(urls, "https://example.invalid/"+strings.Repeat("long-path/", 20)+string(rune('a'+index)))
	}
	body := strings.Join(urls, " ") + " " + source.Parents[0] + " " + strings.ToUpper(target) + " " + target
	a.commitMessage = source.Subject + "\n\n" + body
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Children(func() { a.commitInfoContent(c, source, theme.TokensOf(c)) })
	}, 480, 360)
	r, ok := tt.Find(body)
	if !ok {
		t.Fatal("message missing")
	}
	tt.ClickAt(r.X+5, r.Y+5)
	tt.Key(ui.Shift, ui.KeyF10)
	label := "Jump to Commit · " + target
	if _, ok := tt.AccessByRole(ui.RoleMenuItem, label); !ok {
		t.Fatal("keyboard did not open typed menu")
	}
	count := 0
	for _, item := range tt.AccessAll() {
		if item.Role == ui.RoleMenuItem && item.Name == label {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("same SHA had %d entries", count)
	}
	tt.Key(0, ui.KeyEscape)
	if _, ok := tt.AccessByRole(ui.RoleMenuItem, label); ok {
		t.Fatal("Escape left SHA menu open")
	}
	if a.messageCommitRequest != nil || a.selectedSHA != source.SHA || len(tt.OpenedURLs()) != 0 {
		t.Fatal("Escape dispatched an action")
	}
	tt.Key(ui.Shift, ui.KeyF10)
	for step := 0; step < 12; step++ {
		item, ok := tt.AccessByRole(ui.RoleMenuItem, label)
		if ok && item.Focused {
			break
		}
		tt.Key(0, ui.KeyDown)
	}
	item, ok := tt.AccessByRole(ui.RoleMenuItem, label)
	if !ok || !item.Focused || item.Offscreen || item.Bounds.Y < 0 || item.Bounds.Y+item.Bounds.H > 360 || item.Bounds.X < 0 || item.Bounds.X+item.Bounds.W > 480 {
		t.Fatalf("active SHA target unreachable: %+v", item)
	}
	tt.Key(0, ui.KeyEnter)
	historyCompletion(t, callbacks)()
	if a.selectedSHA != target || len(tt.OpenedURLs()) != 0 || !a.canNavigateHistory(-1) {
		t.Fatal("Enter did not dispatch exact SHA choice")
	}
}
