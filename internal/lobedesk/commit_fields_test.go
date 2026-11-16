package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func focusCommitField(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	field, ok := tt.AccessByRole(ui.RoleTextField, name)
	if !ok {
		t.Fatalf("missing input %q", name)
	}
	tt.ClickAt(field.Bounds.X+field.Bounds.W/2, field.Bounds.Y+field.Bounds.H/2)
}

func TestCommitFieldsHaveExplicitBoundaries(t *testing.T) {
	a := &app{locale: localeEnglish}
	tt := commitFormTester(a)
	focusCommitField(t, tt, "Commit summary (required)")
	tt.Type("A clear title")
	tt.Key(0, ui.KeyEnter)
	if a.commitText != "A clear title" {
		t.Fatalf("title Enter changed message: %q", a.commitText)
	}
	focusCommitField(t, tt, "Description (optional)")
	tt.Type("First paragraph")
	tt.Key(0, ui.KeyEnter)
	tt.Type("second line")
	if a.commitText != "A clear title\n\nFirst paragraph\nsecond line" {
		t.Fatalf("message = %q", a.commitText)
	}
	tt.Key(ui.Cmd, ui.KeyZ)
	if a.commitText == "" {
		t.Fatal("body undo cleared the title")
	}
}

func TestCommitPastePreservesMessageBody(t *testing.T) {
	a := &app{locale: localeEnglish, commitText: "old title\n\nExisting body"}
	tt := commitFormTester(a)
	focusCommitField(t, tt, "Commit summary (required)")
	tt.Key(ui.Cmd, ui.KeyA)
	tt.SetClipboard("Pasted title\n\nPasted paragraph\n\nMore detail")
	tt.Key(ui.Cmd, ui.KeyV)
	want := "Pasted title\n\nPasted paragraph\n\nMore detail\n\nExisting body"
	if a.commitText != want {
		t.Fatalf("multiline paste dropped message bytes: %q", a.commitText)
	}
	// Undo and redo own the paste's cross-field edit rather than dropping body.
	tt.Key(ui.Cmd, ui.KeyZ)
	if a.commitText != "old title\n\nExisting body" {
		t.Fatalf("paste undo = %q", a.commitText)
	}
	tt.Key(ui.Cmd|ui.Shift, ui.KeyZ)
	if a.commitText != want {
		t.Fatalf("paste redo = %q", a.commitText)
	}
}

func TestDraftRenderingDoesNotNormalizeExistingMessage(t *testing.T) {
	for _, original := range []string{"subject\nbody without separator", "subject\n\n\nbody\n", "subject\n", "\nbody", "subject\r\n\r\nbody"} {
		a := &app{locale: localeEnglish, commitText: original}
		tt := commitFormTester(a)
		tt.Frame()
		if a.commitText != original {
			t.Fatalf("render normalized %q to %q", original, a.commitText)
		}
		// A new externally supplied draft is projected, without an old field
		// overwriting operation prefill, root reset or successful-commit clear.
		a.commitText = "replacement\n\nbody"
		tt.Frame()
		focusCommitField(t, tt, "Commit summary (required)")
		tt.Key(ui.Cmd, ui.KeyA)
		tt.Type("edited")
		if a.commitText != "edited\n\nbody" {
			t.Fatalf("field overwrote external draft: %q", a.commitText)
		}
	}
}

func TestCommitPasteKeepsCaretSelectionAndOldSeparators(t *testing.T) {
	for _, original := range []string{"original\nbody", "original\n\n\nbody\n"} {
		a := &app{locale: localeEnglish, commitText: original}
		tt := commitFormTester(a)
		focusCommitField(t, tt, "Commit summary (required)")
		tt.Key(ui.Cmd, ui.KeyA)
		tt.SetClipboard("新标题\n\n粘贴正文")
		tt.Key(ui.Cmd, ui.KeyV)
		tt.Key(ui.Cmd, ui.KeyZ)
		if a.commitText != original {
			t.Fatalf("undo normalized original %q to %q", original, a.commitText)
		}
	}
	a := &app{locale: localeEnglish, commitText: "前后"}
	tt := commitFormTester(a)
	focusCommitField(t, tt, "Commit summary (required)")
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyRight)
	tt.Key(0, ui.KeyLeft)
	tt.SetClipboard("缀\n正文")
	tt.Key(ui.Cmd, ui.KeyV)
	if a.commitText != "前缀\n\n正文后" {
		t.Fatalf("paste lost insertion-point suffix: %q", a.commitText)
	}
}

func TestCommitFieldsDoNotInsertNewlineOnCommitShortcut(t *testing.T) {
	a := &app{locale: localeEnglish, commitText: "title\n\nbody"}
	tt := commitFormTester(a)
	for _, label := range []string{"Commit summary (required)", "Description (optional)"} {
		focusCommitField(t, tt, label)
		tt.Key(ui.Cmd, ui.KeyEnter)
		if a.commitText != "title\n\nbody" { t.Fatalf("disabled submit from %s edited the message: %q", label, a.commitText) }
	}
}

func TestCommitRequiresFirstLineTitle(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "fixture"}, commitText: "\n\nbody", commitIdentity: readyCommitIdentity("fixture"), status: gitrepo.Status{Entries: []gitrepo.Entry{{Path: "a", Index: 'M'}}}}
	if a.canCommit() {
		t.Fatal("body without a first-line title permits commit")
	}
}
