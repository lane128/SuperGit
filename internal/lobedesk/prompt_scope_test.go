package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestPromptCanOpenNextStepAfterSubmission(t *testing.T) {
	a := &app{locale: localeEnglish, repo: &gitrepo.Repository{Root: "first"}}
	a.askPrompt("Tag name", "Name", "release", "Next", func(name string) {
		a.askPrompt("Tag message", "Message", name, "Create", func(string) {})
	})
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.promptDialog(c) }, 640, 420)
	button, ok := tt.Find("Next")
	if !ok {
		t.Fatal("first step unavailable")
	}
	tt.ClickAt(button.X+button.W/2, button.Y+button.H/2)
	tt.Frame()
	if !tt.HasText("Tag message") || a.prompt.value != "release" {
		t.Fatal("pending protection blocked annotated-tag next step")
	}
}

func TestRepositoryPromptOnlyDrawsInMainWindow(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	a.askPrompt("Main input", "Name", "draft", "Save", func(string) {})
	settings := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.promptDialogFor(c, toastSettings) }, 640, 420)
	if settings.HasText("Main input") {
		t.Fatal("repository prompt moved to Settings")
	}
	main := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.promptDialog(c) }, 640, 420)
	if !main.HasText("Main input") {
		t.Fatal("main window lost its input prompt")
	}
}

func TestNewPromptDoesNotReplacePendingDraft(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	acted := ""
	a.askPrompt("First target", "Name", "draft", "Save first", func(value string) { acted = "first:" + value })
	a.prompt.value = "edited draft"
	a.askPrompt("Second target", "Name", "other", "Save second", func(value string) { acted = "second:" + value })
	if a.prompt.title != "First target" || a.prompt.value != "edited draft" {
		t.Fatal("new prompt overwrote pending target or draft")
	}
	if len(a.notices[toastMain]) != 1 || a.notices[toastMain][0].kind != "warning" {
		t.Fatal("rejected prompt must explain how to retry")
	}
	a.prompt.action(a.prompt.value)
	if acted != "first:edited draft" {
		t.Fatal("prompt lost original callback")
	}
}

func TestRepositoryPromptExitsOnRootChange(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	a.askPrompt("Old target", "Name", "draft", "Save", func(string) {})
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.promptDialog(c) }, 640, 420)
	if !tt.HasText("Old target") {
		t.Fatal("fixture did not open prompt")
	}
	a.repo = &gitrepo.Repository{Root: "first"}
	tt.Frame()
	if !tt.HasText("Old target") || a.prompt.value != "draft" {
		t.Fatal("same-root refresh discarded prompt draft")
	}
	a.repo = &gitrepo.Repository{Root: "second"}
	tt.Frame()
	if a.prompt.open || tt.HasText("Old target") {
		t.Fatal("old prompt survived repository change")
	}
}

func TestRepositoryPromptCallbackRejectsChangedRoot(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	acted := false
	a.askPrompt("Old target", "Name", "draft", "Save", func(string) { acted = true })
	callback := a.prompt.action
	a.repo = &gitrepo.Repository{Root: "second"}
	callback("new value")
	if acted {
		t.Fatal("old prompt callback ran against another repository")
	}
}

func TestCurrentPromptCanReplaceStaleRepositoryPrompt(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	a.askPrompt("Old target", "Name", "old", "Save", func(string) {})
	a.repo = &gitrepo.Repository{Root: "second"}
	a.askPrompt("Current target", "Name", "new", "Save", func(string) {})
	if a.prompt.title != "Current target" || a.prompt.value != "new" {
		t.Fatal("stale prompt blocked a current request")
	}
}
