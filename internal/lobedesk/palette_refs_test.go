package lobedesk

import (
	"path/filepath"
	"strings"
	"testing"
)

func pickRefParameter(t *testing.T, a *app, id string) {
	t.Helper()
	for _, row := range a.paletteChoices() {
		if row.id == id {
			if row.reason != "" {
				t.Fatalf("disabled parameter %q: %s", id, row.reason)
			}
			a.acceptPaletteChoice(row, a.commandPalette.generation)
			return
		}
	}
	t.Fatalf("missing parameter %q in %+v", id, a.paletteChoices())
}

func TestPaletteRefsRenameUsesOriginallyChosenTip(t *testing.T) {
	for _, stale := range []bool{false, true} {
		a, callbacks, old := quickCheckoutFixture(t)
		a.openCommandPalette()
		a.runPaletteCommand("branch.rename", a.commandPalette.generation)
		pickRefParameter(t, a, "topic")
		step := a.commandPalette.step
		if step == nil || step.submit == nil || a.commandPalette.query != "topic" {
			t.Fatal("expected prefilled branch name")
		}
		if stale {
			pullFixtureGit(t, a.repo.Root, "branch", "-f", "topic", "main")
		}
		a.finishPaletteStep(a.commandPalette.generation, func() { step.submit("new-name") })
		waitQuickCheckout(t, a, callbacks)
		if stale {
			if got := pullFixtureGit(t, a.repo.Root, "branch", "--list", "new-name"); got != "" {
				t.Fatal("renamed a replacement branch")
			}
		} else if got := pullFixtureGit(t, a.repo.Root, "rev-parse", "new-name"); got != old {
			t.Fatalf("renamed branch points to %s, expected %s", got, old)
		}
	}
}

func TestPaletteRefsRemoteOnlyTagReturnsIntoSameCommand(t *testing.T) {
	a, callbacks, _ := quickCheckoutFixture(t)
	bare := filepath.Join(t.TempDir(), "bare.git")
	pullFixtureGit(t, a.repo.Root, "clone", "--bare", a.repo.Root, bare)
	pullFixtureGit(t, bare, "tag", "only-on-server", "topic")
	pullFixtureGit(t, a.repo.Root, "remote", "add", "peer", bare)
	var err error
	a.remotes, err = a.repo.Remotes(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.openCommandPalette()
	a.runPaletteCommand("tag.deleteRemote", a.commandPalette.generation)
	pickRefParameter(t, a, "peer")
	waitQuickCheckout(t, a, callbacks)
	if !a.commandPalette.open || a.commandPalette.actionID != "tag.deleteRemote" || a.commandPalette.step.parent == nil {
		t.Fatal("remote read lost command or Back navigation")
	}
	pickRefParameter(t, a, "only-on-server")
	if !a.confirm.open || !a.confirm.danger || !strings.Contains(a.confirm.message, "peer/only-on-server") {
		t.Fatal("remote-only tag did not open its named deletion confirmation")
	}
	if got := pullFixtureGit(t, bare, "tag", "-l"); got != "only-on-server" {
		t.Fatal("opening the confirmation changed the remote")
	}
}
